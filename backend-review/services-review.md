# Services Review

## Verify

- Business rules only inside services.
- Services do not know HTTP.
- Services are testable.
- Authorization happens for sensitive use cases.
- Transactions wrap multi-write operations.
- Context is first parameter.

## Reject

- SQL inside services.
- HTTP response concerns inside services.
- Large service files.
- Hidden globals.
- Missing transaction where consistency is required.

## Common Findings

High: service deletes account data in multiple repositories without transaction. Impact: partial failure can corrupt account state. Recommendation: start transaction at service layer and pass transactional repository handle.

