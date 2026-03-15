#pragma once
#include "robotos/hal.hpp"
#include <chrono>
#include <cmath>
#include <random>

namespace robotos::hal::mock {

class MockSensorDriver : public SensorDriver {
public:
  bool init() override { return true; }
  bool read_imu(IMUData &out) override {
    auto t = elapsed_seconds();
    auto noise = [&]() { return dist_(rng_) * 0.02; };

    out.ax = std::sin(t * 0.5) * 0.1 + noise();
    out.ay = std::cos(t * 0.3) * 0.1 + noise();
    out.az = 9.81 + noise();
    out.gx = noise();
    out.gy = noise();
    out.gz = std::sin(t * 0.2) * 0.05 + noise();
    return true;
  }

private:
  double elapsed_seconds() {
    auto now = std::chrono::steady_clock::now();
    return std::chrono::duration<double>(now - start_).count();
  }

  std::chrono::steady_clock::time_point start_{
      std::chrono::steady_clock::now()};
  std::mt19937 rng_{std::random_device{}()};
  std::uniform_real_distribution<double> dist_{-0.5, 0.5};
};
} // namespace robotos::hal::mock