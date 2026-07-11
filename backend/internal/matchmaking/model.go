package matchmaking

import (
	"time"

	"github.com/google/uuid"
)

// Supported competitive modes for this phase.
const (
	ModeRankedStandard = "ranked_standard"
)

// Match lifecycle statuses (matches.status).
const (
	MatchStatusMatched       = "matched"
	MatchStatusActive        = "active"
	MatchStatusCompleted     = "completed"
	MatchStatusCancelled     = "cancelled"
	MatchStatusFailedToStart = "failed_to_start"
)

// Match participant statuses (match_players.status).
const (
	ParticipantStatusAssigned  = "assigned"
	ParticipantStatusActive    = "active"
	ParticipantStatusCompleted = "completed"
	ParticipantStatusCancelled = "cancelled"
	ParticipantStatusFailed    = "failed"
)

// Ephemeral queue entry states stored in Redis.
const (
	QueueStateSearching = "searching"
	QueueStateClaimed   = "claimed"
)

// Public status values returned to clients.
const (
	PublicStatusNotQueued              = "not_queued"
	PublicStatusSearching              = "searching"
	PublicStatusMatched                = "matched"
	PublicStatusTemporarilyUnavailable = "temporarily_unavailable"
)

// Match is the durable ranked match record backed by matches.
type Match struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	FormationKey string     `gorm:"type:text;not null;uniqueIndex"`
	GameID       uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
	Mode         string     `gorm:"type:text;not null"`
	Status       string     `gorm:"type:text;not null"`
	MatchedAt    time.Time  `gorm:"type:timestamptz;not null"`
	StartedAt    *time.Time `gorm:"type:timestamptz"`
	CompletedAt  *time.Time `gorm:"type:timestamptz"`
	ClosedAt     *time.Time `gorm:"type:timestamptz"`
	FailureCode  *string    `gorm:"type:text"`
	CreatedAt    time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt    time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the database table name.
func (Match) TableName() string {
	return "matches"
}

// MatchPlayer is a durable match participant backed by match_players.
type MatchPlayer struct {
	MatchID      uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID       uuid.UUID  `gorm:"type:uuid;primaryKey"`
	GamePlayerID uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
	Status       string     `gorm:"type:text;not null"`
	AssignedAt   time.Time  `gorm:"type:timestamptz;not null"`
	CompletedAt  *time.Time `gorm:"type:timestamptz"`
	ClosedAt     *time.Time `gorm:"type:timestamptz"`
}

// TableName returns the database table name.
func (MatchPlayer) TableName() string {
	return "match_players"
}

// ActiveAssignment is a privacy-safe durable assignment used for status recovery.
type ActiveAssignment struct {
	MatchID   uuid.UUID
	GameID    uuid.UUID
	Mode      string
	Status    string
	MatchedAt time.Time
	UserID    uuid.UUID
}

// IsTerminalMatch reports whether a match status is immutable.
func IsTerminalMatch(status string) bool {
	switch status {
	case MatchStatusCompleted, MatchStatusCancelled, MatchStatusFailedToStart:
		return true
	default:
		return false
	}
}

// IsActiveParticipant reports whether a participant still holds an active assignment.
func IsActiveParticipant(status string) bool {
	return status == ParticipantStatusAssigned || status == ParticipantStatusActive
}

// SupportedMode reports whether mode is approved for this phase.
func SupportedMode(mode string) bool {
	return mode == ModeRankedStandard
}
