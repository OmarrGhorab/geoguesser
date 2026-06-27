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
