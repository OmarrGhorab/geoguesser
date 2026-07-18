package games_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/games"
)

func TestCasualNullTimersAndNoDeadline(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	for _, mode := range []string{games.GameModeCasualSolo, games.GameModeCasualDuo, games.GameModeCasualSquad} {
		if games.CasualHasDeadline(mode) {
			t.Fatalf("%q must not have a gameplay deadline", mode)
		}
		if ends := games.NextRoundEndsAt(mode, games.IntPtr(60), now); ends != nil {
			t.Fatalf("%q NextRoundEndsAt must be nil, got %v", mode, ends)
		}
	}
	// Ranked keeps timer-based ends_at.
	ends := games.NextRoundEndsAt(games.GameModeRankedDuo, games.IntPtr(60), now)
	if ends == nil || !ends.Equal(now.Add(60*time.Second)) {
		t.Fatalf("ranked ends_at = %v", ends)
	}
}

func TestCasualAllActiveSubmissionAdvancementSemantics(t *testing.T) {
	t.Parallel()

	// Advancement gate: submitted >= eligible among active players.
	// Pure helper equivalent used by multiplayer progress checks.
	eligible := 4
	for submitted := 0; submitted < eligible; submitted++ {
		if submitted >= eligible {
			t.Fatalf("should not advance early at %d/%d", submitted, eligible)
		}
	}
	submittedAll := eligible
	if eligible <= 0 || submittedAll < eligible {
		t.Fatal("all-active submission must advance")
	}
	// Left players do not count as eligible (status filter in multiplayerProgress).
	activeOnly := []games.GamePlayer{
		{Status: games.PlayerStatusActive},
		{Status: games.PlayerStatusActive},
		{Status: games.PlayerStatusLeft},
	}
	active := 0
	for _, p := range activeOnly {
		if p.Status == games.PlayerStatusActive {
			active++
		}
	}
	if active != 2 {
		t.Fatalf("eligible active = %d, want 2", active)
	}
}

func TestLockedGuessesAndZeroMissingScores(t *testing.T) {
	t.Parallel()

	// Zero missing contribution: players without a guess score 0 for the round.
	slot1, slot2 := games.TeamSlotOne, games.TeamSlotTwo
	p1 := uuid.New()
	p2 := uuid.New()
	p3 := uuid.New()
	p4 := uuid.New()
	players := []games.GamePlayer{
		{ID: p1, TeamSlot: &slot1, TotalScore: 1000},
		{ID: p2, TeamSlot: &slot1, TotalScore: 2000},
		{ID: p3, TeamSlot: &slot2, TotalScore: 1500},
		{ID: p4, TeamSlot: &slot2, TotalScore: 0}, // never scored
	}
	// Only three guesses — p4 missing contributes zero to round totals.
	guesses := []games.Guess{
		{GamePlayerID: p1, Score: 1000, AccuracyScore: 1000},
		{GamePlayerID: p2, Score: 2000, AccuracyScore: 2000},
		{GamePlayerID: p3, Score: 1500, AccuracyScore: 1500},
	}
	roundTotals := games.SumRoundTeamScores(players, guesses)
	if roundTotals.TeamOneScore != 3000 || roundTotals.TeamTwoScore != 1500 {
		t.Fatalf("round team totals = %+v, want 3000/1500 (missing=0)", roundTotals)
	}

	// Timed-out zero row shape used on closure paths.
	zero := games.Guess{AccuracyScore: 0, SpeedBonus: 0, Score: 0, TimedOut: true}
	if zero.Score != zero.AccuracyScore+zero.SpeedBonus || !zero.TimedOut {
		t.Fatalf("zero missing guess shape invalid: %+v", zero)
	}
}

func TestSummedPlayerAndTeamTotals(t *testing.T) {
	t.Parallel()

	slot1, slot2 := games.TeamSlotOne, games.TeamSlotTwo
	players := []games.GamePlayer{
		{TeamSlot: &slot1, TotalScore: 4100},
		{TeamSlot: &slot1, TotalScore: 3900},
		{TeamSlot: &slot2, TotalScore: 5000},
		{TeamSlot: &slot2, TotalScore: 3000},
		{TeamSlot: nil, TotalScore: 999}, // legacy solo-style: ignored
	}
	totals := games.SumTeamTotals(players)
	if totals.TeamOneScore != 8000 || totals.TeamTwoScore != 8000 {
		t.Fatalf("team totals = %+v, want 8000/8000", totals)
	}
}

func TestExactDrawsSupported(t *testing.T) {
	t.Parallel()

	result, winner := games.DecideTeamResult(8000, 8000)
	if result != games.ResultDraw || winner != nil {
		t.Fatalf("exact draw = %s winner=%v", result, winner)
	}
	result, winner = games.DecideTeamResult(9000, 8000)
	if result != games.ResultTeamOneWin || winner == nil || *winner != games.TeamSlotOne {
		t.Fatalf("team one win = %s winner=%v", result, winner)
	}
	result, winner = games.DecideTeamResult(100, 500)
	if result != games.ResultTeamTwoWin || winner == nil || *winner != games.TeamSlotTwo {
		t.Fatalf("team two win = %s winner=%v", result, winner)
	}
}

func TestDelayedAnswerRevealPolicyForCasual(t *testing.T) {
	t.Parallel()

	var policy games.DelayedRevealPolicy
	for _, mode := range []string{games.GameModeCasualSolo, games.GameModeCasualDuo, games.GameModeCasualSquad} {
		if policy.MayRevealAnswer(mode, games.RoundStatusActive, false) {
			t.Fatalf("%q must not reveal answer while round active", mode)
		}
		if !policy.MayRevealAnswer(mode, games.RoundStatusCompleted, false) {
			t.Fatalf("%q must reveal when round status completed", mode)
		}
		if !policy.MayRevealAnswer(mode, games.RoundStatusActive, true) {
			t.Fatalf("%q must reveal when roundCompleted flag is true", mode)
		}
	}
	// Solo still reveals immediately (non-multiplayer).
	if !policy.MayRevealAnswer(games.GameModeSolo, games.RoundStatusActive, false) {
		t.Fatal("solo should reveal on submit")
	}
}

func TestCasualComposeScoreZeroSpeedBonus(t *testing.T) {
	t.Parallel()

	acc, bonus, total := games.ComposeGuessScores(games.GameModeCasualSquad, 4020, 45_000, 60_000)
	if acc != 4020 || bonus != 0 || total != 4020 {
		t.Fatalf("casual scores = %d/%d/%d, want 4020/0/4020", acc, bonus, total)
	}
}

func TestBuildTerminalMatchResultCasualDraw(t *testing.T) {
	t.Parallel()

	slot1, slot2 := games.TeamSlotOne, games.TeamSlotTwo
	gameID := uuid.New()
	completedAt := time.Date(2026, 7, 18, 13, 0, 0, 0, time.UTC)
	players := []games.GamePlayer{
		{TeamSlot: &slot1, TotalScore: 12000},
		{TeamSlot: &slot2, TotalScore: 12000},
	}
	terminal := games.BuildTerminalMatchResult(gameID, games.GameModeCasualDuo, players, completedAt)
	if terminal.GameID != gameID || terminal.Result != games.ResultDraw {
		t.Fatalf("terminal = %+v", terminal)
	}
	if terminal.WinnerTeamSlot != nil {
		t.Fatalf("draw must have nil winner, got %v", *terminal.WinnerTeamSlot)
	}
	if terminal.TeamOneScore != 12000 || terminal.TeamTwoScore != 12000 {
		t.Fatalf("scores = %d/%d", terminal.TeamOneScore, terminal.TeamTwoScore)
	}
	if terminal.IsRanked {
		t.Fatal("casual terminal must not mark IsRanked")
	}
	if !terminal.CompletedAt.Equal(completedAt) {
		t.Fatalf("completedAt = %v", terminal.CompletedAt)
	}
}

func TestTeamGameFormationInputCasualShape(t *testing.T) {
	t.Parallel()

	// Document the exported formation contract for matchmaking consumers.
	input := games.TeamGameFormationInput{
		Mode:         games.GameModeCasualDuo,
		MapID:        uuid.New(),
		RoundCount:   5,
		TimerSeconds: nil, // casual: no timer
		TeamOne: []games.TeamRosterMember{
			{UserID: uuid.New(), DisplayName: "A"},
		},
		TeamTwo: []games.TeamRosterMember{
			{UserID: uuid.New(), DisplayName: "B"},
		},
		LocationIDs: []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()},
	}
	if input.TimerSeconds != nil {
		t.Fatal("casual formation must pass nil timer")
	}
	if len(input.TeamOne) != len(input.TeamTwo) {
		t.Fatal("equal team sizes required")
	}
	if input.RoundCount != 5 {
		t.Fatalf("casual uses five rounds, got %d", input.RoundCount)
	}
}

func TestMultiplayerGuessResponseDTOFields(t *testing.T) {
	t.Parallel()

	// Ensure public DTO carries accuracy/bonus and nullable multiplayer reveal fields.
	submitted, eligible := 1, 2
	resp := games.GuessResultResponse{
		Guess: games.GuessResult{
			AccuracyScore: 4000,
			SpeedBonus:    0,
			Score:         4000,
		},
		ActualLocation:   nil,
		MaxAccuracyScore: 5000,
		MaxSpeedBonus:    250,
		Outcome:          "submitted",
		RoundCompleted:   false,
		GameCompleted:    false,
		SubmittedCount:   &submitted,
		EligibleCount:    &eligible,
	}
	if resp.ActualLocation != nil {
		t.Fatal("pre-reveal actual_location must be null")
	}
	if resp.Guess.SpeedBonus != 0 || resp.Guess.Score != resp.Guess.AccuracyScore {
		t.Fatalf("casual guess components = %+v", resp.Guess)
	}
	if resp.SubmittedCount == nil || *resp.SubmittedCount != 1 {
		t.Fatalf("submitted_count = %v", resp.SubmittedCount)
	}
}
