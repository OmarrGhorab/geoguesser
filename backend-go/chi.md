# Chi Router

## Concepts

Chi is the required router. It is small, idiomatic, composable, and context-aware. Use it for REST routing, middleware chains, and route grouping.

## Architecture Decisions

- Build one router in `internal/http`.
- Register feature routes through functions.
- Use middleware for cross-cutting transport concerns only.
- Use `chi.URLParam` for path params and validate immediately.

## Trade-offs

Chi gives explicit control with little magic. It does not impose application structure, so the codebase must enforce handler/service/repository boundaries.

## Anti-patterns

- Gin, Fiber, Echo, Beego, or Revel.
- Business logic in route declarations.
- Middleware that performs domain use cases.
- Global router mutation from `init`.

## Common Mistakes

- Missing method-specific routes.
- Not validating path params.
- Registering broad middleware on routes that do not need it.
- Losing request context.
- Returning inconsistent JSON errors.

## Production Examples

```go
func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()
	r.Use(RequestID)
	r.Use(Recoverer(deps.Log))
	r.Use(StructuredLogger(deps.Log))
	r.Get("/live", deps.Health.Live)
	r.Get("/ready", deps.Health.Ready)
	RegisterUserRoutes(r, deps.Users)
	return r
}
```

## Go Code Samples

```go
func RegisterUserRoutes(r chi.Router, h *UserHandler) {
	r.Route("/v1/users", func(r chi.Router) {
		r.With(RequireAuth).Get("/{id}", h.Get)
		r.With(RequireAuth, RequireRole("admin")).Post("/", h.Create)
	})
}
```

## Performance Considerations

Keep middleware chains short. Avoid per-request allocations in middleware. Prefer precompiled regex only when needed.

## Security Considerations

Apply auth middleware to protected route groups. Keep CORS, CSRF, security headers, request size limits, and rate limits near routing.

## Scalability Considerations

Route registration functions keep features modular and make API versioning manageable.

