package halgateway

import (
	"errors"
	"math"

	pb "github.com/mannsub/robotos/proto/v1/gen/go/v1"
)

const (
	maxTorque   = 10.0
	maxJointRad = math.Pi
)

type SafetyController struct{}

func NewSafetyController() *SafetyController {
	return &SafetyController{}
}

func (s *SafetyController) Validate(cmd *pb.MotionCommand) error {
	if cmd.SafetyMode == pb.SafetyMode_SAFETY_MODE_TORQUE_OFF {
		return nil
	}
	for i, t := range cmd.JointTorques {
		if math.Abs(float64(t)) > maxTorque {
			return errors.New("torque limit exceeded on joint " + string(rune('0'+i)))
		}
	}
	for i, p := range cmd.JointTargets {
		if math.Abs(float64(p)) > maxJointRad {
			return errors.New("range limit exceede on joint " + string(rune('0'+i)))
		}
	}
	return nil
}
