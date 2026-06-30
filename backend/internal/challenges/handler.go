package challenges

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

type Handler struct {
	service ServiceAPI
	logger  *slog.Logger
}

type ServiceAPI interface {
	GetDaily(context.Context, *session.Context, string) (*ChallengeMetadataResponse, error)
	StartDailyAttempt(context.Context, *session.Context, string) (*ChallengeAttemptResponse, error)
	CreateShared(context.Context, *session.Context, string, CreateSharedChallengeRequest) (*ChallengeMetadataResponse, error)
	GetShared(context.Context, *session.Context, string) (*ChallengeMetadataResponse, error)
	StartChallengeAttempt(context.Context, *session.Context, string, string) (*ChallengeAttemptResponse, error)
	GetResults(context.Context, *session.Context, string) (*ResultResponse, error)
	GetLeaderboard(context.Context, *session.Context, string, int, string) (*LeaderboardResponse, error)
	GetDailyStreak(context.Context, *session.Context) (*StreakSummary, error)
	GetMissions(context.Context, *session.Context) ([]MissionSummary, error)
	ClaimMission(context.Context, *session.Context, string, string) (*MissionSummary, error)
}

func NewHandler(service ServiceAPI, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/challenges/daily", h.GetDaily)
	r.Post("/challenges/daily/attempts", h.StartDailyAttempt)
	r.Post("/challenges/shared", h.CreateShared)
	r.Get("/challenges/shared/{code}", h.GetShared)
	r.Post("/challenges/{challengeId}/attempts", h.StartChallengeAttempt)
	r.Get("/challenges/{challengeId}/results", h.GetResults)
	r.Get("/challenges/{challengeId}/leaderboard", h.GetLeaderboard)
	r.Get("/missions", h.GetMissions)
	r.Post("/missions/{missionId}/claim", h.ClaimMission)
	r.Get("/streaks/daily", h.GetDailyStreak)
}

func (h *Handler) GetDaily(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.GetDaily(r.Context(), appmiddleware.SessionFromContext(r.Context()), r.URL.Query().Get("date"))
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) StartDailyAttempt(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.StartDailyAttempt(r.Context(), appmiddleware.SessionFromContext(r.Context()), r.Header.Get("Idempotency-Key"))
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) CreateShared(w http.ResponseWriter, r *http.Request) {
	var req CreateSharedChallengeRequest
	if err := apphttp.DecodeJSON(w, r, &req); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}
	resp, err := h.service.CreateShared(r.Context(), appmiddleware.SessionFromContext(r.Context()), r.Header.Get("Idempotency-Key"), req)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.Created(w, r, resp)
}

func (h *Handler) GetShared(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.GetShared(r.Context(), appmiddleware.SessionFromContext(r.Context()), chi.URLParam(r, "code"))
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) StartChallengeAttempt(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.StartChallengeAttempt(r.Context(), appmiddleware.SessionFromContext(r.Context()), r.Header.Get("Idempotency-Key"), chi.URLParam(r, "challengeId"))
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) GetResults(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.GetResults(r.Context(), appmiddleware.SessionFromContext(r.Context()), chi.URLParam(r, "challengeId"))
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	cursor := r.URL.Query().Get("cursor")
	resp, err := h.service.GetLeaderboard(r.Context(), appmiddleware.SessionFromContext(r.Context()), chi.URLParam(r, "challengeId"), limit, cursor)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) GetDailyStreak(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.GetDailyStreak(r.Context(), appmiddleware.SessionFromContext(r.Context()))
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) GetMissions(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.GetMissions(r.Context(), appmiddleware.SessionFromContext(r.Context()))
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, map[string]any{"missions": resp})
}

func (h *Handler) ClaimMission(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.ClaimMission(r.Context(), appmiddleware.SessionFromContext(r.Context()), r.Header.Get("Idempotency-Key"), chi.URLParam(r, "missionId"))
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) mapError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidChallengeInput):
		apphttp.Error(w, r, h.logger, apphttp.ErrValidationFailed.WithCause(err))
	case errors.Is(err, ErrForbidden):
		apphttp.Error(w, r, h.logger, apphttp.ErrForbidden.WithCause(err))
	case errors.Is(err, ErrChallengeNotFound):
		apphttp.Error(w, r, h.logger, apphttp.ErrNotFound.WithCause(err))
	case errors.Is(err, ErrDuplicateAttempt), errors.Is(err, ErrIdempotencyConflict):
		apphttp.Error(w, r, h.logger, apphttp.ErrConflict.WithCause(err))
	case errors.Is(err, ErrChallengeUnavailable), errors.Is(err, ErrNotEnoughLocations), errors.Is(err, ErrResultsNotReady):
		apphttp.Error(w, r, h.logger, apphttp.ErrUnprocessable.WithCause(err))
	default:
		apphttp.Error(w, r, h.logger, err)
	}
}
