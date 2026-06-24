# Observability

## Concepts

Use OpenTelemetry, Prometheus, Grafana, Sentry, metrics, tracing, structured logging, correlation IDs, request IDs, health endpoints, readiness probes, liveness probes, dashboards, and alerting.

## Architecture Decisions

- Instrument HTTP middleware with traces and metrics.
- Emit Prometheus metrics on `/metrics`.
- Export traces through OpenTelemetry.
- Send panics and unexpected errors to Sentry.
- Use slog JSON logs with request and correlation IDs.
- Build Grafana dashboards for golden signals.

## Trade-offs

More telemetry costs CPU, memory, network, and storage. Keep labels low-cardinality and sample high-volume traces.

## Anti-patterns

- High-cardinality metrics such as user ID labels.
- Logs without request IDs.
- Traces without propagation.
- Sentry for expected validation errors.
- Health endpoints that check nothing useful.

## Common Mistakes

- Missing shutdown flush for telemetry.
- No dashboard ownership.
- Alerts without runbooks.
- Different names for the same field across logs/metrics/traces.
- Logging PII.

## Production Examples

Expose `/metrics`, `/live`, `/ready`, and include request duration histograms, error counters, dependency health, and trace IDs in logs.

## Go Code Samples

```go
var requestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{Name: "http_request_duration_seconds"},
	[]string{"method", "route", "status"},
)
```

## Performance Considerations

Keep metric labels bounded. Batch telemetry export. Avoid tracing every tiny internal function.

## Security Considerations

Do not export secrets or PII in logs, spans, metrics, Sentry breadcrumbs, or dashboard labels.

## Scalability Considerations

Use standardized telemetry names across services. Dashboards and alerts should scale with service count.

