import json
import os
import time
import trio
import redis
from dataclasses import dataclass, field
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from tasks.hal_client import HalGatewayClient

REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379")


@dataclass
class RobotState:
    nav_state: str = "IDLE"
    battery_pct: float = 100.0
    is_held: bool = False
    obstacle: bool = False
    updated_at: float = field(default_factory=time.monotonic)


class Poller:
    """
    Collects robot state at 25Hz from Redis (nav:state) and HalGatewayClient.

    Data sources:
      - nav:state  (Redis pub/sub, published by the Go robotos bridge)
      - HalGatewayClient.state (battery, obstacle, is_held from HAL Gateway gRPC)
    """

    def __init__(self, state: RobotState, hal_client: "HalGatewayClient") -> None:
        self._state = state
        self._hal = hal_client
        self._rdb = redis.from_url(REDIS_URL, decode_responses=True)

    async def run(self) -> None:
        async for _ in _periodic(1 / 25):
            await trio.to_thread.run_sync(self._poll_sync, cancellable=True)

    def _poll_sync(self) -> None:
        # Pull latest nav:state from Redis (GET, not subscribe — low-latency snapshot)
        raw = self._rdb.get("nav:state")
        if raw:
            try:
                data = json.loads(raw)
                self._state.nav_state = data.get("status", "IDLE").upper()
            except (json.JSONDecodeError, AttributeError):
                pass

        # Copy sensor state from HalGatewayClient (updated by its own async stream)
        hal = self._hal.state
        self._state.battery_pct = hal.battery_pct
        self._state.obstacle = hal.obstacle
        self._state.is_held = hal.is_held
        self._state.updated_at = time.monotonic()


async def _periodic(interval: float):
    """Yields at a fixed interval, accounting for execution time."""
    while True:
        start = trio.current_time()
        yield
        elapsed = trio.current_time() - start
        sleep_for = interval - elapsed
        if sleep_for > 0:
            await trio.sleep(sleep_for)
