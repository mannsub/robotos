package main

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"

	v1 "github.com/mannsub/robotos/proto/v1/gen/go/v1"
)

// streamDef pairs a Redis key, Foxglove topic, and a sample proto message.
type streamDef struct {
	redisKey string
	topic    string
	sample   proto.Message
}

// streams defines all channels the bridge will advertise and forward.
var streams = []streamDef{
	{"sensor:data", "/sensor", &v1.SensorData{}},
	{"neodm:state", "/neodm/state", &v1.NeoDMState{}},
	{"hal:motion", "/motion_command", &v1.MotionCommand{}},
}

var upgrader = websocket.Upgrader{
	CheckOrigin:  func(r *http.Request) bool { return true },
	Subprotocols: []string{foxgloveSubprotocol},
}

// channelInfo holds precomputed data for a single Foxglove channel.
type channelInfo struct {
	id         uint32
	topic      string
	schemaName string
	schemaB64  string // base64-encoded FileDescriptorSet bytes
	redisKey   string
}

// clientState tracks subscription state for one connected Foxglove client.
// Per the Foxglove ws-protocol spec, the client chooses its own subscriptionId;
// the server must use that id (not the server channelId) in binary MessageData frames.
type clientState struct {
	conn *websocket.Conn
	wmu  sync.Mutex // serializes concurrent writes to conn

	smu  sync.Mutex
	subs map[uint32]uint32 // serverChannelID → clientSubscriptionID
}

func newClientState(conn *websocket.Conn) *clientState {
	return &clientState{
		conn: conn,
		subs: make(map[uint32]uint32),
	}
}

func (c *clientState) subscribe(subID, channelID uint32) {
	c.smu.Lock()
	defer c.smu.Unlock()
	c.subs[channelID] = subID
}

func (c *clientState) unsubscribe(subID uint32) {
	c.smu.Lock()
	defer c.smu.Unlock()
	for ch, sid := range c.subs {
		if sid == subID {
			delete(c.subs, ch)
		}
	}
}

// subIDFor returns the client-chosen subscriptionId for the given server channelID,
// or (0, false) if the client has not subscribed to that channel.
func (c *clientState) subIDFor(channelID uint32) (uint32, bool) {
	c.smu.Lock()
	defer c.smu.Unlock()
	id, ok := c.subs[channelID]
	return id, ok
}

func (c *clientState) writeMessage(msgType int, data []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	return c.conn.WriteMessage(msgType, data)
}

// Bridge manages WebSocket clients and forwards Redis messages to them.
type Bridge struct {
	rdb      *redis.Client
	channels []channelInfo
	// redisToChannelID maps Redis key to Foxglove channel ID.
	redisToChannelID map[string]uint32

	mu      sync.RWMutex
	clients map[*websocket.Conn]*clientState
}

// NewBridge initializes the bridge by precomputing all channel metadata.
func NewBridge(rdb *redis.Client) *Bridge {
	b := &Bridge{
		rdb:              rdb,
		redisToChannelID: make(map[string]uint32),
		clients:          make(map[*websocket.Conn]*clientState),
	}

	for i, s := range streams {
		id := uint32(i + 1)
		fds := buildSchemaData(s.sample)

		b.channels = append(b.channels, channelInfo{
			id:         id,
			topic:      s.topic,
			schemaName: schemaName(s.sample),
			schemaB64:  base64.StdEncoding.EncodeToString(fds),
			redisKey:   s.redisKey,
		})
		b.redisToChannelID[s.redisKey] = id
	}

	return b
}

// ServeHTTP handles incoming WebSocket upgrade requests.
func (b *Bridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[foxglove-bridge] upgrade error: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("[foxglove-bridge] client connected: %s", r.RemoteAddr)
	client := newClientState(conn)
	b.registerClient(client)
	defer b.unregisterClient(conn)

	// Send ServerInfo.
	info, _ := json.Marshal(ServerInfo{
		Op:           "serverInfo",
		Name:         "RobotOS Foxglove Bridge",
		Capabilities: []string{},
	})
	if err := client.writeMessage(websocket.TextMessage, info); err != nil {
		return
	}

	// Advertise all channels.
	ads := make([]ChannelAdvertisement, 0, len(b.channels))
	for _, ch := range b.channels {
		ads = append(ads, ChannelAdvertisement{
			ID:             ch.id,
			Topic:          ch.topic,
			Encoding:       "protobuf",
			SchemaName:     ch.schemaName,
			Schema:         ch.schemaB64,
			SchemaEncoding: "protobuf",
		})
	}
	adv, _ := json.Marshal(Advertise{Op: "advertise", Channels: ads})
	if err := client.writeMessage(websocket.TextMessage, adv); err != nil {
		return
	}

	// Read loop: handle subscribe / unsubscribe messages from the client.
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var op struct {
			Op string `json:"op"`
		}
		if err := json.Unmarshal(raw, &op); err != nil {
			continue
		}
		switch op.Op {
		case "subscribe":
			var msg ClientSubscribe
			if err := json.Unmarshal(raw, &msg); err == nil {
				for _, s := range msg.Subscriptions {
					client.subscribe(s.ID, s.ChannelID)
					log.Printf("[foxglove-bridge] client %s subscribed channelId=%d as subId=%d",
						r.RemoteAddr, s.ChannelID, s.ID)
				}
			}
		case "unsubscribe":
			var msg ClientUnsubscribe
			if err := json.Unmarshal(raw, &msg); err == nil {
				for _, id := range msg.SubscriptionIDs {
					client.unsubscribe(id)
				}
			}
		}
	}

	log.Printf("[foxglove-bridge] client disconnected: %s", r.RemoteAddr)
}

// publishToClients sends a binary MessageData frame to every client that has
// subscribed to the given channelID, using the client's own subscriptionId.
// Foxglove binary message format (op=0x01):
// [1 byte: 0x01][4 bytes: subscriptionId LE][8 bytes: timestamp ns LE][payload]
func (b *Bridge) publishToClients(channelID uint32, timestampNs uint64, data []byte) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, client := range b.clients {
		subID, ok := client.subIDFor(channelID)
		if !ok {
			continue // this client has not subscribed to this channel
		}

		header := make([]byte, 13)
		header[0] = 0x01 // op: MessageData
		binary.LittleEndian.PutUint32(header[1:5], subID)
		binary.LittleEndian.PutUint64(header[5:13], timestampNs)
		frame := append(header, data...)

		if err := client.writeMessage(websocket.BinaryMessage, frame); err != nil {
			log.Printf("[foxglove-bridge] write error: %v", err)
		}
	}
}

func (b *Bridge) registerClient(c *clientState) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clients[c.conn] = c
}

func (b *Bridge) unregisterClient(conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, conn)
}

// Run subscribes to Redis and forwards messages to all connected WebSocket clients.
func (b *Bridge) Run(ctx context.Context) {
	keys := make([]string, 0, len(b.redisToChannelID))
	for k := range b.redisToChannelID {
		keys = append(keys, k)
	}

	sub := b.rdb.Subscribe(ctx, keys...)
	defer sub.Close()

	msgCh := sub.Channel()
	log.Printf("[foxglove-bridge] subscribed to Redis keys: %v", keys)

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgCh:
			if !ok {
				return
			}
			chanID, found := b.redisToChannelID[msg.Channel]
			if !found {
				continue
			}
			b.publishToClients(
				chanID,
				uint64(time.Now().UnixNano()),
				[]byte(msg.Payload),
			)
		}
	}
}
