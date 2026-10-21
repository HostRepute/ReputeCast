-- Approvals for publishing landing and sales pages (internal/pages, design
-- expansion phase 7, M29): the workspace rule "Publishing a post or page
-- needs approval" and "The author lacks the send permission" cover pages.
-- One request covers a page as it is (its hash: the content, the address,
-- the settings and the publish time).

-- name: PageApprovalForHash :one
-- The approved request that covers a page as it is, if any.
SELECT id FROM approval_requests
WHERE website_page_id = @page_id AND status = 'approved' AND snapshot ->> 'draft_hash' = @draft_hash::text
ORDER BY decided_at DESC NULLS LAST
LIMIT 1;

-- name: PageApprovalPending :one
SELECT r.id, coalesce(r.snapshot ->> 'draft_hash', '')::text AS draft_hash, r.requested_at, coalesce(u.name, '')::text AS requested_by_name
FROM approval_requests r LEFT JOIN users u ON u.id = r.requested_by
WHERE r.website_page_id = @page_id AND r.status = 'pending'
LIMIT 1;

-- name: PageApprovalCancelPending :exec
-- A new request replaces a pending one.
UPDATE approval_requests SET status = 'cancelled', decided_at = @now
WHERE website_page_id = @page_id AND status = 'pending';

-- name: PageApprovalInvalidate :many
-- A page that changed after its request: pending and approved requests for
-- the page as it was are invalidated (a change needs a new approval).
UPDATE approval_requests
SET status = 'invalidated', invalidated_at = @now, invalidated_reason = @reason
WHERE website_page_id = @page_id AND status IN ('pending', 'approved') AND coalesce(snapshot ->> 'draft_hash', '') <> @draft_hash::text
  AND NOT (snapshot ? 'published')
RETURNING id, requested_by;

-- name: PageApprovalMarkPublished :exec
-- The approved request a publish relied on remembers it went out.
UPDATE approval_requests
SET snapshot = jsonb_set(snapshot, '{published}', 'true'::jsonb)
WHERE website_page_id = @page_id AND status = 'approved' AND snapshot ->> 'draft_hash' = @draft_hash::text;
