package matchplay

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	apphttp "github.com/raven/geoguess/backend/internal/http"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
)

// RegisterChatRoutes mounts team chat/mute/report routes on the provided router.
// Callers must wrap this group with registered auth, CSRF (for unsafe methods),
// and per-route rate limits. Prefer mounting alongside snapshot routes in routes.go.
//
// Routes:
//
//	POST   /matches/{matchId}/messages
//	GET    /matches/{matchId}/messages
//	GET    /matches/{matchId}/messages/{messageId}/attachment
//	PUT    /matches/{matchId}/mutes/{userId}
//	DELETE /matches/{matchId}/mutes/{userId}
//	POST   /matches/{matchId}/messages/{messageId}/report
func (h *Handler) RegisterChatRoutes(r chi.Router) {
	r.Post("/matches/{matchId}/messages", h.SendMessage)
	r.Get("/matches/{matchId}/messages", h.ListMessages)
	r.Get("/matches/{matchId}/messages/{messageId}/attachment", h.GetAttachment)
	r.Put("/matches/{matchId}/mutes/{userId}", h.MuteTeammate)
	r.Delete("/matches/{matchId}/mutes/{userId}", h.UnmuteTeammate)
	r.Post("/matches/{matchId}/messages/{messageId}/report", h.ReportMessage)
}

// SendMessage handles POST /matches/{matchId}/messages.
// Returns 201 on create and 200 on idempotent client_message_id replay.
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	if h.chatService == nil {
		h.mapError(w, r, ErrUnavailable)
		return
	}
	matchID, err := parseUUIDParam(r, "matchId")
	if err != nil {
		h.mapError(w, r, ErrNotFound)
		return
	}
	var req SendMessageRequest
	if err := apphttp.DecodeJSON(w, r, &req); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}
	result, err := h.chatService.SendMessage(r.Context(), appmiddleware.SessionFromContext(r.Context()), matchID, req)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	if result.Created {
		apphttp.Created(w, r, result.Message)
		return
	}
	apphttp.OK(w, r, result.Message)
}

// ListMessages handles GET /matches/{matchId}/messages?after={sequence}&limit=50.
func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	if h.chatService == nil {
		h.mapError(w, r, ErrUnavailable)
		return
	}
	matchID, err := parseUUIDParam(r, "matchId")
	if err != nil {
		h.mapError(w, r, ErrNotFound)
		return
	}
	afterSeq, err := parseAfterSequence(r)
	if err != nil {
		h.mapError(w, r, ErrInvalidRequest)
		return
	}
	limit, err := parseMessageLimit(r)
	if err != nil {
		h.mapError(w, r, ErrInvalidRequest)
		return
	}
	resp, err := h.chatService.ListMessages(r.Context(), appmiddleware.SessionFromContext(r.Context()), matchID, afterSeq, limit)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

// GetAttachment handles GET /matches/{matchId}/messages/{messageId}/attachment.
func (h *Handler) GetAttachment(w http.ResponseWriter, r *http.Request) {
	if h.chatService == nil {
		h.mapError(w, r, ErrUnavailable)
		return
	}
	matchID, err := parseUUIDParam(r, "matchId")
	if err != nil {
		h.mapError(w, r, ErrNotFound)
		return
	}
	messageID, err := parseUUIDParam(r, "messageId")
	if err != nil {
		h.mapError(w, r, ErrNotFound)
		return
	}
	resp, err := h.chatService.GetAttachmentURL(r.Context(), appmiddleware.SessionFromContext(r.Context()), matchID, messageID)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

// MuteTeammate handles PUT /matches/{matchId}/mutes/{userId}.
func (h *Handler) MuteTeammate(w http.ResponseWriter, r *http.Request) {
	if h.chatService == nil {
		h.mapError(w, r, ErrUnavailable)
		return
	}
	matchID, err := parseUUIDParam(r, "matchId")
	if err != nil {
		h.mapError(w, r, ErrNotFound)
		return
	}
	targetID, err := parseUUIDParam(r, "userId")
	if err != nil {
		h.mapError(w, r, ErrNotFound)
		return
	}
	if err := h.chatService.MuteTeammate(r.Context(), appmiddleware.SessionFromContext(r.Context()), matchID, targetID); err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.NoContent(w)
}

// UnmuteTeammate handles DELETE /matches/{matchId}/mutes/{userId}.
func (h *Handler) UnmuteTeammate(w http.ResponseWriter, r *http.Request) {
	if h.chatService == nil {
		h.mapError(w, r, ErrUnavailable)
		return
	}
	matchID, err := parseUUIDParam(r, "matchId")
	if err != nil {
		h.mapError(w, r, ErrNotFound)
		return
	}
	targetID, err := parseUUIDParam(r, "userId")
	if err != nil {
		h.mapError(w, r, ErrNotFound)
		return
	}
	if err := h.chatService.UnmuteTeammate(r.Context(), appmiddleware.SessionFromContext(r.Context()), matchID, targetID); err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.NoContent(w)
}

// ReportMessage handles POST /matches/{matchId}/messages/{messageId}/report.
// Always responds 202 on success (including idempotent replay).
func (h *Handler) ReportMessage(w http.ResponseWriter, r *http.Request) {
	if h.chatService == nil {
		h.mapError(w, r, ErrUnavailable)
		return
	}
	matchID, err := parseUUIDParam(r, "matchId")
	if err != nil {
		h.mapError(w, r, ErrNotFound)
		return
	}
	messageID, err := parseUUIDParam(r, "messageId")
	if err != nil {
		h.mapError(w, r, ErrNotFound)
		return
	}
	var req ReportMessageRequest
	if err := apphttp.DecodeJSON(w, r, &req); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}
	resp, err := h.chatService.ReportMessage(r.Context(), appmiddleware.SessionFromContext(r.Context()), matchID, messageID, req)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.JSON(w, r, http.StatusAccepted, resp)
}

func parseAfterSequence(r *http.Request) (int64, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("after"))
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 {
		return 0, ErrInvalidRequest
	}
	return n, nil
}

func parseMessageLimit(r *http.Request) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return DefaultMessagePageLimit, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > MaxMessagePageLimit {
		return 0, ErrInvalidRequest
	}
	return n, nil
}

// classifyChatRoute returns a bounded rate-limit route label for chat endpoints.
func classifyChatRoute(r *http.Request) string {
	path := r.URL.Path
	switch {
	case strings.HasSuffix(path, "/report") && r.Method == http.MethodPost:
		return "chat-report"
	case strings.Contains(path, "/mutes/") && (r.Method == http.MethodPut || r.Method == http.MethodDelete):
		return "chat-mute"
	case strings.HasSuffix(path, "/attachment") && r.Method == http.MethodGet:
		return "chat-attachment"
	case strings.HasSuffix(path, "/messages") && r.Method == http.MethodPost:
		return "chat-send"
	case strings.HasSuffix(path, "/messages") && r.Method == http.MethodGet:
		return "chat-list"
	default:
		return "chat"
	}
}
