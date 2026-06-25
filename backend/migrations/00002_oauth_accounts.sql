-- +goose Up

-- OAuth-only accounts may not have a local password or a local email.
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;
ALTER TABLE users ALTER COLUMN email DROP NOT NULL;

CREATE TABLE user_oauth_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    provider_account_id TEXT NOT NULL,
    email TEXT,
    display_name TEXT,
    avatar_url TEXT,
    access_token TEXT,
    refresh_token TEXT,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT user_oauth_accounts_provider_account_id_key UNIQUE (provider, provider_account_id)
);

CREATE TRIGGER user_oauth_accounts_updated_at BEFORE UPDATE ON user_oauth_accounts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX user_oauth_accounts_user_id_provider ON user_oauth_accounts (user_id, provider);

-- Partial index to enforce one OAuth account per provider per user.
CREATE UNIQUE INDEX user_oauth_accounts_user_provider_unique ON user_oauth_accounts (user_id, provider);

-- +goose Down

DROP TABLE IF EXISTS user_oauth_accounts CASCADE;

ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;
ALTER TABLE users ALTER COLUMN email SET NOT NULL;
