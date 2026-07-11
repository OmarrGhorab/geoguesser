-- +goose Up
-- Friends social graph: sorted-pair friendships with pending/accepted/blocked lifecycle.

CREATE TABLE friendships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_a_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    user_b_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    requested_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status TEXT NOT NULL,
    blocked_by_user_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT friendships_distinct_users_check CHECK (user_a_id <> user_b_id),
    CONSTRAINT friendships_sorted_pair_check CHECK (user_a_id < user_b_id),
    CONSTRAINT friendships_pair_unique UNIQUE (user_a_id, user_b_id),
    CONSTRAINT friendships_status_check CHECK (status IN ('pending', 'accepted', 'blocked')),
    CONSTRAINT friendships_requester_in_pair_check CHECK (
        requested_by_user_id = user_a_id OR requested_by_user_id = user_b_id
    ),
    CONSTRAINT friendships_blocker_in_pair_check CHECK (
        blocked_by_user_id IS NULL
        OR blocked_by_user_id = user_a_id
        OR blocked_by_user_id = user_b_id
    ),
    CONSTRAINT friendships_lifecycle_check CHECK (
        (
            status = 'pending'
            AND accepted_at IS NULL
            AND blocked_by_user_id IS NULL
        )
        OR (
            status = 'accepted'
            AND accepted_at IS NOT NULL
            AND blocked_by_user_id IS NULL
        )
        OR (
            status = 'blocked'
            AND blocked_by_user_id IS NOT NULL
            AND accepted_at IS NULL
        )
    )
);

CREATE TRIGGER friendships_updated_at BEFORE UPDATE ON friendships
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX friendships_user_a_status_created_idx
    ON friendships (user_a_id, status, created_at DESC, id DESC);

CREATE INDEX friendships_user_b_status_created_idx
    ON friendships (user_b_id, status, created_at DESC, id DESC);

CREATE INDEX friendships_requested_by_status_created_idx
    ON friendships (requested_by_user_id, status, created_at DESC, id DESC);

CREATE INDEX friendships_blocked_by_status_created_idx
    ON friendships (blocked_by_user_id, status, created_at DESC, id DESC)
    WHERE status = 'blocked';

-- +goose Down
DROP TABLE IF EXISTS friendships;
