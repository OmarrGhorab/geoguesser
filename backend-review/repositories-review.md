# Repositories Review

## Verify

- Single responsibility.
- Context propagation.
- Transactions.
- Prepared/parameterized queries.
- Pagination.
- Filtering.
- Sorting.
- Domain error mapping.

## Reject

- Business logic.
- Complex joins without reason.
- N+1 queries.
- Duplicate queries.
- Unbounded list queries.
- Returning `*gorm.DB` to services.

## Common Findings

Medium: repository list method has no limit. Impact: large tables can exhaust memory and degrade PostgreSQL. Recommendation: require bounded pagination.

