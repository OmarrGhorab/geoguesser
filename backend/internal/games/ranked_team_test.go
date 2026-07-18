package games_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/games"
	"gorm.io/gorm"
)

// Ranked 1v1/2v2/4v4 PostgreSQL round tests: identical deadlines, all-submit/timeout
// closure, delayed reveal, idempotent retries, and summed bonus totals.

func TestRankedIdenticalDeadlinesAcrossPlayers(t *testing.T) {
	repo, db := setupGamesRepositoryTest(t)
	ctx := context.Background()
	mapID, locationIDs := seedGameMap(t, db, 5)
	now := time.Date(2026, 7, 18, 14, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name     string
		mode     string
		teamSize int
	}{
		{name: "1v1", mode: games.GameModeRankedSolo, teamSize: 1},
		{name: "2v2", mode: games.GameModeRankedDuo, teamSize: 2},
		{name: "4v4", mode: games.GameModeRankedSquad, teamSize: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			formed := formRankedTeamGame(t, repo, db, mapID, locationIDs, tc.mode, tc.teamSize, now)
			if formed.Game.TimerSeconds == nil || *formed.Game.TimerSeconds != games.RankedRoundTimerSeconds {
				t.Fatalf("timer_seconds = %v, want %d", formed.Game.TimerSeconds, games.RankedRoundTimerSeconds)
			}
			var activeRounds []games.Round
			if err := db.Where("game_id = ? AND status = ?", formed.Game.ID, games.RoundStatusActive).Find(&activeRounds).Error; err != nil {
				t.Fatalf("load active rounds: %v", err)
			}
			if len(activeRounds) != 1 {
				t.Fatalf("active rounds = %d, want 1", len(activeRounds))
			}
			round := activeRounds[0]
			if round.StartsAt == nil || round.EndsAt == nil {
				t.Fatalf("ranked round must have starts_at and ends_at, got starts=%v ends=%v", round.StartsAt, round.EndsAt)
			}
			wantEnd := round.StartsAt.Add(time.Duration(games.RankedRoundTimerSeconds) * time.Second)
			if !round.EndsAt.Equal(wantEnd) {
				t.Fatalf("ends_at = %v, want starts+60s = %v", round.EndsAt, wantEnd)
			}
			// All players share one round row (identical deadline), not per-player timers.
			if len(formed.Players) != tc.teamSize*2 {
				t.Fatalf("players = %d, want %d", len(formed.Players), tc.teamSize*2)
			}
			state, err := repo.GetMultiplayerRoundState(ctx, formed.Game.ID)
			if err != nil || state == nil {
				t.Fatalf("round state: %+v err=%v", state, err)
			}
			if state.EndsAt == nil || !state.EndsAt.Equal(*round.EndsAt) {
				t.Fatalf("shared state ends_at = %v, want %v", state.EndsAt, round.EndsAt)
			}
			if state.EligibleCount != tc.teamSize*2 {
				t.Fatalf("eligible = %d, want %d", state.EligibleCount, tc.teamSize*2)
			}
		})
	}
}

func TestRankedAllSubmitClosureAndDelayedReveal(t *testing.T) {
	repo, db := setupGamesRepositoryTest(t)
	ctx := context.Background()
	mapID, locationIDs := seedGameMap(t, db, 2)
	now := time.Date(2026, 7, 18, 15, 0, 0, 0, time.UTC)

	formed := formRankedTeamGame(t, repo, db, mapID, locationIDs, games.GameModeRankedDuo, 2, now)
	round := activeRound(t, db, formed.Game.ID)

	// First three submissions must NOT complete the round or reveal the answer.
	for i := 0; i < 3; i++ {
		submitAt := now.Add(time.Duration(i+1) * time.Second)
		out, answer, err := repo.SubmitMultiplayerGuessTx(ctx, formed.Game.ID, round.ID, formed.Players[i].ID,
			games.Guess{Latitude: 30.04, Longitude: 31.23}, submitAt, games.MultiplayerTxHooks{})
		if err != nil {
			t.Fatalf("guess %d: %v", i, err)
		}
		if out == nil || answer == nil {
			t.Fatalf("guess %d: nil outcome", i)
		}
		if out.RoundCompleted {
			t.Fatalf("guess %d must not complete round early", i)
		}
		if out.Guess.Score != out.Guess.AccuracyScore+out.Guess.SpeedBonus {
			t.Fatalf("score invariant broken: %+v", out.Guess)
		}
		if out.Guess.SpeedBonus <= 0 && out.Guess.AccuracyScore > 0 {
			// Near-start submit with positive accuracy should earn some bonus.
			t.Fatalf("expected positive speed bonus early in ranked window, got %+v", out.Guess)
		}
	}

	// Shared results must stay unrevealed while the round is active.
	if _, err := repo.LoadSharedRoundResults(ctx, formed.Game.ID, round.ID); !errors.Is(err, games.ErrResultsNotReady) {
		t.Fatalf("pre-close shared results err = %v, want ErrResultsNotReady", err)
	}

	// Final submission closes the round (all-active submit).
	out, answer, err := repo.SubmitMultiplayerGuessTx(ctx, formed.Game.ID, round.ID, formed.Players[3].ID,
		games.Guess{Latitude: 30.05, Longitude: 31.24}, now.Add(4*time.Second), games.MultiplayerTxHooks{})
	if err != nil {
		t.Fatalf("final guess: %v", err)
	}
	if out == nil || !out.RoundCompleted || answer == nil {
		t.Fatalf("expected round completion, got %+v", out)
	}
	if out.SubmittedCount != 4 || out.EligibleCount != 4 {
		t.Fatalf("progress after close = %d/%d", out.SubmittedCount, out.EligibleCount)
	}

	shared, err := repo.LoadSharedRoundResults(ctx, formed.Game.ID, round.ID)
	if err != nil || shared == nil {
		t.Fatalf("shared results after close: %+v err=%v", shared, err)
	}
	if shared.ActualLocation.CountryCode == "" {
		t.Fatal("revealed actual_location required after close")
	}
	if len(shared.Guesses) != 4 {
		t.Fatalf("revealed guesses = %d, want 4", len(shared.Guesses))
	}
	for _, g := range shared.Guesses {
		if g.Guess.Score != g.Guess.AccuracyScore+g.Guess.SpeedBonus {
			t.Fatalf("revealed score invariant: %+v", g.Guess)
		}
	}
}

func TestRankedTimeoutClosureZeroMissingScores(t *testing.T) {
	repo, db := setupGamesRepositoryTest(t)
	ctx := context.Background()
	mapID, locationIDs := seedGameMap(t, db, 2)
	now := time.Date(2026, 7, 18, 16, 0, 0, 0, time.UTC)

	formed := formRankedTeamGame(t, repo, db, mapID, locationIDs, games.GameModeRankedDuo, 2, now)
	round := activeRound(t, db, formed.Game.ID)

	// Only one player submits before deadline.
	submitAt := now.Add(10 * time.Second)
	out, _, err := repo.SubmitMultiplayerGuessTx(ctx, formed.Game.ID, round.ID, formed.Players[0].ID,
		games.Guess{Latitude: 30.04, Longitude: 31.23}, submitAt, games.MultiplayerTxHooks{})
	if err != nil {
		t.Fatalf("early guess: %v", err)
	}
	if out.RoundCompleted {
		t.Fatal("single guess must not close duo round")
	}
	earlyBonus := out.Guess.SpeedBonus
	earlyScore := out.Guess.Score
	if earlyBonus <= 0 {
		t.Fatalf("ranked early submit should earn bonus, got %d", earlyBonus)
	}

	// Past deadline: close and zero-fill missing players.
	afterDeadline := round.EndsAt.Add(time.Second)
	closed, err := repo.CloseExpiredMultiplayerRound(ctx, formed.Game.ID, afterDeadline, games.MultiplayerTxHooks{})
	if err != nil {
		t.Fatalf("close expired: %v", err)
	}
	if closed == nil || !closed.RoundCompleted {
		t.Fatalf("expected timeout closure, got %+v", closed)
	}
	if closed.SubmittedCount != 4 || closed.EligibleCount != 4 {
		t.Fatalf("after timeout counts = %d/%d", closed.SubmittedCount, closed.EligibleCount)
	}

	var guesses []games.Guess
	if err := db.Where("round_id = ?", round.ID).Find(&guesses).Error; err != nil {
		t.Fatalf("load guesses: %v", err)
	}
	if len(guesses) != 4 {
		t.Fatalf("guesses after timeout = %d, want 4", len(guesses))
	}
	timedOut := 0
	for _, g := range guesses {
		if g.GamePlayerID == formed.Players[0].ID {
			if g.Score != earlyScore || g.SpeedBonus != earlyBonus {
				t.Fatalf("early submit mutated on timeout close: got %+v want score=%d bonus=%d", g, earlyScore, earlyBonus)
			}
			continue
		}
		if !g.TimedOut || g.Score != 0 || g.AccuracyScore != 0 || g.SpeedBonus != 0 {
			t.Fatalf("missing player must be timed-out zero, got %+v", g)
		}
		timedOut++
	}
	if timedOut != 3 {
		t.Fatalf("timed-out zeros = %d, want 3", timedOut)
	}

	// Late normal submit after close is rejected.
	_, _, err = repo.SubmitMultiplayerGuessTx(ctx, formed.Game.ID, round.ID, formed.Players[1].ID,
		games.Guess{Latitude: 1, Longitude: 1}, afterDeadline.Add(time.Second), games.MultiplayerTxHooks{})
	if !errors.Is(err, games.ErrRoundClosed) {
		t.Fatalf("late submit err = %v, want ErrRoundClosed", err)
	}
}

func TestRankedIdempotentRetryReturnsStoredScores(t *testing.T) {
	repo, db := setupGamesRepositoryTest(t)
	ctx := context.Background()
	mapID, locationIDs := seedGameMap(t, db, 1)
	now := time.Date(2026, 7, 18, 17, 0, 0, 0, time.UTC)

	formed := formRankedTeamGame(t, repo, db, mapID, locationIDs, games.GameModeRankedSolo, 1, now)
	round := activeRound(t, db, formed.Game.ID)
	key := "ranked-idem-" + uuid.NewString()
	lat, lng := 30.0444, 31.2357

	first, _, err := repo.SubmitMultiplayerGuessTx(ctx, formed.Game.ID, round.ID, formed.Players[0].ID,
		games.Guess{Latitude: lat, Longitude: lng, IdempotencyKey: &key}, now.Add(5*time.Second), games.MultiplayerTxHooks{})
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}
	if first == nil {
		t.Fatal("nil first outcome")
	}

	// Service-level idempotent replay uses GetGuessByIdempotencyKey — verify durable row stable.
	loaded, err := repo.GetGuessByIdempotencyKey(ctx, formed.Players[0].ID, key)
	if err != nil || loaded == nil {
		t.Fatalf("load by key: %+v err=%v", loaded, err)
	}
	if loaded.AccuracyScore != first.Guess.AccuracyScore ||
		loaded.SpeedBonus != first.Guess.SpeedBonus ||
		loaded.Score != first.Guess.Score {
		t.Fatalf("stored guess drifted: got %+v want %+v", loaded, first.Guess)
	}
	if loaded.Score != loaded.AccuracyScore+loaded.SpeedBonus {
		t.Fatalf("stored invariant broken: %+v", loaded)
	}

	// Same-key body for retry comparison (service compares lat/lng).
	same, err := repo.GetGuessByIdempotencyKey(ctx, formed.Players[0].ID, key)
	if err != nil || same == nil {
		t.Fatalf("retry load: %v", err)
	}
	if same.Latitude != lat || same.Longitude != lng {
		t.Fatalf("idempotent body = %v,%v", same.Latitude, same.Longitude)
	}

	// Second physical insert for same player/round must fail (locked guess).
	_, _, err = repo.SubmitMultiplayerGuessTx(ctx, formed.Game.ID, round.ID, formed.Players[0].ID,
		games.Guess{Latitude: lat, Longitude: lng, IdempotencyKey: &key}, now.Add(6*time.Second), games.MultiplayerTxHooks{})
	if err == nil {
		t.Fatal("duplicate submit must fail at repository unique/insert path")
	}
}

func TestRankedSummedBonusTotalsIntoPlayerAndTeamScores(t *testing.T) {
	repo, db := setupGamesRepositoryTest(t)
	ctx := context.Background()
	mapID, locationIDs := seedGameMap(t, db, 1)
	now := time.Date(2026, 7, 18, 18, 0, 0, 0, time.UTC)

	formed := formRankedTeamGame(t, repo, db, mapID, locationIDs, games.GameModeRankedDuo, 2, now)
	round := activeRound(t, db, formed.Game.ID)

	// Stagger submits so bonuses differ, then close via all-submit.
	var (
		teamOneScore int
		teamTwoScore int
		teamOneBonus int
		teamTwoBonus int
	)
	for i, p := range formed.Players {
		submitAt := now.Add(time.Duration(i+1) * 5 * time.Second)
		out, _, err := repo.SubmitMultiplayerGuessTx(ctx, formed.Game.ID, round.ID, p.ID,
			games.Guess{Latitude: float64(30 + i), Longitude: float64(31 + i)}, submitAt, games.MultiplayerTxHooks{})
		if err != nil {
			t.Fatalf("guess %d: %v", i, err)
		}
		if out.Guess.Score != out.Guess.AccuracyScore+out.Guess.SpeedBonus {
			t.Fatalf("invariant: %+v", out.Guess)
		}
		if p.TeamSlot == nil {
			t.Fatal("ranked team player must have team_slot")
		}
		switch *p.TeamSlot {
		case games.TeamSlotOne:
			teamOneBonus += out.Guess.SpeedBonus
			teamOneScore += out.Guess.Score
		case games.TeamSlotTwo:
			teamTwoBonus += out.Guess.SpeedBonus
			teamTwoScore += out.Guess.Score
		}
	}

	if teamOneBonus <= 0 || teamTwoBonus <= 0 {
		t.Fatalf("expected positive team speed bonuses, got %d / %d", teamOneBonus, teamTwoBonus)
	}

	players, err := repo.ListPlayers(ctx, formed.Game.ID)
	if err != nil {
		t.Fatalf("list players: %v", err)
	}
	totals := games.SumTeamTotals(players)
	if totals.TeamOneScore != teamOneScore || totals.TeamTwoScore != teamTwoScore {
		t.Fatalf("player total_score team sums = %d/%d, want %d/%d (accuracy+bonus)",
			totals.TeamOneScore, totals.TeamTwoScore, teamOneScore, teamTwoScore)
	}

	// Player.TotalScore must equal sum of stored guess scores (single round here).
	for _, p := range players {
		var sum int
		if err := db.Model(&games.Guess{}).Select("COALESCE(SUM(score),0)").
			Where("game_player_id = ?", p.ID).Scan(&sum).Error; err != nil {
			t.Fatalf("sum scores: %v", err)
		}
		if p.TotalScore != sum {
			t.Fatalf("player %s TotalScore=%d sum(guess.score)=%d", p.ID, p.TotalScore, sum)
		}
	}

	shared, err := repo.LoadSharedRoundResults(ctx, formed.Game.ID, round.ID)
	if err != nil || shared == nil {
		t.Fatalf("shared: %+v err=%v", shared, err)
	}
	if shared.TeamOneScore != teamOneScore || shared.TeamTwoScore != teamTwoScore {
		t.Fatalf("shared team scores = %d/%d, want %d/%d",
			shared.TeamOneScore, shared.TeamTwoScore, teamOneScore, teamTwoScore)
	}
}

func TestRankedDefaultTimerWhenFormationOmitsTimer(t *testing.T) {
	repo, db := setupGamesRepositoryTest(t)
	ctx := context.Background()
	mapID, locationIDs := seedGameMap(t, db, 1)
	now := time.Date(2026, 7, 18, 19, 0, 0, 0, time.UTC)
	users := seedRankedUsers(t, db, 2, now)

	formed, err := repo.CreateTeamGameBundle(ctx, games.TeamGameFormationInput{
		Mode:          games.GameModeRankedSolo,
		MapID:         mapID,
		RoundCount:    1,
		TimerSeconds:  nil, // omitted — games must default to 60
		StartedAt:     now,
		RoundStartsAt: now,
		TeamOne:       []games.TeamRosterMember{{UserID: users[0], DisplayName: "A"}},
		TeamTwo:       []games.TeamRosterMember{{UserID: users[1], DisplayName: "B"}},
		LocationIDs:   locationIDs,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if formed.Game.TimerSeconds == nil || *formed.Game.TimerSeconds != 60 {
		t.Fatalf("default timer = %v, want 60", formed.Game.TimerSeconds)
	}
	round := formed.Rounds[0]
	if round.EndsAt == nil || !round.EndsAt.Equal(now.Add(60*time.Second)) {
		t.Fatalf("default ends_at = %v", round.EndsAt)
	}
}

func TestCasualStillNullTimersAndZeroBonus(t *testing.T) {
	repo, db := setupGamesRepositoryTest(t)
	ctx := context.Background()
	mapID, locationIDs := seedGameMap(t, db, 1)
	now := time.Date(2026, 7, 18, 20, 0, 0, 0, time.UTC)
	users := seedRankedUsers(t, db, 2, now)

	formed, err := repo.CreateTeamGameBundle(ctx, games.TeamGameFormationInput{
		Mode:          games.GameModeCasualSolo,
		MapID:         mapID,
		RoundCount:    1,
		TimerSeconds:  games.IntPtr(60), // must be forced off for casual
		StartedAt:     now,
		RoundStartsAt: now,
		TeamOne:       []games.TeamRosterMember{{UserID: users[0], DisplayName: "A"}},
		TeamTwo:       []games.TeamRosterMember{{UserID: users[1], DisplayName: "B"}},
		LocationIDs:   locationIDs,
	})
	if err != nil {
		t.Fatalf("create casual: %v", err)
	}
	if formed.Game.TimerSeconds != nil {
		t.Fatalf("casual timer_seconds must be nil, got %v", *formed.Game.TimerSeconds)
	}
	round := formed.Rounds[0]
	if round.EndsAt != nil {
		t.Fatalf("casual ends_at must be nil, got %v", round.EndsAt)
	}

	out, _, err := repo.SubmitMultiplayerGuessTx(ctx, formed.Game.ID, round.ID, formed.Players[0].ID,
		games.Guess{Latitude: 30.04, Longitude: 31.23}, now.Add(time.Second), games.MultiplayerTxHooks{})
	if err != nil {
		t.Fatalf("casual guess: %v", err)
	}
	if out.Guess.SpeedBonus != 0 || out.Guess.Score != out.Guess.AccuracyScore {
		t.Fatalf("casual must keep speed_bonus=0, got %+v", out.Guess)
	}
}

func formRankedTeamGame(
	t *testing.T,
	repo *games.Repository,
	db *gorm.DB,
	mapID uuid.UUID,
	locationIDs []uuid.UUID,
	mode string,
	teamSize int,
	now time.Time,
) *games.TeamGameFormationResult {
	t.Helper()
	users := seedRankedUsers(t, db, teamSize*2, now)
	teamOne := make([]games.TeamRosterMember, teamSize)
	teamTwo := make([]games.TeamRosterMember, teamSize)
	for i := 0; i < teamSize; i++ {
		teamOne[i] = games.TeamRosterMember{UserID: users[i], DisplayName: "T1-" + users[i].String()[:8]}
		teamTwo[i] = games.TeamRosterMember{UserID: users[teamSize+i], DisplayName: "T2-" + users[teamSize+i].String()[:8]}
	}
	roundCount := len(locationIDs)
	if roundCount > 5 {
		roundCount = 5
	}
	formed, err := repo.CreateTeamGameBundle(context.Background(), games.TeamGameFormationInput{
		Mode:          mode,
		MapID:         mapID,
		RoundCount:    roundCount,
		TimerSeconds:  games.DefaultRankedTimerSeconds(),
		StartedAt:     now,
		RoundStartsAt: now,
		TeamOne:       teamOne,
		TeamTwo:       teamTwo,
		LocationIDs:   locationIDs[:roundCount],
	})
	if err != nil {
		t.Fatalf("form ranked game: %v", err)
	}
	return formed
}

func activeRound(t *testing.T, db *gorm.DB, gameID uuid.UUID) games.Round {
	t.Helper()
	var round games.Round
	if err := db.Where("game_id = ? AND status = ?", gameID, games.RoundStatusActive).First(&round).Error; err != nil {
		t.Fatalf("active round: %v", err)
	}
	return round
}

func seedRankedUsers(t *testing.T, db *gorm.DB, n int, now time.Time) []uuid.UUID {
	t.Helper()
	ids := make([]uuid.UUID, n)
	for i := 0; i < n; i++ {
		id := uuid.New()
		ids[i] = id
		if err := db.Exec(
			`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
			 VALUES (?, ?, 'x', 'user', 'active', ?, ?) ON CONFLICT DO NOTHING`,
			id, id.String()+"@ranked-test.example", now, now,
		).Error; err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}
	return ids
}
