package games_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/games"
	"github.com/raven/geoguess/backend/internal/locations"
	"github.com/raven/geoguess/backend/internal/maps"
	"github.com/raven/geoguess/backend/internal/platform/clock"
	"github.com/raven/geoguess/backend/internal/session"
)

func TestPracticeRepositorySupportsMoreThanTenRoundsAndConcurrentReplay(t *testing.T) {
	repo, db := setupGamesRepositoryTest(t)
	ctx := context.Background()
	mapID, locationIDs := seedGameMap(t, db, 12)
	guest := "practice-open-ended-" + uuid.NewString()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	game := &games.Game{Mode: games.GameModePractice, Status: games.GameStatusPending, MapID: mapID, RoundCount: 1, ScoringVersion: games.ScoringVersionV1}
	player := &games.GamePlayer{GuestIdentityHash: &guest, DisplayName: "Practice", Role: games.PlayerRolePlayer, Status: games.PlayerStatusActive}
	rounds := []games.Round{{LocationID: locationIDs[0], RoundNumber: 1, Status: games.RoundStatusPending}}
	if err := repo.CreateGameBundle(ctx, game, player, rounds); err != nil {
		t.Fatalf("create Practice bundle: %v", err)
	}
	if _, err := repo.StartGame(ctx, game.ID, now, nil); err != nil {
		t.Fatalf("start Practice: %v", err)
	}

	for roundNumber := 1; roundNumber <= 11; roundNumber++ {
		current, err := repo.GetCurrentRound(ctx, game.ID)
		if err != nil || current == nil || current.RoundNumber != roundNumber {
			t.Fatalf("current round %d: current=%+v err=%v", roundNumber, current, err)
		}
		guessKey := fmt.Sprintf("practice-guess-%02d-%s", roundNumber, uuid.NewString())
		if _, _, completed, err := repo.SubmitGuessTx(ctx, game.ID, current.RoundID, player.ID, games.Guess{
			Latitude: 30, Longitude: 31, IdempotencyKey: &guessKey,
		}, now.Add(time.Duration(roundNumber)*time.Second)); err != nil || completed {
			t.Fatalf("submit round %d: completed=%v err=%v", roundNumber, completed, err)
		}
		if roundNumber == 11 {
			break
		}
		nextKey := fmt.Sprintf("practice-next-%02d-%s", roundNumber+1, uuid.NewString())
		next, err := repo.AppendPracticeRound(ctx, game.ID, locationIDs[roundNumber], nextKey, now.Add(time.Duration(roundNumber)*time.Minute))
		if err != nil || next == nil || next.RoundNumber != roundNumber+1 {
			t.Fatalf("append round %d: next=%+v err=%v", roundNumber+1, next, err)
		}
	}

	key := "practice-concurrent-" + uuid.NewString()
	type appendResult struct {
		ID     uuid.UUID
		Number int
	}
	results := make(chan *appendResult, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			round, err := repo.AppendPracticeRound(ctx, game.ID, locationIDs[11], key, now.Add(12*time.Minute))
			if round == nil {
				results <- nil
			} else {
				results <- &appendResult{ID: round.RoundID, Number: round.RoundNumber}
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent append: %v", err)
		}
	}
	var first *appendResult
	for result := range results {
		if result == nil {
			t.Fatal("concurrent append returned nil round")
		}
		if first == nil {
			first = result
			continue
		}
		if result.ID != first.ID || result.Number != first.Number || result.Number != 12 {
			t.Fatalf("concurrent replay mismatch: first=%+v result=%+v", first, result)
		}
	}

	var stored games.Game
	if err := db.First(&stored, "id = ?", game.ID).Error; err != nil {
		t.Fatalf("reload Practice game: %v", err)
	}
	if stored.RoundCount != 12 || stored.Status != games.GameStatusActive {
		t.Fatalf("Practice state after round 12: %+v", stored)
	}
	if _, err := repo.EndPractice(ctx, game.ID, now.Add(13*time.Minute)); err != nil {
		t.Fatalf("end Practice: %v", err)
	}
	if _, err := repo.AppendPracticeRound(ctx, game.ID, locationIDs[0], "practice-after-end-"+uuid.NewString(), now.Add(14*time.Minute)); !errors.Is(err, games.ErrGameNotActive) {
		t.Fatalf("append after end error=%v, want ErrGameNotActive", err)
	}
}

func TestPracticeGuessReplayDoesNotUseRedis(t *testing.T) {
	repo, db := setupGamesRepositoryTest(t)
	ctx := context.Background()
	mapID, locationIDs := seedGameMap(t, db, 1)
	guest := "practice-no-redis-" + uuid.NewString()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	game := &games.Game{Mode: games.GameModePractice, Status: games.GameStatusPending, MapID: mapID, RoundCount: 1, ScoringVersion: games.ScoringVersionV1}
	player := &games.GamePlayer{GuestIdentityHash: &guest, DisplayName: "Practice", Role: games.PlayerRolePlayer, Status: games.PlayerStatusActive}
	if err := repo.CreateGameBundle(ctx, game, player, []games.Round{{LocationID: locationIDs[0], RoundNumber: 1, Status: games.RoundStatusPending}}); err != nil {
		t.Fatalf("create Practice bundle: %v", err)
	}
	if _, err := repo.StartGame(ctx, game.ID, now, nil); err != nil {
		t.Fatalf("start Practice: %v", err)
	}
	current, err := repo.GetCurrentRound(ctx, game.ID)
	if err != nil || current == nil {
		t.Fatalf("current Practice round=%+v err=%v", current, err)
	}

	idempotency := &failingPracticeIdempotency{}
	svc := games.NewServiceWithOptions(repo, fixedLocationSelector{}, locations.StaticProvider{}, clock.Fixed(now.Add(time.Second)), slog.Default(), idempotency, nil)
	sess := &session.Context{Kind: session.KindGuest, GuestID: &guest}
	key := "practice-guess-" + uuid.NewString()
	request := games.SubmitGuessRequest{Latitude: 30, Longitude: 31}
	first, err := svc.SubmitGuess(ctx, sess, game.ID.String(), current.RoundID.String(), key, request)
	if err != nil {
		t.Fatalf("first Practice guess: %v", err)
	}
	replay, err := svc.SubmitGuess(ctx, sess, game.ID.String(), current.RoundID.String(), key, request)
	if err != nil {
		t.Fatalf("Practice guess replay: %v", err)
	}
	if replay.Guess.ID != first.Guess.ID || idempotency.claims != 0 {
		t.Fatalf("replay=%+v first=%+v Redis claims=%d", replay.Guess, first.Guess, idempotency.claims)
	}
}

type fixedLocationSelector struct{}

func (fixedLocationSelector) SelectLocations(context.Context, uuid.UUID, int) ([]maps.SelectedLocation, error) {
	return nil, nil
}

type failingPracticeIdempotency struct {
	claims int
}

func (s *failingPracticeIdempotency) Claim(context.Context, string, time.Duration) (bool, error) {
	s.claims++
	return false, errors.New("Redis unavailable")
}

func (*failingPracticeIdempotency) Release(context.Context, string) error {
	return errors.New("Redis unavailable")
}
