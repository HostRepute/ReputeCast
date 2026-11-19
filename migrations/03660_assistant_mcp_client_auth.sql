-- +goose Up
-- Assistant security follow-ups: how ReputeCast's OAuth client for an MCP
-- server identifies and authenticates itself.

-- client_metadata: the client is the instance's Client ID Metadata
-- Document (MCP authorization 2025-11-25): client_id is the document's
-- https URL, and the authorization server fetches it.
-- token_auth_methods: the client authentication methods the authorization
-- server's metadata lists for its token endpoint (empty when it lists
-- none). token_auth_method: the one the client uses, for the token,
-- refresh and revocation requests: none without a client secret; with one,
-- client_secret_basic when offered (or nothing is listed), else
-- client_secret_post; or what a dynamic registration returned. Empty for
-- a server set up before: basic with a secret, else none.
ALTER TABLE mcp_servers
    ADD COLUMN client_metadata boolean NOT NULL DEFAULT false,
    ADD COLUMN token_auth_methods text[] NOT NULL DEFAULT '{}',
    ADD COLUMN token_auth_method text NOT NULL DEFAULT ''
        CHECK (token_auth_method IN ('', 'none', 'client_secret_basic', 'client_secret_post'));

-- A queued revocation authenticates as its client did, after the server
-- is gone.
ALTER TABLE mcp_token_revocations
    ADD COLUMN token_auth_method text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE mcp_token_revocations DROP COLUMN token_auth_method;
ALTER TABLE mcp_servers DROP COLUMN token_auth_method, DROP COLUMN token_auth_methods, DROP COLUMN client_metadata;
