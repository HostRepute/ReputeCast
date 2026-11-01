-- +goose NO TRANSACTION
-- +goose Up
-- The events waiting to be folded into a batch (03387), oldest first per
-- endpoint. A row leaves the index when its batch is made. Built
-- concurrently, and the new CHECK validated outside the swap's lock.
CREATE INDEX CONCURRENTLY IF NOT EXISTS webhook_deliveries_batching_idx ON webhook_deliveries (webhook_id, created_at, id)
    WHERE state = 'batching';
ALTER TABLE webhook_deliveries VALIDATE CONSTRAINT webhook_deliveries_state_check;

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS webhook_deliveries_batching_idx;
