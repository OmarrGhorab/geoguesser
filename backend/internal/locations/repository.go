package locations

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository owns location queries.
type Repository struct {
	db *gorm.DB
}

// NewRepository returns a new locations repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// GetLocationByID returns a location by id.
func (r *Repository) GetLocationByID(ctx context.Context, id uuid.UUID) (*Location, error) {
	var l Location
	if err := r.db.WithContext(ctx).First(&l, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get location: %w", err)
	}
	return &l, nil
}

// ListMapAccessForLocation returns map authorization metadata for every map
// attached to a location so access decisions are not dependent on join order.
func (r *Repository) ListMapAccessForLocation(ctx context.Context, locationID uuid.UUID) ([]mapAccess, error) {
	var rows []mapAccess
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			m.id AS map_id,
			m.visibility,
			m.access_tier,
			m.status,
			l.status AS location_status
		FROM locations l
		JOIN map_locations ml ON ml.location_id = l.id
		JOIN maps m ON m.id = ml.map_id
		WHERE l.id = ?
	`, locationID).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to list map access for location: %w", err)
	}
	return rows, nil
}
