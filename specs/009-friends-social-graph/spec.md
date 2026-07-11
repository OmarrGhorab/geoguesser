# Feature Specification: Friends And Social Graph Backend

**Feature Branch**: `009-friends-social-graph`  
**Created**: 2026-07-11  
**Status**: Approved for backend implementation  
**Input**: Phase 10 friends social graph and access controls; phase design sources for friendships, friend APIs, and friends leaderboards.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Send And Resolve Friend Requests (Priority: P1)

A registered user can send a friend request to another registered user by public user UUID, inspect incoming and outgoing pending requests, and the recipient can accept or decline the request. Declining deletes the pending relationship so either party may request again later.

**Why this priority**: Request lifecycle is the entry point for every durable social edge.

**Independent Test**: User A sends a request to User B; B sees it as incoming, A sees it as outgoing; B accepts or declines; guests and unauthorized users cannot mutate requests.

**Acceptance Scenarios**:

1. **Given** two active registered users with no relationship, **When** A sends a request to B, **Then** the system creates a pending friendship and returns a privacy-safe request payload.
2. **Given** a pending request from A to B, **When** B lists incoming requests, **Then** B sees A's public profile fields only.
3. **Given** a pending request from A to B, **When** A lists outgoing requests, **Then** A sees B's public profile fields only.
4. **Given** a pending request, **When** the recipient accepts, **Then** the relationship becomes accepted for both participants.
5. **Given** a pending request, **When** the recipient declines, **Then** the pending row is deleted and neither list shows the request.
6. **Given** a guest or unauthenticated caller, **When** they attempt any friend mutation, **Then** the API returns unauthorized without leaking social graph data.
7. **Given** a self-request, missing/inactive target, blocked pair, or duplicate/reciprocal conflict, **When** a request is submitted, **Then** the system rejects with stable validation, not-found, or conflict outcomes that remain privacy-safe.

---

### User Story 2 - View And Remove Accepted Friends (Priority: P1)

A registered user can list accepted friends with cursor pagination and remove an accepted friendship. Removal is symmetric: either participant can remove, and both lists update immediately.

**Why this priority**: Accepted friendships power friends leaderboards and future social surfaces.

**Independent Test**: After accepting a request, both users see each other in accepted lists; either removes the friendship; both lists update and subsequent list pages remain consistent.

**Acceptance Scenarios**:

1. **Given** an accepted friendship, **When** either user lists friends, **Then** the other user appears with public-safe profile projection only.
2. **Given** an accepted friendship, **When** either user removes the friend, **Then** the row is deleted and both lists no longer include each other.
3. **Given** a remove for a missing or already-removed friendship, **When** the owner retries, **Then** the response is idempotent success without leaking whether the other account exists.
4. **Given** inactive users, **When** friends are listed, **Then** inactive counterparts are excluded from results.

---

### User Story 3 - Block And Unblock Users (Priority: P1)

A registered user can block another user on social surfaces, inspect users they blocked, and later remove only their own block. Blocking replaces any pending or accepted relationship for the pair.

**Why this priority**: Blocking is the primary abuse-control and privacy boundary for social APIs.

**Independent Test**: Blocking replaces pending/accepted state, hides the pair from friend/request lists and friends leaderboards, and only the blocker can reverse the block.

**Acceptance Scenarios**:

1. **Given** no relationship or a pending/accepted one, **When** A blocks B, **Then** the durable state becomes blocked with A as blocker and prior social edges for the pair are cleared.
2. **Given** A blocked B, **When** A lists blocked users, **Then** B appears with public-safe fields; B cannot discover that A owns the block through friend APIs.
3. **Given** A blocked B, **When** either attempts friend requests, **Then** outcomes remain privacy-safe (not-found style) and no pending edge is created.
4. **Given** A blocked B, **When** A unblocks B, **Then** the block row is removed; when B attempts to unblock, the operation is a privacy-safe no-op.
5. **Given** the same blocker repeats a block, **When** the command is retried, **Then** the response is idempotent success.

---

### User Story 4 - Friends Leaderboard (Priority: P2)

An authenticated registered user can view a leaderboard ranked only among self and accepted friends. Rank is recalculated inside that cohort using existing competitive ordering.

**Why this priority**: Friends comparisons are the first competitive product value of the social graph.

**Independent Test**: Given accepted friends, pending requests, blocked users, and unrelated users, the endpoint returns only self plus accepted friends with cohort-relative ranks.

**Acceptance Scenarios**:

1. **Given** accepted friends with scores, **When** the caller reads the friends leaderboard, **Then** only self and accepted active friends appear with ranks relative to that set.
2. **Given** pending, blocked, non-friend, or inactive users, **When** the friends leaderboard is read, **Then** those users are excluded.
3. **Given** pagination parameters, **When** pages are fetched, **Then** cursors remain deterministic under the same ordering rules as other leaderboards.
4. **Given** a guest session, **When** the friends leaderboard is requested, **Then** access is denied.

---

### Edge Cases

- Self-friending, self-blocking, and self-removal are rejected.
- Concurrent duplicate requests and reversed A→B / B→A races serialize on a stable user lock order.
- Target missing, disabled, deleted, or pending_verification is treated as privacy-safe not found for social discovery paths.
- Declines delete pending rows rather than retaining declined history.
- Blocking replaces pending and accepted state; decline history and block ownership never appear in responses or logs.
- Only the blocker may reverse a block; the blocked user sees no special block-ownership signal.
- Friends leaderboard does not use Redis page caching and must stay immediately consistent with PostgreSQL.
- Rate limits: friend-request create 10/min, social actions 30/min, social reads 120/min, all hashed per registered user.
- Responses and logs exclude email, private preferences, tokens, decline history, and block ownership.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Only registered active sessions may use friends commands and friends leaderboard reads.
- **FR-002**: Targets MUST be addressed by existing public user UUIDs.
- **FR-003**: Friendship pairs MUST be stored as sorted UUID pairs with uniqueness and self-pair rejection.
- **FR-004**: Supported durable statuses are `pending`, `accepted`, and `blocked`; declines delete pending rows.
- **FR-005**: Request creation MUST reject self-targets, missing/inactive targets, blocked pairs (privacy-safe), and duplicate/reciprocal pending or accepted conflicts.
- **FR-006**: Only the request recipient may accept; accept is transactional.
- **FR-007**: Only the request recipient may decline; decline deletes the pending row.
- **FR-008**: Accepted friends lists are symmetric and cursor-paginated with public-safe profile projection.
- **FR-009**: Either accepted participant may remove the friendship; removal is idempotent for the caller.
- **FR-010**: Block replaces pending/accepted state, records `blocked_by_user_id`, and is idempotent for the same blocker.
- **FR-011**: Only the blocker may unblock; other-party unblock is a privacy-safe no-op.
- **FR-012**: Blocked-user list shows only blocks created by the caller.
- **FR-013**: Blocking applies only to friend APIs and friends leaderboards in this phase (no profile hiding, rooms, or matchmaking exclusions).
- **FR-014**: Friends leaderboard ranks self plus accepted friends using existing score, duration, completion-time, and stable-user order without Redis page caching.
- **FR-015**: OpenAPI contracts, Goose migration, metrics, structured logs, and automated tests MUST cover the behaviors above.

### Key Entities

- **Friendship**: Sorted pair edge with requester, status, optional acceptor/blocker metadata, and timestamps.
- **Public friend profile projection**: user id, display name, avatar, optional country — never email or preferences.
- **Friends leaderboard cohort**: caller plus accepted active friends with relative ranks.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Independent story tests for US1–US4 pass against PostgreSQL-backed integration coverage.
- **SC-002**: Concurrent duplicate request races produce one durable pending edge and no double-accept corruption.
- **SC-003**: p95 list and friends-leaderboard page reads stay within the plan performance budgets under local fixture load.
- **SC-004**: Privacy regression tests prove responses and logs exclude forbidden fields.
- **SC-005**: Backend quality gates (`go test`, race where required, lint, OpenAPI validation) pass for the feature packages.

## Assumptions

- Backend-only delivery in this feature branch.
- No friendship Redis cache, realtime notifications, profile hiding, room restrictions, or matchmaking exclusions.
- Friends rank is relative to self plus accepted friends only.
- Only the blocker may unblock.
- Declines delete pending rows and leave no decline history API.
