-- Account & security, passkeys, password resets, email changes, invitations
-- (accept side) and the first sign-in. Owned by internal/accounts.

-- name: SetUserWebAuthnID :exec
UPDATE users SET webauthn_id = @webauthn_id, updated_at = now() WHERE id = @id AND webauthn_id IS NULL;

-- name: GetUserByWebAuthnID :one
SELECT * FROM users WHERE webauthn_id = @webauthn_id;

-- name: CreatePasskeyRecord :one
INSERT INTO user_passkeys (id, user_id, name, credential_id, public_key, attestation_type, aaguid, sign_count,
                           transports, backup_eligible, backup_state, credential, attachment, provider)
VALUES (@id, @user_id, @name, @credential_id, @public_key, @attestation_type, @aaguid, @sign_count,
        @transports, @backup_eligible, @backup_state, @credential, @attachment, @provider)
RETURNING *;

-- name: GetUserPasskey :one
SELECT * FROM user_passkeys WHERE id = @id AND user_id = @user_id;

-- name: UpdatePasskeyAfterUse :exec
UPDATE user_passkeys SET credential = @credential, sign_count = @sign_count, backup_state = @backup_state, last_used_at = now()
WHERE id = @id;

-- name: SetSessionPasskey :exec
UPDATE sessions SET passkey_id = @passkey_id WHERE id = @id;

-- name: GetUserSession :one
SELECT * FROM sessions WHERE id = @id AND user_id = @user_id AND revoked_at IS NULL AND expires_at > now();

-- name: ListIdentitiesWithProvider :many
SELECT i.*, coalesce(p.name, '')::text AS provider_name, coalesce(p.preset, '')::text AS provider_preset,
       coalesce(p.enforce_domains, '{}')::text[] AS enforce_domains, coalesce(p.status, '')::text AS provider_status
FROM user_identities i
LEFT JOIN sso_providers p ON p.id = i.sso_provider_id
WHERE i.user_id = @user_id
ORDER BY i.created_at;

-- name: SetUserTOTPLabel :exec
UPDATE users SET totp_label = @totp_label, updated_at = now() WHERE id = @id;

-- name: SetRecoveryCodesSaved :exec
UPDATE users SET recovery_codes_saved_at = now(), updated_at = now() WHERE id = @id;

-- name: RecoveryCodeStats :one
SELECT count(*) FILTER (WHERE used_at IS NULL) AS remaining, count(*) AS total
FROM user_recovery_codes WHERE user_id = @user_id;

-- name: LastRecoveryCodeUse :one
SELECT used_at::timestamptz FROM user_recovery_codes
WHERE user_id = @user_id AND used_at IS NOT NULL
ORDER BY used_at DESC
LIMIT 1;

-- name: UpsertAvatar :exec
INSERT INTO user_avatars (user_id, content_type, data, updated_at)
VALUES (@user_id, @content_type, @data, now())
ON CONFLICT (user_id) DO UPDATE SET content_type = excluded.content_type, data = excluded.data, updated_at = now();

-- name: GetAvatar :one
SELECT * FROM user_avatars WHERE user_id = @user_id;

-- name: DeleteAvatar :execrows
DELETE FROM user_avatars WHERE user_id = @user_id;

-- name: SetAvatarUpdatedAt :exec
UPDATE users SET avatar_updated_at = sqlc.narg(avatar_updated_at), updated_at = now() WHERE id = @id;

-- name: UsersShareWorkspace :one
SELECT EXISTS (
    SELECT 1 FROM workspace_members a
    JOIN workspace_members b ON b.workspace_id = a.workspace_id
    WHERE a.user_id = @viewer AND b.user_id = @subject AND a.status = 'active'
);

-- name: EmailTaken :one
SELECT EXISTS (SELECT 1 FROM users WHERE email = lower(@email::text) AND id <> @user_id);

-- name: CreateEmailChange :one
INSERT INTO email_changes (id, user_id, new_email, token_hash, expires_at)
VALUES (@id, @user_id, @new_email, @token_hash, @expires_at)
RETURNING *;

-- name: GetPendingEmailChange :one
SELECT * FROM email_changes
WHERE user_id = @user_id AND used_at IS NULL AND expires_at > now()
ORDER BY created_at DESC
LIMIT 1;

-- name: ConsumeEmailChange :one
UPDATE email_changes SET used_at = now()
WHERE token_hash = @token_hash AND user_id = @user_id AND used_at IS NULL AND expires_at > now()
RETURNING *;

-- name: DeletePendingEmailChanges :execrows
DELETE FROM email_changes WHERE user_id = @user_id AND used_at IS NULL;

-- name: CreatePasswordResetFrom :one
INSERT INTO password_resets (id, user_id, token_hash, expires_at, source)
VALUES (@id, @user_id, @token_hash, @expires_at, @source)
RETURNING *;

-- name: LastPasswordResetAt :one
SELECT created_at FROM password_resets
WHERE user_id = @user_id AND source = 'email'
ORDER BY created_at DESC
LIMIT 1;

-- name: LastAuditAt :one
-- When an action was last recorded for a target (throttles repeated emails).
SELECT occurred_at FROM audit_events
WHERE action = @action AND target_id = @target_id
ORDER BY occurred_at DESC
LIMIT 1;

-- name: GetUsablePasswordReset :one
SELECT r.*, u.email, u.status AS user_status
FROM password_resets r JOIN users u ON u.id = r.user_id
WHERE r.token_hash = @token_hash AND r.used_at IS NULL AND r.expires_at > now();

-- name: InvalidatePasswordResets :exec
UPDATE password_resets SET used_at = now() WHERE user_id = @user_id AND used_at IS NULL;

-- name: ClearUserLockout :exec
UPDATE users SET failed_attempts = 0, locked_until = NULL, updated_at = now() WHERE id = @id;

-- name: GetInvitationView :one
SELECT i.*, r.name AS role_name, r.key AS role_key, r.builtin AS role_builtin, r.permissions AS role_permissions,
       w.name AS workspace_name, w.slug AS workspace_slug, w.approval_policy AS workspace_approval_policy,
       coalesce(u.name, '')::text AS invited_by_name, coalesce(u.email, '')::text AS invited_by_email,
       coalesce(u.status, '')::text AS invited_by_status
FROM invitations i
JOIN roles r ON r.id = i.role_id
JOIN workspaces w ON w.id = i.workspace_id AND w.deleted_at IS NULL
LEFT JOIN users u ON u.id = i.invited_by
WHERE i.token_hash = @token_hash;

-- name: AcceptPendingInvitation :execrows
UPDATE invitations SET accepted_at = now(), accepted_by = @accepted_by
WHERE id = @id AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at > now();

-- name: MarkReinviteRequested :execrows
UPDATE invitations SET reinvite_requested_at = now()
WHERE id = @id AND (reinvite_requested_at IS NULL OR reinvite_requested_at < now() - interval '24 hours');

-- name: ListNamesOfLists :many
SELECT id, name FROM lists WHERE workspace_id = @workspace_id AND id = ANY(@ids::uuid[]) ORDER BY name;

-- name: ListMembershipsOfUser :many
SELECT m.id AS member_id, m.status, w.id AS workspace_id, w.slug, w.name, r.key AS role_key, r.name AS role_name,
       r.builtin AS role_builtin, r.permissions AS role_permissions,
       -- Members as the team list counts them: deactivated members included.
       (SELECT count(*) FROM workspace_members m2 WHERE m2.workspace_id = w.id) AS member_count,
       (SELECT count(*) FROM workspace_members m3 JOIN roles r3 ON r3.id = m3.role_id
         WHERE m3.workspace_id = w.id AND m3.status = 'active' AND r3.key = 'owner') AS owner_count
FROM workspace_members m
JOIN workspaces w ON w.id = m.workspace_id AND w.deleted_at IS NULL
JOIN roles r ON r.id = m.role_id
WHERE m.user_id = @user_id AND m.status = 'active'
ORDER BY w.created_at;

-- name: ConfirmInstanceBasics :exec
UPDATE instance_settings SET basics_confirmed_at = now(), updated_at = now() WHERE id;

-- name: EnabledSSOProviders :many
SELECT id, name, preset, allowed_domains, enforce_domains FROM sso_providers WHERE status = 'enabled' ORDER BY created_at;

-- name: SystemStreamProviderName :one
-- The provider of the default System-stream route of a workspace (read-only
-- view of the routing domain, for "emails go out … via Postmark").
SELECT p.name
FROM routing_rules r
JOIN providers p ON p.id = (r.splits -> 0 ->> 'provider_id')::uuid AND p.disconnected_at IS NULL
WHERE r.workspace_id = @workspace_id AND r.stream = 'system' AND r.enabled AND r.conditions = '{}'::jsonb
ORDER BY r.priority
LIMIT 1;
