-- +goose Up
-- Deliveries at mailbox providers that report complaints per message
-- (review F-2): a provider's complaint rate divides by these, so Gmail's
-- and iCloud's mail, whose complaints never come back one by one, doesn't
-- dilute it. Days counted before the column are unknown (NULL), not 0;
-- days that start after it count from 0.
ALTER TABLE provider_daily_stats ADD COLUMN delivered_reporting bigint;
ALTER TABLE provider_daily_stats ALTER COLUMN delivered_reporting SET DEFAULT 0;

-- +goose Down
ALTER TABLE provider_daily_stats DROP COLUMN delivered_reporting;
