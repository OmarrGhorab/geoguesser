# Project Structure

## Concepts

Structure Go services by executable, internal feature packages, platform adapters, migrations, OpenAPI, and deployment files. Prefer package-by-feature over package-by-layer when features are substantial.

## Architecture Decisions

- Use `cmd/api/main.go` for wiring.
- Use `internal/<feature>` for domain features.
- Use `internal/platform/<adapter>` for PostgreSQL, Redis, telemetry, Sentry, email, and object storage.
- Keep migrations under `migrations`.
- Keep OpenAPI specs under `openapi`.

## Trade-offs

Package-by-feature improves ownership. Shared platform packages prevent duplicated clients. Avoid excessive nesting; Go package names should stay short.

## Anti-patterns

- `controllers`, `models`, `services` top-level folders with cross-feature coupling.
- `pkg` for application internals.
- Giant `common` packages.
- Circular dependencies.
- Generated code mixed with hand-written domain logic.

## Common Mistakes

- Using plural or stuttered package names.
- Exporting everything.
- Creating package-level config.
- Putting tests far from code.
- Naming packages after frameworks.

## Production Examples

```text
cmd/api/main.go
internal/config/config.go
internal/http/router.go
internal/users/
internal/orders/
internal/platform/postgres/
internal/platform/redis/
internal/platform/otel/
migrations/
openapi/openapi.yaml
```

## Go Code Samples

```go
func main() {
	ctx := context.Background()
	cfg := config.MustLoad()
	app, err := buildApp(ctx, cfg)
	if err != nil {
		panic(err)
	}
	app.Run(ctx)
}
```

Use panic only in `main` for startup failure; never panic in request path.

## Performance Considerations

Package boundaries should not force unnecessary allocations or interface dispatch in hot code. Keep generated OpenAPI types separate if they are not domain-friendly.

## Security Considerations

Use `internal` to prevent accidental external imports. Keep secrets and config parsing in one audited package.

## Scalability Considerations

Feature packages allow separate owners and future extraction. Stable interfaces around repositories and platform clients reduce churn.

