-- +goose Up
-- Review 2026-11 EN-1: an automation event or run that fails no longer
-- holds up the rest. Each failure is counted with its error and retried
-- later (1, 2, 4, 8 minutes); after five, the event is set aside and the
-- run fails with a "step failing" alert on its flow.
ALTER TABLE automation_events
    ADD COLUMN attempts   smallint    NOT NULL DEFAULT 0,
    ADD COLUMN last_error text        NOT NULL DEFAULT '',
    ADD COLUMN retry_at   timestamptz;
ALTER TABLE automation_runs
    ADD COLUMN attempts   smallint    NOT NULL DEFAULT 0,
    ADD COLUMN last_error text        NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE automation_runs DROP COLUMN attempts, DROP COLUMN last_error;
ALTER TABLE automation_events DROP COLUMN attempts, DROP COLUMN last_error, DROP COLUMN retry_at;
