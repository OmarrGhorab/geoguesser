# OpenAPI Review

## Verify

- OpenAPI 3.1.
- Request schemas.
- Response schemas.
- Examples.
- Tags.
- Authentication documentation.
- Error documentation.
- Pagination schemas.
- Versioning.
- Status codes.

## Reject

- Undocumented endpoints.
- Missing request/response schemas.
- Missing auth/cookie/JWT documentation.
- Missing error responses.
- Examples that do not match schemas.
- OpenAPI drift from handlers.

## Common Findings

Medium: handler returns 409 conflict but OpenAPI only documents 200 and 500. Impact: generated clients cannot handle real API behavior. Recommendation: add 409 response with shared error schema and example.

