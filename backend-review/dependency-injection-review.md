# Dependency Injection Review

## Verify

- Constructor injection.
- Dependency ownership.
- Interface placement.
- Unnecessary interfaces.
- Service wiring.
- Testability.
- Explicit dependencies.

## Reject

- Service locator.
- Global singletons.
- Hidden dependencies through context.
- `init`-time wiring.
- Interfaces in producer packages without a consumer need.

## Common Findings

High: service reads global Redis client. Impact: tests share mutable infrastructure and production wiring is hidden. Recommendation: inject a cache interface through the service constructor.

Low: interface mirrors a concrete type with one implementation and no consumer-side need. Impact: abstraction noise. Recommendation: accept concrete type until tests or multiple implementations require an interface.

