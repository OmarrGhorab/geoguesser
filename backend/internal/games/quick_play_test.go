package games_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/games"
	"github.com/raven/geoguess/backend/internal/platform/clock"
	"github.com/raven/geoguess/backend/internal/session"
)

func TestGameModeQuickPlayConstant(t *testing.T) {
	t.Parallel()
	if games.GameModeQuickPlay != "quick_play" {
		t.Fatalf("GameModeQuickPlay = %q, want quick_play", games.GameModeQuickPlay)
	}
}

func TestQuickPlayIsNotMultiplayer(t *testing.T) {
	t.Parallel()
	if games.IsMultiplayerMode(games.GameModeQuickPlay) {
		t.Fatal("quick_play must not be multiplayer")
	}
	if games.IsOpenEndedMode(games.GameModeQuickPlay) {
		t.Fatal("quick_play is fixed-round, not open-ended")
	}
	if games.IsProgressionNeutralMode(games.GameModeQuickPlay) {
		t.Fatal("quick_play should count toward progression like solo")
	}
	if games.IsRankedMode(games.GameModeQuickPlay) || games.IsCasualMode(games.GameModeQuickPlay) {
		t.Fatal("quick_play is neither ranked nor casual matchmade")
	}
}

func TestQuickPlayRevealPolicyMatchesSolo(t *testing.T) {
	t.Parallel()
	var policy games.DelayedRevealPolicy
	if !policy.MayRevealAnswer(games.GameModeQuickPlay, games.RoundStatusActive, false) {
		t.Fatal("quick_play must allow immediate answer reveal like solo")
	}
	if !policy.MayRevealAnswer(games.GameModeSolo, games.RoundStatusActive, false) {
		t.Fatal("solo baseline for immediate reveal")
	}
}

func TestQuickPlayScoringHasNoSpeedBonus(t *testing.T) {
	t.Parallel()
	acc, bonus, total := games.ComposeGuessScores(games.GameModeQuickPlay, 4000, 60_000, 60_000)
	if acc != 4000 || bonus != 0 || total != 4000 {
		t.Fatalf("quick_play scores = %d/%d/%d, want 4000/0/4000", acc, bonus, total)
	}
	soloAcc, soloBonus, soloTotal := games.ComposeGuessScores(games.GameModeSolo, 4000, 60_000, 60_000)
	if acc != soloAcc || bonus != soloBonus || total != soloTotal {
		t.Fatalf("quick_play scoring must match solo")
	}
}

func TestWithQuickPlayDefaults(t *testing.T) {
	t.Parallel()
	mapID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	svc := games.NewService(nil, nil, clock.NewSystem(), slog.Default())
	// Non-positive rounds/timer fall back to product defaults 5 / 60.
	if got := svc.WithQuickPlayDefaults(mapID, 0, 0); got != svc {
		t.Fatal("WithQuickPlayDefaults should return the same service")
	}
	// Explicit positive overrides are accepted.
	svc.WithQuickPlayDefaults(mapID, 3, 90)
	// Nil service is a no-op.
	var nilSvc *games.Service
	if got := nilSvc.WithQuickPlayDefaults(mapID, 5, 60); got != nil {
		t.Fatal("nil receiver should return nil")
	}
}

func TestStartQuickPlayValidation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mapID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	guestID := "guest-identity-hash"
	guestSess := &session.Context{Kind: session.KindGuest, GuestID: &guestID}
	key := "idempotency-key-16chars"

	// No session.
	svc := games.NewService(nil, nil, clock.NewSystem(), slog.Default()).
		WithQuickPlayDefaults(mapID, 5, 60)
	if _, err := svc.StartQuickPlay(ctx, nil, key); !errors.Is(err, games.ErrForbidden) {
		t.Fatalf("nil session error = %v, want ErrForbidden", err)
	}

	// Short idempotency key.
	if _, err := svc.StartQuickPlay(ctx, guestSess, "short"); !errors.Is(err, games.ErrInvalidGameRequest) {
		t.Fatalf("short key error = %v, want ErrInvalidGameRequest", err)
	}

	// Empty key.
	if _, err := svc.StartQuickPlay(ctx, guestSess, ""); !errors.Is(err, games.ErrInvalidGameRequest) {
		t.Fatalf("empty key error = %v, want ErrInvalidGameRequest", err)
	}

	// Unconfigured map → operator misconfiguration (503-style, not client error).
	unconfigured := games.NewService(nil, nil, clock.NewSystem(), slog.Default())
	if _, err := unconfigured.StartQuickPlay(ctx, guestSess, key); !errors.Is(err, games.ErrQuickPlayUnavailable) {
		t.Fatalf("missing map error = %v, want ErrQuickPlayUnavailable", err)
	}
}

func TestIsMultiplayerModeExcludesQuickPlay(t *testing.T) {
	t.Parallel()
	// Regression: ensure the non-multiplayer set used by state_test still holds for quick_play.
	for _, mode := range []string{games.GameModeSolo, games.GameModePractice, games.GameModeDaily, games.GameModeQuickPlay, ""} {
		if games.IsMultiplayerMode(mode) {
			t.Fatalf("%q should not be multiplayer", mode)
		}
	}
}
