# Code Quality Review

## Verify

- Package names.
- Naming.
- Comments.
- Readability.
- Function size.
- Package size.
- Cyclomatic complexity.
- Magic strings.
- Magic numbers.
- Duplication.
- Dead code.
- Unused imports.
- Consistency.
- Go idioms.
- Package cohesion.
- Dependency direction.
- Coupling.
- SOLID where useful.
- Effective Go.
- Go Proverbs.

## Reject

- Fat interfaces.
- Empty interfaces without reason.
- Premature abstractions.
- Cleverness over simplicity.
- Ignored errors.
- Panic usage in application code.

## Common Findings

Medium: interface has nine methods and only one implementation. Impact: broad interface increases coupling and makes tests harder to focus. Recommendation: define small consumer-side interfaces around use cases.

