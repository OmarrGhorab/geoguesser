# Data Model: Friends And Social Graph

## Entity: Friendship

| Column | Type | Required | Notes |
| --- | --- | --- | --- |
| `id` | `uuid` | yes | Primary key. |
| `user_a_id` | `uuid` | yes | Lower sorted user UUID, FK → `users.id`. |
| `user_b_id` | `uuid` | yes | Higher sorted user UUID, FK → `users.id`. |
| `requested_by_user_id` | `uuid` | yes | FK → requester; must be `user_a_id` or `user_b_id`. |
| `status` | `text` | yes | `pending`, `accepted`, `blocked`. |
| `blocked_by_user_id` | `uuid` | conditional | Required when `blocked`; null otherwise; must be pair member. |
| `accepted_at` | `timestamptz` | conditional | Required when `accepted`; null for pending/blocked. |
| `created_at` | `timestamptz` | yes | Defaults to `now()`. |
| `updated_at` | `timestamptz` | yes | Trigger-maintained. |

### Constraints

- `user_a_id <> user_b_id`
- unique `(user_a_id, user_b_id)`
- `requested_by_user_id IN (user_a_id, user_b_id)`
- lifecycle checks:
  - `pending`: `accepted_at IS NULL`, `blocked_by_user_id IS NULL`
  - `accepted`: `accepted_at IS NOT NULL`, `blocked_by_user_id IS NULL`
  - `blocked`: `blocked_by_user_id IS NOT NULL` and in pair, `accepted_at IS NULL`
- foreign keys to `users` with restrict/no cascade deletes from social edges (application removes edges explicitly when needed)

### Indexes

- unique pair index (constraint)
- `(user_a_id, status, created_at DESC, id DESC)`
- `(user_b_id, status, created_at DESC, id DESC)`
- partial/support indexes for pending requester and blocked_by lookup as needed by list queries

### State Transitions

```text
(none) --request--> pending
pending --accept--> accepted
pending --decline--> (deleted)
pending|accepted|(none) --block--> blocked
accepted --remove--> (deleted)
blocked --unblock by blocker--> (deleted)
blocked --same blocker reblock--> blocked (idempotent)
```

## Public Projection

Joined from `users` + `user_profiles` for list endpoints:

- `user_id`
- `display_name`
- `avatar_url` (optional)
- `country_code` (optional)

Never includes email, password hash, preferences, tokens, role, or account status distinctions beyond exclusion of inactive users from lists.

## Friends Leaderboard Cohort

Input set:

- caller `user_id`
- accepted friends where the other user is `active`

Ranking source: existing global `leaderboard_entries` competitive fields:

- score DESC
- completion_duration_ms ASC NULLS LAST
- completed_at ASC
- user_id ASC

Rank is recomputed inside the cohort, not global rank.
