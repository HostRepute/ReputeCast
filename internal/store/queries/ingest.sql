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
-- One event per provider and dedupe key: the key is claimed in
-- delivery_event_keys first, so the check holds across monthly partitions
-- (review 2026-11 OP-8). An event without a key (or a provider) is always
-- stored. ON CONFLICT DO NOTHING keeps the unpartitioned table's unique
-- index from failing on an event stored before the keys table existed.
WITH claimed AS (
    INSERT INTO delivery_event_keys (provider_id, dedupe_key, received_at)
    SELECT sqlc.narg(provider_id)::uuid, sqlc.arg(dedupe_key)::text, sqlc.arg(received_at)::timestamptz
    WHERE sqlc.narg(provider_id)::uuid IS NOT NULL AND sqlc.arg(dedupe_key)::text <> ''
    ON CONFLICT DO NOTHING
    RETURNING 1
)
INSERT INTO delivery_events (id, workspace_id, message_id, campaign_id, contact_id, provider_id, raw_event_id, type, occurred_at,
                             mailbox_provider, original_type, original_code, diagnostic, bounce_class, url, user_agent,
                             recipient, provider_message_id, source, received_at, dedupe_key)
SELECT sqlc.arg(id), sqlc.arg(workspace_id), sqlc.narg(message_id), sqlc.narg(campaign_id), sqlc.narg(contact_id), sqlc.narg(provider_id),
       sqlc.narg(raw_event_id), sqlc.arg(type), sqlc.arg(occurred_at),
       sqlc.arg(mailbox_provider), sqlc.arg(original_type), sqlc.arg(original_code), sqlc.arg(diagnostic), sqlc.arg(bounce_class),
       sqlc.arg(url), sqlc.arg(user_agent), sqlc.arg(recipient), sqlc.arg(provider_message_id), sqlc.arg(source),
       sqlc.arg(received_at), sqlc.arg(dedupe_key)
WHERE sqlc.narg(provider_id)::uuid IS NULL OR sqlc.arg(dedupe_key)::text = '' OR EXISTS (SELECT 1 FROM claimed)
ON CONFLICT DO NOTHING
RETURNING id;

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

-- name: IngestBindToken :one
-- Ties the one-time token of a webhook signature to the payload it first
-- came with (both hashed), and returns the payload it is tied to: the one
-- already there while it is current, this one otherwise. In one statement,
-- so of two requests with the same token at once, the second gets the
-- first one's payload.
INSERT INTO webhook_tokens AS w (scope, token_hash, payload_hash, expires_at)
VALUES (@scope, @token_hash, @payload_hash, @expires_at)
ON CONFLICT (scope, token_hash) DO UPDATE SET
    payload_hash = CASE WHEN w.expires_at < @now::timestamptz THEN excluded.payload_hash ELSE w.payload_hash END,
    expires_at   = CASE WHEN w.expires_at < @now::timestamptz THEN excluded.expires_at ELSE w.expires_at END
RETURNING payload_hash;

-- name: IngestPurgeTokens :execrows
DELETE FROM webhook_tokens WHERE expires_at < @before;
