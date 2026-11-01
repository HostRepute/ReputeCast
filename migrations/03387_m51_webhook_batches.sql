-- +goose Up
-- Review 2026-11 EN-8: an endpoint can take its email.* events in batches.
-- Such an event waits as a 'batching' row until the dispatcher folds up to
-- 100 of them into one delivery. The CHECK is replaced NOT VALID so the
-- swap doesn't scan the table under its lock; 03388 validates it.
ALTER TABLE webhooks ADD COLUMN batch_email boolean NOT NULL DEFAULT false;
ALTER TABLE webhook_deliveries DROP CONSTRAINT webhook_deliveries_state_check;
ALTER TABLE webhook_deliveries ADD CONSTRAINT webhook_deliveries_state_check
    CHECK (state IN ('pending','delivering','delivered','retrying','dead','held','batching')) NOT VALID;

-- +goose Down
DELETE FROM webhook_deliveries WHERE state = 'batching';
ALTER TABLE webhook_deliveries DROP CONSTRAINT webhook_deliveries_state_check;
ALTER TABLE webhook_deliveries ADD CONSTRAINT webhook_deliveries_state_check
    CHECK (state IN ('pending','delivering','delivered','retrying','dead','held'));
ALTER TABLE webhooks DROP COLUMN batch_email;
