-- +goose Up
-- Mailgun signs a timestamp and a one-time token, not the event, so each
-- token is honored with the one event it first came with. The tokens are kept here, where every process and every
-- restart sees them, for as long as their signature is accepted; the
-- ingest sweep drops them after. Hashes only: neither the token nor the
-- event is kept.
CREATE TABLE webhook_tokens (
    -- Whose tokens these are: the adapter ("mailgun").
    scope        text        NOT NULL,
    token_hash   bytea       NOT NULL,
    payload_hash bytea       NOT NULL,
    expires_at   timestamptz NOT NULL,
    PRIMARY KEY (scope, token_hash)
);
CREATE INDEX webhook_tokens_expires_idx ON webhook_tokens (expires_at);

-- +goose Down
DROP TABLE webhook_tokens;
