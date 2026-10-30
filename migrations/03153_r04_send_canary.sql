-- +goose Up
-- The canary choice a person made on Review & send (C1, GS-10): auto (the
-- sending engine decides), on or off, kept with what they confirmed for a
-- scheduled send.
ALTER TABLE campaign_send_confirmations
    ADD COLUMN canary text NOT NULL DEFAULT 'auto' CHECK (canary IN ('auto', 'on', 'off'));

-- +goose Down
ALTER TABLE campaign_send_confirmations DROP COLUMN canary;
