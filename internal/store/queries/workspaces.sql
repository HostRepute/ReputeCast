-- name: CreateWorkspace :one
INSERT INTO workspaces (id, slug, name, timezone, created_by)
VALUES (@id, @slug, @name, @timezone, @created_by)
RETURNING *;

-- name: GetWorkspace :one
SELECT * FROM workspaces WHERE id = @id AND deleted_at IS NULL;

-- name: GetWorkspaceBySlug :one
SELECT * FROM workspaces WHERE slug = @slug AND deleted_at IS NULL;

-- name: ListWorkspaces :many
SELECT * FROM workspaces WHERE deleted_at IS NULL ORDER BY created_at;

-- name: SlugTaken :one
SELECT EXISTS (SELECT 1 FROM workspaces WHERE slug = @slug AND deleted_at IS NULL);

-- name: UpdateWorkspaceGeneral :one
UPDATE workspaces SET name = @name, timezone = @timezone, default_sender_id = @default_sender_id, updated_at = now()
WHERE id = @id
RETURNING *;

-- name: UpdateWorkspaceApprovalPolicy :one
UPDATE workspaces SET approval_policy = @approval_policy, updated_at = now() WHERE id = @id RETURNING *;

-- name: UpdateWorkspaceJSON :one
UPDATE workspaces
SET sunset_policy = coalesce(sqlc.narg(sunset_policy), sunset_policy),
    branding = coalesce(sqlc.narg(branding), branding),
    send_thresholds = coalesce(sqlc.narg(send_thresholds), send_thresholds),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: SoftDeleteWorkspace :exec
UPDATE workspaces SET deleted_at = now(), slug = slug || '-deleted-' || extract(epoch FROM now())::bigint WHERE id = @id;

-- name: CreateRole :one
INSERT INTO roles (id, workspace_id, key, name, description, builtin, permissions, created_by)
VALUES (@id, @workspace_id, @key, @name, @description, @builtin, @permissions, @created_by)
RETURNING *;

-- name: GetRole :one
SELECT * FROM roles WHERE id = @id;

-- name: GetRoleByKey :one
SELECT * FROM roles WHERE workspace_id = @workspace_id AND key = @key;

-- name: ListRoles :many
SELECT r.*, (SELECT count(*) FROM workspace_members m WHERE m.role_id = r.id AND m.status = 'active') AS member_count
FROM roles r WHERE r.workspace_id = @workspace_id
ORDER BY r.builtin DESC, r.created_at;

-- name: UpdateRole :one
UPDATE roles SET name = @name, description = @description, permissions = @permissions, updated_at = now()
WHERE id = @id AND NOT builtin
RETURNING *;

-- name: DeleteRole :execrows
DELETE FROM roles r WHERE r.id = sqlc.arg(role_id)::uuid AND NOT r.builtin
  AND NOT EXISTS (SELECT 1 FROM workspace_members m WHERE m.role_id = sqlc.arg(role_id)::uuid)
  AND NOT EXISTS (SELECT 1 FROM invitations i WHERE i.role_id = sqlc.arg(role_id)::uuid AND i.accepted_at IS NULL AND i.revoked_at IS NULL);

-- name: CreateMember :one
INSERT INTO workspace_members (id, workspace_id, user_id, role_id, scope, source, invited_by)
VALUES (@id, @workspace_id, @user_id, @role_id, @scope, @source, @invited_by)
RETURNING *;

-- name: GetMembership :one
SELECT m.*, r.key AS role_key, r.name AS role_name, r.builtin AS role_builtin, r.permissions AS role_permissions
FROM workspace_members m JOIN roles r ON r.id = m.role_id
WHERE m.workspace_id = @workspace_id AND m.user_id = @user_id;

-- name: GetMember :one
SELECT m.*, r.key AS role_key, r.name AS role_name, r.builtin AS role_builtin, r.permissions AS role_permissions,
       u.email, u.name AS user_name, u.status AS user_status, u.totp_enabled_at, u.twofa_grace_until,
       u.last_active_at AS user_last_active_at, u.password_hash IS NOT NULL AS has_password
FROM workspace_members m
JOIN roles r ON r.id = m.role_id
JOIN users u ON u.id = m.user_id
WHERE m.id = sqlc.arg(member_id)::uuid AND m.workspace_id = sqlc.arg(workspace_id)::uuid;

-- name: ListMembers :many
SELECT m.*, r.key AS role_key, r.name AS role_name, r.builtin AS role_builtin, r.permissions AS role_permissions,
       u.email, u.name AS user_name, u.status AS user_status, u.totp_enabled_at, u.twofa_grace_until,
       u.last_active_at AS user_last_active_at, u.password_hash IS NOT NULL AS has_password,
       (SELECT count(*) FROM user_passkeys p WHERE p.user_id = u.id) AS passkey_count,
       (SELECT coalesce(array_agg(DISTINCT i.kind), '{}') FROM user_identities i WHERE i.user_id = u.id)::text[] AS identity_kinds
FROM workspace_members m
JOIN roles r ON r.id = m.role_id
JOIN users u ON u.id = m.user_id
WHERE m.workspace_id = @workspace_id
ORDER BY CASE r.key WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 WHEN 'editor' THEN 2 WHEN 'viewer' THEN 3 ELSE 4 END, u.name;

-- name: ListUserWorkspaces :many
SELECT w.*, r.key AS role_key, r.name AS role_name, m.status AS member_status,
       (SELECT count(*) FROM workspace_members m2 WHERE m2.workspace_id = w.id AND m2.status = 'active') AS member_count
FROM workspace_members m
JOIN workspaces w ON w.id = m.workspace_id AND w.deleted_at IS NULL
JOIN roles r ON r.id = m.role_id
WHERE m.user_id = @user_id AND m.status = 'active'
ORDER BY w.created_at;

-- name: UpdateMemberRole :one
UPDATE workspace_members SET role_id = @role_id, scope = @scope WHERE id = @id AND workspace_id = @workspace_id RETURNING *;

-- name: SetMemberStatus :exec
UPDATE workspace_members
SET status = @status, deactivated_at = CASE WHEN @status::text = 'deactivated' THEN now() ELSE NULL END
WHERE id = @id AND workspace_id = @workspace_id;

-- name: DeleteMember :execrows
DELETE FROM workspace_members WHERE id = @id AND workspace_id = @workspace_id;

-- name: CountActiveOwners :one
SELECT count(*) FROM workspace_members m JOIN roles r ON r.id = m.role_id
WHERE m.workspace_id = @workspace_id AND r.key = 'owner' AND m.status = 'active';

-- name: TouchMemberActive :exec
UPDATE workspace_members SET last_active_at = now()
WHERE workspace_id = @workspace_id AND user_id = @user_id AND (last_active_at IS NULL OR last_active_at < now() - interval '1 minute');

-- name: ListMembersWithRoles :many
-- Used to name who can help when a permission is missing.
SELECT u.name, r.key AS role_key, r.builtin, r.permissions
FROM workspace_members m
JOIN roles r ON r.id = m.role_id
JOIN users u ON u.id = m.user_id AND u.status = 'active'
WHERE m.workspace_id = @workspace_id AND m.status = 'active'
ORDER BY CASE r.key WHEN 'admin' THEN 0 WHEN 'owner' THEN 1 ELSE 2 END, u.name;

-- name: CreateInvitation :one
INSERT INTO invitations (id, workspace_id, email, role_id, scope, message, token_hash, invited_by, sent_via, last_sent_at, expires_at)
VALUES (@id, @workspace_id, @email, @role_id, @scope, @message, @token_hash, @invited_by, @sent_via, @last_sent_at, @expires_at)
RETURNING *;

-- name: ListPendingInvitations :many
SELECT i.*, r.name AS role_name, r.key AS role_key, u.name AS invited_by_name
FROM invitations i
JOIN roles r ON r.id = i.role_id
LEFT JOIN users u ON u.id = i.invited_by
WHERE i.workspace_id = @workspace_id AND i.accepted_at IS NULL AND i.revoked_at IS NULL
ORDER BY i.created_at DESC;

-- name: GetInvitationByToken :one
SELECT i.*, r.name AS role_name, r.key AS role_key, w.name AS workspace_name, w.slug AS workspace_slug,
       u.name AS invited_by_name
FROM invitations i
JOIN roles r ON r.id = i.role_id
JOIN workspaces w ON w.id = i.workspace_id
LEFT JOIN users u ON u.id = i.invited_by
WHERE i.token_hash = @token_hash;

-- name: AcceptInvitation :exec
UPDATE invitations SET accepted_at = now(), accepted_by = @accepted_by WHERE id = @id;

-- name: RevokeInvitation :execrows
UPDATE invitations SET revoked_at = now() WHERE id = @id AND workspace_id = @workspace_id AND accepted_at IS NULL;

-- name: RefreshInvitation :one
UPDATE invitations SET token_hash = @token_hash, last_sent_at = now(), expires_at = @expires_at, sent_via = @sent_via
WHERE id = @id AND workspace_id = @workspace_id AND accepted_at IS NULL AND revoked_at IS NULL
RETURNING *;

-- name: CreateAPIKey :one
INSERT INTO api_keys (id, workspace_id, kind, name, description, prefix, token_hash, permissions, scope, created_by, expires_at, rotated_from)
VALUES (@id, @workspace_id, @kind, @name, @description, @prefix, @token_hash, @permissions, @scope, @created_by, @expires_at, @rotated_from)
RETURNING *;

-- name: GetAPIKeyByToken :one
SELECT k.*, w.slug AS workspace_slug
FROM api_keys k JOIN workspaces w ON w.id = k.workspace_id AND w.deleted_at IS NULL
WHERE k.token_hash = @token_hash;

-- name: ListAPIKeys :many
SELECT k.*, u.name AS created_by_name
FROM api_keys k LEFT JOIN users u ON u.id = k.created_by
WHERE k.workspace_id = @workspace_id AND k.kind = @kind
ORDER BY k.revoked_at IS NOT NULL, k.created_at DESC;

-- name: GetAPIKey :one
SELECT * FROM api_keys WHERE id = @id AND workspace_id = @workspace_id;

-- name: TouchAPIKey :exec
UPDATE api_keys SET last_used_at = now(), last_used_ip = @ip
WHERE id = @id AND (last_used_at IS NULL OR last_used_at < now() - interval '1 minute');

-- name: RevokeAPIKey :execrows
UPDATE api_keys SET revoked_at = now() WHERE id = @id AND workspace_id = @workspace_id AND revoked_at IS NULL;

-- name: RevokeAPIKeysCreatedBy :execrows
UPDATE api_keys SET revoked_at = now() WHERE workspace_id = @workspace_id AND created_by = @created_by AND revoked_at IS NULL;
