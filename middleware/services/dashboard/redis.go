package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"
	v1 "github.com/mannsub/robotos/proto/v1/gen/go/v1"
	"google.golang.org/protobuf/proto"
)

type outMsg struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

func broadcast(h *hub, msgType string, data any) {
	b, err := json.Marshal(outMsg{Type: msgType, Data: data})
	if err != nil {
		return
	}
	h.broadcast <- b
}

func subscribeRedis(addr string, h *hub) {
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	ctx := context.Background()

	// Bridge: forward browser commands to Redis
	h.publishCmd = func(msgType string, raw []byte) {
		switch msgType {
		case "set_goal":
			rdb.Publish(ctx, "nav:goal", raw)
		case "set_obstacle":
			rdb.Publish(ctx, "nav:obstacle", raw)
		case "reset_map":
			rdb.Publish(ctx, "nav:reset", raw)
		case "reset_robot":
			rdb.Publish(ctx, "nav:reset_robot", raw)
		case "set_maze":
			rdb.Publish(ctx, "nav:maze", raw)
		}
	}

	sub := rdb.Subscribe(ctx, "neodm:state", "sensor:data", "hal:motion", "nav:state")
	defer sub.Close()

	log.Printf("[dashboard] subscribed to Redis at %s", addr)

	for msg := range sub.Channel() {
		switch msg.Channel {
		case "neodm:state":
			var pb v1.NeoDMState
			if err := proto.Unmarshal([]byte(msg.Payload), &pb); err != nil {
				continue
			}
			broadcast(h, "neodm", map[string]any{
				"decision":  pb.GetDecision(),
				"loop_hz":   pb.GetLoopHz(),
				"timestamp": pb.GetTimestamp(),
				"emotion": map[string]any{
					"label":   pb.GetEmotion().GetEyeState(),
					"valence": pb.GetEmotion().GetValence(),
					"arousal": pb.GetEmotion().GetArousal(),
				},
			})

		case "sensor:data":
			var pb v1.SensorData
			if err := proto.Unmarshal([]byte(msg.Payload), &pb); err != nil {
				continue
			}
			broadcast(h, "sensor", map[string]any{
				"timestamp_us": pb.GetTimestampUs(),
				"imu": map[string]any{
					"accel_x": pb.GetImu().GetAccelX(),
					"accel_y": pb.GetImu().GetAccelY(),
					"accel_z": pb.GetImu().GetAccelZ(),
					"gyro_x":  pb.GetImu().GetGyroX(),
					"gyro_y":  pb.GetImu().GetGyroY(),
					"gyro_z":  pb.GetImu().GetGyroZ(),
				},
				"battery": map[string]any{
					"pct":         pb.GetBattery().GetPct(),
					"voltage":     pb.GetBattery().GetVoltage(),
					"is_charging": pb.GetBattery().GetIsCharging(),
				},
				"joint_state": map[string]any{
					"position": pb.GetJointState().GetPosition(),
					"velocity": pb.GetJointState().GetVelocity(),
					"torque":   pb.GetJointState().GetTorque(),
				},
			})

		case "hal:motion":
			var pb v1.MotionCommand
			if err := proto.Unmarshal([]byte(msg.Payload), &pb); err != nil {
				continue
			}
			broadcast(h, "motion", map[string]any{
				"joint_targets": pb.GetJointTargets(),
				"joint_torques": pb.GetJointTorques(),
				"safety_mode":   pb.GetSafetyMode().String(),
			})

		case "nav:state":
			// Published as JSON by the main robotos bridge
			broadcast(h, "nav", json.RawMessage(msg.Payload))
		}
	}
}
