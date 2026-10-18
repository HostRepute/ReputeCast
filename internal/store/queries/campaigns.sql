-- name: ListCampaigns :many
-- One page of the campaigns list. Status groups: drafts = draft, approval,
-- approved; sending = sending, paused; sent = sent, cancelled, failed.
SELECT c.id, c.name, c.status, c.audience, c.audience_count, c.ab_test, c.scheduled_at, c.schedule,
       c.created_at, c.updated_at, c.started_at, c.completed_at, c.origin, c.note, c.channel,
       coalesce(u.name, '')::text AS updated_by_name,
       coalesce(t.sent, 0)::bigint AS sent,
       coalesce(t.delivered, 0)::bigint AS delivered,
       coalesce(t.opened_human, 0)::bigint AS opened_human,
       coalesce(t.clicked, 0)::bigint AS clicked,
       coalesce(r.verdict, '')::text AS readiness_verdict,
       coalesce(r.issues, 0)::bigint AS readiness_issues,
       -- An issue of a recurring digest (internal/digests): its Digest badge and status.
       c.digest_id, coalesce(dg.name, '')::text AS digest_name,
       coalesce(dgi.status, '')::text AS digest_issue_status, coalesce(dgi.item_count, 0)::integer AS digest_posts,
       dgi.prepared_at AS digest_prepared_at, dgi.send_at AS digest_send_at,
       coalesce((SELECT ar.id::text FROM approval_requests ar WHERE ar.campaign_id = c.id AND ar.status IN ('pending', 'approved') ORDER BY ar.requested_at DESC LIMIT 1), '')::text AS digest_approval_id,
       coalesce((SELECT ar.status FROM approval_requests ar WHERE ar.campaign_id = c.id AND ar.status IN ('pending', 'approved') ORDER BY ar.requested_at DESC LIMIT 1), '')::text AS digest_approval_status
FROM campaigns c
LEFT JOIN users u ON u.id = c.updated_by
LEFT JOIN digests dg ON dg.id = c.digest_id AND c.digest_role = 'issue'
LEFT JOIN LATERAL (
    SELECT di.status, di.item_count, di.prepared_at, di.send_at FROM digest_issues di WHERE di.campaign_id = c.id ORDER BY di.created_at DESC LIMIT 1
) dgi ON c.digest_role = 'issue'
LEFT JOIN LATERAL (
    SELECT sum(cc.sent) AS sent, sum(cc.delivered) AS delivered, sum(cc.opened_human) AS opened_human, sum(cc.clicked) AS clicked
    FROM campaign_counters cc WHERE cc.campaign_id = c.id
) t ON true
LEFT JOIN LATERAL (
    -- A blocked campaign counts its blocking issues ("3 blocking issues"),
    -- any other its warnings ("Readiness · 2").
    SELECT rr.verdict,
           (SELECT count(*) FROM jsonb_array_elements(coalesce(rr.checks -> 'checks', '[]'::jsonb)) e
            WHERE e ->> 'status' = 'block' OR (rr.verdict <> 'blocked' AND e ->> 'status' = 'warn')) AS issues
    FROM readiness_runs rr WHERE rr.campaign_id = c.id ORDER BY rr.ran_at DESC LIMIT 1
) r ON c.status IN ('draft', 'approval', 'approved', 'scheduled')
WHERE c.workspace_id = @workspace_id AND c.deleted_at IS NULL
  AND c.digest_role IN ('', 'issue')
  AND (@status_group::text = 'all'
       OR (@status_group = 'drafts' AND c.status IN ('draft', 'approval', 'approved'))
       OR (@status_group = 'scheduled' AND c.status = 'scheduled')
       OR (@status_group = 'sending' AND c.status IN ('sending', 'paused'))
       OR (@status_group = 'sent' AND c.status IN ('sent', 'cancelled', 'failed')))
  AND (@query::text = '' OR c.name ILIKE '%' || @query::text || '%')
  -- The channel filter: All, Email, SMS, Push, WhatsApp (design expansion phase 8).
  AND (@channel::text = '' OR c.channel = @channel::text)
ORDER BY CASE c.status WHEN 'sending' THEN 0 WHEN 'paused' THEN 0 WHEN 'scheduled' THEN 1 WHEN 'approval' THEN 2 WHEN 'approved' THEN 2 WHEN 'draft' THEN 3 ELSE 4 END,
         CASE WHEN c.status = 'scheduled' THEN c.scheduled_at END ASC,
         coalesce(c.started_at, c.updated_at) DESC,
         c.id DESC
LIMIT @lim OFFSET @off;

-- name: CountCampaigns :one
SELECT count(*) FROM campaigns c
WHERE c.workspace_id = @workspace_id AND c.deleted_at IS NULL
  AND c.digest_role IN ('', 'issue')
  AND (@status_group::text = 'all'
       OR (@status_group = 'drafts' AND c.status IN ('draft', 'approval', 'approved'))
       OR (@status_group = 'scheduled' AND c.status = 'scheduled')
       OR (@status_group = 'sending' AND c.status IN ('sending', 'paused'))
       OR (@status_group = 'sent' AND c.status IN ('sent', 'cancelled', 'failed')))
  AND (@query::text = '' OR c.name ILIKE '%' || @query::text || '%')
  -- The channel filter: All, Email, SMS, Push, WhatsApp (design expansion phase 8).
  AND (@channel::text = '' OR c.channel = @channel::text);

-- name: CountCampaignsByStatus :many
SELECT status, count(*)::bigint AS n FROM campaigns
WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND digest_role IN ('', 'issue')
GROUP BY status;

-- name: ListCampaignRouteShares :many
-- Per-provider share of each campaign: planned recipients and what was sent.
SELECT cr.campaign_id, p.id AS provider_id, p.name AS provider_name, p.adapter, cr.planned,
       coalesce((SELECT sum(cc.sent) FROM campaign_counters cc WHERE cc.campaign_id = cr.campaign_id AND cc.provider_id = cr.provider_id), 0)::bigint AS sent
FROM campaign_routes cr
JOIN providers p ON p.id = cr.provider_id
WHERE cr.campaign_id = ANY(@campaign_ids::uuid[])
ORDER BY cr.campaign_id, cr.planned DESC;

-- name: ListCampaignIncidentSummaries :many
SELECT campaign_id, kind, mailbox_provider, metric, count(*)::bigint AS n
FROM campaign_incidents
WHERE campaign_id = ANY(@campaign_ids::uuid[])
GROUP BY campaign_id, kind, mailbox_provider, metric;

-- name: CampaignTotalsSince :one
SELECT count(DISTINCT c.id)::bigint AS campaigns,
       coalesce(sum(cc.sent), 0)::bigint AS sent,
       coalesce(sum(cc.delivered), 0)::bigint AS delivered,
       coalesce(sum(cc.hard_bounced + cc.soft_bounced), 0)::bigint AS bounced,
       coalesce(sum(cc.complained), 0)::bigint AS complained,
       coalesce(sum(cc.opened_human), 0)::bigint AS opened_human,
       coalesce(sum(cc.clicked), 0)::bigint AS clicked
FROM campaigns c
JOIN campaign_counters cc ON cc.campaign_id = c.id
WHERE c.workspace_id = @workspace_id AND c.deleted_at IS NULL AND c.origin = '' AND c.started_at >= @since;

-- name: CampaignCostSince :many
SELECT p.name AS provider_name, p.adapter, coalesce(sum(cr.cost_micros), 0)::bigint AS cost_micros
FROM campaign_routes cr
JOIN campaigns c ON c.id = cr.campaign_id
JOIN providers p ON p.id = cr.provider_id
WHERE c.workspace_id = @workspace_id AND c.deleted_at IS NULL AND c.origin = '' AND c.started_at >= @since
GROUP BY p.name, p.adapter
ORDER BY cost_micros DESC, p.name;

-- name: ListWorkspaceLists :many
SELECT id, name, public FROM lists WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY created_at;

-- name: BroadcastDefaultSplit :one
SELECT splits FROM routing_rules
WHERE workspace_id = @workspace_id AND stream = 'broadcast' AND enabled AND conditions = '{}'::jsonb
ORDER BY priority LIMIT 1;

-- name: ProviderNames :many
SELECT id, name, adapter FROM providers WHERE workspace_id = @workspace_id AND disconnected_at IS NULL;

-- name: GetComposerCampaign :one
-- One campaign with what the composer header shows.
SELECT c.id, c.workspace_id, c.name, c.status, c.subject, c.preheader, c.sender_id, c.reply_to, c.audience, c.audience_count,
       c.editor_mode, c.mjml, c.html, c.plain_text, c.version, c.ab_test, c.schedule, c.content_hash, c.content_settings,
       c.created_by, c.updated_by, c.created_at, c.updated_at, c.autosaved_at, c.scheduled_at, c.started_at, c.digest_id, c.digest_role,
       coalesce(u.name, '')::text AS updated_by_name,
       coalesce(s.name, '')::text AS sender_name,
       coalesce(s.email, '')::text AS sender_email,
       (SELECT count(*) FROM campaign_versions v WHERE v.campaign_id = c.id)::bigint AS version_count
FROM campaigns c
LEFT JOIN users u ON u.id = c.updated_by
LEFT JOIN senders s ON s.id = c.sender_id
WHERE c.workspace_id = @workspace_id AND c.id = @id AND c.deleted_at IS NULL;

-- name: ComposerLockCampaign :one
-- Row lock for autosave: the version check and the new version are atomic.
SELECT c.id, c.status, c.version, c.subject, c.preheader, c.mjml, c.html, c.ab_test, c.content_hash, c.content_settings,
       c.updated_by, c.autosaved_at, c.updated_at,
       coalesce(u.name, '')::text AS updated_by_name
FROM campaigns c
LEFT JOIN users u ON u.id = c.updated_by
WHERE c.workspace_id = @workspace_id AND c.id = @id AND c.deleted_at IS NULL
FOR UPDATE OF c;

-- name: InsertComposerCampaign :exec
INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, sender_id, audience, audience_count,
                       editor_mode, mjml, html, plain_text, version, ab_test, content_hash, created_by, updated_by, created_at, updated_at, autosaved_at)
VALUES (@id, @workspace_id, @name, 'draft', @subject, @preheader, sqlc.narg(sender_id), @audience, sqlc.narg(audience_count),
        'visual', @mjml, @html, @plain_text, 1, sqlc.narg(ab_test), @content_hash, @created_by, @created_by, @now, @now, sqlc.narg(autosaved_at));

-- name: ComposerUpdateCampaignMeta :exec
UPDATE campaigns
SET name = @name, sender_id = sqlc.narg(sender_id), reply_to = @reply_to, audience = @audience,
    audience_count = sqlc.narg(audience_count), content_settings = @content_settings,
    updated_by = @updated_by, updated_at = @now
WHERE workspace_id = @workspace_id AND id = @id;

-- name: SaveComposerContent :exec
UPDATE campaigns
SET subject = @subject, preheader = @preheader, mjml = @mjml, html = @html, plain_text = @plain_text,
    ab_test = sqlc.narg(ab_test), content_hash = @content_hash, version = @version,
    updated_by = @updated_by, updated_at = @now, autosaved_at = @now
WHERE workspace_id = @workspace_id AND id = @id;

-- name: InsertComposerVersion :exec
INSERT INTO campaign_versions (campaign_id, version, subject, preheader, mjml, html, plain_text, ab_test, saved_by, saved_at, label, source, size_bytes)
VALUES (@campaign_id, @version, @subject, @preheader, @mjml, @html, @plain_text, sqlc.narg(ab_test), sqlc.narg(saved_by), @saved_at, @label, @source, @size_bytes);

-- name: ListComposerVersions :many
SELECT v.version, v.saved_at, v.label, v.source, v.size_bytes, v.saved_by, coalesce(u.name, '')::text AS saved_by_name
FROM campaign_versions v
LEFT JOIN users u ON u.id = v.saved_by
WHERE v.campaign_id = @campaign_id
ORDER BY v.version DESC
LIMIT @lim;

-- name: GetComposerVersion :one
SELECT v.campaign_id, v.version, v.subject, v.preheader, v.mjml, v.html, v.plain_text, v.ab_test, v.saved_by, v.saved_at, v.label, v.source, v.size_bytes,
       coalesce(u.name, '')::text AS saved_by_name
FROM campaign_versions v
LEFT JOIN users u ON u.id = v.saved_by
WHERE v.campaign_id = @campaign_id AND v.version = @version;

-- name: PruneComposerVersions :execrows
-- Keeps the newest @keep versions; the first version and anything that is
-- not an autosave (restores, imports, ⌘S) are always kept.
DELETE FROM campaign_versions v
WHERE v.campaign_id = @campaign_id AND v.source = 'autosave' AND v.version <> 1
  AND v.version <= (SELECT max(x.version) FROM campaign_versions x WHERE x.campaign_id = @campaign_id) - @keep::int;

-- name: ListComposerSenders :many
SELECT s.id, s.name, s.email, d.domain, d.status AS domain_status
FROM senders s
JOIN sending_domains d ON d.id = s.domain_id
WHERE s.workspace_id = @workspace_id
ORDER BY s.created_at, s.email;

-- name: GetComposerSender :one
SELECT s.id, s.name, s.email, d.domain
FROM senders s
JOIN sending_domains d ON d.id = s.domain_id
WHERE s.workspace_id = @workspace_id AND s.id = @id;

-- name: ComposerDefaultSender :one
SELECT coalesce(w.default_sender_id::text, '')::text AS sender_id, w.branding
FROM workspaces w WHERE w.id = @id;

-- name: ComposerAudienceLists :many
-- Lists with their subscribed, active contacts (suppressed addresses excluded).
SELECT l.id, l.name,
       (SELECT count(*) FROM list_memberships m
        JOIN contacts c ON c.id = m.contact_id
        WHERE m.list_id = l.id AND m.status = 'subscribed' AND c.status = 'active' AND c.erased_at IS NULL)::bigint AS subscribers
FROM lists l
WHERE l.workspace_id = @workspace_id AND l.deleted_at IS NULL
ORDER BY l.created_at;

-- name: ComposerAudienceSegments :many
SELECT id, name, cached_count FROM segments WHERE workspace_id = @workspace_id ORDER BY name;

-- name: ComposerCountListAudience :one
-- Recipients of lists minus excluded lists and suppressions. A deleted list
-- (restorable for 30 days) adds nobody; as an exclusion it still excludes.
SELECT count(DISTINCT c.id)::bigint
FROM contacts c
JOIN list_memberships m ON m.contact_id = c.id
WHERE c.workspace_id = @workspace_id AND c.status = 'active' AND c.erased_at IS NULL
  AND m.status = 'subscribed' AND m.list_id = ANY(ARRAY(SELECT ll.id FROM lists ll WHERE ll.id = ANY(@lists::uuid[]) AND ll.deleted_at IS NULL))
  AND NOT EXISTS (SELECT 1 FROM list_memberships x WHERE x.contact_id = c.id AND x.status = 'subscribed' AND x.list_id = ANY(@exclude_lists::uuid[]))
  AND NOT EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = c.workspace_id AND s.email = c.email);

-- name: ComposerCountMissingField :one
-- Audience contacts whose first name is empty (they see the fallback).
SELECT count(DISTINCT c.id)::bigint
FROM contacts c
JOIN list_memberships m ON m.contact_id = c.id
WHERE c.workspace_id = @workspace_id AND c.status = 'active' AND c.erased_at IS NULL
  AND m.status = 'subscribed' AND m.list_id = ANY(ARRAY(SELECT ll.id FROM lists ll WHERE ll.id = ANY(@lists::uuid[]) AND ll.deleted_at IS NULL)) AND c.first_name = '';

-- name: ComposerPreviewContacts :many
-- Contacts to preview as: the campaign's audience first, people with a first name first.
SELECT c.id, c.email, c.first_name, c.last_name, c.attributes, c.tier, c.language
FROM contacts c
WHERE c.workspace_id = @workspace_id AND c.erased_at IS NULL AND c.status = 'active'
  AND (@query::text = '' OR c.email ILIKE '%' || @query::text || '%' OR (c.first_name || ' ' || c.last_name) ILIKE '%' || @query::text || '%')
  AND (cardinality(@lists::uuid[]) = 0 OR EXISTS (SELECT 1 FROM list_memberships m WHERE m.contact_id = c.id AND m.status = 'subscribed' AND m.list_id = ANY(ARRAY(SELECT ll.id FROM lists ll WHERE ll.id = ANY(@lists::uuid[]) AND ll.deleted_at IS NULL))))
ORDER BY (c.first_name = '') ASC, c.last_open_at DESC NULLS LAST, c.email
LIMIT @lim;

-- name: ComposerPreviewContactByID :one
SELECT c.id, c.email, c.first_name, c.last_name, c.attributes, c.tier, c.language
FROM contacts c
WHERE c.workspace_id = @workspace_id AND c.id = @id AND c.erased_at IS NULL;

-- name: ComposerRandomContact :one
SELECT c.id, c.email, c.first_name, c.last_name, c.attributes, c.tier, c.language
FROM contacts c
WHERE c.workspace_id = @workspace_id AND c.erased_at IS NULL AND c.status = 'active' AND c.tier = 'active'
ORDER BY random()
LIMIT 1;

-- name: ComposerCustomFields :many
SELECT key, label, type FROM custom_fields WHERE workspace_id = @workspace_id ORDER BY created_at, key;

-- name: ComposerTestRecipients :many
-- Active members of the workspace, for "Send test" (name and email).
SELECT u.id, u.name, u.email
FROM workspace_members m
JOIN users u ON u.id = m.user_id AND u.status = 'active'
WHERE m.workspace_id = @workspace_id AND m.status = 'active'
ORDER BY u.name;

-- name: ComposerLastSentCampaign :one
SELECT c.id, c.name, c.mjml, coalesce(c.started_at, c.updated_at)::timestamptz AS sent_at
FROM campaigns c
WHERE c.workspace_id = @workspace_id AND c.deleted_at IS NULL AND c.status = 'sent' AND c.origin = '' AND c.mjml <> ''
ORDER BY c.started_at DESC NULLS LAST
LIMIT 1;


-- name: ComposerLatestReadiness :one
SELECT rr.verdict,
       (SELECT count(*) FROM jsonb_array_elements(coalesce(rr.checks -> 'checks', '[]'::jsonb)) e WHERE e ->> 'status' IN ('warn', 'block'))::bigint AS issues
FROM readiness_runs rr WHERE rr.campaign_id = @campaign_id ORDER BY rr.ran_at DESC LIMIT 1;

-- name: ComposerVersionSources :many
-- The MJML of recent versions, newest first (to find when a section was removed).
SELECT version, saved_at, mjml, coalesce(u.name, '')::text AS saved_by_name
FROM campaign_versions v
LEFT JOIN users u ON u.id = v.saved_by
WHERE v.campaign_id = @campaign_id
ORDER BY v.version DESC
LIMIT @lim;
