module github.com/mannsub/robotos

go 1.25.6

require (
	google.golang.org/grpc v1.79.3
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/mannsub/robotos/neodmpb v0.0.0-00010101000000-000000000000 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
)

replace github.com/mannsub/robotos/neodmpb => ./services/neodm/proto/neodmpb
