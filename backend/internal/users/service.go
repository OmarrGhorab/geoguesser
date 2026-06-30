package users

import (
	"context"
	"time"

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

// GetUserGameHistory returns a cursor-paginated list of a user's games.
func (s *Service) GetUserGameHistory(ctx context.Context, userID string, limit int, cursor string) (*UserGameHistoryResponse, error) {
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

	games, err := s.repo.ListUserGameHistory(ctx, id, limit, cursor)
	if err != nil {
		return nil, err
	}

	var nextCursor *string
	if len(games) == limit {
		last := games[len(games)-1]
		c := last.CreatedAt.Format(time.RFC3339Nano) + "|" + last.ID.String()
		nextCursor = &c
	}

	return &UserGameHistoryResponse{
		Games: games,
		Page:  PageInfo{Limit: limit, NextCursor: nextCursor},
	}, nil
}
