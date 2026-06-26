package maps

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	apphttp "github.com/raven/geoguess/backend/internal/http"
)

// Handler handles map HTTP endpoints.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler returns a new maps handler.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// RegisterRoutes mounts map routes.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/maps", h.ListMaps)
	r.Get("/maps/{mapId}", h.GetMap)
}

// ListMaps handles GET /maps.
func (h *Handler) ListMaps(w http.ResponseWriter, r *http.Request) {
	filters := ListFilters{
		AccessTier: r.URL.Query().Get("access_tier"),
		Difficulty: r.URL.Query().Get("difficulty"),
	}
	cursor := r.URL.Query().Get("cursor")

	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			apphttp.Error(w, r, h.logger, apphttp.NewAPIError(http.StatusBadRequest, apphttp.ErrCodeValidationFailed, "invalid limit"))
			return
		}
		limit = n
	}

	resp, err := h.service.ListMaps(r.Context(), filters, cursor, limit)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCursor):
			apphttp.Error(w, r, h.logger, apphttp.NewAPIError(http.StatusBadRequest, apphttp.ErrCodeValidationFailed, "invalid cursor"))
		case errors.Is(err, ErrInvalidFilter):
			apphttp.Error(w, r, h.logger, apphttp.NewAPIError(http.StatusBadRequest, apphttp.ErrCodeValidationFailed, "invalid map filter"))
		default:
			apphttp.Error(w, r, h.logger, err)
		}
		return
	}
	apphttp.OK(w, r, resp)
}

// GetMap handles GET /maps/{mapId}.
func (h *Handler) GetMap(w http.ResponseWriter, r *http.Request) {
	mapID := chi.URLParam(r, "mapId")
	resp, err := h.service.GetMap(r.Context(), mapID)
	if err != nil {
		switch {
		case errors.Is(err, ErrMapNotFound), errors.Is(err, ErrInvalidMapID):
			apphttp.Error(w, r, h.logger, apphttp.ErrNotFound)
		default:
			apphttp.Error(w, r, h.logger, err)
		}
		return
	}
	apphttp.OK(w, r, resp)
}
