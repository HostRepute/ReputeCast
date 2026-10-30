-- name: MailboxPendingDomains :many
-- Company domains of contacts that were never classified from their MX
-- records, skipping domains whose last lookup failed a moment ago.
SELECT DISTINCT split_part(email, '@', 2)::text AS domain
FROM contacts
WHERE mailbox_checked_at IS NULL AND mailbox_provider = 'other' AND erased_at IS NULL
  AND (sqlc.narg(workspace_id)::uuid IS NULL OR workspace_id = sqlc.narg(workspace_id)::uuid)
  AND NOT (split_part(email, '@', 2) = ANY(@skip::text[]))
LIMIT @lim;

-- name: MailboxClassifyContacts :execrows
-- Applies a domain's classification to every contact still waiting for it.
UPDATE contacts
SET mailbox_provider = @provider, mailbox_detail = @detail, mailbox_checked_at = now()
WHERE mailbox_checked_at IS NULL AND mailbox_provider = 'other' AND erased_at IS NULL
  AND split_part(email, '@', 2) = @domain::text;
