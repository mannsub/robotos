package log

import (
	"bytes"
	"strings"
	"testing"
)

func TestLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter("test", LevelWarn, &buf)

	l.Debug("should not appear")
	l.Info("should not appear")
	l.Warn("should appear")
	l.Error("should appear")

	out := buf.String()
	if strings.Contains(out, "DEBUG") {
		t.Error("DEBUG should be filtered out")
	}
	if strings.Contains(out, "INFO") {
		t.Error("INFO should be filtered out")
	}
	if !strings.Contains(out, "WARN") {
		t.Error("WARN should appear")
	}
	if !strings.Contains(out, "ERROR") {
		t.Error("ERROR should appear")
	}
}

func TestLogFormat(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter("navigation", LevelDebug, &buf)

	l.Info("started")

	out := buf.String()
	if !strings.Contains(out, "[INFO]") {
		t.Error("missing level in output")
	}
	if !strings.Contains(out, "[navigation]") {
		t.Error("missing service name in output")
	}
	if !strings.Contains(out, "started") {
		t.Error("missing message in output")
	}
}

func TestFormattedLog(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter("motion", LevelDebug, &buf)

	l.Infof("joint %d torque %.2f Nm", 3, 1.23)

	out := buf.String()
	if !strings.Contains(out, "joint 3 torque 1.23 Nm") {
		t.Errorf("unexpected output: %s", out)
	}
}
