-- API keys and MCP agents. Both live in api_keys (kind api_key | mcp_agent).

-- name: KeysList :many
SELECT k.id, k.workspace_id, k.kind, k.name, k.description, k.prefix, k.permissions, k.scope, k.created_by, k.created_at,
       k.expires_at, k.last_used_at, k.last_used_ip, k.last_used_client, k.last_call, k.revoked_at, k.rotated_from,
       coalesce(u.name, '')::text AS created_by_name, coalesce(u.avatar_color, '')::text AS created_by_color
FROM api_keys k
LEFT JOIN users u ON u.id = k.created_by
WHERE k.workspace_id = @workspace_id AND k.kind = @kind
  AND (k.revoked_at IS NULL)
  AND (k.expires_at IS NULL OR k.expires_at > now() - interval '90 days')
  -- Integrations' keys are listed apart ("Managed by integrations").
  AND k.integration_id IS NULL
ORDER BY (k.expires_at IS NOT NULL AND k.expires_at <= now()), coalesce(k.last_used_at, k.created_at) DESC, k.created_at DESC;

-- name: KeysGet :one
SELECT k.id, k.workspace_id, k.kind, k.name, k.description, k.prefix, k.permissions, k.scope, k.created_by, k.created_at,
       k.expires_at, k.last_used_at, k.last_used_ip, k.last_used_client, k.last_call, k.revoked_at, k.rotated_from,
       coalesce(u.name, '')::text AS created_by_name, coalesce(u.avatar_color, '')::text AS created_by_color
FROM api_keys k
LEFT JOIN users u ON u.id = k.created_by
WHERE k.id = @id AND k.workspace_id = @workspace_id AND k.kind = @kind;

-- name: KeysCreate :one
INSERT INTO api_keys (id, workspace_id, kind, name, description, prefix, token_hash, permissions, scope, created_by, expires_at, rotated_from)
VALUES (@id, @workspace_id, @kind, @name, @description, @prefix, @token_hash, @permissions, @scope, @created_by, @expires_at, @rotated_from)
RETURNING id, created_at;

-- name: KeysRevoke :execrows
UPDATE api_keys SET revoked_at = now() WHERE id = @id AND workspace_id = @workspace_id AND kind = @kind AND revoked_at IS NULL;

-- name: KeysDelete :execrows
-- Only expired or revoked keys can be deleted (the audit log keeps their history).
DELETE FROM api_keys WHERE id = @id AND workspace_id = @workspace_id AND kind = @kind
  AND (revoked_at IS NOT NULL OR (expires_at IS NOT NULL AND expires_at <= now()));

-- name: KeysRejections :one
-- Calls rejected in the last 7 days, with the busiest client address.
SELECT coalesce(sum(calls), 0)::bigint AS calls,
       min(first_at)::timestamptz AS since,
       coalesce((SELECT r2.ip FROM api_key_rejections r2 WHERE r2.api_key_id = @api_key_id AND r2.day >= (now() - interval '7 days')::date
                 GROUP BY r2.ip ORDER BY sum(r2.calls) DESC LIMIT 1), '')::text AS top_ip,
       coalesce((SELECT r3.client FROM api_key_rejections r3 WHERE r3.api_key_id = @api_key_id AND r3.day >= (now() - interval '7 days')::date
                 GROUP BY r3.client ORDER BY sum(r3.calls) DESC LIMIT 1), '')::text AS top_client
FROM api_key_rejections
WHERE api_key_id = @api_key_id AND day >= (now() - interval '7 days')::date;

-- name: KeysFirstRejection :one
SELECT coalesce(min(first_at), now())::timestamptz FROM api_key_rejections WHERE api_key_id = @api_key_id;

-- name: KeysRecordRejection :exec
INSERT INTO api_key_rejections (api_key_id, day, ip, client, reason, calls, first_at, last_at)
VALUES (@api_key_id, current_date, @ip, @client, @reason, 1, now(), now())
ON CONFLICT (api_key_id, day, ip) DO UPDATE SET calls = api_key_rejections.calls + 1, last_at = now(),
    client = CASE WHEN excluded.client <> '' THEN excluded.client ELSE api_key_rejections.client END;

-- name: KeysTouch :exec
UPDATE api_keys SET last_used_at = now(), last_used_ip = @ip, last_used_client = @client, last_call = @last_call
WHERE id = @id;

-- name: KeysByToken :one
SELECT k.id, k.workspace_id, k.kind, k.name, k.prefix, k.permissions, k.scope, k.created_by, k.expires_at, k.revoked_at,
       w.slug AS workspace_slug, w.name AS workspace_name, w.mcp_settings
FROM api_keys k JOIN workspaces w ON w.id = k.workspace_id AND w.deleted_at IS NULL
WHERE k.token_hash = @token_hash;

-- name: KeysAgentActions :many
SELECT e.id, e.occurred_at, e.actor_id, e.actor_name, e.action, e.target_type, e.target_id, e.target_name, e.result, e.reason, e.metadata
FROM audit_events e
WHERE e.workspace_id = @workspace_id AND e.actor_type = 'mcp_agent'
ORDER BY e.occurred_at DESC, e.id DESC
LIMIT @lim;

-- name: KeysCountActive :one
SELECT count(*) FILTER (WHERE kind = 'api_key')::bigint AS api_keys, count(*) FILTER (WHERE kind = 'mcp_agent')::bigint AS agents
FROM api_keys WHERE workspace_id = @workspace_id AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > now())
  AND integration_id IS NULL;
