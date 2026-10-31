-- +goose Up
-- Long audience actions run in the background (review 2026-11 AU-4): an
-- export of every contact a filter matches, and a bulk "Resend opt-in".
-- selection is the filter or ids as they were asked for; actor is who
-- asked, with their list scope, so the job sees what they saw. after_id is
-- the keyset the job has reached, so a restart goes on from there. An
-- export's file lives under the files directory until expires_at; the
-- maintenance job removes it.
CREATE TABLE audience_jobs (
    id          uuid PRIMARY KEY,
    workspace_id uuid       NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    kind        text        NOT NULL CHECK (kind IN ('export', 'resend_optin')),
    status      text        NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'done', 'failed')),
    selection   jsonb       NOT NULL,
    actor       jsonb       NOT NULL,
    created_by  uuid        REFERENCES users(id) ON DELETE SET NULL,
    total       bigint      NOT NULL DEFAULT 0,
    done        bigint      NOT NULL DEFAULT 0,
    after_id    uuid,
    result      jsonb       NOT NULL DEFAULT '{}',
    file_key    text        NOT NULL DEFAULT '',
    error       text        NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    started_at  timestamptz,
    finished_at timestamptz,
    expires_at  timestamptz
);
CREATE INDEX audience_jobs_workspace_idx ON audience_jobs (workspace_id, created_at DESC);
CREATE INDEX audience_jobs_expires_idx ON audience_jobs (expires_at) WHERE file_key <> '';

-- +goose Down
DROP TABLE audience_jobs;
