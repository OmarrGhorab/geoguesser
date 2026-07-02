# Contract: Profiles Stats Progress

This feature updates the maintained API contract in `backend/openapi/openapi.yaml`. This document records the intended profile, public stats, and saved-progress contract behavior before task generation.

## Security Requirements

- Current-profile reads require a registered authenticated session.
- Profile updates require a registered authenticated session and CSRF protection for unsafe cookie-authenticated requests.
- Guest sessions must receive an authorization failure for current-profile endpoints.
- Public stats and public history endpoints are readable without private account access but must return only safe public data.
- Profile update endpoints must be rate-limited.

## `GET /api/v1/profile`

Returns the current registered player's profile, public-safe stats summary, and saved-progress summary needed by account surfaces.

### Success Response

```json
{
  "profile": {
    "user_id": "00000000-0000-0000-0000-000000000000",
    "display_name": "Raven",
    "avatar_url": "https://example.com/avatar.png",
    "country_code": "EG",
    "locale": "en",
    "timezone": "Africa/Cairo",
    "preferences": {
      "distance_unit": "km"
    },
    "created_at": "2026-07-01T10:00:00Z",
    "updated_at": "2026-07-01T10:00:00Z"
  },
  "stats": {
    "games_played": 12,
    "total_score": 43800,
    "average_score": 3650.0,
    "best_score": 4920,
    "last_played_at": "2026-07-01T09:30:00Z"
  },
  "progress": {
    "recent_games": [],
    "page": {
      "limit": 10,
      "next_cursor": null
    }
  }
}
```

### Error Responses

- `401 unauthorized`: missing, guest-only, expired, or invalid registered session.
- `404 not_found`: registered account exists but profile cannot be loaded.

## `PATCH /api/v1/profile`

Updates editable fields on the current registered player's profile. Omitted fields are preserved. Optional fields can be cleared only through explicit null values where supported by the final OpenAPI schema.

### Request

```json
{
  "display_name": "Raven",
  "avatar_url": "https://example.com/avatar.png",
  "country_code": "EG",
  "locale": "en",
  "timezone": "Africa/Cairo",
  "preferences": {
    "distance_unit": "km"
  }
}
```

### Validation

- `display_name`: trimmed, 2 to 32 user-visible characters, no unsupported control characters.
- `locale`: allowlisted supported locale.
- `country_code`: valid two-letter country code when present.
- `timezone`: valid timezone identifier when present.
- `avatar_url`: safe supported image reference when present.
- `preferences`: only known safe preference keys are accepted.

### Success Response

Returns the same shape as `GET /api/v1/profile`.

### Error Responses

- `400 bad_request`: malformed body or validation failure.
- `401 unauthorized`: missing, guest-only, expired, or invalid registered session.
- `403 forbidden`: CSRF failure or account cannot update profile.
- `429 rate_limited`: too many profile update attempts.

## `GET /api/v1/users/{userId}/stats`

Returns public-safe aggregate stats for a registered user.

### Success Response

```json
{
  "profile": {
    "user_id": "00000000-0000-0000-0000-000000000000",
    "display_name": "Raven",
    "avatar_url": "https://example.com/avatar.png",
    "country_code": "EG"
  },
  "stats": {
    "games_played": 12,
    "total_score": 43800,
    "average_score": 3650.0,
    "best_score": 4920,
    "last_played_at": "2026-07-01T09:30:00Z"
  }
}
```

### Privacy Guarantees

The response must not include email, auth/session data, private preferences, hidden locations, location IDs, answer coordinates, raw guess coordinates, or answer-revealing provider metadata.

### Error Responses

- `404 not_found`: user is missing or unavailable. Response must not distinguish private account states.

## `GET /api/v1/users/{userId}/games`

Returns cursor-paginated public-safe game history or saved-progress summaries for a registered user.

### Query Parameters

- `limit`: bounded page size, default 20, maximum 100.
- `cursor`: opaque cursor from a previous response.

### Success Response

```json
{
  "games": [
    {
      "id": "00000000-0000-0000-0000-000000000000",
      "map_id": "00000000-0000-0000-0000-000000000001",
      "mode": "solo",
      "status": "completed",
      "round_count": 5,
      "current_round_number": null,
      "total_score": 4920,
      "started_at": "2026-07-01T09:00:00Z",
      "completed_at": "2026-07-01T09:30:00Z",
      "created_at": "2026-07-01T08:59:00Z"
    }
  ],
  "page": {
    "limit": 20,
    "next_cursor": null
  }
}
```

### Privacy Guarantees

History summaries must not include hidden round answers, location IDs, answer coordinates, raw guess coordinates, or provider metadata. Detailed per-round results remain behind participant-authorized game result flows.

### Error Responses

- `400 bad_request`: invalid pagination parameters.
- `404 not_found`: user is missing or unavailable. Response must not distinguish private account states.

## OpenAPI Update Checklist

- Add or harden `ProfileResponse`, `UpdateProfileRequest`, `PublicProfileSummary`, `UserStatsResponse`, `UserGameHistoryResponse`, `UserGameHistoryItem`, and shared pagination schemas.
- Add `429` response for profile update rate limiting.
- Confirm `GET /api/v1/profile` and `PATCH /api/v1/profile` use registered-session security.
- Confirm public stats/history schemas do not contain private account or hidden gameplay fields.

## Final Schema Names (implemented)

The implemented `backend/openapi/openapi.yaml` uses `ProfileResponse`, `Profile`, `UpdateProfileRequest`, `ProgressSummary`, `PublicProfileResponse`, `PublicProfileSummary`, `PageInfo`, and an extended `UserStats`/`UserGameHistoryItem` (with `last_played_at` and `current_round_number` added) — matching this contract's shapes above. Validated via `npx pnpm@10.24.0 check:openapi` (0 errors, 29 pre-existing warnings unrelated to this feature).
