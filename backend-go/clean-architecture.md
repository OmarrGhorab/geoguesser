# Clean Architecture

## Concepts

Clean Architecture keeps business rules independent from delivery and storage. In Go, apply it pragmatically: handlers call services, services call repositories, repositories call GORM/PostgreSQL.

## Architecture Decisions

- Define interfaces at the consumer boundary.
- Keep DTOs separate from persistence models when API shape differs.
- Keep transactions in services for multi-repository use cases.
- Keep infrastructure in `internal/platform`.

## Trade-offs

Strict layering improves testability but can overfit simple CRUD. Collapse layers only for trivial internal tools; preserve boundaries for auth, payments, multi-tenant data, and core business logic.

## Anti-patterns

- Anemic services that only forward to repositories.
- Repositories returning `*gorm.DB`.
- Services importing `net/http`.
- Handlers importing `gorm`.
- Overly abstract "usecase" packages with no domain language.

## Common Mistakes

- Putting interfaces in producer packages.
- Sharing one generic repository for all models.
- Treating GORM models as API responses.
- Duplicating validation in every layer.
- Starting with microservices before domain boundaries are known.

## Production Examples

`CreateOrder` should validate, authorize, start a transaction, create records, publish events, and return a DTO without exposing GORM.

## Go Code Samples

```go
type OrderRepository interface {
	Create(ctx context.Context, tx *gorm.DB, order *Order) error
}

func (s *OrderService) Create(ctx context.Context, input CreateOrderInput) (*OrderDTO, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validate order: %w", err)
	}
	return s.repo.CreateWithTransaction(ctx, input)
}
```

## Performance Considerations

Layers must not add needless marshaling. Pass typed structs and contexts; do not convert to maps or JSON between internal layers.

## Security Considerations

Authorize in services, not handlers only. A service may be called by HTTP, background jobs, or tests; authorization belongs with the use case.

## Scalability Considerations

Clean boundaries support moving a feature to another process later, but design for a modular monolith first.

