package bus

import (
	"testing"
	"time"
)

func TestPubSub(t *testing.T) {
	b := New()
	ch := b.Sub("sensor/imu", 10)

	b.Pub("sensor/imu", []byte(`{"az":999.99}`))

	select {
	case msg := <-ch:
		t.Logf("received: topic=%s payload=%s", msg.Topic, msg.Payload)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	b := New()
	ch1 := b.Sub("robot/state", 10)
	ch2 := b.Sub("robot/state", 10)

	b.Pub("robot/state", []byte(`{"status":"active"}`))

	for i, ch := range []<-chan Message{ch1, ch2} {
		select {
		case msg := <-ch:
			t.Logf("subscriber %d received: %s", i+1, msg.Payload)
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d timed out", i+1)
		}
	}
}

func TestSlowConsumerDrop(t *testing.T) {
	b := New()
	ch := b.Sub("test/drop", 1) // buffer size 1

	// publish 2 messages to a buffer-1 channel
	b.Pub("test/drop", []byte("msg1"))
	b.Pub("test/drop", []byte("msg2"))

	msg := <-ch
	t.Logf("received: %s (second message dropped as expected)", msg.Payload)
}

func TestIntentionalFailure(t *testing.T) {
	t.Fatal("intentional failure to verify CI blocking")
}
