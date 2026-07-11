-- +goose Up
-- Durable ranked match foundations for Phase 9 matchmaking.
-- Redis remains authoritative only while searching/claimed; PostgreSQL is
-- authoritative after a complete match + game destination commits.

CREATE TABLE matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    formation_key TEXT NOT NULL,
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE RESTRICT,
    mode TEXT NOT NULL,
    status TEXT NOT NULL,
    matched_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,
    failure_code TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT matches_formation_key_unique UNIQUE (formation_key),
    CONSTRAINT matches_game_id_unique UNIQUE (game_id),
    CONSTRAINT matches_mode_check CHECK (mode IN ('ranked_standard')),
    CONSTRAINT matches_status_check CHECK (status IN ('matched', 'active', 'completed', 'cancelled', 'failed_to_start')),
    CONSTRAINT matches_lifecycle_check CHECK (
        (
            status = 'matched'
            AND started_at IS NULL
            AND completed_at IS NULL
            AND closed_at IS NULL
        )
        OR (
            status = 'active'
            AND started_at IS NOT NULL
            AND completed_at IS NULL
            AND closed_at IS NULL
        )
        OR (
            status = 'completed'
            AND started_at IS NOT NULL
            AND completed_at IS NOT NULL
            AND closed_at IS NULL
        )
        OR (
            status IN ('cancelled', 'failed_to_start')
            AND closed_at IS NOT NULL
            AND completed_at IS NULL
        )
    ),
    CONSTRAINT matches_failure_code_check CHECK (
		(
			status = 'failed_to_start'
			AND failure_code IS NOT NULL
			AND char_length(failure_code) BETWEEN 1 AND 64
		)
		OR (
			status <> 'failed_to_start'
			AND failure_code IS NULL
		)
    )
);

CREATE TRIGGER matches_updated_at BEFORE UPDATE ON matches
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX matches_mode_status_matched_at_idx
    ON matches (mode, status, matched_at DESC, id DESC);

CREATE INDEX matches_status_updated_at_idx
    ON matches (status, updated_at);

CREATE TABLE match_players (
    match_id UUID NOT NULL REFERENCES matches(id) ON DELETE RESTRICT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    game_player_id UUID NOT NULL REFERENCES game_players(id) ON DELETE RESTRICT,
    status TEXT NOT NULL,
    assigned_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,
    PRIMARY KEY (match_id, user_id),
    CONSTRAINT match_players_game_player_id_unique UNIQUE (game_player_id),
    CONSTRAINT match_players_match_game_player_unique UNIQUE (match_id, game_player_id),
    CONSTRAINT match_players_status_check CHECK (status IN ('assigned', 'active', 'completed', 'cancelled', 'failed')),
    CONSTRAINT match_players_lifecycle_check CHECK (
        (
            status = 'assigned'
            AND completed_at IS NULL
            AND closed_at IS NULL
        )
        OR (
            status = 'active'
            AND completed_at IS NULL
            AND closed_at IS NULL
        )
        OR (
            status = 'completed'
            AND completed_at IS NOT NULL
            AND closed_at IS NULL
        )
        OR (
            status IN ('cancelled', 'failed')
            AND closed_at IS NOT NULL
            AND completed_at IS NULL
        )
    )
);

-- Final concurrency guard: one active ranked assignment per registered user.
CREATE UNIQUE INDEX match_players_active_user_uidx
    ON match_players (user_id)
    WHERE status IN ('assigned', 'active');

CREATE INDEX match_players_user_match_idx
    ON match_players (user_id, match_id);

-- +goose Down
DROP INDEX IF EXISTS match_players_user_match_idx;
DROP INDEX IF EXISTS match_players_active_user_uidx;
DROP TABLE IF EXISTS match_players;
DROP INDEX IF EXISTS matches_status_updated_at_idx;
DROP INDEX IF EXISTS matches_mode_status_matched_at_idx;
DROP TABLE IF EXISTS matches;
