# Handlers

## Concepts

Handlers are transport adapters. They parse HTTP input, call services, map errors to status codes, and write responses. They do not contain business logic or database queries.

## Architecture Decisions

- Inject services into handlers.
- Decode JSON with limits.
- Validate transport shape before service call.
- Return consistent JSON envelopes.
- Use request context for every call.

## Trade-offs

Thin handlers keep use cases reusable. Some validation belongs in handlers when it is transport-specific; business validation belongs in services/domain.

## Anti-patterns

- Calling GORM from handlers.
- Starting goroutines from handlers without lifecycle control.
- Panicking on bad input.
- Writing ad hoc JSON errors.
- Ignoring body decode errors.

## Common Mistakes

- Forgetting `defer r.Body.Close()` when appropriate.
- Not limiting request body size.
- Treating query params as trusted.
- Leaking internal error text.
- Double-writing responses.

## Production Examples

```go
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	user, err := h.users.Create(r.Context(), req.ToInput())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}
```

## Go Code Samples

```go
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
```

## Performance Considerations

Avoid decoding into maps. Reuse response helpers. Keep response payloads bounded and paginate lists.

## Security Considerations

Limit body size, reject unknown fields for strict APIs, validate params, redact internal errors, and set content type.

## Scalability Considerations

Consistent handlers allow generated clients, OpenAPI alignment, and operational dashboards to scale across teams.

