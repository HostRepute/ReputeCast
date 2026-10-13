-- Roles & permissions (W03).

-- name: RolesList :many
SELECT r.id, r.workspace_id, r.key, r.name, r.description, r.builtin, r.permissions, r.created_by, r.created_at, r.updated_at,
       coalesce(u.name, '')::text AS created_by_name
FROM roles r
LEFT JOIN users u ON u.id = r.created_by
WHERE r.workspace_id = @workspace_id
ORDER BY r.builtin DESC, CASE r.key WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 WHEN 'editor' THEN 2 WHEN 'viewer' THEN 3 ELSE 4 END, r.created_at;

-- name: RolesGet :one
SELECT r.id, r.workspace_id, r.key, r.name, r.description, r.builtin, r.permissions, r.created_by, r.created_at, r.updated_at,
       coalesce(u.name, '')::text AS created_by_name
FROM roles r
LEFT JOIN users u ON u.id = r.created_by
WHERE r.id = @id AND r.workspace_id = @workspace_id;

-- name: RolesMembers :many
-- Members of every role in the workspace (member counts and impact previews).
SELECT m.id AS member_id, m.role_id, m.status, m.user_id, u.name, u.email, u.avatar_color,
       u.totp_enabled_at, (SELECT count(*) FROM user_passkeys p WHERE p.user_id = u.id)::bigint AS passkey_count,
       (SELECT count(*) FROM user_identities i WHERE i.user_id = u.id AND i.kind = 'sso')::bigint AS sso_identities
FROM workspace_members m
JOIN users u ON u.id = m.user_id
WHERE m.workspace_id = @workspace_id
ORDER BY m.status = 'deactivated', u.name;

-- name: RolesPendingInvites :many
SELECT i.role_id, count(*)::bigint AS n
FROM invitations i
WHERE i.workspace_id = @workspace_id AND i.accepted_at IS NULL AND i.revoked_at IS NULL AND i.expires_at > now()
GROUP BY i.role_id;

-- name: RolesNameTaken :one
SELECT EXISTS (
    SELECT 1 FROM roles WHERE workspace_id = @workspace_id AND lower(name) = lower(@name::text) AND id <> @exclude_id
);

-- name: RolesKeyTaken :one
SELECT EXISTS (SELECT 1 FROM roles WHERE workspace_id = @workspace_id AND key = @key);

-- name: RolesCreate :one
INSERT INTO roles (id, workspace_id, key, name, description, builtin, permissions, created_by)
VALUES (@id, @workspace_id, @key, @name, @description, false, @permissions, @created_by)
RETURNING id;

-- name: RolesUpdate :execrows
UPDATE roles SET name = @name, description = @description, permissions = @permissions, updated_at = now()
WHERE id = @id AND workspace_id = @workspace_id AND NOT builtin;

-- name: RolesDelete :execrows
DELETE FROM roles r WHERE r.id = @id AND r.workspace_id = @workspace_id AND NOT r.builtin
  AND NOT EXISTS (SELECT 1 FROM workspace_members m WHERE m.role_id = r.id)
  AND NOT EXISTS (SELECT 1 FROM invitations i WHERE i.role_id = r.id AND i.accepted_at IS NULL AND i.revoked_at IS NULL);

-- name: RolesKeysOfMembers :many
-- Active API keys and MCP agents created by the given users in the workspace.
SELECT k.id, k.kind, k.name, k.permissions, k.created_by
FROM api_keys k
WHERE k.workspace_id = @workspace_id AND k.created_by = ANY(@user_ids::uuid[]) AND k.revoked_at IS NULL
  AND (k.expires_at IS NULL OR k.expires_at > now());
