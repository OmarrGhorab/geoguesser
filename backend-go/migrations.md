# Migrations

## Concepts

Use Goose for database migrations. Migrations are reviewed code, not generated afterthoughts. Schema changes must be reversible when possible and safe for rolling deployments.

## Architecture Decisions

- Store migrations in `migrations`.
- Use SQL migrations for clarity.
- Do not use GORM AutoMigrate in production.
- Apply migrations in CI/CD or controlled release jobs.
- Include indexes, constraints, and backfill strategy.

## Trade-offs

Reversible migrations improve rollback but some data migrations are irreversible. Mark irreversible downs explicitly and document the operational path.

## Anti-patterns

- AutoMigrate in production.
- Destructive migrations without backfill/rollback plan.
- Adding non-null columns without defaults or staged deploy.
- Long blocking migrations during peak traffic.
- Mixing schema and app deploy without compatibility.

## Common Mistakes

- Missing indexes with new foreign keys.
- Forgetting down migration.
- Renaming columns without dual-read/write phase.
- Not testing migrations against realistic data.
- Running migrations with app superuser credentials.

## Production Examples

```sql
-- +goose Up
CREATE TABLE products (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE products;
```

## Go Code Samples

```go
func RunMigrations(db *sql.DB, dir string) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	return goose.Up(db, dir)
}
```

## Performance Considerations

Use concurrent indexes when needed. Break large backfills into batches. Avoid table rewrites on large tables.

## Security Considerations

Run migrations with scoped credentials. Review data migrations for PII exposure.

## Scalability Considerations

Use expand-contract migrations for zero downtime: add new schema, deploy compatible code, backfill, switch reads, remove old schema later.

