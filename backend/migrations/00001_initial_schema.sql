-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "citext";
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';
-- +goose StatementEnd

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email CITEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user',
    status TEXT NOT NULL DEFAULT 'pending_verification',
    email_verified_at TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT users_email_key UNIQUE (email),
    CONSTRAINT users_role_check CHECK (role IN ('user', 'moderator', 'admin')),
    CONSTRAINT users_status_check CHECK (status IN ('active', 'disabled', 'pending_verification', 'deleted'))
);

CREATE TRIGGER users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX users_created_at_id_desc ON users (created_at DESC, id DESC);

CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name TEXT NOT NULL,
    avatar_url TEXT,
    country_code TEXT,
    locale TEXT NOT NULL DEFAULT 'en',
    timezone TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT user_profiles_display_name_length CHECK (char_length(display_name) BETWEEN 2 AND 32),
    CONSTRAINT user_profiles_country_code_length CHECK (country_code IS NULL OR char_length(country_code) = 2)
);

CREATE TRIGGER user_profiles_updated_at BEFORE UPDATE ON user_profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE auth_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash TEXT NOT NULL,
    user_agent_hash TEXT,
    ip_address INET,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ,
    CONSTRAINT auth_sessions_refresh_token_hash_key UNIQUE (refresh_token_hash),
    CONSTRAINT auth_sessions_expires_after_created CHECK (expires_at > created_at)
);

CREATE INDEX auth_sessions_user_id_expires_at ON auth_sessions (user_id, expires_at DESC);

CREATE TABLE maps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    visibility TEXT NOT NULL DEFAULT 'public',
    access_tier TEXT NOT NULL DEFAULT 'free',
    difficulty TEXT NOT NULL DEFAULT 'mixed',
    status TEXT NOT NULL DEFAULT 'draft',
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT maps_slug_key UNIQUE (slug),
    CONSTRAINT maps_visibility_check CHECK (visibility IN ('public', 'private', 'unlisted')),
    CONSTRAINT maps_access_tier_check CHECK (access_tier IN ('free', 'premium', 'admin')),
    CONSTRAINT maps_difficulty_check CHECK (difficulty IN ('mixed', 'easy', 'medium', 'hard')),
    CONSTRAINT maps_status_check CHECK (status IN ('draft', 'active', 'archived'))
);

CREATE TRIGGER maps_updated_at BEFORE UPDATE ON maps
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX maps_status_visibility_access_tier ON maps (status, visibility, access_tier);

CREATE TABLE locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    latitude NUMERIC(9,6) NOT NULL,
    longitude NUMERIC(9,6) NOT NULL,
    country_code TEXT NOT NULL,
    region TEXT,
    locality TEXT,
    difficulty TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_ref TEXT NOT NULL,
    attribution TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    random_key NUMERIC NOT NULL DEFAULT random(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT locations_latitude_range CHECK (latitude BETWEEN -90 AND 90),
    CONSTRAINT locations_longitude_range CHECK (longitude BETWEEN -180 AND 180),
    CONSTRAINT locations_difficulty_check CHECK (difficulty IN ('easy', 'medium', 'hard', 'expert')),
    CONSTRAINT locations_status_check CHECK (status IN ('active', 'disabled', 'needs_review')),
    CONSTRAINT locations_provider_provider_ref_key UNIQUE (provider, provider_ref)
);

CREATE TRIGGER locations_updated_at BEFORE UPDATE ON locations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX locations_selection ON locations (status, country_code, difficulty, random_key);

CREATE TABLE map_locations (
    map_id UUID NOT NULL REFERENCES maps(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
    selection_weight INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (map_id, location_id),
    CONSTRAINT map_locations_selection_weight_positive CHECK (selection_weight > 0)
);

CREATE INDEX map_locations_location_id ON map_locations (location_id);

CREATE TABLE games (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mode TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    map_id UUID NOT NULL REFERENCES maps(id),
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    round_count INT NOT NULL DEFAULT 5,
    timer_seconds INT,
    scoring_version INT NOT NULL DEFAULT 1,
    total_score INT NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT games_mode_check CHECK (mode IN ('solo', 'private_room', 'quick_play', 'daily', 'ranked')),
    CONSTRAINT games_status_check CHECK (status IN ('pending', 'active', 'completed', 'abandoned', 'cancelled')),
    CONSTRAINT games_round_count_range CHECK (round_count BETWEEN 1 AND 10),
    CONSTRAINT games_timer_seconds_range CHECK (timer_seconds IS NULL OR timer_seconds BETWEEN 10 AND 600),
    CONSTRAINT games_total_score_non_negative CHECK (total_score >= 0)
);

CREATE TRIGGER games_updated_at BEFORE UPDATE ON games
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX games_created_by_user_id_created_at ON games (created_by_user_id, created_at DESC, id DESC);
CREATE INDEX games_status_created_at ON games (status, created_at DESC);

CREATE TABLE rounds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES locations(id),
    round_number INT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    revealed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT rounds_game_round_number_key UNIQUE (game_id, round_number),
    CONSTRAINT rounds_round_number_positive CHECK (round_number > 0),
    CONSTRAINT rounds_status_check CHECK (status IN ('pending', 'active', 'completed', 'cancelled'))
);

CREATE INDEX rounds_location_id ON rounds (location_id);

CREATE TABLE game_players (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    guest_identity_hash TEXT,
    display_name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'player',
    status TEXT NOT NULL DEFAULT 'active',
    total_score INT NOT NULL DEFAULT 0,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    left_at TIMESTAMPTZ,
    CONSTRAINT game_players_role_check CHECK (role IN ('host', 'player', 'spectator')),
    CONSTRAINT game_players_status_check CHECK (status IN ('active', 'disconnected', 'left', 'kicked')),
    CONSTRAINT game_players_identity_present CHECK (user_id IS NOT NULL OR guest_identity_hash IS NOT NULL),
    CONSTRAINT game_players_total_score_non_negative CHECK (total_score >= 0)
);

CREATE UNIQUE INDEX game_players_game_user_unique ON game_players (game_id, user_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX game_players_game_guest_unique ON game_players (game_id, guest_identity_hash) WHERE guest_identity_hash IS NOT NULL;
CREATE INDEX game_players_game_id_status ON game_players (game_id, status);

CREATE TABLE guesses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    round_id UUID NOT NULL REFERENCES rounds(id) ON DELETE CASCADE,
    game_player_id UUID NOT NULL REFERENCES game_players(id) ON DELETE CASCADE,
    latitude NUMERIC(9,6) NOT NULL,
    longitude NUMERIC(9,6) NOT NULL,
    distance_meters INT NOT NULL,
    score INT NOT NULL,
    idempotency_key TEXT,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT guesses_round_player_unique UNIQUE (round_id, game_player_id),
    CONSTRAINT guesses_latitude_range CHECK (latitude BETWEEN -90 AND 90),
    CONSTRAINT guesses_longitude_range CHECK (longitude BETWEEN -180 AND 180),
    CONSTRAINT guesses_distance_non_negative CHECK (distance_meters >= 0),
    CONSTRAINT guesses_score_range CHECK (score BETWEEN 0 AND 5000)
);

CREATE INDEX guesses_game_player_id_submitted_at ON guesses (game_player_id, submitted_at DESC);
CREATE INDEX guesses_round_id_submitted_at ON guesses (round_id, submitted_at);

CREATE TABLE rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID UNIQUE REFERENCES games(id) ON DELETE SET NULL,
    code TEXT NOT NULL,
    visibility TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'lobby',
    host_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    max_players INT NOT NULL DEFAULT 8,
    round_count INT NOT NULL DEFAULT 5,
    timer_seconds INT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT rooms_visibility_check CHECK (visibility IN ('private', 'public')),
    CONSTRAINT rooms_status_check CHECK (status IN ('lobby', 'active', 'completed', 'expired', 'cancelled')),
    CONSTRAINT rooms_max_players_range CHECK (max_players BETWEEN 2 AND 50),
    CONSTRAINT rooms_round_count_range CHECK (round_count BETWEEN 1 AND 10),
    CONSTRAINT rooms_timer_seconds_range CHECK (timer_seconds IS NULL OR timer_seconds BETWEEN 10 AND 600)
);

CREATE TRIGGER rooms_updated_at BEFORE UPDATE ON rooms
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE UNIQUE INDEX rooms_code_active_unique ON rooms (code) WHERE status IN ('lobby', 'active');
CREATE INDEX rooms_host_user_id_created_at ON rooms (host_user_id, created_at DESC);
CREATE INDEX rooms_status_expires_at ON rooms (status, expires_at);

CREATE TABLE room_players (
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    game_player_id UUID NOT NULL REFERENCES game_players(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'joined',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    left_at TIMESTAMPTZ,
    PRIMARY KEY (room_id, game_player_id),
    CONSTRAINT room_players_status_check CHECK (status IN ('joined', 'left', 'kicked', 'disconnected'))
);

-- +goose Down
DROP TABLE IF EXISTS room_players CASCADE;
DROP TABLE IF EXISTS rooms CASCADE;
DROP TABLE IF EXISTS guesses CASCADE;
DROP TABLE IF EXISTS game_players CASCADE;
DROP TABLE IF EXISTS rounds CASCADE;
DROP TABLE IF EXISTS games CASCADE;
DROP TABLE IF EXISTS map_locations CASCADE;
DROP TABLE IF EXISTS locations CASCADE;
DROP TABLE IF EXISTS maps CASCADE;
DROP TABLE IF EXISTS auth_sessions CASCADE;
DROP TABLE IF EXISTS user_profiles CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- +goose StatementBegin
DROP EXTENSION IF EXISTS "citext";
DROP EXTENSION IF EXISTS "pgcrypto";
-- +goose StatementEnd
