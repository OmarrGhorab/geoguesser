# GORM

## Concepts

GORM is the required ORM. Use it deliberately: model relationships, scopes, transactions, preloads, constraints, and context-aware queries. Do not let GORM models become the whole domain.

## Architecture Decisions

- Use UUID primary keys.
- Use explicit indexes and foreign keys.
- Use `WithContext`.
- Use transactions for multi-write consistency.
- Use soft deletes only when product/legal requirements justify them.

## Trade-offs

GORM improves productivity and migrations from structs, but hidden preloads and hooks can surprise teams. Prefer explicit query methods over magic.

## Anti-patterns

- AutoMigrate in production.
- GORM calls in handlers.
- Hooks with business side effects.
- `Preload` everywhere.
- Saving user-controlled structs directly.

## Common Mistakes

- Missing `gorm:"type:uuid;primaryKey"`.
- Not enabling PostgreSQL UUID generation.
- Ignoring errors from `Create`, `Save`, `Updates`.
- Updating zero values unexpectedly.
- N+1 queries.

## Production Examples

```go
type User struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Email     string         `gorm:"uniqueIndex;not null"`
	CreatedAt time.Time      `gorm:"not null"`
	UpdatedAt time.Time      `gorm:"not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
```

## Go Code Samples

```go
err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
	if err := tx.Create(order).Error; err != nil {
		return fmt.Errorf("create order: %w", err)
	}
	return tx.Create(items).Error
})
```

## Performance Considerations

Use `Select`, `Omit`, `FindInBatches`, indexes, query plans, and explicit preloading. Disable default transactions only after measuring and understanding write consistency impact.

## Security Considerations

Avoid raw SQL with interpolated values. Use DTO-to-model mapping to prevent mass assignment.

## Scalability Considerations

Keep query methods explicit so read replicas, sharding, and caching can be introduced around known access patterns.

