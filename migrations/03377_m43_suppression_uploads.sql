-- +goose Up
-- A CSV upload of suppressions is one group (review 2026-11 AU-14): the
-- list shows it as a row, and a wrong upload can be taken off whole, with
-- a reason for the audit log.
CREATE TABLE suppression_uploads (
    id             uuid PRIMARY KEY,
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    file_name      text        NOT NULL DEFAULT '',
    reason         text        NOT NULL DEFAULT '',
    note           text        NOT NULL DEFAULT '',
    created_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    removed_at     timestamptz,
    removed_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    removed_reason text        NOT NULL DEFAULT ''
);
CREATE INDEX suppression_uploads_workspace_idx ON suppression_uploads (workspace_id, created_at DESC);
ALTER TABLE suppressions ADD COLUMN upload_id uuid REFERENCES suppression_uploads(id) ON DELETE SET NULL;
CREATE INDEX suppressions_upload_idx ON suppressions (upload_id) WHERE upload_id IS NOT NULL;

-- +goose Down
DROP INDEX suppressions_upload_idx;
ALTER TABLE suppressions DROP COLUMN upload_id;
DROP TABLE suppression_uploads;
