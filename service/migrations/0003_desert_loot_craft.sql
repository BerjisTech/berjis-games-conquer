ALTER TABLE player_profiles
  ADD COLUMN IF NOT EXISTS last_desert_at TIMESTAMPTZ NULL;

CREATE TABLE IF NOT EXISTS craft_orders (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  player_id TEXT NOT NULL,
  recipe_id TEXT NOT NULL,
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completes_at TIMESTAMPTZ NOT NULL,
  status TEXT NOT NULL DEFAULT 'queued'
);

CREATE INDEX IF NOT EXISTS idx_craft_orders_player ON craft_orders(player_id);
CREATE INDEX IF NOT EXISTS idx_craft_orders_due ON craft_orders(status, completes_at);
