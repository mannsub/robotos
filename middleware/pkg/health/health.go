package health

import (
	"sync"
	"time"
)

// Status represents the health state of a service.
type Status int

const (
	StatusUnknown Status = iota
	StatusHealthy
	StatusUnhealthy
)

func (s Status) String() string {
	switch s {
	case StatusHealthy:
		return "healthy"
	case StatusUnhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}

// ServiceHealth holds the health information of a single service.
type ServiceHealth struct {
	Name     string
	Status   Status
	LastSeen time.Time
	Message  string
}

// Monitor tracks the health of registered services.
type Monitor struct {
	mu       sync.RWMutex
	services map[string]*ServiceHealth
	timeout  time.Duration
}

// New creates a new Monitor with the given timeout duration.
// A service is considered unhealthy if it has not reported within timeout.
func New(timeout time.Duration) *Monitor {
	return &Monitor{
		services: make(map[string]*ServiceHealth),
		timeout:  timeout,
	}
}

// Register adds a service to the monitor.
func (m *Monitor) Register(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.services[name] = &ServiceHealth{
		Name:   name,
		Status: StatusUnknown,
	}
}

// Report marks a service as healthy with an optional message.
func (m *Monitor) Report(name, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if svc, ok := m.services[name]; ok {
		svc.Status = StatusHealthy
		svc.LastSeen = time.Now()
		svc.Message = message
	}
}

// Check evaluates all services and marks stale ones as unhealthy.
func (m *Monitor) Check() []ServiceHealth {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	result := make([]ServiceHealth, 0, len(m.services))

	for _, svc := range m.services {
		if svc.Status == StatusHealthy && now.Sub(svc.LastSeen) > m.timeout {
			svc.Status = StatusUnhealthy
			svc.Message = "timeout"
		}
		result = append(result, *svc)
	}
	return result
}

// IsHealthy returns true if all registered services are healthy.
func (m *Monitor) IsHealthy() bool {
	for _, svc := range m.Check() {
		if svc.Status != StatusHealthy {
			return false
		}
	}
	return true
}
