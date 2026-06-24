# Repositories

## Concepts

Repositories isolate persistence. They translate between domain/data models and GORM/PostgreSQL queries. They do not enforce business use cases.

## Architecture Decisions

- Use repository interfaces near services.
- Use GORM implementation structs in infrastructure or feature package.
- Accept context in every method.
- Use transactions explicitly for multi-step writes.
- Return domain errors such as `ErrNotFound`.

## Trade-offs

Repositories hide ORM details and improve tests. Too-generic repositories obscure query intent and hurt performance.

## Anti-patterns

- Generic CRUD repository for every model.
- Returning `*gorm.DB` to services.
- Business decisions in SQL methods.
- Ignoring `RowsAffected`.
- Unbounded list queries.

## Common Mistakes

- Not using `WithContext`.
- N+1 queries from lazy associations.
- Missing indexes for query patterns.
- Swallowing unique constraint errors.
- Not distinguishing not found from internal errors.

## Production Examples

```go
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &user, nil
}
```

## Go Code Samples

```go
type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}
```

## Performance Considerations

Select only required columns, paginate lists, preload intentionally, and inspect query plans for slow paths.

## Security Considerations

Use parameterized queries through GORM. Never concatenate user input into SQL.

## Scalability Considerations

Repository methods should match use-case query patterns so indexes and cache strategies can evolve.

