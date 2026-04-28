package main

import (
	_ "embed"
	"encoding/json"
	"math"
	"time"
)

//go:embed robot.urdf
var robotURDF string

// JSON channel IDs follow the protobuf channel IDs (streams has 3 entries → 1,2,3).
const (
	channelIDRobotDescription uint32 = 4
	channelIDTF               uint32 = 5
)

var jsonChannelAds = []ChannelAdvertisement{
	{
		ID:             channelIDRobotDescription,
		Topic:          "/robot_description",
		Encoding:       "json",
		SchemaName:     "std_msgs/String",
		Schema:         `{"title":"std_msgs/String","type":"object","properties":{"data":{"type":"string"}}}`,
		SchemaEncoding: "jsonschema",
	},
	{
		ID:             channelIDTF,
		Topic:          "/tf",
		Encoding:       "json",
		SchemaName:     "foxglove.FrameTransform",
		Schema:         `{"title":"foxglove.FrameTransform","type":"object","properties":{"timestamp":{"type":"object","properties":{"sec":{"type":"integer"},"nsec":{"type":"integer"}}},"parent_frame_id":{"type":"string"},"child_frame_id":{"type":"string"},"translation":{"type":"object","properties":{"x":{"type":"number"},"y":{"type":"number"},"z":{"type":"number"}}},"rotation":{"type":"object","properties":{"x":{"type":"number"},"y":{"type":"number"},"z":{"type":"number"},"w":{"type":"number"}}}}}`,
		SchemaEncoding: "jsonschema",
	},
}

// robotDescriptionPayload is the latched JSON payload for /robot_description.
var robotDescriptionPayload []byte

func init() {
	b, _ := json.Marshal(map[string]string{"data": robotURDF})
	robotDescriptionPayload = b
}

// navStateRedis matches the nav:state JSON published by the robotos bridge.
type navStateRedis struct {
	CurrentX float64      `json:"current_x"`
	CurrentY float64      `json:"current_y"`
	Path     [][2]float64 `json:"path,omitempty"`
}

type foxgloveTime struct {
	Sec  int64 `json:"sec"`
	Nsec int64 `json:"nsec"`
}

type vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type quaternion struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
	W float64 `json:"w"`
}

type frameTransform struct {
	Timestamp   foxgloveTime `json:"timestamp"`
	ParentFrame string       `json:"parent_frame_id"`
	ChildFrame  string       `json:"child_frame_id"`
	Translation vec3         `json:"translation"`
	Rotation    quaternion   `json:"rotation"`
}

// navStateToTF converts a nav:state JSON payload into a foxglove.FrameTransform
// JSON payload positioning base_link relative to world.
func navStateToTF(payload []byte) ([]byte, error) {
	var ns navStateRedis
	if err := json.Unmarshal(payload, &ns); err != nil {
		return nil, err
	}

	// Derive heading yaw from the first two path points when available.
	var yaw float64
	if len(ns.Path) >= 2 {
		dx := ns.Path[1][0] - ns.Path[0][0]
		dy := ns.Path[1][1] - ns.Path[0][1]
		if dx != 0 || dy != 0 {
			yaw = math.Atan2(dy, dx)
		}
	}

	// Yaw (rotation about Z) → unit quaternion.
	half := yaw / 2
	now := time.Now()
	tf := frameTransform{
		Timestamp:   foxgloveTime{Sec: now.Unix(), Nsec: int64(now.Nanosecond())},
		ParentFrame: "world",
		ChildFrame:  "base_link",
		// Lift robot so drive wheels (lowest at z=-0.155 from base_link) touch the floor.
		// TF z=0.155: wheel joint z=-0.09, radius=0.065 → lowest point z=-0.155 → floor=0.0
		Translation: vec3{X: ns.CurrentX, Y: ns.CurrentY, Z: 0.155},
		Rotation:    quaternion{X: 0, Y: 0, Z: math.Sin(half), W: math.Cos(half)},
	}
	return json.Marshal(tf)
}
