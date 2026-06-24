# Security Review

## Review Scope

Review XSS, CSRF, cookie usage, Server Actions, Route Handlers, environment variables, exposed secrets, input validation, sanitization, authorization, and storage.

## Blockers

- Authentication tokens in `localStorage`, sessionStorage, Zustand, or client props.
- Server Action or Route Handler mutation without authorization.
- Secrets exposed through `NEXT_PUBLIC_` or client imports.
- Untrusted HTML rendered without sanitization.
- Webhook without signature verification.
- Cross-tenant access possible through user-controlled IDs.

## What To Check

- Server Actions verify authentication and authorization.
- Route Handlers validate method, auth, body, params, and signatures.
- Cookies are HTTP-only, secure, same-site where appropriate.
- CSRF risk is considered for cookie-based mutations.
- Env vars are validated and kept server-only.
- DTOs prevent sensitive fields from reaching clients.
- Error messages do not leak sensitive details.
- Cache scope does not expose private data.

## Severity Guidance

- `Critical`: token exposure, auth bypass, cross-tenant data leak, stored XSS.
- `High`: missing authz on sensitive action, webhook trust gap, secret exposure risk.
- `Medium`: weak validation, unsafe errors, missing rate limits.
- `Low`: hardening suggestions.

## Example Finding

```text
Critical - features/auth/login.tsx:44
The login flow stores the JWT in localStorage.
Why it matters: Any XSS can read the token and impersonate the user; it also bypasses HTTP-only cookie protections.
Recommended fix: issue an HTTP-only secure same-site session cookie from a Server Action or auth provider.
```

