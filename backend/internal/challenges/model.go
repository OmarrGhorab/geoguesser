package challenges

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	TypeDaily  = "daily"
	TypeShared = "shared"

	StatusDraft       = "draft"
	StatusActive      = "active"
	StatusCompleted   = "completed"
	StatusArchived    = "archived"
	StatusUnavailable = "unavailable"

	AttemptStatusPending   = "pending"
	AttemptStatusActive    = "active"
	AttemptStatusCompleted = "completed"
	AttemptStatusAbandoned = "abandoned"
	AttemptStatusExpired   = "expired"

	DefaultRoundCount     = 5
	DefaultScoringVersion = 1

	StreakStatusInactive  = "inactive"
	StreakStatusActive    = "active"
	StreakStatusBroken    = "broken"
	StreakStatusProtected = "protected"

	ProtectionNone      = "none"
	ProtectionAvailable = "available"
	ProtectionConsumed  = "consumed"
	ProtectionExpired   = "expired"
)

// Challenge is a fixed-seed playable event.
type Challenge struct {
	ID               uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Type             string          `gorm:"type:text;not null"`
	SlugOrCode       *string         `gorm:"type:text"`
	Seed             string          `gorm:"type:text;not null"`
	ChallengeDate    *time.Time      `gorm:"type:date"`
	ResetStartsAt    *time.Time      `gorm:"type:timestamptz"`
	ResetEndsAt      *time.Time      `gorm:"type:timestamptz"`
	MapID            uuid.UUID       `gorm:"type:uuid;not null"`
	SettingsSnapshot json.RawMessage `gorm:"type:jsonb;not null"`
	Status           string          `gorm:"type:text;not null;default:'active'"`
	CreatedByUserID  *uuid.UUID      `gorm:"type:uuid"`
	CreatedAt        time.Time       `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt        time.Time       `gorm:"type:timestamptz;not null;default:now()"`
}

func (Challenge) TableName() string { return "challenges" }

// ChallengeLocation stores a selected ordered location snapshot.
type ChallengeLocation struct {
	ChallengeID      uuid.UUID `gorm:"type:uuid;primary_key"`
	RoundNumber      int       `gorm:"type:int;primary_key"`
	LocationID       uuid.UUID `gorm:"type:uuid;not null"`
	SelectionVersion int       `gorm:"type:int;not null;default:1"`
	CreatedAt        time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (ChallengeLocation) TableName() string { return "challenge_locations" }

// ChallengeAttempt connects an actor to a challenge playthrough.
type ChallengeAttempt struct {
	ID                   uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ChallengeID          uuid.UUID  `gorm:"type:uuid;not null"`
	GameID               *uuid.UUID `gorm:"type:uuid"`
	UserID               *uuid.UUID `gorm:"type:uuid"`
	GuestIdentityHash    *string    `gorm:"type:text"`
	Status               string     `gorm:"type:text;not null;default:'pending'"`
	LeaderboardEligible  bool       `gorm:"type:boolean;not null;default:false"`
	StartedAt            *time.Time `gorm:"type:timestamptz"`
	CompletedAt          *time.Time `gorm:"type:timestamptz"`
	TotalScore           int        `gorm:"type:int;not null;default:0"`
	TotalDistanceMeters  int        `gorm:"type:int;not null;default:0"`
	CompletionDurationMS *int64     `gorm:"type:bigint"`
	CreatedAt            time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt            time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

func (ChallengeAttempt) TableName() string { return "challenge_attempts" }

// ChallengeResult stores stable completed-result display data.
type ChallengeResult struct {
	AttemptID            uuid.UUID       `gorm:"type:uuid;primary_key"`
	ChallengeID          uuid.UUID       `gorm:"type:uuid;not null"`
	TotalScore           int             `gorm:"type:int;not null"`
	TotalDistanceMeters  int             `gorm:"type:int;not null"`
	RoundResultsSnapshot json.RawMessage `gorm:"type:jsonb;not null"`
	RankSnapshot         json.RawMessage `gorm:"type:jsonb"`
	CompletedAt          time.Time       `gorm:"type:timestamptz;not null"`
}

func (ChallengeResult) TableName() string { return "challenge_results" }

// LeaderboardEntry is a ranked account-backed result.
type LeaderboardEntry struct {
	ChallengeID          uuid.UUID `gorm:"type:uuid;primary_key"`
	AttemptID            uuid.UUID `gorm:"type:uuid;primary_key"`
	UserID               uuid.UUID `gorm:"type:uuid;not null"`
	DisplayNameSnapshot  string    `gorm:"type:text;not null"`
	Score                int       `gorm:"type:int;not null"`
	CompletionDurationMS *int64    `gorm:"type:bigint"`
	CompletedAt          time.Time `gorm:"type:timestamptz;not null"`
	Rank                 int       `gorm:"type:int;not null"`
	CreatedAt            time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (LeaderboardEntry) TableName() string { return "leaderboard_entries" }

type Streak struct {
	ID                         uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OwnerUserID                *uuid.UUID `gorm:"type:uuid"`
	GuestIdentityHash          *string    `gorm:"type:text"`
	CurrentCount               int        `gorm:"type:int;not null;default:0"`
	BestCount                  int        `gorm:"type:int;not null;default:0"`
	LastCompletedChallengeDate *time.Time `gorm:"type:date"`
	Status                     string     `gorm:"type:text;not null;default:'inactive'"`
	ProtectionState            string     `gorm:"type:text;not null;default:'none'"`
	UpdatedAt                  time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

func (Streak) TableName() string { return "streaks" }

type Mission struct {
	ID             uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Code           string          `gorm:"type:text;not null"`
	TitleKey       string          `gorm:"type:text;not null"`
	DescriptionKey string          `gorm:"type:text;not null"`
	MissionType    string          `gorm:"type:text;not null"`
	TargetValue    int             `gorm:"type:int;not null"`
	ActiveStartsAt time.Time       `gorm:"type:timestamptz;not null"`
	ActiveEndsAt   *time.Time      `gorm:"type:timestamptz"`
	RewardSnapshot json.RawMessage `gorm:"type:jsonb;not null"`
	Status         string          `gorm:"type:text;not null;default:'active'"`
	CreatedAt      time.Time       `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt      time.Time       `gorm:"type:timestamptz;not null;default:now()"`
}

func (Mission) TableName() string { return "missions" }

type MissionProgress struct {
	MissionID         uuid.UUID  `gorm:"type:uuid;primary_key"`
	OwnerUserID       *uuid.UUID `gorm:"type:uuid;primary_key"`
	GuestIdentityHash *string    `gorm:"type:text;primary_key"`
	CurrentValue      int        `gorm:"type:int;not null;default:0"`
	TargetValue       int        `gorm:"type:int;not null"`
	Status            string     `gorm:"type:text;not null;default:'not_started'"`
	CompletedAt       *time.Time `gorm:"type:timestamptz"`
	ClaimedAt         *time.Time `gorm:"type:timestamptz"`
	UpdatedAt         time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

func (MissionProgress) TableName() string { return "mission_progress" }

type MissionProgressEvent struct {
	ID                uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MissionID         uuid.UUID  `gorm:"type:uuid;not null"`
	OwnerUserID       *uuid.UUID `gorm:"type:uuid"`
	GuestIdentityHash *string    `gorm:"type:text"`
	SourceAttemptID   *uuid.UUID `gorm:"type:uuid"`
	SourceChallengeID *uuid.UUID `gorm:"type:uuid"`
	EventType         string     `gorm:"type:text;not null"`
	Delta             int        `gorm:"type:int;not null;default:1"`
	CreatedAt         time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

func (MissionProgressEvent) TableName() string { return "mission_progress_events" }
