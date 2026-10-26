-- name: ComposerVersionsPage :many
-- One page of the version history, newest first: versions older than
-- @before (none: from the newest), only named ones when @named_only.
SELECT v.version, v.saved_at, v.label, v.source, v.size_bytes, v.name, coalesce(u.name, '')::text AS saved_by_name
FROM campaign_versions v
LEFT JOIN users u ON u.id = v.saved_by
WHERE v.campaign_id = @campaign_id
  AND (sqlc.narg(before)::int IS NULL OR v.version < sqlc.narg(before)::int)
  AND (NOT @named_only::bool OR v.name <> '')
ORDER BY v.version DESC
LIMIT @lim;

-- name: ComposerNamedVersionCount :one
SELECT count(*)::bigint FROM campaign_versions WHERE campaign_id = @campaign_id AND name <> '';

-- name: ComposerNameVersion :one
-- Names (or unnames, with '') one version; returns the name it had.
UPDATE campaign_versions v SET name = @name
FROM (SELECT x.name AS old_name FROM campaign_versions x WHERE x.campaign_id = @campaign_id AND x.version = @version FOR UPDATE) o
WHERE v.campaign_id = @campaign_id AND v.version = @version
RETURNING o.old_name::text AS old_name;

-- name: ComposerBumpVersion :exec
-- The current version number moves on without the content changing
-- ("Load theirs" keeps your edits in the history underneath).
UPDATE campaigns
SET version = @version, updated_by = @updated_by, updated_at = @now
WHERE workspace_id = @workspace_id AND id = @id;

-- name: ComposerCountMissingFields :many
-- For each field an email uses, how many of the audience have no value for
-- it (they see the fallback, or nothing): the contacts of the lists minus
-- the excluded lists and suppressions, as ComposerCountListAudience counts
-- them. "first_name" and "last_name" are the contact's own columns; any
-- other key is a custom field in attributes. Each row also carries the
-- number counted. An empty audience returns no rows.
WITH audience AS (
    SELECT DISTINCT c.id, c.first_name, c.last_name, c.attributes
    FROM contacts c
    JOIN list_memberships m ON m.contact_id = c.id
    WHERE c.workspace_id = @workspace_id AND c.status = 'active' AND c.erased_at IS NULL
      AND m.status = 'subscribed' AND m.list_id = ANY(ARRAY(SELECT ll.id FROM lists ll WHERE ll.id = ANY(@lists::uuid[]) AND ll.deleted_at IS NULL))
      AND NOT EXISTS (SELECT 1 FROM list_memberships x WHERE x.contact_id = c.id AND x.status = 'subscribed' AND x.list_id = ANY(@exclude_lists::uuid[]))
      AND NOT EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = c.workspace_id AND s.email = c.email)
)
SELECT k.key::text AS key,
       count(*) FILTER (WHERE CASE k.key
            WHEN 'first_name' THEN btrim(a.first_name) = ''
            WHEN 'last_name' THEN btrim(a.last_name) = ''
            ELSE coalesce(btrim(a.attributes ->> k.key), '') = ''
        END)::bigint AS missing,
       count(*)::bigint AS counted
FROM unnest(@keys::text[]) AS k(key)
CROSS JOIN audience a
GROUP BY k.key;

-- name: ComposerOpenApproval :one
-- A campaign's pending or approved request: who asked, when it was
-- decided and the version it covers (editing cancels it).
SELECT r.id, r.status, r.requested_at, r.decided_at, coalesce(r.campaign_version, 0)::int AS campaign_version,
       coalesce(u.name, '')::text AS requested_by_name
FROM approval_requests r
LEFT JOIN users u ON u.id = r.requested_by
WHERE r.campaign_id = @campaign_id AND r.status IN ('pending', 'approved')
ORDER BY r.requested_at DESC
LIMIT 1;

-- name: ComposerApprovalApprovers :many
-- Who approved a request, in order.
SELECT coalesce(u.name, '')::text AS name
FROM approval_decisions d
LEFT JOIN users u ON u.id = d.approver_id
WHERE d.request_id = @request_id AND d.decision = 'approve'
ORDER BY d.created_at;
