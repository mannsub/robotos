from tasks import DecisionMaker, RobotState
from proto.python import neodm_pb2, neodm_pb2_grpc
import time
import sys
import os
import grpc
from concurrent import futures

sys.path.insert(0, os.path.dirname(__file__))


class NeoDMServicer(neodm_pb2_grpc.NeoDMServicer):
    def __init__(self, state: RobotState, decision_maker: DecisionMaker) -> None:
        self._state = state
        self_dmc = decision_maker

    def GetDecision(self, request, context):
        self._state.nav_state = request.nav_state or self._state.nav_state
        self._state.battery_pct = request.battery_pct or self._state.battery_pct
        self._state.is_held = request.is_held
        self._state.obstacle = request.obstacle

        d = self._dm.current_decision
        return neodm_pb2.DecisionResponse(
            action=d.action,
            target_goal=d.target_goal,
            confidence=d.confidence,
            reason=d.reason,
        )

    def StreamState(self, request, context):
        while context.is_active():
            d = self._dm.current_decision
            yield neodm_pb2.NeoDMState(
                emotion="NEUTRAL",  # Phase 2: emotion model
                decision=d.action,
                loop_hz=25.0,
                timestamp=int(time.time() * 1000),
            )
            time.sleep(1 / 25)

    def create_server(
            state: RobotState,
            decision_maker: DecisionMaker,
            port: int = 50051,
    ) -> grpc.Server:
        server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
        neodm_pb2_grpc.add_NeoDMServicer_to_server(
            NeoDMServicer(state, decision_maker), server
        )
        server.add_insecure_port(f"[::]:{port}")
        return server
