package leaderboards

import (
	"time"

	"github.com/google/uuid"
)

const (
	KindGlobal = "global"
	KindDaily  = "daily"
	KindMap    = "map"

	StatusActive   = "active"
	StatusArchived = "archived"

	RankingRuleBestScore = "best_score"
)

// Leaderboard is a durable read-model definition for a public competitive scope.
type Leaderboard struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Kind        string     `gorm:"type:text;not null"`
	ScopeKey    string     `gorm:"type:text;not null"`
	DisplayName string     `gorm:"type:text;not null"`
	Status      string     `gorm:"type:text;not null;default:'active'"`
	RankingRule string     `gorm:"type:text;not null;default:'best_score'"`
	MapID       *uuid.UUID `gorm:"type:uuid"`
	ChallengeID *uuid.UUID `gorm:"type:uuid"`
	CreatedAt   time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt   time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

func (Leaderboard) TableName() string { return "leaderboards" }

// Entry is a materialized registered-user leaderboard row.
type Entry struct {
	ID                   uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	LeaderboardID        uuid.UUID `gorm:"type:uuid;not null"`
	GameID               uuid.UUID `gorm:"type:uuid;not null"`
	UserID               uuid.UUID `gorm:"type:uuid;not null"`
	DisplayNameSnapshot  string    `gorm:"type:text;not null"`
	Score                int       `gorm:"type:int;not null"`
	GamesPlayed          int       `gorm:"type:int;not null;default:1"`
	CompletionDurationMS *int64    `gorm:"type:bigint"`
	CompletedAt          time.Time `gorm:"type:timestamptz;not null"`
	Rank                 int       `gorm:"type:int;not null"`
	CreatedAt            time.Time `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt            time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (Entry) TableName() string { return "leaderboard_entries" }

type completedGameCandidate struct {
	GameID               uuid.UUID `gorm:"column:game_id"`
	MapID                uuid.UUID `gorm:"column:map_id"`
	UserID               uuid.UUID `gorm:"column:user_id"`
	DisplayName          string    `gorm:"column:display_name"`
	Score                int       `gorm:"column:score"`
	CompletionDurationMS *int64    `gorm:"column:completion_duration_ms"`
	CompletedAt          time.Time `gorm:"column:completed_at"`
}
