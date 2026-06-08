import os
import subprocess
import time
import json
from datetime import datetime
from contextlib import contextmanager
from celery import Celery
from app.config import settings
from app.db import SessionLocal

celery_app = Celery(
    "tasks",
    broker=settings.REDIS_URL,
    backend=settings.REDIS_URL
)

celery_app.conf.update(
    task_serializer="json",
    accept_content=["json"],
    result_serializer="json",
    timezone="UTC",
    enable_utc=True,
)

# Celery Beat Schedule definition
celery_app.conf.beat_schedule = {
    "bulk-insert-chat-every-5-seconds": {
        "task": "app.worker.bulk_insert_chat_task",
        "schedule": 5.0,
    },
    "watchdog-streams-every-15-seconds": {
        "task": "app.worker.watchdog_streams_task",
        "schedule": 15.0,
    },
}

@contextmanager
def get_sync_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()

@celery_app.task(name="app.worker.notify_followers_task")
def notify_followers_task(channel_id: int, stream_id: int):
    print(f"Notifying followers of channel {channel_id} about stream {stream_id}")
    return True

@celery_app.task(name="app.worker.process_vod_task", max_retries=3, default_retry_delay=60)
def process_vod_task(stream_id: int):
    print(f"Starting VOD processing for stream {stream_id}")
    with get_sync_db() as db:
        from app.models import Stream, Recording, Channel
        
        stream = db.query(Stream).filter(Stream.id == stream_id).first()
        if not stream:
            print(f"Stream {stream_id} not found.")
            return False
            
        recording = db.query(Recording).filter(Recording.stream_id == stream_id).first()
        if not recording:
            recording = Recording(stream_id=stream_id, status="pending")
            db.add(recording)
            db.commit()
            
        recording.status = "processing"
        db.commit()
        
        channel = db.query(Channel).filter(Channel.id == stream.channel_id).first()
        channel_id = channel.id if channel else 0
        
        hls_dir = "/app/srs_hls/live"
        vod_dir = settings.VOD_OUTPUT_DIR
        os.makedirs(vod_dir, exist_ok=True)
        
        from app.utils.redis_client import sync_redis_client
        stream_name = sync_redis_client.get(f"channel:{channel_id}:hls") or f"channel_{channel_id}"
        
        playlist_path = os.path.join(hls_dir, f"{stream_name}.m3u8")
        if not os.path.exists(playlist_path):
            playlist_path = os.path.join(hls_dir, f"channel_{channel_id}.m3u8")
            
        if not os.path.exists(playlist_path):
            print(f"Playlist not found: {playlist_path}. Failing recording.")
            recording.status = "failed"
            db.commit()
            return False
            
        ts_files = []
        with open(playlist_path, "r") as f:
            for line in f:
                line = line.strip()
                if line and not line.startswith("#"):
                    ts_path = os.path.join(hls_dir, line)
                    if os.path.exists(ts_path):
                        ts_files.append(ts_path)
                        
        if not ts_files:
            print(f"No TS segments found in playlist. Failing recording.")
            recording.status = "failed"
            db.commit()
            return False
            
        concat_list_path = os.path.join(vod_dir, f"concat_{stream_id}.txt")
        with open(concat_list_path, "w") as f:
            for ts in ts_files:
                f.write(f"file '{ts}'\n")
                
        output_mp4_path = os.path.join(vod_dir, f"channel_{channel_id}_stream_{stream_id}.mp4")
        
        ffmpeg_cmd = [
            "ffmpeg", "-y",
            "-f", "concat",
            "-safe", "0",
            "-i", concat_list_path,
            "-c", "copy",
            "-bsf:a", "aac_adtstoasc",
            output_mp4_path
        ]
        
        try:
            print(f"Running ffmpeg command: {' '.join(ffmpeg_cmd)}")
            result = subprocess.run(ffmpeg_cmd, capture_output=True, text=True, check=True)
            print("FFmpeg output:", result.stdout)
            
            recording.status = "ready"
            recording.s3_url = f"{settings.HLS_BASE_URL}/vod/channel_{channel_id}_stream_{stream_id}.mp4"
            recording.duration_seconds = len(ts_files) * 2
            
            db.commit()
            print(f"VOD generated successfully: {output_mp4_path}")
            
            try:
                import socketio
                mgr = socketio.RedisManager(settings.REDIS_URL)
                mgr.emit("vod_ready", {
                    "stream_id": stream_id,
                    "vod_url": recording.s3_url,
                    "duration": recording.duration_seconds
                }, room=f"channel_{channel_id}")
            except Exception as e:
                print(f"Failed to send Socket.IO notification: {e}")
                
            return True
            
        except subprocess.CalledProcessError as e:
            print(f"FFmpeg failed: {e.stderr}")
            recording.status = "failed"
            db.commit()
            return False
        finally:
            if os.path.exists(concat_list_path):
                os.remove(concat_list_path)

@celery_app.task(name="app.worker.bulk_insert_chat_task")
def bulk_insert_chat_task():
    from app.utils.redis_client import sync_redis_client
    
    messages = []
    for _ in range(100):
        data = sync_redis_client.lpop("chat_queue")
        if not data:
            break
        messages.append(json.loads(data))
        
    if not messages:
        return 0
        
    print(f"Inserting {len(messages)} chat messages into DB...")
    
    with get_sync_db() as db:
        from app.models import ChatMessage
        
        db_messages = []
        for msg in messages:
            db_msg = ChatMessage(
                channel_id=msg["channel_id"],
                user_id=msg["user_id"],
                content=msg["content"],
                timestamp=datetime.fromisoformat(msg["timestamp"]),
                is_deleted=False
            )
            db_messages.append(db_msg)
            
        db.bulk_save_objects(db_messages)
        db.commit()
        
    return len(messages)

@celery_app.task(name="app.worker.watchdog_streams_task")
def watchdog_streams_task():
    with get_sync_db() as db:
        from app.models import Channel, Stream
        import socketio
        
        channels = db.query(Channel).filter(Channel.is_live == True).all()
        if not channels:
            return "No active channels."
            
        hls_dir = "/app/srs_hls/live"
        mgr = socketio.RedisManager(settings.REDIS_URL)
        
        for channel in channels:
            from app.utils.redis_client import sync_redis_client
            stream_name = sync_redis_client.get(f"channel:{channel.id}:hls") or f"channel_{channel.id}"
            playlist_path = os.path.join(hls_dir, f"{stream_name}.m3u8")
            
            is_dead = True
            if os.path.exists(playlist_path):
                mtime = os.path.getmtime(playlist_path)
                elapsed = time.time() - mtime
                if elapsed < 45:
                    is_dead = False
                    
            if is_dead:
                print(f"Watchdog: Stream for channel {channel.id} is dead. Cleaning up.")
                channel.is_live = False
                
                stream = db.query(Stream).filter(
                    (Stream.channel_id == channel.id) & 
                    (Stream.status == "live")
                ).order_by(Stream.start_time.desc()).first()
                
                if stream:
                    stream.status = "interrupted"
                    stream.end_time = datetime.utcnow()
                    
                db.commit()
                
                try:
                    mgr.emit("stream_interrupted", {
                        "channel_id": channel.id,
                        "message": "La transmisión se ha interrumpido."
                    }, room=f"channel_{channel.id}")
                except Exception as e:
                    print(f"Failed to emit Socket.IO stream_interrupted event: {e}")
        
        return "Watchdog verification finished."
