# Architecture

## Concepts

Design Go backends as small, explicit systems. HTTP, business logic, persistence, infrastructure, and observability are separate responsibilities. Use Clean Architecture and DDD where they reduce coupling, not as ceremony.

## Architecture Decisions

- Use Chi for HTTP routing.
- Use services for use cases.
- Use repositories for PostgreSQL/GORM access.
- Use Redis behind explicit cache/session/rate-limit abstractions.
- Wire dependencies in `cmd/api/main.go`.
- Keep domain concepts in `internal/<feature>`.

## Trade-offs

Layering adds files but makes testing and change isolation better. DDD vocabulary helps complex domains but is unnecessary for CRUD-only areas. Interfaces improve tests when placed near consumers; premature interfaces create noise.

## Anti-patterns

- Database calls in handlers.
- Global DB, Redis, logger, or config variables.
- Framework-shaped architecture.
- God packages such as `internal/utils`.
- Domain logic in middleware.

## Common Mistakes

- Creating interfaces for every struct.
- Hiding dependencies in package globals.
- Mixing DTOs, GORM models, and domain objects.
- Letting transactions leak through context values.
- Ignoring cancellation.

## Production Examples

Use a feature package:

```text
internal/orders/
  handler.go
  service.go
  repository.go
  model.go
  dto.go
  errors.go
```

## Go Code Samples

```go
type OrderService struct {
	repo OrderRepository
	log  *slog.Logger
}

func NewOrderService(repo OrderRepository, log *slog.Logger) *OrderService {
	return &OrderService{repo: repo, log: log}
}
```

## Performance Considerations

Keep hot paths simple. Avoid unnecessary allocations in request parsing. Use connection pools, cache only measured bottlenecks, and avoid reflection-heavy generic abstractions in critical loops.

## Security Considerations

Boundaries make security review possible. Handlers validate transport input, services authorize use cases, repositories parameterize database access, and platform packages isolate secrets.

## Scalability Considerations

Small feature packages let teams work independently. Explicit dependencies make services easier to split later if a monolith outgrows a single process.

