-- +goose Up
-- Add a preferences column to user_profiles so registered users can persist
-- profile-scoped settings (e.g. UI preferences) alongside their profile.

ALTER TABLE user_profiles
    ADD COLUMN preferences JSONB NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down

ALTER TABLE user_profiles
    DROP COLUMN preferences;
