package redisclient

import (
	"context"
	"testing"
	"time"
)

type testState struct {
	Action   string  `json:"action"`
	Battery  float32 `json:"battery"`
	Obstacle bool    `json:"obstacle"`
}

func TestSetAndGet(t *testing.T) {
	c := New(DefaultAddr)
	defer c.Close()

	ctx := context.Background()
	if err := c.Ping(ctx); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	state := testState{
		Action:   "NAVIGATE",
		Battery:  85.0,
		Obstacle: false,
	}

	if err := c.Set(ctx, KeyNeoDMState, state, StateTTL); err != nil {
		t.Fatal(err)
	}

	var got testState
	if err := c.Get(ctx, KeyNeoDMState, &got); err != nil {
		t.Fatal(err)
	}

	if got.Action != state.Action {
		t.Errorf("expected %s, got %s", state.Action, got.Action)
	}
	if got.Battery != state.Battery {
		t.Errorf("expected %f, got %f", state.Battery, got.Battery)
	}
	t.Logf("state: %+v", got)
}

func TestTTLExpiry(t *testing.T) {
	c := New(DefaultAddr)
	defer c.Close()

	ctx := context.Background()
	if err := c.Ping(ctx); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	state := testState{Action: "IDLE"}
	if err := c.Set(ctx, "test:ttl", state, 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	time.Sleep(200 * time.Millisecond)

	var got testState
	err := c.Get(ctx, "test:ttl", &got)
	if err == nil {
		t.Error("expected error after TTL expiry, got nil")
	}
	t.Logf("TTL expired as expected: %v", err)
}

func TestPublishSubscribe(t *testing.T) {
	c := New(DefaultAddr)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := c.Ping(ctx); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	sub := c.Subscribe(ctx, KeyNeoDMState)
	defer sub.Close()

	state := testState{Action: "STOP"}
	go func() {
		time.Sleep(100 * time.Millisecond)
		c.Publish(ctx, KeyNeoDMState, state)
	}()

	msg, err := sub.ReceiveMessage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("received: %s", msg.Payload)
}
