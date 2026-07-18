package realtime

import (
	"github.com/prometheus/client_golang/prometheus"
)

// MetricsRecorder is the service-facing metrics surface for realtime transport.
// Labels are bounded: no user IDs, tokens, coordinates, or Redis keys.
type MetricsRecorder interface {
	RecordConnectionOpened(roomCode string)
	RecordConnectionClosed(roomCode, reason string)
	RecordEventDelivered(eventType string)

	// Extended casual/ranked metrics (T086 realtime portion).
	ObserveTicketIssued(channelKind, outcome string)
	ObserveTicketConsumed(channelKind, outcome string)
	ObserveSocketOpened(channelKind string)
	ObserveSocketClosed(channelKind, reason string)
	ObserveSlowConsumer(channelKind string)
	ObserveMarkerCommand(outcome string)
	ObserveCommand(command, outcome string)
}

// NoopMetrics is a no-op MetricsRecorder.
type NoopMetrics struct{}

func (NoopMetrics) RecordConnectionOpened(_ string)    {}
func (NoopMetrics) RecordConnectionClosed(_, _ string) {}
func (NoopMetrics) RecordEventDelivered(_ string)      {}
func (NoopMetrics) ObserveTicketIssued(_, _ string)    {}
func (NoopMetrics) ObserveTicketConsumed(_, _ string)  {}
func (NoopMetrics) ObserveSocketOpened(_ string)       {}
func (NoopMetrics) ObserveSocketClosed(_, _ string)    {}
func (NoopMetrics) ObserveSlowConsumer(_ string)       {}
func (NoopMetrics) ObserveMarkerCommand(_ string)      {}
func (NoopMetrics) ObserveCommand(_, _ string)         {}

// Metrics holds Prometheus instruments for realtime transport.
type Metrics struct {
	// Legacy room instruments (kept for existing room handler).
	ConnectionsOpened *prometheus.CounterVec
	ConnectionsClosed *prometheus.CounterVec
	EventsDelivered   *prometheus.CounterVec
	TicketsIssued     *prometheus.CounterVec
	TicketsConsumed   *prometheus.CounterVec
	SocketsOpened     *prometheus.CounterVec
	SocketsClosed     *prometheus.CounterVec
	SlowConsumers     *prometheus.CounterVec
	MarkerCommands    *prometheus.CounterVec
	CommandsTotal     *prometheus.CounterVec
}

// NewMetrics registers realtime metrics against reg.
func NewMetrics(reg prometheus.Registerer) (*Metrics, error) {
	m := &Metrics{
		ConnectionsOpened: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "realtime_connections_opened_total",
			Help: "Realtime connections opened (legacy room label).",
		}, []string{"room_code_bucket"}),
		ConnectionsClosed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "realtime_connections_closed_total",
			Help: "Realtime connections closed (legacy room label).",
		}, []string{"room_code_bucket", "reason"}),
		EventsDelivered: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "realtime_events_delivered_total",
			Help: "Realtime events delivered by type.",
		}, []string{"event_type"}),
		TicketsIssued: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "realtime_tickets_issued_total",
			Help: "Realtime one-time tickets issued by channel kind and outcome.",
		}, []string{"channel_kind", "outcome"}),
		TicketsConsumed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "realtime_tickets_consumed_total",
			Help: "Realtime one-time tickets consumed by channel kind and outcome.",
		}, []string{"channel_kind", "outcome"}),
		SocketsOpened: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "realtime_sockets_opened_total",
			Help: "Party/match WebSocket connections opened by channel kind.",
		}, []string{"channel_kind"}),
		SocketsClosed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "realtime_sockets_closed_total",
			Help: "Party/match WebSocket connections closed by channel kind and reason.",
		}, []string{"channel_kind", "reason"}),
		SlowConsumers: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "realtime_slow_consumers_total",
			Help: "Clients closed for slow outbound consumption by channel kind.",
		}, []string{"channel_kind"}),
		MarkerCommands: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "realtime_marker_commands_total",
			Help: "Marker WebSocket commands by outcome.",
		}, []string{"outcome"}),
		CommandsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "realtime_commands_total",
			Help: "WebSocket commands by type and outcome.",
		}, []string{"command", "outcome"}),
	}

	for _, c := range []prometheus.Collector{
		m.ConnectionsOpened,
		m.ConnectionsClosed,
		m.EventsDelivered,
		m.TicketsIssued,
		m.TicketsConsumed,
		m.SocketsOpened,
		m.SocketsClosed,
		m.SlowConsumers,
		m.MarkerCommands,
		m.CommandsTotal,
	} {
		if err := reg.Register(c); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// RecordConnectionOpened implements MetricsRecorder (legacy room path).
func (m *Metrics) RecordConnectionOpened(roomCode string) {
	if m == nil || m.ConnectionsOpened == nil {
		return
	}
	// Bucket room codes to avoid high-cardinality labels.
	m.ConnectionsOpened.WithLabelValues(bucketRoom(roomCode)).Inc()
	m.ObserveSocketOpened(ChannelKindRoom)
}

// RecordConnectionClosed implements MetricsRecorder (legacy room path).
func (m *Metrics) RecordConnectionClosed(roomCode, reason string) {
	if m == nil || m.ConnectionsClosed == nil {
		return
	}
	m.ConnectionsClosed.WithLabelValues(bucketRoom(roomCode), boundReason(reason)).Inc()
	m.ObserveSocketClosed(ChannelKindRoom, reason)
}

// RecordEventDelivered implements MetricsRecorder.
func (m *Metrics) RecordEventDelivered(eventType string) {
	if m == nil || m.EventsDelivered == nil {
		return
	}
	m.EventsDelivered.WithLabelValues(boundEventType(eventType)).Inc()
}

// ObserveTicketIssued records ticket issuance.
func (m *Metrics) ObserveTicketIssued(channelKind, outcome string) {
	if m == nil || m.TicketsIssued == nil {
		return
	}
	m.TicketsIssued.WithLabelValues(boundChannelKind(channelKind), boundOutcome(outcome)).Inc()
}

// ObserveTicketConsumed records ticket consumption.
func (m *Metrics) ObserveTicketConsumed(channelKind, outcome string) {
	if m == nil || m.TicketsConsumed == nil {
		return
	}
	m.TicketsConsumed.WithLabelValues(boundChannelKind(channelKind), boundOutcome(outcome)).Inc()
}

// ObserveSocketOpened records a party/match socket open.
func (m *Metrics) ObserveSocketOpened(channelKind string) {
	if m == nil || m.SocketsOpened == nil {
		return
	}
	m.SocketsOpened.WithLabelValues(boundChannelKind(channelKind)).Inc()
}

// ObserveSocketClosed records a party/match socket close.
func (m *Metrics) ObserveSocketClosed(channelKind, reason string) {
	if m == nil || m.SocketsClosed == nil {
		return
	}
	m.SocketsClosed.WithLabelValues(boundChannelKind(channelKind), boundReason(reason)).Inc()
}

// ObserveSlowConsumer records a slow-consumer closure.
func (m *Metrics) ObserveSlowConsumer(channelKind string) {
	if m == nil || m.SlowConsumers == nil {
		return
	}
	m.SlowConsumers.WithLabelValues(boundChannelKind(channelKind)).Inc()
}

// ObserveMarkerCommand records a marker command outcome.
func (m *Metrics) ObserveMarkerCommand(outcome string) {
	if m == nil || m.MarkerCommands == nil {
		return
	}
	m.MarkerCommands.WithLabelValues(boundOutcome(outcome)).Inc()
}

// ObserveCommand records a generic WS command outcome.
func (m *Metrics) ObserveCommand(command, outcome string) {
	if m == nil || m.CommandsTotal == nil {
		return
	}
	m.CommandsTotal.WithLabelValues(boundCommand(command), boundOutcome(outcome)).Inc()
}

func bucketRoom(code string) string {
	if code == "" {
		return "unknown"
	}
	return "room"
}

func boundChannelKind(kind string) string {
	switch kind {
	case ChannelKindParty, ChannelKindMatch, ChannelKindRoom:
		return kind
	default:
		return "other"
	}
}

func boundOutcome(outcome string) string {
	switch outcome {
	case "ok", "issued", "consumed", "invalid", "used", "expired", "forbidden",
		"throttled", "locked", "error", "replay", "version_gap", "unauthorized",
		"accepted", "rejected":
		return outcome
	default:
		return "other"
	}
}

func boundReason(reason string) string {
	switch reason {
	case "closed", "slow_consumer", "unauthorized", "expired", "origin",
		"error", "replaced", "normal":
		return reason
	default:
		return "other"
	}
}

func boundEventType(eventType string) string {
	if IsKnownEventType(eventType) || eventType == EventCommandAccepted {
		return eventType
	}
	return "other"
}

func boundCommand(command string) string {
	switch command {
	case CommandPresenceHeartbeat, CommandRoundMarkerSet, CommandRoundViewUpdate, CommandRoundSpectateSelect:
		return command
	default:
		return "other"
	}
}
