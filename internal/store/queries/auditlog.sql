-- Workspace audit log views, export and stream (W03). The instance-level
-- view lives with the instance admin domain.

-- name: AuditLogList :many
SELECT e.id, e.workspace_id, e.occurred_at, e.actor_type, e.actor_id, e.actor_name, e.action, e.target_type, e.target_id,
       e.target_name, e.result, e.reason, e.ip, e.user_agent, e.request_id, e.metadata,
       coalesce(u.avatar_color, '')::text AS actor_color, coalesce(r.name, '')::text AS actor_role,
       (e.before IS NOT NULL AND e.after IS NOT NULL)::boolean AS has_changes
FROM audit_events e
LEFT JOIN users u ON e.actor_type = 'user' AND u.id = e.actor_id
LEFT JOIN workspace_members m ON e.actor_type = 'user' AND m.workspace_id = e.workspace_id AND m.user_id = e.actor_id
LEFT JOIN roles r ON r.id = m.role_id
WHERE e.workspace_id = @workspace_id
  AND (sqlc.narg(actor_type)::text IS NULL OR e.actor_type = sqlc.narg(actor_type))
  AND (sqlc.narg(actor_id)::uuid IS NULL OR e.actor_id = sqlc.narg(actor_id))
  AND (sqlc.narg(actions)::text[] IS NULL OR e.action = ANY(sqlc.narg(actions)::text[])
       OR split_part(e.action, '.', 1) = ANY(sqlc.narg(actions)::text[]))
  AND (sqlc.narg(result)::text IS NULL OR e.result = sqlc.narg(result))
  AND (sqlc.narg(since)::timestamptz IS NULL OR e.occurred_at >= sqlc.narg(since))
  AND (sqlc.narg(until)::timestamptz IS NULL OR e.occurred_at < sqlc.narg(until))
  AND (sqlc.narg(search)::text IS NULL OR e.target_name ILIKE '%' || sqlc.narg(search)::text || '%' OR e.target_id = sqlc.narg(search)::text
       OR e.request_id = sqlc.narg(search)::text OR e.actor_name ILIKE '%' || sqlc.narg(search)::text || '%')
  AND (sqlc.narg(cursor_at)::timestamptz IS NULL OR e.occurred_at < sqlc.narg(cursor_at)::timestamptz
       OR (e.occurred_at = sqlc.narg(cursor_at)::timestamptz AND e.id < sqlc.narg(cursor_id)::uuid))
ORDER BY e.occurred_at DESC, e.id DESC
LIMIT @lim;

-- name: AuditLogExport :many
-- Same filters as AuditLogList, with before/after for exports.
SELECT e.id, e.workspace_id, e.occurred_at, e.actor_type, e.actor_id, e.actor_name, e.action, e.target_type, e.target_id,
       e.target_name, e.result, e.reason, e.ip, e.user_agent, e.request_id, e.before, e.after, e.metadata,
       coalesce(u.avatar_color, '')::text AS actor_color, coalesce(r.name, '')::text AS actor_role
FROM audit_events e
LEFT JOIN users u ON e.actor_type = 'user' AND u.id = e.actor_id
LEFT JOIN workspace_members m ON e.actor_type = 'user' AND m.workspace_id = e.workspace_id AND m.user_id = e.actor_id
LEFT JOIN roles r ON r.id = m.role_id
WHERE e.workspace_id = @workspace_id
  AND (sqlc.narg(actor_type)::text IS NULL OR e.actor_type = sqlc.narg(actor_type))
  AND (sqlc.narg(actor_id)::uuid IS NULL OR e.actor_id = sqlc.narg(actor_id))
  AND (sqlc.narg(actions)::text[] IS NULL OR e.action = ANY(sqlc.narg(actions)::text[])
       OR split_part(e.action, '.', 1) = ANY(sqlc.narg(actions)::text[]))
  AND (sqlc.narg(result)::text IS NULL OR e.result = sqlc.narg(result))
  AND (sqlc.narg(since)::timestamptz IS NULL OR e.occurred_at >= sqlc.narg(since))
  AND (sqlc.narg(until)::timestamptz IS NULL OR e.occurred_at < sqlc.narg(until))
  AND (sqlc.narg(search)::text IS NULL OR e.target_name ILIKE '%' || sqlc.narg(search)::text || '%' OR e.target_id = sqlc.narg(search)::text
       OR e.request_id = sqlc.narg(search)::text OR e.actor_name ILIKE '%' || sqlc.narg(search)::text || '%')
  AND (sqlc.narg(cursor_at)::timestamptz IS NULL OR e.occurred_at < sqlc.narg(cursor_at)::timestamptz
       OR (e.occurred_at = sqlc.narg(cursor_at)::timestamptz AND e.id < sqlc.narg(cursor_id)::uuid))
ORDER BY e.occurred_at DESC, e.id DESC
LIMIT @lim;

-- name: AuditLogGet :one
SELECT e.id, e.workspace_id, e.occurred_at, e.actor_type, e.actor_id, e.actor_name, e.action, e.target_type, e.target_id,
       e.target_name, e.result, e.reason, e.ip, e.user_agent, e.request_id, e.before, e.after, e.metadata,
       coalesce(u.avatar_color, '')::text AS actor_color, coalesce(r.name, '')::text AS actor_role,
       coalesce(w.name, '')::text AS workspace_name
FROM audit_events e
LEFT JOIN users u ON e.actor_type = 'user' AND u.id = e.actor_id
LEFT JOIN workspace_members m ON e.actor_type = 'user' AND m.workspace_id = e.workspace_id AND m.user_id = e.actor_id
LEFT JOIN roles r ON r.id = m.role_id
LEFT JOIN workspaces w ON w.id = e.workspace_id
WHERE e.id = @id AND e.workspace_id = @workspace_id;

-- name: AuditLogCount :one
SELECT count(*) FROM audit_events WHERE workspace_id = @workspace_id;

-- name: AuditLogActors :many
-- Actors seen in the workspace log, for the actor filter.
SELECT e.actor_type, e.actor_id, max(e.actor_name)::text AS actor_name, count(*)::bigint AS n
FROM audit_events e
WHERE e.workspace_id = @workspace_id AND e.occurred_at > now() - interval '365 days'
GROUP BY e.actor_type, e.actor_id
ORDER BY n DESC
LIMIT 50;

-- name: AuditLogActionGroups :many
SELECT split_part(action, '.', 1)::text AS grp, count(*)::bigint AS n
FROM audit_events
WHERE workspace_id = @workspace_id AND occurred_at > now() - interval '365 days'
GROUP BY 1 ORDER BY 1;

-- name: AuditLogRelated :many
-- Other entries about the same target close in time, for "Related:".
SELECT e.id, e.occurred_at, e.actor_name, e.action, e.target_name, e.metadata
FROM audit_events e
WHERE e.workspace_id = @workspace_id AND e.id <> @id
  AND ((e.target_id <> '' AND e.target_id = @target_id) OR (e.target_type <> '' AND e.target_type = @target_type AND e.target_name <> '' AND e.target_name = @target_name))
  AND e.occurred_at >= @since AND e.occurred_at <= @until
ORDER BY e.occurred_at DESC
LIMIT 3;

-- name: AuditStreamWorkspaces :many
SELECT id, slug, audit_stream_url, audit_stream_secret_enc, audit_stream_cursor_at, audit_stream_cursor_id, audit_stream_status
FROM workspaces
WHERE deleted_at IS NULL AND audit_stream_enabled AND coalesce(audit_stream_url, '') <> '';

-- name: AuditStreamBatch :many
SELECT id, workspace_id, occurred_at, actor_type, actor_id, actor_name, action, target_type, target_id,
       target_name, result, reason, ip, user_agent, request_id, before, after, metadata
FROM audit_events
WHERE workspace_id = @workspace_id
  AND (sqlc.narg(cursor_at)::timestamptz IS NULL OR occurred_at > sqlc.narg(cursor_at)::timestamptz
       OR (occurred_at = sqlc.narg(cursor_at)::timestamptz AND sqlc.narg(cursor_id)::uuid IS NOT NULL AND id > sqlc.narg(cursor_id)::uuid))
ORDER BY occurred_at, id
LIMIT @lim;

-- name: AuditStreamAdvance :exec
UPDATE workspaces SET audit_stream_cursor_at = @cursor_at, audit_stream_cursor_id = @cursor_id, audit_stream_status = @status
WHERE id = @id;

-- name: AuditStreamSetStatus :exec
UPDATE workspaces SET audit_stream_status = @status WHERE id = @id;

-- name: AuditStreamConfigure :exec
UPDATE workspaces
SET audit_stream_enabled = @enabled, audit_stream_url = @url,
    audit_stream_secret_enc = coalesce(sqlc.narg(secret_enc), audit_stream_secret_enc),
    audit_stream_cursor_at = CASE WHEN @reset_cursor::boolean THEN now() ELSE audit_stream_cursor_at END,
    audit_stream_cursor_id = CASE WHEN @reset_cursor::boolean THEN NULL ELSE audit_stream_cursor_id END,
    audit_stream_status = CASE WHEN @reset_cursor::boolean THEN '{}'::jsonb ELSE audit_stream_status END,
    updated_at = now()
WHERE id = @id;

-- name: AuditSetRetention :exec
UPDATE workspaces SET audit_retention_days = @days, updated_at = now() WHERE id = @id;
