package friends

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	apphttp "github.com/raven/geoguess/backend/internal/http"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
)

// Handler serves friends HTTP endpoints.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler returns a friends handler.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// RecordRateLimited records a rate-limit rejection before the handler runs.
func (h *Handler) RecordRateLimited(r *http.Request) {
	if h == nil || h.service == nil || h.service.metrics == nil {
		return
	}
	route := "friends"
	path := r.URL.Path
	switch {
	case strings.HasSuffix(path, "/requests") && r.Method == http.MethodPost:
		route = "friends-request"
	case strings.Contains(path, "/accept") || strings.Contains(path, "/decline") ||
		strings.Contains(path, "/block") || r.Method == http.MethodDelete:
		route = "friends-action"
	case r.Method == http.MethodGet:
		route = "friends-read"
	}
	h.service.metrics.RecordRateLimited(route)
	if h.logger != nil {
		h.logger.InfoContext(r.Context(), "friends rate limited", slog.String("path", path))
	}
}

// CreateRequest handles POST /friends/requests.
func (h *Handler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	var body CreateRequestBody
	if err := apphttp.DecodeJSON(w, r, &body); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}
	resp, err := h.service.CreateRequest(r.Context(), *sess, strings.TrimSpace(body.UserID))
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.Created(w, r, resp)
}

// ListIncoming handles GET /friends/requests/incoming.
func (h *Handler) ListIncoming(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	limit, err := parseLimit(r)
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	resp, err := h.service.ListIncoming(r.Context(), *sess, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}

// ListOutgoing handles GET /friends/requests/outgoing.
func (h *Handler) ListOutgoing(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	limit, err := parseLimit(r)
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	resp, err := h.service.ListOutgoing(r.Context(), *sess, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}

// AcceptRequest handles POST /friends/requests/{requestId}/accept.
func (h *Handler) AcceptRequest(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	resp, err := h.service.AcceptRequest(r.Context(), *sess, chi.URLParam(r, "requestId"))
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}

// DeclineRequest handles POST /friends/requests/{requestId}/decline.
func (h *Handler) DeclineRequest(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	if err := h.service.DeclineRequest(r.Context(), *sess, chi.URLParam(r, "requestId")); err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.NoContent(w)
}

// ListFriends handles GET /friends.
func (h *Handler) ListFriends(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	limit, err := parseLimit(r)
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	resp, err := h.service.ListFriends(r.Context(), *sess, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}

// RemoveFriend handles DELETE /friends/{userId}.
func (h *Handler) RemoveFriend(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	if err := h.service.RemoveFriend(r.Context(), *sess, chi.URLParam(r, "userId")); err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.NoContent(w)
}

// BlockUser handles POST /friends/{userId}/block.
func (h *Handler) BlockUser(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	if err := h.service.Block(r.Context(), *sess, chi.URLParam(r, "userId")); err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.NoContent(w)
}

// UnblockUser handles DELETE /friends/{userId}/block.
func (h *Handler) UnblockUser(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	if err := h.service.Unblock(r.Context(), *sess, chi.URLParam(r, "userId")); err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.NoContent(w)
}

// ListBlocked handles GET /friends/blocked.
func (h *Handler) ListBlocked(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	limit, err := parseLimit(r)
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	resp, err := h.service.ListBlocked(r.Context(), *sess, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}

func parseLimit(r *http.Request) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return 0, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil {
		return 0, ErrInvalidLimit
	}
	return limit, nil
}
