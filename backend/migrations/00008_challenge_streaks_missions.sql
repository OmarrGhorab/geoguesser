-- +goose Up
CREATE TABLE streaks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID REFERENCES users(id),
    guest_identity_hash TEXT,
    current_count INT NOT NULL DEFAULT 0,
    best_count INT NOT NULL DEFAULT 0,
    last_completed_challenge_date DATE,
    status TEXT NOT NULL DEFAULT 'inactive' CHECK (status IN ('active', 'broken', 'protected', 'inactive')),
    protection_state TEXT NOT NULL DEFAULT 'none' CHECK (protection_state IN ('none', 'available', 'consumed', 'expired')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT streak_owner_check CHECK ((owner_user_id IS NULL) <> (guest_identity_hash IS NULL))
);

CREATE UNIQUE INDEX streaks_user_key ON streaks (owner_user_id) WHERE owner_user_id IS NOT NULL;
CREATE UNIQUE INDEX streaks_guest_key ON streaks (guest_identity_hash) WHERE guest_identity_hash IS NOT NULL;

CREATE TABLE streak_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID REFERENCES users(id),
    guest_identity_hash TEXT,
    challenge_id UUID REFERENCES challenges(id) ON DELETE SET NULL,
    challenge_date DATE NOT NULL,
    event_type TEXT NOT NULL,
    previous_count INT NOT NULL,
    new_count INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT streak_event_owner_check CHECK ((owner_user_id IS NULL) <> (guest_identity_hash IS NULL))
);

CREATE UNIQUE INDEX streak_events_user_day_type_key ON streak_events (owner_user_id, challenge_date, event_type) WHERE owner_user_id IS NOT NULL;
CREATE UNIQUE INDEX streak_events_guest_day_type_key ON streak_events (guest_identity_hash, challenge_date, event_type) WHERE guest_identity_hash IS NOT NULL;

CREATE TABLE missions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    title_key TEXT NOT NULL,
    description_key TEXT NOT NULL,
    mission_type TEXT NOT NULL,
    target_value INT NOT NULL,
    active_starts_at TIMESTAMPTZ NOT NULL,
    active_ends_at TIMESTAMPTZ,
    reward_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('draft', 'active', 'archived')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX missions_active_idx ON missions (status, active_starts_at, active_ends_at);

CREATE TABLE mission_progress (
    mission_id UUID NOT NULL REFERENCES missions(id) ON DELETE CASCADE,
    owner_user_id UUID REFERENCES users(id),
    guest_identity_hash TEXT,
    current_value INT NOT NULL DEFAULT 0,
    target_value INT NOT NULL,
    status TEXT NOT NULL DEFAULT 'not_started' CHECK (status IN ('not_started', 'in_progress', 'completed', 'claimed', 'expired')),
    completed_at TIMESTAMPTZ,
    claimed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT mission_progress_owner_check CHECK ((owner_user_id IS NULL) <> (guest_identity_hash IS NULL))
);

CREATE UNIQUE INDEX mission_progress_user_key ON mission_progress (mission_id, owner_user_id) WHERE owner_user_id IS NOT NULL;
CREATE UNIQUE INDEX mission_progress_guest_key ON mission_progress (mission_id, guest_identity_hash) WHERE guest_identity_hash IS NOT NULL;

CREATE TABLE mission_progress_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mission_id UUID NOT NULL REFERENCES missions(id) ON DELETE CASCADE,
    owner_user_id UUID REFERENCES users(id),
    guest_identity_hash TEXT,
    source_attempt_id UUID REFERENCES challenge_attempts(id) ON DELETE SET NULL,
    source_challenge_id UUID REFERENCES challenges(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    delta INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT mission_progress_event_owner_check CHECK ((owner_user_id IS NULL) <> (guest_identity_hash IS NULL))
);

CREATE UNIQUE INDEX mission_events_attempt_user_key ON mission_progress_events (mission_id, owner_user_id, source_attempt_id, event_type) WHERE owner_user_id IS NOT NULL AND source_attempt_id IS NOT NULL;
CREATE UNIQUE INDEX mission_events_attempt_guest_key ON mission_progress_events (mission_id, guest_identity_hash, source_attempt_id, event_type) WHERE guest_identity_hash IS NOT NULL AND source_attempt_id IS NOT NULL;

-- Rollback/readiness: down migration removes mission and streak progression tables.

-- +goose Down
DROP TABLE IF EXISTS mission_progress_events;
DROP TABLE IF EXISTS mission_progress;
DROP TABLE IF EXISTS missions;
DROP TABLE IF EXISTS streak_events;
DROP TABLE IF EXISTS streaks;
