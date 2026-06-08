"""Initial migration

Revision ID: 000000000001
Revises: 
Create Date: 2026-06-08 00:00:00.000000

"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa

# revision identifiers, used by Alembic.
revision: str = '000000000001'
down_revision: Union[str, None] = None
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    # Users table
    op.create_table(
        'users',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('email', sa.String(), nullable=False),
        sa.Column('username', sa.String(), nullable=False),
        sa.Column('hashed_password', sa.String(), nullable=False),
        sa.Column('avatar_url', sa.String(), nullable=True),
        sa.Column('role', sa.String(), nullable=False),
        sa.Column('is_banned', sa.Boolean(), nullable=False),
        sa.Column('created_at', sa.DateTime(), nullable=False),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_users_email'), 'users', ['email'], unique=True)
    op.create_index(op.f('ix_users_id'), 'users', ['id'], unique=False)
    op.create_index(op.f('ix_users_username'), 'users', ['username'], unique=True)

    # Channels table
    op.create_table(
        'channels',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('user_id', sa.Integer(), nullable=False),
        sa.Column('name', sa.String(), nullable=False),
        sa.Column('description', sa.Text(), nullable=True),
        sa.Column('category', sa.String(), nullable=True),
        sa.Column('stream_key_hash', sa.String(), nullable=False),
        sa.Column('key_expires_at', sa.DateTime(), nullable=False),
        sa.Column('is_live', sa.Boolean(), nullable=False),
        sa.Column('slowmode_seconds', sa.Integer(), nullable=False),
        sa.Column('updated_at', sa.DateTime(), nullable=False),
        sa.ForeignKeyConstraint(['user_id'], ['users.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('stream_key_hash'),
        sa.UniqueConstraint('user_id')
    )
    op.create_index(op.f('ix_channels_id'), 'channels', ['id'], unique=False)
    op.create_index('idx_channels_is_live_category', 'channels', ['is_live', 'category'], unique=False)
    op.create_index('idx_channels_user_id', 'channels', ['user_id'], unique=False)

    # Streams table
    op.create_table(
        'streams',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('channel_id', sa.Integer(), nullable=False),
        sa.Column('title', sa.String(), nullable=False),
        sa.Column('start_time', sa.DateTime(), nullable=False),
        sa.Column('end_time', sa.DateTime(), nullable=True),
        sa.Column('status', sa.String(), nullable=False),
        sa.Column('peak_viewers', sa.Integer(), nullable=False),
        sa.Column('created_at', sa.DateTime(), nullable=False),
        sa.ForeignKeyConstraint(['channel_id'], ['channels.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_streams_id'), 'streams', ['id'], unique=False)
    op.create_index('idx_streams_channel_id_status_start_time', 'streams', ['channel_id', 'status', 'start_time'], unique=False)

    # Recordings table
    op.create_table(
        'recordings',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('stream_id', sa.Integer(), nullable=False),
        sa.Column('s3_url', sa.String(), nullable=True),
        sa.Column('duration_seconds', sa.Integer(), nullable=True),
        sa.Column('status', sa.String(), nullable=False),
        sa.Column('is_public', sa.Boolean(), nullable=False),
        sa.Column('created_at', sa.DateTime(), nullable=False),
        sa.ForeignKeyConstraint(['stream_id'], ['streams.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('stream_id')
    )
    op.create_index(op.f('ix_recordings_id'), 'recordings', ['id'], unique=False)

    # ChatMessages table
    op.create_table(
        'chat_messages',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('channel_id', sa.Integer(), nullable=False),
        sa.Column('user_id', sa.Integer(), nullable=False),
        sa.Column('content', sa.Text(), nullable=False),
        sa.Column('timestamp', sa.DateTime(), nullable=False),
        sa.Column('is_deleted', sa.Boolean(), nullable=False),
        sa.Column('deleted_by', sa.Integer(), nullable=True),
        sa.Column('deletion_reason', sa.String(), nullable=True),
        sa.ForeignKeyConstraint(['channel_id'], ['channels.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['deleted_by'], ['users.id'], ondelete='SET NULL'),
        sa.ForeignKeyConstraint(['user_id'], ['users.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_chat_messages_id'), 'chat_messages', ['id'], unique=False)
    op.create_index('idx_chat_messages_channel_id_timestamp', 'chat_messages', ['channel_id', 'timestamp'], unique=False)

    # Followers table
    op.create_table(
        'followers',
        sa.Column('follower_id', sa.Integer(), nullable=False),
        sa.Column('channel_id', sa.Integer(), nullable=False),
        sa.Column('created_at', sa.DateTime(), nullable=False),
        sa.ForeignKeyConstraint(['channel_id'], ['channels.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['follower_id'], ['users.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('follower_id', 'channel_id')
    )
    op.create_index('idx_followers_channel_id', 'followers', ['channel_id'], unique=False)
    op.create_index('idx_followers_follower_id', 'followers', ['follower_id'], unique=False)

    # BlockedTerms table
    op.create_table(
        'blocked_terms',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('channel_id', sa.Integer(), nullable=True),
        sa.Column('term', sa.String(), nullable=False),
        sa.Column('created_by', sa.Integer(), nullable=False),
        sa.Column('created_at', sa.DateTime(), nullable=False),
        sa.ForeignKeyConstraint(['channel_id'], ['channels.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['created_by'], ['users.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_blocked_terms_id'), 'blocked_terms', ['id'], unique=False)


def downgrade() -> None:
    op.drop_index(op.f('ix_blocked_terms_id'), table_name='blocked_terms')
    op.drop_table('blocked_terms')
    op.drop_index('idx_followers_follower_id', table_name='followers')
    op.drop_index('idx_followers_channel_id', table_name='followers')
    op.drop_table('followers')
    op.drop_index('idx_chat_messages_channel_id_timestamp', table_name='chat_messages')
    op.drop_index(op.f('ix_chat_messages_id'), table_name='chat_messages')
    op.drop_table('chat_messages')
    op.drop_index(op.f('ix_recordings_id'), table_name='recordings')
    op.drop_table('recordings')
    op.drop_index('idx_streams_channel_id_status_start_time', table_name='streams')
    op.drop_index(op.f('ix_streams_id'), table_name='streams')
    op.drop_table('streams')
    op.drop_index('idx_channels_user_id', table_name='channels')
    op.drop_index('idx_channels_is_live_category', table_name='channels')
    op.drop_index(op.f('ix_channels_id'), table_name='channels')
    op.drop_table('channels')
    op.drop_index(op.f('ix_users_username'), table_name='users')
    op.drop_index(op.f('ix_users_id'), table_name='users')
    op.drop_index(op.f('ix_users_email'), table_name='users')
    op.drop_table('users')
