-- +goose Up
-- Outgoing webhook dispatch: the oldest open delivery of each endpoint (the
-- probe while it is failing) and the next ready deliveries in order.
CREATE INDEX webhook_deliveries_open_idx ON webhook_deliveries (webhook_id, created_at, id) WHERE state IN ('pending','retrying','delivering');
CREATE INDEX webhook_deliveries_ready_idx ON webhook_deliveries (created_at, id) WHERE state IN ('pending','retrying');

-- +goose Down
DROP INDEX webhook_deliveries_ready_idx;
DROP INDEX webhook_deliveries_open_idx;
