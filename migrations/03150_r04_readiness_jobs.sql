-- +goose Up
-- Send Readiness runs in the background (SR-8). Asking for a run claims the
-- campaign's progress row at once (one run at a time per campaign) and a
-- worker runs the checks; the review screen polls the row. A run that
-- stopped on an error keeps the row with why, until the next run.
ALTER TABLE readiness_progress
    ADD COLUMN failed_at timestamptz,
    ADD COLUMN error     text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE readiness_progress
    DROP COLUMN error,
    DROP COLUMN failed_at;
