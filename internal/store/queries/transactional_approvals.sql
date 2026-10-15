-- Approvals for publishing transactional templates (add-on transactional §3.6).

-- name: TxnApprovalLatest :one
-- The latest request about a template, pending first.
SELECT r.*,
       coalesce(u.name, '')::text AS requested_by_name,
       coalesce(d.name, '')::text AS decided_by_name,
       coalesce(r.snapshot ->> 'draft_hash', '')::text AS draft_hash,
       coalesce((SELECT x.comment FROM approval_decisions x WHERE x.request_id = r.id ORDER BY x.created_at DESC LIMIT 1), '')::text AS last_comment
FROM approval_requests r
LEFT JOIN users u ON u.id = r.requested_by
LEFT JOIN users d ON d.id = r.decided_by
WHERE r.transactional_template_id = @template_id AND r.status <> 'cancelled'
ORDER BY CASE r.status WHEN 'pending' THEN 0 ELSE 1 END, r.requested_at DESC
LIMIT 1;

-- name: TxnApprovalForHash :one
-- The approved request that covers a draft, if any.
SELECT id FROM approval_requests
WHERE transactional_template_id = @template_id AND status = 'approved' AND snapshot ->> 'draft_hash' = @draft_hash::text
ORDER BY decided_at DESC NULLS LAST
LIMIT 1;

-- name: TxnApprovalPending :one
SELECT id, coalesce(snapshot ->> 'draft_hash', '')::text AS draft_hash FROM approval_requests
WHERE transactional_template_id = @template_id AND status = 'pending'
LIMIT 1;

-- name: TxnApprovalCancelPending :exec
-- A newer draft replaces the request for an older one.
UPDATE approval_requests SET status = 'cancelled', decided_at = @now
WHERE transactional_template_id = @template_id AND status = 'pending';

-- name: TxnApprovalMarkPublished :exec
-- The approved request a publish relied on remembers the version it made live.
UPDATE approval_requests
SET snapshot = jsonb_set(snapshot, '{published_version}', to_jsonb(@version::integer))
WHERE id = @id;
