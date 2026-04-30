# RobotOS

An open-source robot OS platform inspired by LOVOTOS (GROOVE X).  
Go microservices middleware + Python NeoDM behavior engine, targeting NVIDIA Jetson.

## Architecture

```
robotos/
├── middleware/
│   ├── cmd/
│   │   ├── robotos/             # entrypoint (wires all services)
│   │   └── hal-gateway/         # standalone HAL gRPC gateway
│   ├── pkg/
│   │   ├── bus/                 # in-process pub/sub message bus
│   │   ├── health/              # service health monitor
│   │   └── log/                 # centralized logger
│   └── services/
│       ├── behavior/            # gRPC client bridge to NeoDM
│       ├── dashboard/           # real-time web UI (Go + Svelte)
│       ├── foxglove-bridge/     # WebSocket bridge → Foxglove Studio
│       ├── hal-gateway/         # HAL gRPC server (sensor/motor)
│       ├── mcap-logger/         # MCAP recording (Protobuf + JSON)
│       ├── motion/              # motor control
│       ├── navigation/          # A* waypoint navigation
│       ├── neodm/               # Python NeoDM (trio 25Hz + gRPC)
│       │   ├── tasks/
│       │   │   ├── poller.py          # state collection (Redis + HAL)
│       │   │   ├── physical_state.py  # posture estimation
│       │   │   └── decision_maker.py  # rule-based decisions
│       │   └── main.py
│       └── simulation/          # maze simulator
├── hal/                         # C++ HAL interfaces + mock drivers
└── docker/
    ├── docker-compose.yaml
    ├── go.Dockerfile
    ├── neodm.Dockerfile
    ├── dashboard.Dockerfile
    ├── foxglove-bridge.Dockerfile
    ├── mcap-logger.Dockerfile
    ├── hal-gateway.Dockerfile
    └── jenkins/                 # Jenkins CI (JCasC, port 8090)
```

## Stack

| Layer      | Technology                              |
| ---------- | --------------------------------------- |
| OS         | L4T (Jetson) / Debian slim              |
| Behavior   | Python + trio (NeoDM, 25Hz loops)       |
| Middleware | Go microservices + Redis pub/sub        |
| HAL        | C++ + gRPC (hal-gateway)                |
| Navigation | A* planner, 200×200 grid (0.1 m/cell)   |
| Viz        | Foxglove Studio (URDF, TF, MCAP)        |
| Dashboard  | Go + Svelte (real-time 2D map + nav UI) |
| CI         | Jenkins (Docker, JCasC, Slack notify)   |

## Service Communication

```
[Browser Dashboard] ──WS──→ [Go dashboard] ──Redis──→ [Go robotos]
                                                            │
                    ┌───────────────────────────────────────┤
                    │                                       │
              [navigation]  ←──────────────────────  [behavior]
              [motion]                                      │
              [mcap-logger]                          [Python NeoDM]
              [foxglove-bridge] ──WS──→ [Foxglove]   ←─gRPC─┘
                                                      ←─gRPC─ [hal-gateway]
```

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

## Dashboard

```
http://<host>:8080
```

**Map tab**
- 🎯 **Goal mode** — click to set navigation target
- ⬛ **Obstacle mode** — draw walls
- 📏 **Line mode** — draw straight walls (Bresenham)
- 🎲 **Maze** — generate random DFS maze
- ✏️ **Erase** — remove obstacles

**Emotion Sim tab**

Interactive 3D emotion simulation using your robot's GLB model.

- Left-click the model to touch different zones (Ear, Head, Face, Arm, Belly, Back, Nose)
- Right-click for rough contact
- Each touch zone affects the emotion parameters (valence, arousal, anxiety, energy) in real time
- Emotion state is displayed in the top-right panel (NEUTRAL / HAPPY / EXCITED / SLEEPY / ANXIOUS / SAD)

## Robot Model Setup

Model assets are **not included** in this repository. You must supply your own 3D model.

```
middleware/services/foxglove-bridge/meshes/
├── <your-model>.glb   ← GLB file served to the Emotion Sim (via foxglove-bridge :8765)
└── <your-model>.stl   ← optional STL for Foxglove Studio URDF visualization
```

**Steps:**

1. Export your robot model as a GLB file (e.g. from Meshy AI, Blender, or any 3D tool)
2. Place it in `middleware/services/foxglove-bridge/meshes/` and update the filename reference in `EmotionSim.svelte` and `main.go`
3. *(Optional)* Export an STL version to the same folder for Foxglove Studio
4. Rebuild the foxglove-bridge container:
   ```bash
   cd docker && docker compose build foxglove-bridge && docker compose up -d foxglove-bridge
   ```

The touch zone hit-boxes are calibrated for a humanoid/character figure roughly 1.8 m tall — adjust the zone definitions in `EmotionSim.svelte` if your model has a different proportions.

## Foxglove Visualization

Connect Foxglove Studio to `ws://<host>:8765`

| Topic               | Type                    | Description              |
| ------------------- | ----------------------- | ------------------------ |
| `/sensor`           | Protobuf SensorData     | IMU, battery, contact    |
| `/neodm/state`      | Protobuf NeoDMState     | decision, emotion, Hz    |
| `/motion_command`   | Protobuf MotionCommand  | joint torque commands    |
| `/robot_description`| JSON std_msgs/String    | URDF model (latched)     |
| `/tf`               | JSON foxglove.FrameTransform | robot pose (world → base_link) |

## CI

Jenkins runs at `http://<host>:8090`

- Multibranch pipeline — auto-builds all branches and PRs
- Stages: Go test → Python test → Docker image build
- Slack notifications on success/failure

## Roadmap

| Phase | Status         | Description                                                                 |
| ----- | -------------- | --------------------------------------------------------------------------- |
| 1     | ✅ Done        | Skeleton: nav, motion, behavior, NeoDM, Docker, CI                          |
| 2     | ✅ Done        | A* navigation, dashboard, Foxglove/URDF/TF, MCAP logging, hal-gateway, maze |
| 2.5   | ✅ Done        | Emotion engine (4-axis), 3D emotion sim, touch zones, dashboard integration |
| 3     | ⬜ Planned     | Real hardware HAL (Jetson), SLAM, sensor fusion                             |
| 4     | ⬜ Planned     | Cloud fleet management + OTA update                                         |
