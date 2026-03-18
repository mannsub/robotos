package motion

import (
	"context"
	"encoding/json"

	"github.com/mannsub/robotos/pkg/bus"
	"github.com/mannsub/robotos/pkg/health"
	"github.com/mannsub/robotos/pkg/log"
)

const (
	topicJointCmd    = "robot/cmd/joints"
	topicJointState  = "robot/state/joints"
	topicMotionState = "robot/state/motion"
)

// JointCmd is the command received from navigation.
type JointCmd struct {
	ID     int     `json:"id"`
	Torque float64 `json:"torque"`
}

// JointState is the state published after applying a command.
type JointState struct {
	ID       int     `json:"id"`
	Position float64 `json:"position"`
	Torque   float64 `json:"torque"`
}

// MotionState is the overall motion service state.
type MotionState struct {
	Status    string  `json:"status"`
	NumJoints int     `json:"num_joints"`
	MaxTorque float64 `json:"max_torque"`
}

// MotorDriver is the interface motion service uses to control hardware.
// Phase 1: MockMotorDriver
// Phase 2: RpiMotorDriver / JetsonMotorDriver
type MotorDriver interface {
	Init() error
	SetTorque(id int, torque float64) error
	GetPosition(id int) float64
}

// Service is the motion control microservice.
type Service struct {
	bus    *bus.Bus
	health *health.Monitor
	logger *log.Logger
	driver MotorDriver
	ready  chan struct{}
}

// New creates a new motion Service with the given motor driver.
func New(b *bus.Bus, h *health.Monitor, driver MotorDriver) *Service {
	return &Service{
		bus:    b,
		health: h,
		logger: log.New("motion", log.LevelDebug),
		driver: driver,
		ready:  make(chan struct{}),
	}
}

// Ready returns a channel closed when the service is ready to receive.
func (s *Service) Ready() <-chan struct{} {
	return s.ready
}

// Run starts the motion service loop.
func (s *Service) Run(ctx context.Context) {
	if err := s.driver.Init(); err != nil {
		s.logger.Errorf("driver init failed: %v", err)
		return
	}

	s.health.Register("motion")
	cmdCh := s.bus.Sub(topicJointCmd, 10)

	close(s.ready)

	s.logger.Info("motion service started")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("motion service stopped")
			return

		case msg := <-cmdCh:
			var cmd JointCmd
			if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
				s.logger.Errorf("invalid joint cmd: %v", err)
				continue
			}
			s.applyCommand(cmd)
			s.health.Report("motion", "ok")
		}
	}
}

func (s *Service) applyCommand(cmd JointCmd) {
	if err := s.driver.SetTorque(cmd.ID, cmd.Torque); err != nil {
		s.logger.Errorf("set torque failed: id=%d torque=%.2f err=%v", cmd.ID, cmd.Torque, err)
		return
	}

	pos := s.driver.GetPosition(cmd.ID)

	state := JointState{
		ID:       cmd.ID,
		Position: pos,
		Torque:   cmd.Torque,
	}
	b, _ := json.Marshal(state)
	s.bus.Pub(topicJointState, b)

	s.publishMotionState(cmd.Torque)
}

func (s *Service) publishMotionState(torque float64) {
	ms := MotionState{
		Status:    "active",
		NumJoints: 4,
		MaxTorque: torque,
	}
	b, _ := json.Marshal(ms)
	s.bus.Pub(topicMotionState, b)
}
