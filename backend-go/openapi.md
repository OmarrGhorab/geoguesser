# OpenAPI

## Concepts

Use OpenAPI 3.1 as the API contract. It documents REST resources, request/response schemas, errors, examples, auth, versioning, and generated Swagger UI documentation.

## Architecture Decisions

- Store specs in `openapi/openapi.yaml`.
- Use OpenAPI 3.1 JSON Schema semantics.
- Keep schemas aligned with Go DTOs.
- Serve Swagger UI in development or protected internal environments.
- Treat spec changes as part of every API change.

## Trade-offs

Hand-written OpenAPI is precise but can drift. Generated OpenAPI reduces drift but can expose internal DTO decisions. Pick one strategy and enforce it in CI.

## Anti-patterns

- Undocumented endpoints.
- Examples that do not match schemas.
- Error responses missing from spec.
- Breaking changes without versioning.
- Public Swagger UI exposing private APIs.

## Common Mistakes

- Missing pagination schemas.
- Inconsistent error envelopes.
- Not documenting cookies/JWT auth.
- Forgetting 401/403/409 responses.
- Duplicating schemas instead of reusing components.

## Production Examples

```yaml
openapi: 3.1.0
info:
  title: Example API
  version: 1.0.0
paths:
  /v1/users/{id}:
    get:
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        "200":
          description: User found
```

## Go Code Samples

```go
func SwaggerUI(next http.Handler) http.Handler {
	return http.StripPrefix("/docs/", http.FileServer(http.Dir("./openapi/swagger-ui")))
}
```

## Performance Considerations

Do not serve Swagger UI publicly in high-traffic production paths. Cache static documentation assets.

## Security Considerations

Document auth schemes accurately. Avoid exposing internal endpoints, secrets, or privileged examples.

## Scalability Considerations

Stable OpenAPI contracts allow generated clients, contract testing, and cross-team API governance.

