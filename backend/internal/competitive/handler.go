package competitive

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apphttp "github.com/raven/geoguess/backend/internal/http"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
)

// Handler serves authenticated competitive profile/history/season/leaderboard routes.
type Handler struct {
	service *Service
	logger  *slog.Logger
	// FeatureEnabled gates the competitive read API (typically RankedTeamModesEnabled).
	FeatureEnabled bool
}

// NewHandler constructs a competitive HTTP handler.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{service: service, logger: logger, FeatureEnabled: true}
}

// WithFeatureEnabled toggles the competitive read feature flag.
func (h *Handler) WithFeatureEnabled(enabled bool) *Handler {
	if h != nil {
		h.FeatureEnabled = enabled
	}
	return h
}

// RecordRateLimited is the rate-limiter observer for competitive reads.
func (h *Handler) RecordRateLimited(r *http.Request) {
	if h == nil || h.logger == nil {
		return
	}
	h.logger.InfoContext(r.Context(), "competitive rate limited", slog.String("path", r.URL.Path))
}

// GetProfile handles GET /competitive/profile.
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	if !h.featureOK(w, r) {
		return
	}
	sess := appmiddleware.SessionFromContext(r.Context())
	if sess == nil {
		apphttp.Error(w, r, h.logger, ToAPIError(ErrUnauthorized))
		return
	}
	resp, err := h.service.GetProfile(r.Context(), *sess)
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}

// GetLeaderboard handles GET /competitive/leaderboard.
func (h *Handler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	if !h.featureOK(w, r) {
		return
	}
	sess := appmiddleware.SessionFromContext(r.Context())
	if sess == nil {
		apphttp.Error(w, r, h.logger, ToAPIError(ErrUnauthorized))
		return
	}
	limit, err := parseLimitParam(r)
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	resp, err := h.service.GetLeaderboard(r.Context(), *sess, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}

// GetHistory handles GET /competitive/history.
func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	if !h.featureOK(w, r) {
		return
	}
	sess := appmiddleware.SessionFromContext(r.Context())
	if sess == nil {
		apphttp.Error(w, r, h.logger, ToAPIError(ErrUnauthorized))
		return
	}
	limit, err := parseLimitParam(r)
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	resp, err := h.service.GetHistory(r.Context(), *sess, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}

// ListSeasons handles GET /competitive/seasons.
func (h *Handler) ListSeasons(w http.ResponseWriter, r *http.Request) {
	if !h.featureOK(w, r) {
		return
	}
	sess := appmiddleware.SessionFromContext(r.Context())
	if sess == nil {
		apphttp.Error(w, r, h.logger, ToAPIError(ErrUnauthorized))
		return
	}
	resp, err := h.service.ListSeasons(r.Context(), *sess)
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}

// GetSeasonLeaderboard handles GET /competitive/seasons/{seasonId}/leaderboard.
func (h *Handler) GetSeasonLeaderboard(w http.ResponseWriter, r *http.Request) {
	if !h.featureOK(w, r) {
		return
	}
	sess := appmiddleware.SessionFromContext(r.Context())
	if sess == nil {
		apphttp.Error(w, r, h.logger, ToAPIError(ErrUnauthorized))
		return
	}
	seasonID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "seasonId")))
	if err != nil || seasonID == uuid.Nil {
		apphttp.Error(w, r, h.logger, apphttp.ErrNotFound)
		return
	}
	limit, err := parseLimitParam(r)
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	resp, err := h.service.GetClosedLeaderboard(r.Context(), *sess, seasonID, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) featureOK(w http.ResponseWriter, r *http.Request) bool {
	if h == nil || h.service == nil {
		apphttp.Error(w, r, h.logger, ToAPIError(ErrDependencyFailure))
		return false
	}
	if !h.FeatureEnabled {
		apphttp.Error(w, r, h.logger, ToAPIError(ErrFeatureDisabled))
		return false
	}
	return true
}

func parseLimitParam(r *http.Request) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return 0, nil // service normalizes default
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 {
		return 0, ErrInvalidLimit
	}
	return limit, nil
}
