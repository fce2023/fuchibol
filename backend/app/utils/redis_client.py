import redis
import redis.asyncio as aioredis
from app.config import settings

# Async client for FastAPI endpoints
redis_client = aioredis.from_url(settings.REDIS_URL, decode_responses=True)

# Sync client for Celery workers and migrations/sync tasks
sync_redis_client = redis.from_url(settings.REDIS_URL, decode_responses=True)
