package profiles

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	apphttp "github.com/raven/geoguess/backend/internal/http"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
)

// Handler handles profile HTTP endpoints.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler returns a new profiles handler.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// RecordRateLimited records and logs a profile update rejected before it
// reaches the handler. It deliberately avoids logging cookie or session data.
func (h *Handler) RecordRateLimited(r *http.Request) {
	if h == nil || h.service == nil {
		return
	}
	h.service.metrics.RecordRateLimited()
	if h.logger != nil {
		h.logger.InfoContext(r.Context(), "profile update rate limited", slog.String("path", r.URL.Path))
	}
}

// RegisterRoutes mounts profile routes. Callers are expected to wrap the
// mutating route with auth, CSRF, and rate-limit middleware as appropriate
// (see app/routes.go).
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/profile", h.GetCurrentProfile)
	r.Patch("/profile", h.UpdateProfile)
	r.Get("/users/{userId}/stats", h.GetPublicProfile)
	r.Get("/users/{userId}/games", h.GetGameHistory)
}

// GetCurrentProfile handles GET /profile.
func (h *Handler) GetCurrentProfile(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	resp, err := h.service.GetCurrentProfile(r.Context(), *sess)
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}

// UpdateProfile handles PATCH /profile.
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())

	var req UpdateProfileRequest
	if err := apphttp.DecodeJSON(w, r, &req); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}

	profile, err := h.service.UpdateProfile(r.Context(), *sess, req)
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, profile)
}

// GetPublicProfile handles GET /users/{userId}/stats.
func (h *Handler) GetPublicProfile(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	resp, err := h.service.GetPublicProfile(r.Context(), userID)
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}

// GetGameHistory handles GET /users/{userId}/games.
func (h *Handler) GetGameHistory(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	cursor := r.URL.Query().Get("cursor")

	resp, err := h.service.GetGameHistory(r.Context(), userID, limit, cursor)
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}
