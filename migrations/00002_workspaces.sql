-- +goose Up
-- Workspaces, roles, members, invitations, API keys / MCP agents, audit log, notifications.

CREATE TABLE workspaces (
    id                uuid PRIMARY KEY,
    slug              text        NOT NULL CHECK (slug ~ '^[a-z0-9][a-z0-9-]{1,62}$'),
    name              text        NOT NULL,
    timezone          text        NOT NULL DEFAULT 'UTC',
    default_sender_id uuid,
    -- Send approval policy (A5.12). See internal/approvals for the shape.
    approval_policy   jsonb       NOT NULL DEFAULT '{}',
    -- Sunset policy for engagement tiers (7.9 List health).
    sunset_policy     jsonb       NOT NULL DEFAULT '{}',
    -- Publisher branding for public pages: accent, ground, logo, footer address.
    branding          jsonb       NOT NULL DEFAULT '{}',
    -- Default auto-pause thresholds for broadcasts.
    send_thresholds   jsonb       NOT NULL DEFAULT '{}',
    audit_retention_days integer  NOT NULL DEFAULT 365 CHECK (audit_retention_days BETWEEN 30 AND 3650),
    audit_stream_url  text,
    created_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    deleted_at        timestamptz
);
CREATE UNIQUE INDEX workspaces_slug_key ON workspaces (slug) WHERE deleted_at IS NULL;

-- Built-in roles (owner, admin, editor, viewer) are created per workspace with
-- builtin = true; their permissions always come from internal/authz so that new
-- permissions reach them on upgrade. Custom roles store their own permissions.
CREATE TABLE roles (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    key          text        NOT NULL CHECK (key ~ '^[a-z0-9][a-z0-9_-]{0,62}$'),
    name         text        NOT NULL,
    description  text        NOT NULL DEFAULT '',
    builtin      boolean     NOT NULL DEFAULT false,
    permissions  text[]      NOT NULL DEFAULT '{}',
    created_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX roles_workspace_key ON roles (workspace_id, key);

CREATE TABLE workspace_members (
    id             uuid PRIMARY KEY,
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id        uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id        uuid        NOT NULL REFERENCES roles(id),
    -- Optional restriction: {"lists":["uuid",...],"sender_domains":["news.acme.io"]}
    scope          jsonb       NOT NULL DEFAULT '{}',
    -- Where the membership came from: invite, sso_jit, sso_group, install, transfer.
    source         text        NOT NULL DEFAULT 'invite',
    status         text        NOT NULL DEFAULT 'active' CHECK (status IN ('active','deactivated')),
    invited_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    joined_at      timestamptz NOT NULL DEFAULT now(),
    deactivated_at timestamptz,
    last_active_at timestamptz
);
CREATE UNIQUE INDEX workspace_members_key ON workspace_members (workspace_id, user_id);
CREATE INDEX workspace_members_user_idx ON workspace_members (user_id);

CREATE TABLE invitations (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email        text        NOT NULL CHECK (email = lower(email)),
    role_id      uuid        NOT NULL REFERENCES roles(id),
    scope        jsonb       NOT NULL DEFAULT '{}',
    message      text        NOT NULL DEFAULT '',
    token_hash   bytea       NOT NULL,
    invited_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    sent_via     text,
    last_sent_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL,
    accepted_at  timestamptz,
    accepted_by  uuid        REFERENCES users(id) ON DELETE SET NULL,
    revoked_at   timestamptz
);
CREATE UNIQUE INDEX invitations_token_key ON invitations (token_hash);
CREATE INDEX invitations_workspace_idx ON invitations (workspace_id) WHERE accepted_at IS NULL AND revoked_at IS NULL;

-- API keys and MCP agents share one table: both hold scoped permission keys
-- and can never exceed their creator's permissions. Only the SHA-256 of the
-- token is stored; prefix is the visible part ("rc_live_3kF9…").
CREATE TABLE api_keys (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    kind         text        NOT NULL DEFAULT 'api_key' CHECK (kind IN ('api_key','mcp_agent')),
    name         text        NOT NULL,
    description  text        NOT NULL DEFAULT '',
    prefix       text        NOT NULL,
    token_hash   bytea       NOT NULL,
    permissions  text[]      NOT NULL,
    scope        jsonb       NOT NULL DEFAULT '{}',
    created_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz,
    last_used_at timestamptz,
    last_used_ip inet,
    revoked_at   timestamptz,
    rotated_from uuid        REFERENCES api_keys(id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX api_keys_token_key ON api_keys (token_hash);
CREATE INDEX api_keys_workspace_idx ON api_keys (workspace_id, kind);

-- Audit log. Entries are immutable: the trigger below rejects UPDATE, and
-- DELETE is only allowed for retention purges that set reputecast.audit_purge.
-- workspace_id and actor_id carry no foreign keys on purpose: entries must
-- outlive the workspace, user or key they mention.
CREATE TABLE audit_events (
    id           uuid PRIMARY KEY,
    workspace_id uuid,
    occurred_at  timestamptz NOT NULL DEFAULT now(),
    actor_type   text        NOT NULL CHECK (actor_type IN ('user','api_key','mcp_agent','system')),
    actor_id     uuid,
    actor_name   text        NOT NULL DEFAULT '',
    action       text        NOT NULL,
    target_type  text        NOT NULL DEFAULT '',
    target_id    text        NOT NULL DEFAULT '',
    target_name  text        NOT NULL DEFAULT '',
    result       text        NOT NULL DEFAULT 'allowed' CHECK (result IN ('allowed','denied','error')),
    reason       text        NOT NULL DEFAULT '',
    ip           inet,
    user_agent   text        NOT NULL DEFAULT '',
    request_id   text        NOT NULL DEFAULT '',
    before       jsonb,
    after        jsonb,
    metadata     jsonb       NOT NULL DEFAULT '{}'
);
CREATE INDEX audit_events_workspace_idx ON audit_events (workspace_id, occurred_at DESC);
CREATE INDEX audit_events_occurred_idx ON audit_events (occurred_at DESC);
CREATE INDEX audit_events_actor_idx ON audit_events (actor_id, occurred_at DESC);
CREATE INDEX audit_events_action_idx ON audit_events (action);

-- +goose StatementBegin
CREATE FUNCTION audit_events_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' AND current_setting('reputecast.audit_purge', true) = 'on' THEN
        RETURN OLD;
    END IF;
    RAISE EXCEPTION 'audit events are immutable';
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER audit_events_no_update BEFORE UPDATE OR DELETE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION audit_events_immutable();

CREATE TABLE notifications (
    id           uuid PRIMARY KEY,
    user_id      uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id uuid        REFERENCES workspaces(id) ON DELETE CASCADE,
    kind         text        NOT NULL,
    severity     text        NOT NULL DEFAULT 'info' CHECK (severity IN ('info','ok','warn','bad')),
    title        text        NOT NULL,
    body         text        NOT NULL DEFAULT '',
    link         text        NOT NULL DEFAULT '',
    data         jsonb       NOT NULL DEFAULT '{}',
    created_at   timestamptz NOT NULL DEFAULT now(),
    read_at      timestamptz
);
CREATE INDEX notifications_user_idx ON notifications (user_id, created_at DESC);
CREATE INDEX notifications_unread_idx ON notifications (user_id) WHERE read_at IS NULL;

-- +goose Down
DROP TABLE notifications;
DROP TRIGGER audit_events_no_update ON audit_events;
DROP FUNCTION audit_events_immutable();
DROP TABLE audit_events;
DROP TABLE api_keys;
DROP TABLE invitations;
DROP TABLE workspace_members;
DROP TABLE roles;
DROP TABLE workspaces;
