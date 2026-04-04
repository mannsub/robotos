package main

// Foxglove webSocket Protocol v2 server-side message type.
// reference: https://github.com/foxglove/ws-protocol/blob/main/docs/spec.md

const foxgloveSubprotocol = "foxglove.websocket.v1"

// ServerInfo is sent to the client upon connection.
type ServerInfo struct {
	Op           string   `json:"op"`
	Name         string   `json:"name"`
	Capabilities []string `json:"capabilities"`
}

// ChannelAdvertisement describes a single channel the server will publish.
type ChannelAdvertisement struct {
	ID             uint32 `json:"id"`
	Topic          string `json:"topic"`
	Encoding       string `json:"encoding"`
	SchemaName     string `json:"schemaName"`
	Schema         string `json:"schema"` // base64-encoded FileDescriptorSet
	SchemaEncoding string `json:"schemaEncoding"`
}

// Advertise is sent after ServerInfo to announce available channels.
type Advertise struct {
	Op       string                 `json:"op"`
	Channels []ChannelAdvertisement `json:"channels"`
}

// ClientSubscription is one entry in a Subscribe request from the client.
// The client chooses its own subscription ID; the server must echo it back
// in binary MessageData frames.
type ClientSubscription struct {
	ID        uint32 `json:"id"`
	ChannelID uint32 `json:"channelId"`
}

// ClientSubscribe is sent by the client to start receiving data on channels.
type ClientSubscribe struct {
	Op            string               `json:"op"`
	Subscriptions []ClientSubscription `json:"subscriptions"`
}

// ClientUnsubscribe is sent by the client to stop receiving data.
type ClientUnsubscribe struct {
	Op              string   `json:"op"`
	SubscriptionIDs []uint32 `json:"subscriptionIds"`
}
