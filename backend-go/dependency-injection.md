# Dependency Injection

## Concepts

Use constructor injection and explicit wiring. Dependencies should be visible in struct fields and constructors. Avoid service locators and global state.

## Architecture Decisions

- Wire concrete dependencies in `cmd/api/main.go`.
- Accept interfaces only where the consumer benefits.
- Keep interfaces small.
- Use fakes/mocks in tests.
- Pass loggers, clocks, repositories, caches, and publishers explicitly.

## Trade-offs

Manual wiring is verbose but transparent. DI containers reduce wiring code but hide dependencies and are rarely needed in Go.

## Anti-patterns

- Service locator.
- Package-level globals.
- `init` wiring.
- Interfaces in producer packages by default.
- Hidden dependencies through context.

## Common Mistakes

- Interface for every struct.
- Constructor doing network calls.
- Cyclic dependencies.
- Tests mutating globals.
- Passing config everywhere instead of specific values.

## Production Examples

```go
func buildApp(ctx context.Context, cfg config.Config) (*App, error) {
	db, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	userRepo := users.NewRepository(db)
	userSvc := users.NewService(userRepo, cfg.Clock)
	userHandler := users.NewHandler(userSvc)
	return NewApp(userHandler), nil
}
```

## Go Code Samples

```go
type UserService struct {
	repo  Repository
	clock Clock
}

func NewService(repo Repository, clock Clock) *UserService {
	return &UserService{repo: repo, clock: clock}
}
```

## Performance Considerations

DI has negligible runtime cost when using concrete wiring. Avoid reflection-based containers in hot services.

## Security Considerations

Explicit dependencies make secrets and privileged clients auditable.

## Scalability Considerations

Manual wiring scales when packages are small and constructors are simple.

