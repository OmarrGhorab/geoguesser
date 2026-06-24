# Testing

## Concepts

Use the Go testing package, Testify, Testcontainers, integration tests, table-driven tests, benchmarks, and the race detector.

## Architecture Decisions

- Unit test services with fake repositories.
- Integration test repositories with PostgreSQL Testcontainers.
- Test Redis behavior with Redis Testcontainers.
- Use table-driven tests for validation and pure logic.
- Use benchmarks for hot code.

## Trade-offs

Unit tests are fast and focused. Integration tests catch schema/query/container issues. Use both; do not mock PostgreSQL behavior for repository correctness.

## Anti-patterns

- Only handler tests with mocks.
- Tests depending on developer machines.
- Sleeping instead of synchronization.
- No authz failure tests.
- Snapshotting JSON without semantic assertions.

## Common Mistakes

- Not calling `t.Helper`.
- Shared mutable test state.
- Ignoring race detector.
- Not cleaning containers.
- Missing negative cases.

## Production Examples

```go
func TestCreateUserValidation(t *testing.T) {
	tests := []struct {
		name string
		req  CreateUserRequest
		want error
	}{
		{name: "missing email", req: CreateUserRequest{}, want: ErrInvalidEmail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.ErrorIs(t, tt.req.Validate(), tt.want)
		})
	}
}
```

## Go Code Samples

```go
func BenchmarkEncodeUser(b *testing.B) {
	user := UserDTO{ID: uuid.New(), Email: "a@example.com"}
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(user)
	}
}
```

## Performance Considerations

Run benchmarks for hot paths and use `-benchmem`. Use `-race` for concurrency changes.

## Security Considerations

Test auth, authz, CSRF, rate limits, invalid payloads, and secret redaction.

## Scalability Considerations

Keep tests parallel-safe. Use Testcontainers for realistic integration without shared mutable infrastructure.

