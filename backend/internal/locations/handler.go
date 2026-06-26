package locations

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	apphttp "github.com/raven/geoguess/backend/internal/http"
	"github.com/raven/geoguess/backend/internal/middleware"
)

// Handler handles location HTTP endpoints.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler returns a new locations handler.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// RegisterRoutes mounts location routes.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/locations/{locationId}/media", h.GetLocationMedia)
}

// GetLocationMedia handles GET /locations/{locationId}/media.
func (h *Handler) GetLocationMedia(w http.ResponseWriter, r *http.Request) {
	locationID := chi.URLParam(r, "locationId")
	sess := middleware.SessionFromContext(r.Context())

	resp, err := h.service.GetLocationMedia(r.Context(), sess, locationID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidLocationID), errors.Is(err, ErrLocationNotFound):
			apphttp.Error(w, r, h.logger, apphttp.ErrNotFound)
		case errors.Is(err, ErrMediaAccessDenied):
			apphttp.Error(w, r, h.logger, apphttp.ErrForbidden)
		case errors.Is(err, ErrMediaUnavailable):
			apphttp.Error(w, r, h.logger, apphttp.ErrNotFound)
		default:
			apphttp.Error(w, r, h.logger, err)
		}
		return
	}
	apphttp.OK(w, r, resp)
}
