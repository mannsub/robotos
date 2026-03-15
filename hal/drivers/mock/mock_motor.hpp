#pragma once
#include "robotos/hal.hpp"
#include <vector>

namespace robotos::hal::mock {

class MockMotorDriver : public MotorDriver {
public:
  explicit MockMotorDriver(uint8_t num_joints) : states_(num_joints) {}

  bool init() override { return true; }

  bool set_torque(uint8_t id, double torque_nm) override {
    if (id >= states_.size()) {
      return false;
    }
    states_[id].position += torque_nm * 0.001;
    states_[id].torque = torque_nm;
    return true;
  }

  bool get_state(uint8_t id, JointState &out) override {
    if (id >= states_.size()) {
      return false;
    }
    out = states_[id];
    return true;
  }

private:
  std::vector<JointState> states_;
};
} // namespace robotos::hal::mock