# Quickstart: Daily And Shared Challenges Validation

## Prerequisites

- Backend PostgreSQL and Redis are reachable through `backend/.env`.
- Backend migrations for challenges, attempts, leaderboards, streaks, and missions are applied.
- At least one active public map has enough active unique locations for the configured daily/shared challenge round count.
- Guest or signed-in auth/session flow is available.
- Frontend `.env.local` points at the backend API.

## Automated Validation

Run backend tests:

```powershell
cd backend
go test ./...
```

Evidence from implementation pass on 2026-06-27:

```text
PASS: go test ./...
```

Run targeted challenge tests once the package exists:

```powershell
cd backend
go test ./internal/challenges/... ./internal/games/...
```

Evidence from implementation pass on 2026-06-27:

```text
PASS: go test ./internal/challenges/... ./internal/games/...
```

Validate OpenAPI from the repository root:

```powershell
npx pnpm@10.24.0 check:openapi
```

Evidence from implementation pass on 2026-06-27:

```text
PASS: npx pnpm@10.24.0 check:openapi
Note: Redocly reported warnings already present in the contract style plus
ambiguous-path warnings for /challenges/shared/{code} beside
/challenges/{challengeId}/..., but validation completed successfully.
```

Run frontend quality gates:

```powershell
npx pnpm@10.24.0 --dir client lint
npx pnpm@10.24.0 --dir client typecheck
npx pnpm@10.24.0 --dir client build
```

Evidence from implementation pass on 2026-06-27:

```text
PASS: npx pnpm@10.24.0 --dir client lint
PASS: npx pnpm@10.24.0 --dir client typecheck
PASS: npx pnpm@10.24.0 --dir client build
```

Run the full workspace check if available:

```powershell
npx pnpm@10.24.0 check
```

## Manual Scenario 1: Daily Challenge Determinism

1. Start backend and frontend.
2. Open the daily challenge page in two separate browsers or profiles.
3. Confirm both sessions show the same:
   - challenge date
   - seed
   - map pool
   - locked settings
   - countdown reset target
4. Start the daily challenge in both sessions.
5. Confirm both sessions receive the same ordered rounds while guesses and scores remain independent.
6. Attempt to change challenge rules from the page or request payload and confirm the system rejects the change.

Expected result:

- Both sessions play identical locations in identical order.
- The current-round UI does not expose hidden coordinates.
- The daily countdown is visible and understandable.
- The daily rules remain locked.

Evidence from implementation pass on 2026-06-27:

- Automated coverage verifies canonical daily reset boundaries and date-bound deterministic seeds in `backend/internal/challenges/seed_test.go`.
- Automated coverage verifies duplicate selected locations are rejected for challenge materialization in `backend/internal/challenges/service_test.go`.
- Frontend production build includes `/[locale]/challenges/daily` as a dynamic server-rendered route.
- Full two-browser playthrough remains environment-dependent because it requires `backend/.env` Postgres/Redis, applied migrations, `CHALLENGE_DEFAULT_MAP_ID`, and seeded active map locations.

## Manual Scenario 2: Shared Challenge Link Stability

1. Create a shared challenge from an active map and fixed settings.
2. Copy the generated shared challenge link.
3. Open the link in two separate browsers or profiles.
4. Confirm both sessions load the same seed, map pool, and locked settings.
5. Start and complete the challenge in both sessions.
6. Reopen the link after completion.

Expected result:

- The link remains stable.
- Both sessions play identical rounds.
- Completed players see durable result summaries.
- Later visits do not mutate the original challenge rules or rounds.

Evidence from implementation pass on 2026-06-27:

- Automated coverage verifies shared code shape and frontend shared challenge identity fixtures.
- Backend handler coverage verifies POST `/challenges/shared`, GET `/challenges/shared/{code}`, and POST `/challenges/{challengeId}/attempts` route to JSON handlers.
- Frontend production build includes `/[locale]/challenges/[challengeId]` as a dynamic server-rendered route.
- Full two-browser shared-link playthrough remains environment-dependent for the same DB/Redis/seeded-map reasons as Scenario 1.

## Manual Scenario 3: Leaderboard And Spoiler Protection

1. Complete the same daily challenge with at least two signed-in accounts.
2. View the daily leaderboard after completion.
3. Open the daily challenge with a third unfinished account or guest.
4. Confirm spoiler-sensitive final details are hidden for the unfinished player while the challenge is playable.
5. Complete the challenge as the unfinished player and reload leaderboard/results.

Expected result:

- Leaderboard ordering is deterministic.
- Ties use stable tie-breakers.
- Unfinished players do not receive answer spoilers.
- Completed results remain stable after reload.

Evidence from implementation pass on 2026-06-27:

- Automated coverage verifies leaderboard tie-breakers by score, duration, completion time, and stable attempt ID in `backend/internal/challenges/leaderboard_test.go`.
- Backend service now logs spoiler-protected result reads for unfinished attempts and only marks result responses visible for completed attempts.
- OpenAPI challenge result schemas expose spoiler-safe visibility state and do not include hidden coordinates.

## Manual Scenario 4: Streaks

1. Complete a daily challenge before reset.
2. Confirm the current daily streak starts or increments.
3. Simulate or wait for the next challenge date.
4. Complete the next daily challenge and confirm consecutive increment.
5. Simulate a missed day or use a test fixture for missed-day behavior.

Expected result:

- Streak count, best count, last qualifying date, and protection state are visible.
- Missed-day behavior matches the documented protection state.
- Guest streaks clearly show persistence limits.

Evidence from implementation pass on 2026-06-27:

- Automated coverage verifies streak start, increment, missed-day reset, same-day idempotency, and protection state preservation in `backend/internal/challenges/streaks_test.go`.
- Frontend challenge panels render streak count, best count, protection state, and guest limitation fields from server DTOs.

## Manual Scenario 5: Missions

1. Open the missions surface before playing.
2. Complete qualifying daily/shared challenge actions:
   - daily completion
   - shared participation
   - score threshold
   - leaderboard milestone
   - streak milestone
   - round accuracy achievement
3. Confirm mission progress updates without manual refresh.
4. Complete a mission and confirm completed/claimable/claimed or status messaging.

Expected result:

- Mission progress updates within 5 seconds under normal conditions.
- Mission states are clear for empty, in-progress, completed, claimed, and expired cases.
- Guest mission progress is supported for the current session/device with visible limits.

Evidence from implementation pass on 2026-06-27:

- Automated coverage verifies the required mission types are present: daily completion, shared participation, score threshold, leaderboard milestone, streak milestone, and round accuracy.
- Backend repository includes idempotent mission progress event and progress upsert methods for account and guest owners.
- Mission update freshness is designed to occur during result finalization and is expected to satisfy the 5 second budget under normal DB latency.

## Manual Scenario 6: Localization And RTL

1. Open daily and shared challenge routes in English.
2. Open the same routes in Arabic.
3. Inspect daily challenge, shared challenge, leaderboard, missions, streaks, countdown, empty, error, disabled, and success states.
4. Navigate with keyboard only through primary controls.

Expected result:

- Visible copy is localized in both languages.
- Arabic uses RTL layout through existing locale direction handling.
- Buttons, tabs, links, and inputs have accessible names and visible focus states.
- Text fits in compact panels and controls without overlap.

Evidence from implementation pass on 2026-06-27:

- English and Arabic `Challenges` namespaces were added in `client/messages/en.json` and `client/messages/ar.json`.
- The existing locale layout applies `dir={getDirection(locale)}`, so Arabic routes render RTL.
- `npx pnpm@10.24.0 --dir client build` completed successfully with challenge routes included.
- Keyboard/browser visual review still needs to be repeated against a running app with seeded backend data before release.

## Operational Validation

- Check logs for challenge creation/materialization, attempt start/completion, leaderboard reads, streak updates, and mission progress without hidden coordinates or tokens.
- Check metrics for challenge attempt latency, completion counts, leaderboard reads, mission progress updates, and streak updates if implemented.
- Confirm health/readiness behavior is not degraded by challenge dependencies.

Evidence from implementation pass on 2026-06-27:

- Backend challenge logs use challenge IDs, attempt IDs, map IDs, scores, and event names only; they do not log hidden coordinates, tokens, or private profile details.
- Performance budget evidence available from automated build/test gates: challenge metadata and attempt-start code paths are bounded DB lookups/inserts with indexed challenge, location, attempt, and leaderboard queries.
- Runtime p95 confirmation should be captured after applying migrations to the `.env` Postgres/Redis and seeding `CHALLENGE_DEFAULT_MAP_ID`.

## Known Environment Note

The current root `docker-compose.yml` may not define local PostgreSQL and Redis services. If local Docker services are required for validation, add or use a compose override before claiming Docker-local manual validation.
