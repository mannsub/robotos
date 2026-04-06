package navigation

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/mannsub/robotos/pkg/bus"
	"github.com/mannsub/robotos/pkg/health"
	"github.com/mannsub/robotos/pkg/log"
)

const (
	topicGoal       = "robot/goal"
	topicObstacle   = "robot/map/obstacle"
	topicMapBatch   = "robot/map/batch"
	topicReset      = "robot/map/reset"
	topicRobotReset = "robot/reset/robot"
	topicJointCmd   = "robot/cmd/joints"
	topicNavState   = "robot/state/navigation"
)

// GoalMsg is the JSON payload received on robot/goal.
type GoalMsg struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// ObstacleMsg is the JSON payload received on robot/map/obstacle.
type ObstacleMsg struct {
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Blocked bool    `json:"blocked"` // true = add obstacle, false = clear
}

// MazeBatchMsg is the JSON payload received on robot/map/batch (full maze upload).
type MazeBatchMsg struct {
	Obstacles []ObstacleMsg `json:"obstacles"`
}

// NavState is the JSON payload published to robot/state/navigation.
type NavState struct {
	Status   string       `json:"status"`
	CurrentX float64      `json:"current_x"`
	CurrentY float64      `json:"current_y"`
	GoalX    float64      `json:"goal_x"`
	GoalY    float64      `json:"goal_y"`
	Distance float64      `json:"distance"`
	Path     [][2]float64 `json:"path,omitempty"`
}

// JointCmd is the joint torque command payload.
type JointCmd struct {
	ID     int     `json:"id"`
	Torque float64 `json:"torque"`
}

// Service is the navigation microservice.
type Service struct {
	bus         *bus.Bus
	health      *health.Monitor
	logger      *log.Logger
	planner     *AStarPlanner
	mu          sync.RWMutex
	pos         Point
	currentPath []Point
	ready       chan struct{}
}

// New creates a new navigation Service backed by an A* planner.
// The default grid is 200×200 cells at 0.1 m/cell (20 m × 20 m).
func New(b *bus.Bus, h *health.Monitor) *Service {
	return &Service{
		bus:     b,
		health:  h,
		logger:  log.New("navigation", log.LevelDebug),
		planner: NewAStarPlanner(200, 200, 0.1),
		ready:   make(chan struct{}),
	}
}

// Run starts the navigation service loop.
//
// Design: a single goroutine drives everything — no goroutine spawning per
// goal. A ticker advances one waypoint every 10 ms. A new goal immediately
// replaces the current one and triggers a fresh A* plan; no cancellation
// races are possible.
func (s *Service) Run(ctx context.Context) {
	s.health.Register("navigation")
	goalCh       := s.bus.Sub(topicGoal, 10)
	obsCh        := s.bus.Sub(topicObstacle, 512)
	batchCh      := s.bus.Sub(topicMapBatch, 4)
	resetCh      := s.bus.Sub(topicReset, 4)
	robotResetCh := s.bus.Sub(topicRobotReset, 4)

	close(s.ready)
	s.logger.Info("navigation service started")

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	var (
		goal *Point  // active goal; nil = idle
		path []Point // planned waypoints
		step int     // next waypoint index
	)

	// replan computes a fresh A* path from the current position to goal.
	// Caller must ensure goal != nil.
	replan := func() {
		from := s.pos // single goroutine — no lock needed for read here

		// If the robot landed on a cell that was later marked as an obstacle
		// (happens during dynamic maze generation), snap to the nearest free cell.
		if s.planner.IsObstacle(from.X, from.Y) {
			if snapped, ok := s.planner.SnapToFree(from.X, from.Y); ok {
				s.mu.Lock()
				s.pos = snapped
				s.mu.Unlock()
				from = snapped
			}
		}

		newPath := s.planner.Plan(from, *goal)

		s.mu.Lock()
		s.currentPath = newPath
		s.mu.Unlock()

		if newPath == nil {
			s.logger.Errorf("no path to (%.2f, %.2f)", goal.X, goal.Y)
			path = nil
		} else {
			s.logger.Infof("path to (%.2f, %.2f): %d waypoints", goal.X, goal.Y, len(newPath))
			path = newPath
		}
		step = 0
		s.publishState(*goal)
	}

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("navigation service stopped")
			return

		case msg := <-batchCh:
			var batch MazeBatchMsg
			if err := json.Unmarshal(msg.Payload, &batch); err != nil {
				s.logger.Errorf("invalid maze batch: %v", err)
				continue
			}
			for _, o := range batch.Obstacles {
				if o.Blocked {
					s.planner.SetObstacle(o.X, o.Y)
				} else {
					s.planner.ClearObstacle(o.X, o.Y)
				}
			}
			s.logger.Infof("maze batch applied: %d obstacles", len(batch.Obstacles))
			if goal != nil {
				replan()
			}

		case <-resetCh:
			s.planner.Reset()
			goal = nil
			path = nil
			step = 0
			s.mu.Lock()
			s.currentPath = nil
			s.mu.Unlock()
			s.logger.Info("map reset")

		case <-robotResetCh:
			goal = nil
			path = nil
			step = 0
			s.mu.Lock()
			s.pos = Point{}
			s.currentPath = nil
			s.mu.Unlock()
			s.logger.Info("robot position reset to origin")

		case msg := <-obsCh:
			// Apply this obstacle then drain any remaining buffered obstacles
			// so we only call A* once per batch (maze generation can queue 500+).
			applyObs := func(raw []byte) {
				var o ObstacleMsg
				if err := json.Unmarshal(raw, &o); err != nil {
					return
				}
				if o.Blocked {
					s.planner.SetObstacle(o.X, o.Y)
				} else {
					s.planner.ClearObstacle(o.X, o.Y)
				}
			}
			applyObs(msg.Payload)
		drain:
			for {
				select {
				case m := <-obsCh:
					applyObs(m.Payload)
				default:
					break drain
				}
			}
			if goal != nil {
				replan()
			}

		case msg := <-goalCh:
			var gm GoalMsg
			if err := json.Unmarshal(msg.Payload, &gm); err != nil {
				s.logger.Errorf("invalid goal message: %v", err)
				continue
			}
			p := Point{X: gm.X, Y: gm.Y}
			goal = &p
			replan()
			s.health.Report("navigation", "ok")

		case <-ticker.C:
			if goal == nil || path == nil || step >= len(path) {
				continue
			}

			waypoint := path[step]
			step++

			dx := waypoint.X - s.pos.X
			dy := waypoint.Y - s.pos.Y
			torque := (dx + dy) * 10.0

			cmd := JointCmd{ID: 0, Torque: torque}
			b, _ := json.Marshal(cmd)
			s.bus.Pub(topicJointCmd, b)

			s.mu.Lock()
			s.pos = waypoint
			s.mu.Unlock()

			s.publishState(*goal)
			s.health.Report("navigation", "ok")

			if step >= len(path) {
				s.logger.Infof("reached goal (%.2f, %.2f)", goal.X, goal.Y)
				goal = nil
				path = nil
				s.mu.Lock()
				s.currentPath = nil
				s.mu.Unlock()
			}
		}
	}
}

// Ready returns a channel that is closed when the service is ready to receive.
func (s *Service) Ready() <-chan struct{} {
	return s.ready
}

func (s *Service) publishState(goal Point) {
	s.mu.RLock()
	pos := s.pos
	path := s.currentPath
	s.mu.RUnlock()

	var pathPairs [][2]float64
	for _, wp := range path {
		pathPairs = append(pathPairs, [2]float64{wp.X, wp.Y})
	}

	state := NavState{
		Status:   "navigating",
		CurrentX: pos.X,
		CurrentY: pos.Y,
		GoalX:    goal.X,
		GoalY:    goal.Y,
		Distance: pos.Distance(goal),
		Path:     pathPairs,
	}
	b, _ := json.Marshal(state)
	s.bus.Pub(topicNavState, b)
}
