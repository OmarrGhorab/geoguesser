package games

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	apphttp "github.com/raven/geoguess/backend/internal/http"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
	"github.com/raven/geoguess/backend/internal/session"
)

// Handler handles solo game endpoints.
type Handler struct {
	service ServiceAPI
	logger  *slog.Logger
}

// ServiceAPI is the games service surface used by HTTP handlers.
type ServiceAPI interface {
	CreateGame(rctx context.Context, sess *session.Context, req CreateGameRequest) (*GameResponse, error)
	StartQuickPlay(rctx context.Context, sess *session.Context, idempotencyKey string) (*GameResponse, error)
	GetGame(rctx context.Context, sess *session.Context, gameID string) (*GameResponse, error)
	StartGame(rctx context.Context, sess *session.Context, gameID string) (*GameResponse, error)
	GetCurrentRound(rctx context.Context, sess *session.Context, gameID string) (*CurrentRoundResponse, error)
	SubmitGuess(rctx context.Context, sess *session.Context, gameID, roundID, idempotencyKey string, req SubmitGuessRequest) (*GuessResultResponse, error)
	ExpireRound(rctx context.Context, sess *session.Context, gameID, roundID string) (*GuessResultResponse, error)
	GetResults(rctx context.Context, sess *session.Context, gameID string) (*GameResultsResponse, error)
	GetSharedRoundResults(rctx context.Context, sess *session.Context, gameID, roundID string) (*SharedRoundResultsResponse, error)
	NextPracticeRound(rctx context.Context, sess *session.Context, gameID, idempotencyKey string) (*CurrentRoundResponse, error)
	GetPracticeHistory(rctx context.Context, sess *session.Context, gameID, cursor string, limit int) (*PracticeHistoryResponse, error)
	EndPractice(rctx context.Context, sess *session.Context, gameID string) (*GameResponse, error)
}

// NewHandler returns a new handler.
func NewHandler(service ServiceAPI, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// Route-class names passed to a RouteLimiterProvider. Production wiring keys
// rate-limit buckets by these values.
const (
	RouteClassGameCreate   = "game-create"
	RouteClassQuickPlay    = "quick-play"
	RouteClassGuess        = "guess"
	RouteClassGuessTimeout = "guess-timeout"
	RouteClassPracticeNext = "practice-next"
	RouteClassPracticeEnd  = "practice-end"
)

// RouteLimiterProvider returns per-route-class middleware (e.g. rate limits)
// for the given route class; a nil return attaches none.
type RouteLimiterProvider func(class string) func(http.Handler) http.Handler

// RegisterRoutes mounts game routes without per-route middleware (tests).
func (h *Handler) RegisterRoutes(r chi.Router) {
	h.RegisterRoutesWith(r, nil)
}

// RegisterRoutesWith is the single source of truth for /games routes; both
// tests (RegisterRoutes) and production wiring (app.NewRouter with a limiter
// provider) mount through it, so the two can never diverge.
func (h *Handler) RegisterRoutesWith(r chi.Router, limiter RouteLimiterProvider) {
	with := func(g chi.Router, class string) chi.Router {
		if limiter != nil {
			if mw := limiter(class); mw != nil {
				return g.With(mw)
			}
		}
		return g
	}
	r.Route("/games", func(g chi.Router) {
		with(g, RouteClassGameCreate).Post("/", h.CreateGame)
		with(g, RouteClassQuickPlay).Post("/quick-play", h.StartQuickPlay)
		g.Get("/{gameId}", h.GetGame)
		g.Post("/{gameId}/start", h.StartGame)
		g.Get("/{gameId}/rounds/current", h.GetCurrentRound)
		with(g, RouteClassGuess).Post("/{gameId}/rounds/{roundId}/guesses", h.SubmitGuess)
		with(g, RouteClassGuessTimeout).Post("/{gameId}/rounds/{roundId}/timeout", h.ExpireRound)
		// Party Lobby / multiplayer shared reveal recovery (participant-only).
		g.Get("/{gameId}/rounds/{roundId}/results", h.GetSharedRoundResults)
		g.Get("/{gameId}/results", h.GetResults)
		with(g, RouteClassPracticeNext).Post("/{gameId}/rounds/next", h.NextPracticeRound)
		g.Get("/{gameId}/rounds", h.GetPracticeHistory)
		with(g, RouteClassPracticeEnd).Post("/{gameId}/end", h.EndPractice)
	})
}

// StartQuickPlay handles POST /games/quick-play.
func (h *Handler) StartQuickPlay(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.StartQuickPlay(
		r.Context(),
		appmiddleware.SessionFromContext(r.Context()),
		r.Header.Get("Idempotency-Key"),
	)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.Created(w, r, resp)
}

func (h *Handler) NextPracticeRound(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.NextPracticeRound(r.Context(), appmiddleware.SessionFromContext(r.Context()), chi.URLParam(r, "gameId"), r.Header.Get("Idempotency-Key"))
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) GetPracticeHistory(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			h.mapError(w, r, ErrInvalidGameRequest)
			return
		}
		limit = parsed
	}
	resp, err := h.service.GetPracticeHistory(r.Context(), appmiddleware.SessionFromContext(r.Context()), chi.URLParam(r, "gameId"), r.URL.Query().Get("cursor"), limit)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) EndPractice(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.EndPractice(r.Context(), appmiddleware.SessionFromContext(r.Context()), chi.URLParam(r, "gameId"))
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

// ExpireRound handles a server-authoritative daily round timeout.
func (h *Handler) ExpireRound(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.ExpireRound(r.Context(), appmiddleware.SessionFromContext(r.Context()), chi.URLParam(r, "gameId"), chi.URLParam(r, "roundId"))
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

// CreateGame handles POST /games.
func (h *Handler) CreateGame(w http.ResponseWriter, r *http.Request) {
	sc := appmiddleware.SessionFromContext(r.Context())
	var req CreateGameRequest
	if err := apphttp.DecodeJSON(w, r, &req); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}
	resp, err := h.service.CreateGame(r.Context(), sc, req)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.Created(w, r, resp)
}

// GetGame handles GET /games/{gameId}.
func (h *Handler) GetGame(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.GetGame(r.Context(), appmiddleware.SessionFromContext(r.Context()), chi.URLParam(r, "gameId"))
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

// StartGame handles POST /games/{gameId}/start.
func (h *Handler) StartGame(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.StartGame(r.Context(), appmiddleware.SessionFromContext(r.Context()), chi.URLParam(r, "gameId"))
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

// GetCurrentRound handles GET /games/{gameId}/rounds/current.
func (h *Handler) GetCurrentRound(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.GetCurrentRound(r.Context(), appmiddleware.SessionFromContext(r.Context()), chi.URLParam(r, "gameId"))
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

// SubmitGuess handles POST /games/{gameId}/rounds/{roundId}/guesses.
func (h *Handler) SubmitGuess(w http.ResponseWriter, r *http.Request) {
	var req SubmitGuessRequest
	if err := apphttp.DecodeJSON(w, r, &req); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}
	resp, err := h.service.SubmitGuess(r.Context(), appmiddleware.SessionFromContext(r.Context()), chi.URLParam(r, "gameId"), chi.URLParam(r, "roundId"), r.Header.Get("Idempotency-Key"), req)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

// GetResults handles GET /games/{gameId}/results.
func (h *Handler) GetResults(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.GetResults(r.Context(), appmiddleware.SessionFromContext(r.Context()), chi.URLParam(r, "gameId"))
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

// GetSharedRoundResults handles GET /games/{gameId}/rounds/{roundId}/results.
// Multiplayer only; returns answer and all guesses after the shared round closes.
func (h *Handler) GetSharedRoundResults(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.GetSharedRoundResults(
		r.Context(),
		appmiddleware.SessionFromContext(r.Context()),
		chi.URLParam(r, "gameId"),
		chi.URLParam(r, "roundId"),
	)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	if resp == nil {
		apphttp.Error(w, r, h.logger, apphttp.ErrNotFound)
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) mapError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidCursor):
		apphttp.Error(w, r, h.logger, apphttp.NewAPIError(http.StatusBadRequest, CodeInvalidCursor, MsgInvalidCursor).WithCause(err))
	case errors.Is(err, ErrInvalidGameRequest), errors.Is(err, ErrInvalidGuess):
		apphttp.Error(w, r, h.logger, apphttp.ErrValidationFailed.WithCause(err))
	case errors.Is(err, ErrForbidden):
		apphttp.Error(w, r, h.logger, apphttp.ErrForbidden.WithCause(err))
	case errors.Is(err, ErrGameNotFound), errors.Is(err, ErrRoundNotFound):
		apphttp.Error(w, r, h.logger, apphttp.ErrNotFound.WithCause(err))
	case errors.Is(err, ErrAlreadyGuessed), errors.Is(err, ErrIdempotencyConflict):
		apphttp.Error(w, r, h.logger, apphttp.ErrConflict.WithCause(err))
	case errors.Is(err, ErrQuickPlayUnavailable):
		apphttp.Error(w, r, h.logger, apphttp.NewAPIError(http.StatusServiceUnavailable, CodeQuickPlayUnavailable, MsgQuickPlayUnavailable).WithCause(err))
	case errors.Is(err, ErrWrongGameMode):
		apphttp.Error(w, r, h.logger, apphttp.NewAPIError(http.StatusUnprocessableEntity, CodeWrongGameMode, MsgWrongGameMode).WithCause(err))
	case errors.Is(err, ErrCurrentRoundIncomplete):
		apphttp.Error(w, r, h.logger, apphttp.NewAPIError(http.StatusUnprocessableEntity, CodeCurrentRoundIncomplete, MsgCurrentRoundIncomplete).WithCause(err))
	case errors.Is(err, ErrInvalidTransition), errors.Is(err, ErrGameNotActive), errors.Is(err, ErrRoundClosed), errors.Is(err, ErrRoundNotCurrent), errors.Is(err, ErrNotEnoughLocations), errors.Is(err, ErrResultsNotReady):
		apphttp.Error(w, r, h.logger, apphttp.ErrUnprocessable.WithCause(err))
	default:
		apphttp.Error(w, r, h.logger, err)
	}
}
