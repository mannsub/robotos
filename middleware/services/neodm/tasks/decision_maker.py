import trio
from dataclasses import dataclass
from .poller import RobotState, _periodic
from .hal_client import HalGatewayClient, HalSensorState


@dataclass
class Decision:
    action: str = "IDLE"  # "NAVIGATE" | "IDLE" | "STOP" | "DOCK"
    target_goal: str = ""
    confidence: float = 1.0
    reason: str = ""


class DecisionMaker:
    """
    Decides next robot action at 25Hz based on RobotState.
    Phase 1: simple rule-based decisions
    Phase 2: emotion model + ML-based decisions
    """

    def __init__(self, state: RobotState, hal_client: HalGatewayClient | None = None) -> None:
        self._state = state
        self._hal = hal_client
        self._decision = Decision()

    @property
    def current_decision(self) -> Decision:
        return self._decision

    async def run(self) -> None:
        async for _ in _periodic(1 / 25):
            await self._decide()

    async def _decide(self) -> None:
        s = self._state
        hal = self._hal.state if self._hal else None

        obstacle = (hal.obstacle if hal else False) or s.obstacle
        battery_pct = hal.battery_pct if hal else s.battery_pct

        if obstacle:
            self._decision = Decision(
                action="STOP",
                confidence=1.0,
                reason="obstacle detected",
            )
        elif battery_pct < 20.0:
            self._decision = Decision(
                action="DOCK",
                confidence=0.9,
                reason=f"low battery: {s.battery_pct:.1f}%",
            )
        elif s.nav_state == "NAVIGATING":
            self._decision = Decision(
                action="NAVIGATE",
                confidence=0.8,
                reason="navigation in progress",
            )
        else:
            self._decision = Decision(
                action="IDLE",
                confidence=1.0,
                reason="no active goal",
            )
