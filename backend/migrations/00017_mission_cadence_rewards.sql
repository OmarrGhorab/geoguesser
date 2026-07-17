-- +goose Up
ALTER TABLE missions
    ADD COLUMN mission_key TEXT NOT NULL DEFAULT 'legacy',
    ADD COLUMN cadence TEXT NOT NULL DEFAULT 'daily' CHECK (cadence IN ('daily', 'weekly')),
    ADD COLUMN period_key TEXT NOT NULL DEFAULT 'legacy',
    ADD COLUMN icon_key TEXT NOT NULL DEFAULT 'daily-completion',
    ADD COLUMN reward_xp INT NOT NULL DEFAULT 0 CHECK (reward_xp >= 0);

-- Legacy records did not have a stable period or reward semantics. Keep their audit history but never serve them.
UPDATE missions SET status = 'archived' WHERE mission_key = 'legacy';
CREATE INDEX missions_active_type_idx ON missions (mission_type, status, active_starts_at, active_ends_at);

-- +goose Down
DROP INDEX IF EXISTS missions_active_type_idx;

-- Before this migration every mission was a legacy record. Restore the
-- pre-migration serving state before removing the discriminator columns.
UPDATE missions SET status = 'active' WHERE mission_key = 'legacy' AND status = 'archived';

ALTER TABLE missions
    DROP COLUMN IF EXISTS reward_xp,
    DROP COLUMN IF EXISTS icon_key,
    DROP COLUMN IF EXISTS period_key,
    DROP COLUMN IF EXISTS cadence,
    DROP COLUMN IF EXISTS mission_key;
