# Security Review

## Verify

- SQL injection prevention.
- CSRF.
- CORS.
- Input validation.
- Password hashing.
- Rate limiting.
- Environment variables.
- Secrets.
- OWASP API Security.
- Security headers.
- HSTS.
- CSP.
- Audit logging.
- Brute-force protection.
- Account lockout.
- Cookie flags: SameSite, Secure, HttpOnly.
- API key authentication where applicable.

## Reject

- Raw SQL with interpolated user input.
- Tokens in localStorage.
- Missing CSRF for cookie-auth mutations.
- Wildcard CORS with credentials.
- Plain secrets in repo or logs.
- Missing rate limit on auth endpoints.

## Common Findings

Critical: SQL query uses `fmt.Sprintf` with request parameter. Impact: SQL injection can leak or modify data. Recommendation: use GORM parameter binding or prepared statements.

