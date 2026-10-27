-- Workspace sending rules and audience defaults (SR-10, Settings ›
-- Sending rules).

-- name: SendingRulesList :many
SELECT sqlc.embed(r), coalesce(u.name, '')::text AS created_by_name
FROM sending_rules r
LEFT JOIN users u ON u.id = r.created_by
WHERE r.workspace_id = @workspace_id
ORDER BY r.created_at, r.id;

-- name: SendingRuleGet :one
SELECT * FROM sending_rules WHERE workspace_id = @workspace_id AND id = @id;

-- name: SendingRuleInsert :exec
INSERT INTO sending_rules (id, workspace_id, name, kind, operator, value, severity, applies_to, enabled, message, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @name, @kind, @operator, @value, @severity, @applies_to, @enabled, @message, sqlc.narg(created_by), @now, @now);

-- name: SendingRuleUpdate :execrows
UPDATE sending_rules SET name = @name, kind = @kind, operator = @operator, value = @value, severity = @severity, applies_to = @applies_to,
    enabled = @enabled, message = @message, updated_at = @now
WHERE workspace_id = @workspace_id AND id = @id;

-- name: SendingRuleDelete :execrows
DELETE FROM sending_rules WHERE workspace_id = @workspace_id AND id = @id;

-- name: ReadinessSendingRules :many
-- The rules a send checks: on, and for its stream (or both).
SELECT sqlc.embed(r), coalesce(u.name, '')::text AS created_by_name
FROM sending_rules r
LEFT JOIN users u ON u.id = r.created_by
WHERE r.workspace_id = @workspace_id AND r.enabled AND r.applies_to IN (@stream::text, 'both')
ORDER BY r.created_at, r.id;

-- name: SendingRuleTriggered :exec
-- A run found the send breaking the rule.
UPDATE sending_rules SET triggered_at = @at, triggered_campaign = @campaign_id, triggered_name = @campaign_name, triggered_count = @count, triggered_fixed = false
WHERE id = @id;

-- name: SendingRuleFixed :exec
-- A later run of the send it last found breaking the rule found it kept.
UPDATE sending_rules SET triggered_fixed = true
WHERE id = @id AND triggered_campaign = @campaign_id AND NOT triggered_fixed;

-- name: SendDefaults :one
-- A workspace's audience defaults (the defaults when it never saved any).
SELECT coalesce(d.exclude_unengaged, true)::bool AS exclude_unengaged,
       coalesce(d.typed_multiple, 1.5)::float8 AS typed_multiple,
       d.updated_at, coalesce(u.name, '')::text AS updated_by_name
FROM workspaces w
LEFT JOIN workspace_send_defaults d ON d.workspace_id = w.id
LEFT JOIN users u ON u.id = d.updated_by
WHERE w.id = @workspace_id;

-- name: SendDefaultsSave :exec
INSERT INTO workspace_send_defaults (workspace_id, exclude_unengaged, typed_multiple, updated_at, updated_by)
VALUES (@workspace_id, @exclude_unengaged, @typed_multiple::float8, @now, sqlc.narg(updated_by))
ON CONFLICT (workspace_id) DO UPDATE SET exclude_unengaged = EXCLUDED.exclude_unengaged,
    typed_multiple = EXCLUDED.typed_multiple, updated_at = EXCLUDED.updated_at, updated_by = EXCLUDED.updated_by;
