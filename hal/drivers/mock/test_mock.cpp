#include "mock_motor.hpp"
#include "mock_sensor.hpp"
#include <cassert>
#include <iostream>

int main() {
  // sensor test
  robotos::hal::mock::MockSensorDriver sensor;
  assert(sensor.init());

  robotos::hal::IMUData imu;
  assert(sensor.read_imu(imu));
  assert(imu.az > 9.0 && imu.az < 10.5); // gravity check
  std::cout << "[PASS] MockSensorDriver\n";

  // motor test
  robotos::hal::mock::MockMotorDriver motor(4);
  assert(motor.init());

  assert(motor.set_torque(0, 0.1));

  robotos::hal::JointState state;
  assert(motor.get_state(0, state));
  assert(state.position > 0.0);
  std::cout << "[PASS] MockMotorDriver\n";

  // out of range test
  assert(!motor.set_torque(10, 1.0));
  assert(!motor.get_state(10, state));
  std::cout << "[PASS] Out of range check\n";

  std::cout << "All tests passed.\n";
  return 0;
}