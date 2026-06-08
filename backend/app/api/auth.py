from datetime import datetime, timedelta
from fastapi import APIRouter, Depends, HTTPException, status
from fastapi.security import OAuth2PasswordRequestForm
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.future import select
from app.db import get_db
from app.models import User, Channel
from app.schemas import UserResponse, UserCreate, Token
from app.utils.security import (
    get_password_hash, verify_password, create_access_token, 
    generate_stream_key, hash_stream_key
)
from app.api.deps import get_current_user
import secrets

router = APIRouter(prefix="/auth", tags=["auth"])

@router.post("/register", response_model=UserResponse, status_code=status.HTTP_201_CREATED)
async def register(user_in: UserCreate, db: AsyncSession = Depends(get_db)):
    # Check if email exists
    result = await db.execute(select(User).where(User.email == user_in.email))
    if result.scalars().first():
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="El correo electrónico ya está registrado."
        )
        
    # Check if username exists
    result = await db.execute(select(User).where(User.username == user_in.username))
    if result.scalars().first():
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="El nombre de usuario ya está en uso."
        )
        
    # Create user
    db_user = User(
        email=user_in.email,
        username=user_in.username,
        hashed_password=get_password_hash(user_in.password),
        role="streamer", # Default to streamer role so they can stream
    )
    db.add(db_user)
    await db.commit()
    await db.refresh(db_user)
    
    # Generate stream key
    raw_key = generate_stream_key()
    hashed_key = hash_stream_key(raw_key)
    
    # Create channel
    db_channel = Channel(
        user_id=db_user.id,
        name=f"Canal de {db_user.username}",
        description="¡Bienvenidos a mi canal!",
        stream_key_hash=hashed_key,
        key_expires_at=datetime.utcnow() + timedelta(days=90),
        slowmode_seconds=0
    )
    db.add(db_channel)
    await db.commit()
    
    return db_user

@router.post("/login", response_model=Token)
async def login(form_data: OAuth2PasswordRequestForm = Depends(), db: AsyncSession = Depends(get_db)):
    # Check by email or username
    result = await db.execute(
        select(User).where((User.email == form_data.username) | (User.username == form_data.username))
    )
    user = result.scalars().first()
    
    if not user or not verify_password(form_data.password, user.hashed_password):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Credenciales incorrectas.",
            headers={"WWW-Authenticate": "Bearer"},
        )
        
    if user.is_banned:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Tu cuenta ha sido suspendida."
        )
        
    # Create JWT
    access_token = create_access_token(
        data={"user_id": user.id, "email": user.email, "role": user.role, "username": user.username}
    )
    return {"access_token": access_token, "token_type": "bearer"}

@router.post("/rotate-stream-key", response_model=dict)
async def rotate_stream_key(
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    # Fetch user's channel
    result = await db.execute(select(Channel).where(Channel.user_id == current_user.id))
    channel = result.scalars().first()
    
    if not channel:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Canal no encontrado."
        )
        
    raw_key = generate_stream_key()
    channel.stream_key_hash = hash_stream_key(raw_key)
    channel.key_expires_at = datetime.utcnow() + timedelta(days=90)
    
    await db.commit()
    
    return {
        "stream_key": raw_key,
        "expires_at": channel.key_expires_at.isoformat()
    }

@router.get("/me", response_model=UserResponse)
async def get_me(current_user: User = Depends(get_current_user)):
    return current_user

@router.get("/channel/key", response_model=dict)
async def get_channel_key(
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    result = await db.execute(select(Channel).where(Channel.user_id == current_user.id))
    channel = result.scalars().first()
    if not channel:
        raise HTTPException(status_code=404, detail="Canal no encontrado")
    
    return {
        "key_expires_at": channel.key_expires_at.isoformat(),
        "info": "Las stream keys se guardan encriptadas. Rota tu clave si la has olvidado."
    }

@router.post("/oauth/google-stub", response_model=Token)
async def google_oauth_stub(email: str, username: str, db: AsyncSession = Depends(get_db)):
    """A mock Google OAuth register/login endpoint for testing."""
    result = await db.execute(select(User).where(User.email == email))
    user = result.scalars().first()
    
    if not user:
        # Create user
        user = User(
            email=email,
            username=username,
            hashed_password=get_password_hash(secrets.token_urlsafe(16)),
            role="streamer",
        )
        db.add(user)
        await db.commit()
        await db.refresh(user)
        
        # Create channel
        raw_key = generate_stream_key()
        channel = Channel(
            user_id=user.id,
            name=f"Canal de {user.username}",
            description="¡Bienvenidos a mi canal!",
            stream_key_hash=hash_stream_key(raw_key),
            key_expires_at=datetime.utcnow() + timedelta(days=90),
            slowmode_seconds=0
        )
        db.add(channel)
        await db.commit()
        
    access_token = create_access_token(
        data={"user_id": user.id, "email": user.email, "role": user.role, "username": user.username}
    )
    return {"access_token": access_token, "token_type": "bearer"}
