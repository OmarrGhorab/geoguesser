# PostgreSQL

## Concepts

PostgreSQL is the only default database. Use UUID primary keys, indexes, foreign keys, constraints, transactions, pagination, and optimized queries.

## Architecture Decisions

- Use UUID primary keys.
- Model foreign keys explicitly.
- Add indexes for filter, sort, join, and uniqueness patterns.
- Use transactions for consistency.
- Use connection pooling.
- Use migrations as the source of schema truth.

## Trade-offs

Constraints push correctness into the database. They require migration discipline but prevent application bugs from corrupting data.

## Anti-patterns

- MySQL or MongoDB unless explicitly requested.
- Missing foreign keys.
- Unbounded queries.
- Soft deletes by default.
- Storing JSONB for relational data without a reason.

## Common Mistakes

- Missing composite indexes for common filters.
- Offset pagination on very large tables without limits.
- Not setting statement timeouts.
- Overusing transactions around read-only work.
- Ignoring lock behavior.

## Production Examples

```sql
CREATE TABLE users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  email text NOT NULL UNIQUE,
  password_hash text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_created_at ON users (created_at DESC);
```

## Go Code Samples

```go
sqlDB, err := db.DB()
if err != nil {
	return err
}
sqlDB.SetMaxOpenConns(25)
sqlDB.SetMaxIdleConns(25)
sqlDB.SetConnMaxLifetime(30 * time.Minute)
```

## Performance Considerations

Use `EXPLAIN ANALYZE`, indexes, bounded pagination, connection pool tuning, prepared statements where useful, and query timeouts.

## Security Considerations

Use least-privilege database users, TLS where required, parameterized queries, and secrets from validated config.

## Scalability Considerations

Design query patterns for read replicas, partitioning, archival, and background backfills.

