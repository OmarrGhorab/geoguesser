# Validation Review

## Verify

- Request validation.
- Response validation when useful.
- Struct validation.
- Shared validation.
- Path/query/body/cookie/header validation.
- Pagination bounds.

## Reject

- Manual validation everywhere.
- Missing validation.
- Binding directly into GORM models.
- Unknown JSON fields accepted for strict APIs.

## Common Findings

High: handler decodes JSON directly into persistence model. Impact: mass assignment and invalid fields can reach database. Recommendation: decode into request DTO and map allowed fields to domain input.

