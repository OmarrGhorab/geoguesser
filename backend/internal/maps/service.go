package maps

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// Service implements map business logic.
type Service struct {
	repo *Repository
}

// NewService returns a new maps service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ListMaps returns a public map list.
func (s *Service) ListMaps(ctx context.Context, filters ListFilters, cursor string, limit int) (*MapListResponse, error) {
	filters.AccessTier = strings.TrimSpace(filters.AccessTier)
	filters.Difficulty = strings.TrimSpace(filters.Difficulty)
	if !validAccessTier(filters.AccessTier) || !validDifficulty(filters.Difficulty) {
		return nil, ErrInvalidFilter
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var parsed *ListCursor
	if cursor != "" {
		c, err := DecodeListCursor(cursor)
		if err != nil {
			return nil, ErrInvalidCursor
		}
		parsed = &c
	}

	maps, err := s.repo.ListMaps(ctx, filters, parsed, limit)
	if err != nil {
		return nil, err
	}

	hasNext := len(maps) > limit
	if hasNext {
		maps = maps[:limit]
	}

	var nextCursor *string
	if hasNext && len(maps) > 0 {
		last := maps[len(maps)-1]
		c := ListCursor{CreatedAt: last.CreatedAt, ID: last.ID}.Encode()
		nextCursor = &c
	}

	resp := &MapListResponse{
		Data: make([]MapDTO, len(maps)),
		Page: PageInfo{Limit: limit, NextCursor: nextCursor},
	}
	for i, m := range maps {
		resp.Data[i] = toMapDTO(m)
	}
	return resp, nil
}

// GetMap returns public map metadata by id or slug.
func (s *Service) GetMap(ctx context.Context, mapID string) (*MapResponse, error) {
	var m *Map
	var err error

	if id, parseErr := uuid.Parse(mapID); parseErr == nil {
		m, err = s.repo.GetMapByID(ctx, id)
	} else {
		m, err = s.repo.GetMapBySlug(ctx, mapID)
	}
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrMapNotFound
	}

	return &MapResponse{Map: toMapDTO(*m)}, nil
}

// SelectLocations returns active gameplay locations for a map.
func (s *Service) SelectLocations(ctx context.Context, mapID uuid.UUID, count int) ([]SelectedLocation, error) {
	return s.repo.SelectLocations(ctx, mapID, count)
}

// SelectLocationsBySeed returns deterministic active gameplay locations for a fixed seed.
func (s *Service) SelectLocationsBySeed(ctx context.Context, mapID uuid.UUID, count int, seed string) ([]SelectedLocation, error) {
	return s.repo.SelectLocationsBySeed(ctx, mapID, count, seed)
}

func toMapDTO(m Map) MapDTO {
	return MapDTO{
		ID:          m.ID,
		Slug:        m.Slug,
		Name:        m.Name,
		Description: m.Description,
		Visibility:  m.Visibility,
		AccessTier:  m.AccessTier,
		Difficulty:  m.Difficulty,
		Status:      m.Status,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func validAccessTier(value string) bool {
	switch value {
	case "", "free", "premium", "admin":
		return true
	default:
		return false
	}
}

func validDifficulty(value string) bool {
	switch value {
	case "", "mixed", "easy", "medium", "hard":
		return true
	default:
		return false
	}
}
