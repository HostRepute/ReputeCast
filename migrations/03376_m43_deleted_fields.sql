-- +goose Up
-- A deleted custom field waits 30 days before its values go (review
-- 2026-11 AU-11): the field moves here, out of every form, segment and
-- export, and can be restored as it was; afterwards the maintenance pass
-- strips its values from contacts and the row goes. Its key stays taken
-- meanwhile, so a new field can't pick up the old values.
CREATE TABLE custom_fields_deleted (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    key          text        NOT NULL,
    label        text        NOT NULL,
    type         text        NOT NULL,
    options      jsonb       NOT NULL DEFAULT '[]',
    display      text        NOT NULL DEFAULT '',
    position     integer     NOT NULL DEFAULT 0,
    created_at   timestamptz NOT NULL,
    deleted_at   timestamptz NOT NULL,
    deleted_by   uuid        REFERENCES users(id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX custom_fields_deleted_key ON custom_fields_deleted (workspace_id, key);
CREATE INDEX custom_fields_deleted_due_idx ON custom_fields_deleted (deleted_at);

-- +goose Down
DROP TABLE custom_fields_deleted;
