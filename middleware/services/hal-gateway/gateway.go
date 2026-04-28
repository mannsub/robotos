package halgateway

import (
	"context"
	"log"
	"net"
	"time"

	pb "github.com/mannsub/robotos/proto/v1/gen/go/v1"
	"google.golang.org/grpc"
)

const (
	DefaultAdd = ":50052"
	SensorHz   = 25
)

type Server struct {
	pb.UnimplementedHalGatewayServer
	safety *SafetyController
	sensor *SensorSimulator
}

func NewServer() *Server {
	return &Server{
		safety: NewSafetyController(),
		sensor: NewSensorSimulator(),
	}
}

func (s *Server) StreamSensorData(req *pb.SensorStreamRequest, stream pb.HalGateway_StreamSensorDataServer) error {
	hz := int(req.Hz)
	if hz <= 0 || hz > SensorHz {
		hz = SensorHz
	}
	interval := time.Second / time.Duration(hz)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("[hal-gateway] StreamSensorData started at %dHz", hz)
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-ticker.C:
			data := s.sensor.Read()
			if err := stream.Send(data); err != nil {
				return err
			}
		}
	}
}

func (s *Server) SendMotionCommand(stream pb.HalGateway_SendMotionCommandServer) error {
	for {
		cmd, err := stream.Recv()
		if err != nil {
			return stream.SendAndClose(&pb.MotionCommandAck{
				Success: true,
				Message: "stream closed",
			})
		}
		if err := s.safety.Validate(cmd); err != nil {
			log.Printf("[hal-gateway] safety violation: %v", err)
			return stream.SendAndClose(&pb.MotionCommandAck{
				Success: false,
				Message: err.Error(),
			})
		}
		log.Printf("[hal-gateway] motion command received: mode=%v", cmd.SafetyMode)
	}
}

func (s *Server) SyncMotion(stream pb.HalGateway_SyncMotionServer) error {
	ticker := time.NewTicker(time.Second / SensorHz)
	defer ticker.Stop()

	ctx := stream.Context()
	cmdCh := make(chan *pb.MotionCommand, 8)

	go func() {
		for {
			cmd, err := stream.Recv()
			if err != nil {
				close(cmdCh)
				return
			}
			cmdCh <- cmd
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case cmd, ok := <-cmdCh:
			if !ok {
				return nil
			}
			if err := s.safety.Validate(cmd); err != nil {
				log.Printf("[hal-gateway] safety violation: %v", err)
			}
		case <-ticker.C:
			data := s.sensor.Read()
			if err := stream.Send(data); err != nil {
				return err
			}
		}
	}
}

func Run(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	srv := grpc.NewServer()
	pb.RegisterHalGatewayServer(srv, NewServer())
	log.Printf("[hal-gateway] listening on %s", addr)
	return srv.Serve(lis)
}

func RunWithContext(ctx context.Context, addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	srv := grpc.NewServer()
	pb.RegisterHalGatewayServer(srv, NewServer())
	go func() {
		<-ctx.Done()
		srv.GracefulStop()
	}()
	log.Printf("[hal-gateway] listening on %s", addr)
	return srv.Serve(lis)
}
