-- +goose Up
CREATE TABLE leaderboards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind TEXT NOT NULL CHECK (kind IN ('global', 'daily', 'map')),
    scope_key TEXT NOT NULL,
    display_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    ranking_rule TEXT NOT NULL DEFAULT 'best_score',
    map_id UUID REFERENCES maps(id) ON DELETE CASCADE,
    challenge_id UUID REFERENCES challenges(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (kind, scope_key)
);

CREATE TRIGGER leaderboards_updated_at BEFORE UPDATE ON leaderboards
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

ALTER TABLE leaderboard_entries DROP CONSTRAINT leaderboard_entries_pkey;
ALTER TABLE leaderboard_entries ADD COLUMN id UUID NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE leaderboard_entries ADD CONSTRAINT leaderboard_entries_pkey PRIMARY KEY (id);
ALTER TABLE leaderboard_entries ALTER COLUMN challenge_id DROP NOT NULL;
ALTER TABLE leaderboard_entries ALTER COLUMN attempt_id DROP NOT NULL;
ALTER TABLE leaderboard_entries ADD COLUMN leaderboard_id UUID REFERENCES leaderboards(id) ON DELETE CASCADE;
ALTER TABLE leaderboard_entries ADD COLUMN game_id UUID REFERENCES games(id) ON DELETE CASCADE;
ALTER TABLE leaderboard_entries ADD COLUMN games_played INT NOT NULL DEFAULT 1;
ALTER TABLE leaderboard_entries ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE TRIGGER leaderboard_entries_updated_at BEFORE UPDATE ON leaderboard_entries
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Compares two leaderboard candidates. Higher score wins, then shorter
-- completion duration, then earlier completion, then stable game id.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION leaderboard_entry_is_better(
    new_score INT,
    new_completion_duration_ms BIGINT,
    new_completed_at TIMESTAMPTZ,
    new_game_id UUID,
    current_score INT,
    current_completion_duration_ms BIGINT,
    current_completed_at TIMESTAMPTZ,
    current_game_id UUID
)
RETURNS BOOLEAN AS $$
BEGIN
    IF new_score <> current_score THEN
        RETURN new_score > current_score;
    END IF;

    IF new_completion_duration_ms IS NOT NULL AND current_completion_duration_ms IS NULL THEN
        RETURN TRUE;
    END IF;

    IF new_completion_duration_ms IS NULL AND current_completion_duration_ms IS NOT NULL THEN
        RETURN FALSE;
    END IF;

    IF new_completion_duration_ms IS NOT NULL
        AND current_completion_duration_ms IS NOT NULL
        AND new_completion_duration_ms <> current_completion_duration_ms THEN
        RETURN new_completion_duration_ms < current_completion_duration_ms;
    END IF;

    IF new_completed_at <> current_completed_at THEN
        RETURN new_completed_at < current_completed_at;
    END IF;

    RETURN new_game_id < current_game_id;
END;
$$ LANGUAGE plpgsql IMMUTABLE;
-- +goose StatementEnd

CREATE UNIQUE INDEX leaderboard_entries_challenge_attempt_key
    ON leaderboard_entries (challenge_id, attempt_id)
    WHERE challenge_id IS NOT NULL AND attempt_id IS NOT NULL;

CREATE UNIQUE INDEX leaderboard_entries_leaderboard_user_key
    ON leaderboard_entries (leaderboard_id, user_id)
    WHERE leaderboard_id IS NOT NULL;

CREATE INDEX leaderboard_entries_leaderboard_rank_idx
    ON leaderboard_entries (leaderboard_id, rank, user_id)
    WHERE leaderboard_id IS NOT NULL;

CREATE INDEX leaderboard_entries_leaderboard_order_idx
    ON leaderboard_entries (leaderboard_id, score DESC, completion_duration_ms ASC, completed_at ASC, user_id ASC)
    WHERE leaderboard_id IS NOT NULL;

CREATE INDEX games_completed_leaderboard_idx
    ON games (status, map_id, completed_at DESC)
    WHERE status = 'completed';

CREATE INDEX game_players_registered_leaderboard_idx
    ON game_players (game_id, user_id, total_score DESC)
    WHERE user_id IS NOT NULL AND status = 'active';

ALTER TABLE leaderboard_entries ADD CONSTRAINT leaderboard_entries_scope_check CHECK (
    (challenge_id IS NOT NULL AND attempt_id IS NOT NULL AND leaderboard_id IS NULL AND game_id IS NULL)
    OR
    (challenge_id IS NULL AND attempt_id IS NULL AND leaderboard_id IS NOT NULL AND game_id IS NOT NULL)
);

-- +goose Down
ALTER TABLE leaderboard_entries DROP CONSTRAINT IF EXISTS leaderboard_entries_scope_check;
DROP INDEX IF EXISTS game_players_registered_leaderboard_idx;
DROP INDEX IF EXISTS games_completed_leaderboard_idx;
DROP INDEX IF EXISTS leaderboard_entries_leaderboard_order_idx;
DROP INDEX IF EXISTS leaderboard_entries_leaderboard_rank_idx;
DROP INDEX IF EXISTS leaderboard_entries_leaderboard_user_key;
DROP INDEX IF EXISTS leaderboard_entries_challenge_attempt_key;

DROP TRIGGER IF EXISTS leaderboard_entries_updated_at ON leaderboard_entries;
DROP FUNCTION IF EXISTS leaderboard_entry_is_better(INT, BIGINT, TIMESTAMPTZ, UUID, INT, BIGINT, TIMESTAMPTZ, UUID);

-- Generic leaderboard rows are derived read-model data and are removed on rollback.
DELETE FROM leaderboard_entries WHERE leaderboard_id IS NOT NULL;

ALTER TABLE leaderboard_entries DROP CONSTRAINT leaderboard_entries_pkey;
ALTER TABLE leaderboard_entries ALTER COLUMN challenge_id SET NOT NULL;
ALTER TABLE leaderboard_entries ALTER COLUMN attempt_id SET NOT NULL;
ALTER TABLE leaderboard_entries ADD CONSTRAINT leaderboard_entries_pkey PRIMARY KEY (challenge_id, attempt_id);
ALTER TABLE leaderboard_entries DROP COLUMN IF EXISTS updated_at;
ALTER TABLE leaderboard_entries DROP COLUMN IF EXISTS games_played;
ALTER TABLE leaderboard_entries DROP COLUMN IF EXISTS game_id;
ALTER TABLE leaderboard_entries DROP COLUMN IF EXISTS leaderboard_id;
ALTER TABLE leaderboard_entries DROP COLUMN IF EXISTS id;

DROP TRIGGER IF EXISTS leaderboards_updated_at ON leaderboards;
DROP TABLE IF EXISTS leaderboards;
