-- Outgoing webhooks and their delivery log.

-- name: HooksList :many
SELECT h.id, h.workspace_id, h.name, h.description, h.url, h.events, h.enabled, h.created_by, h.created_at,
       h.last_status, h.last_delivery_at, h.failing_since, h.secret_rotated_at, h.paused_at, h.previous_secret_until, h.batch_email,
       coalesce(s.total, 0)::bigint AS total_24h, coalesce(s.ok, 0)::bigint AS ok_24h,
       coalesce(q.queued, 0)::bigint AS queued
FROM webhooks h
LEFT JOIN LATERAL (
    SELECT count(*) AS total, count(*) FILTER (WHERE d.state = 'delivered') AS ok
    FROM webhook_deliveries d
    WHERE d.webhook_id = h.id AND d.created_at > now() - interval '24 hours'
) s ON true
LEFT JOIN LATERAL (
    SELECT count(*) AS queued FROM webhook_deliveries d
    WHERE d.webhook_id = h.id AND d.state IN ('pending','retrying','held','delivering','batching')
) q ON true
WHERE h.workspace_id = @workspace_id
ORDER BY h.created_at;

-- name: HooksGet :one
SELECT * FROM webhooks WHERE id = @id AND workspace_id = @workspace_id;

-- name: HooksCreate :one
INSERT INTO webhooks (id, workspace_id, name, description, url, events, batch_email, secret_enc, enabled, created_by, secret_rotated_at)
VALUES (@id, @workspace_id, @name, @description, @url, @events, @batch_email, @secret_enc, true, @created_by, now())
RETURNING *;

-- name: HooksUpdate :one
-- A demo endpoint stays one only while its address is unchanged.
UPDATE webhooks SET name = @name, description = @description, url = @url, events = @events, batch_email = @batch_email,
    demo = demo AND url = @url, updated_at = now()
WHERE id = @id AND workspace_id = @workspace_id
RETURNING *;

-- name: HooksSetSecret :exec
-- The new secret signs at once; the one it replaces keeps signing beside
-- it until previous_until, so receivers can switch (review 2026-11 IN-1).
UPDATE webhooks SET previous_secret_enc = secret_enc, previous_secret_until = sqlc.narg(previous_until)::timestamptz,
       secret_enc = @secret_enc, secret_rotated_at = now(), updated_at = now()
WHERE id = @id AND workspace_id = @workspace_id;

-- name: HooksSetEnabled :exec
UPDATE webhooks SET enabled = @enabled, paused_at = CASE WHEN @enabled::boolean THEN NULL ELSE now() END, updated_at = now()
WHERE id = @id AND workspace_id = @workspace_id;

-- name: HooksDelete :execrows
DELETE FROM webhooks WHERE id = @id AND workspace_id = @workspace_id;

-- name: HooksMatching :many
-- Endpoints subscribed to an event: exact name, "group.*" or "*".
SELECT id, enabled, batch_email FROM webhooks
WHERE workspace_id = @workspace_id
  AND (@event::text = ANY(events) OR split_part(@event::text, '.', 1) || '.*' = ANY(events) OR '*' = ANY(events));

-- name: HooksInsertDelivery :exec
INSERT INTO webhook_deliveries (id, webhook_id, event, event_id, payload, attempt, state, next_attempt_at, max_attempts)
VALUES (@id, @webhook_id, @event, @event_id, @payload, 0, @state, @next_attempt_at, @max_attempts);

-- name: HooksBatching :many
-- Endpoints with email.* events waiting to be folded into a batch (review
-- 2026-11 EN-8), whether or not they still batch: a switched-off setting
-- still sends what was waiting.
SELECT h.id, h.enabled, w.slug FROM webhooks h
JOIN workspaces w ON w.id = h.workspace_id
WHERE EXISTS (SELECT 1 FROM webhook_deliveries d WHERE d.webhook_id = h.id AND d.state = 'batching')
ORDER BY h.id;

-- name: HooksTakeBatch :many
-- Takes an endpoint's oldest waiting events, up to @lim, for one batch.
-- RETURNING doesn't keep the order: the caller sorts by created_at, id.
DELETE FROM webhook_deliveries
WHERE id IN (
    SELECT d.id FROM webhook_deliveries d
    WHERE d.webhook_id = @webhook_id AND d.state = 'batching'
    ORDER BY d.created_at, d.id
    LIMIT @lim
    FOR UPDATE SKIP LOCKED
)
RETURNING id, payload, created_at;

-- name: HooksDeliveries :many
SELECT d.id, d.webhook_id, d.event, d.event_id, d.attempt, d.max_attempts, d.state, d.status_code, d.duration_ms,
       d.error, d.created_at, d.last_attempt_at, d.next_attempt_at, d.delivered_at, h.url AS webhook_url,
       coalesce(d.payload ->> 'subtype', d.payload -> 'data' ->> 'subtype', '')::text AS subtype
FROM webhook_deliveries d
JOIN webhooks h ON h.id = d.webhook_id
WHERE h.workspace_id = @workspace_id
  AND (d.attempt > 0 OR sqlc.narg(state)::text IN ('pending','held'))
  AND (sqlc.narg(webhook_id)::uuid IS NULL OR d.webhook_id = sqlc.narg(webhook_id))
  AND (sqlc.narg(state)::text IS NULL OR d.state = sqlc.narg(state)
       OR (sqlc.narg(state)::text = 'failed' AND d.state IN ('retrying','dead')))
ORDER BY coalesce(d.last_attempt_at, d.created_at) DESC, d.id DESC
LIMIT @lim;

-- name: HooksGetDelivery :one
SELECT d.*, h.url AS webhook_url, h.workspace_id
FROM webhook_deliveries d JOIN webhooks h ON h.id = d.webhook_id
WHERE d.id = @id AND h.workspace_id = @workspace_id;

-- name: HooksClaimDelivery :one
-- Takes a due delivery for one attempt. Returns no row when someone else has it.
UPDATE webhook_deliveries d SET state = 'delivering', last_attempt_at = now()
FROM webhooks h
WHERE d.id = @id AND h.id = d.webhook_id AND h.enabled
  AND d.state IN ('pending','retrying') AND (d.next_attempt_at IS NULL OR d.next_attempt_at <= now() + interval '5 seconds')
RETURNING d.id, d.webhook_id, d.event, d.event_id, d.payload, d.attempt, d.max_attempts, h.url, h.secret_enc, h.workspace_id, h.demo,
          h.previous_secret_enc, h.previous_secret_until;

-- name: HooksFinishDelivery :exec
UPDATE webhook_deliveries
SET state = @state, attempt = attempt + 1, status_code = @status_code, duration_ms = @duration_ms,
    response = @response, error = @error, next_attempt_at = @next_attempt_at,
    delivered_at = CASE WHEN @state::text = 'delivered' THEN now() ELSE delivered_at END,
    enqueued_at = CASE WHEN @state::text = 'retrying' THEN now() ELSE NULL END
WHERE id = @id;

-- name: HooksRecordResult :exec
UPDATE webhooks
SET last_status = @last_status, last_delivery_at = now(),
    failing_since = CASE WHEN @ok::boolean THEN NULL ELSE coalesce(failing_since, now()) END
WHERE id = @id;

-- name: HooksDue :many
-- Deliveries ready for an attempt that have no job queued (or a lost one).
-- While an endpoint is failing only its oldest open delivery is tried (the
-- probe); the rest wait and replay once the endpoint answers 2xx again. The
-- probes are found once per failing endpoint, not once per waiting delivery.
WITH probe AS (
    SELECT DISTINCT ON (d3.webhook_id) d3.webhook_id, d3.id
    FROM webhook_deliveries d3 JOIN webhooks h3 ON h3.id = d3.webhook_id AND h3.enabled AND NOT h3.demo AND h3.failing_since IS NOT NULL
    WHERE d3.state IN ('pending','retrying','delivering')
    ORDER BY d3.webhook_id, d3.created_at, d3.id
)
UPDATE webhook_deliveries d SET enqueued_at = now()
WHERE d.id IN (
    SELECT d2.id FROM webhook_deliveries d2 JOIN webhooks h ON h.id = d2.webhook_id AND h.enabled AND NOT h.demo
    LEFT JOIN probe p ON p.webhook_id = h.id
    WHERE d2.state IN ('pending','retrying') AND coalesce(d2.next_attempt_at, d2.created_at) <= now()
      AND (d2.enqueued_at IS NULL OR d2.enqueued_at < now() - interval '10 minutes')
      AND (h.failing_since IS NULL OR d2.id = p.id)
    ORDER BY d2.created_at, d2.id
    LIMIT @lim
    FOR UPDATE OF d2 SKIP LOCKED
)
RETURNING d.id;

-- name: HooksRequeue :many
-- Retry now / Retry all failed / Replay: back to pending for an immediate attempt.
UPDATE webhook_deliveries d SET state = 'pending', next_attempt_at = now(), enqueued_at = now(),
    attempt = CASE WHEN d.state = 'dead' THEN 0 ELSE d.attempt END
FROM webhooks h
WHERE h.id = d.webhook_id AND h.workspace_id = @workspace_id
  AND d.state IN ('retrying','dead')
  AND (sqlc.narg(id)::uuid IS NULL OR d.id = sqlc.narg(id))
  AND (sqlc.narg(webhook_id)::uuid IS NULL OR d.webhook_id = sqlc.narg(webhook_id))
RETURNING d.id;

-- name: HooksReleaseHeld :many
UPDATE webhook_deliveries SET state = 'pending', next_attempt_at = now(), enqueued_at = now()
WHERE webhook_id = @webhook_id AND state = 'held'
RETURNING id;

-- name: HooksPurgeDeliveries :execrows
DELETE FROM webhook_deliveries
WHERE (state = 'dead' AND coalesce(last_attempt_at, created_at) < now() - interval '7 days')
   OR (state = 'delivered' AND created_at < now() - interval '30 days');

-- name: HooksStaleDelivering :exec
-- A worker that died mid-attempt leaves a row in 'delivering'; give it back.
UPDATE webhook_deliveries SET state = 'retrying', next_attempt_at = now(), enqueued_at = NULL
WHERE state = 'delivering' AND last_attempt_at < now() - interval '5 minutes';

-- name: HooksListen :one
-- Whether any endpoint of the workspace takes the event (a cheap check
-- before building a payload per message).
SELECT webhook_listens(@workspace_id::uuid, @event::text)::boolean;

-- name: HooksEmailMessage :one
-- What an email.* event says about the message.
SELECT m.id, m.stream, m.email, m.contact_id, m.campaign_id, m.automation_id, coalesce(m.automation_node, '')::text AS automation_node,
       coalesce(p.name, '')::text AS provider, coalesce(m.provider_message_id, '')::text AS provider_message_id, m.mailbox_provider, w.slug
FROM messages m
JOIN workspaces w ON w.id = m.workspace_id
LEFT JOIN providers p ON p.id = m.provider_id
WHERE m.id = @id;

-- name: HooksAlertFailing :one
-- Marks a failing endpoint as alerted, once per failing spell: after an
-- hour of failures, or when a delivery is given up on (review 2026-11
-- IN-2). No row when there is nothing to tell.
UPDATE webhooks h SET alerted_at = sqlc.arg(now)::timestamptz
FROM workspaces w
WHERE h.id = sqlc.arg(id)::uuid AND w.id = h.workspace_id AND h.failing_since IS NOT NULL
  AND (h.failing_since <= sqlc.arg(failing_before)::timestamptz OR sqlc.arg(dead)::boolean)
  AND (h.alerted_at IS NULL OR h.alerted_at < h.failing_since)
RETURNING h.id, h.workspace_id, w.slug, h.url, h.failing_since,
          (SELECT count(*) FROM webhook_deliveries d WHERE d.webhook_id = h.id AND d.state IN ('retrying', 'dead') AND d.last_attempt_at >= h.failing_since)::bigint AS failed,
          (SELECT count(*) FROM webhook_deliveries d WHERE d.webhook_id = h.id AND d.state = 'dead' AND d.last_attempt_at >= h.failing_since)::bigint AS dead;
