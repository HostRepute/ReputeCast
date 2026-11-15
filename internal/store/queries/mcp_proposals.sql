-- name: MCPConnectTemplate :one
-- Shared lock fences template edits until the source check/decision commits.
SELECT id, name, mjml FROM templates
WHERE workspace_id = @workspace_id AND id = @id AND kind = 'template' AND deleted_at IS NULL
FOR SHARE;

-- name: MCPConnectSegment :one
SELECT id, name, mode, definition FROM segments
WHERE workspace_id = @workspace_id AND id = @id FOR SHARE;

-- name: MCPConnectSourceInsert :exec
INSERT INTO mcp_connect_sources (handle, grant_id, kind, target_id, base_revision, snapshot, context, projected, created_at, expires_at)
VALUES (@handle, @grant_id, @kind, @target_id, @base_revision, @snapshot, @context, @projected, @created_at, @expires_at);

-- name: MCPConnectSource :one
SELECT * FROM mcp_connect_sources WHERE handle = @handle AND grant_id = @grant_id AND expires_at > @now;

-- name: MCPConnectSources :many
SELECT * FROM mcp_connect_sources WHERE grant_id = @grant_id AND expires_at > @now ORDER BY created_at DESC LIMIT 50;

-- name: MCPConnectSelectedSource :one
SELECT handle, kind, expires_at FROM mcp_connect_sources
WHERE grant_id = @grant_id AND kind = @kind AND target_id = @target_id AND snapshot = @snapshot AND expires_at > @now
ORDER BY created_at DESC LIMIT 1;

-- name: MCPConnectSourceCount :one
SELECT count(*) FROM mcp_connect_sources WHERE grant_id = @grant_id;

-- name: MCPConnectProposalByKey :one
SELECT * FROM mcp_connect_proposals WHERE grant_id = @grant_id AND request_key = @request_key;

-- name: MCPConnectProposalCount :one
SELECT count(*) FROM mcp_connect_proposals WHERE grant_id = @grant_id;

-- name: MCPConnectProposalInsert :one
INSERT INTO mcp_connect_proposals (id, grant_id, workspace_id, user_id, request_key, request_hash, kind, target_id, base_revision, context, content, preview, created_at, expires_at, purge_at)
VALUES (@id, @grant_id, @workspace_id, @user_id, @request_key, @request_hash, @kind, sqlc.narg(target_id), @base_revision, @context, @content, @preview, @created_at, @expires_at, @purge_at)
RETURNING *;

-- name: MCPConnectProposal :one
SELECT * FROM mcp_connect_proposals WHERE id = @id AND workspace_id = @workspace_id;

-- name: MCPConnectProposalLock :one
SELECT * FROM mcp_connect_proposals WHERE id = @id AND workspace_id = @workspace_id FOR UPDATE;

-- name: MCPConnectProposals :many
SELECT p.id, p.grant_id, p.user_id, p.kind, p.target_id, p.base_revision, p.decision, p.created_at, p.expires_at,
       jsonb_build_object('Source', p.context->'Source')::jsonb AS context,
       coalesce(p.context->'Draft'->>'Name', '')::text AS name,
       p.result_id, p.purge_at, p.purged_at, g.revoked_at, c.name AS client_name, c.verified
FROM mcp_connect_proposals p JOIN mcp_connect_grants g ON g.id = p.grant_id JOIN mcp_connect_clients c ON c.id = g.client_id
WHERE p.workspace_id = @workspace_id AND (@manager::boolean OR p.user_id = @user_id)
AND (sqlc.narg(before_id)::uuid IS NULL OR p.id < sqlc.narg(before_id)::uuid)
ORDER BY p.id DESC LIMIT 51;

-- name: MCPConnectProposalDecide :exec
UPDATE mcp_connect_proposals SET decision = @decision, decided_by = @decided_by, decided_at = @now,
result_id = sqlc.narg(result_id), result_revision = sqlc.narg(result_revision), purge_at = LEAST(purge_at, @purge_at)
WHERE id = @id;

-- name: MCPConnectPurgeProposalContent :execrows
-- Keep content-free idempotency receipts until the grant leaves, so a late
-- retry cannot reuse a purged key to create another product draft. Keep the
-- source identity for current scope checks, not its content or bindings.
UPDATE mcp_connect_proposals SET context = jsonb_build_object('Source', context->'Source'), content = '{}', preview = '', purged_at = @now
WHERE id IN (SELECT p.id FROM mcp_connect_proposals p WHERE p.purge_at <= @now AND p.purged_at IS NULL ORDER BY p.purge_at LIMIT 500 FOR UPDATE SKIP LOCKED);

-- name: MCPConnectPurgeSources :execrows
DELETE FROM mcp_connect_sources WHERE handle IN
(SELECT s.handle FROM mcp_connect_sources s WHERE s.expires_at <= @now ORDER BY s.expires_at LIMIT 500 FOR UPDATE SKIP LOCKED);
