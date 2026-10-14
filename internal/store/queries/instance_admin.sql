-- Instance admin: users, workspaces, policies, instance settings and updates.

-- name: InstanceAdminUsers :many
SELECT u.id, u.email, u.name, u.avatar_color, u.avatar_updated_at, u.is_instance_admin, u.break_glass, u.status, u.deactivated_at,
       (u.password_hash IS NOT NULL)::boolean AS has_password, u.totp_enabled_at, u.twofa_grace_until, u.locked_until,
       u.last_sign_in_at, u.last_active_at, u.created_at,
       (SELECT count(*) FROM user_passkeys p WHERE p.user_id = u.id)::bigint AS passkeys,
       (SELECT count(*) FROM sessions s WHERE s.user_id = u.id AND s.revoked_at IS NULL AND s.expires_at > now())::bigint AS sessions,
       (SELECT count(*) FROM api_keys k JOIN workspaces w ON w.id = k.workspace_id AND w.deleted_at IS NULL
         WHERE k.created_by = u.id AND k.revoked_at IS NULL AND (k.expires_at IS NULL OR k.expires_at > now()))::bigint AS api_keys
FROM users u
ORDER BY u.created_at, u.id;

-- name: InstanceAdminUser :one
SELECT u.id, u.email, u.name, u.avatar_color, u.avatar_updated_at, u.is_instance_admin, u.break_glass, u.status, u.deactivated_at,
       (u.password_hash IS NOT NULL)::boolean AS has_password, u.totp_enabled_at, u.twofa_grace_until, u.locked_until,
       u.last_sign_in_at, u.last_active_at, u.created_at,
       (SELECT count(*) FROM user_passkeys p WHERE p.user_id = u.id)::bigint AS passkeys,
       (SELECT count(*) FROM sessions s WHERE s.user_id = u.id AND s.revoked_at IS NULL AND s.expires_at > now())::bigint AS sessions,
       (SELECT count(*) FROM api_keys k JOIN workspaces w ON w.id = k.workspace_id AND w.deleted_at IS NULL
         WHERE k.created_by = u.id AND k.revoked_at IS NULL AND (k.expires_at IS NULL OR k.expires_at > now()))::bigint AS api_keys
FROM users u
WHERE u.id = @id;

-- name: InstanceAdminMemberships :many
-- Every membership in a live workspace, oldest workspace first.
SELECT m.user_id, w.id AS workspace_id, w.slug, w.name AS workspace_name, r.key AS role_key, r.name AS role_name,
       r.builtin AS role_builtin, r.permissions AS role_permissions, m.status
FROM workspace_members m
JOIN workspaces w ON w.id = m.workspace_id AND w.deleted_at IS NULL
JOIN roles r ON r.id = m.role_id
ORDER BY w.created_at, w.id;

-- name: InstanceAdminIdentities :many
SELECT i.user_id, i.kind, i.sso_provider_id, coalesce(p.name, '')::text AS provider_name, coalesce(p.preset, '')::text AS provider_preset,
       coalesce(p.status, '')::text AS provider_status, coalesce(p.twofa_mode, '')::text AS provider_twofa_mode, i.last_used_at
FROM user_identities i
LEFT JOIN sso_providers p ON p.id = i.sso_provider_id
ORDER BY i.created_at, i.id;

-- name: InstanceAdminSoleOwnerships :many
-- Live workspaces where the user is the only active Owner.
SELECT w.id, w.name, w.slug
FROM workspace_members m
JOIN roles r ON r.id = m.role_id AND r.key = 'owner' AND r.builtin
JOIN workspaces w ON w.id = m.workspace_id AND w.deleted_at IS NULL
WHERE m.user_id = @user_id AND m.status = 'active'
  AND NOT EXISTS (
    SELECT 1 FROM workspace_members m2
    JOIN roles r2 ON r2.id = m2.role_id AND r2.key = 'owner' AND r2.builtin
    JOIN users u2 ON u2.id = m2.user_id AND u2.status = 'active'
    WHERE m2.workspace_id = m.workspace_id AND m2.status = 'active' AND m2.user_id <> m.user_id)
ORDER BY w.created_at;

-- name: InstanceAdminSoleOwners :many
-- Every person who is the only active Owner of a live workspace, with the
-- workspace (the users list reads them in one query).
SELECT m.user_id, w.name
FROM workspace_members m
JOIN roles r ON r.id = m.role_id AND r.key = 'owner' AND r.builtin
JOIN workspaces w ON w.id = m.workspace_id AND w.deleted_at IS NULL
WHERE m.status = 'active'
  AND NOT EXISTS (
    SELECT 1 FROM workspace_members m2
    JOIN roles r2 ON r2.id = m2.role_id AND r2.key = 'owner' AND r2.builtin
    JOIN users u2 ON u2.id = m2.user_id AND u2.status = 'active'
    WHERE m2.workspace_id = m.workspace_id AND m2.status = 'active' AND m2.user_id <> m.user_id)
ORDER BY w.created_at, w.id;

-- name: InstanceAdminCountAdmins :one
SELECT count(*) FROM users WHERE is_instance_admin AND status = 'active';

-- name: InstanceAdminLockAdmins :many
-- The active instance admins, locked until the transaction ends: two admins
-- removing (or deactivating) each other can't both succeed, because the
-- second waits and then re-reads who is still an admin.
SELECT id FROM users WHERE is_instance_admin AND status = 'active' ORDER BY id FOR UPDATE;

-- name: InstanceAdminAdminNames :many
SELECT id, name, email FROM users WHERE is_instance_admin AND status = 'active' ORDER BY created_at;

-- name: InstanceAdminDeactivateUser :execrows
UPDATE users SET status = 'deactivated', deactivated_at = @at, updated_at = now()
WHERE id = @id AND status = 'active';

-- name: InstanceAdminDeactivateMemberships :execrows
UPDATE workspace_members SET status = 'deactivated', deactivated_at = @at
WHERE user_id = @user_id AND status = 'active';

-- name: InstanceAdminReactivateUser :execrows
UPDATE users SET status = 'active', deactivated_at = NULL, failed_attempts = 0, locked_until = NULL, updated_at = now()
WHERE id = @id AND status = 'deactivated';

-- name: InstanceAdminReactivateMemberships :execrows
-- Only the memberships deactivated together with the account come back.
UPDATE workspace_members SET status = 'active', deactivated_at = NULL
WHERE user_id = @user_id AND status = 'deactivated' AND deactivated_at = @at;

-- name: InstanceAdminRevokeKeysCreatedBy :execrows
UPDATE api_keys SET revoked_at = now() WHERE created_by = @user_id AND revoked_at IS NULL;

-- name: InstanceAdminUnlockUser :exec
UPDATE users SET failed_attempts = 0, locked_until = NULL, updated_at = now() WHERE id = @id;

-- name: InstanceAdminResetTwoFactor :exec
-- Removes the authenticator and recovery codes; passkeys stay. The grace
-- period restarts so the person can set two-factor up again.
UPDATE users SET totp_secret_enc = NULL, totp_enabled_at = NULL, totp_label = '', recovery_codes_saved_at = NULL,
       twofa_grace_until = @grace_until, updated_at = now()
WHERE id = @id;

-- name: InstanceAdminUsersInGrace :many
-- People in a two-factor grace period without a second factor yet.
SELECT u.id, u.name, u.email, u.twofa_grace_until
FROM users u
WHERE u.status = 'active' AND u.twofa_grace_until > now() AND u.totp_enabled_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM user_passkeys p WHERE p.user_id = u.id)
ORDER BY u.twofa_grace_until;

-- name: InstanceAdminLockedCount :one
SELECT
  (SELECT count(*) FROM users WHERE status = 'active' AND locked_until > now())::bigint AS users,
  (SELECT count(*) FROM sign_in_throttle WHERE locked_until > now())::bigint AS emails;

-- name: InstanceAdminSignInEvents :many
-- The latest sign-in attempts at instance level (Users › Recent sign-in events).
SELECT e.id, e.occurred_at, e.action, e.actor_type, e.actor_name, e.target_name, e.result, e.reason, e.metadata
FROM audit_events e
WHERE e.workspace_id IS NULL AND e.action LIKE 'signin.%'
ORDER BY e.occurred_at DESC, e.id DESC
LIMIT @lim;

-- name: InstanceAdminWorkspaces :many
-- Live workspaces and the ones deleted within the 7-day restore hold.
SELECT w.id, w.slug, w.name, w.timezone, w.created_at, w.deleted_at,
       (SELECT count(*) FROM workspace_members m WHERE m.workspace_id = w.id)::bigint AS members,
       (SELECT count(*) FROM invitations i WHERE i.workspace_id = w.id AND i.accepted_at IS NULL AND i.revoked_at IS NULL AND i.expires_at > now())::bigint AS invited,
       (SELECT count(*) FROM contacts c WHERE c.workspace_id = w.id AND c.erased_at IS NULL)::bigint AS subscribers,
       coalesce((SELECT array_agg(p.adapter || '|' || p.name ORDER BY p.connected_at) FROM providers p WHERE p.workspace_id = w.id AND p.disconnected_at IS NULL), '{}')::text[] AS providers,
       coalesce((SELECT u.name FROM workspace_members m JOIN roles r ON r.id = m.role_id AND r.key = 'owner' AND r.builtin
                 JOIN users u ON u.id = m.user_id WHERE m.workspace_id = w.id AND m.status = 'active' ORDER BY m.joined_at LIMIT 1), '')::text AS owner_name
FROM workspaces w
WHERE w.deleted_at IS NULL OR w.deleted_at > now() - interval '7 days'
ORDER BY w.deleted_at IS NOT NULL, w.created_at, w.id;

-- name: InstanceAdminWorkspace :one
SELECT * FROM workspaces WHERE id = @id;

-- name: InstanceAdminCancelScheduled :execrows
UPDATE campaigns SET status = 'cancelled', updated_at = now()
WHERE workspace_id = @workspace_id AND status IN ('scheduled', 'approved') AND deleted_at IS NULL;

-- name: InstanceAdminSoftDeleteWorkspace :one
-- The slug is freed for new workspaces; the suffix keeps it unique and
-- within the 63-character limit.
UPDATE workspaces
SET deleted_at = now(), updated_at = now(),
    slug = left(slug, 40) || '-deleted-' || extract(epoch FROM now())::bigint
WHERE id = @id AND deleted_at IS NULL
RETURNING deleted_at;

-- name: InstanceAdminRestoreWorkspace :exec
UPDATE workspaces SET deleted_at = NULL, slug = @slug, updated_at = now() WHERE id = @id AND deleted_at IS NOT NULL;

-- name: InstanceAdminPurgeWorkspace :one
-- Deletes a workspace for good once its restore hold is over (cascade).
DELETE FROM workspaces
WHERE id = @id AND deleted_at IS NOT NULL AND deleted_at <= now() - interval '7 days'
RETURNING id, name;

-- name: InstanceAdminSaveSettings :one
-- A new public URL drops the last URL check, which describes the old one.
UPDATE instance_settings
SET name = @name, public_url = @public_url, default_timezone = @default_timezone, updated_at = now(),
    public_url_check = CASE WHEN public_url = @public_url THEN public_url_check END,
    public_url_checked_at = CASE WHEN public_url = @public_url THEN public_url_checked_at END
WHERE id
RETURNING *;

-- name: InstanceAdminSetLogo :exec
UPDATE instance_settings SET logo_svg = sqlc.narg(logo_svg), updated_at = now() WHERE id;

-- name: InstanceAdminSetURLCheck :exec
UPDATE instance_settings SET public_url_check = @result, public_url_checked_at = now() WHERE id;

-- name: InstanceAdminSetUpdateCheck :exec
UPDATE instance_settings SET update_check = @enabled, updated_at = now() WHERE id;

-- name: InstanceAdminRecordRelease :exec
UPDATE instance_settings
SET latest_release = @latest_release, latest_release_at = now(), update_checked_at = now(), update_check_error = ''
WHERE id;

-- name: InstanceAdminRecordReleaseError :exec
UPDATE instance_settings SET update_checked_at = now(), update_check_error = @error WHERE id;

-- name: InstanceAdminSavePolicy :one
UPDATE instance_settings
SET allowed_methods = @allowed_methods, password_min_length = @password_min_length, password_reuse = @password_reuse,
    breached_check = @breached_check, twofa_enforcement = @twofa_enforcement, twofa_grace_days = @twofa_grace_days,
    session_lifetime = @session_lifetime, session_idle = @session_idle, lockout_attempts = @lockout_attempts,
    lockout_minutes = @lockout_minutes, jit_domain_allowlist = @jit_domain_allowlist, social_providers = @social_providers,
    updated_at = now()
WHERE id
RETURNING *;

-- name: InstanceAdminLastDomainRejection :one
-- The latest sign-in turned away because of its email domain (the
-- allowlist's "for example kim@gmail.com this morning").
SELECT target_name, occurred_at FROM audit_events
WHERE workspace_id IS NULL AND result = 'denied' AND action = 'signin.rejected'
  AND (metadata ->> 'code' = 'domain_not_allowed' OR reason LIKE '%domain not allowed%')
ORDER BY occurred_at DESC
LIMIT 1;

-- name: InstanceAdminSaveSocial :exec
UPDATE instance_settings
SET social_providers = @social_providers, social_secrets_enc = sqlc.narg(social_secrets_enc), updated_at = now()
WHERE id;

-- name: InstanceAdminPasswordHistory :many
-- The hashes of a person's previous passwords, newest first.
SELECT password_hash FROM user_password_history WHERE user_id = @user_id ORDER BY id DESC LIMIT @lim;

-- name: InstanceAdminSystemProvider :one
-- The System-stream provider of the oldest workspace that has one, for "the
-- email goes out via Postmark".
SELECT p.name, p.status
FROM routing_rules r
JOIN workspaces w ON w.id = r.workspace_id AND w.deleted_at IS NULL
JOIN providers p ON p.id = (r.splits -> 0 ->> 'provider_id')::uuid AND p.disconnected_at IS NULL
WHERE r.stream = 'system' AND r.enabled AND r.conditions = '{}'::jsonb
ORDER BY w.created_at, r.priority
LIMIT 1;
