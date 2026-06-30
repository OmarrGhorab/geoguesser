package games

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/locations"
	"github.com/raven/geoguess/backend/internal/maps"
	"github.com/raven/geoguess/backend/internal/platform/clock"
	"github.com/raven/geoguess/backend/internal/session"
	"github.com/redis/go-redis/v9"
)

// LocationSelector selects active locations for a map.
type LocationSelector interface {
	SelectLocations(ctx context.Context, mapID uuid.UUID, count int) ([]maps.SelectedLocation, error)
}

type GameCompletionHook interface {
	OnGameCompleted(ctx context.Context, gameID uuid.UUID, completedAt time.Time) error
}

type Service struct {
	repo           *Repository
	selector       LocationSelector
	media          LocationMediaProvider
	clock          clock.Clock
	logger         *slog.Logger
	idempotency    IdempotencyStore
	metrics        MetricsRecorder
	completionHook GameCompletionHook
}

// NewService returns a solo game service.
func NewService(repo *Repository, selector LocationSelector, clk clock.Clock, logger *slog.Logger) *Service {
	return NewServiceWithMedia(repo, selector, locations.StaticProvider{}, clk, logger)
}

// NewServiceWithMedia returns a solo game service with an explicit media provider.
func NewServiceWithMedia(repo *Repository, selector LocationSelector, media LocationMediaProvider, clk clock.Clock, logger *slog.Logger) *Service {
	return NewServiceWithOptions(repo, selector, media, clk, logger, nil, nil)
}

func NewServiceWithOptions(repo *Repository, selector LocationSelector, media LocationMediaProvider, clk clock.Clock, logger *slog.Logger, idempotency IdempotencyStore, metrics MetricsRecorder) *Service {
	return NewServiceWithHook(repo, selector, media, clk, logger, idempotency, metrics, nil)
}

func NewServiceWithHook(repo *Repository, selector LocationSelector, media LocationMediaProvider, clk clock.Clock, logger *slog.Logger, idempotency IdempotencyStore, metrics MetricsRecorder, completionHook GameCompletionHook) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{repo: repo, selector: selector, media: media, clock: clk, logger: logger, idempotency: idempotency, metrics: metrics, completionHook: completionHook}
}

// CreateGame creates a pending solo game.
func (s *Service) CreateGame(ctx context.Context, sess *session.Context, req CreateGameRequest) (*GameResponse, error) {
	owner, err := ownerFromSession(sess)
	if err != nil {
		return nil, err
	}
	if req.Mode != GameModeSolo || req.MapID == uuid.Nil {
		return nil, ErrInvalidGameRequest
	}
	if req.RoundCount == 0 {
		req.RoundCount = 5
	}
	if req.RoundCount < 1 || req.RoundCount > 10 {
		return nil, ErrInvalidGameRequest
	}
	if req.TimerSeconds != nil && (*req.TimerSeconds < 10 || *req.TimerSeconds > 600) {
		return nil, ErrInvalidGameRequest
	}

	selected, err := s.selector.SelectLocations(ctx, req.MapID, req.RoundCount)
	if err != nil {
		return nil, err
	}
	if len(selected) < req.RoundCount {
		return nil, ErrNotEnoughLocations
	}
	selected = uniqueSelectedLocations(selected, req.RoundCount)
	if len(selected) < req.RoundCount {
		return nil, ErrNotEnoughLocations
	}

	game := &Game{
		Mode:            GameModeSolo,
		Status:          GameStatusPending,
		MapID:           req.MapID,
		CreatedByUserID: owner.userID,
		RoundCount:      req.RoundCount,
		TimerSeconds:    req.TimerSeconds,
		ScoringVersion:  ScoringVersionV1,
	}
	player := &GamePlayer{
		UserID:            owner.userID,
		GuestIdentityHash: owner.guestHash,
		DisplayName:       owner.displayName,
		Role:              PlayerRolePlayer,
		Status:            PlayerStatusActive,
	}
	rounds := make([]Round, req.RoundCount)
	for i := range rounds {
		rounds[i] = Round{
			LocationID:  selected[i].ID,
			RoundNumber: i + 1,
			Status:      RoundStatusPending,
		}
	}
	if err := s.repo.CreateGameBundle(ctx, game, player, rounds); err != nil {
		return nil, err
	}
	s.logger.InfoContext(ctx, "solo game created",
		slog.String("game_id", game.ID.String()),
		slog.String("map_id", game.MapID.String()),
		slog.Int("round_count", game.RoundCount),
	)
	return &GameResponse{Game: toGameDTO(*game)}, nil
}

// GetGame returns visible game state for the owner.
func (s *Service) GetGame(ctx context.Context, sess *session.Context, gameID string) (*GameResponse, error) {
	game, player, err := s.loadOwnedGame(ctx, sess, gameID)
	if err != nil {
		return nil, err
	}
	if player == nil {
		return nil, ErrForbidden
	}
	current, err := s.repo.GetCurrentRound(ctx, game.ID)
	if err != nil {
		return nil, err
	}
	if current != nil {
		n := current.RoundNumber
		game.CurrentRoundNumber = &n
	}
	return &GameResponse{Game: toGameDTO(*game)}, nil
}

// StartGame starts a pending solo game.
func (s *Service) StartGame(ctx context.Context, sess *session.Context, gameID string) (*GameResponse, error) {
	game, _, err := s.loadOwnedGame(ctx, sess, gameID)
	if err != nil {
		return nil, err
	}
	if !CanStart(game.Status) {
		return nil, ErrInvalidTransition
	}
	started, err := s.repo.StartGame(ctx, game.ID, s.clock.Now(), game.TimerSeconds)
	if err != nil {
		return nil, err
	}
	if started == nil {
		return nil, ErrGameNotFound
	}
	s.logger.InfoContext(ctx, "solo game started",
		slog.String("game_id", started.ID.String()),
		slog.Int("round_count", started.RoundCount),
	)
	return &GameResponse{Game: toGameDTO(*started)}, nil
}

// GetCurrentRound returns the current round without hidden coordinates.
func (s *Service) GetCurrentRound(ctx context.Context, sess *session.Context, gameID string) (*CurrentRoundResponse, error) {
	game, _, err := s.loadOwnedGame(ctx, sess, gameID)
	if err != nil {
		return nil, err
	}
	if game.Status != GameStatusActive {
		return nil, ErrGameNotActive
	}
	row, err := s.repo.GetCurrentRound(ctx, game.ID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrRoundNotFound
	}
	return &CurrentRoundResponse{Round: s.toRoundDTO(*row)}, nil
}

// SubmitGuess submits one guess for the current round.
func (s *Service) SubmitGuess(ctx context.Context, sess *session.Context, gameID, roundID, idempotencyKey string, req SubmitGuessRequest) (*GuessResultResponse, error) {
	startedAt := time.Now()
	outcome := "accepted"
	defer func() {
		if s.metrics != nil {
			s.metrics.ObserveGuessSubmission(outcome, time.Since(startedAt))
		}
	}()
	if req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 {
		outcome = "rejected"
		s.logger.InfoContext(ctx, "solo game guess rejected", slog.String("reason", "invalid_guess"))
		return nil, ErrInvalidGuess
	}
	game, player, err := s.loadOwnedGame(ctx, sess, gameID)
	if err != nil {
		outcome = "rejected"
		s.logger.InfoContext(ctx, "solo game guess rejected", slog.String("reason", "load_game_failed"), slog.Any("error", err))
		return nil, err
	}
	if game.Status != GameStatusActive {
		outcome = "rejected"
		s.logger.InfoContext(ctx, "solo game guess rejected", slog.String("game_id", game.ID.String()), slog.String("reason", "game_not_active"))
		return nil, ErrGameNotActive
	}
	parsedRoundID, err := uuid.Parse(roundID)
	if err != nil {
		outcome = "rejected"
		s.logger.InfoContext(ctx, "solo game guess rejected", slog.String("game_id", game.ID.String()), slog.String("reason", "round_not_found"))
		return nil, ErrRoundNotFound
	}
	current, err := s.repo.GetCurrentRound(ctx, game.ID)
	if err != nil {
		outcome = "rejected"
		return nil, err
	}
	if current == nil || current.RoundID != parsedRoundID {
		outcome = "rejected"
		s.logger.InfoContext(ctx, "solo game guess rejected", slog.String("game_id", game.ID.String()), slog.String("round_id", parsedRoundID.String()), slog.String("reason", "round_not_current"))
		return nil, ErrRoundNotCurrent
	}
	now := s.clock.Now()
	if current.EndsAt != nil && now.After(*current.EndsAt) {
		outcome = "rejected"
		s.logger.InfoContext(ctx, "solo game guess rejected", slog.String("game_id", game.ID.String()), slog.String("round_id", parsedRoundID.String()), slog.String("reason", "round_closed"))
		return nil, ErrRoundClosed
	}
	guess := Guess{
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	}
	key := strings.TrimSpace(idempotencyKey)
	var releaseClaim func(context.Context)
	if key != "" {
		existing, err := s.repo.GetGuessByIdempotencyKey(ctx, player.ID, key)
		if err != nil {
			outcome = "rejected"
			return nil, err
		}
		if existing != nil {
			if existing.RoundID != parsedRoundID || existing.Latitude != req.Latitude || existing.Longitude != req.Longitude {
				outcome = "conflict"
				s.logger.InfoContext(ctx, "solo game guess rejected", slog.String("game_id", game.ID.String()), slog.String("round_id", parsedRoundID.String()), slog.String("reason", "idempotency_conflict"))
				return nil, ErrIdempotencyConflict
			}
			outcome = "replay"
			return s.guessReplayResponse(ctx, *existing)
		}
		if s.idempotency != nil {
			claimed, err := s.idempotency.Claim(ctx, idempotencyClaimKey(player.ID, key), 2*time.Minute)
			if err != nil {
				outcome = "rejected"
				return nil, err
			}
			if !claimed {
				outcome = "conflict"
				s.logger.InfoContext(ctx, "solo game guess rejected", slog.String("game_id", game.ID.String()), slog.String("round_id", parsedRoundID.String()), slog.String("reason", "idempotency_in_flight"))
				return nil, ErrIdempotencyConflict
			}
			releaseClaim = func(releaseCtx context.Context) {
				_ = s.idempotency.Release(releaseCtx, idempotencyClaimKey(player.ID, key))
			}
		}
		guess.IdempotencyKey = &key
	}
	existing, err := s.repo.GetGuessByRoundPlayer(ctx, parsedRoundID, player.ID)
	if err != nil {
		outcome = "rejected"
		if releaseClaim != nil {
			releaseClaim(ctx)
		}
		return nil, err
	}
	if existing != nil {
		outcome = "conflict"
		if releaseClaim != nil {
			releaseClaim(ctx)
		}
		s.logger.InfoContext(ctx, "solo game guess rejected", slog.String("game_id", game.ID.String()), slog.String("round_id", parsedRoundID.String()), slog.String("reason", "already_guessed"))
		return nil, ErrAlreadyGuessed
	}
	saved, actual, completedGame, err := s.repo.SubmitGuessTx(ctx, game.ID, parsedRoundID, player.ID, guess, now)
	if err != nil {
		outcome = "rejected"
		if releaseClaim != nil {
			releaseClaim(ctx)
		}
		return nil, err
	}
	if saved == nil || actual == nil {
		outcome = "rejected"
		if releaseClaim != nil {
			releaseClaim(ctx)
		}
		return nil, ErrRoundNotFound
	}
	if releaseClaim != nil {
		releaseClaim(ctx)
	}
	s.logger.InfoContext(ctx, "solo game guess accepted",
		slog.String("game_id", game.ID.String()),
		slog.String("round_id", parsedRoundID.String()),
		slog.Int("score", saved.Score),
		slog.Int("distance_meters", saved.DistanceMeters),
	)
	if saved.RoundID == parsedRoundID {
		s.logger.InfoContext(ctx, "solo game round completed",
			slog.String("game_id", game.ID.String()),
			slog.String("round_id", parsedRoundID.String()),
			slog.Int("round_number", current.RoundNumber),
		)
	}
	if completedGame {
		if s.metrics != nil {
			s.metrics.RecordGameCompleted()
		}
		s.logger.InfoContext(ctx, "solo game completed",
			slog.String("game_id", game.ID.String()),
			slog.Int("final_round_score", saved.Score),
		)
		if s.completionHook != nil {
			if err := s.completionHook.OnGameCompleted(ctx, game.ID, now); err != nil {
				s.logger.ErrorContext(ctx, "challenge completion hook failed", slog.String("game_id", game.ID.String()), slog.Any("error", err))
			}
		}
	}
	// Recalculate score from actual coordinates before returning if repository used full answer.
	return &GuessResultResponse{
		Guess: GuessResult{
			ID:             saved.ID,
			Latitude:       saved.Latitude,
			Longitude:      saved.Longitude,
			DistanceMeters: saved.DistanceMeters,
			Score:          saved.Score,
			SubmittedAt:    saved.SubmittedAt,
		},
		ActualLocation: toRevealedLocation(*actual),
	}, nil
}

// IdempotencyStore stores short-lived in-flight idempotency claims.
type IdempotencyStore interface {
	Claim(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Release(ctx context.Context, key string) error
}

// MetricsRecorder records solo game observations.
type MetricsRecorder interface {
	ObserveGuessSubmission(outcome string, duration time.Duration)
	RecordGameCompleted()
}

// RedisIdempotencyStore is a Redis-backed short-lived idempotency claim store.
type RedisIdempotencyStore struct {
	client *redis.Client
}

// NewRedisIdempotencyStore returns a Redis idempotency store.
func NewRedisIdempotencyStore(client *redis.Client) *RedisIdempotencyStore {
	return &RedisIdempotencyStore{client: client}
}

// Claim stores a key only if no in-flight request currently owns it.
func (s *RedisIdempotencyStore) Claim(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if s == nil || s.client == nil {
		return true, nil
	}
	ok, err := s.client.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("claim idempotency key: %w", err)
	}
	return ok, nil
}

// Release removes an in-flight idempotency claim after durable persistence or failure.
func (s *RedisIdempotencyStore) Release(ctx context.Context, key string) error {
	if s == nil || s.client == nil {
		return nil
	}
	if err := s.client.Del(ctx, key).Err(); err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("release idempotency key: %w", err)
	}
	return nil
}

func idempotencyClaimKey(playerID uuid.UUID, key string) string {
	return "games:idempotency:" + playerID.String() + ":" + key
}

// GetResults returns durable final results.
func (s *Service) GetResults(ctx context.Context, sess *session.Context, gameID string) (*GameResultsResponse, error) {
	game, _, err := s.loadOwnedGame(ctx, sess, gameID)
	if err != nil {
		return nil, err
	}
	if game.Status != GameStatusCompleted {
		return nil, ErrResultsNotReady
	}
	loadedGame, players, rounds, err := s.repo.LoadResults(ctx, game.ID)
	if err != nil {
		return nil, err
	}
	if loadedGame == nil {
		return nil, ErrGameNotFound
	}
	playerDTOs := make([]GamePlayerDTO, len(players))
	for i, player := range players {
		playerDTOs[i] = toGamePlayerDTO(player)
	}
	return &GameResultsResponse{
		Game:    toGameDTO(*loadedGame),
		Players: playerDTOs,
		Rounds:  rounds,
	}, nil
}

type ownerIdentity struct {
	userID      *uuid.UUID
	guestHash   *string
	displayName string
}

func ownerFromSession(sess *session.Context) (ownerIdentity, error) {
	if sess == nil {
		return ownerIdentity{}, ErrForbidden
	}
	if sess.IsRegistered() {
		id, err := uuid.Parse(*sess.UserID)
		if err != nil {
			return ownerIdentity{}, ErrForbidden
		}
		return ownerIdentity{userID: &id, displayName: "Player"}, nil
	}
	if sess.IsGuest() {
		guest := *sess.GuestID
		return ownerIdentity{guestHash: &guest, displayName: "Guest"}, nil
	}
	return ownerIdentity{}, ErrForbidden
}

func (s *Service) loadOwnedGame(ctx context.Context, sess *session.Context, gameID string) (*Game, *GamePlayer, error) {
	owner, err := ownerFromSession(sess)
	if err != nil {
		return nil, nil, err
	}
	parsedGameID, err := uuid.Parse(gameID)
	if err != nil {
		return nil, nil, ErrGameNotFound
	}
	game, err := s.repo.GetGameByID(ctx, parsedGameID)
	if err != nil {
		return nil, nil, err
	}
	if game == nil {
		return nil, nil, ErrGameNotFound
	}
	player, err := s.repo.GetSoloPlayer(ctx, game.ID)
	if err != nil {
		return nil, nil, err
	}
	if player == nil || !ownerMatches(owner, *player) {
		return nil, nil, ErrForbidden
	}
	return game, player, nil
}

func ownerMatches(owner ownerIdentity, player GamePlayer) bool {
	if owner.userID != nil && player.UserID != nil {
		return *owner.userID == *player.UserID
	}
	if owner.guestHash != nil && player.GuestIdentityHash != nil {
		return *owner.guestHash == *player.GuestIdentityHash
	}
	return false
}

func uniqueSelectedLocations(selected []maps.SelectedLocation, count int) []maps.SelectedLocation {
	seen := make(map[uuid.UUID]struct{}, count)
	unique := make([]maps.SelectedLocation, 0, count)
	for _, location := range selected {
		if location.ID == uuid.Nil {
			continue
		}
		if _, ok := seen[location.ID]; ok {
			continue
		}
		seen[location.ID] = struct{}{}
		unique = append(unique, location)
		if len(unique) == count {
			return unique
		}
	}
	return unique
}

func (s *Service) toRoundDTO(row currentRoundRow) RoundDTO {
	mediaURL := ""
	if s.media != nil {
		if resolved, err := s.media.MediaURL(row.Provider, row.ProviderRef); err == nil {
			mediaURL = resolved
		}
	}
	return RoundDTO{
		ID:          row.RoundID,
		RoundNumber: row.RoundNumber,
		Status:      row.RoundStatus,
		StartsAt:    row.StartsAt,
		EndsAt:      row.EndsAt,
		Media: &RoundMedia{
			Type:        locations.MediaType(row.Provider),
			URL:         mediaURL,
			Attribution: row.Attribution,
		},
	}
}

func toGameDTO(game Game) GameDTO {
	return GameDTO{
		ID:                 game.ID,
		Mode:               game.Mode,
		Status:             game.Status,
		MapID:              game.MapID,
		RoundCount:         game.RoundCount,
		TimerSeconds:       game.TimerSeconds,
		ScoringVersion:     game.ScoringVersion,
		CurrentRoundNumber: game.CurrentRoundNumber,
		TotalScore:         game.TotalScore,
		StartedAt:          game.StartedAt,
		CompletedAt:        game.CompletedAt,
	}
}

func toGamePlayerDTO(player GamePlayer) GamePlayerDTO {
	return GamePlayerDTO{
		ID:          player.ID,
		UserID:      player.UserID,
		DisplayName: player.DisplayName,
		Role:        player.Role,
		Status:      player.Status,
		TotalScore:  player.TotalScore,
	}
}

func toRevealedLocation(location answerLocation) RevealedLocation {
	return RevealedLocation{
		Latitude:    location.Latitude,
		Longitude:   location.Longitude,
		CountryCode: location.CountryCode,
		Region:      location.Region,
		Locality:    location.Locality,
	}
}

func (s *Service) guessReplayResponse(ctx context.Context, guess Guess) (*GuessResultResponse, error) {
	answer, err := s.repo.GetAnswerForRound(ctx, guess.RoundID)
	if err != nil {
		return nil, err
	}
	if answer == nil {
		return nil, ErrRoundNotFound
	}
	return &GuessResultResponse{
		Guess: GuessResult{
			ID:             guess.ID,
			Latitude:       guess.Latitude,
			Longitude:      guess.Longitude,
			DistanceMeters: guess.DistanceMeters,
			Score:          guess.Score,
			SubmittedAt:    guess.SubmittedAt,
		},
		ActualLocation: toRevealedLocation(*answer),
	}, nil
}
