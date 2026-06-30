-- +goose Up
CREATE TABLE challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL CHECK (type IN ('daily', 'shared')),
    slug_or_code TEXT,
    seed TEXT NOT NULL,
    challenge_date DATE,
    reset_starts_at TIMESTAMPTZ,
    reset_ends_at TIMESTAMPTZ,
    map_id UUID NOT NULL REFERENCES maps(id),
    settings_snapshot JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('draft', 'active', 'completed', 'archived', 'unavailable')),
    created_by_user_id UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT daily_challenge_requires_date CHECK (type <> 'daily' OR challenge_date IS NOT NULL),
    CONSTRAINT shared_challenge_requires_code CHECK (type <> 'shared' OR slug_or_code IS NOT NULL)
);

CREATE UNIQUE INDEX challenges_daily_date_key
    ON challenges (challenge_date)
    WHERE type = 'daily';

CREATE UNIQUE INDEX challenges_shared_code_key
    ON challenges (slug_or_code)
    WHERE type = 'shared' AND slug_or_code IS NOT NULL;

CREATE INDEX challenges_lookup_idx ON challenges (type, status, reset_starts_at, reset_ends_at);
CREATE INDEX challenges_map_idx ON challenges (map_id);

CREATE TABLE challenge_locations (
    challenge_id UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    round_number INT NOT NULL CHECK (round_number > 0),
    location_id UUID NOT NULL REFERENCES locations(id),
    selection_version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (challenge_id, round_number),
    UNIQUE (challenge_id, location_id)
);

CREATE INDEX challenge_locations_location_idx ON challenge_locations (location_id);

CREATE TABLE challenge_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    game_id UUID REFERENCES games(id),
    user_id UUID REFERENCES users(id),
    guest_identity_hash TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'completed', 'abandoned', 'expired')),
    leaderboard_eligible BOOLEAN NOT NULL DEFAULT false,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    total_score INT NOT NULL DEFAULT 0,
    total_distance_meters INT NOT NULL DEFAULT 0,
    completion_duration_ms BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT challenge_attempt_owner_check CHECK ((user_id IS NULL) <> (guest_identity_hash IS NULL))
);

CREATE INDEX challenge_attempts_challenge_idx ON challenge_attempts (challenge_id, status);
CREATE INDEX challenge_attempts_user_idx ON challenge_attempts (user_id) WHERE user_id IS NOT NULL;
CREATE INDEX challenge_attempts_guest_idx ON challenge_attempts (guest_identity_hash) WHERE guest_identity_hash IS NOT NULL;
CREATE UNIQUE INDEX challenge_attempts_user_challenge_key ON challenge_attempts (challenge_id, user_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX challenge_attempts_guest_challenge_key ON challenge_attempts (challenge_id, guest_identity_hash) WHERE guest_identity_hash IS NOT NULL;

-- Rollback/readiness: down migration drops attempts, locations, then challenges.

-- +goose Down
DROP TABLE IF EXISTS challenge_attempts;
DROP TABLE IF EXISTS challenge_locations;
DROP TABLE IF EXISTS challenges;
