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

-- name: CampaignsHoldingSynced :many
-- Campaigns whose content holds a synced block (an element with
-- rc-synced="<kind>"), oldest first (CO-11). pattern is
-- %rc-synced="<kind>"%.
SELECT c.id, c.status,
       EXISTS (SELECT 1 FROM approval_requests r WHERE r.campaign_id = c.id AND r.status IN ('pending', 'approved'))::bool AS in_approval
FROM campaigns c
WHERE c.workspace_id = @workspace_id AND c.deleted_at IS NULL AND c.mjml LIKE @pattern::text
ORDER BY c.created_at;

-- name: ComposerMentionable :many
-- The workspace's active people, for @mentions in comments (CO-10).
SELECT u.id, u.name, u.avatar_color, r.name AS role_name
FROM workspace_members m
JOIN roles r ON r.id = m.role_id
JOIN users u ON u.id = m.user_id AND u.status = 'active'
WHERE m.workspace_id = @workspace_id AND m.status = 'active'
ORDER BY u.name;

-- name: CommentInsert :exec
INSERT INTO campaign_comments (id, workspace_id, campaign_id, thread_id, anchor, anchor_label, body, author_id, reviewer_name, review_link_id, version, mentions, created_at)
VALUES (@id, @workspace_id, @campaign_id, sqlc.narg(thread_id), @anchor, @anchor_label, @body, sqlc.narg(author_id), @reviewer_name, sqlc.narg(review_link_id), @version, @mentions, @created_at);

-- name: CommentsList :many
SELECT c.id, c.thread_id, c.anchor, c.anchor_label, c.body, c.author_id, coalesce(u.name, '')::text AS author_name, coalesce(u.avatar_color, '')::text AS author_color, c.reviewer_name,
       c.review_link_id, c.version, c.mentions, c.resolved_at, coalesce(r.name, '')::text AS resolved_by_name, c.created_at
FROM campaign_comments c
LEFT JOIN users u ON u.id = c.author_id
LEFT JOIN users r ON r.id = c.resolved_by
WHERE c.workspace_id = @workspace_id AND c.campaign_id = @campaign_id
ORDER BY c.created_at, c.id;

-- name: CommentsByLink :many
-- What a reviewer sees: the threads started through their link, with the
-- members' replies to them.
SELECT c.id, c.thread_id, c.anchor, c.anchor_label, c.body, c.author_id, coalesce(u.name, '')::text AS author_name, coalesce(u.avatar_color, '')::text AS author_color, c.reviewer_name,
       c.review_link_id, c.version, c.mentions, c.resolved_at, ''::text AS resolved_by_name, c.created_at
FROM campaign_comments c
LEFT JOIN users u ON u.id = c.author_id
WHERE c.review_link_id = @review_link_id
   OR c.thread_id IN (SELECT t.id FROM campaign_comments t WHERE t.review_link_id = @review_link_id AND t.thread_id IS NULL)
ORDER BY c.created_at, c.id;

-- name: CommentGet :one
SELECT id, thread_id, anchor, anchor_label, resolved_at FROM campaign_comments
WHERE workspace_id = @workspace_id AND campaign_id = @campaign_id AND id = @id;

-- name: CommentResolve :execrows
UPDATE campaign_comments SET resolved_at = sqlc.narg(resolved_at), resolved_by = sqlc.narg(resolved_by)
WHERE workspace_id = @workspace_id AND campaign_id = @campaign_id AND id = @id AND thread_id IS NULL;

-- name: ReviewLinkInsert :exec
INSERT INTO campaign_review_links (id, workspace_id, campaign_id, version, token_hash, token_sealed, created_by, created_at, expires_at)
VALUES (@id, @workspace_id, @campaign_id, @version, @token_hash, @token_sealed, sqlc.narg(created_by), @created_at, @expires_at);

-- name: ReviewLinksActive :many
SELECT l.id, l.version, l.token_sealed, l.created_at, l.expires_at, coalesce(u.name, '')::text AS created_by_name
FROM campaign_review_links l
LEFT JOIN users u ON u.id = l.created_by
WHERE l.workspace_id = @workspace_id AND l.campaign_id = @campaign_id AND l.revoked_at IS NULL AND l.expires_at > @now
ORDER BY l.created_at DESC;

-- name: ReviewLinkRevoke :execrows
UPDATE campaign_review_links SET revoked_at = @now, revoked_by = sqlc.narg(revoked_by)
WHERE workspace_id = @workspace_id AND campaign_id = @campaign_id AND id = @id AND revoked_at IS NULL;

-- name: ReviewLinkByToken :one
SELECT l.id, l.workspace_id, l.campaign_id, l.version, l.expires_at, l.revoked_at, c.name AS campaign_name, w.name AS workspace_name
FROM campaign_review_links l
JOIN campaigns c ON c.id = l.campaign_id AND c.deleted_at IS NULL
JOIN workspaces w ON w.id = l.workspace_id AND w.deleted_at IS NULL
WHERE l.token_hash = @token_hash;

-- name: ReviewLinkCommentCount :one
-- Comments through a link since a time: reviewers are limited per hour.
SELECT count(*)::bigint FROM campaign_comments WHERE review_link_id = @review_link_id AND created_at > @since;

-- name: PresenceTouch :exec
INSERT INTO campaign_presence (campaign_id, user_id, editing, seen_at)
VALUES (@campaign_id, @user_id, @editing, @now)
ON CONFLICT (campaign_id, user_id) DO UPDATE SET editing = EXCLUDED.editing, seen_at = EXCLUDED.seen_at;

-- name: PresenceList :many
-- Who else has a campaign open (seen since a time).
SELECT p.user_id, u.name, u.avatar_color, p.editing, p.seen_at
FROM campaign_presence p
JOIN users u ON u.id = p.user_id
WHERE p.campaign_id = @campaign_id AND p.seen_at > @since AND p.user_id <> @me
ORDER BY p.seen_at DESC;

-- name: PresenceLeave :exec
DELETE FROM campaign_presence WHERE campaign_id = @campaign_id AND user_id = @user_id;

-- name: ComposerNameVersionIfUnnamed :execrows
-- Names a version only when nobody named it (a review link's version).
UPDATE campaign_versions SET name = @name
WHERE campaign_id = @campaign_id AND version = @version AND name = '';

-- name: CheckDismissalsList :many
-- The warnings dismissed in a campaign's composer (CO-14).
SELECT d.id, d.issue_key, d.check_id, d.title, d.reason, d.note, coalesce(u.name, '')::text AS dismissed_by_name, d.created_at
FROM campaign_check_dismissals d
LEFT JOIN users u ON u.id = d.dismissed_by
WHERE d.workspace_id = @workspace_id AND d.campaign_id = @campaign_id
ORDER BY d.created_at;

-- name: CheckDismissalInsert :one
INSERT INTO campaign_check_dismissals (id, workspace_id, campaign_id, issue_key, check_id, title, reason, note, dismissed_by, created_at)
VALUES (@id, @workspace_id, @campaign_id, @issue_key, @check_id, @title, @reason, @note, sqlc.narg(dismissed_by), @created_at)
ON CONFLICT (campaign_id, issue_key) DO UPDATE SET reason = EXCLUDED.reason, note = EXCLUDED.note, dismissed_by = EXCLUDED.dismissed_by, created_at = EXCLUDED.created_at
RETURNING id;

-- name: CheckDismissalDelete :one
DELETE FROM campaign_check_dismissals WHERE workspace_id = @workspace_id AND campaign_id = @campaign_id AND id = @id
RETURNING check_id, title, reason;
