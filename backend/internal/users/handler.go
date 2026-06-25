package users

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	apphttp "github.com/raven/geoguess/backend/internal/http"
)

// Handler handles user HTTP endpoints.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler returns a new users handler.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// RegisterRoutes mounts user routes.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/users/{userId}/stats", h.GetUserStats)
}

// GetUserStats handles GET /users/{userId}/stats.
func (h *Handler) GetUserStats(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	resp, err := h.service.GetUserStats(r.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidUserID), errors.Is(err, ErrUserNotFound):
			apphttp.Error(w, r, h.logger, apphttp.ErrNotFound)
		default:
			apphttp.Error(w, r, h.logger, err)
		}
		return
	}
	apphttp.OK(w, r, resp)
}
