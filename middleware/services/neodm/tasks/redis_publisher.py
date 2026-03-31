import json
import os
import redis
import trio

REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379")

KEY_NEODM_STATE = "neodm:state"
KEY_NEODM_EMOTION = "neodm:emotion"
CHANNEL_NEODM_STATE = "neodm:state"

STATE_TTL = 5


class RedisPublisher:
    """
    Publishes NeoDM state to Redis at 25Hz.
    Other services (Go) can subcribe to neodm:state channel.
    """

    def __init__(self) -> None:
        self._client = redis.from_url(REDIS_URL, decode_responses=True)
        self._running = False

    def publish_state(self, action: str, reason: str, confidence: float) -> None:
        state = {
            "action": action,
            "reason": reason,
            "confidence": confidence,
        }
        data = json.dumps(state)
        self._client.setex(KEY_NEODM_STATE, STATE_TTL, data)
        self._client.publish(CHANNEL_NEODM_STATE, data)

    def ping(self) -> bool:
        try:
            return self._client.ping()
        except Exception:
            return False

    def close(self) -> None:
        self._client.close()
