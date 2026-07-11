# Research: Friends And Social Graph Backend

## Decision 1: Sorted UUID pair uniqueness

**Decision**: Store every undirected relationship as one row with `user_a_id < user_b_id` (byte-wise UUID order) and a unique constraint on `(user_a_id, user_b_id)`.

**Rationale**: Prevents A→B and B→A duplicate edges and matches phase-3 database design.

**Alternatives considered**: Two directed edges (more writes, harder uniqueness); adjacency lists in JSON (poor indexing and concurrency).

## Decision 2: Decline deletes pending rows

**Decision**: Decline removes the pending friendship row rather than setting `status = declined`.

**Rationale**: Fixed product assumption; avoids retaining decline history that could leak through APIs or moderation surfaces not in scope.

**Alternatives considered**: Soft-declined status from phase-3 notes (rejected for this phase).

## Decision 3: Block replaces single-row state

**Decision**: Block upserts the pair row to `blocked` with `blocked_by_user_id`, clearing pending/accepted semantics in place.

**Rationale**: One pair row keeps uniqueness simple and makes block authoritative for social APIs.

**Alternatives considered**: Separate `blocks` table (clearer history, more joins); rejected to keep MVP simpler while preserving unblock ownership.

## Decision 4: Privacy-safe not-found

**Decision**: Missing users, inactive users, and blocked pairs produce the same not-found style outcome on request creation and similar discovery paths; block ownership is never returned to non-blockers.

**Rationale**: Prevents account and relationship enumeration.

## Decision 5: Stable lock order for mutations

**Decision**: Transactional request/accept/block paths lock the two user rows in sorted UUID order before reading/writing the friendship row.

**Rationale**: Prevents deadlocks under concurrent reciprocal requests and block races.

## Decision 6: No Redis friendship cache

**Decision**: All social graph reads and friends leaderboard pages hit PostgreSQL directly.

**Rationale**: Immediate consistency after accept/remove/block is required; graph fan-out is modest relative to global leaderboards.

## Decision 7: Friends leaderboard isolation

**Decision**: Implement cohort ranking inside `internal/leaderboards` with SQL that references `friendships` without importing `friends` package internals.

**Rationale**: Preserves package boundaries; leaderboards already own ranking projection and cursors.

## Decision 8: Rate limits

**Decision**: Hashed per registered-user limits — create request 10/min, mutating social actions 30/min, social reads 120/min; friends leaderboard uses authenticated per-user read limiting.

**Rationale**: Matches abuse sensitivity of social graph growth vs read fan-out.
