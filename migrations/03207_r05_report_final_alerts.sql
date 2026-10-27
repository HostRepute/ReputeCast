-- +goose Up
-- "Email me when it's final" on a report whose results are still coming in
-- (ReportStates): who asked, told once when the 72 hours after the send
-- are over, then forgotten.
CREATE TABLE report_final_alerts (
    campaign_id  uuid        NOT NULL REFERENCES campaigns (id) ON DELETE CASCADE,
    user_id      uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    workspace_id uuid        NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (campaign_id, user_id)
);
CREATE INDEX report_final_alerts_created_idx ON report_final_alerts (created_at);

-- +goose Down
DROP TABLE report_final_alerts;
