# Error Handling

## Concepts

Go uses explicit errors. Return errors, wrap errors with context, classify domain errors, and map them to consistent HTTP responses at the boundary.

## Architecture Decisions

- Use sentinel or typed domain errors for expected cases.
- Wrap infrastructure errors with `%w`.
- Map errors to HTTP in handlers.
- Log unexpected errors once at the boundary.
- Never panic in application code.

## Trade-offs

Explicit errors add code but keep control flow honest. Typed errors improve mapping but can become excessive.

## Anti-patterns

- Ignoring returned errors.
- Panicking in request paths.
- Returning raw database errors to clients.
- Logging and returning errors at every layer.
- String matching error messages.

## Common Mistakes

- Losing wrapping with `%v`.
- Logging sensitive data.
- Mapping every error to 500.
- Not using `errors.Is` or `errors.As`.
- Returning nil value with nil error accidentally.

## Production Examples

```go
var ErrUserNotFound = errors.New("user not found")

func statusFromError(err error) int {
	switch {
	case errors.Is(err, ErrUserNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
```

## Go Code Samples

```go
if err := repo.Save(ctx, user); err != nil {
	return fmt.Errorf("save user %s: %w", user.ID, err)
}
```

## Performance Considerations

Errors should not be used for normal hot-path branching when a boolean is clearer. Avoid stack-heavy custom errors unless needed.

## Security Considerations

Return safe error codes to clients. Log internal details with redaction and request ID correlation.

## Scalability Considerations

Standard error codes allow clients, alerts, dashboards, and Sentry grouping to scale.

