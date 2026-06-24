# Handlers Review

## Verify

- Thin handlers.
- Request decoding.
- Request validation.
- Context propagation.
- Service calls only.
- Consistent JSON responses.
- Proper status codes.
- No database access.

## Reject

- Business logic inside handlers.
- SQL/GORM inside handlers.
- Huge handlers.
- Ignored decode errors.
- Unbounded request bodies.
- Internal errors returned to clients.

## Common Findings

High: handler directly calls `db.Create`. Impact: bypasses service validation, authorization, transactions, and tests. Recommendation: inject service and move persistence behind repository.

