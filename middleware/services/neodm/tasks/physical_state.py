import trio
from .poller import RobotState, _periodic


class PhysicalStateEstimator:
    """
    Estimates robot's physical state at 25Hz.
    Phase 1: skeleton
    Phase 2: IMU-based posture estimation (is_held, fallen, docking)
    """

    def __init__(self, state: RobotState) -> None:
        self._state = state

    async def run(self) -> None:
        async for _ in _periodic(1 / 25):
            await self._estimate()

    async def _estimate(self) -> None:
        # Phase 2: self._state.is_held = self._detect_held()
        pass
