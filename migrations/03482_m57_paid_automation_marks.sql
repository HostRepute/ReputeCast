-- +goose Up
-- Automation triggers for paid members (M57, G-11): when a Stripe
-- subscription's "Becomes a paid member" and "Stops paying" events were
-- queued. Each is set once, in the transaction that queues the event, so a
-- redelivered or replayed Stripe event never starts a flow twice.

ALTER TABLE paid_subscriptions
    ADD COLUMN automation_started_at timestamptz,
    ADD COLUMN automation_ended_at timestamptz;

-- History never starts a flow: people who started paying, or whose access
-- ended, before this release are marked as already told.
UPDATE paid_subscriptions SET automation_started_at = started_at
WHERE kind = 'stripe' AND status IN ('active', 'past_due', 'canceled');
UPDATE paid_subscriptions SET automation_ended_at = ended_at
WHERE ended_at IS NOT NULL;

-- +goose Down
ALTER TABLE paid_subscriptions
    DROP COLUMN automation_started_at,
    DROP COLUMN automation_ended_at;
