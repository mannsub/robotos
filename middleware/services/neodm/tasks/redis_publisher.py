import os
import sys

import redis

# Add the generated proto package directory to the Python path so that
# the v1 package (and its cross-file imports) can be resolved correctly.
_PROTO_GEN_DIR = os.path.join(
    os.path.dirname(__file__),
    "../../../../proto/v1/gen/python",
)
sys.path.insert(0, os.path.abspath(_PROTO_GEN_DIR))

from v1 import neodm_pb2  # noqa: E402  (import after sys.path setup)
from v1 import sensor_pb2  # noqa: E402

REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379")

KEY_NEODM_STATE = "neodm:state"
CHANNEL_NEODM_STATE = "neodm:state"

STATE_TTL = 5


class RedisPublisher:
    """
    Publishes NeoDM state to Redis at 25Hz as serialized protobuf bytes.
    Other services (Go) can subscribe to the neodm:state channel and
    deserialize the payload as robotos.v1.NeoDMState.
    """

    def __init__(self) -> None:
        # decode_responses=False is required to handle binary protobuf payloads.
        self._client = redis.from_url(REDIS_URL, decode_responses=False)
        self._running = False

    def publish_state(self, action: str, reason: str, confidence: float) -> None:
        state = neodm_pb2.NeoDMState(
            decision=action,
            emotion=neodm_pb2.Emotion(
                label=reason,
                valence=confidence,
            ),
        )
        data = state.SerializeToString()
        self._client.setex(KEY_NEODM_STATE, STATE_TTL, data)
        self._client.publish(CHANNEL_NEODM_STATE, data)

    def ping(self) -> bool:
        try:
            return self._client.ping()
        except Exception:
            return False

    def close(self) -> None:
        self._client.close()
