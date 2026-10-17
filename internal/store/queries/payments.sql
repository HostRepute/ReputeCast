-- Payments (internal/earn): payment provider connections and the payments
-- event log.

-- name: PayConnections :many
-- The workspace's connections that aren't disconnected, live first.
SELECT sqlc.embed(pc), coalesce(u.name, '')::text AS connected_by_name
FROM payment_connections pc
LEFT JOIN users u ON u.id = pc.connected_by
WHERE pc.workspace_id = @workspace_id AND pc.status <> 'disconnected'
ORDER BY (pc.mode = 'live') DESC, pc.created_at;

-- name: PayConnection :one
SELECT sqlc.embed(pc), coalesce(u.name, '')::text AS connected_by_name
FROM payment_connections pc
LEFT JOIN users u ON u.id = pc.connected_by
WHERE pc.id = @id AND pc.workspace_id = @workspace_id;

-- name: PayConnectionByToken :one
-- The connection a webhook request is for.
SELECT * FROM payment_connections
WHERE webhook_token = @webhook_token AND status <> 'disconnected';

-- name: PayConnectionsActive :many
-- Every connection of the instance with a key, for the hourly check.
SELECT pc.* FROM payment_connections pc
JOIN workspaces w ON w.id = pc.workspace_id AND w.deleted_at IS NULL
WHERE pc.status = 'connected' AND pc.key_sealed IS NOT NULL;

-- name: PayConnectionInsert :exec
INSERT INTO payment_connections (id, workspace_id, provider, mode, status, step, key_sealed, key_hint, account_id, account_name, account_country,
    currency, tax_enabled, permissions, checked_at, webhook_token, demo, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @provider, @mode, 'setup', @step, @key_sealed, @key_hint, @account_id, @account_name, @account_country,
    @currency, @tax_enabled, @permissions, @checked_at, @webhook_token, @demo, @created_by, @checked_at, @checked_at);

-- name: PayConnectionChecked :exec
-- A key checked again (Check again, or a replaced key).
UPDATE payment_connections SET key_sealed = coalesce(sqlc.narg(key_sealed), key_sealed), key_hint = @key_hint,
    account_id = @account_id, account_name = @account_name, account_country = @account_country, currency = @currency,
    tax_enabled = @tax_enabled, permissions = @permissions, checked_at = @checked_at, key_rejected_at = NULL,
    step = CASE WHEN status = 'setup' THEN @step ELSE step END, updated_at = @checked_at
WHERE id = @id;

-- name: PayConnectionSetStep :exec
UPDATE payment_connections SET step = @step, updated_at = @at WHERE id = @id;

-- name: PayConnectionSetWebhook :exec
-- The endpoint registered in the provider (or added by hand), and its sealed secret.
UPDATE payment_connections SET webhook_token = @webhook_token, webhook_id = @webhook_id, webhook_secret_sealed = @webhook_secret_sealed,
    webhook_manual = @webhook_manual, webhook_events = @webhook_events, webhook_registered_at = @at, webhook_missing_at = NULL, updated_at = @at
WHERE id = @id;

-- name: PayConnectionSetReach :exec
UPDATE payment_connections SET webhook_reach = @webhook_reach, updated_at = @at WHERE id = @id;

-- name: PayConnectionFinish :exec
UPDATE payment_connections SET status = 'connected', step = 'done', portal_configuration = @portal_configuration,
    connected_by = coalesce(connected_by, sqlc.narg(connected_by)), connected_at = coalesce(connected_at, @at::timestamptz), updated_at = @at
WHERE id = @id;

-- name: PayConnectionSetPortal :exec
UPDATE payment_connections SET portal_configuration = @portal_configuration, updated_at = @at WHERE id = @id;

-- name: PayConnectionDisconnect :execrows
-- Disconnecting deletes the key and the signing secret from ReputeCast.
UPDATE payment_connections SET status = 'disconnected', key_sealed = NULL, webhook_secret_sealed = NULL, webhook_token = NULL,
    disconnected_at = @at, disconnected_by = sqlc.narg(by), updated_at = @at
WHERE id = @id AND workspace_id = @workspace_id AND status <> 'disconnected';

-- name: PayConnectionTouchEvent :exec
-- A verified event arrived: freshness for the webhook card.
UPDATE payment_connections SET
    first_event_at = coalesce(first_event_at, @received_at), first_event_type = CASE WHEN first_event_at IS NULL THEN @type ELSE first_event_type END,
    last_event_at = greatest(coalesce(last_event_at, @received_at), @received_at),
    last_event_type = CASE WHEN last_event_at IS NULL OR @received_at >= last_event_at THEN @type ELSE last_event_type END,
    webhook_missing_at = NULL
WHERE id = @id;

-- name: PayConnectionKeyRejected :exec
UPDATE payment_connections SET key_rejected_at = coalesce(key_rejected_at, @at::timestamptz), updated_at = @at WHERE id = @id;

-- name: PayConnectionWebhookMissing :exec
UPDATE payment_connections SET webhook_missing_at = coalesce(webhook_missing_at, @at::timestamptz), updated_at = @at WHERE id = @id;

-- name: PayConnectionGapChecked :exec
UPDATE payment_connections SET gap_checked_at = @at, gap_history = @gap_history WHERE id = @id;

-- name: PayEventInsert :one
-- Stores an event once per provider event id: a replay or a duplicate
-- delivery inserts nothing (no row comes back).
INSERT INTO payment_events (id, workspace_id, connection_id, mode, provider_event_id, type, source, occurred_at, received_at, payload, signature_ok, status)
VALUES (@id, @workspace_id, @connection_id, @mode, @provider_event_id, @type, @source, @occurred_at, @received_at, @payload, @signature_ok, @status)
ON CONFLICT (workspace_id, mode, provider_event_id) WHERE provider_event_id <> '' DO NOTHING
RETURNING id;

-- name: PayEventGet :one
SELECT * FROM payment_events WHERE id = @id;

-- name: PayEventByProviderID :one
SELECT * FROM payment_events WHERE workspace_id = @workspace_id AND mode = @mode AND provider_event_id = @provider_event_id;

-- name: PayEventResult :exec
UPDATE payment_events SET status = @status, outcome = @outcome, outcome_detail = @outcome_detail, contact_id = sqlc.narg(contact_id),
    subscription_id = sqlc.narg(subscription_id), amount_cents = sqlc.narg(amount_cents), currency = @currency, applied_at = @at,
    error = @error, attempts = attempts + 1
WHERE id = @id;

-- name: PayEventFailed :exec
UPDATE payment_events SET status = CASE WHEN attempts + 1 >= 8 THEN 'failed' ELSE status END, error = @error, attempts = attempts + 1 WHERE id = @id;

-- name: PayEventsPending :many
-- Events stored but not applied yet (a job that never ran), oldest first.
SELECT id FROM payment_events WHERE status = 'pending' AND signature_ok AND received_at < @before ORDER BY received_at LIMIT @lim;

-- name: PayEventsPurge :execrows
-- Raw events are kept 90 days.
DELETE FROM payment_events WHERE received_at < @before;

-- name: PayEventLog :many
-- One page of the payments event log, newest first.
SELECT e.id, e.mode, e.provider_event_id, e.type, e.source, e.occurred_at, e.received_at, e.status, e.outcome, e.outcome_detail,
    e.amount_cents, e.currency, e.contact_id, e.subscription_id, e.signature_ok,
    coalesce(nullif(trim(c.first_name || ' ' || c.last_name), ''), c.email, '')::text AS member_name,
    coalesce(c.email, '')::text AS member_email
FROM payment_events e
LEFT JOIN contacts c ON c.id = e.contact_id
WHERE e.workspace_id = @workspace_id AND e.received_at >= @since
  AND (@include_test::boolean OR e.mode = 'live')
  AND (@type::text = '' OR e.type = @type)
  AND (@q::text = '' OR e.type ILIKE '%' || @q || '%' OR e.provider_event_id ILIKE '%' || @q || '%'
       OR c.email ILIKE '%' || @q || '%' OR (c.first_name || ' ' || c.last_name) ILIKE '%' || @q || '%'
       OR e.payload::text ILIKE '%' || @q || '%')
ORDER BY e.received_at DESC, e.id DESC
LIMIT @lim OFFSET @off;

-- name: PayEventLogCount :one
SELECT count(*)::bigint FROM payment_events e
LEFT JOIN contacts c ON c.id = e.contact_id
WHERE e.workspace_id = @workspace_id AND e.received_at >= @since
  AND (@include_test::boolean OR e.mode = 'live')
  AND (@type::text = '' OR e.type = @type)
  AND (@q::text = '' OR e.type ILIKE '%' || @q || '%' OR e.provider_event_id ILIKE '%' || @q || '%'
       OR c.email ILIKE '%' || @q || '%' OR (c.first_name || ' ' || c.last_name) ILIKE '%' || @q || '%'
       OR e.payload::text ILIKE '%' || @q || '%');

-- name: PayEventTypes24h :many
-- Events per type over the last day, for the webhook card.
SELECT type, count(*)::bigint AS n FROM payment_events
WHERE connection_id = @connection_id AND signature_ok AND received_at >= @since
GROUP BY type ORDER BY n DESC, type;

-- name: PayRejectedSince :one
SELECT count(*)::bigint FROM payment_events WHERE connection_id = @connection_id AND NOT signature_ok AND received_at >= @since;

-- name: PayEventTypesSeen :many
-- The event types in the log, for its filter.
SELECT DISTINCT type FROM payment_events WHERE workspace_id = @workspace_id AND type <> '' ORDER BY type;
