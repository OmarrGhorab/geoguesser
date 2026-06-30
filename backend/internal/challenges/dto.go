package challenges

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/games"
)

type SettingsSnapshot struct {
	RoundCount     int    `json:"round_count"`
	TimerSeconds   *int   `json:"timer_seconds"`
	MovementRules  string `json:"movement_rules"`
	ScoringVersion int    `json:"scoring_version"`
}

type MapSummary struct {
	ID uuid.UUID `json:"id"`
}

type ChallengeSummary struct {
	ID            uuid.UUID        `json:"id"`
	Type          string           `json:"type"`
	Seed          string           `json:"seed"`
	ChallengeDate *string          `json:"challenge_date,omitempty"`
	ResetStartsAt *time.Time       `json:"reset_starts_at,omitempty"`
	ResetEndsAt   *time.Time       `json:"reset_ends_at,omitempty"`
	Map           MapSummary       `json:"map"`
	Settings      SettingsSnapshot `json:"settings"`
	Status        string           `json:"status"`
	ShareCode     *string          `json:"share_code,omitempty"`
	ShareURL      *string          `json:"share_url,omitempty"`
}

type AttemptSummary struct {
	ID                  uuid.UUID  `json:"id"`
	ChallengeID         uuid.UUID  `json:"challenge_id"`
	Status              string     `json:"status"`
	LeaderboardEligible bool       `json:"leaderboard_eligible"`
	StartedAt           *time.Time `json:"started_at,omitempty"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	TotalScore          int        `json:"total_score"`
	CurrentRoundNumber  *int       `json:"current_round_number,omitempty"`
	GameID              *uuid.UUID `json:"game_id,omitempty"`
}

type StreakSummary struct {
	CurrentCount               int     `json:"current_count"`
	BestCount                  int     `json:"best_count"`
	LastCompletedChallengeDate *string `json:"last_completed_challenge_date,omitempty"`
	Status                     string  `json:"status"`
	ProtectionState            string  `json:"protection_state"`
	GuestLimited               bool    `json:"guest_limited"`
}

type MissionSummary struct {
	ID             uuid.UUID  `json:"id,omitempty"`
	Code           string     `json:"code"`
	TitleKey       string     `json:"title_key"`
	DescriptionKey string     `json:"description_key"`
	MissionType    string     `json:"mission_type"`
	CurrentValue   int        `json:"current_value"`
	TargetValue    int        `json:"target_value"`
	Status         string     `json:"status"`
	ActiveEndsAt   *time.Time `json:"active_ends_at,omitempty"`
}

type LeaderboardSummary struct {
	Participants int `json:"participants"`
}

type CountdownSummary struct {
	ResetEndsAt      time.Time `json:"reset_ends_at"`
	SecondsRemaining int64     `json:"seconds_remaining"`
}

type ChallengeMetadataResponse struct {
	Challenge          ChallengeSummary   `json:"challenge"`
	AttemptState       *AttemptSummary    `json:"attempt_state,omitempty"`
	Streak             StreakSummary      `json:"streak"`
	MissionsSummary    []MissionSummary   `json:"missions_summary"`
	LeaderboardSummary LeaderboardSummary `json:"leaderboard_summary"`
	Countdown          *CountdownSummary  `json:"countdown,omitempty"`
}

type ChallengeAttemptResponse struct {
	Challenge ChallengeSummary `json:"challenge"`
	Attempt   AttemptSummary   `json:"attempt"`
	Game      *games.GameDTO   `json:"game,omitempty"`
}

type CreateSharedChallengeRequest struct {
	MapID        uuid.UUID `json:"map_id"`
	RoundCount   int       `json:"round_count"`
	TimerSeconds *int      `json:"timer_seconds"`
	DisplayLabel *string   `json:"display_label"`
}

type ResultResponse struct {
	Challenge       ChallengeSummary `json:"challenge"`
	Attempt         AttemptSummary   `json:"attempt"`
	Visible         bool             `json:"visible"`
	TotalScore      *int             `json:"total_score,omitempty"`
	TotalDistance   *int             `json:"total_distance_meters,omitempty"`
	RoundResults    []RoundResultDTO `json:"round_results,omitempty"`
	RankContext     *json.RawMessage `json:"rank_context,omitempty"`
	Streak          *StreakSummary   `json:"streak,omitempty"`
	MissionsSummary []MissionSummary `json:"missions_summary,omitempty"`
	Message         string           `json:"message,omitempty"`
}

type RoundResultDTO struct {
	RoundNumber    int `json:"round_number"`
	Score          int `json:"score"`
	DistanceMeters int `json:"distance_meters"`
}

type LeaderboardResponse struct {
	Challenge ChallengeSummary      `json:"challenge"`
	Entries   []LeaderboardEntryDTO `json:"entries"`
	Page      PageInfo              `json:"page"`
}

type LeaderboardEntryDTO struct {
	Rank                 int       `json:"rank"`
	DisplayName          string    `json:"display_name"`
	Score                int       `json:"score"`
	CompletionDurationMS *int64    `json:"completion_duration_ms,omitempty"`
	CompletedAt          time.Time `json:"completed_at"`
	CurrentPlayer        bool      `json:"current_player,omitempty"`
}

type PageInfo struct {
	Limit      int     `json:"limit"`
	NextCursor *string `json:"next_cursor,omitempty"`
}
