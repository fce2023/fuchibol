import asyncio
import socketio
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from app.api import auth, streams, channels, admin, restream
from app.utils.logging import setup_json_logging

# Inicializar logs estructurados
setup_json_logging()

# Create FastAPI Instance
fastapi_app = FastAPI(
    title="Fuchibol API",
    description="Backend API para la plataforma de streaming Fuchibol",
    version="1.0.0"
)

# Configurar CORS
fastapi_app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Incluir Routers
fastapi_app.include_router(auth.router, prefix="/api/v1")
fastapi_app.include_router(streams.router, prefix="/api/v1")
fastapi_app.include_router(channels.router, prefix="/api/v1")
fastapi_app.include_router(admin.router, prefix="/api/v1")
fastapi_app.include_router(restream.router, prefix="/api/v1")

@fastapi_app.on_event("startup")
async def startup_event():
    # Resume restreams that should be running
    asyncio.create_task(restream.resume_restreams())

@fastapi_app.get("/health")
def health_check():
    return {"status": "ok"}

# Inicializar Servidor Async de Socket.IO
sio = socketio.AsyncServer(async_mode='asgi', cors_allowed_origins='*')

# Registrar manejadores de eventos del chat
from app.api.chat import register_chat_handlers
register_chat_handlers(sio)

# Envolver la app de FastAPI en el servidor ASGI de Socket.IO
# para servir tanto HTTP como WebSockets bajo el puerto 8000
app = socketio.ASGIApp(sio, other_asgi_app=fastapi_app)
