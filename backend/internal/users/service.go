package users

import (
	"context"

	"github.com/google/uuid"
)

// Service implements users business logic.
type Service struct {
	repo *Repository
}

// NewService returns a new users service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetUserStats returns public stats for a user.
func (s *Service) GetUserStats(ctx context.Context, userID string) (*UserStatsResponse, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	stats, err := s.repo.GetUserStats(ctx, id)
	if err != nil {
		return nil, err
	}

	return &UserStatsResponse{Stats: *stats}, nil
}
