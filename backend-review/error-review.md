# Error Handling Review

## Verify

- Wrapped errors.
- `errors.Is`.
- `errors.As`.
- Custom error types where useful.
- Consistent API responses.
- No panic in business logic.
- No ignored errors.
- Safe client messages.

## Reject

- `_ = err`.
- `panic` in handlers/services/repositories.
- String matching errors.
- Raw DB errors returned to clients.
- Logging same error in every layer.

## Common Findings

High: repository ignores `RowsAffected`. Impact: update can silently report success for missing record. Recommendation: check rows affected and return domain not-found error.

