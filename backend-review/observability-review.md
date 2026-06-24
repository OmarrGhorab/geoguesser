# Observability Review

## Verify

- OpenTelemetry instrumentation.
- Prometheus metrics.
- Grafana dashboards.
- Sentry initialization.
- Structured logging.
- Request IDs.
- Correlation IDs.
- Health endpoints.
- Readiness probes.
- Liveness probes.
- Metrics endpoint.
- Tracing.
- Trace propagation.
- Custom business metrics.
- Alerts.

## Reject

- Missing instrumentation.
- Missing metrics.
- Sensitive information sent to Sentry.
- No health endpoints.
- No request IDs.
- No trace propagation.
- No custom metrics.
- Missing alerts.
- Missing dashboards.

## Common Findings

High: new checkout endpoint has no custom metrics or tracing span. Impact: failures and latency regressions will be difficult to detect in production. Recommendation: add request duration/error metrics, trace span attributes, and dashboard/alert coverage.

