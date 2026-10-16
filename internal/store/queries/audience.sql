-- Audience: lists, custom fields, contacts, consent, saved views,
-- suppressions and the daily rollups behind List health.

-- name: AudienceCounts :one
SELECT count(*)::bigint AS contacts,
       count(*) FILTER (WHERE tier = 'active')::bigint AS active,
       count(*) FILTER (WHERE tier = 'cooling')::bigint AS cooling,
       count(*) FILTER (WHERE tier = 'at_risk')::bigint AS at_risk,
       count(*) FILTER (WHERE tier = 'sunset')::bigint AS sunset
FROM contacts
WHERE workspace_id = @workspace_id AND erased_at IS NULL;

-- name: CountWorkspaceSuppressions :one
SELECT count(*)::bigint FROM suppressions WHERE workspace_id = @workspace_id;

-- name: ListListsWithCounts :many
SELECT l.id, l.name, l.description, l.double_opt_in, l.created_at,
       coalesce(m.subscribed, 0)::bigint AS subscribed,
       coalesce(m.pending, 0)::bigint AS pending
FROM lists l
LEFT JOIN LATERAL (
    SELECT count(*) FILTER (WHERE lm.status = 'subscribed') AS subscribed,
           count(*) FILTER (WHERE lm.status = 'pending') AS pending
    FROM list_memberships lm WHERE lm.list_id = l.id
) m ON true
WHERE l.workspace_id = @workspace_id AND l.deleted_at IS NULL
ORDER BY l.created_at, l.id;

-- name: GetList :one
SELECT * FROM lists WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL;

-- name: ListNameTaken :one
SELECT EXISTS (SELECT 1 FROM lists WHERE workspace_id = @workspace_id AND lower(name) = lower(@name) AND deleted_at IS NULL AND id <> @except_id)::boolean;

-- name: CreateList :one
INSERT INTO lists (id, workspace_id, name, description, double_opt_in)
VALUES (@id, @workspace_id, @name, @description, @double_opt_in)
RETURNING *;

-- name: UpdateList :one
UPDATE lists SET name = @name, description = @description, double_opt_in = @double_opt_in
WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteList :execrows
-- The list goes from every screen and send; paused notes what deleting it
-- paused, for a restore.
UPDATE lists SET deleted_at = @at, delete_paused = @paused
WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL;

-- name: LockList :one
SELECT * FROM lists WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL FOR UPDATE;

-- name: LockDeletedList :one
SELECT * FROM lists WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NOT NULL FOR UPDATE;

-- name: RestoreList :one
UPDATE lists SET deleted_at = NULL, delete_paused = '{}' WHERE id = @id RETURNING *;

-- name: ListDeletedAt :one
SELECT deleted_at FROM lists WHERE workspace_id = @workspace_id AND id = @id;

-- name: PurgeDeletedLists :execrows
-- Past the 30 days a deleted list can be restored: the list and its
-- memberships go for good. Contacts and suppressions stay.
DELETE FROM lists WHERE deleted_at < @before;

-- name: ListImpactMembers :one
-- The list's subscribers, and how many of them another list keeps
-- subscribed.
SELECT count(*)::bigint AS members,
       count(*) FILTER (WHERE EXISTS (
           SELECT 1 FROM list_memberships o JOIN lists ol ON ol.id = o.list_id AND ol.deleted_at IS NULL
           WHERE o.contact_id = m.contact_id AND o.list_id <> m.list_id AND o.status = 'subscribed'))::bigint AS elsewhere
FROM list_memberships m
WHERE m.list_id = @id AND m.status = 'subscribed';

-- name: ListImpactOtherLists :many
-- The other lists the subscribers are on, most shared first.
SELECT ol.id, ol.name, count(*)::bigint AS contacts
FROM list_memberships m
JOIN list_memberships o ON o.contact_id = m.contact_id AND o.list_id <> m.list_id AND o.status = 'subscribed'
JOIN lists ol ON ol.id = o.list_id AND ol.deleted_at IS NULL
WHERE m.list_id = @id AND m.status = 'subscribed'
GROUP BY ol.id, ol.name
ORDER BY contacts DESC, ol.name;

-- name: ListImpactForms :many
-- Live signup forms that subscribe people to the list.
SELECT id, name FROM forms
WHERE workspace_id = @workspace_id AND status = 'live' AND @id::uuid = ANY(list_ids)
ORDER BY name, id;

-- name: ListImpactAutomations :many
-- Live automations that start when someone joins the list.
SELECT id, name FROM automations
WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND status = 'live'
  AND trigger ->> 'kind' = 'joined_list' AND trigger ->> 'listId' = @id::text
ORDER BY name, id;

-- name: ListImpactCampaigns :many
-- Scheduled campaigns that send to the list.
SELECT id, name, scheduled_at FROM campaigns
WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND status = 'scheduled'
  AND audience -> 'lists' @> jsonb_build_array(@id::text)
ORDER BY scheduled_at, name, id;

-- name: ListPauseForm :execrows
UPDATE forms SET status = 'paused', updated_by = sqlc.narg(by), updated_at = @at
WHERE workspace_id = @workspace_id AND id = @id AND status = 'live';

-- name: ListResumeForm :execrows
-- Only a form still paused the way deleting the list left it.
UPDATE forms SET status = 'live', updated_by = sqlc.narg(by), updated_at = @at
WHERE workspace_id = @workspace_id AND id = @id AND status = 'paused' AND updated_at = @paused_at;

-- name: ListPauseAutomation :execrows
UPDATE automations SET status = 'paused', paused_at = @at, paused_by = sqlc.narg(by), pause_reason = @reason,
    updated_by = sqlc.narg(by), updated_at = @at
WHERE workspace_id = @workspace_id AND id = @id AND status = 'live' AND deleted_at IS NULL;

-- name: ListResumeAutomation :execrows
UPDATE automations SET status = 'live', paused_at = NULL, paused_by = NULL, pause_reason = '',
    updated_by = sqlc.narg(by), updated_at = @at
WHERE workspace_id = @workspace_id AND id = @id AND status = 'paused' AND paused_at = @paused_at AND deleted_at IS NULL;

-- name: ListUnscheduleCampaign :execrows
UPDATE campaigns SET status = @status, scheduled_at = NULL, updated_by = coalesce(sqlc.narg(by), updated_by), updated_at = @at
WHERE workspace_id = @workspace_id AND id = @id AND status = 'scheduled';

-- name: ListRescheduleCampaign :execrows
-- Only a campaign nobody changed since deleting the list took it off the
-- schedule.
UPDATE campaigns SET status = 'scheduled', scheduled_at = @scheduled_at, updated_by = coalesce(sqlc.narg(by), updated_by), updated_at = @at
WHERE workspace_id = @workspace_id AND id = @id AND status = @status AND scheduled_at IS NULL AND updated_at = @unscheduled_at AND deleted_at IS NULL;

-- name: ListCustomFields :many
SELECT * FROM custom_fields WHERE workspace_id = @workspace_id ORDER BY position, created_at, key;

-- name: GetCustomField :one
SELECT * FROM custom_fields WHERE workspace_id = @workspace_id AND id = @id;

-- name: CreateCustomField :one
INSERT INTO custom_fields (id, workspace_id, key, label, type, options, display, position)
VALUES (@id, @workspace_id, @key, @label, @type, @options, @display,
        (SELECT coalesce(max(position), 0) + 1 FROM custom_fields WHERE workspace_id = @workspace_id))
RETURNING *;

-- name: UpdateCustomField :one
UPDATE custom_fields SET label = @label, options = @options, display = @display
WHERE workspace_id = @workspace_id AND id = @id
RETURNING *;

-- name: DeleteCustomField :execrows
DELETE FROM custom_fields WHERE workspace_id = @workspace_id AND id = @id;

-- name: GetContact :one
SELECT * FROM contacts WHERE workspace_id = @workspace_id AND id = @id AND erased_at IS NULL;

-- name: GetContactByEmail :one
SELECT * FROM contacts WHERE workspace_id = @workspace_id AND email = @email AND erased_at IS NULL;

-- name: ContactListNames :many
-- Lists a page of contacts belongs to (subscribed or pending), in list order.
SELECT lm.contact_id, l.name, lm.status
FROM list_memberships lm
JOIN lists l ON l.id = lm.list_id AND l.deleted_at IS NULL
WHERE lm.contact_id = ANY(@contact_ids::uuid[]) AND lm.status <> 'unsubscribed'
ORDER BY lm.contact_id, l.created_at, l.id;

-- name: ContactMemberships :many
SELECT l.id AS list_id, l.name, l.double_opt_in, lm.status, lm.source, lm.subscribed_at, lm.unsubscribed_at, lm.created_at
FROM list_memberships lm
JOIN lists l ON l.id = lm.list_id AND l.deleted_at IS NULL
WHERE lm.contact_id = @contact_id
ORDER BY l.created_at, l.id;

-- name: ContactConsentEvents :many
SELECT ce.id, ce.list_id, ce.action, ce.source, ce.ip, ce.user_agent, ce.detail, ce.note, ce.consent_version, ce.language, ce.occurred_at,
       coalesce(u.name, '')::text AS actor_name
FROM consent_events ce
LEFT JOIN users u ON u.id = ce.actor_id
WHERE ce.contact_id = @contact_id
ORDER BY ce.occurred_at, ce.id;

-- name: GetContactPreferences :one
SELECT * FROM contact_preferences WHERE contact_id = @contact_id;

-- name: ContactChoseFewer :one
SELECT EXISTS (SELECT 1 FROM unsubscribe_feedback WHERE contact_id = @contact_id AND chose_fewer)::boolean;

-- name: ContactEngagementSince :one
-- Emails sent to a contact since a date (Broadcasts and Automations), how
-- many were opened by a person and how many were clicked.
SELECT count(*)::bigint AS emails,
       count(*) FILTER (WHERE EXISTS (SELECT 1 FROM delivery_events de WHERE de.message_id = m.id AND de.type = 'open' AND NOT de.machine))::bigint AS human_opened,
       count(*) FILTER (WHERE EXISTS (SELECT 1 FROM delivery_events de WHERE de.message_id = m.id AND de.type = 'click'))::bigint AS clicked
FROM messages m
WHERE m.contact_id = @contact_id AND m.stream IN ('broadcast', 'automation') AND m.queued_at >= @since;

-- name: ContactActivityCount :one
SELECT ((SELECT count(*) FROM delivery_events de WHERE de.contact_id = @contact_id AND de.type <> 'sent')
      + (SELECT count(*) FROM consent_events ce WHERE ce.contact_id = @contact_id))::bigint;

-- name: InsertConsentEvent :exec
INSERT INTO consent_events (id, contact_id, list_id, action, source, ip, user_agent, detail, note, consent_version, actor_id, import_id, occurred_at)
VALUES (@id, @contact_id, @list_id, @action, @source, @ip, @user_agent, @detail, @note, @consent_version, @actor_id, @import_id, @occurred_at);

-- name: CreateContact :one
INSERT INTO contacts (id, workspace_id, email, first_name, last_name, company, country, timezone, language,
                      mailbox_provider, mailbox_detail, consent, source, tags, attributes, role_address, last_activity_at)
VALUES (@id, @workspace_id, @email, @first_name, @last_name, @company, @country, @timezone, @language,
        @mailbox_provider, @mailbox_detail, @consent, @source, @tags, @attributes, @role_address, now())
RETURNING *;

-- name: UpdateContactFields :one
UPDATE contacts
SET first_name = @first_name, last_name = @last_name, company = @company, country = @country, timezone = @timezone,
    language = @language, tags = @tags, attributes = @attributes, updated_at = now()
WHERE workspace_id = @workspace_id AND id = @id AND erased_at IS NULL
RETURNING *;

-- name: UpsertMembership :exec
INSERT INTO list_memberships (list_id, contact_id, status, source, subscribed_at)
VALUES (@list_id, @contact_id, @status, @source, CASE WHEN @status::text = 'subscribed' THEN now() END)
ON CONFLICT (list_id, contact_id) DO UPDATE
SET status = excluded.status, source = excluded.source,
    subscribed_at = CASE WHEN excluded.status = 'subscribed' AND list_memberships.status <> 'subscribed' THEN now() ELSE list_memberships.subscribed_at END,
    unsubscribed_at = CASE WHEN excluded.status = 'subscribed' THEN NULL ELSE list_memberships.unsubscribed_at END;

-- name: ListSavedViews :many
SELECT * FROM saved_views
WHERE workspace_id = @workspace_id AND (user_id IS NULL OR user_id = @user_id)
ORDER BY position, created_at, id;

-- name: CreateSavedView :one
INSERT INTO saved_views (id, workspace_id, user_id, name, filters, columns, position, show_in_health)
VALUES (@id, @workspace_id, @user_id, @name, @filters, '{}',
        (SELECT coalesce(max(position), 0) + 1 FROM saved_views WHERE workspace_id = @workspace_id), false)
RETURNING *;

-- name: GetSavedView :one
SELECT * FROM saved_views WHERE workspace_id = @workspace_id AND id = @id;

-- name: DeleteSavedView :execrows
DELETE FROM saved_views WHERE workspace_id = @workspace_id AND id = @id;

-- name: HealthViews :many
SELECT * FROM saved_views WHERE workspace_id = @workspace_id AND show_in_health ORDER BY position, created_at;

-- name: SuppressionCounts :one
SELECT count(*)::bigint AS total,
       count(*) FILTER (WHERE reason = 'hard_bounce')::bigint AS hard_bounce,
       count(*) FILTER (WHERE reason = 'complaint')::bigint AS complaint,
       count(*) FILTER (WHERE reason = 'unsubscribe')::bigint AS unsubscribe,
       count(*) FILTER (WHERE reason = 'manual')::bigint AS manual,
       count(*) FILTER (WHERE reason NOT IN ('hard_bounce', 'complaint', 'unsubscribe', 'manual'))::bigint AS other
FROM suppressions WHERE workspace_id = @workspace_id;

-- name: GetSuppression :one
SELECT * FROM suppressions WHERE workspace_id = @workspace_id AND id = @id;

-- name: GetSuppressionByEmail :one
SELECT * FROM suppressions WHERE workspace_id = @workspace_id AND email = @email;

-- name: InsertSuppression :one
INSERT INTO suppressions (id, workspace_id, email, reason, source, source_provider_id, original_code, detail, created_by,
                          scope, campaign_id, automation_id, automation_step, context, import_id, created_at)
VALUES (@id, @workspace_id, @email, @reason, @source, @source_provider_id, @original_code, @detail, @created_by,
        @scope, @campaign_id, @automation_id, @automation_step, @context, @import_id, @created_at)
ON CONFLICT (workspace_id, email) DO NOTHING
RETURNING *;

-- name: DeleteSuppression :execrows
DELETE FROM suppressions WHERE workspace_id = @workspace_id AND id = @id;

-- name: SuppressionSyncStates :many
-- One row per connected provider with its suppression sync state (kept by
-- the providers domain) and, for providers without a suppression API, the
-- addresses their bounce and complaint events added to the shared list.
SELECT p.id, p.name, p.adapter, p.label, p.capabilities, p.last_event_at,
       coalesce(s.mode, 'off')::text AS mode, s.last_push_at, s.last_pull_at,
       coalesce(s.pushed, 0)::bigint AS pushed, coalesce(s.pulled, 0)::bigint AS pulled,
       coalesce(s.pending, 0)::bigint AS pending,
       coalesce(s.status, '')::text AS status, coalesce(s.status_detail, '')::text AS status_detail,
       (s.provider_id IS NOT NULL)::boolean AS has_sync,
       coalesce(a.n, 0)::bigint AS added, coalesce(a.last_at, p.connected_at)::timestamptz AS added_at
FROM providers p
LEFT JOIN provider_suppression_sync s ON s.provider_id = p.id
LEFT JOIN LATERAL (
    SELECT count(*) AS n, max(x.created_at) AS last_at FROM suppressions x
    WHERE x.workspace_id = p.workspace_id AND x.source_provider_id = p.id
) a ON true
WHERE p.workspace_id = @workspace_id AND p.disconnected_at IS NULL
ORDER BY p.connected_at, p.name;

-- name: WorkspaceProviders :many
SELECT id, name, adapter FROM providers
WHERE workspace_id = @workspace_id AND disconnected_at IS NULL
ORDER BY connected_at, name;

-- name: GetAudienceState :one
SELECT * FROM audience_state WHERE workspace_id = @workspace_id;

-- name: AudienceAutomationByName :one
-- The automation the sunset policy names (List health links to its flow).
SELECT id FROM automations
WHERE workspace_id = @workspace_id AND name = @name AND deleted_at IS NULL
ORDER BY created_at
LIMIT 1;

-- name: UpsertAudienceState :exec
INSERT INTO audience_state (workspace_id, tiers_computed_at, reengaged, won_back, updated_at)
VALUES (@workspace_id, @tiers_computed_at, @reengaged, @won_back, now())
ON CONFLICT (workspace_id) DO UPDATE
SET tiers_computed_at = excluded.tiers_computed_at, reengaged = excluded.reengaged, won_back = excluded.won_back, updated_at = now();

-- name: WorkspaceTierHistory :many
SELECT day, contacts, active, cooling, at_risk, sunset
FROM audience_daily
WHERE workspace_id = @workspace_id AND scope = 'workspace' AND day >= @since
ORDER BY day;

-- name: ScopeTotalsSince :many
-- 30-day sums per list or followed view.
SELECT scope, scope_id,
       coalesce(sum(joined), 0)::bigint AS joined, coalesce(sum(departed), 0)::bigint AS departed,
       coalesce(sum(sent), 0)::bigint AS sent, coalesce(sum(delivered), 0)::bigint AS delivered,
       coalesce(sum(opened_human), 0)::bigint AS opened_human, coalesce(sum(unsubscribed), 0)::bigint AS unsubscribed,
       coalesce(sum(complained), 0)::bigint AS complained, coalesce(sum(bounced), 0)::bigint AS bounced
FROM audience_daily
WHERE workspace_id = @workspace_id AND scope IN ('list', 'view') AND day >= @since
GROUP BY scope, scope_id;

-- name: UpsertAudienceDaily :exec
INSERT INTO audience_daily (workspace_id, scope, scope_id, day, contacts, active, cooling, at_risk, sunset,
                            joined, departed, sent, delivered, opened_human, unsubscribed, complained, bounced)
VALUES (@workspace_id, @scope, @scope_id, @day, @contacts, @active, @cooling, @at_risk, @sunset,
        @joined, @departed, @sent, @delivered, @opened_human, @unsubscribed, @complained, @bounced)
ON CONFLICT (workspace_id, scope, scope_id, day) DO UPDATE
SET contacts = excluded.contacts, active = excluded.active, cooling = excluded.cooling, at_risk = excluded.at_risk,
    sunset = excluded.sunset, joined = excluded.joined, departed = excluded.departed, sent = excluded.sent,
    delivered = excluded.delivered, opened_human = excluded.opened_human, unsubscribed = excluded.unsubscribed,
    complained = excluded.complained, bounced = excluded.bounced;

-- name: ListTierMix :many
-- Tier mix and size of each list (subscribed members).
SELECT lm.list_id,
       count(*)::bigint AS contacts,
       count(*) FILTER (WHERE c.tier = 'active')::bigint AS active,
       count(*) FILTER (WHERE c.tier = 'cooling')::bigint AS cooling,
       count(*) FILTER (WHERE c.tier = 'at_risk')::bigint AS at_risk,
       count(*) FILTER (WHERE c.tier = 'sunset')::bigint AS sunset
FROM list_memberships lm
JOIN lists l ON l.id = lm.list_id AND l.workspace_id = @workspace_id AND l.deleted_at IS NULL
JOIN contacts c ON c.id = lm.contact_id AND c.erased_at IS NULL
WHERE lm.status = 'subscribed'
GROUP BY lm.list_id;

-- name: ListGrowthSince :many
SELECT lm.list_id,
       count(*) FILTER (WHERE lm.subscribed_at >= @since AND lm.status = 'subscribed')::bigint AS joined,
       count(*) FILTER (WHERE lm.unsubscribed_at >= @since)::bigint AS departed
FROM list_memberships lm
JOIN lists l ON l.id = lm.list_id AND l.workspace_id = @workspace_id AND l.deleted_at IS NULL
GROUP BY lm.list_id;

-- name: SetSunsetPolicy :exec
UPDATE workspaces SET sunset_policy = @sunset_policy, updated_at = now() WHERE id = @id;

-- name: TierRunWorkspaces :many
-- Workspaces with their timezone and the last tier run, for the nightly job.
SELECT w.id, w.slug, w.timezone, w.sunset_policy, s.tiers_computed_at
FROM workspaces w
LEFT JOIN audience_state s ON s.workspace_id = w.id
WHERE w.deleted_at IS NULL;
