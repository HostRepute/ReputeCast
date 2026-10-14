-- +goose Up
-- Overview and campaign reports (spec 7.1, 7.7, 5.5): what the report
-- needs beyond the base schema of 00005.

-- Read-only report links. Only the SHA-256 of the token is used to look a
-- link up; the token itself is kept sealed (AES-GCM, associated data
-- "report_share:<id>:token") so members who may share reports can copy the
-- link again from the report, as the design shows. Revoking keeps the row
-- for the audit trail; views count successful opens of the public page.
ALTER TABLE report_shares
    ADD COLUMN token_sealed   bytea,
    ADD COLUMN revoked_by     uuid REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN views          bigint NOT NULL DEFAULT 0,
    ADD COLUMN last_viewed_at timestamptz;
CREATE INDEX report_shares_campaign_idx ON report_shares (campaign_id, created_at DESC);

-- Median time from handing a message to the provider to its delivery event,
-- per route, written when a send completes ("Median time to delivery" in
-- the per-provider comparison). While it is empty the report computes it
-- from the messages of the route.
ALTER TABLE campaign_routes ADD COLUMN delivery_p50_ms integer;

-- Unsubscribe feedback is read per campaign and reason.
CREATE INDEX unsubscribe_feedback_reason_idx ON unsubscribe_feedback (campaign_id, reason);

-- Bounces of a campaign by class (the report's taxonomy table).
CREATE INDEX delivery_events_bounces_idx ON delivery_events (campaign_id, bounce_class)
    WHERE type IN ('hard_bounce', 'soft_bounce');

-- +goose Down
DROP INDEX delivery_events_bounces_idx;
DROP INDEX unsubscribe_feedback_reason_idx;
ALTER TABLE campaign_routes DROP COLUMN delivery_p50_ms;
DROP INDEX report_shares_campaign_idx;
ALTER TABLE report_shares
    DROP COLUMN last_viewed_at,
    DROP COLUMN views,
    DROP COLUMN revoked_by,
    DROP COLUMN token_sealed;
