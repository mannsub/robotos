package behavior

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/mannsub/robotos/neodmpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultNeoDMAddr = "localhost:50051"
	pollInterval     = 40 * time.Millisecond // ~25Hz
)

// BusPublisher is the interface for publishing messages to the bus.
type BusPublisher interface {
	Pub(topic string, payload []byte)
}

// Service is the behavior service.
// It polls NeoDM for decisions and publishes state to the bus.
type Service struct {
	mu        sync.Mutex
	state     State
	bus       BusPublisher
	neodmAddr string
	ready     chan struct{}
}

func New(bus BusPublisher, neodmAddr string) *Service {
	if neodmAddr == "" {
		neodmAddr = defaultNeoDMAddr
	}
	return &Service{
		state:     StateIdle,
		bus:       bus,
		neodmAddr: neodmAddr,
		ready:     make(chan struct{}),
	}
}

func (s *Service) Ready() <-chan struct{} {
	return s.ready
}

func (s *Service) Run(ctx context.Context) error {
	conn, err := grpc.NewClient(
		s.neodmAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := neodmpb.NewNeoDMClient(conn)
	close(s.ready)
	log.Printf("[behavior] connected to NeoDM at %s", s.neodmAddr)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.poll(ctx, client)
		}
	}
}

func (s *Service) poll(ctx context.Context, client neodmpb.NeoDMClient) {
	resp, err := client.GetDecision(ctx, &neodmpb.DecisionRequest{
		NavState: string(s.State()),
	})
	if err != nil {
		log.Printf("[behavior] GetDecision error: %v", err)
		return
	}

	event := Event(resp.Action)

	s.mu.Lock()
	next := nextState(s.state, event)

	if next != s.state {
		log.Printf("[behavior] state transition: %s -> %s (reason: %s)", s.state, next, resp.Reason)
		s.state = next
	}
	s.mu.Unlock()

	s.bus.Pub("robot/state/behavior", []byte(s.State()))
}

func (s *Service) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}
