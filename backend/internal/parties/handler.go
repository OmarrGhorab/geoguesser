package parties

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	apphttp "github.com/raven/geoguess/backend/internal/http"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
)

// Handler serves party HTTP endpoints.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler returns a parties handler.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{service: service, logger: logger}
}

// RegisterRoutes mounts party routes on the provided router.
// Callers are expected to wrap the router with auth and CSRF middleware.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/parties", h.CreateParty)
	r.Get("/parties/current", h.GetCurrentParty)
	r.Get("/parties/{partyId}", h.GetParty)
	r.Post("/parties/{partyId}/invites", h.CreateInvite)
	r.Get("/party-invites", h.ListInvites)
	r.Post("/party-invites/{inviteId}/accept", h.AcceptInvite)
	r.Post("/party-invites/{inviteId}/decline", h.DeclineInvite)
	r.Put("/parties/{partyId}/readiness/me", h.SetReadiness)
	r.Delete("/parties/{partyId}/members/me", h.LeaveParty)
	r.Delete("/parties/{partyId}/members/{userId}", h.KickMember)
	r.Delete("/parties/{partyId}", h.DisbandParty)
}

// RecordRateLimited records a rate-limit rejection before the handler runs.
func (h *Handler) RecordRateLimited(r *http.Request) {
	if h == nil || h.service == nil || h.service.metrics == nil {
		return
	}
	route := "parties"
	path := r.URL.Path
	switch {
	case strings.HasSuffix(path, "/parties") && r.Method == http.MethodPost:
		route = "parties-create"
	case strings.Contains(path, "/invites") && r.Method == http.MethodPost:
		route = "parties-invite"
	case strings.Contains(path, "/accept") || strings.Contains(path, "/decline"):
		route = "parties-invite-action"
	case strings.Contains(path, "/readiness"):
		route = "parties-readiness"
	case r.Method == http.MethodDelete:
		route = "parties-mutate"
	case r.Method == http.MethodGet:
		route = "parties-read"
	}
	h.service.metrics.RecordRateLimited(route)
	if h.logger != nil {
		h.logger.InfoContext(r.Context(), "parties rate limited", slog.String("path", path))
	}
}

// CreateParty handles POST /parties.
func (h *Handler) CreateParty(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	var body CreatePartyRequest
	if err := apphttp.DecodeJSON(w, r, &body); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}
	resp, err := h.service.Create(r.Context(), *sess, body, r.Header.Get("Idempotency-Key"))
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.Created(w, r, resp)
}

// GetCurrentParty handles GET /parties/current.
func (h *Handler) GetCurrentParty(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	resp, err := h.service.Current(r.Context(), *sess)
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}

// GetParty handles GET /parties/{partyId}.
func (h *Handler) GetParty(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	resp, err := h.service.Get(r.Context(), *sess, chi.URLParam(r, "partyId"))
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}

// CreateInvite handles POST /parties/{partyId}/invites.
func (h *Handler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	var body InviteRequest
	if err := apphttp.DecodeJSON(w, r, &body); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}
	resp, err := h.service.Invite(r.Context(), *sess, chi.URLParam(r, "partyId"), body, r.Header.Get("Idempotency-Key"))
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.Created(w, r, resp)
}

// ListInvites handles GET /party-invites.
func (h *Handler) ListInvites(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	resp, err := h.service.ListInvites(r.Context(), *sess)
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}

// AcceptInvite handles POST /party-invites/{inviteId}/accept.
func (h *Handler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	resp, err := h.service.AcceptInvite(r.Context(), *sess, chi.URLParam(r, "inviteId"), r.Header.Get("Idempotency-Key"))
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}

// DeclineInvite handles POST /party-invites/{inviteId}/decline.
func (h *Handler) DeclineInvite(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	if err := h.service.DeclineInvite(r.Context(), *sess, chi.URLParam(r, "inviteId")); err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.NoContent(w)
}

// SetReadiness handles PUT /parties/{partyId}/readiness/me.
func (h *Handler) SetReadiness(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	var body ReadinessRequest
	if err := apphttp.DecodeJSON(w, r, &body); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}
	resp, err := h.service.SetReadiness(r.Context(), *sess, chi.URLParam(r, "partyId"), body)
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, resp)
}

// LeaveParty handles DELETE /parties/{partyId}/members/me.
func (h *Handler) LeaveParty(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	if err := h.service.Leave(r.Context(), *sess, chi.URLParam(r, "partyId")); err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.NoContent(w)
}

// KickMember handles DELETE /parties/{partyId}/members/{userId}.
func (h *Handler) KickMember(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	if err := h.service.Kick(r.Context(), *sess, chi.URLParam(r, "partyId"), chi.URLParam(r, "userId")); err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.NoContent(w)
}

// DisbandParty handles DELETE /parties/{partyId}.
func (h *Handler) DisbandParty(w http.ResponseWriter, r *http.Request) {
	sess := appmiddleware.SessionFromContext(r.Context())
	if err := h.service.Disband(r.Context(), *sess, chi.URLParam(r, "partyId")); err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.NoContent(w)
}
