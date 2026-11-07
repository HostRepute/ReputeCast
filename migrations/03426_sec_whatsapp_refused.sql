-- +goose Up
-- The WhatsApp webhook's address is public, so a delivery whose signature
-- doesn't match the app secret is no longer stored: the account counts
-- them, with the last one's time, for the Events tab. Events for another WhatsApp Business Account or number than the
-- connected one (workspaces sharing one Meta app) are dropped and counted
-- the same way.
ALTER TABLE whatsapp_accounts
    ADD COLUMN rejected_deliveries  bigint NOT NULL DEFAULT 0,
    ADD COLUMN rejected_at          timestamptz,
    ADD COLUMN other_account_events bigint NOT NULL DEFAULT 0,
    ADD COLUMN other_account_at     timestamptz;

-- +goose Down
ALTER TABLE whatsapp_accounts
    DROP COLUMN rejected_deliveries,
    DROP COLUMN rejected_at,
    DROP COLUMN other_account_events,
    DROP COLUMN other_account_at;
