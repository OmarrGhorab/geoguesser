-- +goose Up

CREATE TABLE uploads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    file_name TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    storage_key TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX uploads_owner_status ON uploads (owner_user_id, status);
CREATE INDEX uploads_expires_at ON uploads (expires_at);

CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id UUID UNIQUE REFERENCES uploads(id) ON DELETE SET NULL,
    owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    file_name TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    storage_key TEXT NOT NULL UNIQUE,
    is_public BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX files_owner_created_at ON files (owner_user_id, created_at DESC, id DESC);

-- +goose Down

DROP TABLE IF EXISTS files CASCADE;
DROP TABLE IF EXISTS uploads CASCADE;
