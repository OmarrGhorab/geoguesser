package games

import (
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/maps"
)

// Team slot constants for matchmade Casual/Ranked games.
const (
	TeamSlotOne = 1
	TeamSlotTwo = 2
)

// Terminal result codes produced by games for match lifecycle consumers.
const (
	ResultTeamOneWin = "team_one_win"
	ResultTeamTwoWin = "team_two_win"
	ResultDraw       = "draw"
	ResultForfeit    = "forfeit"
	ResultAbandoned  = "abandoned"
	ResultCancelled  = "cancelled"
)

// MultiplayerStart is returned when a multiplayer game is activated.
type MultiplayerStart struct {
	Game         Game
	CurrentRound Round
}

// MultiplayerGuessOutcome summarizes a multiplayer guess write and any advancement.
type MultiplayerGuessOutcome struct {
	Guess           Guess
	RoundID         uuid.UUID
	SubmittedCount  int
	EligibleCount   int
	RoundCompleted  bool
	GameCompleted   bool
	NextRoundNumber *int
	NextRoundID     *uuid.UUID
	// Terminal is set when the final round closes the game (matchmade team totals).
	Terminal *TerminalMatchResult
}

// MultiplayerRoundState is the safe current-round view for multiplayer consumers.
type MultiplayerRoundState struct {
	RoundID            uuid.UUID
	RoundNumber        int
	Status             string
	StartsAt           *time.Time
	EndsAt             *time.Time
	Provider           string
	ProviderRef        string
	MediaURL           string
	Attribution        *string
	SubmittedCount     int
	EligibleCount      int
	SubmittedPlayerIDs []uuid.UUID `gorm:"-"`
}

// TeamRosterMember seeds one player on a team during matchmade formation.
type TeamRosterMember struct {
	UserID      uuid.UUID
	DisplayName string
}

// TeamGameFormationInput creates a matchmade casual/ranked team game with slots.
// Exported for matchmaking formation so durable game rows stay games-owned.
type TeamGameFormationInput struct {
	Mode       string
	MapID      uuid.UUID
	RoundCount int
	// TimerSeconds is nil for Casual (no deadline). Ranked typically sets 60.
	TimerSeconds   *int
	ScoringVersion int
	// StartedAt stamps game.started_at (and player joined_at).
	StartedAt time.Time
	// RoundStartsAt is the first round starts_at (ranked may schedule a countdown).
	RoundStartsAt time.Time
	TeamOne       []TeamRosterMember
	TeamTwo       []TeamRosterMember
	LocationIDs   []uuid.UUID
}

// TeamGameFormationResult is the durable game state after formation.
type TeamGameFormationResult struct {
	Game    Game
	Players []GamePlayer
	Rounds  []Round
}

// TeamTotals is the sum of player total scores by team slot.
type TeamTotals struct {
	TeamOneScore int
	TeamTwoScore int
}

// SumTeamTotals sums player TotalScore by team slot. Players without a slot contribute 0.
// Missing scores are expected to already be zeroed on player totals (or not submitted).
func SumTeamTotals(players []GamePlayer) TeamTotals {
	var totals TeamTotals
	for _, p := range players {
		if p.TeamSlot == nil {
			continue
		}
		switch *p.TeamSlot {
		case TeamSlotOne:
			totals.TeamOneScore += p.TotalScore
		case TeamSlotTwo:
			totals.TeamTwoScore += p.TotalScore
		}
	}
	return totals
}

// SumRoundTeamScores sums per-guess scores for a single round by team slot.
// Players without a guess for the round contribute zero (caller should pass only existing guesses
// or use BuildRoundTeamScores which zero-fills missing participants).
func SumRoundTeamScores(players []GamePlayer, guesses []Guess) TeamTotals {
	scoreByPlayer := make(map[uuid.UUID]int, len(guesses))
	for _, g := range guesses {
		scoreByPlayer[g.GamePlayerID] = g.Score
	}
	var totals TeamTotals
	for _, p := range players {
		if p.TeamSlot == nil {
			continue
		}
		score := scoreByPlayer[p.ID] // missing => 0
		switch *p.TeamSlot {
		case TeamSlotOne:
			totals.TeamOneScore += score
		case TeamSlotTwo:
			totals.TeamTwoScore += score
		}
	}
	return totals
}

// DecideTeamResult compares team totals and returns a terminal result code plus optional winner slot.
// Exact equal totals are draws (winner nil).
func DecideTeamResult(teamOneScore, teamTwoScore int) (result string, winnerTeamSlot *int) {
	if teamOneScore > teamTwoScore {
		slot := TeamSlotOne
		return ResultTeamOneWin, &slot
	}
	if teamTwoScore > teamOneScore {
		slot := TeamSlotTwo
		return ResultTeamTwoWin, &slot
	}
	return ResultDraw, nil
}

// BuildTerminalMatchResult assembles the games-side terminal payload for lifecycle hooks.
func BuildTerminalMatchResult(gameID uuid.UUID, mode string, players []GamePlayer, completedAt time.Time) TerminalMatchResult {
	totals := SumTeamTotals(players)
	result, winner := DecideTeamResult(totals.TeamOneScore, totals.TeamTwoScore)
	return TerminalMatchResult{
		GameID:         gameID,
		Result:         result,
		WinnerTeamSlot: winner,
		TeamOneScore:   totals.TeamOneScore,
		TeamTwoScore:   totals.TeamTwoScore,
		CompletedAt:    completedAt,
		IsRanked:       IsRankedMode(mode),
	}
}

// CasualHasDeadline reports whether a casual multiplayer game uses a round ends_at deadline.
// Casual never uses gameplay timers.
func CasualHasDeadline(mode string) bool {
	return !IsCasualMode(mode)
}

// NextRoundEndsAt returns the ends_at for an advancing multiplayer round.
// Casual modes always return nil (no deadline).
func NextRoundEndsAt(mode string, timerSeconds *int, now time.Time) *time.Time {
	if IsCasualMode(mode) {
		return nil
	}
	if timerSeconds == nil {
		return nil
	}
	v := now.Add(time.Duration(*timerSeconds) * time.Second)
	return &v
}

// IntPtr returns a pointer to v (team slot / count helpers).
func IntPtr(v int) *int { return &v }

func roundsFromSelected(gameID uuid.UUID, selected []maps.SelectedLocation, count int) []Round {
	rounds := make([]Round, count)
	for i := range rounds {
		rounds[i] = Round{
			GameID:      gameID,
			LocationID:  selected[i].ID,
			RoundNumber: i + 1,
			Status:      RoundStatusPending,
		}
	}
	return rounds
}
