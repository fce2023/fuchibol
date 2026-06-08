from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.future import select
from app.db import get_db
from app.models import Channel, User
from app.schemas import ChannelResponse, ChannelUpdate
from app.api.deps import get_current_user

router = APIRouter(prefix="/channels", tags=["channels"])

@router.get("/by-username/{username}", response_model=ChannelResponse)
async def get_channel_by_username(username: str, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(User).where(User.username == username))
    user = result.scalars().first()
    if not user:
        raise HTTPException(status_code=404, detail="Usuario no encontrado")
        
    chan_result = await db.execute(select(Channel).where(Channel.user_id == user.id))
    channel = chan_result.scalars().first()
    if not channel:
        raise HTTPException(status_code=404, detail="Canal no encontrado")
        
    return channel

@router.patch("/me", response_model=ChannelResponse)
async def update_my_channel(
    channel_update: ChannelUpdate,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    result = await db.execute(select(Channel).where(Channel.user_id == current_user.id))
    channel = result.scalars().first()
    if not channel:
        raise HTTPException(status_code=404, detail="Canal no encontrado")
        
    update_data = channel_update.model_dump(exclude_unset=True)
    for key, value in update_data.items():
        setattr(channel, key, value)
        
    await db.commit()
    await db.refresh(channel)
    
    if "slowmode_seconds" in update_data:
        from app.utils.redis_client import redis_client
        await redis_client.set(f"channel:{channel.id}:slowmode", channel.slowmode_seconds, ex=300)
        
    return channel
