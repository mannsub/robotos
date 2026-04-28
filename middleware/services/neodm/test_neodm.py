import pytest
import trio
from tasks import Poller, PhysicalStateEstimator, DecisionMaker, RobotState

# --- DecisionMaker ---


@pytest.mark.trio
async def test_decision_idle_by_default():
    state = RobotState()
    dm = DecisionMaker(state)
    await dm._decide()
    assert dm.current_decision.action == "IDLE"


@pytest.mark.trio
async def test_decision_stop_on_obstacle():
    state = RobotState(obstacle=True)
    dm = DecisionMaker(state)
    await dm._decide()
    assert dm.current_decision.action == "STOP"
    assert dm.current_decision.confidence == 1.0


@pytest.mark.trio
async def test_decision_dock_on_low_battery():
    state = RobotState(battery_pct=15.0)
    dm = DecisionMaker(state)
    await dm._decide()
    assert dm.current_decision.action == "DOCK"


@pytest.mark.trio
async def test_decision_navigate_when_navigating():
    state = RobotState(nav_state="NAVIGATING")
    dm = DecisionMaker(state)
    await dm._decide()
    assert dm.current_decision.action == "NAVIGATE"


@pytest.mark.trio
async def test_obstacle_priority_over_low_battery():
    state = RobotState(obstacle=True, battery_pct=5.0)
    dm = DecisionMaker(state)
    await dm._decide()
    assert dm.current_decision.action == "STOP"

# --- Poller ---


@pytest.mark.trio
async def test_poller_updates_timestamp():
    from unittest.mock import MagicMock
    from tasks.hal_client import HalSensorState
    state = RobotState()
    before = state.updated_at
    hal_client = MagicMock()
    hal_client.state = HalSensorState()
    poller = Poller(state, hal_client=hal_client)
    poller._rdb = MagicMock()
    poller._rdb.get.return_value = None
    poller._poll_sync()
    assert state.updated_at >= before
