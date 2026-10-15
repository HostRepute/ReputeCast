-- Transactional email: settings, categories, templates, versions, the
-- message log, opt-outs, idempotency, alerts and the first-send guide.

-- name: TxnSettingsEnsure :exec
INSERT INTO transactional_settings (workspace_id) VALUES (@workspace_id) ON CONFLICT DO NOTHING;

-- name: TxnSettings :one
SELECT s.*, coalesce(snd.email, '')::text AS default_sender_email, coalesce(snd.name, '')::text AS default_sender_name,
       coalesce(u.name, '')::text AS updated_by_name
FROM transactional_settings s
LEFT JOIN senders snd ON snd.id = s.default_sender_id
LEFT JOIN users u ON u.id = s.updated_by
WHERE s.workspace_id = @workspace_id;

-- name: TxnSettingsForUpdate :one
SELECT * FROM transactional_settings WHERE workspace_id = @workspace_id FOR UPDATE;

-- name: TxnSettingsUpdate :exec
UPDATE transactional_settings
SET link_contacts = @link_contacts, content_retention_days = @content_retention_days,
    alert_bounce_rate = @alert_bounce_rate, alert_failed = @alert_failed, alert_median_seconds = @alert_median_seconds,
    default_sender_id = sqlc.narg(default_sender_id), default_reply_to = @default_reply_to,
    default_track_opens = @default_track_opens, default_track_clicks = @default_track_clicks,
    updated_by = sqlc.narg(updated_by), updated_at = @updated_at
WHERE workspace_id = @workspace_id;

-- name: TxnSettingsSetGuide :exec
UPDATE transactional_settings SET guide = @guide WHERE workspace_id = @workspace_id;

-- name: TxnCategories :many
-- Categories with their template counts and opt-outs.
SELECT c.*,
       (SELECT count(*) FROM transactional_templates t WHERE t.category_id = c.id AND t.deleted_at IS NULL)::bigint AS templates,
       (SELECT count(*) FROM transactional_opt_outs o WHERE o.category_id = c.id)::bigint AS opted_out
FROM transactional_categories c
WHERE c.workspace_id = @workspace_id
ORDER BY c.position, c.created_at;

-- name: TxnCategory :one
SELECT * FROM transactional_categories WHERE workspace_id = @workspace_id AND id = @id;

-- name: TxnCategoryInsert :exec
INSERT INTO transactional_categories (id, workspace_id, name, kind, description, position, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @name, @kind, @description,
        coalesce((SELECT max(position) + 1 FROM transactional_categories WHERE workspace_id = @workspace_id), 0),
        sqlc.narg(created_by), @created_at, @created_at);

-- name: TxnCategoryUpdate :exec
UPDATE transactional_categories SET name = @name, kind = @kind, description = @description, updated_at = @updated_at
WHERE workspace_id = @workspace_id AND id = @id;

-- name: TxnCategoryDelete :execrows
DELETE FROM transactional_categories WHERE workspace_id = @workspace_id AND id = @id;

-- name: TxnCategoryTemplateCount :one
SELECT count(*)::bigint FROM transactional_templates WHERE category_id = @category_id AND deleted_at IS NULL;

-- name: TxnTemplates :many
-- The workspace's templates (archived ones too), with what the list shows.
SELECT t.id, t.key, t.name, t.description, t.category_id, t.live_version, t.live_since, t.versions_count,
       t.draft_hash, t.draft_revision, t.draft_saved_at, t.archived_at, t.created_at, t.updated_at, t.first_published_at,
       coalesce(c.name, '')::text AS category_name, coalesce(c.kind, '')::text AS category_kind,
       coalesce(v.content_hash, '')::text AS live_hash
FROM transactional_templates t
LEFT JOIN transactional_categories c ON c.id = t.category_id
LEFT JOIN transactional_template_versions v ON v.template_id = t.id AND v.version = t.live_version
WHERE t.workspace_id = @workspace_id AND t.deleted_at IS NULL
ORDER BY t.archived_at IS NOT NULL, t.created_at, t.id;

-- name: TxnTemplate :one
SELECT t.*, coalesce(c.name, '')::text AS category_name, coalesce(c.kind, '')::text AS category_kind,
       coalesce(v.content_hash, '')::text AS live_hash,
       coalesce(snd.email, '')::text AS sender_email, coalesce(snd.name, '')::text AS sender_name,
       coalesce(ds.name, '')::text AS draft_saved_by_name, coalesce(lb.name, '')::text AS live_by_name
FROM transactional_templates t
LEFT JOIN transactional_categories c ON c.id = t.category_id
LEFT JOIN transactional_template_versions v ON v.template_id = t.id AND v.version = t.live_version
LEFT JOIN senders snd ON snd.id = t.sender_id
LEFT JOIN users ds ON ds.id = t.draft_saved_by
LEFT JOIN users lb ON lb.id = t.live_by
WHERE t.workspace_id = @workspace_id AND t.id = @id AND t.deleted_at IS NULL;

-- name: TxnTemplateIDByKey :one
SELECT id FROM transactional_templates WHERE workspace_id = @workspace_id AND key = @key AND deleted_at IS NULL;

-- name: TxnTemplateForUpdate :one
SELECT * FROM transactional_templates WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL FOR UPDATE;

-- name: TxnTemplateKeyTaken :one
SELECT EXISTS (SELECT 1 FROM transactional_templates WHERE workspace_id = @workspace_id AND key = @key AND deleted_at IS NULL AND id <> @id)::boolean;

-- name: TxnTemplateInsert :exec
INSERT INTO transactional_templates (id, workspace_id, key, name, description, category_id, subject, preheader, editor_mode, mjml, html, plain_text,
    variables, test_data, sender_id, reply_to, track_opens, track_clicks, draft_revision, draft_hash, draft_saved_at, draft_saved_by,
    origin, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @key, @name, @description, sqlc.narg(category_id), @subject, @preheader, @editor_mode, @mjml, @html, @plain_text,
    @variables, @test_data, sqlc.narg(sender_id), @reply_to, @track_opens, @track_clicks, 1, @draft_hash, @now, sqlc.narg(created_by),
    @origin, sqlc.narg(created_by), @now, @now);

-- name: TxnTemplateUpdateMeta :exec
UPDATE transactional_templates SET key = @key, name = @name, description = @description, category_id = sqlc.narg(category_id), updated_at = @updated_at
WHERE workspace_id = @workspace_id AND id = @id;

-- name: TxnTemplateSaveDraft :exec
UPDATE transactional_templates
SET subject = @subject, preheader = @preheader, editor_mode = @editor_mode, mjml = @mjml, html = @html, plain_text = @plain_text,
    variables = @variables, test_data = @test_data, sender_id = sqlc.narg(sender_id), reply_to = @reply_to,
    track_opens = @track_opens, track_clicks = @track_clicks,
    draft_revision = @draft_revision, draft_hash = @draft_hash, draft_saved_at = @now, draft_saved_by = sqlc.narg(saved_by), updated_at = @now
WHERE workspace_id = @workspace_id AND id = @id;

-- name: TxnTemplateInsertDraftRevision :exec
INSERT INTO transactional_template_drafts (template_id, revision, subject, preheader, mjml, html, variables, saved_by, saved_at, label, source)
VALUES (@template_id, @revision, @subject, @preheader, @mjml, @html, @variables, sqlc.narg(saved_by), @saved_at, @label, @source)
ON CONFLICT (template_id, revision) DO NOTHING;

-- name: TxnTemplateDraftRevision :one
SELECT d.*, coalesce(u.name, '')::text AS saved_by_name
FROM transactional_template_drafts d LEFT JOIN users u ON u.id = d.saved_by
WHERE d.template_id = @template_id AND d.revision = @revision;

-- name: TxnTemplatePruneDrafts :exec
DELETE FROM transactional_template_drafts WHERE template_id = @template_id AND revision <= @below;

-- name: TxnTemplatePublish :exec
UPDATE transactional_templates
SET live_version = @version, live_since = @now::timestamptz, live_by = sqlc.narg(live_by), versions_count = greatest(versions_count, @version),
    first_published_at = coalesce(first_published_at, @now::timestamptz), variables = @variables, updated_at = @now::timestamptz
WHERE workspace_id = @workspace_id AND id = @id;

-- name: TxnTemplateSetLive :exec
UPDATE transactional_templates SET live_version = @version, live_since = @now::timestamptz, live_by = sqlc.narg(live_by), updated_at = @now::timestamptz
WHERE workspace_id = @workspace_id AND id = @id;

-- name: TxnTemplateArchive :exec
UPDATE transactional_templates SET archived_at = sqlc.narg(archived_at), archived_by = sqlc.narg(archived_by), updated_at = @now
WHERE workspace_id = @workspace_id AND id = @id;

-- name: TxnTemplateDelete :exec
UPDATE transactional_templates SET deleted_at = @now::timestamptz, category_id = NULL, updated_at = @now::timestamptz
WHERE workspace_id = @workspace_id AND id = @id;

-- name: TxnVersionInsert :exec
INSERT INTO transactional_template_versions (template_id, version, subject, preheader, editor_mode, mjml, html, plain_text, variables, test_data,
    sender_id, reply_to, track_opens, track_clicks, content_hash, size_bytes, change, note, published_by, published_at)
VALUES (@template_id, @version, @subject, @preheader, @editor_mode, @mjml, @html, @plain_text, @variables, @test_data,
    sqlc.narg(sender_id), @reply_to, @track_opens, @track_clicks, @content_hash, @size_bytes, @change, @note, sqlc.narg(published_by), @published_at);

-- name: TxnVersions :many
SELECT v.template_id, v.version, v.subject, v.content_hash, v.size_bytes, v.change, v.note, v.published_at, v.published_by,
       coalesce(u.name, '')::text AS published_by_name, coalesce(u.avatar_color, '')::text AS published_by_color
FROM transactional_template_versions v
LEFT JOIN users u ON u.id = v.published_by
WHERE v.template_id = @template_id
ORDER BY v.version DESC;

-- name: TxnVersion :one
SELECT v.*, coalesce(u.name, '')::text AS published_by_name, coalesce(snd.email, '')::text AS sender_email, coalesce(snd.name, '')::text AS sender_name
FROM transactional_template_versions v
LEFT JOIN users u ON u.id = v.published_by
LEFT JOIN senders snd ON snd.id = v.sender_id
WHERE v.template_id = @template_id AND v.version = @version;

-- name: TxnSenders :many
-- The workspace's senders with their domain's transactional flag and
-- verification state.
SELECT s.id, s.name, s.email, s.reply_to, s.verification, d.id AS domain_id, d.domain, d.status AS domain_status, d.transactional
FROM senders s
JOIN sending_domains d ON d.id = s.domain_id
WHERE s.workspace_id = @workspace_id
ORDER BY d.transactional DESC, s.created_at, s.email;

-- name: TxnSenderByEmail :one
SELECT s.id, s.name, s.email, s.reply_to, s.verification, d.domain, d.status AS domain_status, d.transactional
FROM senders s JOIN sending_domains d ON d.id = s.domain_id
WHERE s.workspace_id = @workspace_id AND s.email = @email;

-- name: TxnSenderByID :one
SELECT s.id, s.name, s.email, s.reply_to, s.verification, d.domain, d.status AS domain_status, d.transactional
FROM senders s JOIN sending_domains d ON d.id = s.domain_id
WHERE s.workspace_id = @workspace_id AND s.id = @id;

-- name: TxnRecentCallKeys :many
-- What the template's calls of a period sent: per call, the variable
-- names (the breaking-change check of the publish dialog).
SELECT m.message_id, m.accepted_at, m.data_keys, m.template_version, m.data, m.sensitive, mm.email
FROM transactional_messages m
JOIN messages mm ON mm.id = m.message_id
WHERE m.template_id = @template_id AND m.accepted_at >= @since AND m.source IN ('api', 'batch', 'smtp', 'resend')
ORDER BY m.accepted_at DESC
LIMIT 10000;

-- name: TxnRejectedCallsForTemplate :many
SELECT at, code, detail FROM transactional_rejected_calls
WHERE workspace_id = @workspace_id AND template_key = @template_key AND at >= @since
ORDER BY at DESC LIMIT 1000;

-- name: TxnTemplateStats :many
-- Each template's numbers over a period. Test messages are left out unless
-- include_test; sent counts what a provider accepted; delivered includes
-- messages complained about later; a soft bounce counts as a bounce.
SELECT t.template_id::uuid AS template_id,
       count(*) FILTER (WHERE m.status IN ('sent','delivered','deferred','bounced','complained'))::bigint AS sent,
       count(*) FILTER (WHERE m.status IN ('delivered','complained'))::bigint AS delivered,
       count(*) FILTER (WHERE m.status = 'bounced')::bigint AS bounced,
       count(*) FILTER (WHERE m.status = 'complained')::bigint AS complained,
       count(*) FILTER (WHERE m.status = 'deferred')::bigint AS deferred,
       count(*) FILTER (WHERE t.reject_reason <> '' AND NOT t.test)::bigint AS rejected,
       count(*) FILTER (WHERE m.status = 'failed')::bigint AS failed,
       count(*) FILTER (WHERE m.opened_at IS NOT NULL)::bigint AS opened,
       count(*) FILTER (WHERE m.clicked_at IS NOT NULL)::bigint AS clicked,
       coalesce((percentile_cont(0.5) WITHIN GROUP (ORDER BY extract(epoch FROM m.delivered_at - t.accepted_at) * 1000)
                 FILTER (WHERE m.delivered_at IS NOT NULL AND m.delivered_at >= t.accepted_at)), -1)::bigint AS median_ms,
       coalesce(max(t.accepted_at) FILTER (WHERE m.status IN ('sent','delivered','deferred','bounced','complained')), 'epoch'::timestamptz)::timestamptz AS last_sent_at,
       coalesce(bool_or(jsonb_array_length(t.attachments) > 0), false)::boolean AS attachments
FROM transactional_messages t
JOIN messages m ON m.id = t.message_id
WHERE t.workspace_id = @workspace_id AND t.accepted_at >= @since AND t.template_id IS NOT NULL
  AND (@include_test::boolean OR NOT t.test)
GROUP BY t.template_id;

-- name: TxnRouteCategories :many
-- Transactional categories for the routing screen's conditions.
SELECT id, name FROM transactional_categories WHERE workspace_id = @workspace_id ORDER BY position, created_at;

-- name: TxnRouteTemplates :many
SELECT key, name FROM transactional_templates WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND archived_at IS NULL ORDER BY created_at;

-- name: TxnRouteTemplateCategory :one
SELECT category_id FROM transactional_templates WHERE workspace_id = @workspace_id AND key = @key AND deleted_at IS NULL;

-- name: TxnRouteDomains :many
-- Sending domains marked for transactional mail.
SELECT domain FROM sending_domains WHERE workspace_id = @workspace_id AND transactional ORDER BY created_at, domain;

-- name: TxnRouteSetPriority :exec
INSERT INTO stream_policies (workspace_id, stream, failover, priority)
VALUES (@workspace_id, 'transactional', 'automatic', @priority)
ON CONFLICT (workspace_id, stream) DO UPDATE SET priority = EXCLUDED.priority;

-- name: TxnRouteSuppressedAllMail :one
-- Whether a suppression stops transactional mail: everything except the
-- marketing-only ones (unsubscribes and role addresses).
SELECT EXISTS (SELECT 1 FROM suppressions WHERE workspace_id = @workspace_id AND email = @email
               AND scope = 'all' AND reason NOT IN ('unsubscribe', 'role'))::boolean;

-- name: TxnTemplateSpark :many
-- Messages a provider accepted per template and bucket of a period.
SELECT t.template_id::uuid AS template_id,
       floor(extract(epoch FROM t.accepted_at - @since::timestamptz) / @bucket_seconds::float8)::int AS bucket,
       count(*)::bigint AS n
FROM transactional_messages t
JOIN messages m ON m.id = t.message_id
WHERE t.workspace_id = @workspace_id AND t.accepted_at >= @since::timestamptz AND t.template_id IS NOT NULL
  AND m.status IN ('sent','delivered','deferred','bounced','complained')
  AND (@include_test::boolean OR NOT t.test)
GROUP BY 1, 2;
