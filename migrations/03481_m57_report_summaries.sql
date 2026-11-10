-- +goose Up
-- Sending summary by email (M57 G-14): a workspace member who can view
-- reports asks for a weekly or monthly email of the workspace's campaigns,
-- audience and sender health. Off is no row. last_period_start is the
-- local start date of the last period claimed for sending, so each period
-- is sent once however often the hourly tick runs.

CREATE TABLE report_summaries (
    user_id           uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id      uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    frequency         text        NOT NULL CHECK (frequency IN ('weekly', 'monthly')),
    last_period_start date,
    created_at        timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, workspace_id)
);

-- +goose Down
DROP TABLE report_summaries;
