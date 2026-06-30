package realtime

type MetricsRecorder interface {
	RecordConnectionOpened(roomCode string)
	RecordConnectionClosed(roomCode, reason string)
	RecordEventDelivered(eventType string)
}

type NoopMetrics struct{}

func (NoopMetrics) RecordConnectionOpened(_ string)    {}
func (NoopMetrics) RecordConnectionClosed(_, _ string) {}
func (NoopMetrics) RecordEventDelivered(_ string)      {}
