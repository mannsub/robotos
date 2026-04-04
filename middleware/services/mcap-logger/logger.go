package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/foxglove/mcap/go/mcap"
	"github.com/redis/go-redis/v9"
)

// Logger subscribes to Redis channels and writes proto-encoded message to an MCAP file.
type Logger struct {
	rdb    *redis.Client
	writer *mcap.Writer
	file   *os.File
	// topicToChanID maps MCAP topic string to its registered channel ID.
	topicToChanID map[string]uint16
	// redisToTopic maps Redis pub/sub key to MCAP topic string.
	redisToTopic map[string]string
}

// NewLogger creates and initializes a Logger. It opens the output file,
// creates the MCAP writer, and refisters all schemas and channels.
func NewLogger(rdb *redis.Client, outPath string) (*Logger, error) {
	f, err := os.Create(outPath)
	if err != nil {
		return nil, err
	}

	w, err := mcap.NewWriter(f, &mcap.WriterOptions{
		Chunked:     true,
		Compression: mcap.CompressionZSTD,
		IncludeCRC:  true,
	})
	if err != nil {
		f.Close()
		return nil, err
	}

	l := &Logger{
		rdb:           rdb,
		writer:        w,
		file:          f,
		topicToChanID: make(map[string]uint16),
		redisToTopic:  make(map[string]string),
	}

	// Register schemas and channels for all defined data streams.
	for i, ch := range channels {
		schemaID := uint16(i + 1)
		chanID := uint16(i + 1)

		schema := buildMcapSchema(schemaID, ch.sample)
		if err := w.WriteSchema(schema); err != nil {
			f.Close()
			return nil, err
		}

		channel := buildMcapChannel(chanID, schemaID, ch.topic)
		if err := w.WriteChannel(channel); err != nil {
			f.Close()
			return nil, err
		}

		l.topicToChanID[ch.topic] = chanID
		l.redisToTopic[ch.redisKey] = ch.topic
	}

	return l, nil
}

// Run subscribes to all configured Redis keys and writes incoming messages
// to the MCAP file until the context is cancelled.
func (l *Logger) Run(ctx context.Context) error {
	keys := make([]string, 0, len(l.redisToTopic))
	for k := range l.redisToTopic {
		keys = append(keys, k)
	}

	sub := l.rdb.Subscribe(ctx, keys...)
	defer sub.Close()

	msgCh := sub.Channel()
	log.Printf("[mcap-logger] subscribed to %v", keys)

	var seq uint32
	for {
		select {
		case <-ctx.Done():
			if err := l.writer.Close(); err != nil {
				log.Printf("[mcap-logger] writer close error: %v", err)
			}
			// Sync before close so the OS flushes the summary section to disk.
			if err := l.file.Sync(); err != nil {
				log.Printf("[mcap-logger] file sync error: %v", err)
			}
			return l.file.Close()

		case msg, ok := <-msgCh:
			if !ok {
				if err := l.writer.Close(); err != nil {
					log.Printf("[mcap-logger] writer close error: %v", err)
				}
				if err := l.file.Sync(); err != nil {
					log.Printf("[mcap-logger] file sync error: %v", err)
				}
				return l.file.Close()
			}

			topic, found := l.redisToTopic[msg.Channel]
			if !found {
				continue
			}
			chanID, found := l.topicToChanID[topic]
			if !found {
				continue
			}

			now := uint64(time.Now().UnixNano())
			if err := l.writer.WriteMessage(&mcap.Message{
				ChannelID:   chanID,
				Sequence:    seq,
				LogTime:     now,
				PublishTime: now,
				Data:        []byte(msg.Payload),
			}); err != nil {
				log.Printf("[mcap-logger] WriteMessage error on %s: %v", topic, err)
			}
			seq++
		}
	}
}
