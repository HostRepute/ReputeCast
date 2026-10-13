-- +goose NO TRANSACTION
-- +goose Up
-- Outgoing webhook dispatch: the oldest open delivery of each endpoint (the
-- probe while it is failing) and the next ready deliveries in order. Built
-- concurrently, so an upgrade doesn't hold up deliveries on a large table.
CREATE INDEX CONCURRENTLY IF NOT EXISTS webhook_deliveries_open_idx ON webhook_deliveries (webhook_id, created_at, id) WHERE state IN ('pending','retrying','delivering');
CREATE INDEX CONCURRENTLY IF NOT EXISTS webhook_deliveries_ready_idx ON webhook_deliveries (created_at, id) WHERE state IN ('pending','retrying');

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS webhook_deliveries_ready_idx;
DROP INDEX CONCURRENTLY IF EXISTS webhook_deliveries_open_idx;
