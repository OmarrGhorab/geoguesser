# Security

## Concepts

Cover SQL injection, XSS, CSRF, CORS, rate limiting, request validation, secure cookies, secret management, environment variables, OWASP API Security Top 10, security headers, HSTS, CSP, audit logging, brute-force protection, account lockout, and API key authentication.

## Architecture Decisions

- Validate input at boundaries.
- Authorize in services.
- Use parameterized GORM queries.
- Store secrets in secret managers or protected env vars.
- Use secure headers and strict CORS.
- Use rate limiting and audit logs for sensitive flows.

## Trade-offs

Security controls add friction. Apply stronger controls to high-risk paths: auth, payments, admin, uploads, webhooks, and PII.

## Anti-patterns

- Tokens in localStorage.
- Wildcard CORS with credentials.
- Raw SQL string concatenation.
- Missing CSRF with cookie auth.
- Secrets in GitHub Actions logs.

## Common Mistakes

- Not limiting request body size.
- Trusting file names.
- Missing account lockout.
- Logging passwords or tokens.
- No API key rotation path.

## Production Examples

Use security headers middleware, CSRF tokens, HTTP-only cookies, password hashing, audit logs, and rate limits on auth endpoints.

## Go Code Samples

```go
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}
```

## Performance Considerations

Rate limiting and password hashing must be tuned. Security checks should be indexed and efficient, never skipped.

## Security Considerations

Apply defense in depth. Use OWASP API Security Top 10 as a review checklist.

## Scalability Considerations

Centralized security middleware and policy helpers keep behavior consistent across growing APIs.

