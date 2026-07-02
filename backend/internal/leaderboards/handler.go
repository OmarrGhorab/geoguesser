package leaderboards

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	apphttp "github.com/raven/geoguess/backend/internal/http"
)

type ServiceAPI interface {
	GetGlobal(context.Context, int, string) (*Response, error)
	GetDaily(context.Context, int, string, string) (*Response, error)
	GetMap(context.Context, string, int, string) (*Response, error)
}

type Handler struct {
	service ServiceAPI
	logger  *slog.Logger
}

func NewHandler(service ServiceAPI, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/leaderboards/global", h.GetGlobal)
	r.Get("/leaderboards/daily", h.GetDaily)
	r.Get("/leaderboards/maps/{mapId}", h.GetMap)
}

func (h *Handler) GetGlobal(w http.ResponseWriter, r *http.Request) {
	limit, err := parseLimitParam(r)
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	resp, err := h.service.GetGlobal(r.Context(), limit, r.URL.Query().Get("cursor"))
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) GetDaily(w http.ResponseWriter, r *http.Request) {
	limit, err := parseLimitParam(r)
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	resp, err := h.service.GetDaily(r.Context(), limit, r.URL.Query().Get("cursor"), r.URL.Query().Get("date"))
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) GetMap(w http.ResponseWriter, r *http.Request) {
	limit, err := parseLimitParam(r)
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	resp, err := h.service.GetMap(r.Context(), chi.URLParam(r, "mapId"), limit, r.URL.Query().Get("cursor"))
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}

func parseLimitParam(r *http.Request) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return 0, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > maxLimit {
		return 0, ErrInvalidLimit
	}
	return limit, nil
}
