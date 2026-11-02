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
-- Applies a domain's classification to up to @lim contacts still waiting
-- for it. It never waits for a contact another transaction has locked (a
-- tier run, an edit): that one keeps waiting for the next pass, and the
-- classifier can't take part in a deadlock (review 2026-11 OP-13).
WITH batch AS (
    SELECT id FROM contacts
    WHERE mailbox_checked_at IS NULL AND mailbox_provider = 'other' AND erased_at IS NULL
      AND split_part(email, '@', 2) = @domain::text
    LIMIT @lim
    FOR UPDATE SKIP LOCKED
)
UPDATE contacts c
SET mailbox_provider = @provider, mailbox_detail = @detail, mailbox_checked_at = now()
FROM batch WHERE c.id = batch.id;
