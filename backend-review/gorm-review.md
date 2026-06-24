# GORM Review

## Verify

- Associations.
- Preload usage.
- Indexes.
- Constraints.
- Transactions.
- Scopes.
- Model design.
- Soft deletes.
- Query optimization.
- `WithContext`.

## Reject

- Raw SQL without reason.
- `SELECT *` on large or sensitive models.
- N+1 queries.
- Missing indexes.
- AutoMigrate in production.
- GORM calls in handlers.

## Common Findings

High: loop calls `Preload`/`First` once per item. Impact: N+1 queries cause latency and database load growth. Recommendation: batch load associations with explicit preload or join query.

