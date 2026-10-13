-- Instance admin › Audit log (W14): every workspace's entries and the
-- instance-level ones (workspace_id IS NULL), with the same filters as the
-- workspace view plus a workspace filter.

-- name: InstanceAuditList :many
SELECT e.id, e.workspace_id, e.occurred_at, e.actor_type, e.actor_id, e.actor_name, e.action, e.target_type, e.target_id,
       e.target_name, e.result, e.reason, e.ip, e.user_agent, e.request_id, e.metadata,
       coalesce(u.avatar_color, '')::text AS actor_color, coalesce(w.name, '')::text AS workspace_name, coalesce(w.slug, '')::text AS workspace_slug,
       (e.before IS NOT NULL AND e.after IS NOT NULL)::boolean AS has_changes
FROM audit_events e
LEFT JOIN users u ON e.actor_type = 'user' AND u.id = e.actor_id
LEFT JOIN workspaces w ON w.id = e.workspace_id
WHERE (NOT @instance_only::boolean OR e.workspace_id IS NULL)
  AND (sqlc.narg(workspace_id)::uuid IS NULL OR e.workspace_id = sqlc.narg(workspace_id))
  AND (sqlc.narg(actor_type)::text IS NULL OR e.actor_type = sqlc.narg(actor_type))
  AND (sqlc.narg(result)::text IS NULL OR e.result = sqlc.narg(result))
  AND (sqlc.narg(since)::timestamptz IS NULL OR e.occurred_at >= sqlc.narg(since))
  AND (sqlc.narg(until)::timestamptz IS NULL OR e.occurred_at < sqlc.narg(until))
  AND (sqlc.narg(search)::text IS NULL OR e.target_name ILIKE '%' || sqlc.narg(search)::text || '%' OR e.actor_name ILIKE '%' || sqlc.narg(search)::text || '%'
       OR e.target_id = sqlc.narg(search)::text OR e.request_id = sqlc.narg(search)::text OR e.metadata ->> 'reference' = sqlc.narg(search)::text)
  AND (sqlc.narg(cursor_at)::timestamptz IS NULL OR e.occurred_at < sqlc.narg(cursor_at)::timestamptz
       OR (e.occurred_at = sqlc.narg(cursor_at)::timestamptz AND e.id < sqlc.narg(cursor_id)::uuid))
ORDER BY e.occurred_at DESC, e.id DESC
LIMIT @lim;

-- name: InstanceAuditExport :many
-- Same filters as InstanceAuditList, with before/after for exports.
SELECT e.id, e.workspace_id, e.occurred_at, e.actor_type, e.actor_id, e.actor_name, e.action, e.target_type, e.target_id,
       e.target_name, e.result, e.reason, e.ip, e.user_agent, e.request_id, e.before, e.after, e.metadata,
       coalesce(w.slug, '')::text AS workspace_slug
FROM audit_events e
LEFT JOIN workspaces w ON w.id = e.workspace_id
WHERE (NOT @instance_only::boolean OR e.workspace_id IS NULL)
  AND (sqlc.narg(workspace_id)::uuid IS NULL OR e.workspace_id = sqlc.narg(workspace_id))
  AND (sqlc.narg(actor_type)::text IS NULL OR e.actor_type = sqlc.narg(actor_type))
  AND (sqlc.narg(result)::text IS NULL OR e.result = sqlc.narg(result))
  AND (sqlc.narg(since)::timestamptz IS NULL OR e.occurred_at >= sqlc.narg(since))
  AND (sqlc.narg(until)::timestamptz IS NULL OR e.occurred_at < sqlc.narg(until))
  AND (sqlc.narg(search)::text IS NULL OR e.target_name ILIKE '%' || sqlc.narg(search)::text || '%' OR e.actor_name ILIKE '%' || sqlc.narg(search)::text || '%'
       OR e.target_id = sqlc.narg(search)::text OR e.request_id = sqlc.narg(search)::text OR e.metadata ->> 'reference' = sqlc.narg(search)::text)
  AND (sqlc.narg(cursor_at)::timestamptz IS NULL OR e.occurred_at < sqlc.narg(cursor_at)::timestamptz
       OR (e.occurred_at = sqlc.narg(cursor_at)::timestamptz AND e.id < sqlc.narg(cursor_id)::uuid))
ORDER BY e.occurred_at DESC, e.id DESC
LIMIT @lim;

-- name: InstanceAuditGet :one
SELECT e.id, e.workspace_id, e.occurred_at, e.actor_type, e.actor_id, e.actor_name, e.action, e.target_type, e.target_id,
       e.target_name, e.result, e.reason, e.ip, e.user_agent, e.request_id, e.before, e.after, e.metadata,
       coalesce(u.avatar_color, '')::text AS actor_color, coalesce(w.name, '')::text AS workspace_name, coalesce(w.slug, '')::text AS workspace_slug,
       coalesce(w.timezone, '')::text AS workspace_timezone
FROM audit_events e
LEFT JOIN users u ON e.actor_type = 'user' AND u.id = e.actor_id
LEFT JOIN workspaces w ON w.id = e.workspace_id
WHERE e.id = @id;

-- name: InstanceAuditRelated :many
-- Other entries about the same target close in time, for "Related:".
SELECT e.id, e.occurred_at, e.actor_name, e.action, e.metadata
FROM audit_events e
WHERE e.id <> @id AND e.workspace_id IS NOT DISTINCT FROM sqlc.narg(workspace_id)::uuid
  AND ((e.target_id <> '' AND e.target_id = @target_id) OR (e.target_type <> '' AND e.target_type = @target_type AND e.target_name <> '' AND e.target_name = @target_name))
  AND e.occurred_at >= @since AND e.occurred_at <= @until
ORDER BY e.occurred_at DESC
LIMIT 3;

-- name: InstanceAuditCount :one
SELECT count(*) FROM audit_events WHERE workspace_id IS NULL;

-- name: InstanceAuditWorkspaceStreams :many
-- Workspaces that stream their own log, for "Streaming" on the instance view.
SELECT id, slug, name, audit_stream_url, audit_stream_status
FROM workspaces
WHERE deleted_at IS NULL AND audit_stream_enabled AND coalesce(audit_stream_url, '') <> ''
ORDER BY name;

-- name: InstanceAuditStreamState :one
SELECT audit_retention_days, audit_stream_enabled, audit_stream_url, audit_stream_secret_enc, audit_stream_cursor_at, audit_stream_cursor_id, audit_stream_status
FROM instance_settings WHERE id;

-- name: InstanceAuditSetRetention :exec
UPDATE instance_settings SET audit_retention_days = @days, updated_at = now() WHERE id;

-- name: InstanceAuditStreamConfigure :exec
UPDATE instance_settings
SET audit_stream_enabled = @enabled, audit_stream_url = @url,
    audit_stream_secret_enc = coalesce(sqlc.narg(secret_enc), audit_stream_secret_enc),
    audit_stream_cursor_at = CASE WHEN @reset_cursor::boolean THEN now() ELSE audit_stream_cursor_at END,
    audit_stream_cursor_id = CASE WHEN @reset_cursor::boolean THEN NULL ELSE audit_stream_cursor_id END,
    audit_stream_status = CASE WHEN @reset_cursor::boolean THEN '{}'::jsonb ELSE audit_stream_status END,
    updated_at = now()
WHERE id;

-- name: InstanceAuditStreamBatch :many
SELECT id, workspace_id, occurred_at, actor_type, actor_id, actor_name, action, target_type, target_id,
       target_name, result, reason, ip, user_agent, request_id, before, after, metadata
FROM audit_events
WHERE workspace_id IS NULL
  AND (sqlc.narg(cursor_at)::timestamptz IS NULL OR occurred_at > sqlc.narg(cursor_at)::timestamptz
       OR (occurred_at = sqlc.narg(cursor_at)::timestamptz AND sqlc.narg(cursor_id)::uuid IS NOT NULL AND id > sqlc.narg(cursor_id)::uuid))
ORDER BY occurred_at, id
LIMIT @lim;

-- name: InstanceAuditStreamAdvance :exec
UPDATE instance_settings SET audit_stream_cursor_at = @cursor_at, audit_stream_cursor_id = @cursor_id, audit_stream_status = @status WHERE id;

-- name: InstanceAuditStreamSetStatus :exec
UPDATE instance_settings SET audit_stream_status = @status WHERE id;

-- name: PurgeInstanceAuditEvents :execrows
-- Retention of the instance-level entries (inside the purge transaction).
DELETE FROM audit_events WHERE workspace_id IS NULL AND occurred_at < @before;
