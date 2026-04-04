package halgateway

import (
	"context"
	"net"
	"testing"
	"time"

	pb "github.com/mannsub/robotos/proto/v1/gen/go/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func startTestServer(t *testing.T) (pb.HalGatewayClient, func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer()
	pb.RegisterHalGatewayServer(srv, NewServer())
	go srv.Serve(lis)

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	client := pb.NewHalGatewayClient(conn)
	return client, func() {
		conn.Close()
		srv.Stop()
	}
}

func TestStreamSensorData(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	stream, err := client.StreamSensorData(ctx, &pb.SensorStreamRequest{Hz: 25})
	if err != nil {
		t.Fatal(err)
	}

	data, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}

	if data.JointState == nil {
		t.Error("expected JointState, got nil")
	}
	if data.Imu == nil {
		t.Error("expected IMU, got nil")
	}
	if data.TimestampUs == 0 {
		t.Error("expected non-zero timestamp")
	}
	t.Logf("received sensor data: imu.accel_z=%2.f", data.Imu.AccelZ)
}

func TestSafetyValidate(t *testing.T) {
	s := NewSafetyController()

	t.Run("valid command", func(t *testing.T) {
		cmd := &pb.MotionCommand{
			JointTorques: []float32{1.0, -1.0},
			JointTargets: []float32{0.5, -0.5},
			SafetyMode:   pb.SafetyMode_SAFETY_MODE_NORMAL,
		}
		if err := s.Validate(cmd); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("torque exceede", func(t *testing.T) {
		cmd := &pb.MotionCommand{
			JointTorques: []float32{999.0},
			SafetyMode:   pb.SafetyMode_SAFETY_MODE_NORMAL,
		}
		if err := s.Validate(cmd); err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("range exceeded", func(t *testing.T) {
		cmd := &pb.MotionCommand{
			JointTargets: []float32{999.0},
			SafetyMode:   pb.SafetyMode_SAFETY_MODE_NORMAL,
		}
		if err := s.Validate(cmd); err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("torque off bypasses checks", func(t *testing.T) {
		cmd := &pb.MotionCommand{
			JointTorques: []float32{999.0},
			SafetyMode:   pb.SafetyMode_SAFETY_MODE_TORQUE_OFF,
		}
		if err := s.Validate(cmd); err != nil {
			t.Errorf("expected nil for torque off, got %v", err)
		}
	})
}
