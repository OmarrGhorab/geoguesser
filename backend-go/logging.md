# Logging

## Concepts

Use `log/slog` for JSON structured logs. Include log levels, request IDs, correlation IDs, context logging, and sensitive data redaction.

## Architecture Decisions

- Use JSON logs in production.
- Inject `*slog.Logger`.
- Add request and correlation IDs to context-aware loggers.
- Log at boundaries and important state transitions.
- Redact secrets and PII.

## Trade-offs

Structured logs are slightly more verbose to write but far easier to query, alert, and correlate.

## Anti-patterns

- `fmt.Println` in application code.
- Logging request bodies.
- Logging tokens, passwords, or cookies.
- High-cardinality fields for every log.
- Logging the same error at every layer.

## Common Mistakes

- Missing request ID.
- Using error level for expected 4xx responses.
- No correlation with traces.
- Inconsistent field names.
- Losing context in goroutines.

## Production Examples

```go
logger.InfoContext(ctx, "user created",
	slog.String("user_id", user.ID.String()),
	slog.String("request_id", RequestIDFromContext(ctx)),
)
```

## Go Code Samples

```go
func NewLogger(env string) *slog.Logger {
	level := slog.LevelInfo
	if env == "development" {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
```

## Performance Considerations

Avoid expensive log field computation unless the level is enabled. Keep logs sampled for very hot paths.

## Security Considerations

Centralize redaction. Treat logs as sensitive data and apply retention policies.

## Scalability Considerations

Stable field names make logs queryable across services and dashboards.

