-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS guesses_game_player_id_idempotency_key_unique
    ON guesses (game_player_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS rounds_game_id_status_round_number
    ON rounds (game_id, status, round_number);

-- +goose Down
DROP INDEX IF EXISTS rounds_game_id_status_round_number;
DROP INDEX IF EXISTS guesses_game_player_id_idempotency_key_unique;
