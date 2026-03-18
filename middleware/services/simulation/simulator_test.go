package simulation

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestSimulatorPublishesIMU(t *testing.T) {
	sim := New(100) // 100 Hz
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	imuCh := sim.Bus().Sub(topicIMU, 10)

	go sim.Run(ctx)
	<-sim.Ready()

	select {
	case msg := <-imuCh:
		var data IMUData
		if err := json.Unmarshal(msg.Payload, &data); err != nil {
			t.Fatalf("failed to unmarshal IMU data: %v", err)
		}
		if data.Az < 9.0 || data.Az > 10.5 {
			t.Errorf("unexpected Az value: %f", data.Az)
		}
		t.Logf("IMU: az=%.2f", data.Az)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for IMU data")
	}
}

func TestSimulatorHealthy(t *testing.T) {
	sim := New(100) // 100 Hz
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sim.Run(ctx)
	<-sim.Ready()

	time.Sleep(100 * time.Millisecond) // wait for health reports to accumulate

	if !sim.IsHealthy() {
		t.Fatal("expected simulator to be healthy")
	}
}

func TestSimulatorJointCmd(t *testing.T) {
	sim := New(100) // 100 Hz
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stateCh := sim.Bus().Sub(topicJointState, 20)
	go sim.Run(ctx)
	<-sim.Ready()

	// send torque command to joint 0
	cmd := JointCmd{ID: 0, Torque: 10.0}
	b, _ := json.Marshal(cmd)
	sim.Bus().Pub(topicJointCmd, b)

	// drain state messages and check joint 0
	for {
		select {
		case msg := <-stateCh:
			var state JointStateData
			if err := json.Unmarshal(msg.Payload, &state); err != nil {
				continue
			}
			if state.ID == 0 && state.Position > 0 {
				t.Logf("joint 0 position: %.4f", state.Position)
				return
			}
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for joint state update")
		}
	}
}
