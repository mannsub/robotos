package halgateway

import (
	"math"
	"sync"
	"time"

	pb "github.com/mannsub/robotos/proto/v1/gen/go/v1"
)

const jointCount = 6

type SensorSimulator struct {
	mu sync.Mutex
	t  float64
}

func NewSensorSimulator() *SensorSimulator {
	return &SensorSimulator{}
}

func (s *SensorSimulator) Read() *pb.SensorData {
	s.mu.Lock()
	s.t += 1.0 / SensorHz
	t := s.t
	s.mu.Unlock()

	positions := make([]float32, jointCount)
	velocities := make([]float32, jointCount)
	torques := make([]float32, jointCount)
	for i := range positions {
		positions[i] = float32(math.Sin(t + float64(i)*0.5))
		velocities[i] = float32(math.Cos(t + float64(i)*0.5))
		torques[i] = 0.0
	}

	return &pb.SensorData{
		JointState: &pb.JointState{
			Position: positions,
			Velocity: velocities,
			Torque:   torques,
		},
		Imu: &pb.IMU{
			AccelX: 0.0,
			AccelY: 0.0,
			AccelZ: 9.81,
			GyroX:  0.0,
			GyroY:  0.0,
			GyroZ:  0.0,
		},
		Battery: &pb.Battery{
			Pct:        80.0,
			Voltage:    12.0,
			IsCharging: false,
		},
		Contact: &pb.ContactState{
			Touched:  false,
			IsHeld:   false,
			Obstacle: false,
			Cliff:    false,
		},
		TimestampUs: time.Now().UnixMicro(),
	}
}
