# Services

## Concepts

Services implement use cases. They enforce business rules, authorization decisions, transactions, idempotency, domain events, cache invalidation, and coordination between repositories.

## Architecture Decisions

- Inject repositories, clocks, loggers, event publishers, and caches.
- Accept `context.Context` as the first parameter.
- Return domain errors.
- Own transactions for multi-step writes.
- Keep services framework-independent.

## Trade-offs

Services add a layer but prevent business rules from leaking into handlers or repositories. Do not create pass-through services with no behavior unless future use cases are imminent.

## Anti-patterns

- Services importing `net/http`.
- Services returning HTTP status codes.
- Services using global dependencies.
- Starting unmanaged goroutines.
- Hiding errors behind booleans.

## Common Mistakes

- Missing transaction around multi-write use case.
- Not checking authorization in service.
- Not invalidating cache after write.
- Ignoring context cancellation.
- Overusing interfaces inside the same package.

## Production Examples

```go
func (s *OrderService) Cancel(ctx context.Context, actor Actor, id uuid.UUID) error {
	order, err := s.orders.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}
	if !actor.CanCancel(order) {
		return ErrForbidden
	}
	return s.orders.UpdateStatus(ctx, id, OrderCanceled)
}
```

## Go Code Samples

```go
type Clock interface {
	Now() time.Time
}

type Service struct {
	repo  Repository
	clock Clock
}
```

## Performance Considerations

Batch repository calls. Avoid N+1 orchestration. Use context timeouts for slow dependencies.

## Security Considerations

Authorize in services. Treat handler checks as defense in depth, not the source of truth.

## Scalability Considerations

Services define future service boundaries. Keep use cases cohesive and side effects explicit.

