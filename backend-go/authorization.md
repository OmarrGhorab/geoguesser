# Authorization

## Concepts

Authorization decides what an authenticated actor can do. Use RBAC by default, extend with resource ownership or policy checks where needed.

## Architecture Decisions

- Enforce authorization in services.
- Use middleware only to require authentication or coarse roles.
- Model roles and permissions explicitly.
- Check tenant/resource ownership in use cases.

## Trade-offs

RBAC is simple and auditable. Fine-grained policies are more expressive but harder to test and explain.

## Anti-patterns

- UI-only authorization.
- Handler-only authorization.
- Role strings scattered through code.
- Admin bypasses without audit.
- Caching permissions without invalidation.

## Common Mistakes

- Checking authentication but not ownership.
- Forgetting background jobs and internal APIs.
- Returning different errors that reveal resource existence.
- Missing tests for forbidden paths.
- Not logging sensitive authorization failures.

## Production Examples

```go
func (s *ProjectService) Delete(ctx context.Context, actor Actor, id uuid.UUID) error {
	member, err := s.members.Get(ctx, id, actor.UserID)
	if err != nil {
		return fmt.Errorf("get membership: %w", err)
	}
	if member.Role != RoleOwner {
		return ErrForbidden
	}
	return s.projects.Delete(ctx, id)
}
```

## Go Code Samples

```go
type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)
```

## Performance Considerations

Index membership and permission lookups. Avoid repeated authz queries in one request by loading required policy data once.

## Security Considerations

Deny by default. Audit high-risk decisions. Check authz inside every mutation and sensitive read.

## Scalability Considerations

Centralize policy helpers so teams can add roles and permissions without inconsistent behavior.

