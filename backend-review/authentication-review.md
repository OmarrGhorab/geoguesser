# Authentication Review

## Verify

- JWT.
- Refresh tokens.
- HTTP-only cookies.
- Secure cookies.
- SameSite.
- CSRF.
- RBAC.
- Password hashing.
- Session invalidation.
- Token rotation.

## Reject

- Tokens in localStorage.
- Weak secrets.
- Plain passwords.
- Long-lived access tokens.
- Missing refresh token rotation.
- Missing brute-force protection.

## Common Findings

Critical: access token stored in localStorage. Impact: XSS can steal tokens and impersonate users. Recommendation: use HTTP-only Secure SameSite cookies and server-side refresh token rotation.

