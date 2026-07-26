-- +goose Up
ALTER TABLE conversations ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0;
ALTER TABLE conversations ADD COLUMN pinned_at TEXT;

-- +goose Down
-- SQLite does not support DROP COLUMN on the deployed compatibility floor.
