-- +goose Up
-- Deleting a list pauses what used it: live signup forms, live automations
-- it triggers and scheduled campaigns sending to it. The list keeps a note
-- of what it paused so restoring it (for 30 days) can turn those back on.
ALTER TABLE lists ADD COLUMN delete_paused jsonb NOT NULL DEFAULT '{}';
CREATE INDEX lists_deleted_idx ON lists (deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP INDEX lists_deleted_idx;
ALTER TABLE lists DROP COLUMN delete_paused;
