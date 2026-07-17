package home

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/raven/geoguess/backend/internal/challenges"
	"github.com/raven/geoguess/backend/internal/maps"
	"github.com/raven/geoguess/backend/internal/profiles"
	"github.com/raven/geoguess/backend/internal/session"
)

const recommendedMapLimit = 4

type profileReader interface {
	GetPublicProfile(ctx context.Context, userID uuid.UUID) (*profiles.PublicProfileSummary, error)
	GetStats(ctx context.Context, userID uuid.UUID) (*profiles.StatsSummary, error)
}

type dailyChallengeReader interface {
	GetDaily(ctx context.Context, sess *session.Context, dateOverride string) (*challenges.ChallengeMetadataResponse, error)
}

type mapReader interface {
	ListMaps(ctx context.Context, filters maps.ListFilters, cursor string, limit int) (*maps.MapListResponse, error)
}

// Service composes the authenticated-home read model from existing domains.
type Service struct {
	profiles   profileReader
	challenges dailyChallengeReader
	maps       mapReader
	logger     *slog.Logger
	metrics    *Metrics
}

// NewService returns an authenticated-home composition service.
func NewService(profilesReader profileReader, challengesReader dailyChallengeReader, mapsReader mapReader, logger *slog.Logger, metrics *Metrics) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		profiles:   profilesReader,
		challenges: challengesReader,
		maps:       mapsReader,
		logger:     logger,
		metrics:    metrics,
	}
}

// Get returns a coherent snapshot for an active registered viewer.
func (s *Service) Get(ctx context.Context, sess *session.Context) (*Response, error) {
	startedAt := time.Now()
	userID, err := registeredUserID(sess)
	if err != nil {
		s.observe("unauthorized", startedAt)
		return nil, err
	}

	profile, err := s.profiles.GetPublicProfile(ctx, userID)
	if err != nil {
		s.dependencyFailed(ctx, "profiles", err)
		s.observe("unavailable", startedAt)
		return nil, dependencyError("profiles", err)
	}
	if profile == nil {
		s.logger.InfoContext(ctx, "authenticated home read denied", slog.String("reason", "active_viewer_required"))
		s.observe("unauthorized", startedAt)
		return nil, ErrUnauthorized
	}

	var stats *profiles.StatsSummary
	var daily *challenges.ChallengeMetadataResponse
	var mapPage *maps.MapListResponse

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		var readErr error
		stats, readErr = s.profiles.GetStats(groupCtx, userID)
		if readErr != nil {
			s.dependencyFailed(groupCtx, "stats", readErr)
			return dependencyError("stats", readErr)
		}
		return nil
	})
	group.Go(func() error {
		var readErr error
		daily, readErr = s.challenges.GetDaily(groupCtx, sess, "")
		if readErr != nil {
			s.dependencyFailed(groupCtx, "daily_challenge", readErr)
			return dependencyError("daily_challenge", readErr)
		}
		if daily == nil {
			readErr = nilResponseError("daily_challenge")
			s.dependencyFailed(groupCtx, "daily_challenge", readErr)
			return dependencyError("daily_challenge", readErr)
		}
		return nil
	})
	group.Go(func() error {
		var readErr error
		mapPage, readErr = s.maps.ListMaps(groupCtx, maps.ListFilters{}, "", recommendedMapLimit)
		if readErr != nil {
			s.dependencyFailed(groupCtx, "maps", readErr)
			return dependencyError("maps", readErr)
		}
		if mapPage == nil {
			readErr = nilResponseError("maps")
			s.dependencyFailed(groupCtx, "maps", readErr)
			return dependencyError("maps", readErr)
		}
		return nil
	})

	if err := group.Wait(); err != nil {
		s.observe("unavailable", startedAt)
		return nil, err
	}
	if stats == nil {
		err := nilResponseError("stats")
		s.dependencyFailed(ctx, "stats", err)
		s.observe("unavailable", startedAt)
		return nil, dependencyError("stats", err)
	}

	recommendedMaps := mapPage.Data
	if len(recommendedMaps) > recommendedMapLimit {
		recommendedMaps = recommendedMaps[:recommendedMapLimit]
	}
	boundedMaps := make([]maps.MapDTO, len(recommendedMaps))
	copy(boundedMaps, recommendedMaps)

	response := &Response{
		Viewer:          viewerDTO(profile),
		Stats:           statsDTO(stats),
		DailyChallenge:  *daily,
		RecommendedMaps: boundedMaps,
	}
	s.observe("success", startedAt)
	s.logger.InfoContext(ctx, "authenticated home read completed", slog.String("outcome", "success"))
	return response, nil
}

func registeredUserID(sess *session.Context) (uuid.UUID, error) {
	if sess == nil || !sess.IsRegistered() || sess.UserID == nil {
		return uuid.Nil, ErrUnauthorized
	}
	userID, err := uuid.Parse(*sess.UserID)
	if err != nil {
		return uuid.Nil, ErrUnauthorized
	}
	return userID, nil
}

func dependencyError(dependency string, cause error) error {
	return fmt.Errorf("%w: %s: %v", ErrUnavailable, dependency, cause)
}

func nilResponseError(dependency string) error {
	return fmt.Errorf("%s returned a nil response", dependency)
}

func (s *Service) dependencyFailed(ctx context.Context, dependency string, err error) {
	s.metrics.ObserveDependencyFailure(dependency)
	s.logger.ErrorContext(ctx, "authenticated home dependency failed",
		slog.String("dependency", dependency),
		slog.Any("error", err),
	)
}

func (s *Service) observe(outcome string, startedAt time.Time) {
	s.metrics.ObserveRead(outcome, time.Since(startedAt))
}
