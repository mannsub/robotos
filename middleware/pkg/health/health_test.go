package health

import (
	"testing"
	"time"
)

func TestRegisterAndReport(t *testing.T) {
	m := New(time.Second)
	m.Register("navigation")

	m.Report("navigation", "ok")

	if !m.IsHealthy() {
		t.Fatal("expected healthy after report")
	}
}

func TestTimeout(t *testing.T) {
	m := New(50 * time.Millisecond)
	m.Register("motion")

	m.Report("motion", "ok")
	time.Sleep(100 * time.Millisecond)

	if m.IsHealthy() {
		t.Fatal("expected unhealthy after timeout")
	}
}

func TestUnknownIsNotHealthy(t *testing.T) {
	m := New(time.Second)
	m.Register("perception")

	if m.IsHealthy() {
		t.Fatal("expected not healthy when status in unknown")
	}
}

func TestMultipleServices(t *testing.T) {
	m := New(time.Second)
	m.Register("navigation")
	m.Register("motion")

	m.Report("navigation", "ok")
	// motion not reported

	if m.IsHealthy() {
		t.Fatal("expected unhealthy when one service has not reported")
	}
}
