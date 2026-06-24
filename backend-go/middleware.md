# Middleware

## Concepts

Middleware handles transport cross-cutting concerns: request IDs, correlation IDs, logging, panic recovery, auth extraction, CSRF, CORS, rate limiting, compression, metrics, and tracing.

## Architecture Decisions

- Middleware must be small and composable.
- Store request-scoped values with typed context keys.
- Avoid domain decisions except authn/authz checks needed for route gating.
- Emit metrics and traces around requests.

## Trade-offs

Middleware centralizes repeated HTTP concerns but can hide control flow. Keep it predictable and documented.

## Anti-patterns

- Database queries in generic middleware.
- Context values for dependencies.
- Swallowing errors.
- Recover middleware that hides panics without logging.
- CORS wildcard with credentials.

## Common Mistakes

- Missing request ID propagation.
- Not wrapping `ResponseWriter` for status metrics.
- Creating high-cardinality metric labels.
- Applying auth middleware to health endpoints.
- Logging request bodies with secrets.

## Production Examples

```go
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

## Go Code Samples

```go
type requestIDKey struct{}

func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}
```

## Performance Considerations

Avoid allocations in hot middleware. Keep metric labels low-cardinality. Do not parse JWTs multiple times in one request.

## Security Considerations

Apply security headers, CSRF protection, request limits, auth, CORS, and rate limits consistently.

## Scalability Considerations

Middleware should support distributed tracing and correlation IDs across services.

