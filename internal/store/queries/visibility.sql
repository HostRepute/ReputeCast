-- Conditional content (internal/visibility, design expansion M21).

-- name: VisibilityLists :many
-- The workspace's lists with their subscribed members, for the condition
-- picker and the rules' words.
SELECT l.id, l.name,
       (SELECT count(*) FROM list_memberships m WHERE m.list_id = l.id AND m.status = 'subscribed')::bigint AS members
FROM lists l
WHERE l.workspace_id = @workspace_id AND l.deleted_at IS NULL
ORDER BY l.created_at;

-- name: VisibilitySegments :many
SELECT id, name, coalesce(cached_count, 0)::bigint AS members
FROM segments
WHERE workspace_id = @workspace_id
ORDER BY name;

-- name: VisibilityFields :many
SELECT key, label, type, options
FROM custom_fields
WHERE workspace_id = @workspace_id
ORDER BY key;

-- name: VisibilityTags :many
-- Tags in use, most used first.
SELECT t.tag::text AS tag, count(*)::bigint AS contacts
FROM contacts c, unnest(c.tags) AS t(tag)
WHERE c.workspace_id = @workspace_id AND c.erased_at IS NULL
GROUP BY t.tag
ORDER BY count(*) DESC, t.tag
LIMIT 200;

-- name: VisibilityRecipients :many
-- An audience as Send Readiness counts it (active contacts on the lists or
-- in the segments, minus excluded lists and segments), with what Visibility
-- conditions read. Suppressed, role and duplicate addresses and contacts
-- the sunset policy stopped are left out by the caller, as in Send
-- Readiness; the oldest contact of a mailbox is kept.
SELECT c.id, c.email, c.first_name, c.last_name, c.tier, c.language, c.tags, c.attributes, c.role_address,
       (c.broadcasts_stopped_at IS NOT NULL)::boolean AS broadcasts_stopped,
       coalesce((SELECT array_agg(m.list_id) FROM list_memberships m WHERE m.contact_id = c.id AND m.status = 'subscribed'), '{}')::uuid[] AS list_ids,
       EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = c.workspace_id AND s.email = c.email) AS suppressed
FROM contacts c
WHERE c.workspace_id = @workspace_id AND c.status = 'active' AND c.erased_at IS NULL
  AND (EXISTS (SELECT 1 FROM list_memberships m WHERE m.contact_id = c.id AND m.status = 'subscribed' AND m.list_id = ANY(ARRAY(SELECT ll.id FROM lists ll WHERE ll.id = ANY(@lists::uuid[]) AND ll.deleted_at IS NULL)))
       OR c.id = ANY(@contacts::uuid[]))
  AND NOT EXISTS (SELECT 1 FROM list_memberships x WHERE x.contact_id = c.id AND x.status = 'subscribed' AND x.list_id = ANY(@exclude_lists::uuid[]))
  AND NOT (c.id = ANY(@exclude_contacts::uuid[]))
ORDER BY c.created_at, c.id;

-- name: VisibilityContactFacts :many
-- What conditions read about some contacts (a send's audience page, a
-- preview).
SELECT c.id, c.email, c.first_name, c.last_name, c.tier, c.language, c.tags, c.attributes,
       coalesce((SELECT array_agg(m.list_id) FROM list_memberships m WHERE m.contact_id = c.id AND m.status = 'subscribed'), '{}')::uuid[] AS list_ids
FROM contacts c
WHERE c.workspace_id = @workspace_id AND c.id = ANY(@ids::uuid[]);

-- name: VisibilityDeletions :many
-- Who deleted the segments, lists and fields rules refer to, and when (the
-- broken condition says "Marcus Chen deleted this segment on Tue 14 Oct").
SELECT DISTINCT ON (a.target_type, a.target_id, a.target_name) a.action, a.target_id, a.target_name, a.actor_name, a.occurred_at
FROM audit_events a
WHERE a.workspace_id = @workspace_id AND a.action IN ('segment.delete', 'list.delete', 'field.delete')
  AND (a.target_id = ANY(@ids::text[]) OR a.target_name = ANY(@names::text[]))
ORDER BY a.target_type, a.target_id, a.target_name, a.occurred_at DESC;

-- name: VisibilityGetCount :one
SELECT campaign_id, workspace_id, rules_hash, audience_hash, total, versions, shown, factors, counted_at, duration_ms
FROM visibility_counts
WHERE campaign_id = @campaign_id;

-- name: VisibilitySaveCount :exec
INSERT INTO visibility_counts (campaign_id, workspace_id, rules_hash, audience_hash, total, versions, shown, factors, counted_at, duration_ms)
VALUES (@campaign_id, @workspace_id, @rules_hash, @audience_hash, @total, @versions, @shown, @factors, @counted_at, @duration_ms)
ON CONFLICT (campaign_id) DO UPDATE SET
    rules_hash = EXCLUDED.rules_hash, audience_hash = EXCLUDED.audience_hash, total = EXCLUDED.total, versions = EXCLUDED.versions,
    shown = EXCLUDED.shown, factors = EXCLUDED.factors, counted_at = EXCLUDED.counted_at, duration_ms = EXCLUDED.duration_ms;

-- name: VisibilityCampaign :one
SELECT c.id, c.name, c.status, c.audience, c.mjml, c.html, c.ab_test, c.schedule, c.scheduled_at
FROM campaigns c
WHERE c.workspace_id = @workspace_id AND c.id = @id;

-- name: VisibilityUnsent :many
-- Emails that aren't sent yet and have Visibility rules: drafts, campaigns
-- waiting for approval and scheduled ones ("used by", the warning before a
-- segment, list or field is deleted, and the blocks Scoring lists).
SELECT c.id, c.name, c.status, c.mjml, c.scheduled_at, c.audience
FROM campaigns c
WHERE c.workspace_id = @workspace_id AND c.status IN ('draft', 'approval', 'approved', 'scheduled') AND c.mjml LIKE '%rc-visible%'
ORDER BY c.updated_at DESC;

-- name: VisibilityMessageHidden :one
-- The blocks a message's recipient doesn't get (view online).
SELECT hidden_blocks FROM messages WHERE id = @id AND workspace_id = @workspace_id;
