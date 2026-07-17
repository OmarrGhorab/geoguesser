-- +goose Up
-- Challenge-scoped events (not attempt-scoped events) must only advance a
-- mission once per challenge and owner. This keeps weekly streak progress from
-- increasing when the same daily challenge is replayed.
CREATE UNIQUE INDEX IF NOT EXISTS mission_events_challenge_user_key
    ON mission_progress_events (mission_id, owner_user_id, source_challenge_id, event_type)
    WHERE owner_user_id IS NOT NULL
      AND source_attempt_id IS NULL
      AND source_challenge_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS mission_events_challenge_guest_key
    ON mission_progress_events (mission_id, guest_identity_hash, source_challenge_id, event_type)
    WHERE guest_identity_hash IS NOT NULL
      AND source_attempt_id IS NULL
      AND source_challenge_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS mission_events_challenge_guest_key;
DROP INDEX IF EXISTS mission_events_challenge_user_key;
