-- +goose NO TRANSACTION
-- +goose Up
-- The routing impact preview (review R3.U4a) reads the latest recipients of
-- a workspace's Automations, System or Transactional mail, newest first, on
-- every change to the unsaved routing. Without this index that read scans
-- messages back from the newest row of every workspace and stream. Only
-- mail outside campaigns is indexed (broadcasts are the bulk of the table),
-- built concurrently so an upgrade doesn't hold up sending.
CREATE INDEX CONCURRENTLY IF NOT EXISTS messages_stream_recent_idx ON messages (workspace_id, stream, queued_at DESC) WHERE campaign_id IS NULL;

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS messages_stream_recent_idx;
