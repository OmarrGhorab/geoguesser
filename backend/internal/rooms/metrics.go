package rooms

import "time"

type MetricsRecorder interface {
	ObserveRoomCommand(command, outcome string, duration time.Duration)
	RecordRoomEvent(eventType string)
}

type NoopMetrics struct{}

func (NoopMetrics) ObserveRoomCommand(_, _ string, _ time.Duration) {}
func (NoopMetrics) RecordRoomEvent(_ string)                        {}
