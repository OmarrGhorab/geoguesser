package email

import (
	"context"
	"log/slog"
)

// LoggerSender logs emails via structured logs. It is intended for development
// and testing environments without a real email provider.
type LoggerSender struct {
	logger *slog.Logger
}

// NewLoggerSender returns a development email sender.
func NewLoggerSender(logger *slog.Logger) *LoggerSender {
	return &LoggerSender{logger: logger}
}

// Send logs the email. The body is included so OTPs can be read in dev logs,
// but this implementation must not be used in production.
func (l *LoggerSender) Send(ctx context.Context, msg Message) error {
	l.logger.InfoContext(ctx, "sending email",
		slog.String("to", msg.To),
		slog.String("subject", msg.Subject),
		slog.String("body_preview", truncate(msg.Text, 80)),
	)
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
