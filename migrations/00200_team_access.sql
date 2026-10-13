-- +goose Up
-- Team, API keys, MCP agents and the audit stream (W03).

-- Invitations keep their raw token sealed with the instance key so "Copy
-- link" works after the email went out without invalidating that link.
-- Lookups still use token_hash only.
ALTER TABLE invitations ADD COLUMN token_enc bytea;
ALTER TABLE invitations ADD COLUMN expires_days integer NOT NULL DEFAULT 7 CHECK (expires_days BETWEEN 1 AND 90);

-- What the last call of a key or agent did, and who made it ("10.0.4.22 ·
-- nightly job", "09:36 · get_campaign_report").
ALTER TABLE api_keys ADD COLUMN last_used_client text NOT NULL DEFAULT '';
ALTER TABLE api_keys ADD COLUMN last_call text NOT NULL DEFAULT '';

-- Calls rejected because a key expired or was revoked, per day and client
-- address, so the UI can say "212 calls in the last 7 days from 10.0.9.14".
CREATE TABLE api_key_rejections (
    api_key_id uuid        NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    day        date        NOT NULL,
    ip         text        NOT NULL DEFAULT '',
    client     text        NOT NULL DEFAULT '',
    reason     text        NOT NULL DEFAULT 'expired' CHECK (reason IN ('expired','revoked','disabled')),
    calls      integer     NOT NULL DEFAULT 0,
    first_at   timestamptz NOT NULL DEFAULT now(),
    last_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (api_key_id, day, ip)
);

-- MCP server settings per workspace (Settings › Keys & MCP › MCP server).
ALTER TABLE workspaces ADD COLUMN mcp_settings jsonb NOT NULL
    DEFAULT '{"enabled":true,"rate_limit_per_minute":120,"approval_expiry_hours":2}';

-- Audit stream: new entries are pushed to audit_stream_url, signed with a
-- write-only secret. The cursor is the last entry delivered.
ALTER TABLE workspaces ADD COLUMN audit_stream_enabled boolean NOT NULL DEFAULT false;
ALTER TABLE workspaces ADD COLUMN audit_stream_secret_enc bytea;
ALTER TABLE workspaces ADD COLUMN audit_stream_cursor_at timestamptz;
ALTER TABLE workspaces ADD COLUMN audit_stream_cursor_id uuid;
-- {"last_ok_at":"…","last_error":"…","failing_since":"…","delivered":123}
ALTER TABLE workspaces ADD COLUMN audit_stream_status jsonb NOT NULL DEFAULT '{}';

CREATE INDEX audit_events_ws_target_idx ON audit_events (workspace_id, target_id);

-- +goose Down
DROP INDEX audit_events_ws_target_idx;
ALTER TABLE workspaces DROP COLUMN audit_stream_status;
ALTER TABLE workspaces DROP COLUMN audit_stream_cursor_id;
ALTER TABLE workspaces DROP COLUMN audit_stream_cursor_at;
ALTER TABLE workspaces DROP COLUMN audit_stream_secret_enc;
ALTER TABLE workspaces DROP COLUMN audit_stream_enabled;
ALTER TABLE workspaces DROP COLUMN mcp_settings;
DROP TABLE api_key_rejections;
ALTER TABLE api_keys DROP COLUMN last_call;
ALTER TABLE api_keys DROP COLUMN last_used_client;
ALTER TABLE invitations DROP COLUMN expires_days;
ALTER TABLE invitations DROP COLUMN token_enc;
