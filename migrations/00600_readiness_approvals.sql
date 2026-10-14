-- +goose Up
-- Review & send (spec 5.1, 7.5) and send approvals (add-on 3.8, 5.12).

-- A completed Send Readiness run remembers what it checked (the campaign
-- version and a fingerprint of content, audience, route and schedule), so
-- the review screen knows when the campaign changed since and re-runs, and
-- who ran it. "Continue with provider data only" accepts checks that could
-- not run (HostRepute down); the acceptance is stored on the run.
ALTER TABLE readiness_runs
    ADD COLUMN ran_by              uuid        REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN campaign_version    integer     NOT NULL DEFAULT 0,
    ADD COLUMN fingerprint         text        NOT NULL DEFAULT '',
    ADD COLUMN unknown_accepted_at timestamptz,
    ADD COLUMN unknown_accepted_by uuid        REFERENCES users(id) ON DELETE SET NULL;

-- The run in progress for a campaign (at most one). The review screen polls
-- it to tick the checks in group by group; the row goes when the run ends.
CREATE TABLE readiness_progress (
    campaign_id uuid        PRIMARY KEY REFERENCES campaigns(id) ON DELETE CASCADE,
    run_id      uuid        NOT NULL,
    trigger     text        NOT NULL,
    started_by  uuid        REFERENCES users(id) ON DELETE SET NULL,
    started_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    -- [{"key":"reputation","status":"running","label":"…","done":52,"total":84}]
    groups      jsonb       NOT NULL DEFAULT '[]',
    done        integer     NOT NULL DEFAULT 0,
    total       integer     NOT NULL DEFAULT 0
);

-- Approval requests: authors can cancel them; a request remembers the
-- campaign version and readiness run it covers, who decided, whose edit
-- invalidated it, when approvers were last reminded, and the request it
-- follows (re-requests after changes or invalidation show the diff).
ALTER TABLE approval_requests DROP CONSTRAINT approval_requests_status_check;
ALTER TABLE approval_requests ADD CONSTRAINT approval_requests_status_check
    CHECK (status IN ('pending','approved','changes_requested','rejected','invalidated','bypassed','sent','cancelled'));
ALTER TABLE approval_requests
    ADD COLUMN campaign_version integer,
    ADD COLUMN readiness_run_id uuid REFERENCES readiness_runs(id) ON DELETE SET NULL,
    ADD COLUMN decided_by       uuid REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN invalidated_by   uuid REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN reminded_at      timestamptz,
    ADD COLUMN previous_id      uuid REFERENCES approval_requests(id) ON DELETE SET NULL,
    -- The status before an invalidation (pending or approved), restored when
    -- the campaign is put back as approved ("keeps approval").
    ADD COLUMN invalidated_from text NOT NULL DEFAULT '';
-- One open (pending or approved) request per campaign and per automation.
CREATE UNIQUE INDEX approval_requests_open_campaign ON approval_requests (campaign_id)
    WHERE status IN ('pending','approved') AND campaign_id IS NOT NULL;
CREATE UNIQUE INDEX approval_requests_open_automation ON approval_requests (automation_id)
    WHERE status = 'pending' AND automation_id IS NOT NULL;
CREATE INDEX approval_requests_open_idx ON approval_requests (workspace_id)
    WHERE status IN ('pending','approved');

-- +goose Down
DROP INDEX approval_requests_open_idx;
DROP INDEX approval_requests_open_automation;
DROP INDEX approval_requests_open_campaign;
ALTER TABLE approval_requests
    DROP COLUMN invalidated_from,
    DROP COLUMN previous_id,
    DROP COLUMN reminded_at,
    DROP COLUMN invalidated_by,
    DROP COLUMN decided_by,
    DROP COLUMN readiness_run_id,
    DROP COLUMN campaign_version;
UPDATE approval_requests SET status = 'rejected' WHERE status = 'cancelled';
ALTER TABLE approval_requests DROP CONSTRAINT approval_requests_status_check;
ALTER TABLE approval_requests ADD CONSTRAINT approval_requests_status_check
    CHECK (status IN ('pending','approved','changes_requested','rejected','invalidated','bypassed','sent'));
DROP TABLE readiness_progress;
ALTER TABLE readiness_runs
    DROP COLUMN unknown_accepted_by,
    DROP COLUMN unknown_accepted_at,
    DROP COLUMN fingerprint,
    DROP COLUMN campaign_version,
    DROP COLUMN ran_by;
