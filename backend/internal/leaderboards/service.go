package leaderboards

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/challenges"
	apphttp "github.com/raven/geoguess/backend/internal/http"
	"github.com/raven/geoguess/backend/internal/platform/clock"
	"github.com/raven/geoguess/backend/internal/session"
)

const (
	defaultLimit = 20
	maxLimit     = 100
	maxCursorLen = 512
)

type store interface {
	EnsureGlobalLeaderboard(ctx context.Context) (*Leaderboard, error)
	EnsureMapLeaderboard(ctx context.Context, mapID uuid.UUID) (*Leaderboard, error)
	GetDailyChallengeByDate(ctx context.Context, date time.Time) (*challenges.Challenge, error)
	ListGeneralEntries(ctx context.Context, leaderboardID uuid.UUID, limit int, cursor string) ([]Entry, error)
	ListDailyEntries(ctx context.Context, challengeID uuid.UUID, limit int, cursor string) ([]challenges.LeaderboardEntry, error)
	MaterializeCompletedGame(ctx context.Context, gameID uuid.UUID) ([]uuid.UUID, error)
	DailyCacheScopeForGame(ctx context.Context, gameID uuid.UUID) (*string, error)
}

// Service owns public registered-user leaderboard reads and materialization.
type Service struct {
	repo          store
	cache         pageCache
	clock         clock.Clock
	logger        *slog.Logger
	resetHourUTC  int
	challengeHook GameCompletionHook
}

type GameCompletionHook interface {
	OnGameCompleted(ctx context.Context, gameID uuid.UUID, completedAt time.Time) error
}

type dailyChallengeProvider interface {
	GetDaily(ctx context.Context, sess *session.Context, dateOverride string) (*challenges.ChallengeMetadataResponse, error)
}

func NewService(repo store, cache pageCache, clk clock.Clock, logger *slog.Logger, resetHourUTC int, challengeHook GameCompletionHook) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{repo: repo, cache: cache, clock: clk, logger: logger, resetHourUTC: resetHourUTC, challengeHook: challengeHook}
}

func (s *Service) GetGlobal(ctx context.Context, limit int, cursor string) (*Response, error) {
	limit, err := normalizeLimit(limit)
	if err != nil {
		return nil, err
	}
	if err := validateCursor(cursor); err != nil {
		return nil, err
	}
	board, err := s.repo.EnsureGlobalLeaderboard(ctx)
	if err != nil {
		return nil, err
	}
	return s.getGeneral(ctx, "global", board.ID, limit, cursor)
}

func (s *Service) GetMap(ctx context.Context, rawMapID string, limit int, cursor string) (*Response, error) {
	limit, err := normalizeLimit(limit)
	if err != nil {
		return nil, err
	}
	if err := validateCursor(cursor); err != nil {
		return nil, err
	}
	mapID, err := uuid.Parse(rawMapID)
	if err != nil {
		return nil, ErrInvalidMapID
	}
	board, err := s.repo.EnsureMapLeaderboard(ctx, mapID)
	if err != nil {
		return nil, err
	}
	if board == nil {
		return nil, ErrLeaderboardNotFound
	}
	return s.getGeneral(ctx, "map:"+mapID.String(), board.ID, limit, cursor)
}

func (s *Service) GetDaily(ctx context.Context, limit int, cursor string, dateOverride string) (*Response, error) {
	limit, err := normalizeLimit(limit)
	if err != nil {
		return nil, err
	}
	if err := validateCursor(cursor); err != nil {
		return nil, err
	}
	date, _, _ := challenges.DailyWindow(s.clock.Now(), s.resetHourUTC)
	if strings.TrimSpace(dateOverride) != "" {
		parsed, parseErr := time.Parse("2006-01-02", strings.TrimSpace(dateOverride))
		if parseErr != nil {
			return nil, ErrInvalidDate
		}
		date = parsed
	}
	challenge, err := s.repo.GetDailyChallengeByDate(ctx, date)
	if err != nil {
		return nil, err
	}
	if challenge == nil {
		if provider, ok := s.challengeHook.(dailyChallengeProvider); ok {
			if _, err := provider.GetDaily(ctx, nil, date.Format("2006-01-02")); err != nil {
				if errors.Is(err, challenges.ErrChallengeUnavailable) || errors.Is(err, challenges.ErrNotEnoughLocations) {
					return nil, ErrLeaderboardNotFound
				}
				return nil, err
			}
			challenge, err = s.repo.GetDailyChallengeByDate(ctx, date)
			if err != nil {
				return nil, err
			}
		}
		if challenge == nil {
			return nil, ErrLeaderboardNotFound
		}
	}
	scope := "daily:" + date.Format("2006-01-02")
	key, cacheErr := s.cacheKey(ctx, scope, limit, cursor)
	if cacheErr != nil {
		s.logger.InfoContext(ctx, "leaderboard cache version read failed", slog.String("scope", "daily"), slog.Any("error", cacheErr))
	} else if cached, err := s.cacheGet(ctx, key); err != nil {
		s.logger.InfoContext(ctx, "leaderboard cache read failed", slog.String("scope", "daily"), slog.Any("error", err))
	} else if cached != nil {
		return cached, nil
	}
	entries, err := s.repo.ListDailyEntries(ctx, challenge.ID, limit+1, cursor)
	if err != nil {
		return nil, err
	}
	hasNext := len(entries) > limit
	if hasNext {
		entries = entries[:limit]
	}
	resp := &Response{Data: dailyDTOs(entries), Page: pageInfo(limit, dailyNextCursor(entries, hasNext))}
	if cacheErr == nil {
		if err := s.cacheSet(ctx, key, resp); err != nil {
			s.logger.InfoContext(ctx, "leaderboard cache write failed", slog.String("scope", "daily"), slog.Any("error", err))
		}
	}
	s.logger.InfoContext(ctx, "daily leaderboard read", slog.String("challenge_id", challenge.ID.String()), slog.Int("entries", len(resp.Data)))
	return resp, nil
}

func (s *Service) OnGameCompleted(ctx context.Context, gameID uuid.UUID, completedAt time.Time) error {
	if s.challengeHook != nil {
		if err := s.challengeHook.OnGameCompleted(ctx, gameID, completedAt); err != nil {
			s.logger.ErrorContext(ctx, "challenge completion hook failed", slog.String("game_id", gameID.String()), slog.Any("error", err))
		}
	}
	dailyScope, err := s.repo.DailyCacheScopeForGame(ctx, gameID)
	if err != nil {
		return err
	}
	if dailyScope != nil {
		if err := s.invalidateScope(ctx, *dailyScope); err != nil {
			s.logger.InfoContext(ctx, "daily leaderboard cache invalidation failed", slog.String("scope", *dailyScope), slog.Any("error", err))
		}
	}
	boards, err := s.repo.MaterializeCompletedGame(ctx, gameID)
	if err != nil {
		return err
	}
	for _, boardID := range boards {
		if err := s.invalidateBoard(ctx, boardID); err != nil {
			s.logger.InfoContext(ctx, "leaderboard cache invalidation failed", slog.String("leaderboard_id", boardID.String()), slog.Any("error", err))
		}
	}
	return nil
}

func (s *Service) getGeneral(ctx context.Context, scope string, leaderboardID uuid.UUID, limit int, cursor string) (*Response, error) {
	cacheScope := leaderboardID.String()
	key, cacheErr := s.cacheKey(ctx, cacheScope, limit, cursor)
	if cacheErr != nil {
		s.logger.InfoContext(ctx, "leaderboard cache version read failed", slog.String("scope", scope), slog.Any("error", cacheErr))
	} else if cached, err := s.cacheGet(ctx, key); err != nil {
		s.logger.InfoContext(ctx, "leaderboard cache read failed", slog.String("scope", scope), slog.Any("error", err))
	} else if cached != nil {
		return cached, nil
	}
	entries, err := s.repo.ListGeneralEntries(ctx, leaderboardID, limit+1, cursor)
	if err != nil {
		return nil, err
	}
	hasNext := len(entries) > limit
	if hasNext {
		entries = entries[:limit]
	}
	resp := &Response{Data: generalDTOs(entries), Page: pageInfo(limit, generalNextCursor(entries, hasNext))}
	if cacheErr == nil {
		if err := s.cacheSet(ctx, key, resp); err != nil {
			s.logger.InfoContext(ctx, "leaderboard cache write failed", slog.String("scope", scope), slog.Any("error", err))
		}
	}
	s.logger.InfoContext(ctx, "leaderboard read", slog.String("scope", scope), slog.Int("entries", len(resp.Data)))
	return resp, nil
}

func (s *Service) cacheKey(ctx context.Context, scope string, limit int, cursor string) (string, error) {
	version := int64(1)
	if s.cache != nil {
		loadedVersion, err := s.cache.Version(ctx, scope)
		if err != nil {
			return "", err
		}
		version = loadedVersion
	}
	return cacheKey(scope, version, limit, cursor), nil
}

func (s *Service) cacheGet(ctx context.Context, key string) (*Response, error) {
	if s.cache == nil {
		return nil, nil
	}
	return s.cache.Get(ctx, key)
}

func (s *Service) cacheSet(ctx context.Context, key string, response *Response) error {
	if s.cache == nil {
		return nil
	}
	return s.cache.Set(ctx, key, response)
}

func (s *Service) invalidateBoard(ctx context.Context, boardID uuid.UUID) error {
	return s.invalidateScope(ctx, boardID.String())
}

func (s *Service) invalidateScope(ctx context.Context, scope string) error {
	if s.cache == nil {
		return nil
	}
	return s.cache.BumpVersion(ctx, scope)
}

func normalizeLimit(limit int) (int, error) {
	if limit == 0 {
		return defaultLimit, nil
	}
	if limit < 0 {
		return 0, ErrInvalidLimit
	}
	if limit > maxLimit {
		return 0, ErrInvalidLimit
	}
	return limit, nil
}

func validateCursor(cursor string) error {
	if len(cursor) > maxCursorLen {
		return ErrInvalidCursor
	}
	return nil
}

func cacheKey(scope string, version int64, limit int, cursor string) string {
	return "leaderboard:v1:" + scope + ":version:" + strconv.FormatInt(version, 10) + ":limit:" + strconv.Itoa(limit) + ":cursor:" + cursor
}

func pageInfo(limit int, nextCursor *string) apphttp.PageInfo {
	return apphttp.PageInfo{Limit: limit, NextCursor: nextCursor}
}

func generalDTOs(entries []Entry) []EntryDTO {
	dtos := make([]EntryDTO, 0, len(entries))
	for _, entry := range entries {
		dtos = append(dtos, EntryDTO{
			Rank:        entry.Rank,
			UserID:      entry.UserID,
			DisplayName: entry.DisplayNameSnapshot,
			Score:       entry.Score,
			GamesPlayed: entry.GamesPlayed,
		})
	}
	return dtos
}

func dailyDTOs(entries []challenges.LeaderboardEntry) []EntryDTO {
	dtos := make([]EntryDTO, 0, len(entries))
	for _, entry := range entries {
		dtos = append(dtos, EntryDTO{
			Rank:        entry.Rank,
			UserID:      entry.UserID,
			DisplayName: entry.DisplayNameSnapshot,
			Score:       entry.Score,
			GamesPlayed: 1,
		})
	}
	return dtos
}

func generalNextCursor(entries []Entry, hasNext bool) *string {
	if !hasNext || len(entries) == 0 {
		return nil
	}
	last := entries[len(entries)-1]
	return ptr(encodeCursor(last.Score, last.CompletionDurationMS, last.CompletedAt, last.UserID))
}

func dailyNextCursor(entries []challenges.LeaderboardEntry, hasNext bool) *string {
	if !hasNext || len(entries) == 0 {
		return nil
	}
	last := entries[len(entries)-1]
	return ptr(encodeCursor(last.Score, last.CompletionDurationMS, last.CompletedAt, last.AttemptID))
}

func ptr(s string) *string {
	return &s
}

func wrapInvalidCursor(err error) error {
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCursor, err)
	}
	return nil
}
