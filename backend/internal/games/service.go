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
	"gorm.io/gorm"
)

// LocationSelector selects active locations for a map.
type LocationSelector interface {
	SelectLocations(ctx context.Context, mapID uuid.UUID, count int) ([]maps.SelectedLocation, error)
}

type GameCompletionHook interface {
	OnGameCompleted(ctx context.Context, gameID uuid.UUID, completedAt time.Time) error
}

// RankedLifecycleHook is an optional seam for ranked match lifecycle (owned outside games).
// Non-transactional methods remain for best-effort paths; Tx methods run inside game writes
// so match rows commit or roll back with the game/round change.
type RankedLifecycleHook interface {
	OnRankedGameStarted(ctx context.Context, gameID uuid.UUID, at time.Time) error
	OnRankedGameCompleted(ctx context.Context, gameID uuid.UUID, at time.Time) error
	OnRankedGameCancelled(ctx context.Context, gameID uuid.UUID, at time.Time, failureCode string) error
	ApplyStartedInTx(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time) error
	ApplyCompletedInTx(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time) error
	ApplyCancelledInTx(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time, failureCode string) error
}

type Service struct {
	repo              *Repository
	selector          LocationSelector
	media             LocationMediaProvider
	clock             clock.Clock
	logger            *slog.Logger
	idempotency       IdempotencyStore
	metrics           MetricsRecorder
	completionHook    GameCompletionHook
	rankedHook        RankedLifecycleHook
	matchHook         MatchLifecycleHook
	hostedRoomHook    HostedRoomLifecycleHook
	multiplayerEvents MultiplayerEventSink
	revealPolicy      RoundRevealPolicy
	practiceCursorKey []byte
	// Quick Play server-owned defaults (empty map ID disables the endpoint).
	quickPlayMapID        uuid.UUID
	quickPlayRoundCount   int
	quickPlayTimerSeconds int
	// quickPlayStore is the narrow persistence surface used by StartQuickPlay.
	// It defaults to repo and exists so tests can exercise the persistence
	// path (create-and-start, replay, conflict) without PostgreSQL.
	quickPlayStore quickPlayStore
}

// quickPlayStore is the consumer-side persistence interface for Quick Play.
type quickPlayStore interface {
	GetGameByCreationIdempotencyKey(ctx context.Context, key string) (*Game, error)
	CreateAndStartGameBundle(ctx context.Context, game *Game, player *GamePlayer, rounds []Round, now time.Time) (*Game, error)
}

// WithMultiplayerEvents attaches post-commit round/match realtime fanout.
func (s *Service) WithMultiplayerEvents(events MultiplayerEventSink) *Service {
	s.multiplayerEvents = events
	return s
}

// WithHostedRoomLifecycle attaches the transactional hosted-room lifecycle.
func (s *Service) WithHostedRoomLifecycle(hook HostedRoomLifecycleHook) *Service {
	s.hostedRoomHook = hook
	return s
}

// WithPracticeCursorSigningSecret configures domain-separated HMAC signing for
// Practice history cursors. The secret must be stable across API instances.
func (s *Service) WithPracticeCursorSigningSecret(secret string) *Service {
	s.practiceCursorKey = derivePracticeCursorKey(secret)
	return s
}

// WithQuickPlayDefaults configures server-owned map, round count, and timer for
// POST /games/quick-play. Round count defaults to 5 and timer to 60 when
// non-positive values are provided. A nil map ID leaves Quick Play unavailable.
func (s *Service) WithQuickPlayDefaults(mapID uuid.UUID, rounds, timerSeconds int) *Service {
	if s == nil {
		return s
	}
	s.quickPlayMapID = mapID
	if rounds > 0 {
		s.quickPlayRoundCount = rounds
	} else {
		s.quickPlayRoundCount = 5
	}
	if timerSeconds > 0 {
		s.quickPlayTimerSeconds = timerSeconds
	} else {
		s.quickPlayTimerSeconds = 60
	}
	return s
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
	s := &Service{
		repo:           repo,
		selector:       selector,
		media:          media,
		clock:          clk,
		logger:         logger,
		idempotency:    idempotency,
		metrics:        metrics,
		completionHook: completionHook,
		revealPolicy:   DelayedRevealPolicy{},
	}
	if repo != nil {
		s.quickPlayStore = repo
	}
	return s
}

// WithRankedLifecycle attaches a ranked match lifecycle callback (no games→matchmaking import).
func (s *Service) WithRankedLifecycle(hook RankedLifecycleHook) *Service {
	s.rankedHook = hook
	return s
}

// WithMatchLifecycle attaches match lifecycle hooks used for team terminal results.
func (s *Service) WithMatchLifecycle(hook MatchLifecycleHook) *Service {
	s.matchHook = hook
	return s
}

// WithRevealPolicy overrides the multiplayer answer reveal policy (defaults to delayed).
func (s *Service) WithRevealPolicy(policy RoundRevealPolicy) *Service {
	if policy != nil {
		s.revealPolicy = policy
	}
	return s
}

// SweepExpiredTimedMultiplayerRounds closes a bounded batch of server-authoritative
// deadlines even when no client polls or submits after expiry.
func (s *Service) SweepExpiredTimedMultiplayerRounds(ctx context.Context, limit int) error {
	if s == nil || s.repo == nil {
		return nil
	}
	now := s.clock.Now()
	gameIDs, err := s.repo.ListExpiredTimedMultiplayerGameIDs(ctx, now, limit)
	if err != nil {
		s.observeRoundClose("all", "worker_error")
		return err
	}
	for _, gameID := range gameIDs {
		game, loadErr := s.repo.GetGameByID(ctx, gameID)
		if loadErr != nil {
			s.observeRoundClose("unknown", "worker_error")
			return fmt.Errorf("load expired multiplayer game %s: %w", gameID, loadErr)
		}
		if game == nil {
			continue
		}
		modeClass := "ranked"
		if IsPartyLobbyMode(game.Mode) {
			modeClass = "party_lobby"
		}
		outcome, closeErr := s.repo.CloseExpiredMultiplayerRound(ctx, gameID, now, s.multiplayerTxHooksForMode(game.Mode))
		if closeErr == nil || errors.Is(closeErr, ErrRoundClosed) || errors.Is(closeErr, ErrGameNotActive) {
			if closeErr == nil {
				s.publishMultiplayerOutcome(ctx, gameID, outcome)
				s.observeRoundClose(modeClass, "worker_closed")
			}
			continue
		}
		s.observeRoundClose(modeClass, "worker_error")
		return fmt.Errorf("close expired multiplayer round %s: %w", gameID, closeErr)
	}
	return nil
}

// CloseRoundIfAllSubmitted closes the current multiplayer round when every
// remaining eligible player has already guessed. Rooms call this after a
// departure shrinks the eligible set so the round does not wait out its timer
// on a player who left. Safe no-op for missing, non-multiplayer, or inactive
// games and for rounds still waiting on active players.
func (s *Service) CloseRoundIfAllSubmitted(ctx context.Context, gameID uuid.UUID) error {
	if s == nil || s.repo == nil {
		return nil
	}
	game, err := s.repo.GetGameByID(ctx, gameID)
	if err != nil {
		return err
	}
	if game == nil || !IsMultiplayerMode(game.Mode) || game.Status != GameStatusActive {
		return nil
	}
	outcome, err := s.repo.CompleteMultiplayerRoundIfAllSubmitted(ctx, gameID, s.clock.Now().UTC(), s.multiplayerTxHooksForMode(game.Mode))
	if err != nil {
		return err
	}
	if outcome == nil {
		return nil
	}
	s.publishMultiplayerOutcome(ctx, gameID, outcome)
	if outcome.GameCompleted && !IsProgressionNeutralMode(game.Mode) && s.completionHook != nil {
		if hookErr := s.completionHook.OnGameCompleted(ctx, gameID, s.clock.Now().UTC()); hookErr != nil {
			return fmt.Errorf("finalize multiplayer game after departure: %w", hookErr)
		}
	}
	return nil
}

func (s *Service) observeRoundClose(modeClass, outcome string) {
	type roundCloseMetrics interface {
		ObserveRoundClose(modeClass, outcome string)
	}
	if metrics, ok := s.metrics.(roundCloseMetrics); ok {
		metrics.ObserveRoundClose(modeClass, outcome)
	}
}

// StartQuickPlay atomically creates and starts a quick_play game using
// server-owned map/round/timer defaults (create + start run in a single
// repository transaction). Lifecycle is Solo-compatible (immediate reveal,
// non-multiplayer scoring). Idempotency-Key is required (16–128 chars) and is
// persisted on the game, so an identical retry replays the original game;
// the optional IdempotencyStore additionally guards concurrent double-submits.
func (s *Service) StartQuickPlay(ctx context.Context, sess *session.Context, idempotencyKey string) (*GameResponse, error) {
	owner, err := ownerFromSession(sess)
	if err != nil {
		return nil, err
	}
	key := strings.TrimSpace(idempotencyKey)
	if len(key) < 16 || len(key) > 128 {
		s.observeModeOperation(GameModeQuickPlay, "start", "invalid")
		return nil, ErrInvalidGameRequest
	}
	// Bounds for round count and timer are enforced at startup
	// (config.Validate + WithQuickPlayDefaults); no per-request re-validation.
	mapID := s.quickPlayMapID
	if mapID == uuid.Nil || s.quickPlayStore == nil {
		s.observeModeOperation(GameModeQuickPlay, "start", "unavailable")
		return nil, ErrQuickPlayUnavailable
	}
	roundCount := s.quickPlayRoundCount
	timerSeconds := s.quickPlayTimerSeconds

	creationKey := quickPlayIdempotencyClaimKey(owner, key)
	if replay, replayErr := s.quickPlayGameByCreationKey(ctx, creationKey); replayErr != nil {
		s.observeModeOperation(GameModeQuickPlay, "start", "error")
		return nil, replayErr
	} else if replay != nil {
		s.observeModeOperation(GameModeQuickPlay, "start", "replay")
		return replay, nil
	}

	var releaseClaim func()
	if s.idempotency != nil {
		claimed, claimErr := s.idempotency.Claim(ctx, creationKey, 2*time.Minute)
		if claimErr != nil {
			s.observeModeOperation(GameModeQuickPlay, "start", "error")
			return nil, claimErr
		}
		if !claimed {
			s.observeModeOperation(GameModeQuickPlay, "start", "conflict")
			return nil, ErrIdempotencyConflict
		}
		// Release on failure so a client retry after a failed attempt is not
		// locked out for the claim TTL. Success keeps the claim as a cheap
		// in-flight guard; the persisted creation key is the durable replay.
		// WithoutCancel: the release must still run when the request context
		// is already cancelled (client disconnect / deadline).
		releaseClaim = func() {
			if releaseErr := s.idempotency.Release(context.WithoutCancel(ctx), creationKey); releaseErr != nil {
				s.logger.WarnContext(ctx, "quick play idempotency release failed",
					slog.String("claim_key", creationKey), slog.Any("error", releaseErr))
			}
		}
	}

	selected, err := s.selector.SelectLocations(ctx, mapID, roundCount)
	if err != nil {
		if releaseClaim != nil {
			releaseClaim()
		}
		s.observeModeOperation(GameModeQuickPlay, "start", "error")
		return nil, err
	}
	selected = uniqueSelectedLocations(selected, roundCount)
	if len(selected) < roundCount {
		if releaseClaim != nil {
			releaseClaim()
		}
		s.observeModeOperation(GameModeQuickPlay, "start", "unavailable")
		return nil, ErrNotEnoughLocations
	}

	timer := timerSeconds
	game := &Game{
		Mode:                   GameModeQuickPlay,
		Status:                 GameStatusPending,
		MapID:                  mapID,
		CreatedByUserID:        owner.userID,
		RoundCount:             roundCount,
		TimerSeconds:           &timer,
		ScoringVersion:         ScoringVersionV1,
		CreationIdempotencyKey: &creationKey,
	}
	player := &GamePlayer{
		UserID:            owner.userID,
		GuestIdentityHash: owner.guestHash,
		DisplayName:       owner.displayName,
		Role:              PlayerRolePlayer,
		Status:            PlayerStatusActive,
	}
	rounds := make([]Round, roundCount)
	for i := range rounds {
		rounds[i] = Round{
			LocationID:  selected[i].ID,
			RoundNumber: i + 1,
			Status:      RoundStatusPending,
		}
	}
	started, err := s.quickPlayStore.CreateAndStartGameBundle(ctx, game, player, rounds, s.clock.Now())
	if err != nil {
		if releaseClaim != nil {
			releaseClaim()
		}
		if IsCreationIdempotencyConflict(err) {
			// A concurrent request with the same key won the insert; replay it.
			if replay, replayErr := s.quickPlayGameByCreationKey(ctx, creationKey); replayErr == nil && replay != nil {
				s.observeModeOperation(GameModeQuickPlay, "start", "replay")
				return replay, nil
			}
			s.observeModeOperation(GameModeQuickPlay, "start", "conflict")
			return nil, ErrIdempotencyConflict
		}
		s.observeModeOperation(GameModeQuickPlay, "start", "error")
		return nil, err
	}
	s.observeModeOperation(GameModeQuickPlay, "start", "success")
	s.logger.InfoContext(ctx, "quick play game started",
		slog.String("game_id", started.ID.String()),
		slog.String("map_id", mapID.String()),
		slog.Int("round_count", started.RoundCount),
		slog.Int("timer_seconds", timerSeconds),
	)
	return &GameResponse{Game: toGameDTO(*started)}, nil
}

// quickPlayGameByCreationKey returns the replayed response for a previously
// persisted creation key, with current_round_number populated when available.
func (s *Service) quickPlayGameByCreationKey(ctx context.Context, creationKey string) (*GameResponse, error) {
	existing, err := s.quickPlayStore.GetGameByCreationIdempotencyKey(ctx, creationKey)
	if err != nil || existing == nil {
		return nil, err
	}
	if existing.CurrentRoundNumber == nil && s.repo != nil {
		if current, currentErr := s.repo.GetCurrentRound(ctx, existing.ID); currentErr == nil && current != nil {
			n := current.RoundNumber
			existing.CurrentRoundNumber = &n
		}
	}
	return &GameResponse{Game: toGameDTO(*existing)}, nil
}

func quickPlayIdempotencyClaimKey(owner ownerIdentity, key string) string {
	actor := "anon"
	if owner.userID != nil {
		actor = "user:" + owner.userID.String()
	} else if owner.guestHash != nil {
		actor = "guest:" + *owner.guestHash
	}
	return "game:quick_play:" + actor + ":" + key
}

// CreateGame creates a pending solo game.
func (s *Service) CreateGame(ctx context.Context, sess *session.Context, req CreateGameRequest) (*GameResponse, error) {
	owner, err := ownerFromSession(sess)
	if err != nil {
		return nil, err
	}
	if (req.Mode != GameModeSolo && req.Mode != GameModePractice) || req.MapID == uuid.Nil {
		return nil, ErrInvalidGameRequest
	}
	if req.Mode == GameModePractice {
		if req.TimerSeconds != nil || req.RoundCount < 0 || req.RoundCount > 1 {
			return nil, ErrInvalidGameRequest
		}
		req.RoundCount = 1
	} else if req.RoundCount == 0 {
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
		Mode:            req.Mode,
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
	if game.Mode == GameModePractice {
		s.observeModeOperation(game.Mode, "create", "success")
	}
	s.logger.InfoContext(ctx, "game created",
		slog.String("game_id", game.ID.String()),
		slog.String("mode", game.Mode),
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
	if err := s.finalizeCompletedGame(ctx, game); err != nil {
		return nil, err
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

func (s *Service) StartPrivateRoomGame(ctx context.Context, gameID uuid.UUID) (*MultiplayerStart, error) {
	game, err := s.repo.GetGameByID(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if game == nil {
		return nil, ErrGameNotFound
	}
	selected, err := s.selector.SelectLocations(ctx, game.MapID, game.RoundCount)
	if err != nil {
		return nil, err
	}
	selected = uniqueSelectedLocations(selected, game.RoundCount)
	if len(selected) < game.RoundCount {
		return nil, ErrNotEnoughLocations
	}
	now := s.clock.Now()
	return s.repo.StartPrivateRoomGame(ctx, gameID, roundsFromSelected(gameID, selected, game.RoundCount), now, game.TimerSeconds, s.multiplayerTxHooksForMode(game.Mode))
}

func (s *Service) GetPrivateRoomRoundState(ctx context.Context, gameID uuid.UUID) (*MultiplayerRoundState, error) {
	game, err := s.repo.GetGameByID(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if game == nil || !IsPartyLobbyMode(game.Mode) {
		return nil, ErrGameNotFound
	}
	state, err := s.repo.GetMultiplayerRoundState(ctx, gameID)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	if state != nil && state.Status == RoundStatusActive && state.EndsAt != nil && !now.Before(*state.EndsAt) {
		outcome, closeErr := s.repo.CloseExpiredMultiplayerRound(ctx, gameID, now, s.multiplayerTxHooksForMode(game.Mode))
		if closeErr != nil && !errors.Is(closeErr, ErrRoundClosed) {
			return nil, closeErr
		}
		s.publishMultiplayerOutcome(ctx, gameID, outcome)
		state, err = s.repo.GetMultiplayerRoundState(ctx, gameID)
		if err != nil {
			return nil, err
		}
	}
	s.resolveMultiplayerMedia(state)
	return state, nil
}

// GetCurrentRound returns the current round without hidden coordinates.
// For multiplayer/ranked modes, expired deadlines advance rounds/games before the read.
func (s *Service) GetCurrentRound(ctx context.Context, sess *session.Context, gameID string) (*CurrentRoundResponse, error) {
	game, player, err := s.loadOwnedGame(ctx, sess, gameID)
	if err != nil {
		return nil, err
	}
	if err := s.finalizeCompletedGame(ctx, game); err != nil {
		return nil, err
	}
	if game.Status != GameStatusActive {
		return nil, ErrGameNotActive
	}
	now := s.clock.Now()
	if game.Mode == GameModeDaily {
		row, err := s.repo.GetCurrentRound(ctx, game.ID)
		if err != nil {
			return nil, err
		}
		if row != nil && row.EndsAt != nil && !now.Before(*row.EndsAt) {
			saved, _, completed, err := s.repo.ExpireSoloRoundTx(ctx, game.ID, row.RoundID, player.ID, now)
			if err != nil && !errors.Is(err, ErrRoundClosed) {
				return nil, err
			}
			if completed && saved != nil && s.completionHook != nil {
				if err := s.completionHook.OnGameCompleted(ctx, game.ID, saved.SubmittedAt); err != nil {
					return nil, fmt.Errorf("finalize completed daily game: %w", err)
				}
			}
			if completed {
				return nil, ErrGameNotActive
			}
		}
	}
	if IsMultiplayerMode(game.Mode) {
		row, err := s.repo.GetCurrentRound(ctx, game.ID)
		if err != nil {
			return nil, err
		}
		if row != nil && row.EndsAt != nil && !now.Before(*row.EndsAt) {
			outcome, closeErr := s.repo.CloseExpiredMultiplayerRound(ctx, game.ID, now, s.multiplayerTxHooks())
			if closeErr != nil && !errors.Is(closeErr, ErrRoundClosed) {
				return nil, closeErr
			}
			s.publishMultiplayerOutcome(ctx, game.ID, outcome)
			// Game may have completed via deadline; re-check.
			game, err = s.repo.GetGameByID(ctx, game.ID)
			if err != nil {
				return nil, err
			}
			if game == nil {
				return nil, ErrGameNotFound
			}
			if game.Status != GameStatusActive {
				return nil, ErrGameNotActive
			}
		}
	}
	row, err := s.repo.GetCurrentRound(ctx, game.ID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrRoundNotFound
	}
	dto := s.toRoundDTO(*row)
	// Multiplayer progress fields (submitted/eligible) for Casual and Ranked clients.
	if IsMultiplayerMode(game.Mode) {
		if state, err := s.repo.GetMultiplayerRoundState(ctx, game.ID); err == nil && state != nil && state.RoundID == row.RoundID {
			submitted := state.SubmittedCount
			eligible := state.EligibleCount
			dto.SubmittedCount = &submitted
			dto.EligibleCount = &eligible
		}
	}
	return &CurrentRoundResponse{Round: dto}, nil
}

func (s *Service) multiplayerTxHooks() MultiplayerTxHooks {
	hooks := MultiplayerTxHooks{}
	if s.rankedHook != nil {
		hooks.OnMatchActive = func(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, now time.Time) error {
			return s.rankedHook.ApplyStartedInTx(ctx, tx, gameID, now)
		}
		hooks.OnGameCompleted = func(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, now time.Time) error {
			return s.rankedHook.ApplyCompletedInTx(ctx, tx, gameID, now)
		}
		hooks.OnGameCancelled = func(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, now time.Time, failureCode string) error {
			return s.rankedHook.ApplyCancelledInTx(ctx, tx, gameID, now, failureCode)
		}
	}
	if s.matchHook != nil {
		// Prefer MatchLifecycleHook for terminal completion when present.
		hooks.OnGameCompleted = func(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, now time.Time) error {
			// When terminal result was already applied via OnTerminalResult, this is idempotent.
			if s.rankedHook != nil {
				if err := s.rankedHook.ApplyCompletedInTx(ctx, tx, gameID, now); err != nil {
					return err
				}
			}
			return s.matchHook.ApplyGameCompletedInTx(ctx, tx, gameID, now)
		}
		hooks.OnGameCancelled = func(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, now time.Time, failureCode string) error {
			if s.rankedHook != nil {
				if err := s.rankedHook.ApplyCancelledInTx(ctx, tx, gameID, now, failureCode); err != nil {
					return err
				}
			}
			return s.matchHook.ApplyGameCancelledInTx(ctx, tx, gameID, now, failureCode)
		}
		hooks.OnMatchActive = func(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, now time.Time) error {
			if s.rankedHook != nil {
				if err := s.rankedHook.ApplyStartedInTx(ctx, tx, gameID, now); err != nil {
					return err
				}
			}
			return s.matchHook.ApplyMatchActiveInTx(ctx, tx, gameID, now)
		}
		// Wire terminal team-result finalization when the adapter supports it.
		if applier, ok := s.matchHook.(TerminalResultApplier); ok {
			hooks.OnTerminalResult = func(ctx context.Context, tx *gorm.DB, result TerminalMatchResult) error {
				return applier.ApplyTerminalResultInTx(ctx, tx, result)
			}
		}
	}
	return hooks
}

func (s *Service) multiplayerTxHooksForMode(mode string) MultiplayerTxHooks {
	if IsPartyLobbyMode(mode) {
		hooks := MultiplayerTxHooks{}
		if s.hostedRoomHook != nil {
			hooks.OnMatchActive = func(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, now time.Time) error {
				return s.hostedRoomHook.ApplyRoomStartedInTx(ctx, tx, gameID, now)
			}
			hooks.OnGameCompleted = func(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, now time.Time) error {
				return s.hostedRoomHook.ApplyRoomCompletedInTx(ctx, tx, gameID, now)
			}
		}
		return hooks
	}
	return s.multiplayerTxHooks()
}

// AbandonRankedGame abandons an active ranked multiplayer game and cancels the durable match atomically.
func (s *Service) AbandonRankedGame(ctx context.Context, gameID uuid.UUID) error {
	if s == nil || s.repo == nil {
		return ErrGameNotFound
	}
	game, err := s.repo.GetGameByID(ctx, gameID)
	if err != nil {
		return err
	}
	if game == nil || game.Mode != GameModeRanked {
		return ErrGameNotFound
	}
	return s.repo.CancelMultiplayerGameTx(ctx, gameID, s.clock.Now().UTC(), GameStatusAbandoned, "", s.multiplayerTxHooks())
}

// CancelRankedGame cancels a ranked multiplayer game (optional failureCode maps to failed_to_start when still matched).
func (s *Service) CancelRankedGame(ctx context.Context, gameID uuid.UUID, failureCode string) error {
	if s == nil || s.repo == nil {
		return ErrGameNotFound
	}
	game, err := s.repo.GetGameByID(ctx, gameID)
	if err != nil {
		return err
	}
	if game == nil || game.Mode != GameModeRanked {
		return ErrGameNotFound
	}
	return s.repo.CancelMultiplayerGameTx(ctx, gameID, s.clock.Now().UTC(), GameStatusCancelled, failureCode, s.multiplayerTxHooks())
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
	owner, err := ownerFromSession(sess)
	if err != nil {
		outcome = "rejected"
		return nil, err
	}
	parsedGameID, err := uuid.Parse(gameID)
	if err != nil {
		outcome = "rejected"
		return nil, ErrGameNotFound
	}
	game, err := s.repo.GetGameByID(ctx, parsedGameID)
	if err != nil {
		outcome = "rejected"
		return nil, err
	}
	if game == nil {
		outcome = "rejected"
		return nil, ErrGameNotFound
	}
	var player *GamePlayer
	if IsMultiplayerMode(game.Mode) {
		player, err = s.repo.GetPlayerByOwner(ctx, game.ID, owner)
		if err != nil {
			outcome = "rejected"
			return nil, err
		}
		if player == nil {
			outcome = "rejected"
			return nil, ErrForbidden
		}
	} else {
		player, err = s.repo.GetSoloPlayer(ctx, game.ID)
		if err != nil {
			outcome = "rejected"
			return nil, err
		}
		if player == nil || !ownerMatches(owner, *player) {
			outcome = "rejected"
			if game.Mode == GameModePractice {
				return nil, ErrGameNotFound
			}
			return nil, ErrForbidden
		}
	}
	key := strings.TrimSpace(idempotencyKey)
	if game.Mode == GameModePractice && key != "" {
		if len(key) < 16 || len(key) > 128 {
			outcome = "rejected"
			return nil, ErrInvalidGameRequest
		}
		existing, replayErr := s.repo.GetGuessByIdempotencyKey(ctx, player.ID, key)
		if replayErr != nil {
			outcome = "rejected"
			return nil, replayErr
		}
		if existing != nil {
			replayRoundID, parseErr := uuid.Parse(roundID)
			if parseErr != nil {
				outcome = "rejected"
				return nil, ErrRoundNotFound
			}
			if existing.RoundID != replayRoundID || existing.Latitude != req.Latitude || existing.Longitude != req.Longitude {
				outcome = "conflict"
				return nil, ErrIdempotencyConflict
			}
			outcome = "replay"
			return s.guessReplayResponse(ctx, *existing)
		}
	}
	if game.Status != GameStatusActive {
		outcome = "rejected"
		s.logger.InfoContext(ctx, "solo game guess rejected", slog.String("game_id", game.ID.String()), slog.String("reason", "game_not_active"))
		return nil, ErrGameNotActive
	}
	if IsMultiplayerMode(game.Mode) {
		return s.submitPrivateRoomGuess(ctx, game, player, roundID, idempotencyKey, req)
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
		if s.idempotency != nil && game.Mode != GameModePractice {
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
				// WithoutCancel: the release must still run when the request
				// context is already cancelled (client disconnect / deadline),
				// or retries stay locked out for the full claim TTL.
				if releaseErr := s.idempotency.Release(context.WithoutCancel(releaseCtx), idempotencyClaimKey(player.ID, key)); releaseErr != nil {
					s.logger.WarnContext(releaseCtx, "guess idempotency release failed", slog.Any("error", releaseErr))
				}
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
		if game.Mode == GameModePractice && key != "" {
			replay, replayErr := s.repo.GetGuessByIdempotencyKey(ctx, player.ID, key)
			if replayErr != nil {
				return nil, replayErr
			}
			if replay != nil {
				if replay.RoundID != parsedRoundID || replay.Latitude != req.Latitude || replay.Longitude != req.Longitude {
					outcome = "conflict"
					return nil, ErrIdempotencyConflict
				}
				outcome = "replay"
				return s.guessReplayResponse(ctx, *replay)
			}
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
				return nil, fmt.Errorf("finalize completed game: %w", err)
			}
		}
	}
	loc := toRevealedLocation(*actual)
	resp := soloGuessResponse(*saved, &loc, true, completedGame)
	if game.Mode == GameModePractice {
		resp.NextRoundAvailable = true
	}
	return resp, nil
}

// NextPracticeRound appends one owner-only untimed round after the current
// Practice round completes. Durable idempotency is stored with the round.
func (s *Service) NextPracticeRound(ctx context.Context, sess *session.Context, gameID, idempotencyKey string) (*CurrentRoundResponse, error) {
	game, _, err := s.loadOwnedGame(ctx, sess, gameID)
	if err != nil {
		return nil, err
	}
	if game.Mode != GameModePractice {
		return nil, ErrWrongGameMode
	}
	key := strings.TrimSpace(idempotencyKey)
	if len(key) < 16 || len(key) > 128 {
		return nil, ErrInvalidGameRequest
	}
	replay, err := s.repo.GetPracticeRoundByCreationKey(ctx, game.ID, key)
	if err != nil {
		return nil, err
	}
	if replay != nil {
		return &CurrentRoundResponse{Round: s.toRoundDTO(*replay)}, nil
	}
	selected, err := s.selector.SelectLocations(ctx, game.MapID, 1)
	if err != nil {
		s.observeModeOperation(GameModePractice, "next_round", "error")
		return nil, err
	}
	selected = uniqueSelectedLocations(selected, 1)
	if len(selected) != 1 {
		s.observeModeOperation(GameModePractice, "next_round", "error")
		return nil, ErrNotEnoughLocations
	}
	row, err := s.repo.AppendPracticeRound(ctx, game.ID, selected[0].ID, key, s.clock.Now().UTC())
	if err != nil {
		s.observeModeOperation(GameModePractice, "next_round", "error")
		return nil, err
	}
	if row == nil {
		return nil, ErrRoundNotFound
	}
	s.observeModeOperation(GameModePractice, "next_round", "success")
	return &CurrentRoundResponse{Round: s.toRoundDTO(*row)}, nil
}

// GetPracticeHistory returns one bounded owner-only round page.
func (s *Service) GetPracticeHistory(ctx context.Context, sess *session.Context, gameID, cursor string, limit int) (*PracticeHistoryResponse, error) {
	game, player, err := s.loadOwnedGame(ctx, sess, gameID)
	if err != nil {
		return nil, err
	}
	if game.Mode != GameModePractice {
		return nil, ErrWrongGameMode
	}
	after, err := decodePracticeCursor(s.practiceCursorKey, cursor, game.ID)
	if err != nil {
		return nil, err
	}
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > 100 {
		return nil, ErrInvalidGameRequest
	}
	rows, hasMore, err := s.repo.ListPracticeHistory(ctx, game.ID, player.ID, after, limit)
	if err != nil {
		return nil, err
	}
	items := make([]PracticeRoundHistoryItem, 0, len(rows))
	for _, row := range rows {
		round := s.toRoundDTO(currentRoundRow{
			RoundID: row.RoundID, RoundNumber: row.RoundNumber, RoundStatus: row.RoundStatus,
			StartsAt: row.StartsAt, EndsAt: row.EndsAt, Provider: row.Provider,
			ProviderRef: row.ProviderRef, Attribution: row.Attribution,
		})
		item := PracticeRoundHistoryItem{Round: round}
		if row.GuessID != nil && row.GuessLatitude != nil && row.GuessLongitude != nil && row.DistanceMeters != nil && row.Score != nil && row.SubmittedAt != nil {
			accuracy := *row.Score
			if row.AccuracyScore != nil {
				accuracy = *row.AccuracyScore
			}
			bonus := 0
			if row.SpeedBonus != nil {
				bonus = *row.SpeedBonus
			}
			timedOut := row.TimedOut != nil && *row.TimedOut
			guess := GuessResult{ID: *row.GuessID, Latitude: *row.GuessLatitude, Longitude: *row.GuessLongitude,
				DistanceMeters: *row.DistanceMeters, AccuracyScore: accuracy, SpeedBonus: bonus,
				Score: *row.Score, SubmittedAt: *row.SubmittedAt, TimedOut: timedOut}
			item.Guess = &guess
			actual := RevealedLocation{Latitude: row.AnswerLatitude, Longitude: row.AnswerLongitude,
				CountryCode: row.CountryCode, Region: row.Region, Locality: row.Locality}
			item.ActualLocation = &actual
		}
		items = append(items, item)
	}
	resp := &PracticeHistoryResponse{Items: items, HasMore: hasMore}
	if hasMore && len(rows) > 0 {
		next := encodePracticeCursor(s.practiceCursorKey, game.ID, rows[len(rows)-1].RoundNumber)
		resp.NextCursor = &next
	}
	return resp, nil
}

// EndPractice explicitly terminates an open-ended session while preserving history.
func (s *Service) EndPractice(ctx context.Context, sess *session.Context, gameID string) (*GameResponse, error) {
	game, _, err := s.loadOwnedGame(ctx, sess, gameID)
	if err != nil {
		return nil, err
	}
	if game.Mode != GameModePractice {
		return nil, ErrWrongGameMode
	}
	ended, err := s.repo.EndPractice(ctx, game.ID, s.clock.Now().UTC())
	if err != nil {
		s.observeModeOperation(GameModePractice, "end", "error")
		return nil, err
	}
	s.observeModeOperation(GameModePractice, "end", "success")
	return &GameResponse{Game: toGameDTO(*ended)}, nil
}

func (s *Service) observeModeOperation(mode, operation, outcome string) {
	type modeOperationMetrics interface {
		ObserveModeOperation(mode, operation, outcome string)
	}
	if metrics, ok := s.metrics.(modeOperationMetrics); ok {
		metrics.ObserveModeOperation(mode, operation, outcome)
	}
}

// ExpireRound advances an elapsed daily round with a zero-score timeout.
func (s *Service) ExpireRound(ctx context.Context, sess *session.Context, gameID, roundID string) (*GuessResultResponse, error) {
	owner, err := ownerFromSession(sess)
	if err != nil {
		return nil, err
	}
	parsedGameID, err := uuid.Parse(gameID)
	if err != nil {
		return nil, ErrGameNotFound
	}
	parsedRoundID, err := uuid.Parse(roundID)
	if err != nil {
		return nil, ErrRoundNotFound
	}
	game, err := s.repo.GetGameByID(ctx, parsedGameID)
	if err != nil {
		return nil, err
	}
	if game == nil || game.Mode != GameModeDaily || game.Status != GameStatusActive {
		return nil, ErrGameNotActive
	}
	player, err := s.repo.GetSoloPlayer(ctx, game.ID)
	if err != nil {
		return nil, err
	}
	if player == nil || !ownerMatches(owner, *player) {
		return nil, ErrForbidden
	}
	saved, actual, completed, err := s.repo.ExpireSoloRoundTx(ctx, game.ID, parsedRoundID, player.ID, s.clock.Now())
	if err != nil {
		return nil, err
	}
	if saved == nil || actual == nil {
		return nil, ErrRoundNotFound
	}
	if completed && s.completionHook != nil {
		if err := s.completionHook.OnGameCompleted(ctx, game.ID, saved.SubmittedAt); err != nil {
			return nil, fmt.Errorf("finalize timed-out game: %w", err)
		}
	}
	loc := toRevealedLocation(*actual)
	return soloGuessResponse(*saved, &loc, true, completed), nil
}

func (s *Service) submitPrivateRoomGuess(ctx context.Context, game *Game, player *GamePlayer, roundID, idempotencyKey string, req SubmitGuessRequest) (*GuessResultResponse, error) {
	parsedRoundID, err := uuid.Parse(roundID)
	if err != nil {
		return nil, ErrRoundNotFound
	}
	current, err := s.repo.GetCurrentRound(ctx, game.ID)
	if err != nil {
		return nil, err
	}
	if current == nil || current.RoundID != parsedRoundID {
		return nil, ErrRoundNotCurrent
	}
	now := s.clock.Now()
	if !CanGuessBeforeStart(game.Mode, current.StartsAt, now) {
		return nil, ErrRoundClosed
	}
	if current.EndsAt != nil && now.After(*current.EndsAt) {
		// Advance expired multiplayer deadline, then reject this late guess.
		// Casual has no ends_at, so this path is ranked/private_room only.
		outcome, closeErr := s.repo.CloseExpiredMultiplayerRound(ctx, game.ID, now, s.multiplayerTxHooksForMode(game.Mode))
		if closeErr != nil && !errors.Is(closeErr, ErrRoundClosed) {
			return nil, closeErr
		}
		s.publishMultiplayerOutcome(ctx, game.ID, outcome)
		return nil, ErrRoundClosed
	}
	key := strings.TrimSpace(idempotencyKey)
	guess := Guess{Latitude: req.Latitude, Longitude: req.Longitude}
	if key != "" {
		existing, err := s.repo.GetGuessByIdempotencyKey(ctx, player.ID, key)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			if existing.RoundID != parsedRoundID || existing.Latitude != req.Latitude || existing.Longitude != req.Longitude {
				return nil, ErrIdempotencyConflict
			}
			return s.multiplayerGuessReplayResponse(ctx, game, *existing)
		}
		guess.IdempotencyKey = &key
	}
	existing, err := s.repo.GetGuessByRoundPlayer(ctx, parsedRoundID, player.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// Locked guesses: cannot resubmit or move.
		return nil, ErrAlreadyGuessed
	}
	// Match becomes active on first accepted multiplayer guess (same TX as score write).
	hooks := s.multiplayerTxHooksForMode(game.Mode)
	saved, actual, err := s.repo.SubmitMultiplayerGuessTx(ctx, game.ID, parsedRoundID, player.ID, guess, now, hooks)
	if err != nil {
		return nil, err
	}
	if saved == nil || actual == nil {
		return nil, ErrRoundNotFound
	}
	s.publishMultiplayerOutcome(ctx, game.ID, saved)
	if saved.GameCompleted && !IsProgressionNeutralMode(game.Mode) {
		if s.completionHook != nil {
			if err := s.completionHook.OnGameCompleted(ctx, game.ID, now); err != nil {
				return nil, fmt.Errorf("finalize multiplayer game: %w", err)
			}
		}
		if s.matchHook != nil {
			// Best-effort post-commit path (tx path already ran inside hooks when configured).
			_ = s.matchHook.OnGameCompleted(ctx, game.ID, now)
		}
	}
	return s.multiplayerGuessResponse(game.Mode, saved, actual), nil
}

func (s *Service) publishMultiplayerOutcome(ctx context.Context, gameID uuid.UUID, outcome *MultiplayerGuessOutcome) {
	if s == nil || s.multiplayerEvents == nil || outcome == nil || !outcome.RoundCompleted {
		return
	}
	if err := s.multiplayerEvents.PublishMultiplayerOutcome(ctx, gameID, *outcome); err != nil {
		s.logger.WarnContext(ctx, "multiplayer event publish failed", slog.String("game_id", gameID.String()), slog.Any("error", err))
	}
}

// GetSharedRoundResults returns revealed multiplayer round results after the shared round closes.
func (s *Service) GetSharedRoundResults(ctx context.Context, sess *session.Context, gameID, roundID string) (*SharedRoundResultsResponse, error) {
	game, _, err := s.loadOwnedGame(ctx, sess, gameID)
	if err != nil {
		return nil, err
	}
	if !IsMultiplayerMode(game.Mode) {
		return nil, ErrResultsNotReady
	}
	parsedRoundID, err := uuid.Parse(roundID)
	if err != nil {
		return nil, ErrRoundNotFound
	}
	return s.repo.LoadSharedRoundResults(ctx, game.ID, parsedRoundID)
}

// CreateTeamGame creates a matchmade Casual/Ranked team game (formation entrypoint for matchmaking).
func (s *Service) CreateTeamGame(ctx context.Context, input TeamGameFormationInput) (*TeamGameFormationResult, error) {
	if s == nil || s.repo == nil {
		return nil, ErrInvalidGameRequest
	}
	if input.ScoringVersion == 0 {
		input.ScoringVersion = ScoringVersionV1
	}
	if input.RoundCount == 0 {
		input.RoundCount = 5
	}
	// Ranked: enforce 60s default; Casual: force null timer (no deadline / no speed bonus).
	input.TimerSeconds = ApplyRankedTimerDefaults(input.Mode, input.TimerSeconds)
	return s.repo.CreateTeamGameBundle(ctx, input)
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
	if game.Mode == GameModePractice {
		return nil, ErrWrongGameMode
	}
	if game.Status != GameStatusCompleted {
		return nil, ErrResultsNotReady
	}
	if err := s.finalizeCompletedGame(ctx, game); err != nil {
		return nil, err
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
	resp := &GameResultsResponse{
		Game:    toGameDTO(*loadedGame),
		Players: playerDTOs,
		Rounds:  rounds,
	}
	// Matchmade team totals / exact draws for terminal casual & ranked games.
	if IsCasualMode(loadedGame.Mode) || IsRankedMode(loadedGame.Mode) {
		totals := SumTeamTotals(players)
		one, two := totals.TeamOneScore, totals.TeamTwoScore
		resp.TeamOneScore = &one
		resp.TeamTwoScore = &two
		result, winner := DecideTeamResult(one, two)
		resp.Result = &result
		resp.WinnerTeam = winner
	}
	return resp, nil
}

// finalizeCompletedGame makes post-game projections retriable from every owned read.
// The hook is idempotent, so a transient failure never strands a completed game.
func (s *Service) finalizeCompletedGame(ctx context.Context, game *Game) error {
	if game == nil || game.Status != GameStatusCompleted || s.completionHook == nil || IsProgressionNeutralMode(game.Mode) {
		return nil
	}
	completedAt := s.clock.Now()
	if game.CompletedAt != nil {
		completedAt = *game.CompletedAt
	}
	if err := s.completionHook.OnGameCompleted(ctx, game.ID, completedAt); err != nil {
		return fmt.Errorf("retry completed game finalization: %w", err)
	}
	return nil
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
	var player *GamePlayer
	if IsMultiplayerMode(game.Mode) {
		player, err = s.repo.GetPlayerByOwner(ctx, game.ID, owner)
		if err != nil {
			return nil, nil, err
		}
		if player == nil {
			return nil, nil, ErrForbidden
		}
	} else {
		player, err = s.repo.GetSoloPlayer(ctx, game.ID)
		if err != nil {
			return nil, nil, err
		}
		if player == nil || !ownerMatches(owner, *player) {
			if game.Mode == GameModePractice {
				return nil, nil, ErrGameNotFound
			}
			return nil, nil, ErrForbidden
		}
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
	panoramaID, hasPanoramaID := locations.PanoramaID(row.Provider, row.ProviderRef)
	if !hasPanoramaID && s.media != nil {
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
			PanoramaID:  panoramaID,
			Attribution: row.Attribution,
		},
	}
}

func (s *Service) resolveMultiplayerMedia(state *MultiplayerRoundState) {
	if state == nil {
		return
	}
	if s.media == nil {
		state.MediaURL = ""
		return
	}
	resolved, err := s.media.MediaURL(state.Provider, state.ProviderRef)
	if err != nil {
		state.MediaURL = ""
		return
	}
	state.MediaURL = resolved
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
		OpenEnded:          IsOpenEndedMode(game.Mode),
	}
}

func toGamePlayerDTO(player GamePlayer) GamePlayerDTO {
	return GamePlayerDTO{
		ID:          player.ID,
		UserID:      player.UserID,
		DisplayName: player.DisplayName,
		Role:        player.Role,
		Status:      player.Status,
		TeamSlot:    player.TeamSlot,
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

func soloGuessResponse(guess Guess, actual *RevealedLocation, roundCompleted, gameCompleted bool) *GuessResultResponse {
	return &GuessResultResponse{
		Guess:            toGuessResult(guess),
		ActualLocation:   actual,
		MaxScore:         MaxAccuracyScore(),
		ScorePercent:     accuracyPercent(guess.AccuracyScore, guess.Score),
		MaxAccuracyScore: MaxAccuracyScore(),
		MaxSpeedBonus:    0,
		Outcome:          guessOutcome(guess.AccuracyScore, guess.TimedOut),
		RoundCompleted:   roundCompleted,
		GameCompleted:    gameCompleted,
	}
}

func (s *Service) multiplayerGuessResponse(mode string, out *MultiplayerGuessOutcome, actual *answerLocation) *GuessResultResponse {
	if out == nil {
		return nil
	}
	policy := s.revealPolicy
	if policy == nil {
		policy = DelayedRevealPolicy{}
	}
	mayReveal := policy.MayRevealAnswer(mode, "", out.RoundCompleted)
	var loc *RevealedLocation
	if mayReveal && actual != nil {
		revealed := toRevealedLocation(*actual)
		loc = &revealed
	}
	submitted := out.SubmittedCount
	eligible := out.EligibleCount
	outcome := "submitted"
	if out.RoundCompleted {
		outcome = guessOutcome(out.Guess.AccuracyScore, out.Guess.TimedOut)
	}
	// Ranked advertises the 250-point speed-bonus cap; Casual/private_room stay at 0.
	maxBonus := 0
	if IsRankedMode(mode) {
		maxBonus = MaxSpeedBonusPoints()
	}
	return &GuessResultResponse{
		Guess:            toGuessResult(out.Guess),
		ActualLocation:   loc, // nil until shared round closes (delayed reveal)
		MaxScore:         MaxAccuracyScore(),
		ScorePercent:     accuracyPercent(out.Guess.AccuracyScore, out.Guess.Score),
		MaxAccuracyScore: MaxAccuracyScore(),
		MaxSpeedBonus:    maxBonus,
		Outcome:          outcome,
		RoundCompleted:   out.RoundCompleted,
		GameCompleted:    out.GameCompleted,
		SubmittedCount:   &submitted, // progress: how many active players have locked guesses
		EligibleCount:    &eligible,
		NextRoundNumber:  out.NextRoundNumber,
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
	loc := toRevealedLocation(*answer)
	return soloGuessResponse(guess, &loc, true, false), nil
}

func (s *Service) multiplayerGuessReplayResponse(ctx context.Context, game *Game, guess Guess) (*GuessResultResponse, error) {
	// Determine whether the round has closed so delayed reveal can apply.
	round, err := s.repo.GetRoundByID(ctx, guess.RoundID)
	if err != nil {
		return nil, err
	}
	if round == nil {
		return s.multiplayerGuessResponse(game.Mode, &MultiplayerGuessOutcome{Guess: guess}, nil), nil
	}
	roundCompleted := round.Status == RoundStatusCompleted
	var actual *answerLocation
	if roundCompleted {
		actual, err = s.repo.GetAnswerForRound(ctx, guess.RoundID)
		if err != nil {
			return nil, err
		}
	}
	// Best-effort progress counts for replay payloads.
	state, _ := s.repo.GetMultiplayerRoundState(ctx, game.ID)
	out := &MultiplayerGuessOutcome{
		Guess:          guess,
		RoundCompleted: roundCompleted,
		GameCompleted:  game.Status == GameStatusCompleted,
	}
	if state != nil {
		out.SubmittedCount = state.SubmittedCount
		out.EligibleCount = state.EligibleCount
	}
	return s.multiplayerGuessResponse(game.Mode, out, actual), nil
}

func accuracyPercent(accuracy, score int) int {
	base := accuracy
	if base == 0 && score > 0 && score <= MaxAccuracyScore() {
		base = score
	}
	if base <= 0 {
		return 0
	}
	return base * 100 / MaxAccuracyScore()
}

func guessOutcome(accuracyScore int, timedOut bool) string {
	if timedOut {
		return "timed_out"
	}
	if accuracyScore == MaxAccuracyScore() {
		return "perfect"
	}
	if accuracyScore >= 4000 {
		return "close"
	}
	return "miss"
}
