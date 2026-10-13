-- name: ReadinessCampaign :one
-- A campaign with its sender and sending domain.
SELECT c.id, c.workspace_id, c.name, c.status, c.subject, c.preheader, c.audience, c.audience_count, c.mjml, c.html, c.version,
       c.ab_test, c.schedule, c.thresholds, c.route_override, c.content_hash, c.content_settings, c.reply_to,
       c.created_by, c.updated_by, c.created_at, c.updated_at, c.autosaved_at, c.scheduled_at, c.started_at, c.sender_id,
       coalesce(s.name, '')::text AS sender_name,
       coalesce(s.email, '')::text AS sender_email,
       coalesce(d.domain, '')::text AS sender_domain,
       coalesce(u.name, '')::text AS updated_by_name
FROM campaigns c
LEFT JOIN senders s ON s.id = c.sender_id
LEFT JOIN sending_domains d ON d.id = s.domain_id
LEFT JOIN users u ON u.id = c.updated_by
WHERE c.workspace_id = @workspace_id AND c.id = @id AND c.deleted_at IS NULL;

-- name: ReadinessRecipients :many
-- The audience before suppressions and exclusions by address: active
-- contacts subscribed to the lists (or in the segments), minus excluded lists
-- and segments, with what the counts need per contact.
SELECT c.id, c.email, c.mailbox_provider, c.tier,
       coalesce((SELECT array_agg(m.list_id) FROM list_memberships m WHERE m.contact_id = c.id AND m.status = 'subscribed'), '{}')::uuid[] AS list_ids,
       EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = c.workspace_id AND s.email = c.email) AS suppressed,
       EXISTS (SELECT 1 FROM delivery_events e WHERE e.contact_id = c.id AND e.type IN ('soft_bounce', 'hard_bounce') AND e.occurred_at > @bounced_since::timestamptz) AS bounced
FROM contacts c
WHERE c.workspace_id = @workspace_id AND c.status = 'active' AND c.erased_at IS NULL
  AND (EXISTS (SELECT 1 FROM list_memberships m WHERE m.contact_id = c.id AND m.status = 'subscribed' AND m.list_id = ANY(@lists::uuid[]))
       OR c.id = ANY(@contacts::uuid[]))
  AND NOT EXISTS (SELECT 1 FROM list_memberships x WHERE x.contact_id = c.id AND x.status = 'subscribed' AND x.list_id = ANY(@exclude_lists::uuid[]))
  AND NOT (c.id = ANY(@exclude_contacts::uuid[]));

-- name: ReadinessListNames :many
SELECT id, name, requires_approval FROM lists WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY created_at;

-- name: ReadinessSegmentNames :many
SELECT id, name FROM segments WHERE workspace_id = @workspace_id ORDER BY name;

-- name: ReadinessListCount :one
-- Lists of the workspace, to tell "All subscribers" from a few lists.
SELECT count(*)::bigint FROM lists WHERE workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: ReadinessInsertRun :exec
INSERT INTO readiness_runs (id, campaign_id, trigger, verdict, checks, estimates, ran_at, duration_ms, ran_by, campaign_version, fingerprint)
VALUES (@id, @campaign_id, @trigger, @verdict, @checks, @estimates, @ran_at, @duration_ms, sqlc.narg(ran_by), @campaign_version, @fingerprint);

-- name: ReadinessLatestRun :one
SELECT rr.id, rr.campaign_id, rr.trigger, rr.verdict, rr.checks, rr.estimates, rr.ran_at, rr.duration_ms, rr.ran_by, rr.campaign_version,
       rr.fingerprint, rr.unknown_accepted_at, rr.unknown_accepted_by,
       coalesce(u.name, '')::text AS ran_by_name,
       coalesce(a.name, '')::text AS accepted_by_name
FROM readiness_runs rr
LEFT JOIN users u ON u.id = rr.ran_by
LEFT JOIN users a ON a.id = rr.unknown_accepted_by
WHERE rr.campaign_id = @campaign_id
ORDER BY rr.ran_at DESC
LIMIT 1;

-- name: ReadinessGetRun :one
SELECT rr.id, rr.campaign_id, rr.trigger, rr.verdict, rr.checks, rr.estimates, rr.ran_at, rr.duration_ms, rr.ran_by, rr.campaign_version,
       rr.fingerprint, rr.unknown_accepted_at, rr.unknown_accepted_by,
       coalesce(u.name, '')::text AS ran_by_name,
       coalesce(a.name, '')::text AS accepted_by_name
FROM readiness_runs rr
LEFT JOIN users u ON u.id = rr.ran_by
LEFT JOIN users a ON a.id = rr.unknown_accepted_by
WHERE rr.id = @id AND rr.campaign_id = @campaign_id;

-- name: ReadinessAcceptUnknown :exec
UPDATE readiness_runs SET unknown_accepted_at = @accepted_at, unknown_accepted_by = @accepted_by WHERE id = @id;

-- name: ReadinessPruneRuns :exec
-- Keeps the 20 latest runs of a campaign and any run an approval refers to.
DELETE FROM readiness_runs r
WHERE r.campaign_id = @campaign_id
  AND r.id NOT IN (SELECT x.id FROM readiness_runs x WHERE x.campaign_id = @campaign_id ORDER BY x.ran_at DESC LIMIT 20)
  AND NOT EXISTS (SELECT 1 FROM approval_requests a WHERE a.readiness_run_id = r.id);

-- name: ReadinessStartProgress :one
-- Starts a run unless one is already running (a run that stopped updating
-- for two minutes is taken over).
INSERT INTO readiness_progress (campaign_id, run_id, trigger, started_by, started_at, updated_at, groups, done, total)
VALUES (@campaign_id, @run_id, @trigger, sqlc.narg(started_by), @now, @now, @groups, 0, @total)
ON CONFLICT (campaign_id) DO UPDATE
SET run_id = EXCLUDED.run_id, trigger = EXCLUDED.trigger, started_by = EXCLUDED.started_by, started_at = EXCLUDED.started_at,
    updated_at = EXCLUDED.updated_at, groups = EXCLUDED.groups, done = 0, total = EXCLUDED.total
WHERE readiness_progress.updated_at < @stale_before::timestamptz
RETURNING run_id;

-- name: ReadinessUpdateProgress :exec
UPDATE readiness_progress SET groups = @groups, done = @done, total = @total, updated_at = @now
WHERE campaign_id = @campaign_id AND run_id = @run_id;

-- name: ReadinessEndProgress :exec
DELETE FROM readiness_progress WHERE campaign_id = @campaign_id AND run_id = @run_id;

-- name: ReadinessGetProgress :one
SELECT p.campaign_id, p.run_id, p.trigger, p.started_by, p.started_at, p.updated_at, p.groups, p.done, p.total,
       coalesce(u.name, '')::text AS started_by_name
FROM readiness_progress p
LEFT JOIN users u ON u.id = p.started_by
WHERE p.campaign_id = @campaign_id;

-- name: ReadinessReplaceRoutes :exec
-- The planned split of an unsent campaign, refreshed by each run.
DELETE FROM campaign_routes WHERE campaign_id = @campaign_id AND status = 'pending';

-- name: ReadinessInsertRoute :exec
INSERT INTO campaign_routes (campaign_id, provider_id, planned, status, cost_micros)
VALUES (@campaign_id, @provider_id, @planned, 'pending', @cost_micros)
ON CONFLICT (campaign_id, provider_id) DO UPDATE SET planned = EXCLUDED.planned, cost_micros = EXCLUDED.cost_micros
WHERE campaign_routes.status = 'pending';

-- name: ReadinessSetAudience :exec
UPDATE campaigns SET audience = @audience, audience_count = sqlc.narg(audience_count), updated_by = sqlc.narg(updated_by), updated_at = @now
WHERE workspace_id = @workspace_id AND id = @id;

-- name: ReadinessSetAudienceCount :exec
UPDATE campaigns SET audience_count = @audience_count WHERE id = @id AND audience_count IS DISTINCT FROM @audience_count::bigint;

-- name: ReadinessSetRouteOverride :exec
UPDATE campaigns SET route_override = sqlc.narg(route_override), updated_by = sqlc.narg(updated_by), updated_at = @now
WHERE workspace_id = @workspace_id AND id = @id;

-- name: ReadinessSetContentSettings :exec
UPDATE campaigns SET content_settings = @content_settings WHERE workspace_id = @workspace_id AND id = @id;

-- name: ReadinessSetSendSettings :exec
UPDATE campaigns SET thresholds = @thresholds, schedule = @schedule, updated_by = sqlc.narg(updated_by), updated_at = @now
WHERE workspace_id = @workspace_id AND id = @id;

-- name: ReadinessTestSends :many
-- The latest test emails of a campaign, from the audit log.
SELECT occurred_at, actor_name, after, metadata
FROM audit_events
WHERE workspace_id = @workspace_id AND action = 'campaign.test_send' AND target_id = @campaign_id::text AND result = 'allowed'
ORDER BY occurred_at DESC
LIMIT 6;

-- name: ReadinessScheduledDue :many
-- Scheduled campaigns starting within the window whose latest run is
-- older than the window (checks run again 15 minutes before the send).
SELECT c.id, c.workspace_id, c.scheduled_at
FROM campaigns c
JOIN workspaces w ON w.id = c.workspace_id AND w.deleted_at IS NULL
WHERE c.deleted_at IS NULL AND c.status = 'scheduled' AND c.scheduled_at IS NOT NULL
  AND c.scheduled_at <= @until::timestamptz
  AND NOT EXISTS (SELECT 1 FROM readiness_runs r WHERE r.campaign_id = c.id AND r.ran_at >= c.scheduled_at - @lead::interval)
ORDER BY c.scheduled_at;
