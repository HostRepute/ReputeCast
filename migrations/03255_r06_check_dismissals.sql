-- +goose Up
-- Warnings dismissed in the composer (review 2026-10, CO-14, ComposerWarnings):
-- who dismissed which warning of a campaign, why, and an optional note for
-- reviewers. issue_key names the warning as it was (its check, element and
-- words, and the subject for subject checks), so a warning that changes
-- (the email grew, the subject changed) comes back. Blocking issues are
-- never dismissed, whatever is stored.
CREATE TABLE campaign_check_dismissals (
    id           uuid        PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    campaign_id  uuid        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    issue_key    text        NOT NULL CHECK (length(issue_key) BETWEEN 1 AND 64),
    check_id     text        NOT NULL CHECK (length(check_id) <= 80),
    title        text        NOT NULL DEFAULT '' CHECK (length(title) <= 300),
    reason       text        NOT NULL CHECK (reason IN ('decorative', 'size', 'subject', 'wrong', 'other')),
    note         text        NOT NULL DEFAULT '' CHECK (length(note) <= 500),
    dismissed_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL,
    UNIQUE (campaign_id, issue_key)
);

-- +goose Down
DROP TABLE campaign_check_dismissals;
