-- +goose NO TRANSACTION
-- +goose Up
-- The campaigns list for many campaigns (review 2026-11 CA-2, CA-7): its
-- order, the date filter and tags, built without locking writes.
CREATE INDEX CONCURRENTLY IF NOT EXISTS campaigns_list_when_idx
    ON campaigns (workspace_id, (coalesce(started_at, scheduled_at, updated_at)) DESC, id DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS campaigns_tags_idx ON campaigns USING gin (tags) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS campaigns_deleted_idx ON campaigns (workspace_id, deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS campaigns_deleted_idx;
DROP INDEX CONCURRENTLY IF EXISTS campaigns_tags_idx;
DROP INDEX CONCURRENTLY IF EXISTS campaigns_list_when_idx;
