package navigation

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mannsub/robotos/pkg/bus"
	"github.com/mannsub/robotos/pkg/health"
	"github.com/mannsub/robotos/pkg/log"
)

const (
	topicGoal     = "robot/goal"
	topicObstacle = "robot/map/obstacle"
	topicJointCmd = "robot/cmd/joints"
	topicNavState = "robot/state/navigation"
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

// NavState is the JSON payload published to robot/state/navigation.
type NavState struct {
	Status   string  `json:"status"`
	CurrentX float64 `json:"current_x"`
	CurrentY float64 `json:"current_y"`
	GoalX    float64 `json:"goal_x"`
	GoalY    float64 `json:"goal_y"`
	Distance float64 `json:"distance"`
}

// JointCmd is the joint torque command payload.
type JointCmd struct {
	ID     int     `json:"id"`
	Torque float64 `json:"torque"`
}

// Service is the navigation microservice.
type Service struct {
	bus     *bus.Bus
	health  *health.Monitor
	logger  *log.Logger
	planner *AStarPlanner
	pos     Point
	ready   chan struct{}
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
func (s *Service) Run(ctx context.Context) {
	s.health.Register("navigation")
	goalCh := s.bus.Sub(topicGoal, 10)
	obsCh := s.bus.Sub(topicObstacle, 32)

	close(s.ready)

	s.logger.Info("navigation service started")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("navigation service stopped")
			return

		case msg := <-obsCh:
			var obs ObstacleMsg
			if err := json.Unmarshal(msg.Payload, &obs); err != nil {
				s.logger.Errorf("invalid obstacle message: %v", err)
				continue
			}
			if obs.Blocked {
				s.planner.SetObstacle(obs.X, obs.Y)
			} else {
				s.planner.ClearObstacle(obs.X, obs.Y)
			}

		case msg := <-goalCh:
			var goal GoalMsg
			if err := json.Unmarshal(msg.Payload, &goal); err != nil {
				s.logger.Errorf("invalid goal message: %v", err)
				continue
			}
			s.navigateTo(ctx, Point{X: goal.X, Y: goal.Y})
			s.health.Report("navigation", "ok")
		}
	}
}

// Ready returns a channel that is closed when the service is ready to receive.
func (s *Service) Ready() <-chan struct{} {
	return s.ready
}

func (s *Service) navigateTo(ctx context.Context, goal Point) {
	path := s.planner.Plan(s.pos, goal)
	if path == nil {
		s.logger.Errorf("no path to (%.2f, %.2f)", goal.X, goal.Y)
		return
	}
	s.logger.Infof("navigating to (%.2f, %.2f) via %d waypoints", goal.X, goal.Y, len(path))

	for _, waypoint := range path {
		select {
		case <-ctx.Done():
			return
		default:
		}

		dx := waypoint.X - s.pos.X
		dy := waypoint.Y - s.pos.Y
		torque := (dx + dy) * 10.0

		cmd := JointCmd{ID: 0, Torque: torque}
		b, _ := json.Marshal(cmd)
		s.bus.Pub(topicJointCmd, b)

		s.pos = waypoint
		s.publishState(goal)
		s.health.Report("navigation", "ok")

		time.Sleep(10 * time.Millisecond)
	}

	s.logger.Infof("reached goal (%.2f, %.2f)", goal.X, goal.Y)
}

func (s *Service) publishState(goal Point) {
	state := NavState{
		Status:   "navigating",
		CurrentX: s.pos.X,
		CurrentY: s.pos.Y,
		GoalX:    goal.X,
		GoalY:    goal.Y,
		Distance: s.pos.Distance(goal),
	}
	b, _ := json.Marshal(state)
	s.bus.Pub(topicNavState, b)
}
