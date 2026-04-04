package navigation

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mannsub/robotos/pkg/bus"
	"github.com/mannsub/robotos/pkg/health"
)

// ---------------------------------------------------------------------------
// AStarPlanner unit tests
// ---------------------------------------------------------------------------

func TestAStarPlannerBasic(t *testing.T) {
	p := NewAStarPlanner(200, 200, 0.1)
	path := p.Plan(Point{0, 0}, Point{5, 5})

	if path == nil {
		t.Fatal("expected a path, got nil")
	}
	last := path[len(path)-1]
	if last.Distance(Point{5, 5}) > 0.2 {
		t.Errorf("path does not reach goal: last point (%.2f, %.2f)", last.X, last.Y)
	}
	t.Logf("path length: %d waypoints, last=(%.2f,%.2f)", len(path), last.X, last.Y)
}

func TestAStarPlannerObstacleAvoidance(t *testing.T) {
	p := NewAStarPlanner(200, 200, 0.1)

	// Build a vertical wall at x=2.5 from y=0 to y=4.9 by iterating over cells
	// directly to avoid floating-point accumulation gaps.
	wallCX, _ := p.toCell(2.5, 0)
	for cy := 0; cy < 50; cy++ {
		p.grid[cy][wallCX] = true
	}

	path := p.Plan(Point{0, 0}, Point{5, 2})
	if path == nil {
		t.Fatal("expected a path around obstacle, got nil")
	}

	// Verify no waypoint lands on a blocked cell.
	for _, wp := range path {
		cx, cy := p.toCell(wp.X, wp.Y)
		if p.grid[cy][cx] {
			t.Errorf("path passes through blocked cell at waypoint (%.2f, %.2f)", wp.X, wp.Y)
		}
	}
	last := path[len(path)-1]
	t.Logf("avoided obstacle: %d waypoints, last=(%.2f,%.2f)", len(path), last.X, last.Y)
}

func TestAStarPlannerNoPath(t *testing.T) {
	p := NewAStarPlanner(20, 20, 0.1)

	// surround the goal with obstacles on all sides
	for x := 0.9; x <= 1.1; x += 0.1 {
		for y := 0.9; y <= 1.1; y += 0.1 {
			p.SetObstacle(x, y)
		}
	}
	// block the goal cell itself
	p.SetObstacle(1.0, 1.0)

	path := p.Plan(Point{0, 0}, Point{1.0, 1.0})
	if path != nil {
		t.Errorf("expected nil path (goal blocked), got %d waypoints", len(path))
	}
}

func TestAStarPlannerSameCell(t *testing.T) {
	p := NewAStarPlanner(200, 200, 0.1)
	path := p.Plan(Point{3.0, 3.0}, Point{3.05, 3.05})
	if path == nil || len(path) == 0 {
		t.Fatal("expected non-nil path when start and goal are in the same cell")
	}
}

func TestAStarPlannerOutOfBounds(t *testing.T) {
	p := NewAStarPlanner(10, 10, 0.1) // 1 m × 1 m grid
	path := p.Plan(Point{0, 0}, Point{99, 99})
	if path != nil {
		t.Errorf("expected nil path for out-of-bounds goal, got %d waypoints", len(path))
	}
}

func TestAStarPlannerClearObstacle(t *testing.T) {
	p := NewAStarPlanner(200, 200, 0.1)
	p.SetObstacle(1.0, 1.0)

	// with obstacle at goal — no path
	if path := p.Plan(Point{0, 0}, Point{1.0, 1.0}); path != nil {
		t.Fatal("expected nil when goal is blocked")
	}

	p.ClearObstacle(1.0, 1.0)

	// after clearing — path should exist
	if path := p.Plan(Point{0, 0}, Point{1.0, 1.0}); path == nil {
		t.Fatal("expected valid path after clearing obstacle")
	}
}

// ---------------------------------------------------------------------------
// Service integration tests
// ---------------------------------------------------------------------------

func TestNavigationReachesGoal(t *testing.T) {
	b := bus.New()
	h := health.New(3 * time.Second)
	svc := New(b, h)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stateCh := b.Sub(topicNavState, 50)
	go svc.Run(ctx)
	<-svc.Ready()

	goal := GoalMsg{X: 5.0, Y: 5.0}
	payload, _ := json.Marshal(goal)
	b.Pub(topicGoal, payload)

	var lastState NavState
	timeout := time.After(5 * time.Second)
	for {
		select {
		case msg := <-stateCh:
			json.Unmarshal(msg.Payload, &lastState)
			if lastState.Distance < 0.2 {
				t.Logf("reached goal, distance: %.4f", lastState.Distance)
				return
			}
		case <-timeout:
			t.Fatalf("timeout, last distance: %.4f", lastState.Distance)
		}
	}
}

func TestNavigationObstacleUpdate(t *testing.T) {
	b := bus.New()
	h := health.New(3 * time.Second)
	svc := New(b, h)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go svc.Run(ctx)
	<-svc.Ready()

	// publish obstacle update
	obs := ObstacleMsg{X: 1.0, Y: 1.0, Blocked: true}
	payload, _ := json.Marshal(obs)
	b.Pub(topicObstacle, payload)

	time.Sleep(50 * time.Millisecond)

	cx, cy := svc.planner.toCell(1.0, 1.0)
	if !svc.planner.grid[cy][cx] {
		t.Error("expected obstacle to be set after ObstacleMsg")
	}
}

func TestNavigationHealthReport(t *testing.T) {
	b := bus.New()
	h := health.New(3 * time.Second)
	svc := New(b, h)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go svc.Run(ctx)
	<-svc.Ready()

	goal := GoalMsg{X: 1.0, Y: 1.0}
	payload, _ := json.Marshal(goal)
	b.Pub(topicGoal, payload)

	time.Sleep(500 * time.Millisecond)

	if !h.IsHealthy() {
		t.Fatal("expected navigation service to be healthy")
	}
}
