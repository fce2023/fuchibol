from pydantic import BaseModel, EmailStr, Field
from typing import Optional, List
from datetime import datetime

# Token Schemas
class Token(BaseModel):
    access_token: str
    token_type: str

class TokenData(BaseModel):
    user_id: Optional[int] = None
    email: Optional[str] = None
    role: Optional[str] = None

# User Schemas
class UserBase(BaseModel):
    email: EmailStr
    username: str

class UserCreate(UserBase):
    password: str

class UserUpdate(BaseModel):
    username: Optional[str] = None
    email: Optional[EmailStr] = None
    password: Optional[str] = None

class UserResponse(UserBase):
    id: int
    avatar_url: Optional[str] = None
    role: str
    is_banned: bool
    created_at: datetime

    class Config:
        from_attributes = True

# Channel Schemas
class ChannelBase(BaseModel):
    name: str
    description: Optional[str] = None
    category: Optional[str] = None
    slowmode_seconds: int = Field(default=0, ge=0)
    iptv_urls: str = "[]"
    active_iptv_url: Optional[str] = None
    iptv_enabled: bool = False
    agenda_events: str = "[]"
    whatsapp_link: Optional[str] = None
    tiktok_link: Optional[str] = None

class ChannelCreate(ChannelBase):
    pass

class ChannelUpdate(BaseModel):
    name: Optional[str] = None
    description: Optional[str] = None
    category: Optional[str] = None
    slowmode_seconds: Optional[int] = Field(default=None, ge=0)
    iptv_urls: Optional[str] = None
    active_iptv_url: Optional[str] = None
    iptv_enabled: Optional[bool] = None
    agenda_events: Optional[str] = None
    whatsapp_link: Optional[str] = None
    tiktok_link: Optional[str] = None

class ChannelResponse(ChannelBase):
    id: int
    user_id: int
    is_live: bool
    updated_at: datetime

    class Config:
        from_attributes = True

class ChannelWithStreamKeyResponse(ChannelResponse):
    stream_key: str

    class Config:
        from_attributes = True

# Stream Schemas
class StreamBase(BaseModel):
    title: str

class StreamCreate(StreamBase):
    channel_id: int

class StreamResponse(StreamBase):
    id: int
    channel_id: int
    start_time: datetime
    end_time: Optional[datetime] = None
    status: str
    peak_viewers: int
    created_at: datetime

    class Config:
        from_attributes = True

# Recording Schemas
class RecordingResponse(BaseModel):
    id: int
    stream_id: int
    s3_url: Optional[str] = None
    duration_seconds: Optional[int] = None
    status: str
    is_public: bool
    created_at: datetime

    class Config:
        from_attributes = True

# Chat Message Schemas
class ChatMessageCreate(BaseModel):
    content: str

class ChatMessageResponse(BaseModel):
    id: int
    channel_id: int
    user_id: int
    username: str
    content: str
    timestamp: datetime
    is_deleted: bool
    deleted_by: Optional[int] = None
    deletion_reason: Optional[str] = None

    class Config:
        from_attributes = True

# Follower Schemas
class FollowerResponse(BaseModel):
    follower_id: int
    channel_id: int
    created_at: datetime

    class Config:
        from_attributes = True

# BlockedTerm Schemas
class BlockedTermCreate(BaseModel):
    term: str
    channel_id: Optional[int] = None # Null means global (admin only)

class BlockedTermResponse(BaseModel):
    id: int
    channel_id: Optional[int] = None
    term: str
    created_by: int
    created_at: datetime

    class Config:
        from_attributes = True
