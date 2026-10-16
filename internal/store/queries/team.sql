-- Team: members, invitations, sign-in methods and member detail.

-- name: TeamListMembers :many
SELECT m.id, m.workspace_id, m.user_id, m.role_id, m.scope, m.source, m.status, m.joined_at, m.deactivated_at,
       m.last_active_at AS member_last_active_at,
       r.key AS role_key, r.name AS role_name, r.builtin AS role_builtin, r.permissions AS role_permissions,
       u.email, u.name AS user_name, u.avatar_color, u.status AS user_status, u.is_instance_admin, u.break_glass,
       u.totp_enabled_at, u.twofa_grace_until, u.last_active_at AS user_last_active_at,
       (u.password_hash IS NOT NULL)::boolean AS has_password,
       (SELECT count(*) FROM user_passkeys p WHERE p.user_id = u.id)::bigint AS passkey_count
FROM workspace_members m
JOIN roles r ON r.id = m.role_id
JOIN users u ON u.id = m.user_id
WHERE m.workspace_id = @workspace_id
ORDER BY m.status = 'deactivated', CASE r.key WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 WHEN 'editor' THEN 2 WHEN 'viewer' THEN 4 ELSE 3 END, u.name;

-- name: TeamGetMember :one
SELECT m.id, m.workspace_id, m.user_id, m.role_id, m.scope, m.source, m.status, m.joined_at, m.deactivated_at,
       m.last_active_at AS member_last_active_at,
       r.key AS role_key, r.name AS role_name, r.builtin AS role_builtin, r.permissions AS role_permissions,
       u.email, u.name AS user_name, u.avatar_color, u.status AS user_status, u.is_instance_admin, u.break_glass,
       u.totp_enabled_at, u.twofa_grace_until, u.last_active_at AS user_last_active_at,
       (u.password_hash IS NOT NULL)::boolean AS has_password,
       (SELECT count(*) FROM user_passkeys p WHERE p.user_id = u.id)::bigint AS passkey_count
FROM workspace_members m
JOIN roles r ON r.id = m.role_id
JOIN users u ON u.id = m.user_id
WHERE m.id = @id AND m.workspace_id = @workspace_id;

-- name: TeamIdentities :many
-- Social and SSO identities of the given users, with the SSO provider name.
SELECT i.user_id, i.kind, coalesce(p.name, '')::text AS provider_name, coalesce(p.preset, '')::text AS provider_preset,
       i.email
FROM user_identities i
LEFT JOIN sso_providers p ON p.id = i.sso_provider_id
WHERE i.user_id = ANY(@user_ids::uuid[])
ORDER BY i.created_at;

-- name: TeamActiveSessionCount :one
SELECT count(*) FROM sessions WHERE user_id = @user_id AND revoked_at IS NULL AND expires_at > now();

-- name: TeamSetMemberRole :exec
UPDATE workspace_members SET role_id = @role_id, scope = @scope WHERE id = @id AND workspace_id = @workspace_id;

-- name: TeamSetMemberStatus :exec
UPDATE workspace_members
SET status = @status, deactivated_at = CASE WHEN @status::text = 'deactivated' THEN now() ELSE NULL END
WHERE id = @id AND workspace_id = @workspace_id;

-- name: TeamDeleteMember :execrows
DELETE FROM workspace_members WHERE id = @id AND workspace_id = @workspace_id;

-- name: TeamEndTwoFAGrace :execrows
-- "Require 2FA now": the grace period ends immediately for users without a second factor.
UPDATE users SET twofa_grace_until = now(), updated_at = now()
WHERE id = ANY(@user_ids::uuid[]) AND totp_enabled_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM user_passkeys p WHERE p.user_id = users.id);

-- name: TeamRevokeUserSessions :execrows
UPDATE sessions SET revoked_at = now() WHERE user_id = @user_id AND revoked_at IS NULL;

-- name: TeamCountActiveOwners :one
SELECT count(*) FROM workspace_members m JOIN roles r ON r.id = m.role_id
WHERE m.workspace_id = @workspace_id AND r.key = 'owner' AND r.builtin AND m.status = 'active';

-- name: TeamListInvitations :many
SELECT i.id, i.workspace_id, i.email, i.role_id, i.scope, i.message, i.invited_by, i.sent_via, i.last_sent_at,
       i.created_at, i.expires_at, i.expires_days, i.token_enc,
       r.name AS role_name, r.key AS role_key, r.builtin AS role_builtin,
       coalesce(u.name, '')::text AS invited_by_name
FROM invitations i
JOIN roles r ON r.id = i.role_id
LEFT JOIN users u ON u.id = i.invited_by
WHERE i.workspace_id = @workspace_id AND i.accepted_at IS NULL AND i.revoked_at IS NULL
ORDER BY i.created_at DESC;

-- name: TeamGetInvitation :one
SELECT i.id, i.workspace_id, i.email, i.role_id, i.scope, i.message, i.invited_by, i.sent_via, i.last_sent_at,
       i.created_at, i.expires_at, i.expires_days, i.token_enc,
       r.name AS role_name, r.key AS role_key, r.builtin AS role_builtin, r.permissions AS role_permissions,
       coalesce(u.name, '')::text AS invited_by_name
FROM invitations i
JOIN roles r ON r.id = i.role_id
LEFT JOIN users u ON u.id = i.invited_by
WHERE i.id = @id AND i.workspace_id = @workspace_id AND i.accepted_at IS NULL AND i.revoked_at IS NULL;

-- name: TeamCreateInvitation :one
INSERT INTO invitations (id, workspace_id, email, role_id, scope, message, token_hash, token_enc, invited_by, sent_via, last_sent_at, expires_at, expires_days)
VALUES (@id, @workspace_id, @email, @role_id, @scope, @message, @token_hash, @token_enc, @invited_by, @sent_via, @last_sent_at, @expires_at, @expires_days)
RETURNING id, created_at, expires_at;

-- name: TeamMarkInvitationSent :exec
UPDATE invitations SET sent_via = @sent_via, last_sent_at = now(), expires_at = @expires_at
WHERE id = @id AND workspace_id = @workspace_id;

-- name: TeamRevokeInvitation :execrows
UPDATE invitations SET revoked_at = now() WHERE id = @id AND workspace_id = @workspace_id AND accepted_at IS NULL AND revoked_at IS NULL;

-- name: TeamPendingInviteEmails :many
SELECT email FROM invitations
WHERE workspace_id = @workspace_id AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at > now() AND email = ANY(@emails::text[]);

-- name: TeamMemberEmails :many
SELECT u.email FROM workspace_members m JOIN users u ON u.id = m.user_id
WHERE m.workspace_id = @workspace_id AND u.email = ANY(@emails::text[]);

-- name: TeamSystemProviderName :one
-- The provider of the System stream's default rule (first split), for
-- "Sent on the System stream via Postmark".
SELECT p.name
FROM routing_rules r
JOIN providers p ON p.id = (r.splits -> 0 ->> 'provider_id')::uuid AND p.disconnected_at IS NULL
WHERE r.workspace_id = @workspace_id AND r.stream = 'system' AND r.enabled
ORDER BY (r.conditions = '{}'::jsonb) DESC, r.priority
LIMIT 1;

-- name: TeamJITProviders :many
-- SSO providers that create accounts on first sign-in for this workspace.
SELECT p.id, p.name, p.preset, p.allowed_domains, coalesce(p.jit_role_key, '')::text AS jit_role_key
FROM sso_providers p
WHERE p.status = 'enabled' AND p.jit_enabled AND p.jit_workspace_id = @workspace_id;

-- name: TeamEnforcedSSO :many
SELECT p.name, p.preset, p.enforce_domains FROM sso_providers p
WHERE p.status = 'enabled' AND cardinality(p.enforce_domains) > 0;

-- name: TeamWorkspaceLists :many
SELECT id, name FROM lists WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY created_at;

-- name: TeamWorkspaceSenderDomains :many
SELECT domain FROM sending_domains WHERE workspace_id = @workspace_id ORDER BY domain;

-- name: TeamMemberAPIKeys :many
-- Integrations' keys are listed apart: the deactivate and remove dialogs
-- take them over or disconnect them.
SELECT k.id, k.kind, k.name, k.prefix, k.permissions, k.scope, k.created_at, k.expires_at, k.revoked_at, k.last_used_at
FROM api_keys k
WHERE k.workspace_id = @workspace_id AND k.created_by = @user_id AND k.integration_id IS NULL
ORDER BY k.revoked_at IS NOT NULL, k.created_at DESC;

-- name: TeamApprovalsGiven :many
SELECT d.decision, d.created_at, coalesce(c.name, '')::text AS campaign_name, a.campaign_id
FROM approval_decisions d
JOIN approval_requests a ON a.id = d.request_id
LEFT JOIN campaigns c ON c.id = a.campaign_id
WHERE a.workspace_id = @workspace_id AND d.approver_id = @user_id
ORDER BY d.created_at DESC
LIMIT 10;

-- name: TeamApprovalsRequested :many
SELECT a.id, a.status, a.requested_at, coalesce(c.name, '')::text AS campaign_name, a.campaign_id
FROM approval_requests a
LEFT JOIN campaigns c ON c.id = a.campaign_id
WHERE a.workspace_id = @workspace_id AND a.requested_by = @user_id
ORDER BY a.requested_at DESC
LIMIT 10;

-- name: TeamMemberActivity :many
SELECT id, occurred_at, action, target_type, target_id, target_name, result, reason, metadata
FROM audit_events
WHERE workspace_id = @workspace_id AND actor_type = 'user' AND actor_id = @user_id
ORDER BY occurred_at DESC, id DESC
LIMIT @lim;

-- name: TeamRevokeKeysCreatedBy :many
-- Integrations' keys stay: deactivating takes them over or disconnects them.
UPDATE api_keys SET revoked_at = now()
WHERE workspace_id = @workspace_id AND created_by = @user_id AND revoked_at IS NULL AND integration_id IS NULL
RETURNING id, name, kind;
