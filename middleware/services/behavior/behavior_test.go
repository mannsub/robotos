package behavior

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/mannsub/robotos/neodmpb"
	"google.golang.org/grpc"
)

// mockBus captures published messages for assertions.
type mockBus struct {
	mu      sync.Mutex
	topic   string
	payload []byte
}

func (m *mockBus) Pub(topic string, payload []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.topic = topic
	m.payload = payload
}

func (m *mockBus) Topic() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.topic
}

// mockNeoDMServer returns a fixed action for testing.
type mockNeoDMServer struct {
	neodmpb.UnimplementedNeoDMServer
	action string
	reason string
}

func (m *mockNeoDMServer) GetDecision(_ context.Context, _ *neodmpb.DecisionRequest) (*neodmpb.DecisionResponse, error) {
	return &neodmpb.DecisionResponse{
		Action:     m.action,
		Confidence: 1.0,
		Reason:     m.reason,
	}, nil
}

func startMockNeoDM(t *testing.T, action, reason string) string {
	t.Helper()
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	srv := grpc.NewServer()
	neodmpb.RegisterNeoDMServer(srv, &mockNeoDMServer{action: action, reason: reason})
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)
	return lis.Addr().String()
}

func TestBehaviorInitialStateIsIdle(t *testing.T) {
	bus := &mockBus{}
	svc := New(bus, "")
	if svc.State() != StateIdle {
		t.Errorf("expected IDLE, got %s", svc.State())
	}
}

func TestBehaviorTransitionsToNavigating(t *testing.T) {
	addr := startMockNeoDM(t, "NAVIGATE", "navigation in progress")
	bus := &mockBus{}
	svc := New(bus, addr)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go svc.Run(ctx)
	<-svc.Ready()
	time.Sleep(100 * time.Millisecond)

	if svc.State() != StateNavigating {
		t.Errorf("expected NAVIGATING, got %s", svc.State())
	}
	if bus.Topic() != "robot/state/behavior" {
		t.Errorf("expected topic robot/state/behavior, got %s", bus.Topic())
	}
}

func TestBehaviorTransitionsToStopped(t *testing.T) {
	addr := startMockNeoDM(t, "STOP", "obstacle detected")
	bus := &mockBus{}
	svc := New(bus, addr)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go svc.Run(ctx)
	<-svc.Ready()
	time.Sleep(100 * time.Millisecond)

	if svc.State() != StateStopped {
		t.Errorf("expected STOPPED, got %s", svc.State())
	}
}

func TestBehaviorPublishesStateToBus(t *testing.T) {
	addr := startMockNeoDM(t, "IDLE", "no active goal")
	bus := &mockBus{}
	svc := New(bus, addr)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go svc.Run(ctx)
	<-svc.Ready()
	time.Sleep(100 * time.Millisecond)

	if bus.Topic() != "robot/state/behavior" {
		t.Errorf("expected topic robot/state/behavior, got %s", bus.Topic())
	}
}
