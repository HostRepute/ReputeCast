-- +goose Up
-- Instance admin › Audit log (W14): instance-level entries (sign-ins, SSO,
-- policies, users) can be streamed to their own webhook, like a
-- workspace's log; the retention is instance_settings.audit_retention_days.
ALTER TABLE instance_settings
    ADD COLUMN audit_stream_enabled    boolean     NOT NULL DEFAULT false,
    ADD COLUMN audit_stream_url        text,
    ADD COLUMN audit_stream_secret_enc bytea,
    ADD COLUMN audit_stream_cursor_at  timestamptz,
    ADD COLUMN audit_stream_cursor_id  uuid,
    ADD COLUMN audit_stream_status     jsonb       NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE instance_settings
    DROP COLUMN audit_stream_enabled,
    DROP COLUMN audit_stream_url,
    DROP COLUMN audit_stream_secret_enc,
    DROP COLUMN audit_stream_cursor_at,
    DROP COLUMN audit_stream_cursor_id,
    DROP COLUMN audit_stream_status;
