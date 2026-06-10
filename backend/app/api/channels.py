from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.future import select
from sqlalchemy import delete
from app.db import get_db
from app.models import Channel, User, Follower
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

@router.get("/primary")
async def get_primary_channel(db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(Channel).order_by(Channel.id.asc()))
    channel = result.scalars().first()
    if not channel:
        raise HTTPException(status_code=404, detail="No hay canales disponibles")
        
    user_res = await db.execute(select(User).where(User.id == channel.user_id))
    user = user_res.scalars().first()
    
    return {"username": user.username}

@router.get("/me", response_model=ChannelResponse)
async def get_my_channel(
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    result = await db.execute(select(Channel).where(Channel.user_id == current_user.id))
    channel = result.scalars().first()
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

@router.post("/{channel_id}/follow")
async def follow_channel(
    channel_id: int,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    # Check if channel exists
    chan_result = await db.execute(select(Channel).where(Channel.id == channel_id))
    channel = chan_result.scalars().first()
    if not channel:
        raise HTTPException(status_code=404, detail="Canal no encontrado")
    
    # Can't follow yourself
    if channel.user_id == current_user.id:
        raise HTTPException(status_code=400, detail="No puedes seguir tu propio canal")
    
    # Check if already following
    follow_result = await db.execute(
        select(Follower).where(
            (Follower.follower_id == current_user.id) & 
            (Follower.channel_id == channel_id)
        )
    )
    if follow_result.scalars().first():
        return {"status": "already_following"}
    
    new_follow = Follower(follower_id=current_user.id, channel_id=channel_id)
    db.add(new_follow)
    await db.commit()
    return {"status": "followed"}

@router.post("/{channel_id}/unfollow")
async def unfollow_channel(
    channel_id: int,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    await db.execute(
        delete(Follower).where(
            (Follower.follower_id == current_user.id) & 
            (Follower.channel_id == channel_id)
        )
    )
    await db.commit()
    return {"status": "unfollowed"}

@router.get("/{channel_id}/following")
async def is_following(
    channel_id: int,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    result = await db.execute(
        select(Follower).where(
            (Follower.follower_id == current_user.id) & 
            (Follower.channel_id == channel_id)
        )
    )
    is_following = result.scalars().first() is not None
    
    # Count followers
    count_result = await db.execute(
        select(User).join(Follower, Follower.follower_id == User.id).where(Follower.channel_id == channel_id)
    )
    count = len(count_result.scalars().all())
    
    return {"is_following": is_following, "follower_count": count}

@router.get("/{channel_id}/followers_list")
async def get_followers_list(
    channel_id: int,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    # Verify ownership
    result = await db.execute(select(Channel).where(Channel.id == channel_id))
    channel = result.scalars().first()
    if not channel:
        raise HTTPException(status_code=404, detail="Canal no encontrado")
    
    if channel.user_id != current_user.id and current_user.role != "admin":
        raise HTTPException(status_code=403, detail="No autorizado")
    
    # Get followers with user info
    followers_result = await db.execute(
        select(User).join(Follower, Follower.follower_id == User.id).where(Follower.channel_id == channel_id)
    )
    followers = followers_result.scalars().all()
    
    return [
        {
            "id": f.id,
            "username": f.username,
            "avatar_url": f.avatar_url,
            "email": f.email if current_user.role == "admin" else None
        } for f in followers
    ]
