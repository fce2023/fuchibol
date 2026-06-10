import asyncio
from httpx import AsyncClient

async def run():
    async with AsyncClient() as client:
        res = await client.post(
            "http://localhost:8000/api/v1/streams/webhook/on_publish",
            json={
                "action": "on_publish",
                "client_id": 1,
                "ip": "172.22.0.6",
                "vhost": "__defaultVhost__",
                "app": "live",
                "tcUrl": "rtmp://srs:1935/live",
                "stream": "channel_1",
                "param": "?internal_secret=fuchibol_super_secret_key_change_me_in_prod"
            }
        )
        print(res.status_code)
        print(res.text)

asyncio.run(run())
