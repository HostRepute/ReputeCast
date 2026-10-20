-- Agency accounts (design expansion phase 8, M32; internal/agency).

-- ---------------------------------------------------------------------------
-- Agency mode

-- name: AgencySettings :one
SELECT a.enabled, a.name, a.default_template_id, a.enabled_by, a.enabled_at, a.updated_at,
       coalesce(u.name, '')::text AS enabled_by_name, i.name AS instance_name, i.public_url AS instance_public_url
FROM agency_settings a
CROSS JOIN instance_settings i
LEFT JOIN users u ON u.id = a.enabled_by
WHERE a.id;

-- name: AgencySettingsForUpdate :one
SELECT enabled, name, default_template_id, enabled_by, enabled_at FROM agency_settings WHERE id FOR UPDATE;

-- name: AgencySaveSettings :exec
UPDATE agency_settings
SET enabled = @enabled, name = @name, default_template_id = sqlc.narg(default_template_id),
    enabled_by = sqlc.narg(enabled_by), enabled_at = sqlc.narg(enabled_at), updated_by = sqlc.narg(updated_by), updated_at = now()
WHERE id;

-- ---------------------------------------------------------------------------
-- Staff

-- name: AgencyStaffOf :one
-- A person's staff row: their own, or one waiting for their email.
SELECT id, user_id, email, level, title, permissions, added_by, added_at
FROM agency_staff
WHERE user_id = @user_id OR (user_id IS NULL AND email = @email)
ORDER BY user_id IS NULL
LIMIT 1;

-- name: AgencyBindStaff :exec
UPDATE agency_staff SET user_id = @user_id WHERE user_id IS NULL AND email = @email;

-- name: AgencyStaffRows :many
-- Every staff row with the account behind it, when there is one.
SELECT s.id, s.user_id, s.email, s.level, s.title, s.permissions, s.added_at,
       coalesce(u.name, '')::text AS name, coalesce(u.avatar_color, 'neutral')::text AS avatar_color,
       coalesce(u.is_instance_admin, false)::boolean AS is_instance_admin, coalesce(u.status, 'active')::text AS user_status,
       u.last_active_at
FROM agency_staff s
LEFT JOIN users u ON u.id = s.user_id
ORDER BY s.added_at, s.email;

-- name: AgencyInstanceAdmins :many
-- Instance admins are agency managers, with or without a staff row.
SELECT id, email, name, avatar_color, last_active_at, status
FROM users
WHERE is_instance_admin AND status = 'active'
ORDER BY created_at;

-- name: AgencyStaffInsert :exec
INSERT INTO agency_staff (id, user_id, email, level, title, permissions, added_by)
VALUES (@id, sqlc.narg(user_id), @email, @level, @title, sqlc.narg(permissions)::text[], sqlc.narg(added_by));

-- name: AgencyStaffUpdate :execrows
UPDATE agency_staff SET level = @level, title = @title, permissions = sqlc.narg(permissions)::text[] WHERE id = @id;

-- name: AgencyStaffDelete :execrows
DELETE FROM agency_staff WHERE id = @id;

-- name: AgencyStaffIn :many
-- Which of these people are agency staff while agency mode is on: staff
-- rows and instance admins. Client sign-off doesn't count their approvals.
SELECT u.id
FROM users u
WHERE u.id = ANY(@user_ids::uuid[])
  AND (SELECT enabled FROM agency_settings WHERE id)
  AND (u.is_instance_admin OR EXISTS (SELECT 1 FROM agency_staff s WHERE s.user_id = u.id OR (s.user_id IS NULL AND s.email = u.email)));

-- name: AgencyStaffLevels :many
-- Instance admin › Users: each staff member's level (instance admins are
-- agency managers).
SELECT u.id AS user_id,
       CASE WHEN u.is_instance_admin THEN 'agency_manager' ELSE coalesce(s.level, '') END::text AS level
FROM users u
LEFT JOIN agency_staff s ON s.user_id = u.id OR (s.user_id IS NULL AND s.email = u.email)
WHERE (SELECT enabled FROM agency_settings WHERE id)
  AND (u.is_instance_admin OR s.id IS NOT NULL);

-- ---------------------------------------------------------------------------
-- The agency's workspaces

-- name: AgencyWorkspaces :many
SELECT w.id, w.slug, w.name, w.timezone, w.created_at, a.kind, a.account_manager_id, a.client_since, a.template_id, a.setup_steps, a.color, a.added_at,
       coalesce(u.name, '')::text AS account_manager_name, coalesce(u.email, '')::text AS account_manager_email,
       coalesce(u.avatar_color, 'neutral')::text AS account_manager_color,
       coalesce(t.name, '')::text AS template_name,
       (SELECT count(*) FROM contacts c WHERE c.workspace_id = w.id AND c.erased_at IS NULL)::bigint AS contacts
FROM agency_workspaces a
JOIN workspaces w ON w.id = a.workspace_id AND w.deleted_at IS NULL
LEFT JOIN users u ON u.id = a.account_manager_id
LEFT JOIN agency_templates t ON t.id = a.template_id
ORDER BY a.kind = 'agency', w.name;

-- name: AgencyWorkspace :one
SELECT w.id, w.slug, w.name, w.timezone, w.created_at, a.kind, a.account_manager_id, a.client_since, a.template_id, a.setup_steps, a.color, a.added_at,
       coalesce(u.name, '')::text AS account_manager_name, coalesce(u.email, '')::text AS account_manager_email,
       coalesce(u.avatar_color, 'neutral')::text AS account_manager_color,
       coalesce(t.name, '')::text AS template_name
FROM agency_workspaces a
JOIN workspaces w ON w.id = a.workspace_id AND w.deleted_at IS NULL
LEFT JOIN users u ON u.id = a.account_manager_id
LEFT JOIN agency_templates t ON t.id = a.template_id
WHERE a.workspace_id = @workspace_id;

-- name: AgencyWorkspaceInsert :exec
INSERT INTO agency_workspaces (workspace_id, kind, account_manager_id, client_since, template_id, setup_steps, color, added_by)
VALUES (@workspace_id, @kind, sqlc.narg(account_manager_id), @client_since, sqlc.narg(template_id), @setup_steps::text[], @color, sqlc.narg(added_by))
ON CONFLICT (workspace_id) DO UPDATE SET kind = EXCLUDED.kind;

-- name: AgencyWorkspaceSetManager :execrows
UPDATE agency_workspaces SET account_manager_id = sqlc.narg(account_manager_id) WHERE workspace_id = @workspace_id;

-- name: AgencyWorkspaceSetTemplate :exec
UPDATE agency_workspaces SET template_id = @template_id,
       setup_steps = CASE WHEN cardinality(setup_steps) = 0 THEN @setup_steps::text[] ELSE setup_steps END
WHERE workspace_id = @workspace_id;

-- name: AgencyWorkspaceRemove :execrows
DELETE FROM agency_workspaces WHERE workspace_id = @workspace_id;

-- name: AgencyWorkspaceKinds :many
-- Every live workspace, and whether it's one of the agency's.
SELECT w.id, w.slug, w.name, w.created_at, coalesce(a.kind, '')::text AS kind
FROM workspaces w
LEFT JOIN agency_workspaces a ON a.workspace_id = w.id
WHERE w.deleted_at IS NULL
ORDER BY w.created_at, w.id;

-- name: AgencyIsClientWorkspace :one
SELECT EXISTS (SELECT 1 FROM agency_workspaces a JOIN agency_settings s ON s.id AND s.enabled
               WHERE a.workspace_id = @workspace_id AND a.kind = 'client');

-- name: AgencyMemberships :many
-- Staff members' memberships in the agency's workspaces (the assignment
-- matrix) and every client person, with their roles.
SELECT m.id, m.workspace_id, m.user_id, m.role_id, m.source, m.status, m.joined_at,
       r.key AS role_key, r.name AS role_name, r.builtin AS role_builtin, r.permissions AS role_permissions
FROM workspace_members m
JOIN roles r ON r.id = m.role_id
JOIN agency_workspaces a ON a.workspace_id = m.workspace_id
JOIN workspaces w ON w.id = m.workspace_id AND w.deleted_at IS NULL
WHERE m.user_id = ANY(@user_ids::uuid[]) AND m.status = 'active';

-- name: AgencyWorkspaceRoles :many
SELECT id, key, name, builtin, permissions FROM roles WHERE workspace_id = @workspace_id ORDER BY builtin DESC, created_at;

-- name: AgencyClientPeople :many
-- The people of a client who aren't staff: members and pending invitations.
SELECT u.id AS user_id, u.name, u.email, u.avatar_color, r.key AS role_key, r.name AS role_name, r.builtin AS role_builtin, false AS invited
FROM workspace_members m
JOIN users u ON u.id = m.user_id
JOIN roles r ON r.id = m.role_id
WHERE m.workspace_id = @workspace_id AND m.status = 'active' AND NOT (m.user_id = ANY(@staff_ids::uuid[]))
UNION ALL
SELECT NULL::uuid, ''::text, i.email, 'neutral'::text, r.key, r.name, r.builtin, true
FROM invitations i
JOIN roles r ON r.id = i.role_id
WHERE i.workspace_id = @workspace_id AND i.accepted_at IS NULL AND i.revoked_at IS NULL AND i.expires_at > now();

-- ---------------------------------------------------------------------------
-- Rebill rates

-- name: AgencyRates :many
SELECT r.workspace_id, r.effective_from, r.rate, r.set_by, r.set_at, coalesce(u.name, '')::text AS set_by_name
FROM agency_rates r
LEFT JOIN users u ON u.id = r.set_by
WHERE r.workspace_id = ANY(@workspace_ids::uuid[])
ORDER BY r.workspace_id, r.effective_from;

-- name: AgencyRateUpsert :exec
INSERT INTO agency_rates (workspace_id, effective_from, rate, set_by, set_at)
VALUES (@workspace_id, @effective_from, @rate, sqlc.narg(set_by), now())
ON CONFLICT (workspace_id, effective_from) DO UPDATE SET rate = EXCLUDED.rate, set_by = EXCLUDED.set_by, set_at = now();

-- ---------------------------------------------------------------------------
-- Workspace templates

-- name: AgencyTemplates :many
SELECT t.id, t.name, t.description, t.content, t.created_from, t.created_at, t.updated_at,
       coalesce(cu.name, '')::text AS created_by_name, coalesce(uu.name, '')::text AS updated_by_name
FROM agency_templates t
LEFT JOIN users cu ON cu.id = t.created_by
LEFT JOIN users uu ON uu.id = t.updated_by
WHERE t.deleted_at IS NULL
ORDER BY t.created_at, t.name;

-- name: AgencyTemplate :one
SELECT t.id, t.name, t.description, t.content, t.created_from, t.created_at, t.updated_at,
       coalesce(cu.name, '')::text AS created_by_name, coalesce(uu.name, '')::text AS updated_by_name
FROM agency_templates t
LEFT JOIN users cu ON cu.id = t.created_by
LEFT JOIN users uu ON uu.id = t.updated_by
WHERE t.id = @id AND t.deleted_at IS NULL;

-- name: AgencyTemplateNameTaken :one
SELECT EXISTS (SELECT 1 FROM agency_templates WHERE lower(name) = lower(@name) AND deleted_at IS NULL AND id <> @exclude_id);

-- name: AgencyTemplateInsert :exec
INSERT INTO agency_templates (id, name, description, content, created_from, created_by, updated_by)
VALUES (@id, @name, @description, @content, sqlc.narg(created_from), sqlc.narg(created_by), sqlc.narg(created_by));

-- name: AgencyTemplateUpdate :execrows
UPDATE agency_templates SET name = @name, description = @description, content = @content, updated_by = sqlc.narg(updated_by), updated_at = now()
WHERE id = @id AND deleted_at IS NULL;

-- name: AgencyTemplateDelete :execrows
UPDATE agency_templates SET deleted_at = now() WHERE id = @id AND deleted_at IS NULL;

-- name: AgencyTemplateUses :many
-- The workspaces each template set up or was applied to, oldest first.
SELECT x.template_id, x.workspace_id, w.name AS workspace_name, min(x.at)::timestamptz AS first_at
FROM (
    SELECT template_id, workspace_id, applied_at AS at FROM agency_template_applications
    UNION ALL
    SELECT template_id, workspace_id, added_at FROM agency_workspaces WHERE template_id IS NOT NULL
) x
JOIN workspaces w ON w.id = x.workspace_id AND w.deleted_at IS NULL
GROUP BY x.template_id, x.workspace_id, w.name
ORDER BY first_at, w.name;

-- name: AgencyTemplateApplicationInsert :exec
INSERT INTO agency_template_applications (id, template_id, workspace_id, applied_by, applied_at, result)
VALUES (@id, @template_id, @workspace_id, sqlc.narg(applied_by), @applied_at, @result);

-- name: AgencyLatestApplication :one
SELECT a.id, a.template_id, a.applied_at, a.result, coalesce(u.name, '')::text AS applied_by_name, t.name AS template_name
FROM agency_template_applications a
JOIN agency_templates t ON t.id = a.template_id
LEFT JOIN users u ON u.id = a.applied_by
WHERE a.workspace_id = @workspace_id
ORDER BY a.applied_at DESC
LIMIT 1;

-- ---------------------------------------------------------------------------
-- Shared library

-- name: AgencyLibraryItems :many
SELECT i.id, i.kind, i.name, i.version, i.fields, i.created_at, i.updated_at,
       coalesce(u.name, '')::text AS updated_by_name,
       (SELECT count(*) FROM agency_library_copies c JOIN workspaces w ON w.id = c.workspace_id AND w.deleted_at IS NULL WHERE c.item_id = i.id)::bigint AS copies,
       (SELECT count(DISTINCT c.workspace_id) FROM agency_library_copies c JOIN workspaces w ON w.id = c.workspace_id AND w.deleted_at IS NULL WHERE c.item_id = i.id)::bigint AS clients,
       (SELECT count(*) FROM agency_library_copies c JOIN workspaces w ON w.id = c.workspace_id AND w.deleted_at IS NULL WHERE c.item_id = i.id AND c.version < i.version)::bigint AS behind
FROM agency_library_items i
LEFT JOIN users u ON u.id = i.updated_by
WHERE i.deleted_at IS NULL
ORDER BY i.updated_at DESC, i.name;

-- name: AgencyLibraryItem :one
SELECT i.id, i.kind, i.name, i.version, i.fields, i.created_at, i.updated_at, coalesce(u.name, '')::text AS updated_by_name
FROM agency_library_items i
LEFT JOIN users u ON u.id = i.updated_by
WHERE i.id = @id AND i.deleted_at IS NULL;

-- name: AgencyLibraryVersions :many
SELECT v.item_id, v.version, v.content, v.summary, v.note, v.source_workspace_id, v.created_at, coalesce(u.name, '')::text AS created_by_name
FROM agency_library_versions v
LEFT JOIN users u ON u.id = v.created_by
WHERE v.item_id = @item_id
ORDER BY v.version DESC;

-- name: AgencyLibraryVersion :one
SELECT v.item_id, v.version, v.content, v.summary, v.note, v.created_at
FROM agency_library_versions v
WHERE v.item_id = @item_id AND v.version = @version;

-- name: AgencyLibraryNameTaken :one
SELECT EXISTS (SELECT 1 FROM agency_library_items WHERE kind = @kind AND lower(name) = lower(@name) AND deleted_at IS NULL AND id <> @exclude_id);

-- name: AgencyLibraryInsertItem :exec
INSERT INTO agency_library_items (id, kind, name, version, fields, created_by, created_at, updated_by, updated_at)
VALUES (@id, @kind, @name, 1, @fields, sqlc.narg(created_by), @at, sqlc.narg(created_by), @at);

-- name: AgencyLibraryInsertVersion :exec
INSERT INTO agency_library_versions (item_id, version, content, summary, note, source_workspace_id, created_by, created_at)
VALUES (@item_id, @version, @content, @summary::text[], @note, sqlc.narg(source_workspace_id), sqlc.narg(created_by), @at);

-- name: AgencyLibraryBump :execrows
UPDATE agency_library_items SET version = @version, name = @name, fields = @fields, updated_by = sqlc.narg(updated_by), updated_at = @at
WHERE id = @id AND deleted_at IS NULL;

-- name: AgencyLibraryDelete :execrows
UPDATE agency_library_items SET deleted_at = now() WHERE id = @id AND deleted_at IS NULL;

-- name: AgencyLibraryCopies :many
SELECT c.id, c.item_id, c.version, c.workspace_id, c.target_kind, c.target_id, c.copied_at, c.updated_at,
       w.name AS workspace_name, w.slug AS workspace_slug
FROM agency_library_copies c
JOIN workspaces w ON w.id = c.workspace_id AND w.deleted_at IS NULL
WHERE c.item_id = @item_id
ORDER BY c.copied_at, w.name;

-- name: AgencyLibraryCopy :one
SELECT c.id, c.item_id, c.version, c.workspace_id, c.target_kind, c.target_id, c.copied_at, c.updated_at
FROM agency_library_copies c
WHERE c.id = @id;

-- name: AgencyLibraryCopyOf :one
-- The library item a workspace's template, automation or transactional
-- template was copied from.
SELECT c.id, c.item_id, c.version, i.version AS latest, i.name, i.kind
FROM agency_library_copies c
JOIN agency_library_items i ON i.id = c.item_id AND i.deleted_at IS NULL
WHERE c.target_kind = @target_kind AND c.target_id = @target_id AND c.workspace_id = @workspace_id;

-- name: AgencyLibraryInsertCopy :exec
INSERT INTO agency_library_copies (id, item_id, version, workspace_id, target_kind, target_id, copied_by, copied_at, updated_at)
VALUES (@id, @item_id, @version, @workspace_id, @target_kind, @target_id, sqlc.narg(copied_by), @at, @at);

-- name: AgencyLibraryUpdateCopy :execrows
UPDATE agency_library_copies SET version = @version, updated_at = @at WHERE id = @id;

-- name: AgencyLibraryCopiesIn :many
-- The copies in one workspace, for its gallery's "Northwind library" tab.
SELECT c.item_id, c.version, c.target_kind, c.target_id FROM agency_library_copies c WHERE c.workspace_id = @workspace_id;

-- ---------------------------------------------------------------------------
-- Usage

-- name: AgencyUsageRange :many
-- Usage per workspace, channel, dimension and key between two days.
SELECT workspace_id, channel, dimension, key, max(label)::text AS label,
       sum(units)::bigint AS units, sum(cost_micros)::bigint AS cost_micros
FROM agency_usage
WHERE workspace_id = ANY(@workspace_ids::uuid[]) AND day >= @from_day AND day < @to_day
GROUP BY workspace_id, channel, dimension, key
ORDER BY workspace_id, channel, dimension, key;

-- name: AgencyUsageComputedAt :one
SELECT coalesce(extract(epoch FROM max(computed_at)), 0)::bigint FROM agency_usage
WHERE workspace_id = ANY(@workspace_ids::uuid[]) AND day >= @from_day AND day < @to_day;

-- name: AgencyUsageMonthHasDemo :one
SELECT EXISTS (SELECT 1 FROM agency_usage WHERE workspace_id = @workspace_id AND demo AND day >= @from_day AND day < @to_day);

-- name: AgencyUsageClearDay :exec
DELETE FROM agency_usage WHERE workspace_id = @workspace_id AND day = @day AND NOT demo;

-- name: AgencyUsagePut :exec
INSERT INTO agency_usage (workspace_id, day, channel, dimension, key, label, units, cost_micros, computed_at)
VALUES (@workspace_id, @day, @channel, @dimension, @key, @label, @units, @cost_micros, @computed_at)
ON CONFLICT (workspace_id, day, channel, dimension, key) DO UPDATE
SET label = EXCLUDED.label, units = EXCLUDED.units, cost_micros = EXCLUDED.cost_micros, computed_at = EXCLUDED.computed_at
WHERE NOT agency_usage.demo;

-- name: AgencyUsageEmailStreams :many
-- Emails a workspace's providers accepted on one day, per stream.
SELECT m.stream, count(*)::bigint AS n
FROM messages m
WHERE m.provider_id = ANY(@provider_ids::uuid[]) AND m.sent_at >= @from_at AND m.sent_at < @to_at
GROUP BY m.stream;

-- name: AgencyUsageProviderDays :many
-- Each email provider's sends and metered cost on one day.
SELECT p.id, p.name, p.adapter, coalesce(d.sent, 0)::bigint AS sent, coalesce(d.cost_micros, 0)::bigint AS cost_micros
FROM providers p
JOIN provider_daily_stats d ON d.provider_id = p.id AND d.day = @day
WHERE p.workspace_id = @workspace_id;

-- name: AgencyUsageProviders :many
-- A workspace's email providers, connected or not, with their pricing.
SELECT id, workspace_id, name, adapter, label, pricing, connected_at, disconnected_at
FROM providers
WHERE workspace_id = ANY(@workspace_ids::uuid[])
ORDER BY connected_at;

-- ---------------------------------------------------------------------------
-- Sender health snapshots

-- name: AgencyHealthSince :many
SELECT workspace_id, day, state, score, tone, hostrepute, listings, computed_at
FROM agency_health
WHERE workspace_id = ANY(@workspace_ids::uuid[]) AND day >= @since
ORDER BY workspace_id, day;

-- name: AgencyHealthPut :exec
INSERT INTO agency_health (workspace_id, day, state, score, tone, hostrepute, listings, computed_at)
VALUES (@workspace_id, @day, @state, @score, @tone, @hostrepute, @listings, @computed_at)
ON CONFLICT (workspace_id, day) DO UPDATE
SET state = EXCLUDED.state, score = EXCLUDED.score, tone = EXCLUDED.tone, hostrepute = EXCLUDED.hostrepute,
    listings = EXCLUDED.listings, computed_at = EXCLUDED.computed_at
WHERE NOT agency_health.demo;

-- ---------------------------------------------------------------------------
-- The overview's rows

-- name: AgencyPlannedSends :many
-- Sends with a time, in the given workspaces: scheduled, approved, or
-- waiting for approval with the time they asked for. Digest templates
-- aren't sends.
SELECT c.workspace_id, c.id, c.name, c.status, c.audience_count, c.schedule,
       coalesce(c.scheduled_at, CASE WHEN coalesce(c.schedule->>'at', '') <> '' THEN (c.schedule->>'at')::timestamptz END)::timestamptz AS send_at
FROM campaigns c
WHERE c.workspace_id = ANY(@workspace_ids::uuid[]) AND c.deleted_at IS NULL
  AND c.status IN ('scheduled', 'approval', 'approved')
  AND c.digest_role NOT IN ('template')
  AND coalesce(c.scheduled_at, CASE WHEN coalesce(c.schedule->>'at', '') <> '' THEN (c.schedule->>'at')::timestamptz END) >= @since
ORDER BY send_at, c.id
LIMIT 500;

-- name: AgencyLatestDrafts :many
-- Each workspace's most recently changed draft.
SELECT DISTINCT ON (c.workspace_id) c.workspace_id, c.name
FROM campaigns c
WHERE c.workspace_id = ANY(@workspace_ids::uuid[]) AND c.deleted_at IS NULL AND c.status = 'draft' AND c.digest_role NOT IN ('template', 'next')
ORDER BY c.workspace_id, c.updated_at DESC;

-- name: AgencyPendingApprovals :many
SELECT a.workspace_id, a.id, a.requested_at, a.matched_rules, a.campaign_id, a.requested_by, a.approvals_required,
       coalesce(c.name, '')::text AS campaign_name, c.scheduled_at, coalesce(c.schedule, '{}')::jsonb AS schedule,
       coalesce(c.audience_count, 0)::bigint AS audience_count, a.snapshot,
       coalesce(u.name, '')::text AS requested_by_name
FROM approval_requests a
JOIN campaigns c ON c.id = a.campaign_id AND c.deleted_at IS NULL
LEFT JOIN users u ON u.id = a.requested_by
WHERE a.workspace_id = ANY(@workspace_ids::uuid[]) AND a.status = 'pending'
ORDER BY a.requested_at;

-- name: AgencyApprovalDecisionsFor :many
SELECT d.request_id, d.approver_id, d.decision FROM approval_decisions d WHERE d.request_id = ANY(@request_ids::uuid[]);

-- name: AgencyApprovers :many
-- Active members of the given workspaces with their roles: who holds
-- campaigns:approve is decided in Go (built-in roles come from code).
SELECT m.workspace_id, m.user_id, u.name, r.key AS role_key, r.builtin, r.permissions
FROM workspace_members m
JOIN users u ON u.id = m.user_id AND u.status = 'active'
JOIN roles r ON r.id = m.role_id
WHERE m.workspace_id = ANY(@workspace_ids::uuid[]) AND m.status = 'active'
ORDER BY m.joined_at;

-- name: AgencyRecentIncidents :many
SELECT i.workspace_id, i.id, i.kind, i.severity, i.title, i.detail, i.occurred_at, i.resolved_at, i.campaign_id, i.provider_id,
       coalesce(c.name, '')::text AS campaign_name, coalesce(p.name, '')::text AS provider_name
FROM incidents i
LEFT JOIN campaigns c ON c.id = i.campaign_id
LEFT JOIN providers p ON p.id = i.provider_id
WHERE i.workspace_id = ANY(@workspace_ids::uuid[]) AND i.occurred_at >= @since
  AND i.kind IN ('auto_pause', 'provider_down', 'webhooks_silent', 'sms_budget_reached', 'store_disconnected')
ORDER BY i.occurred_at DESC
LIMIT 500;

-- name: AgencyAttentionFacts :many
-- What puts a workspace under Needs attention, cheaply: incidents in the
-- last 7 days and the oldest approval waiting, plus the next send.
SELECT w.id AS workspace_id,
  (SELECT count(*) FROM incidents i WHERE i.workspace_id = w.id AND i.occurred_at >= @since
     AND i.kind IN ('auto_pause', 'provider_down', 'webhooks_silent', 'sms_budget_reached', 'store_disconnected'))::bigint AS incidents,
  (SELECT count(*) FROM approval_requests a JOIN campaigns c ON c.id = a.campaign_id AND c.deleted_at IS NULL
     WHERE a.workspace_id = w.id AND a.status = 'pending')::bigint AS approvals,
  -- Times as Unix seconds, 0 when there is none.
  coalesce((SELECT extract(epoch FROM min(a.requested_at)) FROM approval_requests a JOIN campaigns c ON c.id = a.campaign_id AND c.deleted_at IS NULL
     WHERE a.workspace_id = w.id AND a.status = 'pending'), 0)::bigint AS oldest_approval_unix,
  coalesce((SELECT extract(epoch FROM min(coalesce(c.scheduled_at, CASE WHEN coalesce(c.schedule->>'at', '') <> '' THEN (c.schedule->>'at')::timestamptz END)))
     FROM campaigns c WHERE c.workspace_id = w.id AND c.deleted_at IS NULL AND c.status IN ('scheduled', 'approval', 'approved')
       AND c.digest_role NOT IN ('template')
       AND coalesce(c.scheduled_at, CASE WHEN coalesce(c.schedule->>'at', '') <> '' THEN (c.schedule->>'at')::timestamptz END) >= @now), 0)::bigint AS next_send_unix,
  (SELECT count(*) FROM contacts ct WHERE ct.workspace_id = w.id AND ct.erased_at IS NULL)::bigint AS contacts
FROM workspaces w
WHERE w.id = ANY(@workspace_ids::uuid[]);

-- name: AgencyAutoPauses :many
-- A campaign's auto-pauses with how each ended.
SELECT ci.id, ci.campaign_id, ci.scope, ci.mailbox_provider, ci.metric, ci.value, ci.threshold, ci.held, ci.tripped_at,
       ci.resolved_at, ci.resolution, coalesce(u.name, '')::text AS resolved_by_name
FROM campaign_incidents ci
LEFT JOIN users u ON u.id = ci.resolved_by
WHERE ci.campaign_id = ANY(@campaign_ids::uuid[]) AND ci.kind = 'threshold'
ORDER BY ci.tripped_at DESC;

-- name: AgencyFailoversBetween :many
-- The streams that failed over while a provider was down.
SELECT f.stream, f.mode, coalesce(p.name, '')::text AS fallback_name
FROM stream_failovers f
LEFT JOIN providers p ON p.id = f.fallback_provider_id
WHERE f.provider_id = @provider_id AND f.started_at >= @from_at AND f.started_at <= @to_at
ORDER BY f.started_at;

-- name: AgencySetupFacts :one
-- What a new client has done of its setup.
SELECT
  (SELECT count(*) FROM providers p WHERE p.workspace_id = @workspace_id AND p.disconnected_at IS NULL)::bigint AS providers,
  (SELECT count(*) FROM sending_domains d WHERE d.workspace_id = @workspace_id AND d.status = 'verified')::bigint AS verified_domains,
  (SELECT coalesce(min(d.domain), '') FROM sending_domains d WHERE d.workspace_id = @workspace_id)::text AS first_domain,
  (SELECT count(*) FROM workspace_members m JOIN roles r ON r.id = m.role_id
     WHERE m.workspace_id = @workspace_id AND m.status = 'active' AND r.key IN ('client_approver', 'client_viewer'))::bigint AS client_members,
  (SELECT count(*) FROM invitations i JOIN roles r ON r.id = i.role_id
     WHERE i.workspace_id = @workspace_id AND i.accepted_at IS NULL AND i.revoked_at IS NULL AND i.expires_at > now()
       AND r.key IN ('client_approver', 'client_viewer'))::bigint AS client_invites;

-- ---------------------------------------------------------------------------
-- Search across clients

-- name: AgencySearchNames :many
-- Campaigns, automations, templates and forms whose name contains the
-- text, in the given workspaces.
SELECT x.workspace_id, x.kind, x.id, x.name, x.status, x.updated_at FROM (
    SELECT c.workspace_id, 'campaign'::text AS kind, c.id, c.name, c.status, c.updated_at
    FROM campaigns c WHERE c.workspace_id = ANY(sqlc.arg(workspace_ids)::uuid[]) AND c.deleted_at IS NULL AND c.digest_role NOT IN ('template', 'next')
      AND c.name ILIKE sqlc.arg(pattern)::text
    UNION ALL
    SELECT a.workspace_id, 'automation'::text, a.id, a.name, a.status, a.updated_at
    FROM automations a WHERE a.workspace_id = ANY(sqlc.arg(workspace_ids)::uuid[]) AND a.deleted_at IS NULL AND a.name ILIKE sqlc.arg(pattern)::text
    UNION ALL
    SELECT t.workspace_id, t.kind, t.id, t.name, 'saved'::text, t.updated_at
    FROM templates t WHERE t.workspace_id = ANY(sqlc.arg(workspace_ids)::uuid[]) AND t.deleted_at IS NULL AND t.name ILIKE sqlc.arg(pattern)::text
    UNION ALL
    SELECT fm.workspace_id, 'form'::text, fm.id, fm.name, fm.status, fm.updated_at
    FROM forms fm WHERE fm.workspace_id = ANY(sqlc.arg(workspace_ids)::uuid[]) AND fm.name ILIKE sqlc.arg(pattern)::text
) x
ORDER BY x.updated_at DESC
LIMIT 200;

-- name: AgencySearchLibrary :many
SELECT i.id, i.kind, i.name, i.version FROM agency_library_items i
WHERE i.deleted_at IS NULL AND i.name ILIKE @pattern
ORDER BY i.updated_at DESC
LIMIT 50;

-- name: AgencySearchContacts :many
-- Contacts with exactly this email, in the given workspaces.
SELECT c.id, c.workspace_id, c.email, c.created_at,
       coalesce((SELECT array_agg(l.name ORDER BY l.name) FROM list_memberships lm JOIN lists l ON l.id = lm.list_id AND l.deleted_at IS NULL
                 WHERE lm.contact_id = c.id AND lm.status = 'subscribed'), '{}')::text[] AS lists,
       coalesce((SELECT array_agg(l.name ORDER BY l.name) FROM list_memberships lm JOIN lists l ON l.id = lm.list_id AND l.deleted_at IS NULL
                 WHERE lm.contact_id = c.id AND lm.status = 'unsubscribed'), '{}')::text[] AS left_lists,
       coalesce((SELECT extract(epoch FROM min(lm.subscribed_at)) FROM list_memberships lm WHERE lm.contact_id = c.id), 0)::bigint AS subscribed_unix,
       coalesce((SELECT extract(epoch FROM max(lm.unsubscribed_at)) FROM list_memberships lm WHERE lm.contact_id = c.id AND lm.status = 'unsubscribed'), 0)::bigint AS unsubscribed_unix
FROM contacts c
WHERE c.workspace_id = ANY(@workspace_ids::uuid[]) AND c.erased_at IS NULL AND lower(c.email) = lower(@email)
ORDER BY c.created_at;

-- name: AgencyUserCandidates :many
-- Accounts that can join the staff: active, not staff yet, and not a
-- client's own people (client people never become agency staff).
SELECT u.id, u.name, u.email, u.avatar_color FROM users u
WHERE u.status = 'active' AND NOT u.is_instance_admin
  AND NOT EXISTS (SELECT 1 FROM agency_staff s WHERE s.user_id = u.id OR s.email = u.email)
  AND NOT EXISTS (SELECT 1 FROM workspace_members m JOIN roles r ON r.id = m.role_id
                  WHERE m.user_id = u.id AND r.key IN ('client_approver', 'client_viewer'))
ORDER BY u.name, u.email;

-- name: AgencyHoldsClientRole :one
SELECT EXISTS (SELECT 1 FROM workspace_members m JOIN roles r ON r.id = m.role_id
               WHERE m.user_id = @user_id AND r.key IN ('client_approver', 'client_viewer'));

-- name: AgencyUserByEmail :one
SELECT * FROM users WHERE email = @email;

-- name: AgencySetMemberSource :exec
UPDATE workspace_members SET source = @source WHERE id = @id;

-- ---------------------------------------------------------------------------
-- Workspace templates: reading a client's setup, and applying one

-- name: AgencyCaptureLists :many
SELECT id, name, description, double_opt_in FROM lists WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY created_at;

-- name: AgencyCaptureAutomations :many
-- Each flow's latest version (its draft, or the live one when there's no draft).
SELECT a.id, a.name, a.description, a.status, v.graph
FROM automations a
JOIN automation_versions v ON v.automation_id = a.id AND v.version = a.draft_version
WHERE a.workspace_id = @workspace_id AND a.deleted_at IS NULL
ORDER BY a.created_at;

-- name: AgencyCaptureAutomationEmails :many
SELECT e.id, e.automation_id, e.name, v.subject, v.preheader, v.mjml, v.html, v.plain_text
FROM automation_emails e
JOIN automation_email_versions v ON v.email_id = e.id AND v.version = e.latest_version
WHERE e.workspace_id = @workspace_id
ORDER BY e.created_at;

-- name: AgencyCaptureTemplates :many
SELECT id, kind, name, description, mjml, html, thumb
FROM templates WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY created_at;

-- name: AgencyCaptureForms :many
SELECT id, name, kind, list_ids, double_opt_in, fields, content, spam_protection, consent_text
FROM forms WHERE workspace_id = @workspace_id ORDER BY created_at;

-- name: AgencyCaptureRoles :many
SELECT id, key, name, description, builtin, permissions FROM roles WHERE workspace_id = @workspace_id ORDER BY created_at;

-- name: AgencySenderName :one
SELECT name FROM senders WHERE id = @id;

-- name: AgencyAutomationNameTaken :one
SELECT EXISTS (SELECT 1 FROM automations WHERE workspace_id = @workspace_id AND lower(name) = lower(@name) AND deleted_at IS NULL);

-- name: AgencyFormNameTaken :one
SELECT EXISTS (SELECT 1 FROM forms WHERE workspace_id = @workspace_id AND lower(name) = lower(@name));

-- name: AgencyAddBrandingKeys :exec
-- Adds the keys the workspace's branding doesn't have; never overwrites.
UPDATE workspaces SET branding = @patch::jsonb || branding, updated_at = now() WHERE id = @id;

-- name: AgencyAddSendingDefaults :exec
UPDATE workspaces SET sending_defaults = @patch::jsonb || sending_defaults, updated_at = now() WHERE id = @id;

-- name: AgencyInsertTemplate :exec
INSERT INTO templates (id, workspace_id, kind, name, description, mjml, html, thumb, size_bytes, created_by, updated_by, created_at, updated_at)
VALUES (@id, @workspace_id, @kind, @name, @description, @mjml, @html, @thumb, @size_bytes, sqlc.narg(created_by), sqlc.narg(created_by), @at, @at);

-- name: AgencyUpdateTemplateContent :execrows
UPDATE templates SET mjml = @mjml, html = @html, thumb = @thumb, size_bytes = @size_bytes, updated_by = sqlc.narg(updated_by), updated_at = @at
WHERE id = @id AND workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: AgencyGetTemplate :one
SELECT id, kind, name, description, mjml, html, thumb, updated_at FROM templates WHERE id = @id AND workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: AgencyGetAutomation :one
SELECT a.id, a.name, a.description, a.status, v.graph, a.draft_version
FROM automations a
JOIN automation_versions v ON v.automation_id = a.id AND v.version = a.draft_version
WHERE a.id = @id AND a.workspace_id = @workspace_id AND a.deleted_at IS NULL;

-- name: AgencyAutomationEmailsOf :many
SELECT e.id, e.name, v.subject, v.preheader, v.mjml, v.html, v.plain_text
FROM automation_emails e
JOIN automation_email_versions v ON v.email_id = e.id AND v.version = e.latest_version
WHERE e.automation_id = @automation_id
ORDER BY e.created_at;

-- name: AgencyGetTransactionalTemplate :one
SELECT id, key, name, description, subject, preheader, editor_mode, mjml, html, plain_text, variables, test_data
FROM transactional_templates WHERE id = @id AND workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: AgencyCopiesOfWorkspace :many
-- Library copies in a workspace with the item's latest version.
SELECT c.id, c.item_id, c.version, c.target_kind, c.target_id, i.version AS latest, i.name, i.kind
FROM agency_library_copies c
JOIN agency_library_items i ON i.id = c.item_id AND i.deleted_at IS NULL
WHERE c.workspace_id = @workspace_id;
