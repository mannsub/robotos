package main

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	dpb "google.golang.org/protobuf/types/descriptorpb"
)

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
