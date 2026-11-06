-- +goose Up
-- Since when a link tracking domain has been verified, as the pages' and
-- the short links' domains record it, so the first workspace to verify a
-- host keeps it. Nobody recorded it for domains verified before: a host
-- two workspaces verified before the upgrade then needs the TXT record of
-- whichever workspace keeps it, as the pages' domain already did.
ALTER TABLE sending_domains ADD COLUMN tracking_verified_at timestamptz;

-- 'refused': the CNAME points here, but nothing shows the domain is this
-- workspace's: another workspace verified it first, or it's under the
-- instance's own address, and its TXT record is missing; or it's one of
-- the instance's own hosts.
ALTER TABLE sending_domains DROP CONSTRAINT sending_domains_tracking_status_check;
ALTER TABLE sending_domains ADD CONSTRAINT sending_domains_tracking_status_check
    CHECK (tracking_status IN ('', 'pending', 'verified', 'wrong_value', 'missing', 'error', 'refused'));

-- +goose Down
UPDATE sending_domains SET tracking_status = 'pending' WHERE tracking_status = 'refused';
ALTER TABLE sending_domains DROP CONSTRAINT sending_domains_tracking_status_check;
ALTER TABLE sending_domains ADD CONSTRAINT sending_domains_tracking_status_check
    CHECK (tracking_status IN ('', 'pending', 'verified', 'wrong_value', 'missing', 'error'));
ALTER TABLE sending_domains DROP COLUMN tracking_verified_at;
