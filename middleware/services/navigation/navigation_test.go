package navigation

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mannsub/robotos/pkg/bus"
	"github.com/mannsub/robotos/pkg/health"
)

func TestSimpleLinearPlanner(t *testing.T) {
	p := &SimpleLinearPlanner{}
	path := p.Plan(Point{0, 0}, Point{10, 10})

	if len(path) == 0 {
		t.Fatal("expected non-empty path")
	}
	last := path[len(path)-1]
	if last.X != 10 || last.Y != 10 {
		t.Errorf("expected goal (10, 10), got (%.2f,%.2f)", last.X, last.Y)
	}
	t.Logf("path length: %d waypoints", len(path))
}

func TestNavigationReachesGoal(t *testing.T) {
	b := bus.New()
	h := health.New(3 * time.Second)
	svc := New(b, h)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stateCh := b.Sub(topicNavState, 20)
	go svc.Run(ctx)

	svc.Ready() // wait for service to start

	// send goal
	goal := GoalMsg{X: 5.0, Y: 5.0}
	payload, _ := json.Marshal(goal)
	b.Pub(topicGoal, payload)

	// wait for navigation state updates
	var lastState NavState
	timeout := time.After(2 * time.Second)
	for {
		select {
		case msg := <-stateCh:
			json.Unmarshal(msg.Payload, &lastState)
			if lastState.Distance < 0.1 {
				t.Logf("reached goal, distance: %.4f", lastState.Distance)
				return
			}
		case <-timeout:
			t.Fatalf("timeout, last distance: %.4f", lastState.Distance)
		}
	}
}

func TestNavigationHealthReport(t *testing.T) {
	b := bus.New()
	h := health.New(3 * time.Second)
	svc := New(b, h)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go svc.Run(ctx)

	svc.Ready() // wait for service to start

	goal := GoalMsg{X: 1.0, Y: 1.0}
	payload, _ := json.Marshal(goal)
	b.Pub(topicGoal, payload)

	time.Sleep(300 * time.Millisecond)

	if !h.IsHealthy() {
		t.Fatal("expected navigation service to be healthy")
	}
}
