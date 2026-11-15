-- name: MCPConnectCreateClient :one
INSERT INTO mcp_connect_clients (id, name, redirect_uris, auth_method, secret_hash, verified, expires_at)
VALUES (@id, @name, @redirect_uris, @auth_method, @secret_hash, @verified, @expires_at) RETURNING *;

-- name: MCPConnectClient :one
SELECT * FROM mcp_connect_clients WHERE id = @id;

-- name: MCPConnectClientInUse :one
SELECT EXISTS (SELECT 1 FROM mcp_connect_grants WHERE client_id = @client_id AND expires_at > @now AND revoked_at IS NULL);

-- name: MCPConnectCreateRequest :exec
INSERT INTO mcp_connect_requests (handle_hash, client_id, redirect_uri, resource, state, challenge, scopes, expires_at)
VALUES (@handle_hash, @client_id, @redirect_uri, @resource, @state, @challenge, @scopes, @expires_at);

-- name: MCPConnectRequestLock :one
SELECT * FROM mcp_connect_requests WHERE handle_hash = @handle_hash FOR UPDATE;

-- name: MCPConnectBindRequest :exec
UPDATE mcp_connect_requests SET user_id = @user_id, session_id = @session_id WHERE handle_hash = @handle_hash;

-- name: MCPConnectDecideRequest :exec
UPDATE mcp_connect_requests SET status = @status WHERE handle_hash = @handle_hash;

-- name: MCPConnectCreateGrant :one
INSERT INTO mcp_connect_grants (id, client_id, workspace_id, user_id, resource, scopes, permissions, scope, allowed_ips, created_at, expires_at)
VALUES (@id, @client_id, @workspace_id, @user_id, @resource, @scopes, @permissions, @scope, @allowed_ips, @created_at, @expires_at) RETURNING *;

-- name: MCPConnectGrantLock :one
SELECT * FROM mcp_connect_grants WHERE id = @id FOR UPDATE;

-- name: MCPConnectGrantRead :one
SELECT * FROM mcp_connect_grants WHERE id = @id FOR SHARE;

-- name: MCPConnectRevoke :exec
UPDATE mcp_connect_grants SET revoked_at = @now, revoked_reason = @reason WHERE id = @id AND revoked_at IS NULL;

-- name: MCPConnectGrants :many
SELECT g.*, c.name AS client_name, c.verified, u.name AS user_name
FROM mcp_connect_grants g JOIN mcp_connect_clients c ON c.id = g.client_id JOIN users u ON u.id = g.user_id
WHERE g.workspace_id = @workspace_id AND (@manager::boolean OR g.user_id = @user_id)
AND (sqlc.narg(before_id)::uuid IS NULL OR g.id < sqlc.narg(before_id)::uuid)
ORDER BY g.id DESC LIMIT 51;

-- name: MCPConnectConnection :one
SELECT g.*, c.name AS client_name, c.verified, u.name AS user_name
FROM mcp_connect_grants g JOIN mcp_connect_clients c ON c.id = g.client_id JOIN users u ON u.id = g.user_id
WHERE g.id = @id AND g.workspace_id = @workspace_id AND (@manager::boolean OR g.user_id = @user_id);

-- name: MCPConnectCreateCode :exec
INSERT INTO mcp_connect_codes (hash, grant_id, redirect_uri, challenge, expires_at)
VALUES (@hash, @grant_id, @redirect_uri, @challenge, @expires_at);

-- name: MCPConnectCode :one
SELECT * FROM mcp_connect_codes WHERE hash = @hash;

-- name: MCPConnectUseCode :one
UPDATE mcp_connect_codes SET used_at = @now WHERE hash = @hash AND used_at IS NULL AND expires_at > @now RETURNING *;

-- name: MCPConnectCreateToken :exec
INSERT INTO mcp_connect_tokens (hash, grant_id, kind, expires_at) VALUES (@hash, @grant_id, @kind, @expires_at);

-- name: MCPConnectToken :one
SELECT * FROM mcp_connect_tokens WHERE hash = @hash;

-- name: MCPConnectUseRefresh :one
UPDATE mcp_connect_tokens SET used_at = @now WHERE hash = @hash AND kind = 'refresh' AND used_at IS NULL AND expires_at > @now RETURNING *;

-- name: MCPConnectAuthority :one
SELECT m.scope, m.status AS member_status, r.key AS role_key, r.builtin, r.permissions,
       u.status AS user_status, u.must_change_password, w.deleted_at, w.mcp_settings
FROM workspace_members m JOIN roles r ON r.id = m.role_id
JOIN users u ON u.id = m.user_id JOIN workspaces w ON w.id = m.workspace_id
WHERE m.workspace_id = @workspace_id AND m.user_id = @user_id
FOR SHARE OF m, r, u, w;

-- name: MCPConnectSessionLock :one
SELECT * FROM sessions WHERE id = @id FOR SHARE;

-- name: MCPConnectPolicy :one
SELECT twofa_enforcement FROM instance_settings WHERE id FOR SHARE;

-- name: MCPConnectClaimSlot :one
INSERT INTO mcp_connect_slots (grant_id, lease_id, expires_at) VALUES (@grant_id, @lease_id, @expires_at)
ON CONFLICT (grant_id) DO UPDATE SET lease_id = EXCLUDED.lease_id, expires_at = EXCLUDED.expires_at
WHERE mcp_connect_slots.expires_at <= @now RETURNING lease_id;

-- name: MCPConnectReleaseSlot :exec
DELETE FROM mcp_connect_slots WHERE grant_id = @grant_id AND lease_id = @lease_id;

-- name: MCPConnectLimit :one
INSERT INTO mcp_connect_limits (key, bucket, count) VALUES (@key, @bucket, 1)
ON CONFLICT (key, bucket) DO UPDATE SET count = mcp_connect_limits.count + 1
WHERE mcp_connect_limits.count < @ceiling::integer RETURNING count;

-- name: MCPConnectPurgeRequests :execrows
DELETE FROM mcp_connect_requests WHERE handle_hash IN
(SELECT r.handle_hash FROM mcp_connect_requests r WHERE r.expires_at <= @now ORDER BY r.expires_at LIMIT 500 FOR UPDATE SKIP LOCKED);

-- name: MCPConnectPurgeCodes :execrows
DELETE FROM mcp_connect_codes WHERE hash IN
(SELECT c.hash FROM mcp_connect_codes c WHERE c.expires_at <= @now ORDER BY c.expires_at LIMIT 500 FOR UPDATE SKIP LOCKED);

-- name: MCPConnectPurgeTokens :execrows
DELETE FROM mcp_connect_tokens WHERE hash IN
(SELECT t.hash FROM mcp_connect_tokens t WHERE t.expires_at <= @now ORDER BY t.expires_at LIMIT 500 FOR UPDATE SKIP LOCKED);

-- name: MCPConnectPurgeLimits :execrows
DELETE FROM mcp_connect_limits WHERE (key, bucket) IN
(SELECT l.key, l.bucket FROM mcp_connect_limits l WHERE l.bucket < @before ORDER BY l.bucket LIMIT 500 FOR UPDATE SKIP LOCKED);

-- name: MCPConnectPurgeClients :execrows
DELETE FROM mcp_connect_clients c WHERE c.id IN
(SELECT id FROM mcp_connect_clients c WHERE c.expires_at <= @now
 AND NOT EXISTS (SELECT 1 FROM mcp_connect_grants g WHERE g.client_id = c.id)
 ORDER BY c.expires_at LIMIT 500 FOR UPDATE SKIP LOCKED);

-- name: MCPConnectPurgeGrants :execrows
DELETE FROM mcp_connect_grants WHERE id IN
(SELECT g.id FROM mcp_connect_grants g WHERE g.expires_at <= @before ORDER BY g.expires_at LIMIT 500 FOR UPDATE SKIP LOCKED);
