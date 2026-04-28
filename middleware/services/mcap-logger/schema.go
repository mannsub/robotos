package main

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	dpb "google.golang.org/protobuf/types/descriptorpb"

	"github.com/foxglove/mcap/go/mcap"

	v1 "github.com/mannsub/robotos/proto/v1/gen/go/v1"
)

// channelDef pairs a Redis key, MCAP topic, and a sample proto message
// used to extract the FileDescriptorSet for schema registration.
type channelDef struct {
	redisKey string
	topic    string
	sample   proto.Message
}

// channels defines all protobuf data streams to be logged.
var channels = []channelDef{
	{"sensor:data", "/sensor", &v1.SensorData{}},
	{"neodm:state", "/neodm/state", &v1.NeoDMState{}},
	{"hal:motion", "/motion_command", &v1.MotionCommand{}},
}

// jsonChannelDef pairs a Redis key with an MCAP topic for JSON-encoded streams.
type jsonChannelDef struct {
	redisKey string
	topic    string
	schema   string // JSON Schema string
}

// jsonChannels defines JSON-encoded streams (e.g. nav:state published by Go bridge).
var jsonChannels = []jsonChannelDef{
	{
		redisKey: "nav:state",
		topic:    "/nav/state",
		schema:   `{"title":"NavState","type":"object","properties":{"status":{"type":"string"},"current_x":{"type":"number"},"current_y":{"type":"number"},"goal_x":{"type":"number"},"goal_y":{"type":"number"},"distance":{"type":"number"},"path":{"type":"array"}}}`,
	},
}

// buildMcapJSONSchema constructs an mcap.Schema for a JSON-encoded channel.
func buildMcapJSONSchema(id uint16, name, jsonSchema string) *mcap.Schema {
	return &mcap.Schema{
		ID:       id,
		Name:     name,
		Encoding: "jsonschema",
		Data:     []byte(jsonSchema),
	}
}

// buildMcapJSONChannel constructs an mcap.Channel for a JSON-encoded stream.
func buildMcapJSONChannel(id uint16, schemaID uint16, topic string) *mcap.Channel {
	return &mcap.Channel{
		ID:              id,
		SchemaID:        schemaID,
		Topic:           topic,
		MessageEncoding: "json",
		Metadata:        map[string]string{},
	}
}

// collectFileDeps recursively walks proto file imports (depth-first) and
// appends each FileDescriptorProto exactly once, dependencies before dependents.
// Foxglove requires all transitive imports to be present in the FileDescriptorSet.
func collectFileDeps(fd protoreflect.FileDescriptor, seen map[string]bool, out *[]*dpb.FileDescriptorProto) {
	if seen[fd.Path()] {
		return
	}
	seen[fd.Path()] = true
	for i := 0; i < fd.Imports().Len(); i++ {
		collectFileDeps(fd.Imports().Get(i), seen, out)
	}
	*out = append(*out, protodesc.ToFileDescriptorProto(fd))
}

// buildSchemaData serializes a FileDescriptorSet that includes the file
// descriptor of the given message and all of its transitive proto imports.
// Foxglove requires the full closure so it can resolve cross-file type references.
func buildSchemaData(msg proto.Message) []byte {
	fd := msg.ProtoReflect().Descriptor().ParentFile()
	seen := make(map[string]bool)
	var files []*dpb.FileDescriptorProto
	collectFileDeps(fd, seen, &files)
	fds := &dpb.FileDescriptorSet{File: files}
	data, _ := proto.Marshal(fds)
	return data
}

// schemaName returns the fully-qualified protobuf message name,
// which Foxglove uses to match schema to message type.
func schemaName(msg proto.Message) string {
	return string(msg.ProtoReflect().Descriptor().FullName())
}

// buildMcapSchema constructs an mcap.Schema for the given proto message.
// The schema ID is assigned by the caller.
func buildMcapSchema(id uint16, msg proto.Message) *mcap.Schema {
	return &mcap.Schema{
		ID:       id,
		Name:     schemaName(msg),
		Encoding: "protobuf",
		Data:     buildSchemaData(msg),
	}
}

// buildMcapChannel constructs an mcap.Channel referencing the given schema ID.
func buildMcapChannel(id uint16, schemaID uint16, topic string) *mcap.Channel {
	return &mcap.Channel{
		ID:              id,
		SchemaID:        schemaID,
		Topic:           topic,
		MessageEncoding: "protobuf",
		Metadata:        map[string]string{},
	}
}
