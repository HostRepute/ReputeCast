-- +goose NO TRANSACTION
-- +goose Up
-- The retention jobs' reads (review 2026-11 DP-4, DP-6): opens and clicks
-- that still keep an IP address or user agent, by age; contacts still
-- waiting for their double opt-in, by age. A row leaves each index once the
-- job has dealt with it. Built concurrently: delivery_events is the largest
-- table.
CREATE INDEX CONCURRENTLY IF NOT EXISTS delivery_events_detail_idx ON delivery_events (workspace_id, occurred_at)
    WHERE type IN ('open', 'click') AND (ip IS NOT NULL OR user_agent <> '');
CREATE INDEX CONCURRENTLY IF NOT EXISTS contacts_pending_idx ON contacts (workspace_id, created_at)
    WHERE consent = 'pending' AND erased_at IS NULL;

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS contacts_pending_idx;
DROP INDEX CONCURRENTLY IF EXISTS delivery_events_detail_idx;
