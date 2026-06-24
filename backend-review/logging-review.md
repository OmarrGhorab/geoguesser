# Logging Review

## Verify

- `log/slog`.
- Structured logging.
- JSON logging.
- Log levels.
- Context-aware logging.
- Correlation IDs.
- Request IDs.
- No PII.
- No secrets.
- Proper error logging.
- Consistent fields.

## Reject

- `fmt.Println`.
- Sensitive data in logs.
- Missing request ID.
- Logging tokens/cookies/passwords.
- Unstructured production logs.

## Common Findings

Medium: auth failure logs include raw email and request body. Impact: logs can contain PII and secrets. Recommendation: log stable user/account IDs when available and redact request bodies.

