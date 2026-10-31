-- +goose Up
-- Review 2026-11 EN-5: an automation alert (a step failing, sending held)
-- tells the people who edit automations once, when it opens.
ALTER TABLE automation_alerts ADD COLUMN notified_at timestamptz;
-- Alerts open before this release were seen on the Automations list.
UPDATE automation_alerts SET notified_at = first_at WHERE resolved_at IS NULL;

-- +goose Down
ALTER TABLE automation_alerts DROP COLUMN notified_at;
