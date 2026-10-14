-- Sync of the shared suppression list with each provider. The list
-- itself (screens, manual adds, imports) belongs to the audience domain.

-- name: SuppSyncList :many
SELECT * FROM provider_suppression_sync WHERE provider_id = ANY(@provider_ids::uuid[]);

-- name: SuppSyncGet :one
SELECT * FROM provider_suppression_sync WHERE provider_id = @provider_id;

-- name: SuppSyncUpsert :exec
INSERT INTO provider_suppression_sync (provider_id, mode, last_push_at, last_pull_at, pushed, pulled, pending, status, status_detail)
VALUES (@provider_id, @mode, @last_push_at, @last_pull_at, @pushed, @pulled, @pending, @status, @status_detail)
ON CONFLICT (provider_id) DO UPDATE
SET mode = EXCLUDED.mode, last_push_at = EXCLUDED.last_push_at, last_pull_at = EXCLUDED.last_pull_at,
    pushed = EXCLUDED.pushed, pulled = EXCLUDED.pulled, pending = EXCLUDED.pending,
    status = EXCLUDED.status, status_detail = EXCLUDED.status_detail;

-- name: SuppCount :one
SELECT count(*)::bigint FROM suppressions WHERE workspace_id = @workspace_id;

-- name: SuppReasonCounts :many
SELECT reason, count(*)::bigint AS n FROM suppressions WHERE workspace_id = @workspace_id GROUP BY reason ORDER BY 2 DESC;

-- name: SuppImportSources :many
-- Suppressions brought in by imports ("Brevo (import)"), per source.
SELECT source, count(*)::bigint AS n, min(created_at)::timestamptz AS first_at
FROM suppressions
WHERE workspace_id = @workspace_id AND source LIKE 'import:%'
GROUP BY source ORDER BY 3;

-- name: SuppToPush :many
-- Addresses of the shared list created after a time that didn't come from
-- the provider being pushed to (no point echoing its own list back).
SELECT email, reason FROM suppressions
WHERE workspace_id = @workspace_id AND created_at > @since
  AND (source_provider_id IS NULL OR source_provider_id <> @provider_id)
ORDER BY created_at
LIMIT @lim;

-- name: SuppPendingCount :one
SELECT count(*)::bigint FROM suppressions
WHERE workspace_id = @workspace_id AND created_at > @since
  AND (source_provider_id IS NULL OR source_provider_id <> @provider_id);

-- name: SuppLatestCreated :one
SELECT coalesce(max(created_at), '-infinity'::timestamptz)::timestamptz FROM suppressions WHERE workspace_id = @workspace_id;

-- name: SuppAddedByProvider :many
-- Addresses each provider's bounce and complaint events added to the shared
-- list (the "pulled in" count of providers without a suppression API).
SELECT source_provider_id::uuid AS provider_id, count(*)::bigint AS n, max(created_at)::timestamptz AS last_at
FROM suppressions
WHERE workspace_id = @workspace_id AND source_provider_id IS NOT NULL
GROUP BY source_provider_id;
