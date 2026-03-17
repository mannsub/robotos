package motion

// MockMotorDriver implements MotorDriver for testing.
type MockMotorDriver struct {
	positions []float64
	initErr   error
}

// NewMockDriver creates a MockMotorDriver with the given number of joints.
func NewMockDriver(numJoints int) *MockMotorDriver {
	return &MockMotorDriver{
		positions: make([]float64, numJoints),
	}
}

// WithInitError makes Init() return an error (for error path testing).
func (m *MockMotorDriver) WithInitError(err error) *MockMotorDriver {
	m.initErr = err
	return m
}

func (m *MockMotorDriver) Init() error {
	return m.initErr
}

func (m *MockMotorDriver) SetTorque(id int, torque float64) error {
	if id >= len(m.positions) {
		return nil
	}
	m.positions[id] += torque * 0.001
	return nil
}

func (m *MockMotorDriver) GetPosition(id int) float64 {
	if id >= len(m.positions) {
		return 0
	}
	return m.positions[id]
}
