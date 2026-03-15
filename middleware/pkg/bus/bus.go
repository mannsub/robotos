package bus

import (
	"sync"
	"time"
)

// Message is the unit of communication between service.
type Message struct {
	Topic   string
	Payload []byte
	Ts      time.Time
}

// Bus is an in-process pub/sub message bus for inter-service communication
type Bus struct {
	mu   sync.RWMutex
	subs map[string][]chan Message
}

// New creates a new Bus instance.
func New() *Bus {
	return &Bus{subs: make(map[string][]chan Message)}
}

// Pub publishes a message to the given topic.
func (b *Bus) Pub(topic string, payload []byte) {
	msg := Message{Topic: topic, Payload: payload, Ts: time.Now()}
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs[topic] {
		select {
		case ch <- msg:
		default: // drop slow consumers to preserve real-time priority
		}
	}
}

// Sub subscribes to a topic and returns a receive-only channel.
func (b *Bus) Sub(topic string, buf int) <-chan Message {
	ch := make(chan Message, buf)
	b.mu.Lock()
	b.subs[topic] = append(b.subs[topic], ch)
	b.mu.Unlock()
	return ch
}
