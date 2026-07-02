-- +goose Up
-- Support the profile "saved progress" game history view: looking up the
-- active round for a game (to surface current_round_number) and paginating
-- a user's completed/active/abandoned games together.

CREATE INDEX IF NOT EXISTS rounds_game_id_status_idx
    ON rounds (game_id, status);

CREATE INDEX IF NOT EXISTS games_status_idx
    ON games (status);

-- +goose Down

DROP INDEX IF EXISTS games_status_idx;
DROP INDEX IF EXISTS rounds_game_id_status_idx;
