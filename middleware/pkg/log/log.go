package log

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Level represents the severity of a log message.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger writes structured log lines to an output writer.
type Logger struct {
	mu      sync.Mutex
	out     io.Writer
	level   Level
	service string
}

// New creates a Logger for the given service name.
func New(service string, level Level) *Logger {
	return &Logger{
		out:     os.Stdout,
		level:   level,
		service: service,
	}
}

// NewWithWriter creates a Logger with a custom writer (useful for testing).
func NewWithWriter(service string, level Level, out io.Writer) *Logger {
	return &Logger{
		out:     out,
		level:   level,
		service: service,
	}
}

func (l *Logger) log(level Level, msg string) {
	if level < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	ts := time.Now().Format("2006-01-02T15:04:05.000Z07:00")
	fmt.Fprintf(l.out, "%s [%s] [%s] %s\n", ts, level, l.service, msg)
}

// Debug logs a debug-level message.
func (l *Logger) Debug(msg string) { l.log(LevelDebug, msg) }

// Info logs an info-level message.
func (l *Logger) Info(msg string) { l.log(LevelInfo, msg) }

// Warn logs a warning-level message.
func (l *Logger) Warn(msg string) { l.log(LevelWarn, msg) }

// Error logs an error-level message.
func (l *Logger) Error(msg string) { l.log(LevelError, msg) }

// Debugf logs a formatted debug-level message.
func (l *Logger) Debugf(format string, args ...any) { l.log(LevelDebug, fmt.Sprintf(format, args...)) }

// Infof logs a formatted info-level message.
func (l *Logger) Infof(format string, args ...any) { l.log(LevelInfo, fmt.Sprintf(format, args...)) }

// Warnf logs a formatted warning-level message.
func (l *Logger) Warnf(format string, args ...any) { l.log(LevelWarn, fmt.Sprintf(format, args...)) }

// Errorf logs a formatted error-level message.
func (l *Logger) Errorf(format string, args ...any) { l.log(LevelError, fmt.Sprintf(format, args...)) }
