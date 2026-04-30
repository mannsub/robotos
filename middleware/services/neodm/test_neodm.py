import pytest
import trio
from tasks import Poller, PhysicalStateEstimator, DecisionMaker, RobotState
from tasks import EmotionEngine, EmotionState, EyeState, TouchZone, TouchEvent

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

# --- EmotionEngine ---


def _make_engine(emotion: EmotionState | None = None) -> tuple[RobotState, EmotionEngine]:
    state = RobotState()
    if emotion:
        state.emotion = emotion
    engine = EmotionEngine(state)
    engine._redis = type("_", (), {"publish_emotion": lambda *a, **kw: None})()
    return state, engine


def test_emotion_energy_drains_over_time():
    state, engine = _make_engine()
    initial = state.emotion.energy
    for _ in range(25):  # 1 second of ticks
        engine._tick()
    assert state.emotion.energy < initial


def test_emotion_valence_decays_toward_zero():
    state, engine = _make_engine(EmotionState(valence=1.0))
    for _ in range(25):
        engine._tick()
    assert state.emotion.valence < 1.0
    assert state.emotion.valence > 0.0


def test_emotion_arousal_drifts_to_half():
    state, engine = _make_engine(EmotionState(arousal=1.0))
    for _ in range(250):  # 10 seconds
        engine._tick()
    assert state.emotion.arousal < 1.0
    assert state.emotion.arousal > 0.5


def test_emotion_anxiety_decays():
    state, engine = _make_engine(EmotionState(anxiety=1.0))
    for _ in range(25):
        engine._tick()
    assert state.emotion.anxiety < 1.0


def test_touch_head_raises_valence_lowers_arousal():
    state, engine = _make_engine(EmotionState(valence=0.0, arousal=0.5))
    state.touch_events.append(TouchEvent(zone=TouchZone.HEAD))
    engine._tick()
    assert state.emotion.valence > 0.0
    assert state.emotion.arousal < 0.5


def test_touch_nose_poke_lowers_valence():
    state, engine = _make_engine(EmotionState(valence=0.0))
    state.touch_events.append(TouchEvent(zone=TouchZone.NOSE_POKE))
    engine._tick()
    assert state.emotion.valence < 0.0


def test_touch_events_cleared_after_tick():
    state, engine = _make_engine()
    state.touch_events.append(TouchEvent(zone=TouchZone.HEAD))
    engine._tick()
    assert len(state.touch_events) == 0


def test_eye_state_happy():
    e = EmotionState(valence=0.8, arousal=0.6, anxiety=0.1)
    assert e.eye_state == EyeState.HAPPY


def test_eye_state_sleepy():
    e = EmotionState(arousal=0.2)
    assert e.eye_state == EyeState.SLEEPY


def test_eye_state_anxious():
    e = EmotionState(anxiety=0.8, arousal=0.5)
    assert e.eye_state == EyeState.ANXIOUS


def test_eye_state_excited():
    e = EmotionState(valence=0.9, arousal=0.9, anxiety=0.1)
    assert e.eye_state == EyeState.EXCITED


def test_eye_state_sad():
    e = EmotionState(valence=-0.5, arousal=0.4)
    assert e.eye_state == EyeState.SAD


def test_emotion_values_clamped():
    state, engine = _make_engine(EmotionState(valence=0.9, arousal=0.5))
    for _ in range(10):
        state.touch_events.append(TouchEvent(zone=TouchZone.ROUGH))
    engine._tick()
    assert state.emotion.valence >= -1.0
    assert state.emotion.anxiety <= 1.0


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
