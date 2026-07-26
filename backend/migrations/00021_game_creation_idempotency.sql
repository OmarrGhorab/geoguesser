-- +goose Up
-- Durable creation idempotency for atomically-created games (Quick Play).
-- The column stores an actor-qualified composite key ("user:<uuid>:<key>" or
-- "guest:<hash>:<key>"), so one global unique index scopes retries per actor
-- for both registered users and guests (games.created_by_user_id is NULL for
-- guest-owned games and cannot participate in a per-actor index).
ALTER TABLE games ADD COLUMN creation_idempotency_key TEXT;
CREATE UNIQUE INDEX games_creation_idempotency_uidx
  ON games (creation_idempotency_key)
  WHERE creation_idempotency_key IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS games_creation_idempotency_uidx;
ALTER TABLE games DROP COLUMN IF EXISTS creation_idempotency_key;
