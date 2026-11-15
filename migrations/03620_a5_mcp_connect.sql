-- +goose Up
CREATE TABLE mcp_connect_clients (
    id text PRIMARY KEY,
    name text NOT NULL,
    redirect_uris text[] NOT NULL,
    auth_method text NOT NULL CHECK (auth_method IN ('none', 'client_secret_basic', 'client_secret_post')),
    secret_hash bytea,
    verified boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz,
    CHECK ((auth_method = 'none' AND secret_hash IS NULL) OR (auth_method <> 'none' AND octet_length(secret_hash) = 32))
);
CREATE INDEX mcp_connect_clients_expiry ON mcp_connect_clients (expires_at) WHERE expires_at IS NOT NULL;

CREATE TABLE mcp_connect_requests (
    handle_hash bytea PRIMARY KEY CHECK (octet_length(handle_hash) = 32),
    client_id text NOT NULL REFERENCES mcp_connect_clients(id) ON DELETE CASCADE,
    redirect_uri text NOT NULL,
    resource text NOT NULL,
    state text NOT NULL,
    challenge text NOT NULL,
    scopes text[] NOT NULL,
    user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    session_id uuid REFERENCES sessions(id) ON DELETE CASCADE,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'allowed', 'denied')),
    expires_at timestamptz NOT NULL
);
CREATE INDEX mcp_connect_requests_expiry ON mcp_connect_requests (expires_at);

CREATE TABLE mcp_connect_grants (
    id uuid PRIMARY KEY,
    client_id text NOT NULL REFERENCES mcp_connect_clients(id),
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    resource text NOT NULL,
    scopes text[] NOT NULL,
    permissions text[] NOT NULL,
    scope jsonb NOT NULL,
    allowed_ips cidr[] NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    revoked_reason text NOT NULL DEFAULT ''
);
CREATE INDEX mcp_connect_grants_workspace ON mcp_connect_grants (workspace_id, id DESC);
CREATE INDEX mcp_connect_grants_client ON mcp_connect_grants (client_id, expires_at);
CREATE INDEX mcp_connect_grants_expiry ON mcp_connect_grants (expires_at);

CREATE TABLE mcp_connect_codes (
    hash bytea PRIMARY KEY CHECK (octet_length(hash) = 32),
    grant_id uuid NOT NULL REFERENCES mcp_connect_grants(id) ON DELETE CASCADE,
    redirect_uri text NOT NULL,
    challenge text NOT NULL,
    expires_at timestamptz NOT NULL,
    used_at timestamptz
);
CREATE INDEX mcp_connect_codes_expiry ON mcp_connect_codes (expires_at);

CREATE TABLE mcp_connect_tokens (
    hash bytea PRIMARY KEY CHECK (octet_length(hash) = 32),
    grant_id uuid NOT NULL REFERENCES mcp_connect_grants(id) ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind IN ('access', 'refresh')),
    expires_at timestamptz NOT NULL,
    used_at timestamptz
);
CREATE INDEX mcp_connect_tokens_grant ON mcp_connect_tokens (grant_id);
CREATE INDEX mcp_connect_tokens_expiry ON mcp_connect_tokens (expires_at);

CREATE TABLE mcp_connect_slots (
    grant_id uuid PRIMARY KEY REFERENCES mcp_connect_grants(id) ON DELETE CASCADE,
    lease_id uuid NOT NULL,
    expires_at timestamptz NOT NULL
);

CREATE TABLE mcp_connect_limits (
    key bytea NOT NULL,
    bucket timestamptz NOT NULL,
    count integer NOT NULL CHECK (count > 0),
    PRIMARY KEY (key, bucket)
);
CREATE INDEX mcp_connect_limits_expiry ON mcp_connect_limits (bucket);

-- +goose Down
DROP TABLE mcp_connect_limits;
DROP TABLE mcp_connect_slots;
DROP TABLE mcp_connect_tokens;
DROP TABLE mcp_connect_codes;
DROP TABLE mcp_connect_grants;
DROP TABLE mcp_connect_requests;
DROP TABLE mcp_connect_clients;
