-- +goose Up
-- Allowed IPs for API keys and MCP agents (M55, G-5): the addresses and
-- networks a key may be called from. Empty means any address. Calls from
-- elsewhere are refused after the token is verified and counted as
-- rejections with the reason ip_not_allowed.

ALTER TABLE api_keys ADD COLUMN allowed_ips cidr[] NOT NULL DEFAULT '{}';

ALTER TABLE api_key_rejections
    DROP CONSTRAINT api_key_rejections_reason_check,
    ADD CONSTRAINT api_key_rejections_reason_check CHECK (reason IN ('expired', 'revoked', 'disabled', 'ip_not_allowed'));

-- +goose Down
DELETE FROM api_key_rejections WHERE reason = 'ip_not_allowed';

ALTER TABLE api_key_rejections
    DROP CONSTRAINT api_key_rejections_reason_check,
    ADD CONSTRAINT api_key_rejections_reason_check CHECK (reason IN ('expired', 'revoked', 'disabled'));

ALTER TABLE api_keys DROP COLUMN allowed_ips;
