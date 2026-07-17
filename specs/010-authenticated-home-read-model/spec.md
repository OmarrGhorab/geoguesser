# Feature Specification: Authenticated Home Read Model

## User Story

As a registered active player, I can load one authenticated home snapshot so the
home screen can show real profile, gameplay-stat, daily-challenge, and map data
without coordinating multiple HTTP requests.

## Acceptance Scenarios

1. Given an active registered session, when `GET /api/v1/home` is requested,
   then the response contains a public-safe viewer summary, aggregate gameplay
   stats, current daily-challenge metadata, and at most four active public maps.
2. Given an anonymous, guest, malformed, disabled, deleted, or otherwise
   inactive account session, when the endpoint is requested, then it returns a
   privacy-safe unauthorized response.
3. Given a profile, challenge, map, or database dependency failure, when the
   snapshot cannot be composed coherently, then the endpoint returns a stable
   service-unavailable error without leaking internal details.
4. Given a successful active-viewer validation, stats, daily-challenge, and map
   branches start concurrently and remain bounded to those three branches.

## Requirements

- **FR-001**: The endpoint MUST be `GET /api/v1/home` and require a registered
  access-cookie session.
- **FR-002**: Viewer data MUST include only user ID, display name, avatar URL,
  and country code.
- **FR-003**: Stats MUST use the existing completed-game aggregate definition.
- **FR-004**: Daily challenge data MUST use the existing challenge contract,
  including attempt state, daily streak, mission summary, participant count,
  and countdown when available.
- **FR-005**: Recommended maps MUST use the existing active-public map ordering
  and be limited to four entries.
- **FR-006**: After active-viewer validation, the service MUST use fixed,
  bounded concurrency for stats, daily challenge, and maps, and propagate
  request cancellation.
- **FR-007**: OpenAPI, authentication, rate limiting, metrics, structured logs,
  and automated tests MUST cover the endpoint.
- **FR-008**: Automatic daily-challenge materialization MUST skip active public
  maps that cannot supply the configured number of unique active locations.

## Out of Scope

- Frontend integration.
- Online-friend presence or friend previews.
- Level, XP, credits, wallets, economy, billing, subscriptions, or entitlements.
- Competitive win streaks and live game-mode activity counts.
- Personalized recommendation ranking or new map thumbnail storage.

## Success Criteria

- Focused service, handler, route, metrics, and OpenAPI tests pass.
- Backend `go test ./...`, `go vet ./...`, lint, OpenAPI validation, and build
  gates pass or have an exact environment blocker recorded.
- The endpoint starts no more than three concurrent read branches and returns no
  more than four maps.
