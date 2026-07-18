package matchplay

import (
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/matchmaking"
)

// Re-export lifecycle constants for matchplay-local clarity.
const (
	MatchStatusMatched       = matchmaking.MatchStatusMatched
	MatchStatusActive        = matchmaking.MatchStatusActive
	MatchStatusCompleted     = matchmaking.MatchStatusCompleted
	MatchStatusCancelled     = matchmaking.MatchStatusCancelled
	MatchStatusFailedToStart = matchmaking.MatchStatusFailedToStart

	MatchResultTeamOneWin = matchmaking.MatchResultTeamOneWin
	MatchResultTeamTwoWin = matchmaking.MatchResultTeamTwoWin
	MatchResultDraw       = matchmaking.MatchResultDraw
	MatchResultForfeit    = matchmaking.MatchResultForfeit
	MatchResultAbandoned  = matchmaking.MatchResultAbandoned
	MatchResultCancelled  = matchmaking.MatchResultCancelled

	AbandonReasonExplicitLeave     = matchmaking.AbandonReasonExplicitLeave
	AbandonReasonDisconnectTimeout = matchmaking.AbandonReasonDisconnectTimeout
	AbandonReasonAccountIneligible = matchmaking.AbandonReasonAccountIneligible

	ParticipantStatusAssigned  = matchmaking.ParticipantStatusAssigned
	ParticipantStatusActive    = matchmaking.ParticipantStatusActive
	ParticipantStatusCompleted = matchmaking.ParticipantStatusCompleted
	ParticipantStatusCancelled = matchmaking.ParticipantStatusCancelled
	ParticipantStatusFailed    = matchmaking.ParticipantStatusFailed

	PlaylistCasual = matchmaking.PlaylistCasual
	PlaylistRanked = matchmaking.PlaylistRanked

	FormatSolo  = matchmaking.FormatSolo
	FormatDuo   = matchmaking.FormatDuo
	FormatSquad = matchmaking.FormatSquad

	// Player connection statuses projected into snapshots (game_players / presence).
	PlayerStatusActive       = "active"
	PlayerStatusDisconnected = "disconnected"
	PlayerStatusLeft         = "left"

	// Default grace and inactivity windows (overridable via service config).
	DefaultReconnectGrace   = 90 * time.Second
	DefaultCasualInactivity = 10 * time.Minute
	DefaultSweepBatchSize   = 100
	DefaultChatAccessWindow = 15 * time.Minute
)

// Match is the durable matchmade record with team-mode extensions.
// Backed by the matches table (migration 00019).
type Match struct {
	ID                     uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	FormationKey           string     `gorm:"type:text;not null"`
	GameID                 uuid.UUID  `gorm:"type:uuid;not null"`
	Mode                   string     `gorm:"type:text;not null"`
	Status                 string     `gorm:"type:text;not null"`
	Playlist               string     `gorm:"type:text;not null"`
	Format                 string     `gorm:"type:text;not null"`
	TeamSize               int        `gorm:"type:smallint;not null"`
	SeasonID               *uuid.UUID `gorm:"type:uuid"`
	TeamOneScore           int        `gorm:"type:int;not null;default:0"`
	TeamTwoScore           int        `gorm:"type:int;not null;default:0"`
	WinnerTeamSlot         *int       `gorm:"type:smallint"`
	Result                 *string    `gorm:"type:text"`
	LastActivityAt         time.Time  `gorm:"type:timestamptz;not null"`
	ChatAccessUntil        *time.Time `gorm:"type:timestamptz"`
	ProgressionFinalizedAt *time.Time `gorm:"type:timestamptz"`
	MatchedAt              time.Time  `gorm:"type:timestamptz;not null"`
	StartedAt              *time.Time `gorm:"type:timestamptz"`
	CompletedAt            *time.Time `gorm:"type:timestamptz"`
	ClosedAt               *time.Time `gorm:"type:timestamptz"`
	FailureCode            *string    `gorm:"type:text"`
	CreatedAt              time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt              time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the database table name.
func (Match) TableName() string { return "matches" }

// MatchParticipant is a durable match roster row with team/abandon extensions.
type MatchParticipant struct {
	MatchID       uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	GamePlayerID  uuid.UUID  `gorm:"type:uuid;not null"`
	Status        string     `gorm:"type:text;not null"`
	TeamSlot      int        `gorm:"type:smallint;not null"`
	PartyID       *uuid.UUID `gorm:"type:uuid"`
	AbandonedAt   *time.Time `gorm:"type:timestamptz"`
	AbandonReason *string    `gorm:"type:text"`
	AssignedAt    time.Time  `gorm:"type:timestamptz;not null"`
	CompletedAt   *time.Time `gorm:"type:timestamptz"`
	ClosedAt      *time.Time `gorm:"type:timestamptz"`
}

// TableName returns the database table name.
func (MatchParticipant) TableName() string { return "match_players" }

// GamePlayerRow is the game_players projection needed for snapshots.
type GamePlayerRow struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key"`
	GameID      uuid.UUID  `gorm:"type:uuid;not null"`
	UserID      *uuid.UUID `gorm:"type:uuid"`
	DisplayName string     `gorm:"type:text;not null"`
	Status      string     `gorm:"type:text;not null"`
	TotalScore  int        `gorm:"type:int;not null"`
	TeamSlot    *int       `gorm:"type:smallint"`
	LeftAt      *time.Time `gorm:"type:timestamptz"`
}

// TableName returns the database table name.
func (GamePlayerRow) TableName() string { return "game_players" }

// RoundRow is the rounds projection for current-round snapshot data.
type RoundRow struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key"`
	GameID      uuid.UUID  `gorm:"type:uuid;not null"`
	LocationID  uuid.UUID  `gorm:"type:uuid;not null"`
	RoundNumber int        `gorm:"type:int;not null"`
	Status      string     `gorm:"type:text;not null"`
	StartsAt    *time.Time `gorm:"type:timestamptz"`
	EndsAt      *time.Time `gorm:"type:timestamptz"`
	RevealedAt  *time.Time `gorm:"type:timestamptz"`
}

// TableName returns the database table name.
func (RoundRow) TableName() string { return "rounds" }

// GuessRow is a scored guess with accuracy/bonus components.
type GuessRow struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key"`
	RoundID        uuid.UUID `gorm:"type:uuid;not null"`
	GamePlayerID   uuid.UUID `gorm:"type:uuid;not null"`
	Latitude       float64   `gorm:"type:numeric(9,6);not null"`
	Longitude      float64   `gorm:"type:numeric(9,6);not null"`
	DistanceMeters int       `gorm:"type:int;not null"`
	AccuracyScore  int       `gorm:"type:int;not null"`
	SpeedBonus     int       `gorm:"type:int;not null;default:0"`
	Score          int       `gorm:"type:int;not null"`
	TimedOut       bool      `gorm:"type:boolean;not null;default:false"`
	SubmittedAt    time.Time `gorm:"type:timestamptz;not null"`
}

// TableName returns the database table name.
func (GuessRow) TableName() string { return "guesses" }

// AnswerLocation is the hidden round answer, exposed only after reveal.
type AnswerLocation struct {
	Latitude    float64
	Longitude   float64
	CountryCode string
	Region      *string
	Locality    *string
	Provider    string
	ProviderRef string
	Attribution *string
}

// SnapshotBundle is the authorized raw data needed to project a match snapshot.
type SnapshotBundle struct {
	Match         Match
	Participants  []MatchParticipant
	Players       []GamePlayerRow
	CurrentRound  *RoundRow
	SubmittedIDs  map[uuid.UUID]bool // game_player_id -> submitted this round
	EligibleCount int
	// LastCompletedRound is the most recently completed round (for last_round_result).
	LastCompletedRound *RoundRow
	LastRoundGuesses   []GuessRow
	LastRoundAnswer    *AnswerLocation
	// RealtimeVersion is optional; zero when the version store is unavailable.
	RealtimeVersion int64
}

// RoundResultBundle is the full revealed data for one completed round.
type RoundResultBundle struct {
	Match        Match
	Participants []MatchParticipant
	Players      []GamePlayerRow
	Round        RoundRow
	Guesses      []GuessRow
	Answer       AnswerLocation
	TeamOneScore int
	TeamTwoScore int
}

// TerminalResultBundle is the terminal match outcome payload.
type TerminalResultBundle struct {
	Match        Match
	Participants []MatchParticipant
	Players      []GamePlayerRow
	Rounds       []RoundResultBundle
	// ViewerProgression is the caller's competitive impact when loaded by the service.
	ViewerProgression *ProgressionDTO
}

// LeaveOutcome describes the durable result of an explicit leave or forfeit.
type LeaveOutcome struct {
	Match           Match
	AlreadyTerminal bool
	// AbandonedUserIDs are participants newly marked abandoned by this call.
	AbandonedUserIDs []uuid.UUID
	// RestoredPartyIDs are parties that should return to forming after terminal.
	RestoredPartyIDs []uuid.UUID
	// Version is the post-commit realtime version if published.
	Version int64
}

// DisconnectCandidate is a participant past reconnect grace.
type DisconnectCandidate struct {
	MatchID        uuid.UUID
	UserID         uuid.UUID
	GamePlayerID   uuid.UUID
	TeamSlot       int
	DisconnectedAt time.Time
	Playlist       string
	GameID         uuid.UUID
	PartyID        *uuid.UUID
}

// InactivityCandidate is a casual match with no activity past the inactivity window.
type InactivityCandidate struct {
	MatchID        uuid.UUID
	GameID         uuid.UUID
	LastActivityAt time.Time
	PartyIDs       []uuid.UUID
}

// IsTerminalMatch reports whether a match status is immutable.
func IsTerminalMatch(status string) bool {
	return matchmaking.IsTerminalMatch(status)
}

// IsActiveParticipant reports whether a participant still holds an active assignment.
func IsActiveParticipant(status string) bool {
	return matchmaking.IsActiveParticipant(status)
}

// IsCasualMatch reports whether the match is on the casual playlist.
func IsCasualMatch(m Match) bool {
	if m.Playlist == PlaylistCasual {
		return true
	}
	return matchmaking.IsCasual(m.Mode)
}

// IsRankedMatch reports whether the match is on the ranked playlist.
func IsRankedMatch(m Match) bool {
	if m.Playlist == PlaylistRanked {
		return true
	}
	return matchmaking.IsRanked(m.Mode)
}

// WinningTeamForForfeit returns the opposing team slot when teamSlot forfeits.
func WinningTeamForForfeit(teamSlot int) int {
	if teamSlot == 1 {
		return 2
	}
	return 1
}

// ResultForWinner maps a winning team slot to a terminal result code.
func ResultForWinner(winnerSlot int) string {
	if winnerSlot == 1 {
		return MatchResultTeamOneWin
	}
	return MatchResultTeamTwoWin
}
