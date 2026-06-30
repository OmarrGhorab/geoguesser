-- +goose Up
CREATE TABLE challenge_results (
    attempt_id UUID PRIMARY KEY REFERENCES challenge_attempts(id) ON DELETE CASCADE,
    challenge_id UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    total_score INT NOT NULL,
    total_distance_meters INT NOT NULL,
    round_results_snapshot JSONB NOT NULL,
    rank_snapshot JSONB,
    completed_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX challenge_results_challenge_idx ON challenge_results (challenge_id, completed_at DESC);

CREATE TABLE leaderboard_entries (
    challenge_id UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    attempt_id UUID NOT NULL REFERENCES challenge_attempts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    display_name_snapshot TEXT NOT NULL,
    score INT NOT NULL,
    completion_duration_ms BIGINT,
    completed_at TIMESTAMPTZ NOT NULL,
    rank INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (challenge_id, attempt_id)
);

CREATE UNIQUE INDEX leaderboard_entries_attempt_key ON leaderboard_entries (attempt_id);
CREATE INDEX leaderboard_entries_rank_idx ON leaderboard_entries (challenge_id, rank, attempt_id);
CREATE INDEX leaderboard_entries_order_idx ON leaderboard_entries (challenge_id, score DESC, completion_duration_ms ASC, completed_at ASC, attempt_id ASC);

-- Rollback/readiness: down migration removes derived leaderboard/result facts only.

-- +goose Down
DROP TABLE IF EXISTS leaderboard_entries;
DROP TABLE IF EXISTS challenge_results;
