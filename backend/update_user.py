import asyncio
from app.db import AsyncSessionLocal
from app.models import User
from sqlalchemy.future import select

async def update():
    async with AsyncSessionLocal() as db:
        result = await db.execute(select(User).where(User.username == "roy"))
        user = result.scalars().first()
        if user:
            user.role = "admin"
            await db.commit()
            print("User roy updated to admin")
        else:
            print("User roy not found")

asyncio.run(update())
