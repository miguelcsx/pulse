-- +goose Up

ALTER TABLE asks ADD COLUMN anonymous BOOLEAN NOT NULL DEFAULT false;

-- Browsing the Commons filters public asks that have at least one response,
-- ordered by recency — index the visibility lookup.
CREATE INDEX idx_asks_visibility_created_at ON asks (visibility, created_at DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_asks_visibility_created_at;
ALTER TABLE asks DROP COLUMN IF EXISTS anonymous;
