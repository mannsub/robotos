from server import create_server
from tasks import Poller, PhysicalStateEstimator, DecisionMaker, RobotState, EmotionEngine
from tasks.hal_client import HalGatewayClient
import signal
import logging
import trio
import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../proto/v1/gen/python"))


logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
log = logging.getLogger("neodm")

GRPC_PORT = 50051


async def main() -> None:
    log.info("NeoDM starting up...")

    state = RobotState()
    hal_client = HalGatewayClient()
    poller = Poller(state, hal_client=hal_client)
    physical_state = PhysicalStateEstimator(state)
    decision_maker = DecisionMaker(state, hal_client=hal_client)
    emotion_engine = EmotionEngine(state)

    grpc_server = create_server(state, decision_maker, port=GRPC_PORT)
    grpc_server.start()
    log.info(f"gRPC server listening on : {GRPC_PORT}")

    shutdown = trio.Event()

    def _handel_signal(sig, frame):
        log.info(f"Received signal {sig}, shutting down...")
        shutdown.set()

    signal.signal(signal.SIGINT, _handel_signal)
    signal.signal(signal.SIGTERM, _handel_signal)

    try:
        async with trio.open_nursery() as nursery:
            nursery.start_soon(poller.run)
            nursery.start_soon(physical_state.run)
            nursery.start_soon(decision_maker.run)
            nursery.start_soon(emotion_engine.run)
            nursery.start_soon(hal_client.run)
            nursery.start_soon(_shutdown_watcher, shutdown, nursery.cancel_scope)
            log.info("All tasks started (25Hz loops running)")
    finally:
        grpc_server.stop(grace=1.0)
        log.info("NeoDM shutdown complete")


async def _shutdown_watcher(
        shutdown: trio.Event,
        cancel_scope: trio.CancelScope,
) -> None:
    await shutdown.wait()
    cancel_scope.cancel()

if __name__ == "__main__":
    trio.run(main)
