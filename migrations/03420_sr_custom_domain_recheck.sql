-- +goose Up
-- A custom domain verified before its host had to be free of other
-- workspaces' uses is checked again on the domains
-- job's next run, within ten minutes, rather than at its daily check: a
-- host another workspace used first stops serving these pages then, unless
-- this workspace's TXT record proves it's its own.
UPDATE public_page_settings SET domain_checked_at = NULL
WHERE custom_domain IS NOT NULL AND domain_status = 'verified';

-- +goose Down
-- Nothing to undo: each check sets domain_checked_at again.
SELECT 1;
