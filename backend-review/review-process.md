# Review Process

## Review Focus

Review the diff as production code for a Go modular monolith. Look beyond compilation and tests. Verify architecture, dependency direction, Go idioms, security, observability, database behavior, and operations.

## Steps

1. Identify changed packages, handlers, services, repositories, migrations, OpenAPI, tests, Docker, and CI files.
2. Read surrounding code before writing findings.
3. Classify files by layer: HTTP, service, repository, infrastructure, config, tests, deployment.
4. Check critical risks first: auth, data integrity, database queries, secrets, goroutines, migrations, and observability.
5. Load the matching review files.
6. Write findings with severity, explanation, impact, recommendation, and example fix.
7. Score with `scoring.md`.
8. Format with `report-template.md`.

## Strict Rules

- Never approve poor architecture because it passes tests.
- Reject hidden dependencies, global mutable state, and service locators.
- Reject database access from handlers.
- Reject missing validation for external input.
- Reject missing observability for production paths.
- Reject missing tests for business logic.

