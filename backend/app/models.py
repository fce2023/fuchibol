import datetime
from sqlalchemy import (
    Column, Integer, String, Boolean, DateTime, ForeignKey, 
    Index, Text, UniqueConstraint
)
from sqlalchemy.orm import relationship
from app.db import Base

class User(Base):
    __tablename__ = "users"
    
    id = Column(Integer, primary_key=True, index=True)
    email = Column(String, unique=True, index=True, nullable=False)
    username = Column(String, unique=True, index=True, nullable=False)
    hashed_password = Column(String, nullable=False)
    avatar_url = Column(String, nullable=True)
    role = Column(String, default="user", nullable=False) # admin, mod, streamer, user
    is_banned = Column(Boolean, default=False, nullable=False)
    created_at = Column(DateTime, default=datetime.datetime.utcnow, nullable=False)
    
    # Relationships
    channel = relationship("Channel", back_populates="user", uselist=False)
    chat_messages = relationship("ChatMessage", foreign_keys="ChatMessage.user_id", back_populates="user")
    deleted_chat_messages = relationship("ChatMessage", foreign_keys="ChatMessage.deleted_by", back_populates="moderator")
    following = relationship("Follower", foreign_keys="Follower.follower_id", back_populates="follower")
    blocked_terms = relationship("BlockedTerm", back_populates="creator")

class Channel(Base):
    __tablename__ = "channels"
    
    id = Column(Integer, primary_key=True, index=True)
    user_id = Column(Integer, ForeignKey("users.id", ondelete="CASCADE"), unique=True, nullable=False)
    name = Column(String, nullable=False)
    description = Column(Text, nullable=True)
    category = Column(String, nullable=True)
    stream_key_hash = Column(String, unique=True, nullable=False)
    key_expires_at = Column(DateTime, nullable=False)
    is_live = Column(Boolean, default=False, nullable=False)
    slowmode_seconds = Column(Integer, default=0, nullable=False)
    updated_at = Column(DateTime, default=datetime.datetime.utcnow, onupdate=datetime.datetime.utcnow, nullable=False)
    
    # IPTV fields
    iptv_urls = Column(String, default="[]", nullable=False) # JSON encoded list of URLs
    active_iptv_url = Column(String, nullable=True)
    iptv_enabled = Column(Boolean, default=False, nullable=False)
    
    # Agenda
    agenda_events = Column(String, default="[]", nullable=False) # JSON encoded list of events
    
    # WhatsApp
    whatsapp_link = Column(String, nullable=True)
    
    # TikTok
    tiktok_link = Column(String, nullable=True)
    
    # Relationships
    user = relationship("User", back_populates="channel")
    streams = relationship("Stream", back_populates="channel")
    chat_messages = relationship("ChatMessage", back_populates="channel")
    followers = relationship("Follower", foreign_keys="Follower.channel_id", back_populates="channel")
    blocked_terms = relationship("BlockedTerm", back_populates="channel")

    __table_args__ = (
        Index("idx_channels_is_live_category", "is_live", "category"),
        Index("idx_channels_user_id", "user_id"),
    )

class Stream(Base):
    __tablename__ = "streams"
    
    id = Column(Integer, primary_key=True, index=True)
    channel_id = Column(Integer, ForeignKey("channels.id", ondelete="CASCADE"), nullable=False)
    title = Column(String, nullable=False)
    start_time = Column(DateTime, default=datetime.datetime.utcnow, nullable=False)
    end_time = Column(DateTime, nullable=True)
    status = Column(String, default="live", nullable=False) # live, ended, interrupted
    peak_viewers = Column(Integer, default=0, nullable=False)
    created_at = Column(DateTime, default=datetime.datetime.utcnow, nullable=False)
    
    # Relationships
    channel = relationship("Channel", back_populates="streams")
    recording = relationship("Recording", back_populates="stream", uselist=False)

    __table_args__ = (
        Index("idx_streams_channel_id_status_start_time", "channel_id", "status", "start_time"),
    )

class Recording(Base):
    __tablename__ = "recordings"
    
    id = Column(Integer, primary_key=True, index=True)
    stream_id = Column(Integer, ForeignKey("streams.id", ondelete="CASCADE"), unique=True, nullable=False)
    s3_url = Column(String, nullable=True)
    duration_seconds = Column(Integer, nullable=True)
    status = Column(String, default="pending", nullable=False) # pending, processing, ready, failed
    is_public = Column(Boolean, default=True, nullable=False)
    created_at = Column(DateTime, default=datetime.datetime.utcnow, nullable=False)
    
    # Relationships
    stream = relationship("Stream", back_populates="recording")

class ChatMessage(Base):
    __tablename__ = "chat_messages"
    
    id = Column(Integer, primary_key=True, index=True)
    channel_id = Column(Integer, ForeignKey("channels.id", ondelete="CASCADE"), nullable=False)
    user_id = Column(Integer, ForeignKey("users.id", ondelete="CASCADE"), nullable=False)
    content = Column(Text, nullable=False)
    timestamp = Column(DateTime, default=datetime.datetime.utcnow, nullable=False)
    is_deleted = Column(Boolean, default=False, nullable=False)
    deleted_by = Column(Integer, ForeignKey("users.id", ondelete="SET NULL"), nullable=True)
    deletion_reason = Column(String, nullable=True)
    
    # Relationships
    channel = relationship("Channel", back_populates="chat_messages")
    user = relationship("User", foreign_keys=[user_id], back_populates="chat_messages")
    moderator = relationship("User", foreign_keys=[deleted_by], back_populates="deleted_chat_messages")

    __table_args__ = (
        Index("idx_chat_messages_channel_id_timestamp", "channel_id", "timestamp"),
    )

class Follower(Base):
    __tablename__ = "followers"
    
    follower_id = Column(Integer, ForeignKey("users.id", ondelete="CASCADE"), primary_key=True)
    channel_id = Column(Integer, ForeignKey("channels.id", ondelete="CASCADE"), primary_key=True)
    created_at = Column(DateTime, default=datetime.datetime.utcnow, nullable=False)
    
    # Relationships
    follower = relationship("User", foreign_keys=[follower_id], back_populates="following")
    channel = relationship("Channel", foreign_keys=[channel_id], back_populates="followers")

    __table_args__ = (
        Index("idx_followers_channel_id", "channel_id"),
        Index("idx_followers_follower_id", "follower_id"),
    )

class BlockedTerm(Base):
    __tablename__ = "blocked_terms"
    
    id = Column(Integer, primary_key=True, index=True)
    channel_id = Column(Integer, ForeignKey("channels.id", ondelete="CASCADE"), nullable=True) # null = global
    term = Column(String, nullable=False)
    created_by = Column(Integer, ForeignKey("users.id", ondelete="CASCADE"), nullable=False)
    created_at = Column(DateTime, default=datetime.datetime.utcnow, nullable=False)
    
    # Relationships
    channel = relationship("Channel", back_populates="blocked_terms")
    creator = relationship("User", back_populates="blocked_terms")
