-- +goose Up
ALTER TABLE paths ADD COLUMN IF NOT EXISTS system_generated BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE paths DROP COLUMN IF EXISTS system_generated;
