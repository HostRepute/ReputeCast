-- +goose Up
-- Outgoing webhooks: one delivery row per event and endpoint, updated
-- on every attempt. Failed deliveries retry with backoff up to max_attempts,
-- then wait in the dead-letter queue ("dead") for 7 days.

ALTER TABLE webhooks ADD COLUMN description text NOT NULL DEFAULT '';
ALTER TABLE webhooks ADD COLUMN secret_rotated_at timestamptz;
ALTER TABLE webhooks ADD COLUMN paused_at timestamptz;
ALTER TABLE webhooks ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();

ALTER TABLE webhook_deliveries ADD COLUMN event_id uuid;
ALTER TABLE webhook_deliveries ADD COLUMN state text NOT NULL DEFAULT 'pending'
    CHECK (state IN ('pending','delivering','delivered','retrying','dead','held'));
ALTER TABLE webhook_deliveries ADD COLUMN max_attempts integer NOT NULL DEFAULT 8;
ALTER TABLE webhook_deliveries ADD COLUMN next_attempt_at timestamptz;
ALTER TABLE webhook_deliveries ADD COLUMN enqueued_at timestamptz;
ALTER TABLE webhook_deliveries ADD COLUMN last_attempt_at timestamptz;
ALTER TABLE webhook_deliveries ADD COLUMN delivered_at timestamptz;
-- attempt now counts attempts made so far (0 before the first one).
ALTER TABLE webhook_deliveries ALTER COLUMN attempt SET DEFAULT 0;

CREATE INDEX webhook_deliveries_due_idx ON webhook_deliveries (next_attempt_at) WHERE state IN ('pending','retrying');
CREATE INDEX webhook_deliveries_state_idx ON webhook_deliveries (webhook_id, state);

-- +goose Down
DROP INDEX webhook_deliveries_state_idx;
DROP INDEX webhook_deliveries_due_idx;
ALTER TABLE webhook_deliveries ALTER COLUMN attempt SET DEFAULT 1;
ALTER TABLE webhook_deliveries DROP COLUMN delivered_at;
ALTER TABLE webhook_deliveries DROP COLUMN last_attempt_at;
ALTER TABLE webhook_deliveries DROP COLUMN enqueued_at;
ALTER TABLE webhook_deliveries DROP COLUMN next_attempt_at;
ALTER TABLE webhook_deliveries DROP COLUMN max_attempts;
ALTER TABLE webhook_deliveries DROP COLUMN state;
ALTER TABLE webhook_deliveries DROP COLUMN event_id;
ALTER TABLE webhooks DROP COLUMN updated_at;
ALTER TABLE webhooks DROP COLUMN paused_at;
ALTER TABLE webhooks DROP COLUMN secret_rotated_at;
ALTER TABLE webhooks DROP COLUMN description;
