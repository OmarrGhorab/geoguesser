package rooms

import (
	"time"

	"github.com/google/uuid"
)

const (
	VisibilityPrivate = "private"
	VisibilityPublic  = "public"

	StatusLobby     = "lobby"
	StatusActive    = "active"
	StatusCompleted = "completed"
	StatusExpired   = "expired"
	StatusCancelled = "cancelled"

	ParticipantStatusJoined       = "joined"
	ParticipantStatusLeft         = "left"
	ParticipantStatusKicked       = "kicked"
	ParticipantStatusDisconnected = "disconnected"

	PresenceConnected    = "connected"
	PresenceDisconnected = "disconnected"

	PlayerRoleHost      = "host"
	PlayerRolePlayer    = "player"
	PlayerRoleSpectator = "spectator"
)

// Room is the durable private-room record backed by rooms.
type Room struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	GameID       *uuid.UUID `gorm:"type:uuid"`
	Code         string     `gorm:"type:text;not null"`
	Visibility   string     `gorm:"type:text;not null"`
	Status       string     `gorm:"type:text;not null"`
	HostUserID   *uuid.UUID `gorm:"type:uuid"`
	MaxPlayers   int        `gorm:"type:int;not null"`
	RoundCount   int        `gorm:"type:int;not null"`
	TimerSeconds *int       `gorm:"type:int"`
	ExpiresAt    time.Time  `gorm:"type:timestamptz;not null"`
	CreatedAt    time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt    time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

func (Room) TableName() string {
	return "rooms"
}

// RoomPlayer is the durable room membership record backed by room_players.
type RoomPlayer struct {
	RoomID       uuid.UUID  `gorm:"type:uuid;primaryKey"`
	GamePlayerID uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Status       string     `gorm:"type:text;not null"`
	JoinedAt     time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	LeftAt       *time.Time `gorm:"type:timestamptz"`
}

func (RoomPlayer) TableName() string {
	return "room_players"
}

// Participant combines room membership with the linked game player snapshot.
type Participant struct {
	RoomPlayer
	UserID            *uuid.UUID
	GuestIdentityHash *string
	DisplayName       string
	Role              string
	GameStatus        string
	TotalScore        int
}

func CanJoin(status string) bool {
	return status == StatusLobby
}

func CanRejoin(status string) bool {
	return status == StatusLobby || status == StatusActive
}

func CanUpdateSettings(status string) bool {
	return status == StatusLobby
}

func CanStart(status string) bool {
	return status == StatusLobby
}

func IsTerminal(status string) bool {
	switch status {
	case StatusCompleted, StatusExpired, StatusCancelled:
		return true
	default:
		return false
	}
}

func IsActiveParticipant(status string) bool {
	return status == ParticipantStatusJoined || status == ParticipantStatusDisconnected
}
