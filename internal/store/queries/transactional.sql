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

-- name: TxnKey :one
-- A key's transactional settings (send-only, test mode, scope, rate).
SELECT id, name, kind, preset, mode, templates, categories, test_addresses, rate_per_second, permissions, created_by
FROM api_keys WHERE id = @id AND workspace_id = @workspace_id;

-- name: TxnIdemInsert :one
INSERT INTO transactional_idempotency (workspace_id, key, request_hash, created_at)
VALUES (@workspace_id, @key, @request_hash, @created_at)
ON CONFLICT (workspace_id, key) DO NOTHING
RETURNING key;

-- name: TxnIdemGet :one
SELECT * FROM transactional_idempotency WHERE workspace_id = @workspace_id AND key = @key;

-- name: TxnIdemSave :exec
UPDATE transactional_idempotency SET status = @status, response = @response WHERE workspace_id = @workspace_id AND key = @key;

-- name: TxnIdemPurge :execrows
DELETE FROM transactional_idempotency WHERE created_at < @before;

-- name: TxnSuppression :one
-- The suppression that stops transactional mail to an address, if any:
-- everything except the marketing-only ones (unsubscribes, role addresses).
SELECT s.reason, s.source, s.original_code, s.detail, s.created_at, coalesce(p.name, '')::text AS provider_name
FROM suppressions s LEFT JOIN providers p ON p.id = s.source_provider_id
WHERE s.workspace_id = @workspace_id AND s.email = @email AND s.scope = 'all' AND s.reason NOT IN ('unsubscribe', 'role');

-- name: TxnRoleSuppressed :one
-- Whether an address carries a marketing-only role suppression (the log
-- marks it "Role address").
SELECT EXISTS (SELECT 1 FROM suppressions WHERE workspace_id = @workspace_id AND email = @email AND reason = 'role')::boolean;

-- name: TxnOptOut :one
SELECT o.opted_out_at, o.source, c.name AS category_name
FROM transactional_opt_outs o JOIN transactional_categories c ON c.id = o.category_id
WHERE o.workspace_id = @workspace_id AND o.email = @email AND o.category_id = @category_id;

-- name: TxnOptOutsFor :many
-- An address's opt-outs of Optional categories.
SELECT o.category_id, o.opted_out_at, o.source, o.reason, c.name AS category_name
FROM transactional_opt_outs o JOIN transactional_categories c ON c.id = o.category_id
WHERE o.workspace_id = @workspace_id AND o.email = @email
ORDER BY c.position, c.created_at;

-- name: TxnOptOutInsert :execrows
-- Turns a category off for an address; the first date is kept.
INSERT INTO transactional_opt_outs (workspace_id, email, category_id, opted_out_at, source, message_id)
VALUES (@workspace_id, @email, @category_id, @opted_out_at, @source, sqlc.narg(message_id))
ON CONFLICT (workspace_id, email, category_id) DO NOTHING;

-- name: TxnOptOutDelete :execrows
DELETE FROM transactional_opt_outs WHERE workspace_id = @workspace_id AND email = @email AND category_id = @category_id;

-- name: TxnOptOutReason :execrows
UPDATE transactional_opt_outs SET reason = @reason
WHERE workspace_id = @workspace_id AND email = @email AND category_id = @category_id;

-- name: TxnMessageCategory :one
-- The address and category a transactional message went out with (its
-- unsubscribe and preference links).
SELECT m.email, t.category_id, t.category_name, t.category_kind
FROM messages m JOIN transactional_messages t ON t.message_id = m.id
WHERE m.id = @id AND m.workspace_id = @workspace_id;

-- name: TxnContactByEmail :one
SELECT id, mailbox_provider, first_name, last_name FROM contacts
WHERE workspace_id = @workspace_id AND email = @email AND erased_at IS NULL;

-- name: TxnMemberEmails :many
SELECT u.email FROM workspace_members m JOIN users u ON u.id = m.user_id
WHERE m.workspace_id = @workspace_id AND m.status = 'active' AND u.status = 'active';

-- name: TxnInsertMessage :exec
INSERT INTO messages (id, workspace_id, stream, contact_id, email, provider_id, mailbox_provider, status, queued_at, next_attempt_at, kind, error, route_note)
VALUES (@id, @workspace_id, 'transactional', sqlc.narg(contact_id), @email, sqlc.narg(provider_id), @mailbox_provider, @status, @queued_at,
        sqlc.narg(next_attempt_at), @kind, @error, @route_note);

-- name: TxnInsertMessageLog :exec
INSERT INTO transactional_messages (message_id, workspace_id, source, template_id, template_key, template_name, template_version,
    category_id, category_name, category_kind, key_id, key_name, user_id, ip, idempotency_key, subject, from_email, from_name, reply_to,
    to_name, cc, bcc, tags, metadata, data, data_keys, sensitive, attachments, size_bytes, test, test_delivered, send_at,
    reject_reason, reject_detail, track_opens, track_clicks, links, route, provider_since, accepted_at, rendered_at, content_expires_at, resent_from)
VALUES (@message_id, @workspace_id, @source, sqlc.narg(template_id), @template_key, @template_name, sqlc.narg(template_version),
    sqlc.narg(category_id), @category_name, @category_kind, sqlc.narg(key_id), @key_name, sqlc.narg(user_id), sqlc.narg(ip), @idempotency_key,
    @subject, @from_email, @from_name, @reply_to, @to_name, @cc, @bcc, @tags, @metadata, @data, @data_keys, @sensitive, @attachments,
    @size_bytes, @test, @test_delivered, sqlc.narg(send_at), @reject_reason, @reject_detail, @track_opens, @track_clicks, @links, @route,
    sqlc.narg(provider_since), @accepted_at, sqlc.narg(rendered_at), sqlc.narg(content_expires_at), sqlc.narg(resent_from));

-- name: TxnInsertContent :exec
INSERT INTO transactional_message_content (message_id, html, text, headers, expires_at) VALUES (@message_id, @html, @text, @headers, @expires_at);

-- name: TxnInsertOutbox :exec
INSERT INTO transactional_outbox (message_id, sealed, created_at) VALUES (@message_id, @sealed, @created_at);

-- name: TxnInsertRejectedCall :exec
INSERT INTO transactional_rejected_calls (id, workspace_id, key_id, user_id, at, status, code, detail, template_key, recipient, idempotency_key, ip)
VALUES (@id, @workspace_id, sqlc.narg(key_id), sqlc.narg(user_id), @at, @status, @code, @detail, @template_key, @recipient, @idempotency_key, sqlc.narg(ip));

-- name: TxnMessageEvent :one
-- What a message's webhook events carry.
SELECT t.message_id, t.template_key, t.template_version, t.category_name, t.tags, t.metadata, t.test, t.reject_reason, t.source,
       m.email, m.status, m.error, coalesce(m.provider_message_id, '')::text AS provider_message_id,
       coalesce(p.name, '')::text AS provider_name, t.events_emitted
FROM transactional_messages t
JOIN messages m ON m.id = t.message_id
LEFT JOIN providers p ON p.id = m.provider_id
WHERE t.message_id = @message_id;

-- name: TxnMarkEmitted :exec
UPDATE transactional_messages SET events_emitted = array_append(events_emitted, @event::text) WHERE message_id = @message_id;

-- name: TxnOutboxProviders :many
-- Providers with transactional mail to send now.
SELECT DISTINCT m.provider_id::uuid AS provider_id FROM messages m
WHERE m.status = 'queued' AND m.stream = 'transactional' AND m.provider_id IS NOT NULL
  AND (m.next_attempt_at IS NULL OR m.next_attempt_at <= @now::timestamptz);

-- name: TxnOutboxHeadroom :one
-- How much of a provider's second transactional mail takes before
-- Broadcasts: its due transactional messages when the workspace keeps
-- "first in line on shared providers" on (the default), else 0.
SELECT CASE WHEN coalesce((SELECT sp.priority FROM stream_policies sp JOIN providers p ON p.workspace_id = sp.workspace_id
                          WHERE p.id = @provider_id AND sp.stream = 'transactional'), true)
            THEN (SELECT count(*) FROM messages m
                  WHERE m.provider_id = @provider_id AND m.status = 'queued' AND m.stream = 'transactional'
                    AND (m.next_attempt_at IS NULL OR m.next_attempt_at <= @now::timestamptz))
            ELSE 0 END::bigint;

-- name: TxnOutboxDue :one
-- How many transactional messages a provider has to send now.
SELECT count(*) FROM messages m
WHERE m.provider_id = @provider_id AND m.status = 'queued' AND m.stream = 'transactional'
  AND (m.next_attempt_at IS NULL OR m.next_attempt_at <= @now::timestamptz);

-- name: TxnOutboxClaim :many
-- Claims up to lim due transactional messages of one provider, with their
-- sealed payload and what failover needs.
WITH picked AS (
    SELECT m.id FROM messages m
    WHERE m.provider_id = @provider_id AND m.status = 'queued' AND m.stream = 'transactional'
      AND (m.claimed_at IS NULL OR m.claimed_at < @stale::timestamptz)
      AND (m.next_attempt_at IS NULL OR m.next_attempt_at <= @now::timestamptz)
    ORDER BY m.id
    LIMIT @lim
    FOR UPDATE SKIP LOCKED
), upd AS (
    UPDATE messages m SET claimed_at = @now::timestamptz, attempts = m.attempts + 1
    FROM picked WHERE m.id = picked.id
    RETURNING m.id, m.workspace_id, m.email, m.mailbox_provider, m.attempts
)
SELECT upd.id, upd.workspace_id, upd.email, upd.mailbox_provider, upd.attempts,
       coalesce(o.sealed, ''::bytea)::bytea AS sealed, t.provider_since, t.route, t.attempts AS tried, t.template_key, t.accepted_at, t.track_opens, t.track_clicks
FROM upd
LEFT JOIN transactional_outbox o ON o.message_id = upd.id
JOIN transactional_messages t ON t.message_id = upd.id
ORDER BY upd.id;

-- name: TxnOutboxQueuedOn :many
-- Queued transactional messages of a provider (failover when it is down).
SELECT m.id, m.workspace_id, t.provider_since, t.route, t.attempts AS tried
FROM messages m JOIN transactional_messages t ON t.message_id = m.id
WHERE m.provider_id = @provider_id AND m.status = 'queued' AND m.stream = 'transactional'
  AND (m.claimed_at IS NULL OR m.claimed_at < @stale::timestamptz)
ORDER BY m.id
LIMIT 500;

-- name: TxnOutboxOrphans :many
-- Queued transactional messages whose provider was removed.
SELECT m.id, m.workspace_id, t.route, t.attempts AS tried, t.provider_since
FROM messages m JOIN transactional_messages t ON t.message_id = m.id
WHERE m.status = 'queued' AND m.stream = 'transactional' AND m.provider_id IS NULL AND t.reject_reason = ''
LIMIT 500;

-- name: TxnOutboxDone :exec
-- The provider accepted these messages, or every provider failed: their
-- sealed payload (and its sensitive values) goes.
DELETE FROM transactional_outbox WHERE message_id = ANY(@ids::uuid[]);

-- name: TxnAppendAttempts :exec
UPDATE transactional_messages t SET attempts = t.attempts || x.attempt
FROM (SELECT unnest(@ids::uuid[]) AS id, unnest(@attempts::jsonb[]) AS attempt) AS x
WHERE t.message_id = x.id;

-- name: TxnMoveToProvider :exec
-- A message moves to the next provider of its route (failover).
UPDATE messages SET provider_id = @provider_id, claimed_at = NULL, next_attempt_at = NULL, route_note = 'failover', failover = true
WHERE id = @id;

-- name: TxnMarkFailedOver :exec
UPDATE transactional_messages SET provider_since = @at::timestamptz, failed_over_from = sqlc.narg(from_provider), failed_over_at = @at::timestamptz
WHERE message_id = @message_id;

-- name: TxnMarkFailed :exec
UPDATE messages SET status = 'failed', claimed_at = NULL, next_attempt_at = NULL, error = @error WHERE id = @id;

-- name: TxnSetBounced :exec
-- A soft bounce of a transactional message is its final answer: the log
-- shows it as Bounced (soft), where campaigns count it as a deferral.
UPDATE messages SET status = 'bounced', error = @error WHERE id = @id AND status IN ('sent', 'deferred', 'queued');

-- name: TxnSoftBounces :one
SELECT count(*)::bigint FROM delivery_events
WHERE workspace_id = @workspace_id AND recipient = @recipient AND type = 'soft_bounce' AND occurred_at >= @since;

-- name: TxnSuppressSoftBounces :one
INSERT INTO suppressions (id, workspace_id, email, reason, source, source_provider_id, original_code, detail, created_at, scope, context)
VALUES (@id, @workspace_id, @email, 'invalid', 'webhook', sqlc.narg(source_provider_id), @original_code, @detail, @created_at, 'all', 'Transactional')
ON CONFLICT (workspace_id, email) DO NOTHING
RETURNING id;

-- name: TxnMessageLinks :one
SELECT links FROM transactional_messages WHERE message_id = @message_id;

-- name: TxnTrackingDomainVerified :one
SELECT EXISTS (SELECT 1 FROM sending_domains WHERE workspace_id = @workspace_id AND tracking_domain = @domain AND tracking_status = 'verified')::boolean;

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

-- name: TxnLog :many
-- The message log, newest first. status is what the log shows: test,
-- rejected, queued, sent, delivered, deferred, bounced, complained or
-- failed. before_* pages back; after_* returns only newer messages (live).
WITH log AS (
    SELECT t.message_id, t.accepted_at, m.email, m.contact_id, t.to_name, t.source, t.template_key, t.template_name, t.template_version, t.subject,
           m.provider_id, coalesce(p.name, '')::text AS provider_name, m.error, m.sent_at, m.delivered_at, m.opened_at, m.clicked_at,
           t.test, t.test_delivered, t.reject_reason, t.reject_detail, t.track_opens, t.track_clicks,
           t.failed_over_from, coalesce(fp.name, '')::text AS failed_over_from_name, t.failed_over_at,
           t.key_id, t.key_name, t.idempotency_key, t.tags, t.metadata, t.category_name, t.category_kind, t.workspace_id,
           (CASE WHEN t.test THEN 'test'
                 WHEN t.reject_reason <> '' OR m.status IN ('rejected', 'cancelled') THEN 'rejected'
                 WHEN m.status = 'held' THEN 'queued'
                 ELSE m.status END)::text AS status
    FROM transactional_messages t
    JOIN messages m ON m.id = t.message_id
    LEFT JOIN providers p ON p.id = m.provider_id
    LEFT JOIN providers fp ON fp.id = t.failed_over_from
    WHERE t.workspace_id = @workspace_id AND t.accepted_at >= @since::timestamptz AND t.accepted_at < @until::timestamptz
      AND (sqlc.narg(template)::text IS NULL OR (sqlc.narg(template)::text = 'raw' AND t.source = 'smtp' AND t.template_key = '') OR t.template_key = sqlc.narg(template)::text)
      AND (sqlc.narg(provider_id)::uuid IS NULL OR m.provider_id = sqlc.narg(provider_id)::uuid)
      AND (sqlc.narg(key_id)::uuid IS NULL OR t.key_id = sqlc.narg(key_id)::uuid)
      AND (sqlc.narg(q)::text IS NULL OR m.email ILIKE '%' || sqlc.narg(q)::text || '%' OR t.message_id::text = sqlc.narg(q)::text
           OR t.idempotency_key = sqlc.narg(q)::text OR sqlc.narg(q)::text = ANY(t.tags)
           OR EXISTS (SELECT 1 FROM jsonb_each_text(t.metadata) kv WHERE kv.value = sqlc.narg(q)::text))
      AND (sqlc.narg(before_at)::timestamptz IS NULL OR (t.accepted_at, t.message_id) < (sqlc.narg(before_at)::timestamptz, sqlc.narg(before_id)::uuid))
      AND (sqlc.narg(after_at)::timestamptz IS NULL OR (t.accepted_at, t.message_id) > (sqlc.narg(after_at)::timestamptz, sqlc.narg(after_id)::uuid))
)
SELECT log.message_id, log.accepted_at, log.email, log.contact_id, log.to_name, log.source, log.template_key, log.template_name, log.template_version, log.subject,
       log.provider_id, log.provider_name, log.error, log.sent_at, log.delivered_at, log.opened_at, log.clicked_at,
       log.test, log.test_delivered, log.reject_reason, log.reject_detail, log.track_opens, log.track_clicks,
       log.failed_over_from, log.failed_over_from_name, log.failed_over_at,
       log.key_id, log.key_name, log.idempotency_key, log.tags, log.metadata, log.category_name, log.category_kind, log.status,
       (SELECT count(*) FROM delivery_events e WHERE e.message_id = log.message_id AND e.type = 'deferred')::int AS deferrals,
       coalesce((SELECT e.type || ':' || e.bounce_class || ':' || e.original_code || ' ' || e.diagnostic FROM delivery_events e
                 WHERE e.message_id = log.message_id AND e.type IN ('hard_bounce', 'soft_bounce') ORDER BY e.occurred_at DESC LIMIT 1), '')::text AS bounce,
       EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = log.workspace_id AND s.email = log.email AND s.reason = 'role')::boolean AS role_address
FROM log
WHERE sqlc.narg(status)::text IS NULL OR log.status = sqlc.narg(status)::text
ORDER BY log.accepted_at DESC, log.message_id DESC
LIMIT @lim;

-- name: TxnLogCounts :many
-- The log's numbers per status for the same filters.
SELECT (CASE WHEN t.test THEN 'test'
             WHEN t.reject_reason <> '' OR m.status IN ('rejected', 'cancelled') THEN 'rejected'
             WHEN m.status = 'held' THEN 'queued'
             ELSE m.status END)::text AS status,
       count(*)::bigint AS n
FROM transactional_messages t
JOIN messages m ON m.id = t.message_id
WHERE t.workspace_id = @workspace_id AND t.accepted_at >= @since::timestamptz AND t.accepted_at < @until::timestamptz
  AND (sqlc.narg(template)::text IS NULL OR (sqlc.narg(template)::text = 'raw' AND t.source = 'smtp' AND t.template_key = '') OR t.template_key = sqlc.narg(template)::text)
  AND (sqlc.narg(provider_id)::uuid IS NULL OR m.provider_id = sqlc.narg(provider_id)::uuid)
  AND (sqlc.narg(key_id)::uuid IS NULL OR t.key_id = sqlc.narg(key_id)::uuid)
  AND (sqlc.narg(q)::text IS NULL OR m.email ILIKE '%' || sqlc.narg(q)::text || '%' OR t.message_id::text = sqlc.narg(q)::text
       OR t.idempotency_key = sqlc.narg(q)::text OR sqlc.narg(q)::text = ANY(t.tags)
       OR EXISTS (SELECT 1 FROM jsonb_each_text(t.metadata) kv WHERE kv.value = sqlc.narg(q)::text))
GROUP BY 1;

-- name: TxnLogKeys :many
-- Keys that can send, for the log's key filter.
SELECT id, name, revoked_at FROM api_keys
WHERE workspace_id = @workspace_id AND kind = 'api_key' AND 'transactional:send' = ANY(permissions)
ORDER BY created_at;

-- name: TxnLogTemplates :many
-- Templates for the log's template filter, archived ones included.
SELECT key, name FROM transactional_templates WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY created_at;

-- name: TxnMessageDetail :one
SELECT t.message_id, t.workspace_id, t.source, t.template_id, t.template_key, t.template_name, t.template_version, t.category_id, t.category_name,
       t.category_kind, t.key_id, t.key_name, t.user_id, t.ip, t.idempotency_key, t.subject, t.from_email, t.from_name, t.reply_to, t.to_name,
       t.cc, t.bcc, t.tags, t.metadata, t.data, t.data_keys, t.sensitive, t.attachments, t.size_bytes, t.test, t.test_delivered, t.send_at,
       t.reject_reason, t.reject_detail, t.track_opens, t.track_clicks, t.route, t.attempts, t.failed_over_from, t.failed_over_at, t.provider_since,
       t.accepted_at, t.rendered_at, t.content_expires_at, t.content_removed_at, t.resent_from,
       m.email, m.contact_id, m.provider_id, coalesce(m.provider_message_id, '')::text AS provider_message_id, m.mailbox_provider,
       m.status AS raw_status, m.error, m.sent_at, m.delivered_at, m.opened_at, m.clicked_at,
       coalesce(p.name, '')::text AS provider_name,
       coalesce(fp.name, '')::text AS failed_over_from_name,
       coalesce(u.name, '')::text AS user_name,
       coalesce(nullif(trim(c.first_name || ' ' || c.last_name), ''), '')::text AS contact_name,
       (c.id IS NOT NULL)::boolean AS contact_exists,
       EXISTS (SELECT 1 FROM transactional_message_content x WHERE x.message_id = t.message_id)::boolean AS has_content,
       EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = t.workspace_id AND s.email = m.email AND s.reason = 'role')::boolean AS role_address,
       (CASE WHEN t.test THEN 'test'
             WHEN t.reject_reason <> '' OR m.status IN ('rejected', 'cancelled') THEN 'rejected'
             WHEN m.status = 'held' THEN 'queued'
             ELSE m.status END)::text AS status
FROM transactional_messages t
JOIN messages m ON m.id = t.message_id
LEFT JOIN providers p ON p.id = m.provider_id
LEFT JOIN providers fp ON fp.id = t.failed_over_from
LEFT JOIN users u ON u.id = t.user_id
LEFT JOIN contacts c ON c.id = m.contact_id AND c.erased_at IS NULL
WHERE t.message_id = @id AND t.workspace_id = @workspace_id;

-- name: TxnMessageEvents :many
-- A message's provider events, oldest first.
SELECT e.type, e.occurred_at, e.original_type, e.original_code, e.diagnostic, e.bounce_class, e.machine, e.url, e.user_agent,
       e.provider_id, coalesce(p.name, '')::text AS provider_name, e.provider_message_id
FROM delivery_events e LEFT JOIN providers p ON p.id = e.provider_id
WHERE e.message_id = @message_id
ORDER BY e.occurred_at, e.received_at;

-- name: TxnMessageWebhooks :many
-- The webhook deliveries of a message's message.* events.
SELECT d.event, d.state, d.created_at, d.delivered_at, d.status_code, d.attempt, w.url
FROM webhook_deliveries d JOIN webhooks w ON w.id = d.webhook_id
WHERE (d.payload->'data'->>'message_id') = @message_id::text AND d.event LIKE 'message.%' AND w.workspace_id = @workspace_id
ORDER BY d.created_at;

-- name: TxnSuppressionFor :one
-- An address's suppression as it is now, whatever its scope.
SELECT s.id, s.reason, s.source, s.original_code, s.detail, s.created_at, s.scope, coalesce(p.name, '')::text AS provider_name
FROM suppressions s LEFT JOIN providers p ON p.id = s.source_provider_id
WHERE s.workspace_id = @workspace_id AND s.email = @email;

-- name: TxnContent :one
SELECT html, text, headers, expires_at FROM transactional_message_content WHERE message_id = @message_id;

-- name: TxnRetentionWorkspaces :many
SELECT workspace_id, content_retention_days FROM transactional_settings;

-- name: TxnRetentionContentDue :many
-- A workspace's messages whose rendered content is past its retention
-- (expired, or sent before cutoff), up to lim.
SELECT x.message_id FROM transactional_message_content x
JOIN transactional_messages t ON t.message_id = x.message_id
WHERE t.workspace_id = @workspace_id AND (x.expires_at <= @now::timestamptz OR t.accepted_at <= @cutoff::timestamptz)
LIMIT @lim;

-- name: TxnRetentionContentRemove :exec
DELETE FROM transactional_message_content WHERE message_id = ANY(@ids::uuid[]);

-- name: TxnRetentionContentRemoved :exec
UPDATE transactional_messages SET content_removed_at = @now::timestamptz WHERE message_id = ANY(@ids::uuid[]);

-- name: TxnRetentionMessages :execrows
-- Removes up to lim transactional messages older than before (90 days):
-- their log entry and content go with them.
DELETE FROM messages m
WHERE m.id IN (SELECT t.message_id FROM transactional_messages t WHERE t.accepted_at < @before::timestamptz ORDER BY t.accepted_at LIMIT @lim)
  AND m.stream = 'transactional';

-- name: TxnRetentionRejectedCalls :execrows
DELETE FROM transactional_rejected_calls WHERE at < @before::timestamptz;

-- name: TxnContactMessages :many
-- A contact's transactional messages for its activity timeline.
SELECT t.message_id, t.accepted_at, t.template_key, t.template_name, t.source, t.subject, t.test, t.reject_reason, t.category_name,
       coalesce(p.name, '')::text AS provider_name, m.delivered_at,
       (CASE WHEN t.test THEN 'test'
             WHEN t.reject_reason <> '' OR m.status IN ('rejected', 'cancelled') THEN 'rejected'
             WHEN m.status = 'held' THEN 'queued'
             ELSE m.status END)::text AS status
FROM transactional_messages t
JOIN messages m ON m.id = t.message_id
LEFT JOIN providers p ON p.id = m.provider_id
WHERE t.workspace_id = @workspace_id AND (m.contact_id = @contact_id OR m.email = @email)
ORDER BY t.accepted_at DESC
LIMIT @lim;

-- name: TxnAlertWorkspaces :many
-- Workspaces to check: transactional mail in the window, or an open alert.
SELECT DISTINCT workspace_id FROM transactional_messages WHERE accepted_at >= @since::timestamptz
UNION
SELECT workspace_id FROM transactional_alerts WHERE ended_at IS NULL;

-- name: TxnAlertTemplateHour :many
-- Each template's last hour: messages a provider accepted, bounces and
-- hard bounces (test messages left out).
SELECT t.template_id::uuid AS template_id, coalesce(max(tt.key), max(t.template_key))::text AS template_key,
       coalesce(max(tt.name), max(t.template_name))::text AS template_name,
       count(*) FILTER (WHERE m.status IN ('sent', 'delivered', 'deferred', 'bounced', 'complained'))::bigint AS sent,
       count(*) FILTER (WHERE m.status = 'bounced')::bigint AS bounced,
       count(*) FILTER (WHERE m.status = 'bounced'
                          AND EXISTS (SELECT 1 FROM delivery_events e WHERE e.message_id = m.id AND e.type = 'hard_bounce'))::bigint AS hard
FROM transactional_messages t
JOIN messages m ON m.id = t.message_id
LEFT JOIN transactional_templates tt ON tt.id = t.template_id
WHERE t.workspace_id = @workspace_id AND t.accepted_at >= @since::timestamptz AND t.template_id IS NOT NULL AND NOT t.test
GROUP BY t.template_id;

-- name: TxnAlertBounceDomains :many
-- The domains a template's bounces of the window went to, most first.
SELECT split_part(m.email, '@', 2)::text AS domain, count(*)::bigint AS n
FROM transactional_messages t
JOIN messages m ON m.id = t.message_id
WHERE t.workspace_id = @workspace_id AND t.template_id = @template_id AND t.accepted_at >= @since::timestamptz AND m.status = 'bounced' AND NOT t.test
GROUP BY 1
ORDER BY 2 DESC, 1
LIMIT 3;

-- name: TxnAlertHour :one
-- The workspace's last hour: failed messages and the median delivery time.
SELECT count(*) FILTER (WHERE m.status = 'failed')::bigint AS failed,
       count(*) FILTER (WHERE m.delivered_at IS NOT NULL AND m.delivered_at >= t.accepted_at)::bigint AS delivered,
       coalesce((percentile_cont(0.5) WITHIN GROUP (ORDER BY extract(epoch FROM m.delivered_at - t.accepted_at) * 1000)
                 FILTER (WHERE m.delivered_at IS NOT NULL AND m.delivered_at >= t.accepted_at)), -1)::bigint AS median_ms
FROM transactional_messages t
JOIN messages m ON m.id = t.message_id
WHERE t.workspace_id = @workspace_id AND t.accepted_at >= @since::timestamptz AND NOT t.test;

-- name: TxnAlertFailovers :many
-- Messages that left a provider of their route since a time: how many,
-- how many were lost, where they went and how fast they were delivered.
SELECT t.failed_over_from::uuid AS provider_id, coalesce(max(p.name), '')::text AS provider_name,
       count(*)::bigint AS messages,
       count(*) FILTER (WHERE m.status = 'failed')::bigint AS lost,
       min(t.failed_over_at)::timestamptz AS first_at,
       max(t.failed_over_at)::timestamptz AS last_at,
       coalesce(max(np.name), '')::text AS fallback_name,
       coalesce((percentile_cont(0.5) WITHIN GROUP (ORDER BY extract(epoch FROM m.delivered_at - t.accepted_at) * 1000)
                 FILTER (WHERE m.delivered_at IS NOT NULL AND m.delivered_at >= t.accepted_at)), -1)::bigint AS median_ms
FROM transactional_messages t
JOIN messages m ON m.id = t.message_id
LEFT JOIN providers p ON p.id = t.failed_over_from
LEFT JOIN providers np ON np.id = m.provider_id
WHERE t.workspace_id = @workspace_id AND t.failed_over_from IS NOT NULL AND t.failed_over_at >= @since::timestamptz
GROUP BY t.failed_over_from;

-- name: TxnAlertInsert :one
-- Opens an alert; nothing happens when one of its kind (and template) is open.
INSERT INTO transactional_alerts (id, workspace_id, kind, template_id, provider_id, value, threshold, detail, started_at)
VALUES (@id, @workspace_id, @kind, sqlc.narg(template_id), sqlc.narg(provider_id), @value, @threshold, @detail, @started_at)
ON CONFLICT (workspace_id, kind, coalesce(template_id, '00000000-0000-0000-0000-000000000000'::uuid)) WHERE ended_at IS NULL DO NOTHING
RETURNING id;

-- name: TxnAlertRefresh :exec
-- Keeps an open alert's worst value and latest detail.
UPDATE transactional_alerts SET value = greatest(value, @value::numeric), detail = @detail
WHERE workspace_id = @workspace_id AND kind = @kind AND ended_at IS NULL
  AND coalesce(template_id, '00000000-0000-0000-0000-000000000000'::uuid) = coalesce(sqlc.narg(template_id)::uuid, '00000000-0000-0000-0000-000000000000'::uuid);

-- name: TxnAlertEnd :execrows
UPDATE transactional_alerts SET ended_at = @now::timestamptz
WHERE workspace_id = @workspace_id AND kind = @kind AND ended_at IS NULL
  AND coalesce(template_id, '00000000-0000-0000-0000-000000000000'::uuid) = coalesce(sqlc.narg(template_id)::uuid, '00000000-0000-0000-0000-000000000000'::uuid);

-- name: TxnAlertsOpenFor :many
-- A workspace's open alerts of a kind.
SELECT id, template_id FROM transactional_alerts WHERE workspace_id = @workspace_id AND kind = @kind AND ended_at IS NULL;

-- name: TxnAlerts :many
-- Open alerts and those of a period, newest first.
SELECT a.id, a.kind, a.template_id, a.provider_id, a.value::float8 AS value, a.threshold::float8 AS threshold, a.detail, a.started_at, a.ended_at, a.seen_at,
       coalesce(t.key, '')::text AS template_key, coalesce(t.name, '')::text AS template_name, coalesce(p.name, '')::text AS provider_name
FROM transactional_alerts a
LEFT JOIN transactional_templates t ON t.id = a.template_id
LEFT JOIN providers p ON p.id = a.provider_id
WHERE a.workspace_id = @workspace_id AND (a.ended_at IS NULL OR a.started_at >= @since::timestamptz)
ORDER BY a.started_at DESC
LIMIT 50;

-- name: TxnAlertSeen :execrows
UPDATE transactional_alerts SET seen_at = @now::timestamptz, seen_by = @user_id WHERE id = @id AND workspace_id = @workspace_id;

-- name: TxnOverviewStats :one
-- The workspace's numbers over a period (Raw SMTP mail included).
SELECT count(*) FILTER (WHERE m.status IN ('sent', 'delivered', 'deferred', 'bounced', 'complained'))::bigint AS sent,
       count(*) FILTER (WHERE m.status IN ('delivered', 'complained'))::bigint AS delivered,
       count(*) FILTER (WHERE m.status = 'bounced')::bigint AS bounced,
       count(*) FILTER (WHERE m.status = 'complained')::bigint AS complained,
       count(*) FILTER (WHERE m.status = 'deferred')::bigint AS deferred,
       count(*) FILTER (WHERE t.reject_reason = 'suppressed' AND NOT t.test)::bigint AS suppressed,
       count(*) FILTER (WHERE t.reject_reason = 'opted_out' AND NOT t.test)::bigint AS opted_out,
       count(*) FILTER (WHERE m.status = 'failed')::bigint AS failed,
       count(*) FILTER (WHERE m.opened_at IS NOT NULL)::bigint AS opened,
       count(*) FILTER (WHERE m.clicked_at IS NOT NULL)::bigint AS clicked,
       coalesce((percentile_cont(0.5) WITHIN GROUP (ORDER BY extract(epoch FROM m.delivered_at - t.accepted_at) * 1000)
                 FILTER (WHERE m.delivered_at IS NOT NULL AND m.delivered_at >= t.accepted_at)), -1)::bigint AS median_ms,
       count(DISTINCT t.key_id) FILTER (WHERE t.source <> 'smtp' AND t.key_id IS NOT NULL)::bigint AS keys,
       (count(*) FILTER (WHERE t.source = 'smtp') > 0)::boolean AS smtp,
       coalesce(string_agg(DISTINCT t.template_key, ',') FILTER (WHERE m.status = 'complained'), '')::text AS complained_templates
FROM transactional_messages t
JOIN messages m ON m.id = t.message_id
WHERE t.workspace_id = @workspace_id AND t.accepted_at >= @since::timestamptz
  AND (@include_test::boolean OR NOT t.test);

-- name: TxnSendingKeys :many
-- Keys holding transactional:send, for the overview and Setup.
SELECT k.id, k.name, k.prefix, k.preset, k.mode, k.templates, k.categories, k.test_addresses, k.rate_per_second,
       k.created_at, k.expires_at, k.last_used_at, k.last_used_ip, k.revoked_at, coalesce(u.name, '')::text AS created_by_name
FROM api_keys k LEFT JOIN users u ON u.id = k.created_by
WHERE k.workspace_id = @workspace_id AND k.kind = 'api_key' AND 'transactional:send' = ANY(k.permissions) AND k.revoked_at IS NULL
ORDER BY k.created_at;

-- name: TxnFirstRun :one
-- What the first-run state and the guide need to know.
SELECT (SELECT count(*) FROM transactional_templates t WHERE t.workspace_id = @workspace_id AND t.deleted_at IS NULL)::bigint AS templates,
       (SELECT count(*) FROM transactional_templates t WHERE t.workspace_id = @workspace_id AND t.deleted_at IS NULL AND t.live_version IS NOT NULL)::bigint AS published,
       (SELECT count(*) FROM transactional_messages m WHERE m.workspace_id = @workspace_id)::bigint AS messages;

-- name: TxnRoutePriority :one
-- "First in line on shared providers" (on unless turned off).
SELECT coalesce((SELECT priority FROM stream_policies WHERE workspace_id = @workspace_id AND stream = 'transactional'), true)::boolean;

-- name: TxnHourStats :one
-- The last hour's numbers for the alert banner.
SELECT count(*) FILTER (WHERE m.status IN ('sent', 'delivered', 'deferred', 'bounced', 'complained'))::bigint AS sent,
       count(*) FILTER (WHERE m.status = 'bounced')::bigint AS bounced,
       coalesce((percentile_cont(0.5) WITHIN GROUP (ORDER BY extract(epoch FROM m.delivered_at - t.accepted_at) * 1000)
                 FILTER (WHERE m.delivered_at IS NOT NULL AND m.delivered_at >= t.accepted_at)), -1)::bigint AS median_ms
FROM transactional_messages t
JOIN messages m ON m.id = t.message_id
WHERE t.workspace_id = @workspace_id AND t.accepted_at >= @since::timestamptz AND NOT t.test;

-- name: TxnSettingsSetDefaultSender :exec
-- The default From of transactional templates (Settings › Senders).
UPDATE transactional_settings SET default_sender_id = sqlc.narg(sender_id), updated_by = sqlc.narg(updated_by), updated_at = @updated_at
WHERE workspace_id = @workspace_id;

-- name: TxnSMTPFromAddresses :many
-- From addresses mail arriving through the SMTP relay used lately.
SELECT DISTINCT from_email FROM transactional_messages
WHERE workspace_id = @workspace_id AND source = 'smtp' AND accepted_at >= @since::timestamptz AND from_email <> '';

-- name: TxnSetDomainTransactional :exec
UPDATE sending_domains SET transactional = @transactional WHERE id = @id;
