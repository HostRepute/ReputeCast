-- +goose Up
-- Every read-only report link expires (review AN-16), but 03205 left the
-- links made before it without an expiry: they kept working until someone
-- revoked them. Each of those links that still works now expires 30 days
-- after this migration runs, the default for a new link, which gives the
-- people who use one a month to ask for a new link. Revoked links stay as
-- they were.
UPDATE report_shares SET expires_at = now() + interval '30 days'
WHERE expires_at IS NULL AND revoked_at IS NULL;

-- +goose Down
-- The links keep their expiry: a link that never expires is what this
-- migration ends, and the links it gave one can't be told apart from links
-- made for 30 days.
SELECT 1;
