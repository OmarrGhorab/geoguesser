# Pagination

## Concepts

Pagination covers offset pagination, cursor pagination, filtering, sorting, and searching. Every list endpoint must be bounded.

## Architecture Decisions

- Use cursor pagination for large or frequently changing datasets.
- Use offset pagination for small admin lists where exact pages matter.
- Validate limit, cursor, sort, and filters.
- Document pagination in OpenAPI.
- Return next cursor and total only when efficient.

## Trade-offs

Offset is simple but slow and inconsistent on large changing tables. Cursor pagination is stable and fast but less flexible for random page jumps.

## Anti-patterns

- Unbounded `Find`.
- Client-controlled sort column without allowlist.
- Total count on every large query.
- Cursor format exposing internal details.
- Search without indexes.

## Common Mistakes

- Missing deterministic tie-breaker.
- No max limit.
- Cursor not scoped to filters.
- SQL injection through sort params.
- Inconsistent response envelope.

## Production Examples

```json
{
  "data": [],
  "page": {
    "limit": 50,
    "next_cursor": "eyJjcmVhdGVkX2F0IjoiLi4uIn0="
  }
}
```

## Go Code Samples

```go
var allowedSorts = map[string]string{
	"created_at": "created_at",
	"name":       "name",
}
```

## Performance Considerations

Use indexed sort/filter columns. Avoid large offsets. Use full-text indexes for search.

## Security Considerations

Allowlist sort/filter fields. Sign or encode cursors to prevent tampering.

## Scalability Considerations

Cursor pagination scales to large tables and high write rates.

