-- +goose Up
ALTER TABLE user_profiles
    ADD COLUMN experience_points BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN level INT NOT NULL DEFAULT 1,
    ADD CONSTRAINT user_profiles_experience_points_nonnegative CHECK (experience_points >= 0),
    ADD CONSTRAINT user_profiles_level_positive CHECK (level >= 1);

ALTER TABLE challenge_attempts
    ADD COLUMN awarded_xp INT NOT NULL DEFAULT 0,
    ADD COLUMN daily_game_number INT NOT NULL DEFAULT 1,
    ADD CONSTRAINT challenge_attempts_awarded_xp_nonnegative CHECK (awarded_xp >= 0);

ALTER TABLE challenge_attempts
    ADD CONSTRAINT challenge_attempts_daily_game_number_range CHECK (daily_game_number BETWEEN 1 AND 5);

DROP INDEX challenge_attempts_user_challenge_key;
DROP INDEX challenge_attempts_guest_challenge_key;
CREATE UNIQUE INDEX challenge_attempts_user_challenge_game_key ON challenge_attempts (challenge_id, user_id, daily_game_number) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX challenge_attempts_guest_challenge_game_key ON challenge_attempts (challenge_id, guest_identity_hash, daily_game_number) WHERE guest_identity_hash IS NOT NULL;

ALTER TABLE guesses
    ADD COLUMN timed_out BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
-- Irreversible after multiple attempts exist for one daily challenge.
DO $$ BEGIN RAISE EXCEPTION '00016 is irreversible after daily progression data exists'; END $$;
