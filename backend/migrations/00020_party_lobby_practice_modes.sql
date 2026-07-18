-- +goose Up
ALTER TABLE games DROP CONSTRAINT IF EXISTS games_mode_check;
ALTER TABLE games ADD CONSTRAINT games_mode_check CHECK (
  mode IN (
    'solo', 'private_room', 'party_lobby', 'practice', 'quick_play', 'daily', 'ranked',
    'casual_solo', 'casual_duo', 'casual_squad',
    'ranked_solo', 'ranked_duo', 'ranked_squad'
  )
);

ALTER TABLE games DROP CONSTRAINT IF EXISTS games_round_count_range;
ALTER TABLE games ADD CONSTRAINT games_round_count_range CHECK (
  (mode = 'practice' AND round_count >= 1)
  OR
  (mode <> 'practice' AND round_count BETWEEN 1 AND 10)
);

ALTER TABLE rounds ADD COLUMN creation_idempotency_key TEXT;
CREATE UNIQUE INDEX rounds_game_creation_idempotency_uidx
  ON rounds (game_id, creation_idempotency_key)
  WHERE creation_idempotency_key IS NOT NULL;

-- +goose Down
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM games WHERE mode IN ('party_lobby', 'practice')) THEN
    RAISE EXCEPTION 'migration 00020 is irreversible while party_lobby or practice games exist';
  END IF;
END $$;

DROP INDEX IF EXISTS rounds_game_creation_idempotency_uidx;
ALTER TABLE rounds DROP COLUMN IF EXISTS creation_idempotency_key;

ALTER TABLE games DROP CONSTRAINT IF EXISTS games_round_count_range;
ALTER TABLE games ADD CONSTRAINT games_round_count_range CHECK (round_count BETWEEN 1 AND 10);

ALTER TABLE games DROP CONSTRAINT IF EXISTS games_mode_check;
ALTER TABLE games ADD CONSTRAINT games_mode_check CHECK (
  mode IN (
    'solo', 'private_room', 'quick_play', 'daily', 'ranked',
    'casual_solo', 'casual_duo', 'casual_squad',
    'ranked_solo', 'ranked_duo', 'ranked_squad'
  )
);
