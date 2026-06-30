-- +goose Up
-- Add lookup indexes for private-room realtime membership and reconnect flows.

CREATE INDEX IF NOT EXISTS room_players_room_id_status_idx
    ON room_players (room_id, status);

CREATE INDEX IF NOT EXISTS room_players_game_player_id_idx
    ON room_players (game_player_id);

-- +goose Down

DROP INDEX IF EXISTS room_players_game_player_id_idx;
DROP INDEX IF EXISTS room_players_room_id_status_idx;
