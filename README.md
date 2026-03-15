# RobotOS

An open-source robot OS platform inspired by LOVOTOS.
Built on L4T (Jetson) / Debian slim (Raspberry Pi) with a Go microservices middleware.

## Stack

| Layer      | Technology        |
|------------|-------------------|
| OS         | L4T / Debian slim |
| Middleware | Go microservices  |
| HAL        | C++               |
| Cloud      | Go + Docker       |

## Supported Hardware

- NVIDIA Jetson Orin NX
- Raspberry Pi 4 / 5
- x86 embedded

## Getting Started
```bash
docker pull ghcr.io/mannsub/robotos/runtime:latest
docker run ghcr.io/mannsub/robotos/runtime:latest
```
