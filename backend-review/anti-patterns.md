# Anti-pattern Detection

Automatically detect:

- God handlers.
- God services.
- God repositories.
- Large packages.
- Fat interfaces.
- Empty interfaces.
- Global state.
- Circular dependencies.
- Duplicate validation.
- SQL in handlers.
- Business logic in repositories.
- Panic usage.
- Ignored errors.
- Large transactions.
- Duplicate models.
- Dead code.
- Hidden dependencies.
- Service locators.
- Missing observability.
- Unbounded goroutines.

Severity depends on production impact. Security/data-loss risks are Critical or High. Maintainability-only issues are usually Medium or Low.

