#pragma once
#include <cstdint>
#include <string>

namespace robotos::hal {

// IMU sensor data
struct IMUData {
  double ax, ay, az; // acceleration m/s²
  double gx, gy, gz; // angular velocity rad/s
};

// Joint state
struct JointState {
  double position; // rad
  double velocity; // rad/s
  double torque;   // Nm
};

// Abstract sensor driver interface
class SensorDriver {
public:
  virtual ~SensorDriver() = default;
  virtual bool init() = 0;
  virtual bool read_imu(IMUData &out) = 0;
};

// Abstract motor driver interface
class MotorDriver {
public:
  virtual ~MotorDriver() = default;
  virtual bool init() = 0;
  virtual bool set_torque(uint8_t id, double torque_nm) = 0;
  virtual bool get_state(uint8_t id, JointState &out) = 0;
};
} // namespace robotos::hal