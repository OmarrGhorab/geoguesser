package games

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

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
	GetGame(rctx context.Context, sess *session.Context, gameID string) (*GameResponse, error)
	StartGame(rctx context.Context, sess *session.Context, gameID string) (*GameResponse, error)
	GetCurrentRound(rctx context.Context, sess *session.Context, gameID string) (*CurrentRoundResponse, error)
	SubmitGuess(rctx context.Context, sess *session.Context, gameID, roundID, idempotencyKey string, req SubmitGuessRequest) (*GuessResultResponse, error)
	GetResults(rctx context.Context, sess *session.Context, gameID string) (*GameResultsResponse, error)
}

// NewHandler returns a new handler.
func NewHandler(service ServiceAPI, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// RegisterRoutes mounts game routes.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/games", func(g chi.Router) {
		g.Post("/", h.CreateGame)
		g.Get("/{gameId}", h.GetGame)
		g.Post("/{gameId}/start", h.StartGame)
		g.Get("/{gameId}/rounds/current", h.GetCurrentRound)
		g.Post("/{gameId}/rounds/{roundId}/guesses", h.SubmitGuess)
		g.Get("/{gameId}/results", h.GetResults)
	})
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

func (h *Handler) mapError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidGameRequest), errors.Is(err, ErrInvalidGuess):
		apphttp.Error(w, r, h.logger, apphttp.ErrValidationFailed.WithCause(err))
	case errors.Is(err, ErrForbidden):
		apphttp.Error(w, r, h.logger, apphttp.ErrForbidden.WithCause(err))
	case errors.Is(err, ErrGameNotFound), errors.Is(err, ErrRoundNotFound):
		apphttp.Error(w, r, h.logger, apphttp.ErrNotFound.WithCause(err))
	case errors.Is(err, ErrAlreadyGuessed), errors.Is(err, ErrIdempotencyConflict):
		apphttp.Error(w, r, h.logger, apphttp.ErrConflict.WithCause(err))
	case errors.Is(err, ErrInvalidTransition), errors.Is(err, ErrGameNotActive), errors.Is(err, ErrRoundClosed), errors.Is(err, ErrRoundNotCurrent), errors.Is(err, ErrNotEnoughLocations), errors.Is(err, ErrResultsNotReady):
		apphttp.Error(w, r, h.logger, apphttp.ErrUnprocessable.WithCause(err))
	default:
		apphttp.Error(w, r, h.logger, err)
	}
}
