-- Webhook ingest, normalization and the provider event log.

-- name: IngestInsertRaw :exec
INSERT INTO provider_raw_events (id, workspace_id, provider_id, received_at, content_type, body, headers, signature_ok, source)
VALUES (@id, @workspace_id, @provider_id, @received_at, @content_type, @body, @headers, @signature_ok, @source);

-- name: IngestGetRaw :one
SELECT * FROM provider_raw_events WHERE id = @id;

-- name: IngestMarkNormalized :exec
UPDATE provider_raw_events SET normalized_at = @normalized_at, events = @events, normalize_error = @normalize_error WHERE id = @id;

-- name: IngestStaleRaw :many
-- Verified payloads the normalization job never finished (recovery sweep).
SELECT id FROM provider_raw_events
WHERE normalized_at IS NULL AND signature_ok AND received_at < @before
ORDER BY received_at
LIMIT @lim;

-- name: IngestPurgeRaw :execrows
DELETE FROM provider_raw_events WHERE received_at < @before;

-- name: IngestMessageByID :one
SELECT m.*, coalesce(c.name, '')::text AS campaign_name, coalesce(a.name, '')::text AS automation_name
FROM messages m
LEFT JOIN campaigns c ON c.id = m.campaign_id
LEFT JOIN automations a ON a.id = m.automation_id
WHERE m.id = @id AND m.workspace_id = @workspace_id;

-- name: IngestMessageByProviderID :one
SELECT m.*, coalesce(c.name, '')::text AS campaign_name, coalesce(a.name, '')::text AS automation_name
FROM messages m
LEFT JOIN campaigns c ON c.id = m.campaign_id
LEFT JOIN automations a ON a.id = m.automation_id
WHERE m.provider_id = @provider_id AND m.provider_message_id = @provider_message_id;

-- name: IngestInsertEvent :one
INSERT INTO delivery_events (id, workspace_id, message_id, campaign_id, contact_id, provider_id, raw_event_id, type, occurred_at,
                             mailbox_provider, original_type, original_code, diagnostic, bounce_class, url, user_agent,
                             recipient, provider_message_id, source, received_at, dedupe_key)
VALUES (@id, @workspace_id, @message_id, @campaign_id, @contact_id, @provider_id, @raw_event_id, @type, @occurred_at,
        @mailbox_provider, @original_type, @original_code, @diagnostic, @bounce_class, @url, @user_agent,
        @recipient, @provider_message_id, @source, @received_at, @dedupe_key)
ON CONFLICT (provider_id, dedupe_key) WHERE dedupe_key <> '' DO NOTHING
RETURNING id;

-- name: IngestUpdateMessage :exec
-- Moves a message forward: delivered/deferred are replaced by bounces and
-- complaints, never the other way round.
UPDATE messages
SET status = CASE
        WHEN @status::text = 'complained' THEN 'complained'
        WHEN @status::text = 'bounced' AND status <> 'complained' THEN 'bounced'
        WHEN @status::text = 'rejected' AND status IN ('queued', 'sent', 'deferred', 'held') THEN 'rejected'
        WHEN @status::text = 'delivered' AND status IN ('queued', 'sent', 'deferred', 'held') THEN 'delivered'
        WHEN @status::text = 'deferred' AND status IN ('queued', 'sent', 'held') THEN 'deferred'
        ELSE status END,
    delivered_at = CASE WHEN @status::text = 'delivered' THEN coalesce(delivered_at, @at::timestamptz) ELSE delivered_at END,
    error = CASE WHEN @status::text IN ('bounced', 'rejected', 'deferred') AND @error::text <> '' THEN @error::text ELSE error END
WHERE id = @id;

-- name: IngestBumpCounters :exec
INSERT INTO campaign_counters (campaign_id, provider_id, mailbox_provider, variant, delivered, deferred, hard_bounced, soft_bounced, complained, rejected)
VALUES (@campaign_id, @provider_id, @mailbox_provider, @variant, @delivered, @deferred, @hard_bounced, @soft_bounced, @complained, @rejected)
ON CONFLICT (campaign_id, provider_id, mailbox_provider, variant) DO UPDATE
SET delivered = campaign_counters.delivered + EXCLUDED.delivered,
    deferred = campaign_counters.deferred + EXCLUDED.deferred,
    hard_bounced = campaign_counters.hard_bounced + EXCLUDED.hard_bounced,
    soft_bounced = campaign_counters.soft_bounced + EXCLUDED.soft_bounced,
    complained = campaign_counters.complained + EXCLUDED.complained,
    rejected = campaign_counters.rejected + EXCLUDED.rejected;

-- name: IngestAddSuppression :one
INSERT INTO suppressions (id, workspace_id, email, reason, source, source_provider_id, original_code, detail, created_at)
VALUES (@id, @workspace_id, @email, @reason, @source, @source_provider_id, @original_code, @detail, @created_at)
ON CONFLICT (workspace_id, email) DO NOTHING
RETURNING id;

-- name: IngestSetContactStatus :exec
UPDATE contacts SET status = @status, updated_at = now()
WHERE workspace_id = @workspace_id AND email = @email AND status = 'active';

-- name: IngestEventLog :many
-- The provider event log: provider events (not ReputeCast's own opens and
-- clicks), newest first, retained 90 days.
-- The message's kind and automation step write its source label in a
-- reader's language (source keeps the English).
SELECT e.id, e.received_at, e.occurred_at, e.type, e.original_type, e.original_code, e.diagnostic, e.bounce_class,
       e.recipient, e.provider_message_id, e.source, e.provider_id, e.campaign_id,
       coalesce(p.name, '')::text AS provider_name, coalesce(p.adapter, '')::text AS adapter,
       coalesce(r.source, '')::text AS raw_source,
       coalesce(m.kind, '')::text AS message_kind, (m.campaign_id IS NOT NULL)::boolean AS message_campaign,
       (m.automation_id IS NOT NULL)::boolean AS message_automation, coalesce(m.automation_node, '')::text AS message_node
FROM delivery_events e
LEFT JOIN providers p ON p.id = e.provider_id
LEFT JOIN provider_raw_events r ON r.id = e.raw_event_id
LEFT JOIN messages m ON m.id = e.message_id
WHERE e.workspace_id = @workspace_id
  AND e.type IN ('delivered', 'deferred', 'hard_bounce', 'soft_bounce', 'complaint', 'rejected')
  AND e.received_at >= @since
  AND (sqlc.narg(provider_id)::uuid IS NULL OR e.provider_id = sqlc.narg(provider_id)::uuid)
  AND (@type::text = '' OR e.type = @type::text)
  AND (@query::text = '' OR e.recipient ILIKE '%' || @query::text || '%' OR e.provider_message_id = @query::text)
ORDER BY e.received_at DESC, e.id DESC
LIMIT @lim OFFSET @off;

-- name: IngestEventLogCount :one
SELECT count(*)::bigint FROM delivery_events e
WHERE e.workspace_id = @workspace_id
  AND e.type IN ('delivered', 'deferred', 'hard_bounce', 'soft_bounce', 'complaint', 'rejected')
  AND e.received_at >= @since
  AND (sqlc.narg(provider_id)::uuid IS NULL OR e.provider_id = sqlc.narg(provider_id)::uuid)
  AND (@type::text = '' OR e.type = @type::text)
  AND (@query::text = '' OR e.recipient ILIKE '%' || @query::text || '%' OR e.provider_message_id = @query::text);
