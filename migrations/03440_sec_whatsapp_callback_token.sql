-- +goose Up
-- Meta's callback URL for a WhatsApp account ends with a random token,
-- /hooks/whatsapp/{workspace}/{token}, as every other inbound hook does, so
-- the workspace's slug alone no longer reaches it. The admin copies the URL
-- into Meta's app settings, so the token is kept sealed (associated data
-- 'whatsapp_account:<id>:callback_token') for the settings screen, and a
-- request is checked against its SHA-256. Accounts connected before get
-- their token the first time the account is read: only the app can seal.
--
-- callback_moved_at is when Meta first called the token's address (its
-- verification request, or a delivery with a valid signature). Until then
-- the old address, /hooks/whatsapp/{workspace}, keeps taking the account's
-- deliveries; from then on it refuses them. An account connected from now
-- on starts moved: it never had the old address.
--
-- Deliveries refused for their address (the old one once moved, or a
-- wrong token) are counted, with the last one's time, like refused
-- signatures, and never stored.
ALTER TABLE whatsapp_accounts
    ADD COLUMN callback_token_hash      bytea,
    ADD COLUMN callback_token_enc       bytea,
    ADD COLUMN callback_moved_at        timestamptz,
    ADD COLUMN wrong_address_deliveries bigint NOT NULL DEFAULT 0,
    ADD COLUMN wrong_address_at         timestamptz;

-- +goose Down
ALTER TABLE whatsapp_accounts
    DROP COLUMN callback_token_hash,
    DROP COLUMN callback_token_enc,
    DROP COLUMN callback_moved_at,
    DROP COLUMN wrong_address_deliveries,
    DROP COLUMN wrong_address_at;
