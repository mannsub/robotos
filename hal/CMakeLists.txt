cmake_minimum_required(VERSION 3.20)
project(robotos_hal CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Header-only library
add_library(robotos_hal INTERFACE)
target_include_directories(robotos_hal INTERFACE include)

# Mock driver test
add_executable(test_mock
    drivers/mock/test_mock.cpp
)
target_link_libraries(test_mock PRIVATE robotos_hal)