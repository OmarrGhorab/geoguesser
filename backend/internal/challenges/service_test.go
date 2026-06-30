package challenges

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/maps"
)

func TestNormalizeSettingsValidation(t *testing.T) {
	if _, err := normalizeSettings(11, nil); !errors.Is(err, ErrInvalidChallengeInput) {
		t.Fatalf("round count error = %v, want invalid input", err)
	}
	tooSmall := 5
	if _, err := normalizeSettings(5, &tooSmall); !errors.Is(err, ErrInvalidChallengeInput) {
		t.Fatalf("timer error = %v, want invalid input", err)
	}
}

func TestSelectUniqueRejectsInsufficientUniqueLocations(t *testing.T) {
	mapID := uuid.New()
	locationID := uuid.New()
	svc := &Service{selector: selectorStub{locations: []maps.SelectedLocation{{ID: locationID}, {ID: locationID}}}}

	if _, err := svc.selectUnique(context.Background(), mapID, 2); !errors.Is(err, ErrNotEnoughLocations) {
		t.Fatalf("selectUnique error = %v, want ErrNotEnoughLocations", err)
	}
}

type selectorStub struct {
	locations []maps.SelectedLocation
	err       error
}

func (s selectorStub) SelectLocations(context.Context, uuid.UUID, int) ([]maps.SelectedLocation, error) {
	return s.locations, s.err
}
