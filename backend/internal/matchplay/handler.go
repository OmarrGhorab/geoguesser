package matchplay

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apphttp "github.com/raven/geoguess/backend/internal/http"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
	"github.com/raven/geoguess/backend/internal/session"
)

// ServiceAPI is the handler-facing service surface for snapshot/result/leave.
type ServiceAPI interface {
	GetSnapshot(ctx context.Context, sess *session.Context, matchID uuid.UUID) (*MatchSnapshotResponse, error)
	GetRoundResults(ctx context.Context, sess *session.Context, matchID, roundID uuid.UUID) (*RoundResultsResponse, error)
	GetTerminalResult(ctx context.Context, sess *session.Context, matchID uuid.UUID) (*TerminalResultResponse, error)
	Leave(ctx context.Context, sess *session.Context, matchID uuid.UUID, req LeaveRequest) error
}

// ChatServiceAPI is the handler-facing service surface for team chat/moderation.
type ChatServiceAPI interface {
	SendMessage(ctx context.Context, sess *session.Context, matchID uuid.UUID, req SendMessageRequest) (*SendMessageResult, error)
	ListMessages(ctx context.Context, sess *session.Context, matchID uuid.UUID, afterSeq int64, limit int) (*MessageListResponse, error)
	GetAttachmentURL(ctx context.Context, sess *session.Context, matchID, messageID uuid.UUID) (*AttachmentURLResponse, error)
	MuteTeammate(ctx context.Context, sess *session.Context, matchID, targetUserID uuid.UUID) error
	UnmuteTeammate(ctx context.Context, sess *session.Context, matchID, targetUserID uuid.UUID) error
	ReportMessage(ctx context.Context, sess *session.Context, matchID, messageID uuid.UUID, req ReportMessageRequest) (*ReportMessageResponse, error)
}

// Handler serves match snapshot/result/leave and team chat HTTP endpoints.
type Handler struct {
	service     ServiceAPI
	chatService ChatServiceAPI
	logger      *slog.Logger
	metrics     MetricsRecorder
}

// NewHandler constructs a matchplay handler.
func NewHandler(service ServiceAPI, logger *slog.Logger) *Handler {
	return NewHandlerWithMetrics(service, logger, nil)
}

// NewHandlerWithMetrics constructs a matchplay handler with metrics observers.
func NewHandlerWithMetrics(service ServiceAPI, logger *slog.Logger, metrics MetricsRecorder) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	if metrics == nil {
		metrics = NoopMetrics{}
	}
	h := &Handler{service: service, logger: logger, metrics: metrics}
	// When the concrete service also implements chat, wire it automatically.
	if cs, ok := service.(ChatServiceAPI); ok {
		h.chatService = cs
	}
	return h
}

// WithChatService attaches a chat service when it is not the same object as ServiceAPI.
func (h *Handler) WithChatService(chat ChatServiceAPI) *Handler {
	if h != nil {
		h.chatService = chat
	}
	return h
}

// RecordRateLimited records a rate-limit rejection before the handler runs.
// Call from middleware when a matchplay route is throttled.
func (h *Handler) RecordRateLimited(r *http.Request) {
	if h == nil {
		return
	}
	route := classifyRoute(r)
	h.metrics.ObserveRateLimited(route)
	if h.logger != nil {
		h.logger.InfoContext(r.Context(), "matchplay rate limited", slog.String("path", r.URL.Path))
	}
}

// RecordCSRFRejection records a CSRF rejection for unsafe matchplay routes.
func (h *Handler) RecordCSRFRejection() {
	if h == nil {
		return
	}
	h.metrics.ObserveCSRFRejection()
}

// RecordIdempotency records an idempotency outcome (replay/conflict/miss).
func (h *Handler) RecordIdempotency(outcome string) {
	if h == nil {
		return
	}
	h.metrics.ObserveIdempotency(outcome)
}

// RegisterRoutes mounts match snapshot/result/leave routes on the provided router.
// Callers must wrap this group with registered auth, CSRF (for unsafe methods),
// and per-route rate limits. Do not wire from routes.go here (T044).
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/matches/{matchId}", h.GetSnapshot)
	r.Get("/matches/{matchId}/rounds/{roundId}/results", h.GetRoundResults)
	r.Get("/matches/{matchId}/results", h.GetTerminalResult)
	r.Post("/matches/{matchId}/leave", h.Leave)
}

// GetSnapshot handles GET /matches/{matchId}.
func (h *Handler) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	matchID, err := parseUUIDParam(r, "matchId")
	if err != nil {
		h.mapError(w, r, ErrNotFound)
		return
	}
	resp, err := h.service.GetSnapshot(r.Context(), appmiddleware.SessionFromContext(r.Context()), matchID)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

// GetRoundResults handles GET /matches/{matchId}/rounds/{roundId}/results.
func (h *Handler) GetRoundResults(w http.ResponseWriter, r *http.Request) {
	matchID, err := parseUUIDParam(r, "matchId")
	if err != nil {
		h.mapError(w, r, ErrNotFound)
		return
	}
	roundID, err := parseUUIDParam(r, "roundId")
	if err != nil {
		h.mapError(w, r, ErrNotFound)
		return
	}
	resp, err := h.service.GetRoundResults(r.Context(), appmiddleware.SessionFromContext(r.Context()), matchID, roundID)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

// GetTerminalResult handles GET /matches/{matchId}/results.
// Ranked progression_pending maps to 202 via MapError.
func (h *Handler) GetTerminalResult(w http.ResponseWriter, r *http.Request) {
	matchID, err := parseUUIDParam(r, "matchId")
	if err != nil {
		h.mapError(w, r, ErrNotFound)
		return
	}
	resp, err := h.service.GetTerminalResult(r.Context(), appmiddleware.SessionFromContext(r.Context()), matchID)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

// Leave handles POST /matches/{matchId}/leave.
// Empty body is accepted; Idempotency-Key is optional and observed for metrics.
// Terminal replay is idempotent 204.
func (h *Handler) Leave(w http.ResponseWriter, r *http.Request) {
	matchID, err := parseUUIDParam(r, "matchId")
	if err != nil {
		h.mapError(w, r, ErrNotFound)
		return
	}

	if key := strings.TrimSpace(r.Header.Get("Idempotency-Key")); key != "" {
		h.metrics.ObserveIdempotency("key_present")
	}

	var req LeaveRequest
	// Empty body is valid for leave.
	if r.Body != nil && r.ContentLength != 0 {
		if err := apphttp.DecodeJSON(w, r, &req); err != nil {
			// Treat empty/EOF body as no payload.
			if err != io.EOF {
				// DecodeJSON already maps invalid JSON; check for empty body variants.
				if !isEmptyBodyError(err) {
					apphttp.Error(w, r, h.logger, err)
					return
				}
			}
		}
	}

	if err := h.service.Leave(r.Context(), appmiddleware.SessionFromContext(r.Context()), matchID, req); err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.NoContent(w)
}

func (h *Handler) mapError(w http.ResponseWriter, r *http.Request, err error) {
	apphttp.Error(w, r, h.logger, MapError(err))
}

func parseUUIDParam(r *http.Request, name string) (uuid.UUID, error) {
	raw := strings.TrimSpace(chi.URLParam(r, name))
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func classifyRoute(r *http.Request) string {
	path := r.URL.Path
	switch {
	case strings.HasSuffix(path, "/leave") && r.Method == http.MethodPost:
		return "match-leave"
	case strings.Contains(path, "/rounds/") && strings.HasSuffix(path, "/results"):
		return "match-round-results"
	case strings.HasSuffix(path, "/results"):
		return "match-results"
	case strings.Contains(path, "/messages") || strings.Contains(path, "/mutes/"):
		return classifyChatRoute(r)
	default:
		return "match-snapshot"
	}
}

func isEmptyBodyError(err error) bool {
	if err == nil {
		return false
	}
	// DecodeJSON wraps invalid JSON; empty body often surfaces as invalid JSON with EOF.
	msg := err.Error()
	return strings.Contains(msg, "EOF") || strings.Contains(msg, "empty")
}
