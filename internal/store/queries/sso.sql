-- SSO providers, social sign-in and the identities they link (W14).

-- name: SSOProviders :many
SELECT * FROM sso_providers ORDER BY created_at, id;

-- name: SSOProvider :one
SELECT * FROM sso_providers WHERE id = @id;

-- name: SSOProviderBySlug :one
SELECT * FROM sso_providers WHERE slug = @slug;

-- name: SSOCreateProvider :one
INSERT INTO sso_providers (id, preset, name, issuer_url, scopes, claim_map, allowed_domains, jit_enabled,
                           jit_workspace_id, jit_role_key, created_by)
VALUES (@id, @preset, @name, @issuer_url, @scopes, @claim_map, @allowed_domains, @jit_enabled,
        @jit_workspace_id, @jit_role_key, @created_by)
RETURNING *;

-- name: SSOUpdateProvider :one
UPDATE sso_providers
SET name = @name, issuer_url = @issuer_url, client_id = @client_id, scopes = @scopes, claim_map = @claim_map,
    allowed_domains = @allowed_domains, jit_enabled = @jit_enabled, jit_workspace_id = @jit_workspace_id,
    jit_role_key = @jit_role_key, group_mappings = @group_mappings, twofa_mode = sqlc.narg(twofa_mode),
    discovery = CASE WHEN issuer_url = @issuer_url THEN discovery ELSE NULL END,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: SSOSetClientSecret :exec
UPDATE sso_providers
SET client_secret_enc = @client_secret_enc, client_secret_set_at = now(), client_secret_set_by = @set_by,
    client_secret_expires_on = sqlc.narg(expires_on), updated_at = now()
WHERE id = @id;

-- name: SSOSetDiscovery :exec
UPDATE sso_providers SET discovery = @discovery, issuer_url = @issuer_url, updated_at = now() WHERE id = @id;

-- name: SSOSetLastTest :exec
UPDATE sso_providers SET last_test = @last_test WHERE id = @id;

-- name: SSOSetStatus :one
UPDATE sso_providers
SET status = @status,
    enabled_at = CASE WHEN @status::text = 'enabled' THEN coalesce(enabled_at, now()) ELSE enabled_at END,
    enforce_domains = CASE WHEN @status::text = 'enabled' THEN enforce_domains ELSE '{}'::text[] END,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: SSOSetEnforcement :exec
UPDATE sso_providers SET enforce_domains = @enforce_domains, updated_at = now() WHERE id = @id;

-- name: SSODeleteProvider :execrows
DELETE FROM sso_providers WHERE id = @id;

-- name: SSODomainEnforcedElsewhere :one
-- Another enabled provider already enforces one of these domains.
SELECT coalesce((SELECT name FROM sso_providers
                 WHERE id <> @id AND status = 'enabled' AND enforce_domains && @domains::text[] LIMIT 1), '')::text;

-- name: SSOIdentityUsers :many
-- The people who sign in with this provider (for the enforcement preview).
SELECT i.user_id, i.last_used_at FROM user_identities i WHERE i.kind = 'sso' AND i.sso_provider_id = @provider_id;

-- name: SSODomainUsers :many
-- Active accounts with an email at one of the domains, with what they sign in with today.
SELECT u.id, u.email, u.name, u.avatar_color, u.avatar_updated_at, u.break_glass, (u.password_hash IS NOT NULL)::boolean AS has_password,
       (u.totp_enabled_at IS NOT NULL)::boolean AS has_totp,
       (SELECT count(*) FROM user_passkeys p WHERE p.user_id = u.id)::bigint AS passkeys,
       coalesce((SELECT array_agg(DISTINCT i.kind) FROM user_identities i WHERE i.user_id = u.id AND i.kind <> 'sso'), '{}')::text[] AS social,
       EXISTS (SELECT 1 FROM user_identities i WHERE i.user_id = u.id AND i.kind = 'sso' AND i.sso_provider_id = @provider_id)::boolean AS linked
FROM users u
WHERE u.status = 'active' AND split_part(u.email, '@', 2) = ANY(@domains::text[])
ORDER BY u.created_at, u.id;

-- name: SSOOwners :many
-- Active Owners of live workspaces, oldest membership first.
SELECT DISTINCT ON (u.id) u.id, u.email, u.name, u.break_glass, (u.password_hash IS NOT NULL)::boolean AS has_password,
       (u.totp_enabled_at IS NOT NULL OR EXISTS (SELECT 1 FROM user_passkeys p WHERE p.user_id = u.id))::boolean AS has_second_factor,
       w.name AS workspace_name, m.joined_at
FROM workspace_members m
JOIN roles r ON r.id = m.role_id AND r.key = 'owner' AND r.builtin
JOIN workspaces w ON w.id = m.workspace_id AND w.deleted_at IS NULL
JOIN users u ON u.id = m.user_id AND u.status = 'active'
WHERE m.status = 'active'
ORDER BY u.id, w.created_at;

-- name: SSOMarkBreakGlass :execrows
UPDATE users SET break_glass = true, updated_at = now() WHERE id = ANY(@ids::uuid[]) AND NOT break_glass;

-- name: SSOWorkspaceRoles :many
-- Workspaces and their roles, for the JIT default and group mapping pickers.
SELECT w.id AS workspace_id, w.slug, w.name AS workspace_name, r.id AS role_id, r.key AS role_key, r.name AS role_name,
       r.builtin, r.permissions
FROM workspaces w
JOIN roles r ON r.workspace_id = w.id
WHERE w.deleted_at IS NULL
ORDER BY w.created_at, w.id,
         CASE r.key WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 WHEN 'editor' THEN 2 WHEN 'viewer' THEN 4 ELSE 3 END, r.name;

-- name: SSOCreateIdentity :one
INSERT INTO user_identities (id, user_id, kind, sso_provider_id, subject, email, username, details, last_used_at)
VALUES (@id, @user_id, @kind, @sso_provider_id, @subject, @email, @username, @details, now())
RETURNING *;

-- name: SSOTouchIdentity :exec
UPDATE user_identities SET last_used_at = now(), email = @email, username = @username, details = @details WHERE id = @id;

-- name: SSOUserIdentityOfKind :one
SELECT * FROM user_identities
WHERE user_id = @user_id AND kind = @kind
  AND coalesce(sso_provider_id, '00000000-0000-0000-0000-000000000000'::uuid) = coalesce(sqlc.narg(sso_provider_id)::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
LIMIT 1;

-- name: SSOCreateUser :one
INSERT INTO users (id, email, name, timezone, avatar_color, twofa_grace_until)
VALUES (@id, @email, @name, @timezone, @avatar_color, @twofa_grace_until)
RETURNING *;

-- name: SSOMembership :one
SELECT m.id, m.role_id, m.status, m.source, r.key AS role_key, r.name AS role_name, r.builtin AS role_builtin
FROM workspace_members m JOIN roles r ON r.id = m.role_id
WHERE m.workspace_id = @workspace_id AND m.user_id = @user_id;

-- name: SSOSetMemberRole :exec
UPDATE workspace_members SET role_id = @role_id, scope = @scope, source = @source WHERE id = @id;

-- name: SSOCreateMember :exec
INSERT INTO workspace_members (id, workspace_id, user_id, role_id, scope, source)
VALUES (@id, @workspace_id, @user_id, @role_id, @scope, @source);

-- name: SSOPendingInvitation :one
SELECT i.id, i.workspace_id, i.email, i.role_id, i.scope, i.expires_at, i.accepted_at, i.revoked_at, i.invited_by,
       w.slug AS workspace_slug, w.name AS workspace_name, r.key AS role_key, r.name AS role_name, r.builtin AS role_builtin,
       r.permissions AS role_permissions, coalesce(u.status, '')::text AS invited_by_status
FROM invitations i
JOIN workspaces w ON w.id = i.workspace_id AND w.deleted_at IS NULL
JOIN roles r ON r.id = i.role_id
LEFT JOIN users u ON u.id = i.invited_by
WHERE i.token_hash = @token_hash;

-- name: SSOSignInFailures :many
-- Recent sign-in rejections for a provider (setup page "sign-ins rejected").
SELECT id, occurred_at, target_name, reason, metadata
FROM audit_events
WHERE workspace_id IS NULL AND action = 'auth.sign_in' AND result = 'denied' AND metadata ->> 'provider_id' = @provider_id::text
ORDER BY occurred_at DESC
LIMIT 5;
