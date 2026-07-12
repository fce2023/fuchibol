package models

import (
	"time"
)

// User model
type User struct {
	ID                  uint           `gorm:"primaryKey;index" json:"id"`
	Email               string         `gorm:"uniqueIndex;not null" json:"email"`
	Username            string         `gorm:"uniqueIndex;not null" json:"username"`
	HashedPassword      string         `gorm:"not null" json:"-"`
	AvatarURL           *string        `json:"avatar_url"`
	Role                string         `gorm:"default:'user';not null" json:"role"`
	IsBanned            bool           `gorm:"default:false;not null" json:"is_banned"`
	CreatedAt           time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	
	// Relationships
	Channel             *Channel       `json:"channel,omitempty"`
	ChatMessages        []ChatMessage  `gorm:"foreignKey:UserID" json:"-"`
	Following           []Follower     `gorm:"foreignKey:FollowerID" json:"-"`
}

// Channel model
type Channel struct {
	ID               uint           `gorm:"primaryKey;index" json:"id"`
	UserID           uint           `gorm:"unique;not null;index" json:"user_id"`
	Name             string         `gorm:"not null" json:"name"`
	Description      *string        `json:"description"`
	Category         *string        `json:"category"`
	StreamKeyHash    string         `gorm:"uniqueIndex;not null" json:"-"`
	KeyExpiresAt     time.Time      `gorm:"not null" json:"-"`
	IsLive           bool           `gorm:"default:false;not null;index:idx_channels_is_live_category" json:"is_live"`
	SlowmodeSeconds  int            `gorm:"default:0;not null" json:"slowmode_seconds"`
	ActiveStreamName string         `gorm:"size:255" json:"active_stream_name"`
	UpdatedAt        time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	
	// IPTV fields
	IptvUrls         string         `gorm:"default:'[]';not null" json:"iptv_urls"`
	ActiveIptvUrl    *string        `json:"active_iptv_url"`
	IptvEnabled      bool           `gorm:"default:false;not null" json:"iptv_enabled"`
	
	// Agenda
	AgendaEvents     string         `gorm:"default:'[]';not null" json:"agenda_events"`
	
	// Social
	WhatsappLink     *string        `json:"whatsapp_link"`
	TiktokLink       *string        `json:"tiktok_link"`
	LogoUrl          *string        `json:"logo_url"`
	
	// Donations
	YapeNumber       *string        `gorm:"size:50" json:"yape_number"`
	PaypalLink       *string        `gorm:"type:text" json:"paypal_link"`
	DonationMessage  *string        `gorm:"type:text" json:"donation_message"`
	DonationLongMessage *string     `gorm:"type:text" json:"donation_long_message"`
	
	// Relationships
	User             *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Streams          []Stream       `json:"-"`
	ChatMessages     []ChatMessage  `json:"-"`
	Followers        []Follower     `json:"-"`
}

// Stream model
type Stream struct {
	ID               uint           `gorm:"primaryKey;index" json:"id"`
	ChannelID        uint           `gorm:"not null;index:idx_streams_channel_id_status_start_time" json:"channel_id"`
	Title            string         `gorm:"not null" json:"title"`
	StartTime        time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_streams_channel_id_status_start_time" json:"start_time"`
	EndTime          *time.Time     `json:"end_time"`
	Status           string         `gorm:"default:'live';not null;index:idx_streams_channel_id_status_start_time" json:"status"`
	PeakViewers      int            `gorm:"default:0;not null" json:"peak_viewers"`
	CreatedAt        time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	
	// Relationships
	Channel          *Channel       `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
	Recording        *Recording     `json:"recording,omitempty"`
}

// Recording model
type Recording struct {
	ID               uint           `gorm:"primaryKey;index" json:"id"`
	StreamID         uint           `gorm:"unique;not null" json:"stream_id"`
	S3Url            *string        `json:"s3_url"`
	DurationSeconds  *int           `json:"duration_seconds"`
	Status           string         `gorm:"default:'pending';not null" json:"status"`
	IsPublic         bool           `gorm:"default:true;not null" json:"is_public"`
	CreatedAt        time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	
	// Relationships
	Stream           *Stream        `gorm:"foreignKey:StreamID" json:"stream,omitempty"`
}

// ChatMessage model
type ChatMessage struct {
	ID               uint           `gorm:"primaryKey;index" json:"id"`
	ChannelID        uint           `gorm:"not null;index:idx_chat_messages_channel_id_timestamp" json:"channel_id"`
	UserID           uint           `gorm:"not null" json:"user_id"`
	Content          string         `gorm:"type:text;not null" json:"content"`
	Timestamp        time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_chat_messages_channel_id_timestamp" json:"timestamp"`
	IsDeleted        bool           `gorm:"default:false;not null" json:"is_deleted"`
	DeletedBy        *uint          `json:"deleted_by"`
	DeletionReason   *string        `json:"deletion_reason"`
	
	// Relationships
	User             *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Channel          *Channel       `gorm:"foreignKey:ChannelID" json:"-"`
}

// Follower model
type Follower struct {
	FollowerID       uint           `gorm:"primaryKey;autoIncrement:false;index:idx_followers_follower_id" json:"follower_id"`
	ChannelID        uint           `gorm:"primaryKey;autoIncrement:false;index:idx_followers_channel_id" json:"channel_id"`
	CreatedAt        time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	
	// Relationships
	Follower         *User          `gorm:"foreignKey:FollowerID" json:"follower,omitempty"`
	Channel          *Channel       `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
}

// BlockedTerm model
type BlockedTerm struct {
	ID               uint           `gorm:"primaryKey;index" json:"id"`
	ChannelID        *uint          `json:"channel_id"`
	Term             string         `gorm:"not null" json:"term"`
	CreatedBy        uint           `gorm:"not null" json:"created_by"`
	CreatedAt        time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	
	// Relationships
	Creator          *User          `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

// AnalyticsLog model to track views and traffic
type AnalyticsLog struct {
	ID        uint      `gorm:"primaryKey;index" json:"id"`
	ChannelID uint      `gorm:"not null;index" json:"channel_id"`
	IPAddress string    `gorm:"size:45" json:"ip_address"`
	Country   string    `gorm:"size:100" json:"country"`
	City      string    `gorm:"size:100" json:"city"`
	UserAgent string    `gorm:"type:text" json:"user_agent"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;index" json:"created_at"`
}

// Donation model to track manual user donations
type Donation struct {
	ID            uint      `gorm:"primaryKey;index" json:"id"`
	ChannelID     uint      `gorm:"not null;index" json:"channel_id"`
	DonorName     string    `gorm:"size:100;not null" json:"donor_name"`
	Amount        float64   `gorm:"type:numeric(10,2);not null" json:"amount"`
	Method        string    `gorm:"size:50;not null" json:"method"` // yape, paypal
	ReferenceCode string    `gorm:"size:100" json:"reference_code"`
	ReceiptUrl    *string   `gorm:"type:text" json:"receipt_url"`
	Status        string    `gorm:"size:50;not null;default:'pending';index" json:"status"` // pending, approved, rejected
	CreatedAt     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}
