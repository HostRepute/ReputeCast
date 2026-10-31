-- +goose Up
-- Review 2026-11 DP-3: every double opt-in confirmation email sent to a
-- contact, whatever sent it (a form, a retry or reminder, the team, an
-- import's re-permission, a partner signup), so no address gets more than
-- one every 10 minutes or five a day. Rows older than a day are purged by
-- the maintenance job.
CREATE TABLE optin_sends (
    contact_id uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    sent_at    timestamptz NOT NULL,
    PRIMARY KEY (contact_id, sent_at)
);

-- +goose Down
DROP TABLE optin_sends;
