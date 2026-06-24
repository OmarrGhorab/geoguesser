# Checklists

## Concepts

Use checklists as production gates. They enforce architecture, security, observability, tests, and deployment readiness.

## Architecture Decisions

- Check every layer boundary.
- Check every external input.
- Check every mutation for transactions and authorization.
- Check every endpoint against OpenAPI.
- Check every deploy path.

## Trade-offs

Checklists add process but prevent repeated incidents. Keep them short enough to actually use.

## Anti-patterns

- Shipping without tests.
- Shipping without migrations reviewed.
- Shipping without OpenAPI update.
- Shipping without observability.
- Skipping lint due to deadline.

## Common Mistakes

- Forgetting Redis invalidation.
- Forgetting `/ready` dependency checks.
- Forgetting Sentry release config.
- Forgetting rate limits on auth.
- Forgetting integration tests for repositories.

## Production Examples

Pre-merge gate:

- `gofmt`, `goimports`, `golangci-lint`.
- `go test ./...`.
- Race detector for concurrency changes.
- Testcontainers for PostgreSQL/Redis changes.
- OpenAPI updated.
- Migrations reviewed.
- Docker image builds.

## Go Code Samples

```makefile
.PHONY: test lint fmt
fmt:
	gofmt -w .
	goimports -w .
lint:
	golangci-lint run
test:
	go test -race ./...
```

## Performance Considerations

Require benchmarks or profile evidence for performance-sensitive changes.

## Security Considerations

Require authz, validation, secret handling, secure cookies, CSRF, CORS, and rate-limit review.

## Scalability Considerations

Check pagination, idempotency, background jobs, backpressure, and dependency limits before high-traffic launch.

