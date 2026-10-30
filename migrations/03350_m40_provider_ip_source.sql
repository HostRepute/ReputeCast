-- Where a provider's sending IPs (ip_mode and ips) come from (review GS-10):
-- its API, which ReputeCast asks when the provider is connected, when its
-- connection step is repeated and every day (providers.IPReporter), or a
-- person who set them by hand for a provider whose API can't list them.
-- It keeps the last check and its error, and who set the IPs and when:
-- {"from":"provider","checkedAt":"…"} or
-- {"from":"manual","setBy":"…","setByName":"Linh Nguyen","setAt":"…"}.
-- '{}' is nobody yet: what connecting set.

-- +goose Up
ALTER TABLE providers ADD COLUMN ip_source jsonb NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE providers DROP COLUMN ip_source;
