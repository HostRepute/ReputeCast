-- The command palette's search (review 2026-11 FE-1): five of a kind,
-- names starting with the text first. @pattern is '%text%' with LIKE's
-- wildcards escaped; @prefix is 'text%'.

-- name: SearchCampaigns :many
SELECT id, name, status, channel, audience_count, started_at, scheduled_at
FROM campaigns
WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND archived_at IS NULL AND digest_role <> 'template'
  AND name ILIKE @pattern::text
ORDER BY (name ILIKE @prefix::text) DESC, coalesce(started_at, scheduled_at, updated_at) DESC
LIMIT 5;

-- name: SearchContacts :many
-- Uses the trigram indexes on email and name (migration 03372); an address
-- with "@" is matched exactly. A member scoped to lists finds only their
-- contacts.
SELECT id, email, first_name, last_name, status
FROM contacts c
WHERE c.workspace_id = @workspace_id AND c.erased_at IS NULL
  AND (coalesce(cardinality(sqlc.arg(scope)::uuid[]), 0) = 0
       OR EXISTS (SELECT 1 FROM list_memberships lm WHERE lm.contact_id = c.id AND lm.list_id = ANY (sqlc.arg(scope)::uuid[])))
  AND CASE WHEN @exact::text <> '' THEN c.email = @exact::text
           ELSE c.email LIKE @pattern::text OR lower(c.first_name || ' ' || c.last_name) LIKE @pattern::text END
ORDER BY (c.email LIKE @prefix::text) DESC, c.created_at DESC
LIMIT 5;

-- name: SearchAutomations :many
SELECT id, name, status FROM automations
WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND name ILIKE @pattern::text
ORDER BY (name ILIKE @prefix::text) DESC, updated_at DESC
LIMIT 5;

-- name: SearchLists :many
SELECT l.id, l.name,
       (SELECT count(*) FROM list_memberships m WHERE m.list_id = l.id AND m.status = 'subscribed')::bigint AS subscribed
FROM lists l
WHERE l.workspace_id = @workspace_id AND l.deleted_at IS NULL AND l.name ILIKE @pattern::text
  AND (coalesce(cardinality(sqlc.arg(scope)::uuid[]), 0) = 0 OR l.id = ANY (sqlc.arg(scope)::uuid[]))
ORDER BY (l.name ILIKE @prefix::text) DESC, l.created_at DESC
LIMIT 5;

-- name: SearchSegments :many
SELECT id, name, cached_count FROM segments
WHERE workspace_id = @workspace_id AND name ILIKE @pattern::text
ORDER BY (name ILIKE @prefix::text) DESC, updated_at DESC
LIMIT 5;

-- name: SearchForms :many
SELECT id, name, kind, status FROM forms
WHERE workspace_id = @workspace_id AND name ILIKE @pattern::text
ORDER BY (name ILIKE @prefix::text) DESC, created_at DESC
LIMIT 5;

-- name: SearchPages :many
SELECT id, name, status, slug FROM pages
WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND name ILIKE @pattern::text
ORDER BY (name ILIKE @prefix::text) DESC, updated_at DESC
LIMIT 5;
