package simulation

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/mannsub/robotos/pkg/bus"
	"github.com/mannsub/robotos/pkg/health"
	"github.com/mannsub/robotos/pkg/log"
)

const (
	topicIMU        = "sensor/imu"
	topicJointCmd   = "robot/cmd/joints"
	topicJointState = "robot/state/joints"
)

// IMUData is the JSON payload published to sensor/imu.
type IMUData struct {
	Ax float64 `json:"ax"`
	Ay float64 `json:"ay"`
	Az float64 `json:"az"`
	Gz float64 `json:"gz"`
}

// JointCmd is the JSON payload for joint torque commands.
type JointCmd struct {
	ID     int     `json:"id"`
	Torque float64 `json:"torque"`
}

// JointStateData is the JSON payload for joint state feedback.
type JointStateData struct {
	ID       int     `json:"id"`
	Position float64 `json:"position"`
	Torque   float64 `json:"torque"`
}

// Simulator ties together bus, mock HAL, health, and logger.
type Simulator struct {
	bus    *bus.Bus
	health *health.Monitor
	logger *log.Logger
	hz     int
	joints []jointState
	ready  chan struct{}
}

type jointState struct {
	position float64
	torque   float64
}

// New creates a Simulator with the given tick rate in Hz.
func New(hz int) *Simulator {
	return &Simulator{
		bus:    bus.New(),
		health: health.New(3 * time.Second),
		logger: log.New("simulator", log.LevelDebug),
		hz:     hz,
		joints: make([]jointState, 4),
		ready:  make(chan struct{}),
	}
}

// Run starts the simulation loop and blocks until ctx is cancelled.
func (s *Simulator) Run(ctx context.Context) {
	s.health.Register("sensor")
	s.health.Register("controller")

	ticker := time.NewTicker(time.Second / time.Duration(s.hz))
	defer ticker.Stop()

	cmdCh := s.bus.Sub(topicJointCmd, 10)

	close(s.ready) // signal ready

	s.logger.Infof("simulator started at %d Hz", s.hz)

	tick := 0
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("simulator stopped")
			return

		case cmd := <-cmdCh:
			var jc JointCmd
			if err := json.Unmarshal(cmd.Payload, &jc); err == nil {
				if jc.ID < len(s.joints) {
					s.joints[jc.ID].position += jc.Torque * 0.001
					s.joints[jc.ID].torque = jc.Torque
				}
			}
		case <-ticker.C:
			s.publishIMU(tick)
			s.publishJointStates()
			s.health.Report("sensor", "ok")
			s.health.Report("controller", "ok")
			tick++
		}
	}
}

// Ready returns a channel that is closed when the simulator is ready.
func (s *Simulator) Ready() <-chan struct{} {
	return s.ready
}

func (s *Simulator) publishIMU(tick int) {
	t := float64(tick) / float64(s.hz)
	data := IMUData{
		Ax: sin(t*0.5) * 0.1,
		Ay: cos(t*0.3) * 0.1,
		Az: 9.81,
		Gz: sin(t*0.2) * 0.05,
	}
	b, _ := json.Marshal(data)
	s.bus.Pub(topicIMU, b)
}

func (s *Simulator) publishJointStates() {
	for i, j := range s.joints {
		data := JointStateData{
			ID:       i,
			Position: j.position,
			Torque:   j.torque,
		}
		b, _ := json.Marshal(data)
		s.bus.Pub(topicJointState, b)
	}
}

// Bus returns the internal bus for external subscribers.
func (s *Simulator) Bus() *bus.Bus { return s.bus }

// IsHealthy returns true if all services are healthy.
func (s *Simulator) IsHealthy() bool { return s.health.IsHealthy() }

func sin(x float64) float64 { return math.Sin(x) }
func cos(x float64) float64 { return math.Cos(x) }
