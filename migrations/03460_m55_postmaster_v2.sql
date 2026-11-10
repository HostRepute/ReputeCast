-- +goose Up
-- Google Postmaster Tools v2 (M55, G-4) no longer reports domain or IP
-- reputation; compliance status lives in HostRepute instead.

ALTER TABLE postmaster_daily DROP COLUMN domain_reputation, DROP COLUMN ip_reputation;

-- +goose Down
ALTER TABLE postmaster_daily
    ADD COLUMN domain_reputation text NOT NULL DEFAULT '',
    ADD COLUMN ip_reputation text NOT NULL DEFAULT '';
