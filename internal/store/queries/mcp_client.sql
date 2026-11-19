-- B5: remote MCP servers, their tools and members' sign-ins.

-- name: MCPServerInsert :exec
INSERT INTO mcp_servers (id, workspace_id, key, name, url, transport, auth, header_name, header_sealed, client_id, client_secret_sealed, access_roles,
    created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @key, @name, @url, @transport, @auth, @header_name, sqlc.narg(header_sealed), @client_id, sqlc.narg(client_secret_sealed),
    sqlc.narg(access_roles), sqlc.narg(created_by), @now, @now);

-- name: MCPServers :many
SELECT * FROM mcp_servers WHERE workspace_id = @workspace_id ORDER BY name, id;

-- name: MCPServer :one
SELECT * FROM mcp_servers WHERE workspace_id = @workspace_id AND id = @id;

-- name: MCPServerLock :one
SELECT * FROM mcp_servers WHERE workspace_id = @workspace_id AND id = @id FOR UPDATE;

-- name: MCPServerKeys :many
SELECT key FROM mcp_servers WHERE workspace_id = @workspace_id;

-- name: MCPServerSave :exec
UPDATE mcp_servers SET name = @name, enabled = @enabled, access_roles = sqlc.narg(access_roles), header_sealed = sqlc.narg(header_sealed),
    client_id = @client_id, client_secret_sealed = sqlc.narg(client_secret_sealed), client_registered = @client_registered,
    issuer = @issuer, authorization_endpoint = @authorization_endpoint, token_endpoint = @token_endpoint, scopes = @scopes,
    resource = @resource, revocation_endpoint = @revocation_endpoint, revision = revision + 1, updated_at = @now
WHERE workspace_id = @workspace_id AND id = @id;

-- name: MCPServerStatus :exec
UPDATE mcp_servers SET status = @status, status_detail = @status_detail, synced_at = coalesce(sqlc.narg(synced_at), synced_at), updated_at = @now
WHERE id = @id;

-- name: MCPServerRevocationsQueue :execrows
-- A server is deleted: its members' refresh tokens are queued for
-- revocation first, in the delete's transaction (the rows outlive it).
INSERT INTO mcp_token_revocations (server_id, user_id, revocation_endpoint, client_id, client_secret_sealed, refresh_sealed, created_at)
SELECT t.server_id, t.user_id, s.revocation_endpoint, s.client_id, s.client_secret_sealed, t.refresh_sealed, now()
FROM mcp_tokens t JOIN mcp_servers s ON s.id = t.server_id
WHERE s.workspace_id = @workspace_id AND s.id = @id AND t.refresh_sealed IS NOT NULL AND s.revocation_endpoint <> '';

-- name: MCPServerDelete :execrows
DELETE FROM mcp_servers WHERE workspace_id = @workspace_id AND id = @id;

-- name: MCPTools :many
SELECT * FROM mcp_tools WHERE server_id = @server_id ORDER BY name;

-- name: MCPTool :one
SELECT t.* FROM mcp_tools t JOIN mcp_servers s ON s.id = t.server_id WHERE s.workspace_id = @workspace_id AND t.id = @id;

-- name: MCPToolInsert :exec
INSERT INTO mcp_tools (id, server_id, name, model_name, description, input_schema, annotations, hash, first_seen_at)
VALUES (@id, @server_id, @name, @model_name, @description, @input_schema, @annotations, @hash, @now);

-- name: MCPToolSeen :exec
-- A sync saw the tool again: as it is now. A tool whose hash moved from
-- the one decided on is off until a manager decides again.
UPDATE mcp_tools SET description = @description, input_schema = @input_schema, annotations = @annotations, hash = @hash, removed_at = NULL
WHERE id = @id;

-- name: MCPToolGone :exec
UPDATE mcp_tools SET removed_at = @now::timestamptz WHERE id = @id AND removed_at IS NULL;

-- name: MCPToolDecide :exec
-- The tool as reviewed is kept with the decision: a later change shows as
-- a diff against it.
UPDATE mcp_tools SET decision = @decision, approved_hash = @approved_hash, approved_description = description, approved_input_schema = input_schema,
    approved_annotations = annotations, decided_by = sqlc.narg(decided_by), decided_at = @now::timestamptz WHERE id = @id;

-- name: MCPThreadTools :many
-- The tools a thread may offer: on servers its owner chose, enabled, and
-- decided on at their current hash (read-only or writes), still listed.
SELECT t.*, s.name AS server_name, s.key AS server_key, s.auth AS server_auth, s.access_roles AS server_roles
FROM ai_assistant_thread_servers ts
JOIN mcp_servers s ON s.id = ts.server_id
JOIN mcp_tools t ON t.server_id = s.id
WHERE ts.thread_id = @thread_id AND s.workspace_id = @workspace_id AND s.enabled AND t.removed_at IS NULL
  AND t.decision IN ('read_only', 'writes') AND t.approved_hash = t.hash
ORDER BY s.name, s.id, t.name;

-- name: MCPThreadServers :many
SELECT server_id FROM ai_assistant_thread_servers WHERE thread_id = @thread_id ORDER BY server_id;

-- name: MCPThreadServersClear :exec
DELETE FROM ai_assistant_thread_servers WHERE thread_id = @thread_id;

-- name: MCPThreadServerAdd :exec
INSERT INTO ai_assistant_thread_servers (thread_id, server_id) VALUES (@thread_id, @server_id) ON CONFLICT DO NOTHING;

-- name: MCPOAuthStateInsert :exec
INSERT INTO mcp_oauth_states (id, state_hash, server_id, user_id, session_id, verifier_sealed, created_at, expires_at)
VALUES (@id, @state_hash, @server_id, @user_id, @session_id, @verifier_sealed, @now, @expires_at);

-- name: MCPOAuthStateUse :one
-- A state is used once, by the session that started it, within its ten
-- minutes.
UPDATE mcp_oauth_states SET used_at = @now::timestamptz
WHERE state_hash = @state_hash AND user_id = @user_id AND session_id = @session_id AND used_at IS NULL AND expires_at > @now
RETURNING *;

-- name: MCPOAuthStatesPurge :execrows
DELETE FROM mcp_oauth_states WHERE expires_at < @before;

-- name: MCPTokenSave :exec
INSERT INTO mcp_tokens (server_id, user_id, workspace_id, access_sealed, refresh_sealed, expires_at, created_at, updated_at)
VALUES (@server_id, @user_id, @workspace_id, @access_sealed, sqlc.narg(refresh_sealed), sqlc.narg(expires_at), @now, @now)
ON CONFLICT (server_id, user_id) DO UPDATE SET access_sealed = EXCLUDED.access_sealed, refresh_sealed = EXCLUDED.refresh_sealed,
    expires_at = EXCLUDED.expires_at, updated_at = EXCLUDED.updated_at;

-- name: MCPTokenLock :one
SELECT * FROM mcp_tokens WHERE server_id = @server_id AND user_id = @user_id FOR UPDATE;

-- name: MCPToken :one
SELECT * FROM mcp_tokens WHERE server_id = @server_id AND user_id = @user_id;

-- name: MCPTokenDelete :one
-- A person's sign-in to a server goes; its refresh token is queued for
-- revocation when the authorization server has an endpoint for it (data-
-- modifying CTEs run whether or not they're read).
WITH gone AS (
    DELETE FROM mcp_tokens t WHERE t.server_id = @server_id AND t.user_id = @user_id
    RETURNING t.server_id, t.user_id, t.refresh_sealed
), queued AS (
    INSERT INTO mcp_token_revocations (server_id, user_id, revocation_endpoint, client_id, client_secret_sealed, refresh_sealed, created_at)
    SELECT g.server_id, g.user_id, s.revocation_endpoint, s.client_id, s.client_secret_sealed, g.refresh_sealed, now()
    FROM gone g JOIN mcp_servers s ON s.id = g.server_id
    WHERE g.refresh_sealed IS NOT NULL AND s.revocation_endpoint <> ''
    RETURNING 1
)
SELECT count(*) FROM gone;

-- name: MCPTokensRemove :one
-- A person left the workspace or was deactivated (sqlc.narg(workspace_id)
-- NULL: every workspace): their MCP sign-ins go, their refresh tokens
-- queued for revocation as MCPTokenDelete does.
WITH gone AS (
    DELETE FROM mcp_tokens t WHERE t.user_id = @user_id AND (sqlc.narg(workspace_id)::uuid IS NULL OR t.workspace_id = sqlc.narg(workspace_id))
    RETURNING t.server_id, t.user_id, t.refresh_sealed
), queued AS (
    INSERT INTO mcp_token_revocations (server_id, user_id, revocation_endpoint, client_id, client_secret_sealed, refresh_sealed, created_at)
    SELECT g.server_id, g.user_id, s.revocation_endpoint, s.client_id, s.client_secret_sealed, g.refresh_sealed, now()
    FROM gone g JOIN mcp_servers s ON s.id = g.server_id
    WHERE g.refresh_sealed IS NOT NULL AND s.revocation_endpoint <> ''
    RETURNING 1
)
SELECT count(*) FROM gone;

-- name: MCPTokenRevocationsTake :many
-- Up to @lim queued refresh tokens with what revoking them needs, taken
-- off the queue (revocation is best effort: a token is tried once).
WITH taken AS (
    DELETE FROM mcp_token_revocations r WHERE r.id IN (SELECT q.id FROM mcp_token_revocations q ORDER BY q.id LIMIT @lim FOR UPDATE SKIP LOCKED)
    RETURNING r.id, r.server_id, r.user_id, r.refresh_sealed, r.revocation_endpoint, r.client_id, r.client_secret_sealed
)
SELECT taken.id, taken.server_id, taken.user_id, taken.refresh_sealed, taken.revocation_endpoint, taken.client_id, taken.client_secret_sealed
FROM taken ORDER BY taken.id;

-- name: MCPServersDue :many
-- Servers the hourly sync looks at: enabled, synced before but neither
-- synced nor tried since @before (one that fails waits its hour too), with
-- a shared credential or none (an OAuth server is listed as a person: by a
-- manager's sync, and before each call).
SELECT s.id, s.workspace_id FROM mcp_servers s
WHERE s.enabled AND s.auth IN ('none', 'header') AND s.synced_at < @before::timestamptz AND s.updated_at < @before
ORDER BY s.updated_at, s.id LIMIT @lim;

-- name: AIAssistantConfirmationInsert :exec
INSERT INTO ai_assistant_confirmations (id, thread_id, turn_id, step_id, call_index, call_id, server_id, tool_id, server_name, tool_name, model_name, tool_hash,
    arguments_sealed, arguments_digest, done_sealed, created_at)
VALUES (@id, @thread_id, @turn_id, @step_id, @call_index, @call_id, @server_id, @tool_id, @server_name, @tool_name, @model_name, @tool_hash,
    @arguments_sealed, @arguments_digest, @done_sealed, @now);

-- name: AIAssistantConfirmationLock :one
SELECT * FROM ai_assistant_confirmations WHERE thread_id = @thread_id AND id = @id FOR UPDATE;

-- name: AIAssistantConfirmationOpen :one
-- A turn's call waiting for, or carrying, its owner's decision.
SELECT * FROM ai_assistant_confirmations WHERE turn_id = @turn_id AND state IN ('pending', 'allowed', 'denied', 'calling');

-- name: AIAssistantConfirmationDecide :exec
UPDATE ai_assistant_confirmations SET state = @state, decision = @decision, decided_by = sqlc.narg(decided_by), decided_at = @now::timestamptz
WHERE id = @id AND state = 'pending';

-- name: AIAssistantConfirmationSet :exec
UPDATE ai_assistant_confirmations SET state = @state WHERE id = @id;

-- name: AIAssistantConfirmationPending :one
-- A thread's call waiting for its owner, as the thread shows it.
SELECT c.* FROM ai_assistant_confirmations c JOIN ai_assistant_turns t ON t.id = c.turn_id
WHERE c.thread_id = @thread_id AND c.state = 'pending' AND t.state = 'waiting'
ORDER BY c.created_at DESC LIMIT 1;

-- name: AIAssistantConfirmationsClose :exec
-- Stop: a turn's open call is closed without running.
UPDATE ai_assistant_confirmations SET state = 'done' WHERE turn_id = @turn_id AND state IN ('pending', 'allowed', 'denied', 'calling');

-- name: AIAssistantConfirmationsPurge :exec
DELETE FROM ai_assistant_confirmations WHERE thread_id = @thread_id;

-- name: AIAssistantTurnRequeue :exec
-- A waiting turn goes on once its owner decided: a new version for its job.
UPDATE ai_assistant_turns SET state = 'queued', version = @version, queued_at = @now::timestamptz WHERE id = @id AND state = 'waiting';
