package matchplay

import (
	"time"

	"github.com/google/uuid"
)

// MatchSnapshotResponse is the authorized GET /matches/{id} envelope.
type MatchSnapshotResponse struct {
	Match MatchSnapshotDTO `json:"match"`
}

// MatchSnapshotDTO is the caller-authorized canonical match snapshot.
type MatchSnapshotDTO struct {
	ID              uuid.UUID            `json:"id"`
	GameID          uuid.UUID            `json:"game_id"`
	Playlist        string               `json:"playlist"`
	Format          string               `json:"format"`
	Status          string               `json:"status"`
	Result          *string              `json:"result"`
	TeamSize        int                  `json:"team_size"`
	Viewer          ViewerProjection     `json:"viewer"`
	Teams           []TeamProjection     `json:"teams"`
	Round           *RoundProjection     `json:"round,omitempty"`
	LastRoundResult *RevealedRoundResult `json:"last_round_result,omitempty"`
	TeamMarkers     []TeamMarkerDTO      `json:"team_markers"`
	RealtimeVersion int64                `json:"realtime_version"`
	FormedAt        time.Time            `json:"formed_at"`
	// TimerSeconds is 60 for Ranked, null/omitted for Casual (no scoring timer).
	TimerSeconds *int `json:"timer_seconds,omitempty"`
}

// ViewerProjection is the caller's private match view.
type ViewerProjection struct {
	UserID                   uuid.UUID   `json:"user_id"`
	GamePlayerID             uuid.UUID   `json:"game_player_id"`
	TeamSlot                 int         `json:"team_slot"`
	Submitted                bool        `json:"submitted"`
	CanChat                  bool        `json:"can_chat"`
	AllowedSpectatePlayerIDs []uuid.UUID `json:"allowed_spectate_player_ids"`
}

// TeamProjection is one equal team in a match snapshot or result.
type TeamProjection struct {
	Slot    int                     `json:"slot"`
	Score   int                     `json:"score"`
	Players []ParticipantProjection `json:"players"`
}

// ParticipantProjection is a roster entry visible to the authorized caller.
// Private fields (guess coords, distance, accuracy, bonus, answer) are omitted
// before reveal; opponent individual totals may be redacted pre-reveal.
type ParticipantProjection struct {
	GamePlayerID uuid.UUID `json:"game_player_id"`
	UserID       uuid.UUID `json:"user_id"`
	DisplayName  string    `json:"display_name"`
	Status       string    `json:"status"`
	Submitted    bool      `json:"submitted"`
	// TotalScore is omitted (json omitempty via pointer) when redacted pre-reveal for opponents.
	TotalScore *int `json:"total_score,omitempty"`
}

// RoundProjection is the safe current-round view (no answer coordinates).
type RoundProjection struct {
	ID             uuid.UUID      `json:"id"`
	Number         int            `json:"number"`
	Status         string         `json:"status"`
	StartsAt       *time.Time     `json:"starts_at"`
	EndsAt         *time.Time     `json:"ends_at"`
	Media          *RoundMediaDTO `json:"media,omitempty"`
	SubmittedCount int            `json:"submitted_count"`
	EligibleCount  int            `json:"eligible_count"`
}

// RoundMediaDTO is provider-safe media for the current round.
type RoundMediaDTO struct {
	Type        string  `json:"type"`
	PanoramaID  string  `json:"panorama_id,omitempty"`
	URL         string  `json:"url,omitempty"`
	Attribution *string `json:"attribution,omitempty"`
}

// TeamMarkerDTO is a current proposed marker (team-private; may be empty on HTTP snapshot).
type TeamMarkerDTO struct {
	UserID    uuid.UUID `json:"user_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Version   int64     `json:"version"`
}

// RevealedRoundResult is a fully revealed round after closure.
type RevealedRoundResult struct {
	RoundID        uuid.UUID             `json:"round_id"`
	RoundNumber    int                   `json:"round_number"`
	ActualLocation RevealedLocationDTO   `json:"actual_location"`
	Guesses        []RevealedGuessDTO    `json:"guesses"`
	Teams          []TeamScoreProjection `json:"teams"`
}

// RevealedLocationDTO is the answer after reveal.
type RevealedLocationDTO struct {
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	CountryCode string  `json:"country_code"`
	Region      *string `json:"region,omitempty"`
	Locality    *string `json:"locality,omitempty"`
}

// RevealedGuessDTO is one participant's revealed guess or timeout.
type RevealedGuessDTO struct {
	GamePlayerID   uuid.UUID  `json:"game_player_id"`
	UserID         uuid.UUID  `json:"user_id"`
	TeamSlot       int        `json:"team_slot"`
	Latitude       *float64   `json:"latitude"`
	Longitude      *float64   `json:"longitude"`
	DistanceMeters *int       `json:"distance_meters"`
	AccuracyScore  int        `json:"accuracy_score"`
	SpeedBonus     int        `json:"speed_bonus"`
	Score          int        `json:"score"`
	TimedOut       bool       `json:"timed_out"`
	SubmittedAt    *time.Time `json:"submitted_at,omitempty"`
}

// TeamScoreProjection is a team total without full roster.
type TeamScoreProjection struct {
	Slot  int `json:"slot"`
	Score int `json:"score"`
}

// RoundResultsResponse wraps GET /matches/{id}/rounds/{roundId}/results.
type RoundResultsResponse struct {
	Result RevealedRoundResult `json:"result"`
}

// TerminalResultResponse wraps GET /matches/{id}/results.
type TerminalResultResponse struct {
	Result         string                `json:"result"`
	WinnerTeamSlot *int                  `json:"winner_team_slot"`
	Teams          []TeamProjection      `json:"teams"`
	Rounds         []RevealedRoundResult `json:"rounds,omitempty"`
	Progression    ProgressionDTO        `json:"progression"`
}

// ProgressionDTO describes competitive rating impact for the caller.
// Casual always returns Applied=false with Reason="casual".
// Ranked pending finalization uses Applied=false with Reason="progression_pending"
// only in defensive projections; the HTTP path prefers 202 progression_pending.
type ProgressionDTO struct {
	Applied        bool     `json:"applied"`
	Reason         *string  `json:"reason,omitempty"`
	OldRating      *int     `json:"old_rating,omitempty"`
	BaseDelta      *int     `json:"base_delta,omitempty"`
	AbandonPenalty *int     `json:"abandon_penalty,omitempty"`
	TotalDelta     *int     `json:"total_delta,omitempty"`
	NewRating      *int     `json:"new_rating,omitempty"`
	OldRank        *RankDTO `json:"old_rank,omitempty"`
	NewRank        *RankDTO `json:"new_rank,omitempty"`
	// Placement is 1-based placement match number after this result while still placing.
	Placement *int `json:"placement,omitempty"`
	// PlacementsCompleted is how many of the five placement matches are done after this result.
	PlacementsCompleted *int `json:"placements_completed,omitempty"`
	// PlacementsRequired is always 5 when placement fields are present.
	PlacementsRequired *int `json:"placements_required,omitempty"`
	Top500Position     *int `json:"top500_position,omitempty"`
}

// RankDTO is a competitive rank code projection.
type RankDTO struct {
	Code string `json:"code"`
}

// LeaveRequest is an optional POST /matches/{id}/leave body.
// Empty body is accepted; unknown fields are rejected by the decoder.
type LeaveRequest struct {
	// Reason is optional client-supplied context; server authority uses explicit_leave.
	Reason *string `json:"reason,omitempty"`
}

// CasualNoProgression returns the fixed casual progression projection.
func CasualNoProgression() ProgressionDTO {
	reason := "casual"
	return ProgressionDTO{
		Applied: false,
		Reason:  &reason,
	}
}

// ProgressionPending returns a defensive pending projection (HTTP uses 202 instead).
func ProgressionPending() ProgressionDTO {
	reason := "progression_pending"
	return ProgressionDTO{
		Applied: false,
		Reason:  &reason,
	}
}

// NewRankedProgression builds an applied ranked progression projection for the caller.
func NewRankedProgression(
	oldRating, baseDelta, abandonPenalty, totalDelta, newRating int,
	oldRankCode, newRankCode string,
	placement, placementsCompleted *int,
	top500 *int,
) ProgressionDTO {
	dto := ProgressionDTO{
		Applied:        true,
		OldRating:      &oldRating,
		BaseDelta:      &baseDelta,
		AbandonPenalty: &abandonPenalty,
		TotalDelta:     &totalDelta,
		NewRating:      &newRating,
		Placement:      placement,
		Top500Position: top500,
	}
	if oldRankCode != "" {
		dto.OldRank = &RankDTO{Code: oldRankCode}
	}
	if newRankCode != "" {
		dto.NewRank = &RankDTO{Code: newRankCode}
	}
	if placementsCompleted != nil {
		dto.PlacementsCompleted = placementsCompleted
		req := 5
		dto.PlacementsRequired = &req
	}
	return dto
}

// NewTeamProjections builds slot-ordered team projections from participants and players.
// redactedUserIDs controls which players have TotalScore omitted (pre-reveal opponents).
func NewTeamProjections(
	participants []MatchParticipant,
	players []GamePlayerRow,
	submitted map[uuid.UUID]bool,
	teamOneScore, teamTwoScore int,
	redactedUserIDs map[uuid.UUID]bool,
) []TeamProjection {
	playerByID := make(map[uuid.UUID]GamePlayerRow, len(players))
	for _, p := range players {
		playerByID[p.ID] = p
	}

	teams := []TeamProjection{
		{Slot: 1, Score: teamOneScore, Players: make([]ParticipantProjection, 0)},
		{Slot: 2, Score: teamTwoScore, Players: make([]ParticipantProjection, 0)},
	}

	for _, mp := range participants {
		proj := ParticipantProjection{
			GamePlayerID: mp.GamePlayerID,
			UserID:       mp.UserID,
			DisplayName:  "Player",
			Status:       PlayerStatusActive,
			Submitted:    submitted[mp.GamePlayerID],
		}
		if gp, ok := playerByID[mp.GamePlayerID]; ok {
			proj.DisplayName = gp.DisplayName
			proj.Status = projectPlayerStatus(mp, gp)
			if !redactedUserIDs[mp.UserID] {
				score := gp.TotalScore
				proj.TotalScore = &score
			}
		} else if !redactedUserIDs[mp.UserID] {
			zero := 0
			proj.TotalScore = &zero
		}
		// Prefer match-level abandon status.
		if mp.AbandonedAt != nil {
			proj.Status = PlayerStatusLeft
		}
		idx := mp.TeamSlot - 1
		if idx < 0 || idx > 1 {
			continue
		}
		teams[idx].Players = append(teams[idx].Players, proj)
	}
	return teams
}

func projectPlayerStatus(mp MatchParticipant, gp GamePlayerRow) string {
	if mp.AbandonedAt != nil {
		return PlayerStatusLeft
	}
	switch gp.Status {
	case PlayerStatusDisconnected:
		return PlayerStatusDisconnected
	case PlayerStatusLeft, "kicked":
		return PlayerStatusLeft
	default:
		if !IsActiveParticipant(mp.Status) {
			return PlayerStatusLeft
		}
		return PlayerStatusActive
	}
}
