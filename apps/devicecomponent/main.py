import asyncio
from fastapi import FastAPI, WebSocket
from fastapi.middleware.cors import CORSMiddleware
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession
from sqlalchemy.orm import sessionmaker, declarative_base
from sqlalchemy import Column, Integer, String, JSON, DateTime, ForeignKey
from sqlalchemy.future import select
from datetime import datetime
import json
import os

DATABASE_URL = os.getenv("DATABASE_URL", "postgresql+asyncpg://postgres:postgres@smarthome-postgres:5432/smarthome")

engine = create_async_engine(DATABASE_URL, echo=True)
SessionLocal = sessionmaker(engine, expire_on_commit=False, class_=AsyncSession)
Base = declarative_base()


class Device(Base):
    __tablename__ = "devices"
    id = Column(Integer, primary_key=True, index=True)
    type = Column(String)
    model = Column(String)
    status = Column(String)
    home_id = Column(Integer)
    name = Column(String, unique=True, index=True)
    description = Column(String)
    ip = Column(String)
    port = Column(Integer)


class Telemetry(Base):
    __tablename__ = "telemetry"
    id = Column(Integer, primary_key=True, index=True)
    device_id = Column(Integer, ForeignKey("devices.id"))
    timestamp = Column(DateTime, default=datetime.utcnow)
    parameter = Column(String)
    value = Column(String)
    unit = Column(String)

app = FastAPI()

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

async def init_db():
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    async with SessionLocal() as session:
        # Ensure default devices exist (примерные значения для обязательных полей)
        default_devices = [
            {
                "name": "bulb",
                "type": "bulb",
                "model": "bulb-v1",
                "status": "online",
                "home_id": 1,
                "description": "Smart bulb",
                "ip": "192.168.0.10",
                "port": 10001
            },
            {
                "name": "gate",
                "type": "gate",
                "model": "gate-v1",
                "status": "online",
                "home_id": 1,
                "description": "Smart gate",
                "ip": "192.168.0.11",
                "port": 10002
            }
        ]
        for dev in default_devices:
            result = await session.execute(select(Device).where(Device.name == dev["name"]))
            if not result.scalar():
                session.add(Device(**dev))
        await session.commit()

@app.on_event("startup")
async def on_startup():
    await init_db()
    asyncio.create_task(ws_telemetry_listener())

@app.get("/devices")
async def get_devices():
    async with SessionLocal() as session:
        result = await session.execute(select(Device))
        return [
            {
                "id": d.id,
                "type": d.type,
                "model": d.model,
                "status": d.status,
                "home_id": d.home_id,
                "name": d.name,
                "description": d.description,
                "ip": d.ip,
                "port": int(d.port) if d.port is not None else None
            }
            for d in result.scalars()
        ]

@app.get("/devices/{device_id}")
async def get_device(device_id: int):
    async with SessionLocal() as session:
        device = await session.get(Device, device_id)
        if not device:
            return {"error": "Device not found"}
        return {
            "id": device.id,
            "type": device.type,
            "model": device.model,
            "status": device.status,
            "home_id": device.home_id,
            "name": device.name,
            "description": device.description,
            "ip": device.ip,
            "port": int(device.port) if device.port is not None else None
        }

@app.get("/devices/{device_id}/telemetry")
async def get_telemetry(device_id: int):
    async with SessionLocal() as session:
        result = await session.execute(
            select(Telemetry).where(Telemetry.device_id == device_id).order_by(Telemetry.timestamp.desc())
        )
        return [
            {
                "id": t.id,
                "device_id": t.device_id,
                "timestamp": t.timestamp.isoformat() if t.timestamp else None,
                "parameter": t.parameter,
                "value": t.value,
                "unit": t.unit
            }
            for t in result.scalars()
        ]

# WebSocket telemetry listener (devicecontroller -> this service)
async def ws_telemetry_listener():
    ws_url = os.getenv("DEVICECONTROLLER_WS_URL", "ws://devicecontroller:41200/ws")
    while True:
        try:
            async with WebSocketClient(ws_url) as ws:
                async for msg in ws:
                    await handle_telemetry(msg)
        except Exception as e:
            print(f"WebSocket error: {e}")
            await asyncio.sleep(5)

# Minimal async WebSocket client (using websockets lib)
import websockets
class WebSocketClient:
    def __init__(self, url):
        self.url = url
    async def __aenter__(self):
        self.conn = await websockets.connect(self.url)
        return self
    async def __aexit__(self, exc_type, exc, tb):
        await self.conn.close()
    def __aiter__(self):
        return self
    async def __anext__(self):
        msg = await self.conn.recv()
        return msg

async def handle_telemetry(msg):
    try:
        data = json.loads(msg)
        print(f"[Telemetry] Получена телеметрия: {data}")
        device_id = data.get("device_id")
        async with SessionLocal() as session:
            device = await session.get(Device, device_id)
            if device:
                if data.get("timestamp"):
                    ts = datetime.fromisoformat(data["timestamp"])
                else:
                    ts = datetime.utcnow()
                telemetry = Telemetry(
                    device_id=device.id,
                    timestamp=ts,
                    parameter=data.get("parameter", "unknown"),
                    value=str(data.get("value", "")),
                    unit=data.get("unit", "")
                )
                session.add(telemetry)
                await session.commit()
    except Exception as e:
        print(f"Telemetry handle error: {e}")
