package realtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	apphttp "github.com/raven/geoguess/backend/internal/http"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
)

// TicketIssueRequest is POST /realtime/tickets body.
type TicketIssueRequest struct {
	ChannelKind string `json:"channel_kind"`
	ChannelID   string `json:"channel_id"`
}

// TicketIssueResponse is the 201 body for a one-time realtime ticket.
type TicketIssueResponse struct {
	Ticket       string    `json:"ticket"`
	ExpiresAt    time.Time `json:"expires_at"`
	WebsocketURL string    `json:"websocket_url"`
}

// TicketHandler issues authenticated one-time WebSocket tickets.
type TicketHandler struct {
	tickets    TicketStore
	authorizer ChannelAuthorizer
	ttl        time.Duration
	publicWS   string // optional absolute base like wss://api.example; empty -> relative
	logger     *slog.Logger
	metrics    MetricsRecorder
	enabled    bool
}

// TicketHandlerConfig configures ticket issuance.
type TicketHandlerConfig struct {
	TTL             time.Duration
	PublicWSBaseURL string // e.g. wss://api.example.com (no path)
	FeatureEnabled  bool
}

// NewTicketHandler constructs a ticket issuance handler.
func NewTicketHandler(
	tickets TicketStore,
	authorizer ChannelAuthorizer,
	cfg TicketHandlerConfig,
	logger *slog.Logger,
	metrics MetricsRecorder,
) *TicketHandler {
	if logger == nil {
		logger = slog.Default()
	}
	if metrics == nil {
		metrics = NoopMetrics{}
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 30 * time.Second
	}
	return &TicketHandler{
		tickets:    tickets,
		authorizer: authorizer,
		ttl:        cfg.TTL,
		publicWS:   strings.TrimRight(strings.TrimSpace(cfg.PublicWSBaseURL), "/"),
		logger:     logger,
		metrics:    metrics,
		enabled:    cfg.FeatureEnabled,
	}
}

// Issue handles POST /api/v1/realtime/tickets.
func (h *TicketHandler) Issue(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.tickets == nil {
		apphttp.Error(w, r, h.logger, apphttp.NewAPIError(http.StatusServiceUnavailable, CodeFeatureDisabled, "Realtime tickets are temporarily unavailable."))
		return
	}
	if !h.enabled {
		h.metrics.ObserveTicketIssued("unknown", "forbidden")
		apphttp.Error(w, r, h.logger, apphttp.NewAPIError(http.StatusServiceUnavailable, CodeFeatureDisabled, "Realtime tickets are temporarily unavailable."))
		return
	}

	sess := appmiddleware.SessionFromContext(r.Context())
	if sess == nil || !sess.IsRegistered() || sess.UserID == nil {
		h.metrics.ObserveTicketIssued("unknown", "unauthorized")
		apphttp.Error(w, r, h.logger, apphttp.ErrUnauthorized)
		return
	}
	userID, err := uuid.Parse(strings.TrimSpace(*sess.UserID))
	if err != nil {
		h.metrics.ObserveTicketIssued("unknown", "unauthorized")
		apphttp.Error(w, r, h.logger, apphttp.ErrUnauthorized)
		return
	}

	var req TicketIssueRequest
	if err := apphttp.DecodeJSON(w, r, &req); err != nil {
		h.metrics.ObserveTicketIssued("unknown", "invalid")
		return
	}
	kind := strings.TrimSpace(req.ChannelKind)
	channelID := strings.TrimSpace(req.ChannelID)
	if kind != ChannelKindParty && kind != ChannelKindMatch {
		h.metrics.ObserveTicketIssued(kind, "invalid")
		apphttp.Error(w, r, h.logger, apphttp.NewAPIError(http.StatusBadRequest, apphttp.ErrCodeValidationFailed, "channel_kind must be party or match"))
		return
	}
	if _, err := uuid.Parse(channelID); err != nil {
		h.metrics.ObserveTicketIssued(kind, "invalid")
		apphttp.Error(w, r, h.logger, apphttp.NewAPIError(http.StatusBadRequest, apphttp.ErrCodeValidationFailed, "channel_id must be a UUID"))
		return
	}

	// Membership authorization — missing/unauthorized both map to privacy-safe not_found.
	if h.authorizer != nil {
		if _, authErr := h.authorizer.Authorize(r.Context(), userID, kind, channelID); authErr != nil {
			h.metrics.ObserveTicketIssued(kind, "forbidden")
			apphttp.Error(w, r, h.logger, mapTicketAuthError(authErr))
			return
		}
	}

	token, expiresAt, err := h.tickets.Issue(r.Context(), userID, kind, channelID, h.ttl)
	if err != nil {
		h.metrics.ObserveTicketIssued(kind, "error")
		h.logger.WarnContext(r.Context(), "realtime ticket issue failed",
			slog.String("channel_kind", kind),
			// Never log the opaque ticket value.
			slog.Any("error", err),
		)
		apphttp.Error(w, r, h.logger, apphttp.NewAPIError(http.StatusServiceUnavailable, CodeFeatureDisabled, "Realtime tickets are temporarily unavailable."))
		return
	}
	if expiresAt.IsZero() {
		expiresAt = time.Now().UTC().Add(h.ttl)
	}

	wsURL := h.websocketURL(r, kind, channelID)
	h.metrics.ObserveTicketIssued(kind, "issued")
	apphttp.Created(w, r, TicketIssueResponse{
		Ticket:       token,
		ExpiresAt:    expiresAt.UTC(),
		WebsocketURL: wsURL,
	})
}

func (h *TicketHandler) websocketURL(r *http.Request, kind, channelID string) string {
	path := "/realtime/matches/" + url.PathEscape(channelID)
	if kind == ChannelKindParty {
		path = "/realtime/parties/" + url.PathEscape(channelID)
	}
	if h.publicWS != "" {
		return h.publicWS + path
	}
	// Derive from request host when absolute base is not configured.
	scheme := "wss"
	if r.TLS == nil {
		// Honor reverse-proxy headers lightly.
		if proto := r.Header.Get("X-Forwarded-Proto"); strings.EqualFold(proto, "https") {
			scheme = "wss"
		} else {
			scheme = "ws"
		}
	}
	host := r.Host
	if host == "" {
		return path
	}
	return fmt.Sprintf("%s://%s%s", scheme, host, path)
}

func mapTicketAuthError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotParticipant) || errors.Is(err, ErrAuthRequired) {
		return apphttp.ErrNotFound
	}
	// Privacy-safe default for membership failures.
	var api *apphttp.APIError
	if errors.As(err, &api) {
		return api
	}
	return apphttp.ErrNotFound
}

// DecodeTicketSubprotocols extracts geoguess.v1 and the opaque ticket from
// Sec-WebSocket-Protocol values. Tickets must never be logged.
func DecodeTicketSubprotocols(protocols []string) (hasV1 bool, ticket string) {
	for _, p := range protocols {
		p = strings.TrimSpace(p)
		if p == SubprotocolV1 {
			hasV1 = true
			continue
		}
		if strings.HasPrefix(p, TicketSubprotocolPrefix) {
			ticket = strings.TrimPrefix(p, TicketSubprotocolPrefix)
		}
	}
	return hasV1, ticket
}

// ParseSecWebSocketProtocol splits the header into protocol tokens.
func ParseSecWebSocketProtocol(header string) []string {
	if strings.TrimSpace(header) == "" {
		return nil
	}
	parts := strings.Split(header, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// StrictTicketIssueRequest rejects unknown JSON fields via DecodeJSON when configured.
// Used by tests to ensure contract shape.
func StrictTicketIssueRequest(raw []byte) (TicketIssueRequest, error) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var req TicketIssueRequest
	if err := dec.Decode(&req); err != nil {
		return TicketIssueRequest{}, err
	}
	return req, nil
}
