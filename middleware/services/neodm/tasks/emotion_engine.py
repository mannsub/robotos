from .poller import RobotState, _periodic
from .emotion_state import EmotionState, TouchZone, TouchEvent
from .redis_publisher import RedisPublisher

_DT = 1 / 25  # seconds per tick at 25Hz

# Per-second decay rates
_ENERGY_DRAIN_PER_SEC = 2.0 / 60      # 2 units/min
_ENERGY_CHARGE_PER_SEC = 6.0 / 60     # 6 units/min when charging
_VALENCE_TAU = 120.0                   # valence returns to 0 over ~2 min
_AROUSAL_TAU = 60.0                    # arousal drifts to 0.5 over ~1 min
_ANXIETY_TAU = 300.0                   # anxiety decays over ~5 min

# (valence_delta, arousal_delta, anxiety_delta) per touch event at intensity=1.0
_TOUCH_EFFECTS: dict[TouchZone, tuple[float, float, float]] = {
    TouchZone.HEAD:      (+0.10, -0.05, -0.05),
    TouchZone.BELLY:     (+0.15, -0.10, -0.08),
    TouchZone.CHIN:      (+0.10, -0.15, -0.05),
    TouchZone.BACK:      (+0.08, -0.05, -0.03),
    TouchZone.NOSE_POKE: (-0.20, +0.10, +0.10),
    TouchZone.ROUGH:     (-0.30, +0.15, +0.20),
}


class EmotionEngine:
    """
    Updates EmotionState at 25Hz via temporal decay and touch event processing.
    Grounded in Russell's Circumplex Model (valence × arousal).
    """

    def __init__(self, state: RobotState) -> None:
        self._state = state
        self._redis = RedisPublisher()

    async def run(self) -> None:
        async for _ in _periodic(_DT):
            self._tick()

    def _tick(self) -> None:
        e = self._state.emotion

        # Temporal decay
        e.energy -= _ENERGY_DRAIN_PER_SEC * _DT
        if self._state.battery_pct < 20.0:
            e.energy += _ENERGY_CHARGE_PER_SEC * _DT

        e.valence -= e.valence * (_DT / _VALENCE_TAU)
        e.arousal += (0.5 - e.arousal) * (_DT / _AROUSAL_TAU)
        e.anxiety -= e.anxiety * (_DT / _ANXIETY_TAU)

        # Touch events
        events = self._state.touch_events[:]
        self._state.touch_events.clear()
        for event in events:
            self._apply_touch(e, event)

        e.clamp()

        try:
            self._redis.publish_emotion(
                energy=e.energy,
                valence=e.valence,
                arousal=e.arousal,
                anxiety=e.anxiety,
                eye_state=e.eye_state.value,
            )
        except Exception:
            pass

    def _apply_touch(self, e: EmotionState, event: TouchEvent) -> None:
        effect = _TOUCH_EFFECTS.get(event.zone)
        if effect is None:
            return
        v_delta, a_delta, ax_delta = effect
        s = event.intensity
        e.valence += v_delta * s
        e.arousal += a_delta * s
        e.anxiety += ax_delta * s
