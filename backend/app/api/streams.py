import urllib.parse
from datetime import datetime
import time
from fastapi import APIRouter, Depends, HTTPException, status, Request
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.future import select
from app.db import get_db
from app.models import Channel, User, Stream, Recording
from app.utils.security import hash_stream_key
from app.utils.redis_client import redis_client
from app.config import settings

router = APIRouter(prefix="/streams", tags=["streams"])

def extract_stream_keys(payload: dict):
    stream = payload.get("stream")
    param = payload.get("param", "")
    
    raw_key = None
    if param:
        parsed_params = urllib.parse.parse_qs(param.lstrip("?"))
        raw_key = parsed_params.get("key", [None])[0]
        
    if not raw_key:
        raw_key = stream
        
    return raw_key, stream

async def check_rate_limit(ip: str) -> bool:
    now = time.time()
    key = f"rate_limit:{ip}"
    async with redis_client.pipeline(transaction=True) as pipe:
        pipe.zadd(key, {str(now): now})
        pipe.zremrangebyscore(key, 0, now - 60)
        pipe.zcard(key)
        pipe.expire(key, 65)
        results = await pipe.execute()
    count = results[2]
    return count <= 10

@router.post("/webhook/on_publish")
async def on_publish(request: Request, db: AsyncSession = Depends(get_db)):
    payload = await request.json()
    ip = payload.get("ip", "unknown")
    
    # 1. Rate Limiting check
    if not await check_rate_limit(ip):
        raise HTTPException(status_code=429, detail="Too many connection attempts")
        
    raw_key, stream_name = extract_stream_keys(payload)
    if not raw_key:
        raise HTTPException(status_code=400, detail="Missing stream key")
        
    # 2. Hash stream key and lookup channel
    hashed_key = hash_stream_key(raw_key)
    result = await db.execute(
        select(Channel).where(Channel.stream_key_hash == hashed_key)
    )
    channel = result.scalars().first()
    
    if not channel:
        raise HTTPException(status_code=404, detail="Channel not found or invalid stream key")
        
    # 3. Check expiration
    if channel.key_expires_at < datetime.utcnow():
        raise HTTPException(status_code=403, detail="Stream key expired")
        
    # 4. Check user status
    user_result = await db.execute(select(User).where(User.id == channel.user_id))
    user = user_result.scalars().first()
    if not user or user.is_banned:
        raise HTTPException(status_code=403, detail="User is banned or not found")
        
    # 5. Start stream
    channel.is_live = True
    
    # Create stream record
    db_stream = Stream(
        channel_id=channel.id,
        title=channel.name,
        status="live",
        start_time=datetime.utcnow()
    )
    db.add(db_stream)
    await db.commit()
    await db.refresh(db_stream)
    
    # Cache stream info in Redis
    await redis_client.set(f"channel:{channel.id}:hls", stream_name)
    await redis_client.set(f"channel:{channel.id}:viewers", 0)
    
    # Create pending recording
    db_recording = Recording(
        stream_id=db_stream.id,
        status="pending",
        is_public=True
    )
    db.add(db_recording)
    await db.commit()
    
    # Notify followers (enqueue Celery task)
    try:
        from app.worker import notify_followers_task
        notify_followers_task.delay(channel.id, db_stream.id)
    except Exception as e:
        print(f"Warning: could not enqueue notify_followers_task: {e}")
        
    return 0  # 0 indicates success to SRS

@router.post("/webhook/on_unpublish")
async def on_unpublish(request: Request, db: AsyncSession = Depends(get_db)):
    payload = await request.json()
    raw_key, stream_name = extract_stream_keys(payload)
    
    if not raw_key:
        raise HTTPException(status_code=400, detail="Missing stream key")
        
    hashed_key = hash_stream_key(raw_key)
    result = await db.execute(
        select(Channel).where(Channel.stream_key_hash == hashed_key)
    )
    channel = result.scalars().first()
    
    if not channel:
        raise HTTPException(status_code=404, detail="Channel not found")
        
    # Stop stream
    channel.is_live = False
    
    # Find current active stream
    stream_result = await db.execute(
        select(Stream).where(
            (Stream.channel_id == channel.id) & 
            ((Stream.status == "live") | (Stream.status == "interrupted"))
        ).order_by(Stream.start_time.desc())
    )
    stream = stream_result.scalars().first()
    
    if stream:
        stream.status = "ended"
        stream.end_time = datetime.utcnow()
        await db.commit()
        
        # Enqueue VOD processing task in Celery
        try:
            from app.worker import process_vod_task
            process_vod_task.delay(stream.id)
        except Exception as e:
            print(f"Warning: could not enqueue process_vod_task: {e}")
            
    else:
        await db.commit()
        
    # Cleanup in Redis
    await redis_client.delete(f"channel:{channel.id}:hls")
    await redis_client.delete(f"channel:{channel.id}:viewers")
    
    return 0  # 0 indicates success to SRS

@router.get("/playback/{channel_id}")
async def get_playback_url(channel_id: int, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(Channel).where(Channel.id == channel_id))
    channel = result.scalars().first()
    if not channel:
        raise HTTPException(status_code=404, detail="Channel not found")
        
    # Retrieve actual stream playlist name from Redis
    stream_name = await redis_client.get(f"channel:{channel.id}:hls")
    if not stream_name:
        stream_name = f"channel_{channel.id}" # default fallback
        
    playback_url = f"{settings.HLS_BASE_URL}/live/{stream_name}.m3u8"
    
    return {
        "is_live": channel.is_live,
        "playback_url": playback_url
    }
