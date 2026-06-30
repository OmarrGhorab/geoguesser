-- +goose Up
-- Add indexes to support cursor-paginated registered-user game history queries.

CREATE INDEX IF NOT EXISTS game_players_user_id_status_idx
    ON game_players (user_id, status)
    WHERE user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS games_created_at_id_desc_idx
    ON games (created_at DESC, id DESC);

-- +goose Down

DROP INDEX IF EXISTS games_created_at_id_desc_idx;
DROP INDEX IF EXISTS game_players_user_id_status_idx;
