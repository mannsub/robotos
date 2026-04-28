import trio
import grpc
from dataclasses import dataclass
from v1 import sensor_pb2, hal_pb2, hal_pb2_grpc


import os
HAL_GATEWAY_ADDR = os.environ.get("HAL_GATEWAY_ADDR", "hal-gateway:50052")


@dataclass
class HalSensorState:
    accel_z: float = 9.81
    battery_pct: float = 100.0
    obstacle: bool = False
    touched: bool = False
    is_held: bool = False


class HalGatewayClient:
    """
    Connects to hal-gateway and streams sensor data at 25Hz.
    Updates HalSensorState for NeoDM decision making.
    """

    def __init__(self, addr: str = HAL_GATEWAY_ADDR) -> None:
        self._addr = addr
        self.state = HalSensorState()
        self._running = False

    async def run(self) -> None:
        self._running = True
        while self._running:
            try:
                await self._stream()
            except Exception as e:
                print(f"[hal-client] connection error: {e}, retrying in 1s")
                await trio.sleep(1.0)

    async def _stream(self) -> None:
        async with grpc.aio.insecure_channel(self._addr) as channel:
            stub = hal_pb2_grpc.HalGatewayStub(channel)
            req = hal_pb2.SensorStreamRequest(hz=25)
            async for data in stub.StreamSensorData(req):
                self._update(data)

    def _update(self, data: sensor_pb2.SensorData) -> None:
        if data.imu:
            self.state.accel_z = data.imu.accel_z
        if data.battery:
            self.state.battery_pct = data.battery.pct
        if data.contact:
            self.state.obstacle = data.contact.obstacle
            self.state.touched = data.contact.touched
            self.state.is_held = data.contact.is_held

    def stop(self) -> None:
        self._running = False
