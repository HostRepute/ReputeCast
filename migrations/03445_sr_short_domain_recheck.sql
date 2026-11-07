-- +goose Up
-- A short-link domain verified before its host had to be free of other
-- workspaces' uses is checked again on the short-link job's first run,
-- within ten minutes, rather than once its day comes round: a host another
-- workspace verified first stops carrying this workspace's links then,
-- unless this workspace's TXT record shows the host is its own.
UPDATE sms_settings SET short_domain_checked_at = NULL
WHERE short_domain <> '' AND short_domain_status = 'verified';

-- +goose Down
-- Nothing to undo: each check sets short_domain_checked_at again.
SELECT 1;
