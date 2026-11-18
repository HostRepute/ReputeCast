-- +goose Up
-- B5: remote MCP servers for the assistant (assistant brief D2, §3.6).
-- Managers connect servers; each tool waits for a manager's decision;
-- members connect their own OAuth accounts. Secrets and tokens are sealed.

CREATE TABLE mcp_servers (
    id                 uuid PRIMARY KEY,
    workspace_id       uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    -- key prefixes the server's tools for the model: mcp__<key>__<tool>.
    key                text NOT NULL CHECK (key ~ '^[a-z0-9]{1,16}$'),
    name               text NOT NULL CHECK (length(name) BETWEEN 1 AND 80),
    url                text NOT NULL,
    transport          text NOT NULL DEFAULT 'streamable' CHECK (transport IN ('streamable', 'sse')),
    auth               text NOT NULL CHECK (auth IN ('none', 'header', 'oauth')),
    header_name        text NOT NULL DEFAULT '',
    header_sealed      bytea,
    -- OAuth: the client (registered dynamically or entered by a manager)
    -- and the authorization server found by discovery.
    client_id          text NOT NULL DEFAULT '',
    client_secret_sealed bytea,
    client_registered  boolean NOT NULL DEFAULT false,
    issuer             text NOT NULL DEFAULT '',
    authorization_endpoint text NOT NULL DEFAULT '',
    token_endpoint     text NOT NULL DEFAULT '',
    scopes             text NOT NULL DEFAULT '',
    -- Who may use it: everyone with ai:use, or the listed roles.
    access_roles       text[],
    enabled            boolean NOT NULL DEFAULT true,
    status             text NOT NULL DEFAULT 'new' CHECK (status IN ('new', 'ok', 'unreachable', 'sign_in')),
    status_detail      text NOT NULL DEFAULT '',
    synced_at          timestamptz,
    revision           bigint NOT NULL DEFAULT 1,
    created_by         uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at         timestamptz NOT NULL,
    updated_at         timestamptz NOT NULL,
    UNIQUE (workspace_id, key),
    UNIQUE (workspace_id, id),
    CHECK ((auth = 'header') = (header_sealed IS NOT NULL))
);

-- A server's tools as its last sync listed them. hash covers the
-- description, input schema and annotations; approved_hash is what a
-- manager decided on, and the approved_ columns keep that version's text so
-- a tool that changed afterwards is reviewed as a diff. A tool whose hash
-- moved is off until decided again.
CREATE TABLE mcp_tools (
    id            uuid PRIMARY KEY,
    server_id     uuid NOT NULL REFERENCES mcp_servers(id) ON DELETE CASCADE,
    name          text NOT NULL CHECK (length(name) BETWEEN 1 AND 128),
    model_name    text NOT NULL CHECK (model_name ~ '^mcp__[a-z0-9]{1,16}__[A-Za-z0-9_-]{1,40}$'),
    description   text NOT NULL DEFAULT '',
    input_schema  jsonb NOT NULL DEFAULT '{}',
    annotations   jsonb NOT NULL DEFAULT '{}',
    hash          bytea NOT NULL CHECK (octet_length(hash) = 32),
    decision      text NOT NULL DEFAULT 'pending' CHECK (decision IN ('pending', 'read_only', 'writes', 'blocked')),
    approved_hash bytea CHECK (octet_length(approved_hash) = 32),
    decided_by    uuid REFERENCES users(id) ON DELETE SET NULL,
    decided_at    timestamptz,
    first_seen_at timestamptz NOT NULL,
    removed_at    timestamptz,
    approved_description  text,
    approved_input_schema jsonb,
    approved_annotations  jsonb,
    UNIQUE (server_id, name),
    UNIQUE (server_id, model_name)
);

-- A member's OAuth sign-in in progress: single use, bound to their
-- session, ten minutes.
CREATE TABLE mcp_oauth_states (
    id              uuid PRIMARY KEY,
    state_hash      bytea NOT NULL UNIQUE CHECK (octet_length(state_hash) = 32),
    server_id       uuid NOT NULL REFERENCES mcp_servers(id) ON DELETE CASCADE,
    user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id      uuid NOT NULL,
    verifier_sealed bytea NOT NULL,
    created_at      timestamptz NOT NULL,
    expires_at      timestamptz NOT NULL,
    used_at         timestamptz
);
CREATE INDEX mcp_oauth_states_expiry_idx ON mcp_oauth_states (expires_at);

-- A member's tokens for an OAuth server, sealed for (server, person).
CREATE TABLE mcp_tokens (
    server_id      uuid NOT NULL REFERENCES mcp_servers(id) ON DELETE CASCADE,
    user_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id   uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    access_sealed  bytea NOT NULL,
    refresh_sealed bytea,
    expires_at     timestamptz,
    created_at     timestamptz NOT NULL,
    updated_at     timestamptz NOT NULL,
    PRIMARY KEY (server_id, user_id)
);
CREATE INDEX mcp_tokens_member_idx ON mcp_tokens (workspace_id, user_id);

-- The servers a thread may use, as its owner chose.
CREATE TABLE ai_assistant_thread_servers (
    thread_id uuid NOT NULL REFERENCES ai_assistant_threads(id) ON DELETE CASCADE,
    server_id uuid NOT NULL REFERENCES mcp_servers(id) ON DELETE CASCADE,
    PRIMARY KEY (thread_id, server_id)
);

-- A call that writes outside ReputeCast, waiting for its owner: the turn
-- is "waiting". Its arguments are sealed; done holds the results of the
-- response's earlier calls (sealed), to go on from there.
CREATE TABLE ai_assistant_confirmations (
    id              uuid PRIMARY KEY,
    thread_id       uuid NOT NULL REFERENCES ai_assistant_threads(id) ON DELETE CASCADE,
    turn_id         uuid NOT NULL REFERENCES ai_assistant_turns(id) ON DELETE CASCADE,
    step_id         uuid NOT NULL,
    call_index      integer NOT NULL CHECK (call_index >= 0),
    call_id         text NOT NULL,
    -- The tool as it was offered: its row may change or go meanwhile.
    server_id       uuid REFERENCES mcp_servers(id) ON DELETE SET NULL,
    tool_id         uuid REFERENCES mcp_tools(id) ON DELETE SET NULL,
    server_name     text NOT NULL,
    tool_name       text NOT NULL,
    model_name      text NOT NULL,
    tool_hash       bytea NOT NULL CHECK (octet_length(tool_hash) = 32),
    arguments_sealed bytea NOT NULL,
    arguments_digest bytea NOT NULL CHECK (octet_length(arguments_digest) = 32),
    done_sealed     bytea NOT NULL,
    -- pending → allowed or denied (the owner) → calling → done; Stop
    -- closes it as done without a decision.
    state           text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'allowed', 'denied', 'calling', 'done')),
    decision        text CHECK (decision IN ('allow', 'deny')),
    decided_by      uuid REFERENCES users(id) ON DELETE SET NULL,
    decided_at      timestamptz,
    created_at      timestamptz NOT NULL,
    CHECK ((decision IS NULL) = (decided_at IS NULL)),
    CHECK (state NOT IN ('allowed', 'denied', 'calling') OR decision IS NOT NULL)
);
CREATE UNIQUE INDEX ai_assistant_confirmations_open_idx ON ai_assistant_confirmations (turn_id) WHERE state IN ('pending', 'allowed', 'denied', 'calling');
CREATE INDEX ai_assistant_confirmations_thread_idx ON ai_assistant_confirmations (thread_id);

-- A turn queued again after its owner decided on a call: the sweep's
-- half hour counts from then, not from the person's message.
ALTER TABLE ai_assistant_turns ADD COLUMN queued_at timestamptz;

-- +goose Down
ALTER TABLE ai_assistant_turns DROP COLUMN queued_at;
DROP TABLE ai_assistant_confirmations;
DROP TABLE ai_assistant_thread_servers;
DROP TABLE mcp_tokens;
DROP TABLE mcp_oauth_states;
DROP TABLE mcp_tools;
DROP TABLE mcp_servers;
