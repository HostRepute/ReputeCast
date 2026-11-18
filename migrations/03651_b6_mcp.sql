-- +goose Up
-- B6: what the assistant's MCP client keeps for OAuth discovery and
-- revocation, sign-ins waiting to be revoked, and confirmations that expire.

-- The protected resource's identifier when its metadata names the server's
-- origin rather than its endpoint (found at the root well-known address),
-- and the authorization server's revocation endpoint (RFC 7009).
ALTER TABLE mcp_servers
    ADD COLUMN resource text NOT NULL DEFAULT '',
    ADD COLUMN revocation_endpoint text NOT NULL DEFAULT '';

-- A member's refresh token removed from mcp_tokens (they disconnected,
-- left or were removed, or the server was deleted): revoked at the
-- authorization server once, then deleted, whatever its answer. Each row
-- carries what revoking needs, so it outlives the server: the endpoint,
-- the client and its secret. The token and the secret are the sealed bytes
-- of mcp_tokens and mcp_servers, whose associated data names the server's
-- and the person's ids, kept here as plain values (no foreign key).
-- Rows are queued by the statements that delete sign-ins (several at once),
-- so the key is an identity, not a UUID from Go.
CREATE TABLE mcp_token_revocations (
    id                   bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    server_id            uuid NOT NULL,
    user_id              uuid NOT NULL,
    revocation_endpoint  text NOT NULL CHECK (revocation_endpoint <> ''),
    client_id            text NOT NULL,
    client_secret_sealed bytea,
    refresh_sealed       bytea NOT NULL,
    created_at           timestamptz NOT NULL
);
CREATE INDEX mcp_token_revocations_created_idx ON mcp_token_revocations (created_at);

-- A call waiting for its owner for a day expires: the turn ends, nothing
-- is sent.
ALTER TABLE ai_assistant_confirmations DROP CONSTRAINT ai_assistant_confirmations_state_check;
ALTER TABLE ai_assistant_confirmations ADD CONSTRAINT ai_assistant_confirmations_state_check
    CHECK (state IN ('pending', 'allowed', 'denied', 'calling', 'done', 'expired'));
CREATE INDEX ai_assistant_confirmations_pending_idx ON ai_assistant_confirmations (created_at) WHERE state = 'pending';

-- +goose Down
DROP INDEX ai_assistant_confirmations_pending_idx;
UPDATE ai_assistant_confirmations SET state = 'done' WHERE state = 'expired';
ALTER TABLE ai_assistant_confirmations DROP CONSTRAINT ai_assistant_confirmations_state_check;
ALTER TABLE ai_assistant_confirmations ADD CONSTRAINT ai_assistant_confirmations_state_check
    CHECK (state IN ('pending', 'allowed', 'denied', 'calling', 'done'));
DROP TABLE mcp_token_revocations;
ALTER TABLE mcp_servers DROP COLUMN revocation_endpoint, DROP COLUMN resource;
