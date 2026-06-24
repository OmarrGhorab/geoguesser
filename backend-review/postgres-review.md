# PostgreSQL Review

## Verify

- Indexes.
- Foreign keys.
- UUIDs.
- Transactions.
- Isolation.
- Constraints.
- Normalization.
- Pagination.
- Query plans.
- EXPLAIN plans.
- VACUUM considerations.
- ANALYZE.
- Composite indexes.
- Partial indexes.

## Reject

- Missing foreign keys.
- Missing indexes for query patterns.
- Unbounded scans.
- Unsafe isolation assumptions.
- Large migrations without operational plan.

## Common Findings

High: new query filters by `(tenant_id, status, created_at)` but migration only indexes `tenant_id`. Impact: production list endpoint can degrade into slow scans. Recommendation: add composite index matching filter and sort order, then verify with EXPLAIN.

