-- Approvals for publishing posts of the publication website (design
-- expansion phase 7): one request covers a post as it is (its hash: the
-- content, the address, the visibility and the publish time).

-- name: PostApprovalLatest :one
-- The latest request about a post, pending first.
SELECT r.*,
       coalesce(u.name, '')::text AS requested_by_name,
       coalesce(d.name, '')::text AS decided_by_name,
       coalesce(r.snapshot ->> 'draft_hash', '')::text AS draft_hash,
       coalesce((SELECT x.comment FROM approval_decisions x WHERE x.request_id = r.id ORDER BY x.created_at DESC LIMIT 1), '')::text AS last_comment
FROM approval_requests r
LEFT JOIN users u ON u.id = r.requested_by
LEFT JOIN users d ON d.id = r.decided_by
WHERE r.website_post_id = @post_id AND r.status <> 'cancelled'
ORDER BY CASE r.status WHEN 'pending' THEN 0 ELSE 1 END, r.requested_at DESC
LIMIT 1;

-- name: PostApprovalForHash :one
-- The approved request that covers a post as it is, if any.
SELECT id FROM approval_requests
WHERE website_post_id = @post_id AND status = 'approved' AND snapshot ->> 'draft_hash' = @draft_hash::text
ORDER BY decided_at DESC NULLS LAST
LIMIT 1;

-- name: PostApprovalPending :one
SELECT r.id, coalesce(r.snapshot ->> 'draft_hash', '')::text AS draft_hash, r.requested_at, coalesce(u.name, '')::text AS requested_by_name
FROM approval_requests r LEFT JOIN users u ON u.id = r.requested_by
WHERE r.website_post_id = @post_id AND r.status = 'pending'
LIMIT 1;

-- name: PostApprovalCancelPending :exec
-- A new request replaces a pending one.
UPDATE approval_requests SET status = 'cancelled', decided_at = @now
WHERE website_post_id = @post_id AND status = 'pending';

-- name: PostApprovalInvalidate :many
-- A post that changed after its request: pending and approved requests for
-- the post as it was are invalidated (a change needs a new approval).
UPDATE approval_requests
SET status = 'invalidated', invalidated_at = @now, invalidated_reason = @reason
WHERE website_post_id = @post_id AND status IN ('pending', 'approved') AND coalesce(snapshot ->> 'draft_hash', '') <> @draft_hash::text
  AND NOT (snapshot ? 'published')
RETURNING id, requested_by;

-- name: PostApprovalMarkPublished :exec
-- The approved request a publish relied on remembers it went out.
UPDATE approval_requests
SET snapshot = jsonb_set(snapshot, '{published}', 'true'::jsonb)
WHERE website_post_id = @post_id AND status = 'approved' AND snapshot ->> 'draft_hash' = @draft_hash::text;
