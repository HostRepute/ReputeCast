-- name: InsertAuditEvent :exec
INSERT INTO audit_events (id, workspace_id, occurred_at, actor_type, actor_id, actor_name, action,
                          target_type, target_id, target_name, result, reason, ip, user_agent,
                          request_id, before, after, metadata)
VALUES (@id, @workspace_id, @occurred_at, @actor_type, @actor_id, @actor_name, @action,
        @target_type, @target_id, @target_name, @result, @reason, @ip, @user_agent,
        @request_id, @before, @after, @metadata);

-- name: ListAuditEvents :many
SELECT * FROM audit_events
WHERE (sqlc.narg(workspace_id)::uuid IS NULL OR workspace_id = sqlc.narg(workspace_id))
  AND (sqlc.narg(actor_type)::text IS NULL OR actor_type = sqlc.narg(actor_type))
  AND (sqlc.narg(actor_id)::uuid IS NULL OR actor_id = sqlc.narg(actor_id))
  AND (sqlc.narg(action_prefix)::text IS NULL OR action LIKE sqlc.narg(action_prefix) || '%')
  AND (sqlc.narg(target_type)::text IS NULL OR target_type = sqlc.narg(target_type))
  AND (sqlc.narg(result)::text IS NULL OR result = sqlc.narg(result))
  AND (sqlc.narg(since)::timestamptz IS NULL OR occurred_at >= sqlc.narg(since))
  AND (sqlc.narg(until)::timestamptz IS NULL OR occurred_at < sqlc.narg(until))
  AND (sqlc.narg(before_id)::uuid IS NULL OR id < sqlc.narg(before_id))
ORDER BY id DESC
LIMIT @lim;

-- name: GetAuditEvent :one
SELECT * FROM audit_events WHERE id = @id;

-- name: EnableAuditPurge :exec
-- Must run inside the same transaction as PurgeAuditEvents (retention job only).
SELECT set_config('reputecast.audit_purge', 'on', true);

-- name: PurgeAuditEvents :execrows
DELETE FROM audit_events
WHERE occurred_at < @before AND (sqlc.narg(workspace_id)::uuid IS NULL OR workspace_id = sqlc.narg(workspace_id));
