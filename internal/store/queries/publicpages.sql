-- Public pages (spec 7.12): the archive, view online and what an
-- unsubscribe records about the email it came from.

-- name: ArchiveMissingIssues :many
-- Sent campaigns of the archived lists without an archive row yet, oldest
-- first so slugs are handed out in publishing order.
SELECT c.id, c.name, c.subject, c.audience,
       coalesce(c.started_at, c.completed_at, c.created_at)::timestamptz AS sent_at
FROM campaigns c
LEFT JOIN archive_issues ai ON ai.campaign_id = c.id
WHERE c.workspace_id = @workspace_id AND c.status = 'sent' AND c.deleted_at IS NULL AND ai.campaign_id IS NULL
  AND EXISTS (SELECT 1 FROM jsonb_array_elements_text(CASE WHEN jsonb_typeof(c.audience -> 'lists') = 'array' THEN c.audience -> 'lists' ELSE '[]'::jsonb END) AS l(id)
              WHERE l.id = ANY (@lists::text[]))
ORDER BY sent_at, c.id;

-- name: ArchiveSlugTaken :one
SELECT EXISTS (SELECT 1 FROM archive_issues WHERE workspace_id = @workspace_id AND slug = @slug)::boolean;

-- name: ArchiveLock :exec
-- Serializes archive syncs of a workspace, so two requests never hand out
-- the same slug.
SELECT pg_advisory_xact_lock(hashtextextended('archive:' || CAST(@workspace_id::uuid AS text), 0));

-- name: ArchiveInsertIssue :exec
INSERT INTO archive_issues (campaign_id, workspace_id, slug, list_id, label, created_at, updated_at)
VALUES (@campaign_id, @workspace_id, @slug, @list_id, @label, @at, @at)
ON CONFLICT (campaign_id) DO NOTHING;

-- name: ArchiveIssues :many
SELECT ai.campaign_id, ai.slug, ai.list_id, coalesce(l.name, '')::text AS list_name, ai.title, ai.excerpt, ai.label, ai.hidden,
       c.name, c.subject, c.preheader, c.plain_text,
       coalesce(c.started_at, c.completed_at, c.created_at)::timestamptz AS sent_at,
       coalesce(c.completed_at, c.started_at, c.created_at)::timestamptz AS updated_at
FROM archive_issues ai
JOIN campaigns c ON c.id = ai.campaign_id AND c.deleted_at IS NULL AND c.status = 'sent'
LEFT JOIN lists l ON l.id = ai.list_id
WHERE ai.workspace_id = @workspace_id
  AND ai.list_id = ANY (@lists::uuid[])
  AND (@include_hidden::boolean OR NOT ai.hidden)
  AND (sqlc.narg(list_id)::uuid IS NULL OR ai.list_id = sqlc.narg(list_id)::uuid)
  AND (@q::text = '' OR ai.title ILIKE @q OR c.subject ILIKE @q OR ai.excerpt ILIKE @q OR c.preheader ILIKE @q
       OR EXISTS (SELECT 1 FROM content_translations ct
                  WHERE ct.owner_kind = 'campaign' AND ct.owner_id = c.id AND ct.owner_version = 0 AND ct.status = 'translated' AND NOT ct.held
                    AND (ct.subject ILIKE @q OR ct.preheader ILIKE @q)))
ORDER BY sent_at DESC, c.id DESC
LIMIT @lim OFFSET @off;

-- name: ArchiveIssueOne :one
-- One issue of the Archive tab, as ArchiveIssues lists it.
SELECT ai.campaign_id, ai.slug, ai.list_id, coalesce(l.name, '')::text AS list_name, ai.title, ai.excerpt, ai.label, ai.hidden,
       c.name, c.subject, c.preheader, c.plain_text,
       coalesce(c.started_at, c.completed_at, c.created_at)::timestamptz AS sent_at,
       coalesce(c.completed_at, c.started_at, c.created_at)::timestamptz AS updated_at
FROM archive_issues ai
JOIN campaigns c ON c.id = ai.campaign_id AND c.deleted_at IS NULL AND c.status = 'sent'
LEFT JOIN lists l ON l.id = ai.list_id
WHERE ai.workspace_id = @workspace_id AND ai.campaign_id = @campaign_id;

-- name: ArchiveIssueCount :one
SELECT count(*)::bigint
FROM archive_issues ai
JOIN campaigns c ON c.id = ai.campaign_id AND c.deleted_at IS NULL AND c.status = 'sent'
WHERE ai.workspace_id = @workspace_id
  AND ai.list_id = ANY (@lists::uuid[])
  AND (@include_hidden::boolean OR NOT ai.hidden)
  AND (sqlc.narg(list_id)::uuid IS NULL OR ai.list_id = sqlc.narg(list_id)::uuid)
  AND (@q::text = '' OR ai.title ILIKE @q OR c.subject ILIKE @q OR ai.excerpt ILIKE @q OR c.preheader ILIKE @q
       OR EXISTS (SELECT 1 FROM content_translations ct
                  WHERE ct.owner_kind = 'campaign' AND ct.owner_id = c.id AND ct.owner_version = 0 AND ct.status = 'translated' AND NOT ct.held
                    AND (ct.subject ILIKE @q OR ct.preheader ILIKE @q)));

-- name: ArchiveSummary :one
-- Every published issue: how many and since when.
SELECT count(*)::bigint AS issues,
       coalesce(min(coalesce(c.started_at, c.completed_at, c.created_at)), now())::timestamptz AS first_at,
       coalesce(max(coalesce(c.completed_at, c.started_at, c.created_at)), now())::timestamptz AS last_at
FROM archive_issues ai
JOIN campaigns c ON c.id = ai.campaign_id AND c.deleted_at IS NULL AND c.status = 'sent'
WHERE ai.workspace_id = @workspace_id AND ai.list_id = ANY (@lists::uuid[]) AND NOT ai.hidden;

-- name: ArchiveIssueBySlug :one
SELECT ai.campaign_id, ai.slug, ai.list_id, coalesce(l.name, '')::text AS list_name, ai.title, ai.excerpt, ai.label, ai.hidden,
       c.name, c.subject, c.preheader, c.html, c.plain_text,
       coalesce(c.started_at, c.completed_at, c.created_at)::timestamptz AS sent_at,
       coalesce(c.completed_at, c.started_at, c.created_at)::timestamptz AS updated_at
FROM archive_issues ai
JOIN campaigns c ON c.id = ai.campaign_id AND c.deleted_at IS NULL AND c.status = 'sent'
LEFT JOIN lists l ON l.id = ai.list_id
WHERE ai.workspace_id = @workspace_id AND ai.slug = @slug AND ai.list_id = ANY (@lists::uuid[]) AND NOT ai.hidden;

-- name: ArchiveIssueGet :one
SELECT * FROM archive_issues WHERE workspace_id = @workspace_id AND campaign_id = @campaign_id;

-- name: ArchiveUpdateIssue :one
UPDATE archive_issues
SET hidden = @hidden, title = @title, excerpt = @excerpt, label = @label, list_id = @list_id, updated_by = @updated_by, updated_at = @at
WHERE workspace_id = @workspace_id AND campaign_id = @campaign_id
RETURNING *;

-- name: ArchiveListCounts :many
SELECT ai.list_id::uuid AS list_id, count(*)::bigint AS issues
FROM archive_issues ai
JOIN campaigns c ON c.id = ai.campaign_id AND c.deleted_at IS NULL AND c.status = 'sent'
WHERE ai.workspace_id = @workspace_id AND ai.list_id = ANY (@lists::uuid[]) AND NOT ai.hidden
GROUP BY ai.list_id;

-- name: PublicMessage :one
SELECT m.id, m.workspace_id, m.stream, m.campaign_id, m.contact_id, m.provider_id, m.mailbox_provider, m.variant, m.email
FROM messages m
WHERE m.id = @id;

-- name: PublicCampaign :one
SELECT c.id, c.workspace_id, c.name, c.subject, c.preheader, c.html, c.plain_text, c.ab_test, c.audience,
       coalesce(c.started_at, c.completed_at, c.created_at)::timestamptz AS sent_at
FROM campaigns c
WHERE c.workspace_id = @workspace_id AND c.id = @id AND c.deleted_at IS NULL;

-- name: PublicScheduledCampaigns :many
-- Upcoming sends, for "Next issue: Tuesday 14 October" on the confirmation page.
SELECT c.id, c.audience, c.scheduled_at::timestamptz AS scheduled_at
FROM campaigns c
WHERE c.workspace_id = @workspace_id AND c.status IN ('scheduled', 'approved') AND c.deleted_at IS NULL
  AND c.scheduled_at IS NOT NULL AND c.scheduled_at > @after::timestamptz
ORDER BY c.scheduled_at
LIMIT 50;

-- name: PublicLatestIssueSlug :one
SELECT ai.slug
FROM archive_issues ai
JOIN campaigns c ON c.id = ai.campaign_id AND c.deleted_at IS NULL AND c.status = 'sent'
WHERE ai.workspace_id = @workspace_id AND ai.list_id = ANY (@lists::uuid[]) AND NOT ai.hidden
ORDER BY coalesce(c.started_at, c.completed_at, c.created_at) DESC
LIMIT 1;

-- name: PublicBumpUnsubscribed :exec
INSERT INTO campaign_counters (campaign_id, provider_id, mailbox_provider, variant, unsubscribed)
VALUES (@campaign_id, @provider_id, @mailbox_provider, @variant, 1)
ON CONFLICT (campaign_id, provider_id, mailbox_provider, variant) DO UPDATE
SET unsubscribed = campaign_counters.unsubscribed + 1;

-- name: PublicInsertUnsubscribeEvent :exec
INSERT INTO delivery_events (id, workspace_id, message_id, campaign_id, contact_id, provider_id, type, occurred_at, received_at,
                             mailbox_provider, original_type, user_agent, ip, recipient, source)
VALUES (@id, @workspace_id, @message_id, @campaign_id, @contact_id, @provider_id, 'unsubscribe', @at, @at,
        @mailbox_provider, @original_type, @user_agent, @ip, @recipient, @source);

-- name: PublicInsertFeedback :exec
INSERT INTO unsubscribe_feedback (id, workspace_id, contact_id, campaign_id, reason, comment, chose_fewer, created_at)
VALUES (@id, @workspace_id, @contact_id, @campaign_id, @reason, @comment, @chose_fewer, @at);

-- name: PublicUnsubscribeStats :one
-- The unsubscribe page tab: departures and answers in the last 30 days.
SELECT
    (SELECT count(DISTINCT ce.contact_id) FROM consent_events ce JOIN contacts c ON c.id = ce.contact_id
     WHERE c.workspace_id = @workspace_id AND ce.action = 'unsubscribed' AND ce.occurred_at >= @since
       AND ce.source IN ('unsubscribe_page', 'one_click', 'preference_center'))::bigint AS unsubscribed,
    (SELECT count(DISTINCT ce.contact_id) FROM consent_events ce JOIN contacts c ON c.id = ce.contact_id
     WHERE c.workspace_id = @workspace_id AND ce.action = 'unsubscribed' AND ce.occurred_at >= @since
       AND ce.source = 'one_click')::bigint AS one_click,
    (SELECT count(*) FROM unsubscribe_feedback f WHERE f.workspace_id = @workspace_id AND f.created_at >= @since AND f.chose_fewer)::bigint AS chose_fewer,
    (SELECT count(*) FROM unsubscribe_feedback f WHERE f.workspace_id = @workspace_id AND f.created_at >= @since AND NOT f.chose_fewer
       AND f.reason <> 'one_click')::bigint AS answered;
