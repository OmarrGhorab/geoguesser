package matchmaking

import (
	"context"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/maps"
)

// MapsLocationSelector adapts maps.Service location selection to matchmaking's UUID-only interface.
type MapsLocationSelector struct {
	selector interface {
		SelectLocations(ctx context.Context, mapID uuid.UUID, count int) ([]maps.SelectedLocation, error)
	}
}

// NewMapsLocationSelector wraps a maps location selector.
func NewMapsLocationSelector(selector interface {
	SelectLocations(ctx context.Context, mapID uuid.UUID, count int) ([]maps.SelectedLocation, error)
}) *MapsLocationSelector {
	return &MapsLocationSelector{selector: selector}
}

// SelectLocations returns distinct active location IDs for the map.
func (s *MapsLocationSelector) SelectLocations(ctx context.Context, mapID uuid.UUID, count int) ([]uuid.UUID, error) {
	if s == nil || s.selector == nil {
		return nil, ErrContentUnavailable
	}
	selected, err := s.selector.SelectLocations(ctx, mapID, count)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(selected))
	seen := make(map[uuid.UUID]struct{}, len(selected))
	for _, loc := range selected {
		if loc.ID == uuid.Nil {
			continue
		}
		if _, ok := seen[loc.ID]; ok {
			continue
		}
		seen[loc.ID] = struct{}{}
		ids = append(ids, loc.ID)
		if len(ids) >= count {
			break
		}
	}
	return ids, nil
}
