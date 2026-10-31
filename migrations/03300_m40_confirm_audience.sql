-- +goose Up
-- A scheduled send's audience as the person confirmed it (review R1.L3c):
-- editing its lists, segments or exclusions after scheduling holds the send
-- at the send-time recheck, as content, sender and route edits do. Rows
-- kept before have no digest and are compared on the recipient count only.
ALTER TABLE campaign_send_confirmations ADD COLUMN audience_digest text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE campaign_send_confirmations DROP COLUMN audience_digest;
