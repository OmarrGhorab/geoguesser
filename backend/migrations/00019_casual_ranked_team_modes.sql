-- +goose Up
-- Casual/Ranked team modes: parties, competitive seasons, team chat/moderation,
-- match/game/guess extensions, and legacy ranked_standard backfill.

-- ---------------------------------------------------------------------------
-- Parties
-- ---------------------------------------------------------------------------

CREATE TABLE parties (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    format TEXT NOT NULL,
    capacity SMALLINT NOT NULL,
    leader_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status TEXT NOT NULL,
    version INT NOT NULL DEFAULT 0,
    active_match_id UUID REFERENCES matches(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at TIMESTAMPTZ,
    CONSTRAINT parties_format_check CHECK (format IN ('duo', 'squad')),
    CONSTRAINT parties_capacity_format_check CHECK (
        (format = 'duo' AND capacity = 2)
        OR (format = 'squad' AND capacity = 4)
    ),
    CONSTRAINT parties_status_check CHECK (status IN ('forming', 'queued', 'in_match', 'closed')),
    CONSTRAINT parties_version_non_negative CHECK (version >= 0),
    CONSTRAINT parties_in_match_active_match_check CHECK (
        (status = 'in_match' AND active_match_id IS NOT NULL)
        OR (status <> 'in_match')
    ),
    CONSTRAINT parties_closed_at_check CHECK (
        (status = 'closed' AND closed_at IS NOT NULL)
        OR (status <> 'closed' AND closed_at IS NULL)
    )
);

CREATE TRIGGER parties_updated_at BEFORE UPDATE ON parties
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX parties_leader_status_idx
    ON parties (leader_user_id, status);

CREATE INDEX parties_status_updated_at_idx
    ON parties (status, updated_at);

CREATE INDEX parties_active_match_id_idx
    ON parties (active_match_id)
    WHERE active_match_id IS NOT NULL;

CREATE TABLE party_members (
    party_id UUID NOT NULL REFERENCES parties(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status TEXT NOT NULL,
    ready BOOLEAN NOT NULL DEFAULT false,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    left_at TIMESTAMPTZ,
    PRIMARY KEY (party_id, user_id),
    CONSTRAINT party_members_status_check CHECK (status IN ('active', 'left', 'kicked')),
    CONSTRAINT party_members_left_at_check CHECK (
        (status = 'active' AND left_at IS NULL)
        OR (status IN ('left', 'kicked') AND left_at IS NOT NULL)
    )
);

CREATE UNIQUE INDEX party_members_active_user_uidx
    ON party_members (user_id)
    WHERE status = 'active';

CREATE INDEX party_members_party_status_joined_idx
    ON party_members (party_id, status, joined_at, user_id);

CREATE TABLE party_invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    party_id UUID NOT NULL REFERENCES parties(id) ON DELETE CASCADE,
    inviter_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    invitee_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    responded_at TIMESTAMPTZ,
    CONSTRAINT party_invites_distinct_users_check CHECK (inviter_user_id <> invitee_user_id),
    CONSTRAINT party_invites_status_check CHECK (
        status IN ('pending', 'accepted', 'declined', 'expired', 'revoked')
    ),
    CONSTRAINT party_invites_responded_at_check CHECK (
        (status = 'pending' AND responded_at IS NULL)
        OR (status <> 'pending')
    )
);

CREATE UNIQUE INDEX party_invites_pending_unique
    ON party_invites (party_id, invitee_user_id)
    WHERE status = 'pending';

CREATE INDEX party_invites_invitee_status_idx
    ON party_invites (invitee_user_id, status, created_at DESC);

-- ---------------------------------------------------------------------------
-- Competitive seasons / standings / rating history
-- ---------------------------------------------------------------------------

CREATE TABLE competitive_seasons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sequence INT NOT NULL,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    closed_at TIMESTAMPTZ,
    initial_rating INT NOT NULL DEFAULT 800,
    elo_k INT NOT NULL DEFAULT 32,
    reset_factor_bps INT NOT NULL DEFAULT 5000,
    top500_min_matches INT NOT NULL DEFAULT 25,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT competitive_seasons_sequence_unique UNIQUE (sequence),
    CONSTRAINT competitive_seasons_slug_unique UNIQUE (slug),
    CONSTRAINT competitive_seasons_sequence_positive CHECK (sequence > 0),
    CONSTRAINT competitive_seasons_status_check CHECK (status IN ('scheduled', 'active', 'closed')),
    CONSTRAINT competitive_seasons_ends_after_starts CHECK (ends_at > starts_at),
    CONSTRAINT competitive_seasons_initial_rating_non_negative CHECK (initial_rating >= 0),
    CONSTRAINT competitive_seasons_elo_k_positive CHECK (elo_k > 0),
    CONSTRAINT competitive_seasons_reset_factor_bps_range CHECK (reset_factor_bps BETWEEN 0 AND 10000),
    CONSTRAINT competitive_seasons_top500_min_matches_non_negative CHECK (top500_min_matches >= 0),
    CONSTRAINT competitive_seasons_closed_lifecycle_check CHECK (
        (status = 'closed' AND closed_at IS NOT NULL)
        OR (status <> 'closed' AND closed_at IS NULL)
    )
);

CREATE TRIGGER competitive_seasons_updated_at BEFORE UPDATE ON competitive_seasons
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE UNIQUE INDEX competitive_seasons_one_active_uidx
    ON competitive_seasons ((true))
    WHERE status = 'active';

CREATE TABLE competitive_standings (
    season_id UUID NOT NULL REFERENCES competitive_seasons(id) ON DELETE RESTRICT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    rating INT NOT NULL,
    placements_completed SMALLINT NOT NULL DEFAULT 0,
    matches_played INT NOT NULL DEFAULT 0,
    wins INT NOT NULL DEFAULT 0,
    losses INT NOT NULL DEFAULT 0,
    draws INT NOT NULL DEFAULT 0,
    abandons INT NOT NULL DEFAULT 0,
    peak_rating INT NOT NULL,
    rating_reached_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_match_id UUID REFERENCES matches(id) ON DELETE RESTRICT,
    final_position INT,
    ending_rank_code TEXT,
    peak_rank_code TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (season_id, user_id),
    CONSTRAINT competitive_standings_rating_non_negative CHECK (rating >= 0),
    CONSTRAINT competitive_standings_peak_rating_check CHECK (peak_rating >= rating),
    CONSTRAINT competitive_standings_placements_range CHECK (placements_completed BETWEEN 0 AND 5),
    CONSTRAINT competitive_standings_counts_non_negative CHECK (
        matches_played >= 0
        AND wins >= 0
        AND losses >= 0
        AND draws >= 0
        AND abandons >= 0
    ),
    CONSTRAINT competitive_standings_final_position_positive CHECK (
        final_position IS NULL OR final_position > 0
    )
);

CREATE TRIGGER competitive_standings_updated_at BEFORE UPDATE ON competitive_standings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX competitive_standings_top500_idx
    ON competitive_standings (
        season_id,
        rating DESC,
        wins DESC,
        rating_reached_at ASC,
        user_id ASC
    )
    INCLUDE (matches_played, placements_completed, peak_rating);

CREATE TABLE competitive_rating_changes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    season_id UUID NOT NULL REFERENCES competitive_seasons(id) ON DELETE RESTRICT,
    match_id UUID NOT NULL REFERENCES matches(id) ON DELETE RESTRICT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    team_slot SMALLINT NOT NULL,
    outcome TEXT NOT NULL,
    own_team_average INT NOT NULL,
    opponent_team_average INT NOT NULL,
    expected_bps INT NOT NULL,
    base_delta INT NOT NULL,
    abandon_penalty INT NOT NULL DEFAULT 0,
    old_rating INT NOT NULL,
    new_rating INT NOT NULL,
    total_delta INT NOT NULL,
    old_rank_code TEXT,
    new_rank_code TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT competitive_rating_changes_match_user_unique UNIQUE (match_id, user_id),
    CONSTRAINT competitive_rating_changes_team_slot_check CHECK (team_slot IN (1, 2)),
    CONSTRAINT competitive_rating_changes_outcome_check CHECK (outcome IN ('win', 'loss', 'draw')),
    CONSTRAINT competitive_rating_changes_abandon_penalty_check CHECK (abandon_penalty IN (0, -15)),
    CONSTRAINT competitive_rating_changes_ratings_non_negative CHECK (
        old_rating >= 0 AND new_rating >= 0
    )
);

CREATE INDEX competitive_rating_changes_season_user_created_idx
    ON competitive_rating_changes (season_id, user_id, created_at DESC, id DESC);

CREATE INDEX competitive_rating_changes_match_idx
    ON competitive_rating_changes (match_id);

-- ---------------------------------------------------------------------------
-- Seed Season 1 (active, 84-day window from migration time)
-- ---------------------------------------------------------------------------

INSERT INTO competitive_seasons (
    sequence,
    slug,
    name,
    status,
    starts_at,
    ends_at,
    initial_rating,
    elo_k,
    reset_factor_bps,
    top500_min_matches
) VALUES (
    1,
    'season-1',
    'Season 1',
    'active',
    now(),
    now() + INTERVAL '84 days',
    800,
    32,
    5000,
    25
);

-- ---------------------------------------------------------------------------
-- Match extensions + mode expansion + legacy backfill
-- ---------------------------------------------------------------------------

ALTER TABLE matches DROP CONSTRAINT matches_mode_check;
ALTER TABLE matches ADD CONSTRAINT matches_mode_check CHECK (
    mode IN (
        'ranked_standard',
        'casual_solo',
        'casual_duo',
        'casual_squad',
        'ranked_solo',
        'ranked_duo',
        'ranked_squad'
    )
);

ALTER TABLE matches
    ADD COLUMN playlist TEXT,
    ADD COLUMN format TEXT,
    ADD COLUMN team_size SMALLINT,
    ADD COLUMN season_id UUID REFERENCES competitive_seasons(id) ON DELETE RESTRICT,
    ADD COLUMN team_one_score INT NOT NULL DEFAULT 0,
    ADD COLUMN team_two_score INT NOT NULL DEFAULT 0,
    ADD COLUMN winner_team_slot SMALLINT,
    ADD COLUMN result TEXT,
    ADD COLUMN last_activity_at TIMESTAMPTZ,
    ADD COLUMN chat_access_until TIMESTAMPTZ,
    ADD COLUMN progression_finalized_at TIMESTAMPTZ;

UPDATE matches
SET
    playlist = 'ranked',
    format = 'solo',
    team_size = 1,
    season_id = (SELECT id FROM competitive_seasons WHERE sequence = 1 LIMIT 1),
    last_activity_at = COALESCE(completed_at, started_at, matched_at, created_at)
WHERE mode = 'ranked_standard';

ALTER TABLE matches
    ALTER COLUMN playlist SET NOT NULL,
    ALTER COLUMN format SET NOT NULL,
    ALTER COLUMN team_size SET NOT NULL,
    ALTER COLUMN last_activity_at SET NOT NULL,
    ALTER COLUMN last_activity_at SET DEFAULT now();

ALTER TABLE matches
    ADD CONSTRAINT matches_playlist_check CHECK (playlist IN ('casual', 'ranked')),
    ADD CONSTRAINT matches_format_check CHECK (format IN ('solo', 'duo', 'squad')),
    ADD CONSTRAINT matches_team_size_format_check CHECK (
        (format = 'solo' AND team_size = 1)
        OR (format = 'duo' AND team_size = 2)
        OR (format = 'squad' AND team_size = 4)
    ),
    ADD CONSTRAINT matches_season_playlist_check CHECK (
        (playlist = 'ranked' AND season_id IS NOT NULL)
        OR (playlist = 'casual' AND season_id IS NULL)
    ),
    ADD CONSTRAINT matches_mode_playlist_format_check CHECK (
        (mode = 'ranked_standard' AND playlist = 'ranked' AND format = 'solo' AND team_size = 1)
        OR (mode = 'casual_solo' AND playlist = 'casual' AND format = 'solo' AND team_size = 1)
        OR (mode = 'casual_duo' AND playlist = 'casual' AND format = 'duo' AND team_size = 2)
        OR (mode = 'casual_squad' AND playlist = 'casual' AND format = 'squad' AND team_size = 4)
        OR (mode = 'ranked_solo' AND playlist = 'ranked' AND format = 'solo' AND team_size = 1)
        OR (mode = 'ranked_duo' AND playlist = 'ranked' AND format = 'duo' AND team_size = 2)
        OR (mode = 'ranked_squad' AND playlist = 'ranked' AND format = 'squad' AND team_size = 4)
    ),
    ADD CONSTRAINT matches_team_scores_non_negative CHECK (
        team_one_score >= 0 AND team_two_score >= 0
    ),
    ADD CONSTRAINT matches_winner_team_slot_check CHECK (
        winner_team_slot IS NULL OR winner_team_slot IN (1, 2)
    ),
    ADD CONSTRAINT matches_result_check CHECK (
        result IS NULL
        OR result IN ('team_one_win', 'team_two_win', 'draw', 'forfeit', 'abandoned', 'cancelled')
    );

CREATE INDEX matches_playlist_format_status_matched_idx
    ON matches (playlist, format, status, matched_at DESC, id);

CREATE INDEX matches_season_status_completed_idx
    ON matches (season_id, status, completed_at);

CREATE INDEX matches_status_last_activity_idx
    ON matches (status, last_activity_at);

-- ---------------------------------------------------------------------------
-- Match player extensions + deterministic team-slot backfill
-- ---------------------------------------------------------------------------

ALTER TABLE match_players
    ADD COLUMN team_slot SMALLINT,
    ADD COLUMN party_id UUID REFERENCES parties(id) ON DELETE RESTRICT,
    ADD COLUMN abandoned_at TIMESTAMPTZ,
    ADD COLUMN abandon_reason TEXT;

WITH ordered AS (
    SELECT
        match_id,
        user_id,
        ROW_NUMBER() OVER (
            PARTITION BY match_id
            ORDER BY assigned_at ASC, user_id ASC
        ) AS rn
    FROM match_players
)
UPDATE match_players mp
SET team_slot = ordered.rn::SMALLINT
FROM ordered
WHERE mp.match_id = ordered.match_id
  AND mp.user_id = ordered.user_id;

ALTER TABLE match_players
    ALTER COLUMN team_slot SET NOT NULL;

ALTER TABLE match_players
    ADD CONSTRAINT match_players_team_slot_check CHECK (team_slot IN (1, 2)),
    ADD CONSTRAINT match_players_abandon_reason_check CHECK (
        abandon_reason IS NULL
        OR abandon_reason IN ('explicit_leave', 'disconnect_timeout', 'account_ineligible')
    ),
    ADD CONSTRAINT match_players_abandon_lifecycle_check CHECK (
        (abandoned_at IS NULL AND abandon_reason IS NULL)
        OR (abandoned_at IS NOT NULL AND abandon_reason IS NOT NULL)
    );

CREATE INDEX match_players_match_team_status_user_idx
    ON match_players (match_id, team_slot, status, user_id);

-- ---------------------------------------------------------------------------
-- Game player team slot (nullable for non-matchmade games)
-- ---------------------------------------------------------------------------

ALTER TABLE game_players
    ADD COLUMN team_slot SMALLINT;

UPDATE game_players gp
SET team_slot = mp.team_slot
FROM match_players mp
WHERE mp.game_player_id = gp.id;

ALTER TABLE game_players
    ADD CONSTRAINT game_players_team_slot_check CHECK (
        team_slot IS NULL OR team_slot IN (1, 2)
    );

CREATE INDEX game_players_game_team_status_id_idx
    ON game_players (game_id, team_slot, status, id);

-- Expand games.mode to accept casual/ranked team mode values used when
-- matchmade games are created with the match playlist mode.
ALTER TABLE games DROP CONSTRAINT IF EXISTS games_mode_check;
ALTER TABLE games ADD CONSTRAINT games_mode_check CHECK (
  mode IN (
    'solo', 'private_room', 'quick_play', 'daily', 'ranked',
    'casual_solo', 'casual_duo', 'casual_squad',
    'ranked_solo', 'ranked_duo', 'ranked_squad'
  )
);

-- ---------------------------------------------------------------------------
-- Guess accuracy / speed bonus extensions
-- ---------------------------------------------------------------------------

ALTER TABLE guesses
    ADD COLUMN accuracy_score INT,
    ADD COLUMN speed_bonus INT NOT NULL DEFAULT 0;

UPDATE guesses
SET
    accuracy_score = score,
    speed_bonus = 0;

ALTER TABLE guesses
    ALTER COLUMN accuracy_score SET NOT NULL;

ALTER TABLE guesses DROP CONSTRAINT guesses_score_range;

ALTER TABLE guesses
    ADD CONSTRAINT guesses_score_range CHECK (score BETWEEN 0 AND 5250),
    ADD CONSTRAINT guesses_accuracy_score_range CHECK (accuracy_score BETWEEN 0 AND 5000),
    ADD CONSTRAINT guesses_speed_bonus_range CHECK (speed_bonus BETWEEN 0 AND 250),
    ADD CONSTRAINT guesses_score_components_check CHECK (score = accuracy_score + speed_bonus);

-- ---------------------------------------------------------------------------
-- Team messages + moderation
-- ---------------------------------------------------------------------------

CREATE TABLE team_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id UUID NOT NULL REFERENCES matches(id) ON DELETE RESTRICT,
    team_slot SMALLINT NOT NULL,
    author_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    client_message_id UUID NOT NULL,
    sequence BIGSERIAL NOT NULL,
    text TEXT NOT NULL DEFAULT '',
    file_id UUID REFERENCES files(id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT team_messages_team_slot_check CHECK (team_slot IN (1, 2)),
    CONSTRAINT team_messages_text_length_check CHECK (char_length(text) BETWEEN 0 AND 500),
    CONSTRAINT team_messages_content_present_check CHECK (
        char_length(btrim(text)) > 0 OR file_id IS NOT NULL
    ),
    CONSTRAINT team_messages_status_check CHECK (
        status IN ('active', 'reported', 'removed', 'expired')
    ),
    CONSTRAINT team_messages_client_message_unique UNIQUE (match_id, author_user_id, client_message_id),
    CONSTRAINT team_messages_file_id_unique UNIQUE (file_id)
);

CREATE INDEX team_messages_match_team_sequence_idx
    ON team_messages (match_id, team_slot, sequence DESC);

CREATE INDEX team_messages_expires_status_idx
    ON team_messages (expires_at, status);

CREATE TABLE match_mutes (
    match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    muter_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    muted_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (match_id, muter_user_id, muted_user_id),
    CONSTRAINT match_mutes_not_self_check CHECK (muter_user_id <> muted_user_id)
);

CREATE TABLE team_message_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES team_messages(id) ON DELETE RESTRICT,
    match_id UUID NOT NULL REFERENCES matches(id) ON DELETE RESTRICT,
    reporter_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    reason TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    retention_until TIMESTAMPTZ NOT NULL,
    legal_hold BOOLEAN NOT NULL DEFAULT false,
    CONSTRAINT team_message_reports_reason_check CHECK (
        reason IN ('harassment', 'hate', 'sexual', 'violent', 'spam', 'other')
    ),
    CONSTRAINT team_message_reports_status_check CHECK (
        status IN ('pending', 'reviewed', 'dismissed', 'actioned')
    ),
    CONSTRAINT team_message_reports_reporter_message_unique UNIQUE (message_id, reporter_user_id)
);

CREATE INDEX team_message_reports_match_status_idx
    ON team_message_reports (match_id, status, created_at DESC);

CREATE INDEX team_message_reports_retention_idx
    ON team_message_reports (retention_until, legal_hold);

-- ---------------------------------------------------------------------------
-- Upload / file purpose-bound team-chat extensions
-- ---------------------------------------------------------------------------

ALTER TABLE uploads
    ADD COLUMN purpose TEXT,
    ADD COLUMN context_id UUID,
    ADD COLUMN sanitization_status TEXT,
    ADD COLUMN raw_storage_key TEXT;

UPDATE uploads
SET
    purpose = 'general',
    sanitization_status = 'not_required';

ALTER TABLE uploads
    ALTER COLUMN purpose SET NOT NULL,
    ALTER COLUMN sanitization_status SET NOT NULL;

ALTER TABLE uploads
    ADD CONSTRAINT uploads_purpose_check CHECK (purpose IN ('general', 'team_chat')),
    ADD CONSTRAINT uploads_sanitization_status_check CHECK (
        sanitization_status IN ('not_required', 'pending', 'ready', 'rejected')
    ),
    ADD CONSTRAINT uploads_team_chat_context_check CHECK (
        (purpose = 'general')
        OR (purpose = 'team_chat' AND context_id IS NOT NULL)
    );

CREATE INDEX uploads_purpose_context_idx
    ON uploads (purpose, context_id)
    WHERE context_id IS NOT NULL;

ALTER TABLE files
    ADD COLUMN purpose TEXT,
    ADD COLUMN context_id UUID,
    ADD COLUMN sanitization_status TEXT,
    ADD COLUMN raw_storage_key TEXT;

UPDATE files
SET
    purpose = 'general',
    sanitization_status = 'not_required';

ALTER TABLE files
    ALTER COLUMN purpose SET NOT NULL,
    ALTER COLUMN sanitization_status SET NOT NULL;

ALTER TABLE files
    ADD CONSTRAINT files_purpose_check CHECK (purpose IN ('general', 'team_chat')),
    ADD CONSTRAINT files_sanitization_status_check CHECK (
        sanitization_status IN ('not_required', 'pending', 'ready', 'rejected')
    ),
    ADD CONSTRAINT files_team_chat_context_check CHECK (
        (purpose = 'general')
        OR (purpose = 'team_chat' AND context_id IS NOT NULL)
    );

CREATE INDEX files_purpose_context_idx
    ON files (purpose, context_id)
    WHERE context_id IS NOT NULL;

-- +goose Down
-- Guarded down: only safe before party/competitive/chat history or non-legacy
-- modes exist. Never blindly delete match/rating/moderation history.

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM parties LIMIT 1)
        OR EXISTS (SELECT 1 FROM party_members LIMIT 1)
        OR EXISTS (SELECT 1 FROM party_invites LIMIT 1)
        OR EXISTS (SELECT 1 FROM competitive_standings LIMIT 1)
        OR EXISTS (SELECT 1 FROM competitive_rating_changes LIMIT 1)
        OR EXISTS (SELECT 1 FROM team_messages LIMIT 1)
        OR EXISTS (SELECT 1 FROM match_mutes LIMIT 1)
        OR EXISTS (SELECT 1 FROM team_message_reports LIMIT 1)
        OR EXISTS (SELECT 1 FROM matches WHERE mode <> 'ranked_standard' LIMIT 1)
    THEN
        RAISE EXCEPTION
            'migration 00019 is irreversible: party/competitive/team_message rows or non-legacy match modes exist';
    END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS files_purpose_context_idx;
ALTER TABLE files
    DROP CONSTRAINT IF EXISTS files_team_chat_context_check,
    DROP CONSTRAINT IF EXISTS files_sanitization_status_check,
    DROP CONSTRAINT IF EXISTS files_purpose_check;
ALTER TABLE files
    DROP COLUMN IF EXISTS raw_storage_key,
    DROP COLUMN IF EXISTS sanitization_status,
    DROP COLUMN IF EXISTS context_id,
    DROP COLUMN IF EXISTS purpose;

DROP INDEX IF EXISTS uploads_purpose_context_idx;
ALTER TABLE uploads
    DROP CONSTRAINT IF EXISTS uploads_team_chat_context_check,
    DROP CONSTRAINT IF EXISTS uploads_sanitization_status_check,
    DROP CONSTRAINT IF EXISTS uploads_purpose_check;
ALTER TABLE uploads
    DROP COLUMN IF EXISTS raw_storage_key,
    DROP COLUMN IF EXISTS sanitization_status,
    DROP COLUMN IF EXISTS context_id,
    DROP COLUMN IF EXISTS purpose;

DROP INDEX IF EXISTS team_message_reports_retention_idx;
DROP INDEX IF EXISTS team_message_reports_match_status_idx;
DROP TABLE IF EXISTS team_message_reports;
DROP TABLE IF EXISTS match_mutes;
DROP INDEX IF EXISTS team_messages_expires_status_idx;
DROP INDEX IF EXISTS team_messages_match_team_sequence_idx;
DROP TABLE IF EXISTS team_messages;

ALTER TABLE guesses
    DROP CONSTRAINT IF EXISTS guesses_score_components_check,
    DROP CONSTRAINT IF EXISTS guesses_speed_bonus_range,
    DROP CONSTRAINT IF EXISTS guesses_accuracy_score_range,
    DROP CONSTRAINT IF EXISTS guesses_score_range;
ALTER TABLE guesses
    DROP COLUMN IF EXISTS speed_bonus,
    DROP COLUMN IF EXISTS accuracy_score;
ALTER TABLE guesses
    ADD CONSTRAINT guesses_score_range CHECK (score BETWEEN 0 AND 5000);

DROP INDEX IF EXISTS game_players_game_team_status_id_idx;
ALTER TABLE game_players
    DROP CONSTRAINT IF EXISTS game_players_team_slot_check;
ALTER TABLE game_players
    DROP COLUMN IF EXISTS team_slot;

-- Restore original games.mode constraint (guard above rejects non-legacy match
-- modes; game rows using casual/ranked team modes would block this down path).
ALTER TABLE games DROP CONSTRAINT IF EXISTS games_mode_check;
ALTER TABLE games ADD CONSTRAINT games_mode_check CHECK (
  mode IN ('solo', 'private_room', 'quick_play', 'daily', 'ranked')
);

DROP INDEX IF EXISTS match_players_match_team_status_user_idx;
ALTER TABLE match_players
    DROP CONSTRAINT IF EXISTS match_players_abandon_lifecycle_check,
    DROP CONSTRAINT IF EXISTS match_players_abandon_reason_check,
    DROP CONSTRAINT IF EXISTS match_players_team_slot_check;
ALTER TABLE match_players
    DROP COLUMN IF EXISTS abandon_reason,
    DROP COLUMN IF EXISTS abandoned_at,
    DROP COLUMN IF EXISTS party_id,
    DROP COLUMN IF EXISTS team_slot;

DROP INDEX IF EXISTS matches_status_last_activity_idx;
DROP INDEX IF EXISTS matches_season_status_completed_idx;
DROP INDEX IF EXISTS matches_playlist_format_status_matched_idx;
ALTER TABLE matches
    DROP CONSTRAINT IF EXISTS matches_result_check,
    DROP CONSTRAINT IF EXISTS matches_winner_team_slot_check,
    DROP CONSTRAINT IF EXISTS matches_team_scores_non_negative,
    DROP CONSTRAINT IF EXISTS matches_mode_playlist_format_check,
    DROP CONSTRAINT IF EXISTS matches_season_playlist_check,
    DROP CONSTRAINT IF EXISTS matches_team_size_format_check,
    DROP CONSTRAINT IF EXISTS matches_format_check,
    DROP CONSTRAINT IF EXISTS matches_playlist_check;
ALTER TABLE matches
    DROP COLUMN IF EXISTS progression_finalized_at,
    DROP COLUMN IF EXISTS chat_access_until,
    DROP COLUMN IF EXISTS last_activity_at,
    DROP COLUMN IF EXISTS result,
    DROP COLUMN IF EXISTS winner_team_slot,
    DROP COLUMN IF EXISTS team_two_score,
    DROP COLUMN IF EXISTS team_one_score,
    DROP COLUMN IF EXISTS season_id,
    DROP COLUMN IF EXISTS team_size,
    DROP COLUMN IF EXISTS format,
    DROP COLUMN IF EXISTS playlist;
ALTER TABLE matches DROP CONSTRAINT IF EXISTS matches_mode_check;
ALTER TABLE matches ADD CONSTRAINT matches_mode_check CHECK (mode IN ('ranked_standard'));

DROP INDEX IF EXISTS competitive_rating_changes_match_idx;
DROP INDEX IF EXISTS competitive_rating_changes_season_user_created_idx;
DROP TABLE IF EXISTS competitive_rating_changes;
DROP INDEX IF EXISTS competitive_standings_top500_idx;
DROP TABLE IF EXISTS competitive_standings;
DROP INDEX IF EXISTS competitive_seasons_one_active_uidx;
DROP TABLE IF EXISTS competitive_seasons;

DROP INDEX IF EXISTS party_invites_invitee_status_idx;
DROP INDEX IF EXISTS party_invites_pending_unique;
DROP TABLE IF EXISTS party_invites;
DROP INDEX IF EXISTS party_members_party_status_joined_idx;
DROP INDEX IF EXISTS party_members_active_user_uidx;
DROP TABLE IF EXISTS party_members;
DROP INDEX IF EXISTS parties_active_match_id_idx;
DROP INDEX IF EXISTS parties_status_updated_at_idx;
DROP INDEX IF EXISTS parties_leader_status_idx;
DROP TABLE IF EXISTS parties;
