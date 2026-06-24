# Architecture Review

## Verify

- Modular monolith.
- Feature modules.
- Dependency direction.
- Package boundaries.
- Dependency injection.
- Thin handlers.
- Service layer.
- Repository layer.
- No cyclic imports.
- Proper package organization.
- Package cohesion.
- SOLID where useful.
- Effective Go and Go Proverbs.

## Reject

- God packages.
- Huge handlers.
- Business logic inside handlers.
- Database access inside handlers.
- Global mutable state.
- Circular dependencies.
- Massive files.
- Framework-shaped architecture.

## Common Findings

High: handler imports GORM and performs SQL. Impact: business and persistence concerns are mixed, making testing and auth review harder. Recommendation: move query to repository and use service from handler.

Medium: feature package depends on another feature internals. Impact: coupling grows and refactors become risky. Recommendation: expose a narrow service interface or domain event inside the consumer.

