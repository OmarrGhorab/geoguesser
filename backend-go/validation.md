# Validation

## Concepts

Validate all external input: JSON bodies, path params, query params, headers, cookies, multipart metadata, JWT claims, and webhook payloads.

## Architecture Decisions

- Decode and validate transport shape in handlers.
- Validate business invariants in services/domain.
- Use explicit structs and validation methods.
- Keep validation errors machine-readable.

## Trade-offs

Strict validation rejects ambiguous requests and protects data quality. It requires OpenAPI and client updates when contracts change.

## Anti-patterns

- Binding request bodies directly into GORM models.
- Ignoring unknown fields.
- Validation only in the client.
- Regex-heavy validation for domain rules.
- Reusing create DTOs for patch without thought.

## Common Mistakes

- Not validating UUIDs.
- Missing pagination bounds.
- Accepting empty strings as valid.
- Not validating file size/content type.
- Returning inconsistent validation errors.

## Production Examples

```go
type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r CreateUserRequest) Validate() error {
	if !strings.Contains(r.Email, "@") {
		return ErrInvalidEmail
	}
	if len(r.Password) < 12 {
		return ErrWeakPassword
	}
	return nil
}
```

## Go Code Samples

```go
func parseLimit(raw string, def int) (int, error) {
	if raw == "" {
		return def, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > 100 {
		return 0, ErrInvalidLimit
	}
	return limit, nil
}
```

## Performance Considerations

Validate before expensive work. Avoid repeated validation in loops once data is trusted.

## Security Considerations

Validation prevents injection, malformed payloads, path traversal, and abusive query sizes. It does not replace authorization.

## Scalability Considerations

Consistent validation errors reduce client support load and allow generated clients to behave predictably.

