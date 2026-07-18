package competitive

import (
	"time"

	"github.com/google/uuid"

	apphttp "github.com/raven/geoguess/backend/internal/http"
)

// RankDTO is a competitive rank code projection (match progression).
type RankDTO struct {
	Code string `json:"code"`
}

// RankDetailDTO is the richer rank projection for profile/leaderboard screens.
type RankDetailDTO struct {
	Code     string `json:"code"`
	NameKey  string `json:"name_key"`
	Division *int   `json:"division"`
}

// PlacementProgressDTO is visible during the five placement matches.
type PlacementProgressDTO struct {
	Completed int `json:"completed"`
	Required  int `json:"required"`
}

// SeasonResetDTO documents the soft-reset formula shown to clients.
type SeasonResetDTO struct {
	Anchor        int `json:"anchor"`
	FactorPercent int `json:"factor_percent"`
}

// SeasonSummaryDTO is a season projection for profile/history/list responses.
type SeasonSummaryDTO struct {
	ID               uuid.UUID      `json:"id"`
	Sequence         int            `json:"sequence"`
	Slug             string         `json:"slug"`
	Name             string         `json:"name"`
	Status           string         `json:"status"`
	StartsAt         time.Time      `json:"starts_at"`
	EndsAt           time.Time      `json:"ends_at"`
	ClosedAt         *time.Time     `json:"closed_at,omitempty"`
	Reset            SeasonResetDTO `json:"reset"`
	Top500MinMatches int            `json:"top500_min_matches"`
}

// SeasonListEntryDTO is one row in GET /competitive/seasons.
type SeasonListEntryDTO struct {
	Season        SeasonSummaryDTO `json:"season"`
	FinalPosition *int             `json:"final_position,omitempty"`
	EndingRank    *RankDetailDTO   `json:"ending_rank,omitempty"`
	PeakRank      *RankDetailDTO   `json:"peak_rank,omitempty"`
}

// SeasonsResponse is GET /competitive/seasons.
type SeasonsResponse struct {
	Data []SeasonListEntryDTO `json:"data"`
}

// LastChangeDTO is the most recent rating delta on a profile.
type LastChangeDTO struct {
	MatchID uuid.UUID `json:"match_id"`
	Delta   int       `json:"delta"`
}

// ProfileBodyDTO is the caller's active-season competitive profile.
// Rating/rank are null while placements are incomplete.
type ProfileBodyDTO struct {
	Rating              *int                  `json:"rating"`
	PlacementsCompleted int                   `json:"placements_completed"`
	PlacementsRequired  int                   `json:"placements_required"`
	Rank                *RankDetailDTO        `json:"rank"`
	StandardRank        *RankDetailDTO        `json:"standard_rank"`
	DivisionProgress    *int                  `json:"division_progress"`
	MatchesPlayed       int                   `json:"matches_played"`
	Wins                int                   `json:"wins"`
	Losses              int                   `json:"losses"`
	Draws               int                   `json:"draws"`
	Abandons            int                   `json:"abandons"`
	PeakRating          *int                  `json:"peak_rating"`
	Top500Position      *int                  `json:"top500_position"`
	LastChange          *LastChangeDTO        `json:"last_change,omitempty"`
	Placement           *PlacementProgressDTO `json:"placement,omitempty"`
}

// ProfileResponse is GET /competitive/profile.
type ProfileResponse struct {
	Season  SeasonSummaryDTO `json:"season"`
	Profile ProfileBodyDTO   `json:"profile"`
}

// PublicPlayerDTO is the privacy-safe public identity on leaderboard rows.
type PublicPlayerDTO struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
}

// LeaderboardEntryDTO is one eligible top-500 / leaderboard row.
type LeaderboardEntryDTO struct {
	Position      int             `json:"position"`
	Player        PublicPlayerDTO `json:"player"`
	Rating        int             `json:"rating"`
	Wins          int             `json:"wins"`
	MatchesPlayed int             `json:"matches_played"`
	StandardRank  RankDetailDTO   `json:"standard_rank"`
	WorldLegend   bool            `json:"world_legend"`
	Rank          RankDetailDTO   `json:"rank"`
}

// ViewerEntryDTO is the caller's standing projection relative to the leaderboard.
type ViewerEntryDTO struct {
	Position            *int                  `json:"position"`
	Eligible            bool                  `json:"eligible"`
	Rating              *int                  `json:"rating"`
	Wins                int                   `json:"wins"`
	MatchesPlayed       int                   `json:"matches_played"`
	PlacementsCompleted int                   `json:"placements_completed"`
	PlacementsRequired  int                   `json:"placements_required"`
	StandardRank        *RankDetailDTO        `json:"standard_rank"`
	WorldLegend         bool                  `json:"world_legend"`
	Rank                *RankDetailDTO        `json:"rank"`
	Placement           *PlacementProgressDTO `json:"placement,omitempty"`
}

// LeaderboardResponse is GET /competitive/leaderboard and closed-season variant.
type LeaderboardResponse struct {
	Season SeasonSummaryDTO      `json:"season"`
	Data   []LeaderboardEntryDTO `json:"data"`
	Viewer *ViewerEntryDTO       `json:"viewer,omitempty"`
	Page   apphttp.PageInfo      `json:"page"`
}

// HistoryEntryDTO is one rating-change row for the caller.
type HistoryEntryDTO struct {
	ID             uuid.UUID             `json:"id"`
	MatchID        uuid.UUID             `json:"match_id"`
	SeasonID       uuid.UUID             `json:"season_id"`
	Playlist       string                `json:"playlist"`
	Format         string                `json:"format"`
	Outcome        string                `json:"outcome"`
	OldRating      *int                  `json:"old_rating"`
	NewRating      *int                  `json:"new_rating"`
	BaseDelta      *int                  `json:"base_delta"`
	AbandonPenalty *int                  `json:"abandon_penalty"`
	TotalDelta     *int                  `json:"total_delta"`
	OldRank        *RankDTO              `json:"old_rank"`
	NewRank        *RankDTO              `json:"new_rank"`
	CreatedAt      time.Time             `json:"created_at"`
	Placement      *PlacementProgressDTO `json:"placement,omitempty"`
}

// HistoryResponse is GET /competitive/history.
type HistoryResponse struct {
	Data []HistoryEntryDTO `json:"data"`
	Page apphttp.PageInfo  `json:"page"`
}

// PlayerProgressionDTO projects one participant's rating impact for match results.
// While placements are incomplete, rating and rank fields are omitted (nil).
type PlayerProgressionDTO struct {
	UserID         uuid.UUID             `json:"user_id"`
	Applied        bool                  `json:"applied"`
	Reason         *string               `json:"reason,omitempty"`
	OldRating      *int                  `json:"old_rating,omitempty"`
	BaseDelta      *int                  `json:"base_delta,omitempty"`
	AbandonPenalty *int                  `json:"abandon_penalty,omitempty"`
	TotalDelta     *int                  `json:"total_delta,omitempty"`
	NewRating      *int                  `json:"new_rating,omitempty"`
	OldRank        *RankDTO              `json:"old_rank,omitempty"`
	NewRank        *RankDTO              `json:"new_rank,omitempty"`
	Placement      *PlacementProgressDTO `json:"placement,omitempty"`
	Top500Position *int                  `json:"top500_position,omitempty"`
	Outcome        string                `json:"outcome,omitempty"`
	ExpectedBPS    *int                  `json:"expected_bps,omitempty"`
}

// MatchProgressionResult is the outcome of exact-once ranked finalization.
type MatchProgressionResult struct {
	MatchID  uuid.UUID
	SeasonID uuid.UUID
	// Replay is true when progression was already finalized (idempotent return).
	Replay bool
	// Pending is true when finalization could not complete and should be retried.
	Pending     bool
	FinalizedAt time.Time
	Changes     []RatingChange
	// ByUser projects visibility-aware DTOs for each participant.
	ByUser map[uuid.UUID]PlayerProgressionDTO
}

// StandingSnapshot is the rating/placement view for matchmaking eligibility.
type StandingSnapshot struct {
	UserID              uuid.UUID
	SeasonID            uuid.UUID
	Rating              int
	PlacementsCompleted int
	// RankCode is empty while placements are incomplete.
	RankCode string
	// NamedRankTier is a coarse tier index for party spread (0 when hidden).
	NamedRankTier int
}

// CasualProgression returns the fixed casual no-rating projection.
func CasualProgression(userID uuid.UUID) PlayerProgressionDTO {
	reason := "casual"
	return PlayerProgressionDTO{
		UserID:  userID,
		Applied: false,
		Reason:  &reason,
	}
}

// PendingProgression returns a recoverable finalization-pending projection.
func PendingProgression(userID uuid.UUID) PlayerProgressionDTO {
	reason := "progression_pending"
	return PlayerProgressionDTO{
		UserID:  userID,
		Applied: false,
		Reason:  &reason,
	}
}

// ProjectPlayerProgression builds a visibility-aware DTO from a stored change and post-match standing.
func ProjectPlayerProgression(change RatingChange, placementsAfter int) PlayerProgressionDTO {
	dto := PlayerProgressionDTO{
		UserID:  change.UserID,
		Applied: true,
		Outcome: change.Outcome,
	}

	// Placement progress is always visible.
	if placementsAfter < PlacementsRequired {
		p := placementsAfter
		dto.Placement = &PlacementProgressDTO{
			Completed: p,
			Required:  PlacementsRequired,
		}
		// During placements, rating/rank stay hidden.
		return dto
	}

	// Fully ranked: expose numbers. After the 5th placement completes, reveal.
	old := change.OldRating
	base := change.BaseDelta
	pen := change.AbandonPenalty
	total := change.TotalDelta
	newR := change.NewRating
	exp := change.ExpectedBPS
	dto.OldRating = &old
	dto.BaseDelta = &base
	dto.AbandonPenalty = &pen
	dto.TotalDelta = &total
	dto.NewRating = &newR
	dto.ExpectedBPS = &exp

	// If this match completed placements (was <5 before, now ==5), old rank may still be hidden.
	if change.OldRankCode != nil && *change.OldRankCode != "" {
		dto.OldRank = &RankDTO{Code: *change.OldRankCode}
	}
	if change.NewRankCode != nil && *change.NewRankCode != "" {
		dto.NewRank = &RankDTO{Code: *change.NewRankCode}
	}
	// Once placements are done, placement field is null (not in progress).
	return dto
}
