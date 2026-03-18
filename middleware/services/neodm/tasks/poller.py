import time
import trio
from dataclasses import dataclass, field


@dataclass
class RobotState:
    nav_state: str = "IDLE"  # from navigation service
    battery_pct: float = 100.0
    is_held: bool = False  # Phase 2: posture estimation
    obstacle: bool = False
    updated_at: float = field(default_factory=time.monotonic)


class Poller:
    """
    Collects robot state at 25Hz.
    Phase 1: skeleton (mock data)
    Phase 2: fetch from navigation / sensor services via gRPC
    """

    def __init__(self, state: RobotState) -> None:
        self._state = state

    async def run(self) -> None:
        async for _ in _periodic(1 / 25):
            await self._poll()

    async def _poll(self) -> None:
        # Phase 2: fetch real data from other services
        self._state.updated_at = time.monotonic()


async def _periodic(interval: float):
    """Same role as trio_util.periodic"""
    while True:
        start = trio.current_time()
        yield
        elapsed = trio.current_time() - start
        sleep_for = interval - elapsed
        if sleep_for > 0:
            await trio.sleep(sleep_for)
