# RobotOS

An open-source robot OS platform inspired by LOVOTOS (GROOVE X).  
Go microservices middleware + Python NeoDM behavior engine, targeting NVIDIA Jetson.

## Architecture

```
robotos/
├── middleware/                  # Go microservices
│   ├── cmd/robotos/             # entrypoint (wires all services)
│   ├── pkg/
│   │   ├── bus/                 # in-process pub/sub message bus
│   │   ├── health/              # service health monitor
│   │   └── log/                 # centralized logger
│   └── services/
│       ├── behavior/            # gRPC client bridge to NeoDM
│       ├── motion/              # motor control
│       ├── navigation/          # Nav2-style waypoint navigation
│       └── neodm/               # Python NeoDM (trio 25Hz loops + gRPC server)
│           ├── tasks/
│           │   ├── poller.py            # robot state collection
│           │   ├── physical_state.py    # posture estimation
│           │   └── decision_maker.py    # rule-based decisions
│           ├── proto/           # gRPC interface definitions
│           └── main.py          # trio nursery entrypoint
├── hal/                         # C++ HAL interfaces + mock drivers
├── docker/
│   ├── go.Dockerfile            # Go services (arm64)
│   ├── neodm.Dockerfile         # Python NeoDM (arm64)
│   └── docker-compose.yml
└── os/                          # kernel config, rootfs (Phase 3)
```

## Stack

| Layer      | Technology                           |
| ---------- | ------------------------------------ |
| OS         | L4T (Jetson) / Debian slim           |
| Behavior   | Python + trio (NeoDM, 25Hz loops)    |
| Middleware | Go microservices + gRPC              |
| HAL        | C++ (LLVM style)                     |
| Viz        | Foxglove + MCAP + Protobuf (Phase 2) |
| Cloud      | Go + Docker                          |

## Service Communication

```
[Python NeoDM]  ←──gRPC──→  [Go behavior]  ──bus──→  [Go motion / navigation]
  25Hz loops                  state machine            robot/state/behavior
  decision maker              IDLE/NAVIGATING          robot/cmd/joints
  posture estimator           STOPPED/DOCKING
```

## Supported Hardware

- NVIDIA Jetson Orin NX (primary)
- Raspberry Pi 4 / 5 (Phase 3)

## Getting Started

```bash
# Run all services
cd docker
docker compose up

# Run tests (Go)
cd middleware
go test ./...

# Run tests (Python NeoDM)
cd middleware/services/neodm
python3 -m pytest test_neodm.py -v
```

## Roadmap

| Phase | Status         | Description                                              |
| ----- | -------------- | -------------------------------------------------------- |
| 1     | 🔄 In Progress | Full skeleton: nav, motion, behavior, NeoDM, docker, CI  |
| 2     | ⬜ Planned     | A\* navigation + MCAP/Foxglove visualization + dashboard |
| 3     | ⬜ Planned     | Jetson / Raspberry Pi HAL porting + RT kernel            |
| 4     | ⬜ Planned     | Cloud fleet management + OTA update                      |
