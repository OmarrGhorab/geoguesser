# API Design Review

## Verify

- REST conventions.
- Versioning.
- HTTP methods.
- Status codes.
- Consistent JSON responses.
- Pagination.
- Filtering.
- Sorting.
- Idempotency.
- PATCH vs PUT semantics.
- OpenAPI documentation completeness.
- Resource naming.
- Error response shape.

## Reject

- Verb-heavy paths such as `/createUser`.
- Returning 200 for errors.
- Missing pagination on list endpoints.
- Unbounded filters or sorting.
- Non-idempotent retryable operations without idempotency keys.
- Breaking response shape without versioning.

## Common Findings

High: payment-like POST endpoint lacks idempotency. Impact: client retries can create duplicate charges/orders. Recommendation: require an idempotency key bound to actor, route, and request hash.

Medium: list endpoint supports `sort` directly as SQL column. Impact: SQL injection or unstable sorting. Recommendation: allowlist sort fields.

