import socketio
import json
import time
import re
from datetime import datetime
from app.utils.security import decode_access_token
from app.utils.redis_client import redis_client
from app.db import AsyncSessionLocal
from app.models import Channel, BlockedTerm, User
from sqlalchemy.future import select

def register_chat_handlers(sio: socketio.AsyncServer):
    
    @sio.event
    async def connect(sid, environ, auth=None):
        token = None
        if auth and isinstance(auth, dict):
            token = auth.get("token")
        
        if not token:
            # Check query string
            from urllib.parse import parse_qs
            query_string = environ.get("QUERY_STRING", "")
            params = parse_qs(query_string)
            token = params.get("token", [None])[0]
            
        if token:
            payload = decode_access_token(token)
            if payload:
                # Store user details in session context
                await sio.save_session(sid, {
                    "user_id": payload.get("user_id"),
                    "email": payload.get("email"),
                    "role": payload.get("role"),
                    "username": payload.get("username", "Anon")
                })
                print(f"Client {sid} (User) connected successfully. User ID: {payload.get('user_id')}")
                return True
            else:
                print(f"Client {sid} provided invalid token, connecting as Guest.")

        # If no token or invalid token, connect as Guest
        await sio.save_session(sid, {
            "user_id": None,
            "email": None,
            "role": "guest",
            "username": f"Guest_{sid[:4]}"
        })
        print(f"Client {sid} (Guest) connected successfully.")
        return True

    @sio.event
    async def disconnect(sid):
        # Trigger leave room cleanup
        session = await sio.get_session(sid)
        if session:
            channel_id = session.get("channel_id")
            if channel_id:
                viewers = await redis_client.decr(f"channel:{channel_id}:viewers")
                if viewers < 0:
                    await redis_client.set(f"channel:{channel_id}:viewers", 0)
                    viewers = 0
                await sio.emit("viewer_update", {"viewers": viewers}, room=f"channel_{channel_id}")
        print(f"Client {sid} disconnected")

    @sio.event
    async def join_channel(sid, data):
        if not isinstance(data, dict):
            return
        channel_id = data.get("channel_id")
        if not channel_id:
            await sio.emit("error", {"message": "Canal no especificado"}, to=sid)
            return
            
        session = await sio.get_session(sid)
        if not session:
            # Ensure session exists even if connect failed somehow
            session = {"user_id": None, "role": "guest", "username": f"Guest_{sid[:4]}"}
            
        # Add client to channel room
        await sio.enter_room(sid, f"channel_{channel_id}")
        session["channel_id"] = channel_id
        await sio.save_session(sid, session)
        
        # Increment active viewer count in Redis and broadcast
        viewers = await redis_client.incr(f"channel:{channel_id}:viewers")
        await sio.emit("viewer_update", {"viewers": viewers}, room=f"channel_{channel_id}")
        print(f"Client {sid} joined channel room channel_{channel_id}. Viewers: {viewers}")

    @sio.event
    async def leave_channel(sid):
        session = await sio.get_session(sid)
        if not session:
            return
        channel_id = session.get("channel_id")
        if channel_id:
            await sio.leave_room(sid, f"channel_{channel_id}")
            viewers = await redis_client.decr(f"channel:{channel_id}:viewers")
            if viewers < 0:
                await redis_client.set(f"channel:{channel_id}:viewers", 0)
                viewers = 0
            await sio.emit("viewer_update", {"viewers": viewers}, room=f"channel_{channel_id}")
            session["channel_id"] = None
            await sio.save_session(sid, session)
            print(f"Client {sid} left channel room channel_{channel_id}")

    @sio.event
    async def send_message(sid, data):
        if not isinstance(data, dict):
            return
        session = await sio.get_session(sid)
        if not session:
            return
            
        user_id = session.get("user_id")
        if not user_id:
            await sio.emit("error", {"message": "Debes iniciar sesión para chatear"}, to=sid)
            return

        username = session.get("username", "Anon")
        channel_id = session.get("channel_id")
        
        if not channel_id:
            await sio.emit("error", {"message": "No estás en ningún canal"}, to=sid)
            return
            
        content = data.get("content", "").strip()
        if not content:
            return
            
        # 1. Slowmode Validation
        slowmode_key = f"channel:{channel_id}:slowmode"
        slowmode_seconds_str = await redis_client.get(slowmode_key)
        if slowmode_seconds_str is None:
            async with AsyncSessionLocal() as db:
                res = await db.execute(select(Channel).where(Channel.id == channel_id))
                chan = res.scalars().first()
                slowmode_seconds = chan.slowmode_seconds if chan else 0
                await redis_client.set(slowmode_key, slowmode_seconds, ex=300)
        else:
            slowmode_seconds = int(slowmode_seconds_str)
            
        if slowmode_seconds > 0:
            last_msg_key = f"user:{user_id}:channel:{channel_id}:last_message"
            last_msg_time_str = await redis_client.get(last_msg_key)
            if last_msg_time_str:
                elapsed = time.time() - float(last_msg_time_str)
                if elapsed < slowmode_seconds:
                    remaining = int(slowmode_seconds - elapsed)
                    await sio.emit("error", {
                        "message": f"Modo lento activo. Espera {remaining} segundos.",
                        "remaining": remaining
                    }, to=sid)
                    return
            await redis_client.set(last_msg_key, time.time(), ex=slowmode_seconds)

        # 2. Blocked Terms Filtering
        blocked_key = f"channel:{channel_id}:blocked_terms"
        blocked_terms_json = await redis_client.get(blocked_key)
        if blocked_terms_json is None:
            async with AsyncSessionLocal() as db:
                res = await db.execute(
                    select(BlockedTerm.term).where(
                        (BlockedTerm.channel_id == channel_id) | (BlockedTerm.channel_id == None)
                    )
                )
                terms = [row[0] for row in res.all()]
                await redis_client.set(blocked_key, json.dumps(terms), ex=300)
        else:
            terms = json.loads(blocked_terms_json)
            
        filtered_content = content
        for term in terms:
            if term.lower() in filtered_content.lower():
                insensitive_term = re.compile(re.escape(term), re.IGNORECASE)
                filtered_content = insensitive_term.sub("*" * len(term), filtered_content)

        import uuid
        msg_id = str(uuid.uuid4())
        
        # 3. Broadcast message
        msg_payload = {
            "id": msg_id,
            "channel_id": channel_id,
            "user_id": user_id,
            "username": username,
            "content": filtered_content,
            "timestamp": datetime.utcnow().isoformat()
        }
        await sio.emit("message", msg_payload, room=f"channel_{channel_id}")
        
        # 4. Push message to Redis list queue for Celery bulk insert
        chat_msg_db = {
            "id": msg_id,
            "channel_id": channel_id,
            "user_id": user_id,
            "content": filtered_content,
            "timestamp": msg_payload["timestamp"]
        }
        await redis_client.rpush("chat_queue", json.dumps(chat_msg_db))

    @sio.event
    async def delete_message(sid, data):
        if not isinstance(data, dict):
            return
        session = await sio.get_session(sid)
        if not session:
            return
        
        role = session.get("role")
        if role not in ["admin", "mod", "streamer"]:
            await sio.emit("error", {"message": "No tienes permisos para borrar mensajes"}, to=sid)
            return
            
        channel_id = session.get("channel_id")
        msg_id = data.get("msg_id")
        
        if not channel_id or not msg_id:
            return
            
        # Emit delete event to the room
        await sio.emit("message_deleted", {"msg_id": msg_id}, room=f"channel_{channel_id}")
        
        # Ideally, we would also update the database here via Celery or directly
        # but for real-time visual moderation this works.
