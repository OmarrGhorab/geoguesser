# Friends OpenAPI Contract Notes

Authoritative machine contract lives in `backend/openapi/openapi.yaml`. This note summarizes the friends surface for reviewers.

## Endpoints

| Method | Path | Auth | CSRF | Rate limit | Success |
| --- | --- | --- | --- | --- | --- |
| POST | `/friends/requests` | registered | yes | 10/min | 201 request |
| GET | `/friends/requests/incoming` | registered | no | 120/min | 200 page |
| GET | `/friends/requests/outgoing` | registered | no | 120/min | 200 page |
| POST | `/friends/requests/{requestId}/accept` | registered | yes | 30/min | 200 friendship |
| POST | `/friends/requests/{requestId}/decline` | registered | yes | 30/min | 204 |
| GET | `/friends` | registered | no | 120/min | 200 page |
| DELETE | `/friends/{userId}` | registered | yes | 30/min | 204 |
| POST | `/friends/{userId}/block` | registered | yes | 30/min | 204 |
| DELETE | `/friends/{userId}/block` | registered | yes | 30/min | 204 |
| GET | `/friends/blocked` | registered | no | 120/min | 200 page |
| GET | `/leaderboards/friends` | registered | no | 120/min | 200 leaderboard page |

## Shared rules

- Default `limit=20`, max `100`, opaque cursors.
- Error envelope uses existing shared codes: `unauthorized`, `validation_failed`, `not_found`, `conflict`, `rate_limited`, and service-unavailable mapping for dependency failures.
- Privacy-safe `not_found` for missing/inactive/blocked discovery outcomes.
- Public-safe person objects only.

## Example create request body

```json
{ "user_id": "00000000-0000-4000-8000-000000000002" }
```

## Example list item

```json
{
  "request_id": "00000000-0000-4000-8000-0000000000aa",
  "user": {
    "user_id": "00000000-0000-4000-8000-000000000002",
    "display_name": "Alex",
    "avatar_url": null,
    "country_code": "US"
  },
  "created_at": "2026-07-11T12:00:00Z"
}
```
