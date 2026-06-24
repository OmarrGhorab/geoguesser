# API Design

## Concepts

Design REST APIs with consistent resource names, versioning, status codes, error responses, JSON shapes, idempotency, pagination, PATCH vs PUT semantics, filtering, sorting, and search.

## Architecture Decisions

- Use `/v1` version prefix.
- Use nouns for resources.
- Use standard HTTP methods.
- Use JSON request/response bodies.
- Use consistent error envelopes.
- Use idempotency keys for retryable writes.

## Trade-offs

REST is familiar and cache-friendly. Complex workflows may need action endpoints, but name them deliberately and document them.

## Anti-patterns

- Verb-heavy paths such as `/createUser`.
- Inconsistent envelope shapes.
- Returning 200 for every error.
- PUT used for partial updates.
- No idempotency for payment-like operations.

## Common Mistakes

- Missing 409 for conflicts.
- Returning raw database IDs without stable meaning.
- No pagination on list endpoints.
- Query params that are not documented.
- Breaking clients without versioning.

## Production Examples

```text
GET    /v1/products
POST   /v1/products
GET    /v1/products/{id}
PATCH  /v1/products/{id}
DELETE /v1/products/{id}
```

## Go Code Samples

```go
type ErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}
```

## Performance Considerations

List endpoints must paginate and bound filters. Search endpoints need indexes and timeouts.

## Security Considerations

Do not leak resource existence across tenants. Validate idempotency keys and bind them to actor, route, and payload hash.

## Scalability Considerations

Consistent API conventions make clients, docs, logs, and support scalable across teams.

