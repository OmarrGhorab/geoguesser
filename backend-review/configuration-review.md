# Configuration Review

## Verify

- Environment variables.
- Validation.
- Defaults.
- Secret management.
- Production configuration.
- Configuration loading.
- Configuration precedence.
- Duration/URL parsing.

## Reject

- Hardcoded secrets.
- Magic configuration values.
- Reading env vars throughout code.
- Defaults for required production secrets.
- Logging config with secrets.

## Common Findings

High: JWT secret has a fallback default. Impact: production can accidentally run with known signing secret. Recommendation: fail startup if required secret is missing.

