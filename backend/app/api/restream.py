import os
import subprocess
import asyncio
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.future import select
from app.db import get_db, AsyncSessionLocal
from app.models import Channel, User
from app.api.deps import get_current_user
from app.config import settings
from app.utils.redis_client import redis_client
from pydantic import BaseModel
import time

HLS_STORAGE_PATH = os.getenv("HLS_STORAGE_PATH", "/app/srs_hls")
HLS_PLAYLIST_MAX_AGE_SECONDS = int(os.getenv("HLS_PLAYLIST_MAX_AGE_SECONDS", "15"))

RESTREAM_STATE_KEY = "channel:{}:restream"
RESTREAM_PAUSED_KEY = "channel:{}:restream_paused"
OBS_LIVE_KEY = "channel:{}:obs_live"

def local_hls_playlist_recent(channel_id: int, max_age_seconds: int = HLS_PLAYLIST_MAX_AGE_SECONDS) -> bool:
    path = os.path.join(HLS_STORAGE_PATH, "live", f"channel_{channel_id}.m3u8")
    if not os.path.exists(path):
        return False
    age = time.time() - os.path.getmtime(path)
    return age <= max_age_seconds

async def set_restream_state(channel_id: int, active: bool):
    key = RESTREAM_STATE_KEY.format(channel_id)
    if active:
        await redis_client.set(key, "1")
    else:
        await redis_client.delete(key)

async def get_restream_state(channel_id: int) -> bool:
    key = RESTREAM_STATE_KEY.format(channel_id)
    value = await redis_client.get(key)
    return value == "1"

async def set_restream_paused(channel_id: int, paused: bool):
    key = RESTREAM_PAUSED_KEY.format(channel_id)
    if paused:
        await redis_client.set(key, "1")
    else:
        await redis_client.delete(key)

async def get_restream_paused(channel_id: int) -> bool:
    key = RESTREAM_PAUSED_KEY.format(channel_id)
    value = await redis_client.get(key)
    return value == "1"

async def set_obs_live(channel_id: int, active: bool):
    key = OBS_LIVE_KEY.format(channel_id)
    if active:
        await redis_client.set(key, "1")
    else:
        await redis_client.delete(key)

def cleanup_hls_files(channel_id: int):
    """Remove stale HLS files for a channel to prevent playback of old fragments."""
    try:
        # Clean both possible prefixes to be sure
        prefixes = [f"channel_{channel_id}", f"iptv_{channel_id}"]
        live_dir = os.path.join(HLS_STORAGE_PATH, "live")
        if not os.path.exists(live_dir):
            return
            
        for filename in os.listdir(live_dir):
            for prefix in prefixes:
                if filename.startswith(prefix):
                    file_path = os.path.join(live_dir, filename)
                    try:
                        os.remove(file_path)
                        print(f"[HLS] Deleted stale file: {filename}")
                    except Exception as e:
                        print(f"[HLS] Failed to delete {filename}: {e}")
    except Exception as e:
        print(f"[HLS] Error during cleanup: {e}")

async def get_obs_live(channel_id: int) -> bool:
    key = OBS_LIVE_KEY.format(channel_id)
    value = await redis_client.get(key)
    return value == "1"

async def stop_internal_restream(channel_id: int, pause: bool = False):
    # Set state BEFORE stopping process to avoid race conditions with on_unpublish webhook
    if pause:
        await set_restream_paused(channel_id, True)
    else:
        await set_restream_paused(channel_id, False)
        await set_restream_state(channel_id, False)

    if channel_id in active_restreams:
        active_restreams[channel_id]["should_run"] = False
        process = active_restreams[channel_id].get("process")
        if process and process.poll() is None:
            process.terminate()
            try:
                # Use wait_for to not block indefinitely if ffmpeg hangs
                await asyncio.wait_for(asyncio.to_thread(process.wait), timeout=2)
            except Exception:
                try:
                    process.kill()
                except:
                    pass
        del active_restreams[channel_id]
    
    # Clean up HLS files after stopping to ensure no leftover fragments
    cleanup_hls_files(channel_id)

async def resume_restreams():
    """Startup task to resume active restreams from Redis state."""
    print("[RESTREAM] Checking for active restreams to resume...")
    async with AsyncSessionLocal() as db:
        result = await db.execute(select(Channel).where(Channel.iptv_enabled == True))
        channels = result.scalars().all()
        
        for channel in channels:
            is_desired = await get_restream_state(channel.id)
            is_obs_live = await get_obs_live(channel.id)
            
            if is_desired:
                if is_obs_live:
                    print(f"[RESTREAM] Channel {channel.id} restream is desired but OBS is live, keeping paused.")
                    await set_restream_paused(channel.id, True)
                else:
                    if channel.active_iptv_url:
                        print(f"[RESTREAM] Resuming restream for channel {channel.id}")
                        try:
                            # Use start_restream_process which also clears the paused flag
                            await start_restream_process(channel, channel.active_iptv_url, db)
                        except Exception as e:
                            print(f"[RESTREAM] Failed to resume channel {channel.id}: {e}")
                    else:
                        # Desired but no URL? Reset state
                        await set_restream_state(channel.id, False)
                        await set_restream_paused(channel.id, False)



async def start_restream_process(channel: Channel, iptv_url: str, db: AsyncSession):
    if channel.id in active_restreams and active_restreams[channel.id].get("should_run", False):
        return

    # Clean up old fragments before starting new process
    cleanup_hls_files(channel.id)

    active_restreams[channel.id] = {"process": None, "should_run": True}
    # Change stream name to iptv_ to avoid collision with OBS channel_
    rtmp_url = f"rtmp://srs:1935/live/iptv_{channel.id}?vhost=__defaultVhost__&internal_secret={settings.SECRET_KEY}"
    task = asyncio.create_task(ffmpeg_worker(channel.id, iptv_url, rtmp_url))
    active_restreams[channel.id]["task"] = task

    channel.is_live = True
    await db.commit()

    await redis_client.set(f"channel:{channel.id}:hls", f"iptv_{channel.id}")
    await redis_client.set(f"channel:{channel.id}:viewers", 0)
    await set_restream_state(channel.id, True)
    await set_restream_paused(channel.id, False)

router = APIRouter(prefix="/restream", tags=["restream"])

# Global dictionary to keep track of active ffmpeg tasks
# Key: channel_id (int), Value: dict {"process": subprocess.Popen, "task": asyncio.Task, "should_run": bool}
active_restreams = {}

class RestreamStartRequest(BaseModel):
    iptv_url: str

async def ffmpeg_worker(channel_id: int, req_url: str, rtmp_url: str):
    """Background task that keeps FFmpeg running until explicitly stopped."""
    retries = 0
    max_retries = 100 # Allow many retries for long-running streams
    
    print(f"[RESTREAM] Starting worker for channel {channel_id}")
    
    while active_restreams.get(channel_id, {}).get("should_run", False):
        try:
            log_file = open(f"/tmp/ffmpeg_{channel_id}.log", "a")
            log_file.write(f"\n--- Starting FFmpeg restream for channel {channel_id} (Attempt {retries + 1}) ---\n")
            
            command = [
                "ffmpeg", "-y", "-re", "-i", req_url, 
                "-map", "0:v:0", "-map", "0:a:0",
                "-c:v", "libx264", "-preset", "veryfast", "-tune", "zerolatency", 
                "-maxrate", "1500k", "-bufsize", "3000k", "-pix_fmt", "yuv420p",
                "-g", "60", "-c:a", "aac", "-b:a", "128k", "-ar", "44100",
                "-vf", "scale=-2:720,fps=30",
                "-f", "flv", rtmp_url
            ]
            
            print(f"[RESTREAM] Executing FFmpeg for channel {channel_id}")
            process = subprocess.Popen(command, stdout=log_file, stderr=subprocess.STDOUT)
            active_restreams[channel_id]["process"] = process
            
            # Wait for process to exit
            while process.poll() is None:
                if not active_restreams.get(channel_id, {}).get("should_run", False):
                    print(f"[RESTREAM] Stopping FFmpeg for channel {channel_id}")
                    process.terminate()
                    try:
                        process.wait(timeout=3)
                    except:
                        process.kill()
                    break
                await asyncio.sleep(2)
                
            exit_code = process.poll()
            log_file.write(f"\n--- FFmpeg exited with code {exit_code} ---\n")
            log_file.close()
            
            if not active_restreams.get(channel_id, {}).get("should_run", False):
                break # User manually stopped it
                
            # Process died unexpectedly, retry
            retries += 1
            if retries >= max_retries:
                print(f"[RESTREAM] Max retries reached for channel {channel_id}")
                active_restreams[channel_id]["should_run"] = False
                break
                
            print(f"[RESTREAM] FFmpeg process died for channel {channel_id}, restarting in 5s...")
            await asyncio.sleep(5) # Wait before reconnecting
            
        except Exception as e:
            print(f"[RESTREAM] Error in ffmpeg_worker for channel {channel_id}: {e}")
            await asyncio.sleep(10)

    print(f"[RESTREAM] Worker loop finished for channel {channel_id}")
    paused = await get_restream_paused(channel_id)
    obs_live = await get_obs_live(channel_id)
    
    # When worker ends naturally, preserve desired restream state if it was paused while OBS is live.
    if not paused or not obs_live:
        async with AsyncSessionLocal() as session:
            res = await session.execute(select(Channel).where(Channel.id == channel_id))
            chan = res.scalars().first()
            if chan:
                chan.is_live = False
                await session.commit()
                print(f"[RESTREAM] Channel {channel_id} marked as offline in DB")

    await redis_client.delete(f"channel:{channel_id}:hls")
    await redis_client.delete(f"channel:{channel_id}:viewers")
    if not paused:
        await set_restream_state(channel_id, False)

    if channel_id in active_restreams:
        del active_restreams[channel_id]

@router.post("/start")
async def start_restream(
    req: RestreamStartRequest,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    if current_user.role not in ["admin", "streamer"]:
        raise HTTPException(status_code=403, detail="Not authorized")
        
    result = await db.execute(select(Channel).where(Channel.user_id == current_user.id))
    channel = result.scalars().first()
    if not channel:
        raise HTTPException(status_code=404, detail="Channel not found")
        
    channel.active_iptv_url = req.iptv_url
    channel.iptv_enabled = True
    await db.commit()

    await set_restream_state(channel.id, True)

    if await get_obs_live(channel.id):
        await set_restream_paused(channel.id, True)
        return {"status": "queued", "detail": "OBS is live. IPTV restream will start automatically when OBS disconnects."}

    # If already running, stop it first to allow restart with new URL (idempotent)
    if channel.id in active_restreams:
        await stop_internal_restream(channel.id, pause=False)
            
    try:
        await start_restream_process(channel, req.iptv_url, db)
    except Exception as e:
        if channel.id in active_restreams:
            del active_restreams[channel.id]
        raise HTTPException(status_code=500, detail=f"Failed to start worker: {str(e)}")
        
    return {"status": "started"}

@router.post("/stop")
async def stop_restream(
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    if current_user.role not in ["admin", "streamer"]:
        raise HTTPException(status_code=403, detail="Not authorized")
        
    result = await db.execute(select(Channel).where(Channel.user_id == current_user.id))
    channel = result.scalars().first()
    if not channel:
        raise HTTPException(status_code=404, detail="Channel not found")
        
    channel.is_live = False
    await db.commit()

    if channel.id in active_restreams:
        await stop_internal_restream(channel.id, pause=False)
        return {"status": "stopped"}

    await redis_client.delete(f"channel:{channel.id}:hls")
    await redis_client.delete(f"channel:{channel.id}:viewers")
    await set_restream_paused(channel.id, False)
    await set_restream_state(channel.id, False)
    
    return {"status": "not running"}

@router.get("/status")
async def get_restream_status(
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    if current_user.role not in ["admin", "streamer"]:
        raise HTTPException(status_code=403, detail="Not authorized")
        
    result = await db.execute(select(Channel).where(Channel.user_id == current_user.id))
    channel = result.scalars().first()
    if not channel:
        raise HTTPException(status_code=404, detail="Channel not found")
        
    is_active = channel.id in active_restreams and active_restreams[channel.id].get("should_run", False)
    is_desired = await get_restream_state(channel.id)
    is_paused_by_obs = await get_restream_paused(channel.id) and await get_obs_live(channel.id)
    is_playlist_recent = local_hls_playlist_recent(channel.id)

    return {
        "is_live": channel.is_live,
        "is_active_restream": is_active,
        "is_restream_desired": is_desired,
        "is_paused_by_obs": is_paused_by_obs,
        "is_playlist_recent": is_playlist_recent
    }
