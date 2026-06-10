import httpx
from fastapi import APIRouter, Depends, HTTPException, status
from fastapi.responses import PlainTextResponse
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.future import select
from app.db import get_db
from app.models import User, Channel, Stream
from app.api.deps import get_current_user
from app.utils.redis_client import redis_client
from app.config import settings

router = APIRouter(prefix="/admin", tags=["admin"])

async def check_admin(user: User = Depends(get_current_user)):
    if user.role != "admin":
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Se requieren permisos de administrador."
        )
    return user

@router.get("/health")
async def admin_health(
    db: AsyncSession = Depends(get_db),
    admin_user: User = Depends(check_admin)
):
    # 1. DB Check
    db_status = "ok"
    try:
        await db.execute(select(1))
    except Exception as e:
        db_status = f"error: {str(e)}"
        
    # 2. Redis Check
    redis_status = "ok"
    try:
        await redis_client.ping()
    except Exception as e:
        redis_status = f"error: {str(e)}"
        
    # 3. SRS Check
    srs_status = "ok"
    try:
        async with httpx.AsyncClient() as client:
            res = await client.get("http://srs:1985/api/v1/versions", timeout=2.0)
            if res.status_code != 200:
                srs_status = f"status_code: {res.status_code}"
    except Exception as e:
        srs_status = f"error: {str(e)}"
        
    # 4. Celery Check
    celery_status = "ok"
    try:
        from app.worker import celery_app
        inspect = celery_app.control.inspect()
        active = inspect.active()
        if not active:
            celery_status = "no workers active"
    except Exception as e:
        celery_status = f"error: {str(e)}"
        
    # Get active live stream count and viewer count
    live_result = await db.execute(select(Channel).where(Channel.is_live == True))
    live_channels = live_result.scalars().all()
    
    total_viewers = 0
    for channel in live_channels:
        viewers_str = await redis_client.get(f"channel:{channel.id}:viewers")
        total_viewers += int(viewers_str) if viewers_str else 0
        
    return {
        "services": {
            "database": db_status,
            "redis": redis_status,
            "srs": srs_status,
            "celery": celery_status
        },
        "statistics": {
            "active_streams": len(live_channels),
            "total_viewers": total_viewers
        }
    }

@router.get("/metrics", response_class=PlainTextResponse)
async def metrics(db: AsyncSession = Depends(get_db)):
    # Retrieve active channels
    live_result = await db.execute(select(Channel).where(Channel.is_live == True))
    live_channels = live_result.scalars().all()
    
    total_viewers = 0
    for channel in live_channels:
        viewers_str = await redis_client.get(f"channel:{channel.id}:viewers")
        total_viewers += int(viewers_str) if viewers_str else 0
        
    # Expose user count
    user_count_result = await db.execute(select(User))
    user_count = len(user_count_result.scalars().all())
    
    metrics_str = (
        f"# HELP fuchibol_active_streams_count Number of active live streams\n"
        f"# TYPE fuchibol_active_streams_count gauge\n"
        f"fuchibol_active_streams_count {len(live_channels)}\n"
        f"# HELP fuchibol_total_viewers_count Total active viewers across all streams\n"
        f"# TYPE fuchibol_total_viewers_count gauge\n"
        f"fuchibol_total_viewers_count {total_viewers}\n"
        f"# HELP fuchibol_registered_users_count Total registered users on the platform\n"
        f"# TYPE fuchibol_registered_users_count gauge\n"
        f"fuchibol_registered_users_count {user_count}\n"
    )
    return metrics_str

from pydantic import BaseModel
class RoleUpdateRequest(BaseModel):
    role: str

@router.get("/users")
async def list_users(
    db: AsyncSession = Depends(get_db),
    admin_user: User = Depends(check_admin)
):
    result = await db.execute(select(User).order_by(User.id.desc()))
    users = result.scalars().all()
    return [{"id": u.id, "username": u.username, "email": u.email, "role": u.role} for u in users]

@router.post("/users/{target_user_id}/role")
async def update_user_role(
    target_user_id: int,
    role_req: RoleUpdateRequest,
    db: AsyncSession = Depends(get_db),
    admin_user: User = Depends(check_admin)
):
    if role_req.role not in ["admin", "mod", "user", "streamer"]:
        raise HTTPException(status_code=400, detail="Rol inválido")
        
    result = await db.execute(select(User).where(User.id == target_user_id))
    user = result.scalars().first()
    if not user:
        raise HTTPException(status_code=404, detail="Usuario no encontrado")
        
    user.role = role_req.role
    await db.commit()
    
    return {"message": f"Rol actualizado a {role_req.role} para {user.username}"}
