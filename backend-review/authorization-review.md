# Authorization Review

## Verify

- RBAC.
- Ownership checks.
- Tenant boundaries.
- Service-layer enforcement.
- Admin actions audited.
- Deny-by-default behavior.

## Reject

- Handler-only authorization.
- UI-only authorization.
- Role strings scattered through code.
- Missing authorization on repository-backed sensitive reads.

## Common Findings

High: service checks authentication but not project membership. Impact: authenticated users can access cross-tenant resources. Recommendation: check membership/role in the service before repository mutation.

