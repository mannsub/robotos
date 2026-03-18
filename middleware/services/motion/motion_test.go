package motion

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/mannsub/robotos/pkg/bus"
	"github.com/mannsub/robotos/pkg/health"
)

func TestMotionAppliesCommand(t *testing.T) {
	b := bus.New()
	h := health.New(3 * time.Second)
	driver := NewMockDriver(4)
	svc := New(b, h, driver)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stateCh := b.Sub(topicJointState, 10)

	go svc.Run(ctx)
	<-svc.Ready()

	cmd := JointCmd{ID: 0, Torque: 5.0}
	payload, _ := json.Marshal(cmd)
	b.Pub(topicJointCmd, payload)

	select {
	case msg := <-stateCh:
		var state JointState
		if err := json.Unmarshal(msg.Payload, &state); err != nil {
			t.Fatalf("failed to unmarshal state: %v", err)
		}
		if state.ID != 0 {
			t.Errorf("expected joint ID 0, got %d", state.ID)
		}
		if state.Position <= 0 {
			t.Errorf("expected positive position, got %.4f", state.Position)
		}
		t.Logf("joint 0 position: %.4f torque: %.2f", state.Position, state.Torque)
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for joint state")
	}
}

func TestMotionHealthReport(t *testing.T) {
	b := bus.New()
	h := health.New(3 * time.Second)
	driver := NewMockDriver(4)
	svc := New(b, h, driver)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go svc.Run(ctx)
	<-svc.Ready()

	cmd := JointCmd{ID: 1, Torque: 3.0}
	payload, _ := json.Marshal(cmd)
	b.Pub(topicJointCmd, payload)

	time.Sleep(100 * time.Millisecond)

	if !h.IsHealthy() {
		t.Fatalf("expected motion service to be healthy")
	}
}

func TestMotionDriverInitError(t *testing.T) {
	b := bus.New()
	h := health.New(3 * time.Second)
	driver := NewMockDriver(4).WithInitError(errors.New("hardware not found"))
	svc := New(b, h, driver)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run should return early on init error without closing ready
	done := make(chan struct{})
	go func() {
		svc.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
		t.Log("service exited on init error as expected")
	case <-time.After(time.Second):
		t.Fatal("timeout: service should have exited on init error")
	}
}

func TestMotionPublishesMotionState(t *testing.T) {
	b := bus.New()
	h := health.New(3 * time.Second)
	driver := NewMockDriver(4)
	svc := New(b, h, driver)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	motionStateCh := b.Sub(topicMotionState, 10)

	go svc.Run(ctx)
	<-svc.Ready()

	cmd := JointCmd{ID: 2, Torque: 8.0}
	payload, _ := json.Marshal(cmd)
	b.Pub(topicJointCmd, payload)

	select {
	case msg := <-motionStateCh:
		var ms MotionState
		if err := json.Unmarshal(msg.Payload, &ms); err != nil {
			t.Fatalf("failed to unmarshal motion state: %v", err)
		}
		if ms.Status != "active" {
			t.Errorf("expected status 'active', got '%s'", ms.Status)
		}
		t.Logf("motion state: status=%s max_torque=%.2f", ms.Status, ms.MaxTorque)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for motion state")
	}
}
