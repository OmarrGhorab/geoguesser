# Authentication

## Concepts

Use JWT access tokens, refresh tokens, HTTP-only secure cookies, CSRF protection, bcrypt or Argon2 password hashing, and explicit session lifecycle management.

## Architecture Decisions

- Access tokens are short-lived.
- Refresh tokens are rotated and stored securely.
- Cookies are HTTP-only, Secure, SameSite, and path-scoped.
- Passwords use bcrypt or Argon2id.
- Auth services own token issuance and revocation.

## Trade-offs

JWTs reduce session lookup load but complicate revocation. Refresh token rotation improves security but requires durable token state.

## Anti-patterns

- Tokens in localStorage.
- Long-lived access tokens.
- Plain SHA password hashing.
- No CSRF protection with cookie auth.
- Authentication logic in handlers.

## Common Mistakes

- Not rotating refresh tokens.
- Not hashing refresh tokens at rest.
- Missing cookie flags.
- Leaking auth errors that enable enumeration.
- Not rate limiting login.

## Production Examples

Login validates credentials, creates a session, sets HTTP-only cookies, logs audit event, and returns safe user DTO.

## Go Code Samples

```go
func SetAuthCookies(w http.ResponseWriter, access, refresh string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    access,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   900,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refresh,
		Path:     "/v1/auth/refresh",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   60 * 60 * 24 * 30,
	})
}
```

## Performance Considerations

Cache public keys for JWT verification. Keep password hashing cost high enough for security but tested against latency budgets.

## Security Considerations

Use CSRF tokens, secure cookies, token rotation, account lockout, audit logs, and brute-force protection.

## Scalability Considerations

Store refresh token state in PostgreSQL or Redis depending on revocation durability requirements. Design for multi-instance verification.

