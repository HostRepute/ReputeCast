-- Connect a sending provider (the setup wizard), Settings › Providers
-- and the system mailer. Query names start with Setup, Connect or Sysmail.

-- name: SetupCreate :one
INSERT INTO provider_setups (id, workspace_id, adapter, step, state, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @adapter, 'connection', @state, @created_by, @created_at, @created_at)
RETURNING *;

-- name: SetupGet :one
SELECT * FROM provider_setups WHERE workspace_id = @workspace_id AND id = @id;

-- name: SetupGetAny :one
SELECT * FROM provider_setups WHERE id = @id;

-- name: SetupLock :one
SELECT * FROM provider_setups WHERE id = @id FOR UPDATE;

-- name: SetupOpen :many
-- Unfinished setups of a workspace, newest first (resume "Connect SES").
SELECT * FROM provider_setups
WHERE workspace_id = @workspace_id AND status = 'open'
  AND (sqlc.narg(adapter)::text IS NULL OR adapter = sqlc.narg(adapter)::text)
ORDER BY updated_at DESC
LIMIT 20;

-- name: SetupList :many
-- A workspace's setups, newest first, optionally of one adapter or status.
SELECT * FROM provider_setups
WHERE workspace_id = @workspace_id
  AND (sqlc.narg(adapter)::text IS NULL OR adapter = sqlc.narg(adapter)::text)
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text)
ORDER BY updated_at DESC
LIMIT 50;

-- name: SetupByProvider :one
-- The latest setup that connected a provider.
SELECT * FROM provider_setups WHERE provider_id = @provider_id ORDER BY created_at DESC LIMIT 1;

-- name: SetupSave :one
UPDATE provider_setups
SET step = @step, status = @status, state = @state, provider_id = @provider_id, updated_at = @updated_at, finished_at = @finished_at
WHERE id = @id
RETURNING *;

-- name: ConnectInsertProvider :one
INSERT INTO providers (id, workspace_id, adapter, name, label, config, credentials_enc, webhook_token, webhook_secret_enc,
                       status, status_detail, ip_mode, ips, capabilities, limits, health, pricing, connected_by, connected_at, last_checked_at)
VALUES (@id, @workspace_id, @adapter, @name, @label, @config, @credentials_enc, @webhook_token, @webhook_secret_enc,
        @status, @status_detail, @ip_mode, @ips::text[], @capabilities, @limits, @health, @pricing, @connected_by, @connected_at, @connected_at)
RETURNING *;

-- name: ConnectUpdateProvider :one
-- Settings and credentials from a repeated connection step (new key, other region).
UPDATE providers
SET label = @label, config = @config, credentials_enc = @credentials_enc, status = @status, status_detail = @status_detail,
    ip_mode = @ip_mode, ips = @ips::text[], health = @health, last_checked_at = @checked_at
WHERE id = @id
RETURNING *;

-- name: ConnectSetProviderConfig :exec
UPDATE providers SET config = @config WHERE id = @id;

-- name: ConnectSetWebhookSecret :exec
UPDATE providers SET webhook_secret_enc = @webhook_secret_enc WHERE id = @id;

-- name: ConnectProviderNames :many
SELECT name FROM providers WHERE workspace_id = @workspace_id AND disconnected_at IS NULL;

-- name: ConnectSameAdapter :many
-- Other connections of an adapter on this instance (resource-name clashes
-- when two workspaces share one provider account).
SELECT id, workspace_id, config FROM providers
WHERE adapter = @adapter AND disconnected_at IS NULL AND id <> @id;

-- name: ConnectProviderRouted :one
-- Whether any routing rule of the workspace uses the provider.
SELECT EXISTS (
  SELECT 1 FROM routing_rules r
  WHERE r.workspace_id = @workspace_id
    AND (r.fallback_provider_id = @provider_id::uuid OR r.splits @> jsonb_build_array(jsonb_build_object('provider_id', @provider_id::text)))
)::boolean AS routed;

-- name: ConnectMessage :one
SELECT * FROM messages WHERE workspace_id = @workspace_id AND id = @id;

-- name: ConnectMessageEvents :many
SELECT * FROM delivery_events WHERE message_id = @message_id ORDER BY occurred_at, received_at;

-- name: ConnectFirstWebhook :one
-- The first event payload about a message (its provider message id is in
-- the body) a provider sent in a window after the send, by webhook or
-- through the queue ReputeCast polls (PR-4): the test send's "event came
-- back" step and the setup's event check (PR-3), with the body to name the
-- event and how it came.
SELECT received_at, body, source FROM provider_raw_events
WHERE provider_id = @provider_id AND source IN ('webhook', 'queue') AND received_at >= @since AND received_at < @until
  AND position(convert_to(@provider_message_id::text, 'UTF8') IN body) > 0
ORDER BY received_at
LIMIT 1;

-- name: ConnectDeliveryEvent :one
-- The delivery event of a message, reported for its recipient by webhook or
-- through the queue ReputeCast polls (PR-3, PR-4), once normalized: the SES
-- event check passes on the mailbox simulator's Delivery event, not on the
-- Send event SES publishes first (review R3.A1).
SELECT e.received_at, r.source
FROM delivery_events e
JOIN provider_raw_events r ON r.id = e.raw_event_id
WHERE e.provider_id = @provider_id::uuid AND e.type = 'delivered' AND r.source IN ('webhook', 'queue')
  AND (e.message_id = @message_id::uuid OR e.provider_message_id = @provider_message_id::text)
  AND lower(e.recipient) = lower(@recipient::text)
  AND e.received_at >= @since AND e.received_at < @until
ORDER BY e.received_at
LIMIT 1;

-- name: ConnectFirstEventSince :one
-- The first event a provider delivered after a time, by webhook or queue:
-- its endpoint was reached from outside (PR-9).
SELECT received_at, source FROM provider_raw_events
WHERE provider_id = @provider_id AND source IN ('webhook', 'queue') AND received_at >= @since
ORDER BY received_at
LIMIT 1;

-- name: ConnectListNames :many
SELECT name FROM lists WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY created_at, name;

-- name: ConnectCreateList :exec
INSERT INTO lists (id, workspace_id, name, created_at) VALUES (@id, @workspace_id, @name, @created_at);

-- name: ConnectSuppressionSyncMode :exec
INSERT INTO provider_suppression_sync (provider_id, mode, status) VALUES (@provider_id, @mode, 'ok')
ON CONFLICT (provider_id) DO UPDATE SET mode = EXCLUDED.mode;

-- name: ConnectSetSendThresholds :exec
UPDATE workspaces SET send_thresholds = @send_thresholds, updated_at = @updated_at WHERE id = @id;

-- name: ConnectProvidersWithSecrets :many
-- Connected providers whose webhooks are authenticated with a secret.
SELECT * FROM providers
WHERE workspace_id = @workspace_id AND disconnected_at IS NULL AND webhook_secret_enc IS NOT NULL
ORDER BY connected_at;

-- name: SysmailInsertMessage :exec
-- A message the system mailer or a setup's test sends (stream system, or
-- the stream a test goes on).
INSERT INTO messages (id, workspace_id, stream, email, provider_id, mailbox_provider, status, queued_at, kind)
VALUES (@id, @workspace_id, @stream, @email, @provider_id, @mailbox_provider, 'queued', @queued_at, @kind);

-- name: SysmailMarkSent :exec
UPDATE messages
SET status = @status, provider_message_id = @provider_message_id, sent_at = @sent_at, error = @error
WHERE id = @id;

-- name: SysmailWorkspacesForEmail :many
-- The workspaces of the person a message without a workspace goes to,
-- oldest membership first.
SELECT w.id FROM workspaces w
JOIN workspace_members m ON m.workspace_id = w.id AND m.status = 'active'
JOIN users u ON u.id = m.user_id
WHERE u.email = @email AND w.deleted_at IS NULL
ORDER BY m.joined_at, w.created_at;

-- name: SysmailWorkspaces :many
-- Every workspace, oldest first (instance-level mail falls back to the
-- first workspace with a System route).
SELECT id FROM workspaces WHERE deleted_at IS NULL ORDER BY created_at, id;

-- name: ConnectActiveContacts :one
SELECT count(*)::bigint AS n FROM contacts WHERE workspace_id = @workspace_id AND status = 'active' AND erased_at IS NULL;
