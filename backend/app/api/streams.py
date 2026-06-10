import os
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

HLS_STORAGE_PATH = os.getenv("HLS_STORAGE_PATH", "/app/srs_hls")

HLS_PLAYLIST_MAX_AGE_SECONDS = int(os.getenv("HLS_PLAYLIST_MAX_AGE_SECONDS", "15"))

def local_hls_playlist_recent(stream_name: str, max_age_seconds: int = HLS_PLAYLIST_MAX_AGE_SECONDS) -> bool:
    playlist_path = os.path.join(HLS_STORAGE_PATH, "live", f"{stream_name}.m3u8")
    if not os.path.exists(playlist_path):
        return False
    age = time.time() - os.path.getmtime(playlist_path)
    return age <= max_age_seconds


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
    content_type = request.headers.get("content-type", "")
    print(f"[WEBHOOK] on_publish called. Content-Type: {content_type}")
    
    try:
        if "application/json" in content_type:
            payload = await request.json()
        else:
            form_data = await request.form()
            payload = dict(form_data)
    except Exception as e:
        print(f"[WEBHOOK] Error parsing payload: {e}")
        return 0

    print(f"[WEBHOOK] on_publish payload: {payload}")
    
    if not payload:
        print("[WEBHOOK] Empty payload, skipping")
        return 0

    ip = payload.get("ip", "unknown")
    
    # 1. Rate Limiting check
    if not await check_rate_limit(ip):
        print(f"[WEBHOOK] Rate limit exceeded for IP: {ip}")
        raise HTTPException(status_code=429, detail="Too many connection attempts")
        
    raw_key, stream_name = extract_stream_keys(payload)
    if not stream_name:
        print("[WEBHOOK] No stream name in payload, ignoring")
        return 0
    
    # Check if it's an internal FFmpeg restream
    param = payload.get("param", "")
    parsed_params = urllib.parse.parse_qs(param.lstrip("?"))
    internal_secret = parsed_params.get("internal_secret", [None])[0]
    
    if not internal_secret and "?" in stream_name:
        stream_part, query_part = stream_name.split("?", 1)
        parsed_params = urllib.parse.parse_qs(query_part)
        internal_secret = parsed_params.get("internal_secret", [None])[0]
        stream_name = stream_part

    is_internal = (internal_secret == settings.SECRET_KEY)
    print(f"[WEBHOOK] Stream: {stream_name}, Internal: {is_internal}")
    
    channel = None
    if is_internal:
        try:
            # stream_name can be "channel_1" or "iptv_1"
            parts = stream_name.split("_")
            channel_id = int(parts[-1])
            result = await db.execute(select(Channel).where(Channel.id == channel_id))
            channel = result.scalars().first()
        except Exception as e:
            print(f"[WEBHOOK] Error identifying internal channel {stream_name}: {e}")
            pass
    else:
        if not raw_key:
            if "?" in stream_name:
                _, query_part = stream_name.split("?", 1)
                parsed_params = urllib.parse.parse_qs(query_part)
                raw_key = parsed_params.get("key", [None])[0]

        if not raw_key:
            print(f"[WEBHOOK] Missing stream key for stream: {stream_name}")
            raise HTTPException(status_code=400, detail="Missing stream key")
            
        hashed_key = hash_stream_key(raw_key)
        result = await db.execute(
            select(Channel).where(Channel.stream_key_hash == hashed_key)
        )
        channel = result.scalars().first()
    
    if not channel:
        print(f"[WEBHOOK] Channel not found or invalid key for stream: {stream_name}")
        raise HTTPException(status_code=404, detail="Channel not found or invalid stream key")
        
    if channel.key_expires_at < datetime.utcnow():
        raise HTTPException(status_code=403, detail="Stream key expired")
        
    user_result = await db.execute(select(User).where(User.id == channel.user_id))
    user = user_result.scalars().first()
    if not user or user.is_banned:
        raise HTTPException(status_code=403, detail="User is banned or not found")
        
    if not is_internal:
        from app.api.restream import get_restream_state, stop_internal_restream, set_obs_live, cleanup_hls_files
        await set_obs_live(channel.id, True)
        if await get_restream_state(channel.id):
            await stop_internal_restream(channel.id, pause=True)
        cleanup_hls_files(channel.id)

    # Start stream
    channel.is_live = True
    
    db_stream = Stream(
        channel_id=channel.id,
        title=channel.name,
        status="live",
        start_time=datetime.utcnow()
    )
    db.add(db_stream)
    await db.commit()
    await db.refresh(db_stream)
    
    await redis_client.set(f"channel:{channel.id}:hls", stream_name)
    await redis_client.set(f"channel:{channel.id}:viewers", 0)
    
    db_recording = Recording(
        stream_id=db_stream.id,
        status="pending",
        is_public=True
    )
    db.add(db_recording)
    await db.commit()
    
    try:
        from app.worker import notify_followers_task
        notify_followers_task.delay(channel.id, db_stream.id)
    except Exception as e:
        print(f"Warning: could not enqueue notify_followers_task: {e}")
        
    return 0

async def is_stream_active_in_srs(stream_name: str) -> bool:
    """Query SRS API to check if a stream is actually being published."""
    try:
        import httpx
        async with httpx.AsyncClient() as client:
            # Internal Docker network URL
            response = await client.get("http://srs:1985/api/v1/streams/", timeout=1.0)
            if response.status_code == 200:
                data = response.json()
                streams = data.get("streams", [])
                for s in streams:
                    if s.get("name") == stream_name:
                        publish = s.get("publish", {})
                        if publish.get("active"):
                            return True
    except Exception as e:
        print(f"[SRS API] Error checking stream status: {e}")
    return False

@router.post("/webhook/on_unpublish")
async def on_unpublish(request: Request, db: AsyncSession = Depends(get_db)):
    content_type = request.headers.get("content-type", "")
    print(f"[WEBHOOK] on_unpublish called. Content-Type: {content_type}")
    
    try:
        if "application/json" in content_type:
            payload = await request.json()
        else:
            form_data = await request.form()
            payload = dict(form_data)
    except Exception as e:
        print(f"[WEBHOOK] Error parsing unpublish payload: {e}")
        return 0

    print(f"[WEBHOOK] on_unpublish payload: {payload}")
    if not payload:
        return 0

    _, stream_name = extract_stream_keys(payload)
    if not stream_name:
        return 0
        
    param = payload.get("param", "")
    parsed_params = urllib.parse.parse_qs(param.lstrip("?"))
    internal_secret = parsed_params.get("internal_secret", [None])[0]
    
    if not internal_secret and "?" in stream_name:
        stream_part, query_part = stream_name.split("?", 1)
        parsed_params = urllib.parse.parse_qs(query_part)
        internal_secret = parsed_params.get("internal_secret", [None])[0]
        stream_name = stream_part

    is_internal = (internal_secret == settings.SECRET_KEY)
    
    channel = None
    try:
        if "_" in stream_name:
            parts = stream_name.split("_")
            channel_id = int(parts[-1])
            result = await db.execute(select(Channel).where(Channel.id == channel_id))
            channel = result.scalars().first()
    except:
        pass
        
    if not channel:
        return 0

    print(f"[WEBHOOK] Unpublish for channel {channel.id}, internal: {is_internal}")

    from app.api.restream import active_restreams, get_restream_state, get_restream_paused, get_obs_live, set_obs_live, set_restream_paused, stop_internal_restream, start_restream_process, cleanup_hls_files

    if is_internal:
        if await get_restream_paused(channel.id) and await get_obs_live(channel.id):
            print(f"[WEBHOOK] Ignoring internal unpublish for channel {channel.id} because OBS is taking over")
            return 0
    else:
        # Before assuming OBS is gone, check SRS API
        if await is_stream_active_in_srs(f"channel_{channel.id}"):
            print(f"[WEBHOOK] OBS unpublish received for {channel.id} but SRS says stream is still active (reconnect?)")
            return 0

        await set_obs_live(channel.id, False)
        cleanup_hls_files(channel.id)
        if await get_restream_state(channel.id) and await get_restream_paused(channel.id):
            if channel.active_iptv_url:
                try:
                    print(f"[WEBHOOK] OBS ended for channel {channel.id}, resuming IPTV")
                    await start_restream_process(channel, channel.active_iptv_url, db)
                    return 0
                except Exception as e:
                    print(f"[RESTREAM] Failed to resume IPTV restream for channel {channel.id}: {e}")
            await set_restream_paused(channel.id, False)

    # Mark the current session as ended
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
        try:
            from app.worker import process_vod_task
            process_vod_task.delay(stream.id)
        except Exception as e:
            print(f"Warning: could not enqueue process_vod_task: {e}")
    else:
        await db.commit()

    # Final check: is anyone publishing anything?
    if not await is_stream_active_in_srs(f"channel_{channel.id}"):
        # Wait a bit before marking offline to allow for quick reconnections (buffer)
        print(f"[WEBHOOK] Channel {channel.id} seems empty, starting 15s grace period...")
        asyncio.create_task(delayed_offline_cleanup(channel.id))
    else:
        print(f"[WEBHOOK] Channel {channel.id} still active in SRS, keeping live status")
    
    return 0

async def delayed_offline_cleanup(channel_id: int):
    """Wait and verify if the channel is still offline before marking it so."""
    await asyncio.sleep(15)
    
    # Check both possible streams
    is_obs = await is_stream_active_in_srs(f"channel_{channel_id}")
    is_iptv = await is_stream_active_in_srs(f"iptv_{channel_id}")
    
    if not is_obs and not is_iptv:
        async with AsyncSessionLocal() as db:
            result = await db.execute(select(Channel).where(Channel.id == channel_id))
            channel = result.scalars().first()
            if channel and channel.is_live:
                from app.api.restream import set_obs_live, cleanup_hls_files
                
                print(f"[CLEANUP] Channel {channel_id} is still offline after grace period, cleaning up.")
                channel.is_live = False
                await db.commit()
                await set_obs_live(channel.id, False)
                cleanup_hls_files(channel_id)
                await redis_client.delete(f"channel:{channel.id}:hls")
                await redis_client.delete(f"channel:{channel.id}:viewers")
    else:
        print(f"[CLEANUP] Channel {channel_id} reconnected during grace period (OBS:{is_obs}, IPTV:{is_iptv}), cleanup aborted.")

@router.get("/playback/{channel_id}")
async def get_playback_url(channel_id: int, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(Channel).where(Channel.id == channel_id))
    channel = result.scalars().first()
    if not channel:
        raise HTTPException(status_code=404, detail="Channel not found")
        
    stream_name = await redis_client.get(f"channel:{channel.id}:hls")
    if not stream_name:
        stream_name = f"channel_{channel.id}"
        
    playback_url = f"{settings.HLS_BASE_URL}/live/{stream_name}.m3u8"
    
    is_live = channel.is_live
    if not is_live:
        restream_state = await redis_client.get(f"channel:{channel.id}:restream")
        if restream_state == "1":
            is_live = True

    return {
        "is_live": is_live,
        "playback_url": playback_url
    }
