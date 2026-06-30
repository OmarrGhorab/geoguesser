package maps

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository owns map and map-location queries.
type Repository struct {
	db *gorm.DB
}

// NewRepository returns a new maps repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ListFilters carries allowlisted filters for public map listing.
type ListFilters struct {
	AccessTier string
	Difficulty string
}

// ListCursor is an opaque pagination cursor.
type ListCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        uuid.UUID `json:"id"`
}

// Encode serializes the cursor to a base64-encoded JSON string.
func (c ListCursor) Encode() string {
	b, _ := json.Marshal(c)
	return base64.StdEncoding.EncodeToString(b)
}

// DecodeListCursor parses a cursor string.
func DecodeListCursor(s string) (ListCursor, error) {
	var c ListCursor
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	return c, nil
}

// ListMaps returns public active maps matching the filters.
// It requests limit+1 rows to determine whether a next page exists.
func (r *Repository) ListMaps(ctx context.Context, filters ListFilters, cursor *ListCursor, limit int) ([]Map, error) {
	if limit <= 0 {
		limit = 20
	}
	query := r.db.WithContext(ctx).
		Where("status = ? AND visibility = ?", "active", "public")

	if filters.AccessTier != "" {
		query = query.Where("access_tier = ?", filters.AccessTier)
	}
	if filters.Difficulty != "" {
		query = query.Where("difficulty = ?", filters.Difficulty)
	}
	if cursor != nil {
		query = query.Where(
			"(created_at, id) < (?, ?)",
			cursor.CreatedAt, cursor.ID,
		)
	}

	var maps []Map
	if err := query.
		Order("created_at DESC, id DESC").
		Limit(limit + 1).
		Find(&maps).Error; err != nil {
		return nil, fmt.Errorf("failed to list maps: %w", err)
	}
	return maps, nil
}

// GetMapByID returns a public active map by id.
func (r *Repository) GetMapByID(ctx context.Context, id uuid.UUID) (*Map, error) {
	var m Map
	if err := r.db.WithContext(ctx).
		Where("id = ? AND status = ? AND visibility = ?", id, "active", "public").
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get map: %w", err)
	}
	return &m, nil
}

// GetMapBySlug returns a public active map by slug.
func (r *Repository) GetMapBySlug(ctx context.Context, slug string) (*Map, error) {
	var m Map
	if err := r.db.WithContext(ctx).
		Where("slug = ? AND status = ? AND visibility = ?", slug, "active", "public").
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get map by slug: %w", err)
	}
	return &m, nil
}

// SelectLocations returns up to count active locations for the map using an
// indexed random_key pivot instead of ORDER BY random() at scale.
func (r *Repository) SelectLocations(ctx context.Context, mapID uuid.UUID, count int) ([]SelectedLocation, error) {
	if count <= 0 {
		return []SelectedLocation{}, nil
	}
	if count > 100 {
		count = 100
	}

	pivot, err := randomPivot()
	if err != nil {
		return nil, err
	}

	var rows []SelectedLocation
	if err := r.selectLocationsByRandomKey(ctx, mapID, pivot, true, count, &rows); err != nil {
		return nil, err
	}
	if len(rows) < count {
		if err := r.selectLocationsByRandomKey(ctx, mapID, pivot, false, count-len(rows), &rows); err != nil {
			return nil, err
		}
	}
	return rows, nil
}

// SelectLocationsBySeed returns a deterministic active-location ordering for a
// fixed challenge seed. Challenge materialization persists the selected rows, so
// the deterministic order is needed only when the immutable snapshot is first
// created.
func (r *Repository) SelectLocationsBySeed(ctx context.Context, mapID uuid.UUID, count int, seed string) ([]SelectedLocation, error) {
	if count <= 0 {
		return []SelectedLocation{}, nil
	}
	if count > 100 {
		count = 100
	}

	var rows []SelectedLocation
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			l.id,
			l.latitude,
			l.longitude,
			l.country_code,
			l.region,
			l.locality,
			l.difficulty,
			l.provider,
			l.attribution
		FROM locations l
		JOIN map_locations ml ON ml.location_id = l.id
		JOIN maps m ON m.id = ml.map_id
		WHERE ml.map_id = ?
		  AND l.status = 'active'
		  AND m.status = 'active'
		ORDER BY md5(? || ':' || l.id::text), l.id
		LIMIT ?
	`, mapID, seed, count).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to select seeded locations: %w", err)
	}
	return rows, nil
}

func (r *Repository) selectLocationsByRandomKey(ctx context.Context, mapID uuid.UUID, pivot float64, afterPivot bool, limit int, rows *[]SelectedLocation) error {
	query := `
		SELECT
			l.id,
			l.latitude,
			l.longitude,
			l.country_code,
			l.region,
			l.locality,
			l.difficulty,
			l.provider,
			l.attribution
		FROM locations l
		JOIN map_locations ml ON ml.location_id = l.id
		JOIN maps m ON m.id = ml.map_id
		WHERE ml.map_id = ?
		  AND l.status = 'active'
		  AND m.status = 'active'
		  AND l.random_key >= ?
		ORDER BY l.random_key
		LIMIT ?
	`
	if !afterPivot {
		query = `
			SELECT
				l.id,
				l.latitude,
				l.longitude,
				l.country_code,
				l.region,
				l.locality,
				l.difficulty,
				l.provider,
				l.attribution
			FROM locations l
			JOIN map_locations ml ON ml.location_id = l.id
			JOIN maps m ON m.id = ml.map_id
			WHERE ml.map_id = ?
			  AND l.status = 'active'
			  AND m.status = 'active'
			  AND l.random_key < ?
			ORDER BY l.random_key
			LIMIT ?
		`
	}

	var page []SelectedLocation
	if err := r.db.WithContext(ctx).Raw(query, mapID, pivot, limit).Scan(&page).Error; err != nil {
		return fmt.Errorf("failed to select locations: %w", err)
	}
	*rows = append(*rows, page...)
	return nil
}

func randomPivot() (float64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000_000))
	if err != nil {
		return 0, fmt.Errorf("failed to generate random location pivot: %w", err)
	}
	return float64(n.Int64()) / 1_000_000_000, nil
}
