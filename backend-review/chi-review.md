# Chi Review

## Verify

- Middleware.
- Routing.
- Grouping.
- Versioning.
- Context usage.
- Timeout middleware.
- Recover middleware.
- Logging middleware.
- Request ID middleware.
- Rate limiting middleware.

## Reject

- Duplicated routes.
- Large routing files.
- Business logic in middleware.
- Missing request IDs.
- Missing recover middleware.

## Common Findings

Medium: route group lacks timeout middleware. Impact: slow clients or dependencies can hold resources indefinitely. Recommendation: apply timeout middleware to API route groups.

