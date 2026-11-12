-- Comments on posts (internal/website/comments*.go, M61): what members
-- write under the posts they can read, the team's replies, moderation in
-- Website › Comments and the emails that tell a member someone replied.

-- Public: a post's threads as one reader sees them

-- name: WebsiteCommentThreads :many
-- A page of a post's top-level comments as a reader sees them: the
-- published ones, the reader's own waiting for review, and removed ones
-- that still have published replies. Pinned first, then oldest first.
SELECT sqlc.embed(c), coalesce(m.display_name, '')::text AS display_name,
       coalesce(ct.first_name, '')::text AS first_name, coalesce(ct.last_name, '')::text AS last_name
FROM website_comments c
LEFT JOIN website_members m ON m.contact_id = c.contact_id
LEFT JOIN contacts ct ON ct.id = c.contact_id
WHERE c.post_id = @post_id AND c.parent_id IS NULL
  AND (c.status = 'published' OR (c.status = 'pending' AND c.contact_id = sqlc.narg(viewer)::uuid)
       OR (c.status IN ('hidden','deleted') AND EXISTS (SELECT 1 FROM website_comments r WHERE r.parent_id = c.id AND r.status = 'published')))
ORDER BY (CASE WHEN c.status = 'published' THEN c.pinned_at END) NULLS LAST, c.created_at, c.id
LIMIT @lim OFFSET @off;

-- name: WebsiteCommentThreadCount :one
-- How many top-level comments the reader sees (the pages).
SELECT count(*)::bigint FROM website_comments c
WHERE c.post_id = @post_id AND c.parent_id IS NULL
  AND (c.status = 'published' OR (c.status = 'pending' AND c.contact_id = sqlc.narg(viewer)::uuid)
       OR (c.status IN ('hidden','deleted') AND EXISTS (SELECT 1 FROM website_comments r WHERE r.parent_id = c.id AND r.status = 'published')));

-- name: WebsiteCommentThreadsBefore :one
-- How many top-level comments the reader sees before one (its page):
-- pinned is its pinned_at while it is published and pinned.
SELECT count(*)::bigint FROM website_comments c
WHERE c.post_id = @post_id AND c.parent_id IS NULL AND c.id <> @id::uuid
  AND (c.status = 'published' OR (c.status = 'pending' AND c.contact_id = sqlc.narg(viewer)::uuid)
       OR (c.status IN ('hidden','deleted') AND EXISTS (SELECT 1 FROM website_comments r WHERE r.parent_id = c.id AND r.status = 'published')))
  AND (CASE WHEN sqlc.narg(pinned)::timestamptz IS NOT NULL
       THEN c.status = 'published' AND c.pinned_at IS NOT NULL
            AND (c.pinned_at, c.created_at, c.id) < (sqlc.narg(pinned)::timestamptz, @created_at::timestamptz, @id::uuid)
       ELSE (c.status = 'published' AND c.pinned_at IS NOT NULL)
            OR (NOT (c.status = 'published' AND c.pinned_at IS NOT NULL) AND (c.created_at, c.id) < (@created_at::timestamptz, @id::uuid)) END);

-- name: WebsiteCommentReplies :many
-- The replies of threads, oldest first: published ones and the reader's
-- own waiting for review.
SELECT sqlc.embed(c), coalesce(m.display_name, '')::text AS display_name,
       coalesce(ct.first_name, '')::text AS first_name, coalesce(ct.last_name, '')::text AS last_name
FROM website_comments c
LEFT JOIN website_members m ON m.contact_id = c.contact_id
LEFT JOIN contacts ct ON ct.id = c.contact_id
WHERE c.parent_id = ANY (@parents::uuid[])
  AND (c.status = 'published' OR (c.status = 'pending' AND c.contact_id = sqlc.narg(viewer)::uuid))
ORDER BY c.created_at, c.id;

-- name: WebsiteCommentCount :one
-- A post's published comments, replies included ("Comments (13)").
SELECT count(*)::bigint FROM website_comments WHERE post_id = @post_id AND status = 'published';

-- name: WebsiteCommentCountsByPost :many
-- Website › Posts' comment column.
SELECT post_id, count(*) FILTER (WHERE status = 'published')::bigint AS published, count(*) FILTER (WHERE status = 'pending')::bigint AS pending
FROM website_comments WHERE post_id = ANY (@posts::uuid[]) GROUP BY post_id;

-- Members as commenters

-- name: WebsiteCommenter :one
-- A contact as their comments know them: the name shown, a moderator's
-- block, reply emails.
SELECT c.id, c.first_name, c.last_name, c.email, coalesce(m.display_name, '')::text AS display_name, m.comments_blocked_at,
       coalesce(m.comment_emails, true)::boolean AS comment_emails, (m.contact_id IS NOT NULL)::boolean AS member
FROM contacts c LEFT JOIN website_members m ON m.contact_id = c.id
WHERE c.workspace_id = @workspace_id AND c.id = @contact_id;

-- name: WebsiteCommenterName :exec
-- The name a member's comments show (their first comment asks for it).
INSERT INTO website_members (contact_id, workspace_id, first_signed_in, last_signed_in, last_method, display_name)
VALUES (@contact_id, @workspace_id, @at, @at, @method, @display_name)
ON CONFLICT (contact_id) DO UPDATE SET display_name = EXCLUDED.display_name;

-- name: WebsiteCommenterSettings :exec
-- The account page's Comments section: the name and reply emails.
INSERT INTO website_members (contact_id, workspace_id, first_signed_in, last_signed_in, last_method, display_name, comment_emails)
VALUES (@contact_id, @workspace_id, @at, @at, @method, @display_name, @comment_emails)
ON CONFLICT (contact_id) DO UPDATE SET display_name = EXCLUDED.display_name, comment_emails = EXCLUDED.comment_emails;

-- name: WebsiteCommenterMute :execrows
-- "Stop these emails" in a reply email.
UPDATE website_members SET comment_emails = false WHERE contact_id = @contact_id AND workspace_id = @workspace_id;

-- name: WebsiteCommenterBlock :execrows
-- Blocks a member from commenting (at) or lets them again (NULL).
UPDATE website_members SET comments_blocked_at = sqlc.narg(at) WHERE contact_id = @contact_id AND workspace_id = @workspace_id;

-- name: WebsiteCommentsHoldBlocked :execrows
-- A blocked member's comments waiting for review say why they wait.
UPDATE website_comments SET held_reason = 'blocked' WHERE workspace_id = @workspace_id AND contact_id = @contact_id AND status = 'pending';

-- name: WebsiteCommenterHasPublished :one
-- Whether a member has a published comment (review "a member's first comment").
SELECT EXISTS (SELECT 1 FROM website_comments WHERE contact_id = @contact_id AND status = 'published')::boolean;

-- name: WebsiteCommentsByMember :one
-- A member's comments since day_start, and since recent (the limits).
SELECT count(*) FILTER (WHERE created_at > @recent::timestamptz)::bigint AS recent, count(*)::bigint AS day
FROM website_comments WHERE contact_id = @contact_id AND created_at > @day_start::timestamptz;

-- name: WebsiteCommentsByNetwork :one
-- Comments from one network (an IPv4 address or an IPv6 /64) since a moment.
SELECT count(*)::bigint FROM website_comments
WHERE workspace_id = @workspace_id AND created_at > @since AND ip IS NOT NULL AND ip <<= (@network::text)::inet;

-- Writing

-- name: WebsiteCommentInsert :exec
INSERT INTO website_comments (id, workspace_id, post_id, parent_id, contact_id, team, user_id, body, status, held_reason, ip, created_at, published_at)
VALUES (@id, @workspace_id, @post_id, sqlc.narg(parent_id), sqlc.narg(contact_id), @team, sqlc.narg(user_id), @body, @status, @held_reason, sqlc.narg(ip), @at,
        sqlc.narg(published_at));

-- name: WebsiteCommentGet :one
SELECT * FROM website_comments WHERE workspace_id = @workspace_id AND id = @id;

-- name: WebsiteCommentLock :one
SELECT * FROM website_comments WHERE workspace_id = @workspace_id AND id = @id FOR UPDATE;

-- name: WebsiteCommentEdit :exec
-- A member's own edit (within 15 minutes): held again when the site
-- reviews every comment.
UPDATE website_comments SET body = @body, status = @status, held_reason = @held_reason, edited_at = @at,
    decision_id = NULL, undo_until = NULL, before_status = '', before_published_at = NULL
WHERE id = @id;

-- name: WebsiteCommentDeleteOwn :exec
-- A member deletes their own comment: its text goes at once.
UPDATE website_comments SET status = 'deleted', body = '', pinned_at = NULL, decision_id = NULL, undo_until = NULL, before_status = '',
    before_published_at = NULL
WHERE id = @id;

-- Moderation

-- name: WebsiteCommentList :many
-- Website › Comments: one tab's comments, newest first, after a cursor.
SELECT sqlc.embed(c), p.title AS post_title, p.slug AS post_slug,
       coalesce(m.display_name, '')::text AS display_name, coalesce(ct.email, '')::text AS email,
       coalesce(ct.first_name, '')::text AS first_name, coalesce(ct.last_name, '')::text AS last_name,
       (m.comments_blocked_at IS NOT NULL)::boolean AS blocked, coalesce(u.name, '')::text AS user_name,
       coalesce(mb.name, '')::text AS moderated_by_name,
       coalesce(nullif(pm.display_name, ''), nullif(trim(coalesce(pcc.first_name, '') || ' ' || coalesce(pcc.last_name, '')), ''), '')::text AS parent_name,
       coalesce(pc.team, false)::boolean AS parent_team,
       (SELECT count(*) FROM website_comments r WHERE r.parent_id = c.id AND r.status = 'published')::bigint AS replies,
       coalesce(fr.body, '')::text AS first_reply_body, coalesce(fr.team, false)::boolean AS first_reply_team,
       coalesce(fr.name, '')::text AS first_reply_name, coalesce(fr.by_name, '')::text AS first_reply_by,
       coalesce(fr.created_at, c.created_at)::timestamptz AS first_reply_at
FROM website_comments c
JOIN website_posts p ON p.id = c.post_id
-- The first published reply, for the thread's preview.
LEFT JOIN LATERAL (
    SELECT r.body, r.team, r.created_at, coalesce(ru.name, '') AS by_name,
           coalesce(nullif(rm.display_name, ''), nullif(trim(coalesce(rc.first_name, '') || ' ' || coalesce(rc.last_name, '')), ''), '') AS name
    FROM website_comments r
    LEFT JOIN website_members rm ON rm.contact_id = r.contact_id
    LEFT JOIN contacts rc ON rc.id = r.contact_id
    LEFT JOIN users ru ON ru.id = r.user_id
    WHERE r.parent_id = c.id AND r.status = 'published'
    ORDER BY r.created_at, r.id
    LIMIT 1
) fr ON true
LEFT JOIN website_members m ON m.contact_id = c.contact_id
LEFT JOIN contacts ct ON ct.id = c.contact_id
LEFT JOIN users u ON u.id = c.user_id
LEFT JOIN users mb ON mb.id = c.moderated_by
LEFT JOIN website_comments pc ON pc.id = c.parent_id
LEFT JOIN website_members pm ON pm.contact_id = pc.contact_id
LEFT JOIN contacts pcc ON pcc.id = pc.contact_id
WHERE c.workspace_id = @workspace_id
  AND (sqlc.narg(status)::text IS NULL OR c.status = sqlc.narg(status)::text)
  AND (sqlc.narg(id)::uuid IS NULL OR c.id = sqlc.narg(id)::uuid)
  AND (sqlc.narg(post)::uuid IS NULL OR c.post_id = sqlc.narg(post)::uuid)
  AND (sqlc.narg(q)::text IS NULL OR c.body ILIKE sqlc.narg(q)::text OR m.display_name ILIKE sqlc.narg(q)::text OR ct.email ILIKE sqlc.narg(q)::text)
  AND (sqlc.narg(before_at)::timestamptz IS NULL
       OR ((c.pinned_at IS NOT NULL)::int, c.created_at, c.id) < (sqlc.narg(before_pinned)::int, sqlc.narg(before_at)::timestamptz, sqlc.narg(before_id)::uuid))
ORDER BY (c.pinned_at IS NOT NULL) DESC, c.created_at DESC, c.id DESC
LIMIT @lim;

-- name: WebsiteCommentStatusCounts :one
-- The tabs' counts, with the same post and search.
SELECT count(*) FILTER (WHERE c.status = 'pending')::bigint AS pending, count(*) FILTER (WHERE c.status = 'published')::bigint AS published,
       count(*) FILTER (WHERE c.status = 'hidden')::bigint AS hidden
FROM website_comments c
LEFT JOIN website_members m ON m.contact_id = c.contact_id
LEFT JOIN contacts ct ON ct.id = c.contact_id
WHERE c.workspace_id = @workspace_id AND c.status IN ('pending','published','hidden')
  AND (sqlc.narg(post)::uuid IS NULL OR c.post_id = sqlc.narg(post)::uuid)
  AND (sqlc.narg(q)::text IS NULL OR c.body ILIKE sqlc.narg(q)::text OR m.display_name ILIKE sqlc.narg(q)::text OR ct.email ILIKE sqlc.narg(q)::text);

-- name: WebsiteCommentPosts :many
-- The posts that have comments (Website › Comments' post filter), newest
-- comment first.
SELECT p.id, p.title, count(*)::bigint AS comments
FROM website_comments c JOIN website_posts p ON p.id = c.post_id AND p.deleted_at IS NULL
WHERE c.workspace_id = @workspace_id AND c.status IN ('pending','published','hidden')
GROUP BY p.id, p.title
ORDER BY max(c.created_at) DESC
LIMIT 200;

-- name: WebsiteCommentsLock :many
SELECT * FROM website_comments WHERE workspace_id = @workspace_id AND id = ANY (@ids::uuid[]) ORDER BY id FOR UPDATE;

-- name: WebsiteCommentLabels :many
-- Who wrote comments and under which post (audit entries name them).
SELECT c.id, c.contact_id, c.team, coalesce(m.display_name, '')::text AS display_name, coalesce(ct.first_name, '')::text AS first_name,
       coalesce(ct.last_name, '')::text AS last_name, p.title AS post_title
FROM website_comments c JOIN website_posts p ON p.id = c.post_id
LEFT JOIN website_members m ON m.contact_id = c.contact_id
LEFT JOIN contacts ct ON ct.id = c.contact_id
WHERE c.workspace_id = @workspace_id AND c.id = ANY (@ids::uuid[]);

-- name: WebsiteCommentsDecide :exec
-- Publishes, hides or deletes comments, keeping what Undo restores.
UPDATE website_comments SET before_status = status, before_published_at = published_at, status = @status::text,
    published_at = CASE WHEN @status::text = 'published' THEN coalesce(published_at, @at::timestamptz) ELSE published_at END,
    moderated_by = sqlc.narg(moderated_by), moderated_at = @at::timestamptz, decision_id = @decision_id, undo_until = @undo_until
WHERE workspace_id = @workspace_id AND id = ANY (@ids::uuid[]);

-- name: WebsiteCommentsOfDecision :many
SELECT * FROM website_comments WHERE workspace_id = @workspace_id AND decision_id = @decision_id ORDER BY id FOR UPDATE;

-- name: WebsiteCommentsUndo :exec
UPDATE website_comments SET status = before_status, published_at = before_published_at, before_status = '', before_published_at = NULL,
    decision_id = NULL, undo_until = NULL
WHERE workspace_id = @workspace_id AND decision_id = @decision_id;

-- name: WebsiteCommentPin :exec
UPDATE website_comments SET pinned_at = sqlc.narg(pinned_at) WHERE workspace_id = @workspace_id AND id = @id;

-- name: WebsiteCommentsPending :one
SELECT count(*)::bigint FROM website_comments WHERE workspace_id = @workspace_id AND status = 'pending';

-- name: WebsiteCommentsNotified :exec
-- When the people who moderate were last told comments are waiting.
UPDATE website_settings SET comments_notified_at = @at WHERE workspace_id = @workspace_id;

-- Reply emails

-- name: WebsiteCommentNoticesQueue :exec
-- Published replies: each thread's author hears about them, unless they
-- wrote the reply or turned reply emails off.
INSERT INTO website_comment_notices (contact_id, comment_id, queued_at)
SELECT p.contact_id, r.id, @at FROM website_comments r
JOIN website_comments p ON p.id = r.parent_id AND p.status = 'published' AND p.contact_id IS NOT NULL
JOIN website_members m ON m.contact_id = p.contact_id AND m.comment_emails
WHERE r.id = ANY (@replies::uuid[]) AND r.status = 'published' AND p.contact_id IS DISTINCT FROM r.contact_id
ON CONFLICT DO NOTHING;

-- name: WebsiteCommentNoticesUnqueue :exec
-- Replies that left the site before their email went.
DELETE FROM website_comment_notices WHERE comment_id = ANY (@comments::uuid[]) AND sent_at IS NULL;

-- name: WebsiteCommentNoticesDue :many
-- Reply emails to send: the notices of members who got none since @since.
SELECT n.contact_id, n.comment_id, n.queued_at, r.workspace_id
FROM website_comment_notices n JOIN website_comments r ON r.id = n.comment_id
WHERE n.sent_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM website_comment_notices s WHERE s.contact_id = n.contact_id AND s.sent_at > @since)
ORDER BY r.workspace_id, n.contact_id, n.queued_at
LIMIT 2000;

-- name: WebsiteCommentNoticeReply :one
-- What a reply email says: the reply, who wrote it and its post.
SELECT r.id, r.body, r.status, r.contact_id, r.team, r.user_id, coalesce(m.display_name, '')::text AS display_name,
       coalesce(ac.first_name, '')::text AS first_name, coalesce(ac.last_name, '')::text AS last_name,
       p.title AS post_title, p.slug AS post_slug, p.status AS post_status, (p.deleted_at IS NULL)::boolean AS post_live,
       pc.id AS parent_id, pc.status AS parent_status, pc.body AS parent_body, pc.post_id AS parent_post, r.workspace_id
FROM website_comments r
JOIN website_posts p ON p.id = r.post_id
JOIN website_comments pc ON pc.id = r.parent_id
LEFT JOIN website_members m ON m.contact_id = r.contact_id
LEFT JOIN contacts ac ON ac.id = r.contact_id
WHERE r.id = @id;

-- name: WebsiteCommentRecipient :one
-- Who a reply email goes to, in their language.
SELECT c.email, c.first_name, c.language, c.erased_at, coalesce(m.comment_emails, false)::boolean AS comment_emails
FROM contacts c LEFT JOIN website_members m ON m.contact_id = c.id
WHERE c.id = @contact_id;

-- name: WebsiteCommentNoticesSent :exec
UPDATE website_comment_notices SET sent_at = @at WHERE contact_id = @contact_id AND comment_id = ANY (@comments::uuid[]);

-- name: WebsiteCommentNoticesDrop :exec
DELETE FROM website_comment_notices WHERE contact_id = @contact_id AND comment_id = ANY (@comments::uuid[]) AND sent_at IS NULL;

-- Keeping

-- name: WebsiteCommentNoticesPurge :exec
-- Sent notices once the hourly cap no longer needs them, and notices that
-- couldn't go for a week (no System stream).
DELETE FROM website_comment_notices WHERE sent_at < @before OR (sent_at IS NULL AND queued_at < @before);

-- name: WebsiteCommentIPsClear :exec
-- Addresses of comments older than 30 days.
UPDATE website_comments SET ip = NULL WHERE ip IS NOT NULL AND created_at < @before;

-- name: WebsiteCommentsDeletedClear :exec
-- The text of deleted comments once their Undo has passed.
UPDATE website_comments SET body = '' WHERE status = 'deleted' AND body <> '' AND (undo_until IS NULL OR undo_until < @now);
