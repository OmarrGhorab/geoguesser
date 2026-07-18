package games

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/maps"
	"github.com/raven/geoguess/backend/internal/platform/clock"
	"github.com/raven/geoguess/backend/internal/session"
)

func TestCanStart(t *testing.T) {
	t.Parallel()

	if !CanStart(GameStatusPending) {
		t.Fatal("pending game should be startable")
	}
	if CanStart(GameStatusActive) {
		t.Fatal("active game should not be startable")
	}
}

func TestNewService(t *testing.T) {
	t.Parallel()

	svc := NewService(nil, fakeLocationSelector{}, clock.Fixed(time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)), slog.Default())
	if svc == nil {
		t.Fatal("service should be created")
	}
}

func TestFinalizeCompletedGameRetriesIdempotentProjection(t *testing.T) {
	t.Parallel()

	completedAt := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	hook := &recordingCompletionHook{err: errors.New("temporary projection failure")}
	svc := NewServiceWithHook(
		nil,
		fakeLocationSelector{},
		nil,
		clock.Fixed(completedAt.Add(time.Hour)),
		slog.Default(),
		nil,
		nil,
		hook,
	)
	game := &Game{ID: uuid.New(), Status: GameStatusCompleted, CompletedAt: &completedAt}

	if err := svc.finalizeCompletedGame(context.Background(), game); err == nil {
		t.Fatal("finalizeCompletedGame() error = nil, want transient hook error")
	}
	if hook.calls != 1 || !hook.completedAt.Equal(completedAt) {
		t.Fatalf("hook calls = %d at %s", hook.calls, hook.completedAt)
	}

	hook.err = nil
	if err := svc.finalizeCompletedGame(context.Background(), game); err != nil {
		t.Fatalf("finalizeCompletedGame() retry error = %v", err)
	}
	if hook.calls != 2 {
		t.Fatalf("hook calls after retry = %d, want 2", hook.calls)
	}
}

func TestFinalizeCompletedGameSkipsProgressionNeutralModes(t *testing.T) {
	t.Parallel()

	hook := &recordingCompletionHook{}
	svc := NewServiceWithHook(nil, fakeLocationSelector{}, nil, clock.Fixed(time.Now()), slog.Default(), nil, nil, hook)
	for _, mode := range []string{GameModePractice, GameModePartyLobby, GameModePrivateRoom} {
		if err := svc.finalizeCompletedGame(context.Background(), &Game{ID: uuid.New(), Mode: mode, Status: GameStatusCompleted}); err != nil {
			t.Fatalf("mode %s: %v", mode, err)
		}
	}
	if hook.calls != 0 {
		t.Fatalf("progression hook calls = %d, want 0", hook.calls)
	}
}

func TestPracticeCursorBindsGameAndRound(t *testing.T) {
	t.Parallel()

	gameID := uuid.New()
	key := derivePracticeCursorKey("test-practice-cursor-secret")
	cursor := encodePracticeCursor(key, gameID, 42)
	round, err := decodePracticeCursor(key, cursor, gameID)
	if err != nil || round != 42 {
		t.Fatalf("decode = %d, %v", round, err)
	}
	if _, err := decodePracticeCursor(key, cursor, uuid.New()); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("cross-game cursor err = %v", err)
	}
	tamperedPrefix := "A"
	if cursor[0] == 'A' {
		tamperedPrefix = "B"
	}
	tampered := tamperedPrefix + cursor[1:]
	if _, err := decodePracticeCursor(key, tampered, gameID); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("tampered cursor err = %v", err)
	}
}

type recordingCompletionHook struct {
	calls       int
	completedAt time.Time
	err         error
}

func (h *recordingCompletionHook) OnGameCompleted(_ context.Context, _ uuid.UUID, completedAt time.Time) error {
	h.calls++
	h.completedAt = completedAt
	return h.err
}

func TestOwnerFromSession(t *testing.T) {
	t.Parallel()

	userID := uuid.NewString()
	owner, err := ownerFromSession(&session.Context{Kind: session.KindUser, UserID: &userID})
	if err != nil {
		t.Fatalf("registered owner failed: %v", err)
	}
	if owner.userID == nil || owner.userID.String() != userID {
		t.Fatalf("registered owner id = %v, want %s", owner.userID, userID)
	}

	guestID := "guest-hash"
	owner, err = ownerFromSession(&session.Context{Kind: session.KindGuest, GuestID: &guestID})
	if err != nil {
		t.Fatalf("guest owner failed: %v", err)
	}
	if owner.guestHash == nil || *owner.guestHash != guestID {
		t.Fatalf("guest owner hash = %v, want %s", owner.guestHash, guestID)
	}

	if _, err := ownerFromSession(&session.Context{Kind: session.KindAnonymous}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("anonymous error = %v, want ErrForbidden", err)
	}
}

func TestOwnerMatches(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	if !ownerMatches(ownerIdentity{userID: &userID}, GamePlayer{UserID: &userID}) {
		t.Fatal("registered owner should match")
	}
	guest := "guest-hash"
	if !ownerMatches(ownerIdentity{guestHash: &guest}, GamePlayer{GuestIdentityHash: &guest}) {
		t.Fatal("guest owner should match")
	}
	other := uuid.New()
	if ownerMatches(ownerIdentity{userID: &userID}, GamePlayer{UserID: &other}) {
		t.Fatal("different registered owner should not match")
	}
}

func TestCreateGameValidationBeforeRepository(t *testing.T) {
	t.Parallel()

	svc := NewService(nil, fakeLocationSelector{}, clock.Fixed(time.Now().UTC()), slog.Default())
	guestID := "guest-hash"
	sess := &session.Context{Kind: session.KindGuest, GuestID: &guestID}

	cases := []CreateGameRequest{
		{Mode: "private_room", MapID: uuid.New(), RoundCount: 5},
		{Mode: GameModeSolo, MapID: uuid.Nil, RoundCount: 5},
		{Mode: GameModeSolo, MapID: uuid.New(), RoundCount: 11},
		{Mode: GameModeSolo, MapID: uuid.New(), RoundCount: 5, TimerSeconds: intPtr(9)},
	}
	for _, req := range cases {
		if _, err := svc.CreateGame(context.Background(), sess, req); !errors.Is(err, ErrInvalidGameRequest) {
			t.Fatalf("CreateGame(%+v) error = %v, want ErrInvalidGameRequest", req, err)
		}
	}
}

func TestCreateGameRejectsNotEnoughLocations(t *testing.T) {
	t.Parallel()

	svc := NewService(nil, fakeLocationSelector{locations: []maps.SelectedLocation{{ID: uuid.New()}}}, clock.Fixed(time.Now().UTC()), slog.Default())
	guestID := "guest-hash"
	_, err := svc.CreateGame(context.Background(), &session.Context{Kind: session.KindGuest, GuestID: &guestID}, CreateGameRequest{
		Mode:       GameModeSolo,
		MapID:      uuid.New(),
		RoundCount: 2,
	})
	if !errors.Is(err, ErrNotEnoughLocations) {
		t.Fatalf("CreateGame error = %v, want ErrNotEnoughLocations", err)
	}
}

func TestCreateGameRejectsDuplicateSelectedLocations(t *testing.T) {
	t.Parallel()

	locationID := uuid.New()
	svc := NewService(nil, fakeLocationSelector{locations: []maps.SelectedLocation{{ID: locationID}, {ID: locationID}}}, clock.Fixed(time.Now().UTC()), slog.Default())
	guestID := "guest-hash"
	_, err := svc.CreateGame(context.Background(), &session.Context{Kind: session.KindGuest, GuestID: &guestID}, CreateGameRequest{
		Mode:       GameModeSolo,
		MapID:      uuid.New(),
		RoundCount: 2,
	})
	if !errors.Is(err, ErrNotEnoughLocations) {
		t.Fatalf("CreateGame error = %v, want ErrNotEnoughLocations", err)
	}
}

func TestRoundDTOHidesAnswerFields(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	svc := NewServiceWithMedia(nil, nil, fakeMediaProvider{}, clock.Fixed(now), slog.Default())
	dto := svc.toRoundDTO(currentRoundRow{
		RoundID:     uuid.New(),
		RoundNumber: 1,
		RoundStatus: RoundStatusActive,
		StartsAt:    &now,
		LocationID:  uuid.New(),
		Provider:    "image",
		ProviderRef: "https://example.test/location.jpg",
	})
	if dto.Media == nil || dto.Media.URL != "https://example.test/location.jpg" {
		t.Fatalf("media = %+v", dto.Media)
	}
}

func TestRoundDTOProjectsPlayablePanoramaWithoutCoordinates(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	svc := NewServiceWithMedia(nil, nil, fakeMediaProvider{}, clock.Fixed(now), slog.Default())
	dto := svc.toRoundDTO(currentRoundRow{
		RoundID:     uuid.New(),
		RoundNumber: 1,
		RoundStatus: RoundStatusActive,
		StartsAt:    &now,
		LocationID:  uuid.New(),
		Provider:    "google_street_view",
		ProviderRef: "CAoSLEFGMVFpcE5fexample_123-abc",
	})
	if dto.Media == nil || dto.Media.PanoramaID != "CAoSLEFGMVFpcE5fexample_123-abc" || dto.Media.URL != "" {
		t.Fatalf("media = %+v", dto.Media)
	}
}

type fakeLocationSelector struct {
	locations []maps.SelectedLocation
	err       error
}

func (f fakeLocationSelector) SelectLocations(context.Context, uuid.UUID, int) ([]maps.SelectedLocation, error) {
	return f.locations, f.err
}

type fakeMediaProvider struct{}

func (fakeMediaProvider) MediaURL(_, ref string) (string, error) {
	return ref, nil
}

func intPtr(v int) *int {
	return &v
}
