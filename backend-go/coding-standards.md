# Coding Standards

## Concepts

Follow Effective Go and Go Proverbs. Use gofmt, goimports, golangci-lint, explicit errors, wrapped errors, `context.Context` first, receiver naming conventions, package naming conventions, small functions, readable code, and consistent structure.

## Architecture Decisions

- Prefer concrete types until interfaces are needed.
- Keep interfaces close to consumers.
- Package by feature.
- Use explicit dependencies.
- Avoid premature abstractions.
- Never panic in application code.
- Never ignore returned errors.

## Trade-offs

Idiomatic Go can look plain. That is a feature. Prefer boring, readable code over clever generic machinery.

## Anti-patterns

- Stuttered names.
- Package globals.
- Ignored errors with `_`.
- `panic` in handlers/services/repositories.
- Unnecessary interfaces.
- Clever channels instead of simple locks.

## Common Mistakes

- Context not first parameter.
- Receiver names like `this` or `self`.
- Overlong functions.
- Magic strings.
- Comments that repeat code.

## Production Examples

```go
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	user, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}
```

## Go Code Samples

```go
// Good receiver naming.
func (r *Repository) Save(ctx context.Context, user *User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}
```

## Performance Considerations

Readable code is easier to profile and optimize. Avoid allocation-heavy abstractions unless measured.

## Security Considerations

No ignored errors, no panics, and explicit dependencies reduce hidden failure modes.

## Scalability Considerations

Consistent standards allow many engineers and agents to work in one codebase safely.

