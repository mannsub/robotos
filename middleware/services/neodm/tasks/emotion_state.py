from dataclasses import dataclass, field
from enum import Enum


class EyeState(str, Enum):
    NEUTRAL = "NEUTRAL"
    HAPPY = "HAPPY"
    EXCITED = "EXCITED"
    SLEEPY = "SLEEPY"
    ANXIOUS = "ANXIOUS"
    SAD = "SAD"
    SURPRISED = "SURPRISED"


class TouchZone(str, Enum):
    HEAD = "HEAD"
    BELLY = "BELLY"
    CHIN = "CHIN"
    BACK = "BACK"
    NOSE_POKE = "NOSE_POKE"
    ROUGH = "ROUGH"


@dataclass
class TouchEvent:
    zone: TouchZone
    intensity: float = 1.0  # 0.0–1.0


@dataclass
class EmotionState:
    energy: float = 80.0   # 0–100, drains over time
    valence: float = 0.0   # -1.0–1.0, displeasure ~ pleasure
    arousal: float = 0.5   # 0.0–1.0, calm ~ excited
    anxiety: float = 0.3   # 0.0–1.0, familiar ~ unfamiliar

    @property
    def eye_state(self) -> EyeState:
        if self.arousal < 0.25:
            return EyeState.SLEEPY
        if self.anxiety > 0.6:
            return EyeState.ANXIOUS
        if self.valence > 0.7 and self.arousal > 0.7:
            return EyeState.EXCITED
        if self.valence > 0.5 and self.arousal > 0.4:
            return EyeState.HAPPY
        if self.valence < -0.3:
            return EyeState.SAD
        return EyeState.NEUTRAL

    def clamp(self) -> None:
        self.energy = max(0.0, min(100.0, self.energy))
        self.valence = max(-1.0, min(1.0, self.valence))
        self.arousal = max(0.0, min(1.0, self.arousal))
        self.anxiety = max(0.0, min(1.0, self.anxiety))
