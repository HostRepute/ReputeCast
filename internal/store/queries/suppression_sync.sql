-- Sync of the shared suppression list with each provider. The list
-- itself (screens, manual adds, imports) belongs to the audience domain.

-- name: SuppSyncList :many
SELECT * FROM provider_suppression_sync WHERE provider_id = ANY(@provider_ids::uuid[]);

-- name: SuppSyncGet :one
SELECT * FROM provider_suppression_sync WHERE provider_id = @provider_id;

-- name: SuppSyncUpsert :exec
INSERT INTO provider_suppression_sync (provider_id, mode, last_push_at, last_pull_at, pushed, pulled, pending, status, status_detail, scope, push_seq, removed)
VALUES (@provider_id, @mode, @last_push_at, @last_pull_at, @pushed, @pulled, @pending, @status, @status_detail, @scope, @push_seq, @removed)
ON CONFLICT (provider_id) DO UPDATE
SET mode = EXCLUDED.mode, last_push_at = EXCLUDED.last_push_at, last_pull_at = EXCLUDED.last_pull_at,
    pushed = EXCLUDED.pushed, pulled = EXCLUDED.pulled, pending = EXCLUDED.pending,
    status = EXCLUDED.status, status_detail = EXCLUDED.status_detail,
    scope = EXCLUDED.scope, push_seq = EXCLUDED.push_seq, removed = EXCLUDED.removed;

-- name: SuppCount :one
SELECT count(*)::bigint FROM suppressions WHERE workspace_id = @workspace_id;

-- name: SuppReasonCounts :many
SELECT reason, count(*)::bigint AS n FROM suppressions WHERE workspace_id = @workspace_id GROUP BY reason ORDER BY 2 DESC;

-- name: SuppPushSummary :many
-- The shared list by reason and scope, and whether providers get it (PR-1:
-- hard bounces and complaints that apply to all mail).
SELECT reason, scope, (scope = 'all' AND reason IN ('hard_bounce', 'complaint'))::boolean AS pushed, count(*)::bigint AS n
FROM suppressions WHERE workspace_id = @workspace_id
GROUP BY reason, scope ORDER BY pushed DESC, n DESC, reason;

-- name: SuppImportSources :many
-- Suppressions brought in by imports ("Brevo (import)"), per source.
SELECT source, count(*)::bigint AS n, min(created_at)::timestamptz AS first_at
FROM suppressions
WHERE workspace_id = @workspace_id AND source LIKE 'import:%'
GROUP BY source ORDER BY 3;

-- name: SuppToPush :many
-- PR-1: suppressions a provider's list should hold and doesn't yet, in
-- insert order: hard bounces and complaints that apply to all mail (never
-- unsubscribes, role addresses or manual marketing-only entries, which
-- would stop a password reset), not the provider's own (no echo).
SELECT s.email, s.reason, s.seq FROM suppressions s
WHERE s.workspace_id = @workspace_id AND s.scope = 'all' AND s.reason IN ('hard_bounce', 'complaint')
  AND (s.source_provider_id IS NULL OR s.source_provider_id <> @provider_id)
  AND NOT EXISTS (SELECT 1 FROM provider_suppression_entries e WHERE e.provider_id = @provider_id AND e.email = s.email)
ORDER BY s.seq
LIMIT @lim;

-- name: SuppToRemove :many
-- Entries of a provider's list that left the shared list: a pushed one
-- whose suppression is gone (or no longer applies to all mail), a pulled
-- one someone removed from the shared list.
SELECT e.email, e.origin FROM provider_suppression_entries e
WHERE e.provider_id = @provider_id
  AND NOT EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = @workspace_id AND s.email = e.email
                  AND (e.origin = 'pulled' OR (s.scope = 'all' AND s.reason IN ('hard_bounce', 'complaint'))))
ORDER BY e.email
LIMIT @lim;

-- name: SuppPendingCount :one
-- What a sync still has to do for a provider: suppressions to push plus
-- entries to remove.
SELECT ((SELECT count(*) FROM suppressions s
         WHERE s.workspace_id = @workspace_id AND s.scope = 'all' AND s.reason IN ('hard_bounce', 'complaint')
           AND (s.source_provider_id IS NULL OR s.source_provider_id <> @provider_id)
           AND NOT EXISTS (SELECT 1 FROM provider_suppression_entries e WHERE e.provider_id = @provider_id AND e.email = s.email))
      + (SELECT count(*) FROM provider_suppression_entries e
         WHERE e.provider_id = @provider_id
           AND NOT EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = @workspace_id AND s.email = e.email
                           AND (e.origin = 'pulled' OR (s.scope = 'all' AND s.reason IN ('hard_bounce', 'complaint'))))))::bigint;

-- name: SuppEntriesAdd :exec
-- Records addresses a provider's list now holds. A pulled address that was
-- pushed before stays "pushed".
INSERT INTO provider_suppression_entries (provider_id, email, origin, seq, synced_at)
SELECT @provider_id::uuid, unnest(@emails::text[]), @origin::text, unnest(@seqs::bigint[]), @synced_at::timestamptz
ON CONFLICT (provider_id, email) DO NOTHING;

-- name: SuppEntriesDelete :exec
DELETE FROM provider_suppression_entries WHERE provider_id = @provider_id AND email = ANY(@emails::text[]);

-- name: SuppAddedByProvider :many
-- Addresses each provider's bounce and complaint events added to the shared
-- list (the "pulled in" count of providers without a suppression API).
SELECT source_provider_id::uuid AS provider_id, count(*)::bigint AS n, max(created_at)::timestamptz AS last_at
FROM suppressions
WHERE workspace_id = @workspace_id AND source_provider_id IS NOT NULL
GROUP BY source_provider_id;

-- name: SuppEntriesKnown :many
-- Which of these addresses a provider's list already holds as far as
-- ReputeCast knows (pushed there, or pulled before): a pull skips them, so
-- an address a person removed from the shared list isn't pulled back in
-- before its removal reaches the provider.
SELECT email FROM provider_suppression_entries WHERE provider_id = @provider_id AND email = ANY(@emails::text[]);
