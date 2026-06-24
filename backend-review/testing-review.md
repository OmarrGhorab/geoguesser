# Testing Review

## Verify

- Table-driven tests.
- Unit tests.
- Integration tests.
- Testcontainers.
- Coverage.
- Benchmarks.
- Race detector.
- Auth/authz failure tests.
- Repository tests against PostgreSQL.

## Reject

- Untested business logic.
- No integration tests for SQL changes.
- Tests relying on developer machine services.
- No race detector for concurrency changes.
- Meaningless assertions.

## Common Findings

High: new service has branching business rules and no unit tests. Impact: future refactors can break core behavior silently. Recommendation: add table-driven service tests for success and failure paths.

