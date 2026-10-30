-- name: ReadinessCampaign :one
-- A campaign with its sender and sending domain.
SELECT c.id, c.workspace_id, c.name, c.status, c.subject, c.preheader, c.audience, c.audience_count, c.mjml, c.html, c.version,
       c.ab_test, c.schedule, c.thresholds, c.route_override, c.content_hash, c.content_settings, c.reply_to, c.tracking,
       c.created_by, c.updated_by, c.created_at, c.updated_at, c.autosaved_at, c.scheduled_at, c.started_at, c.sender_id,
       coalesce(s.name, '')::text AS sender_name,
       coalesce(s.email, '')::text AS sender_email,
       coalesce(d.domain, '')::text AS sender_domain,
       coalesce(u.name, '')::text AS updated_by_name,
       coalesce(s.reply_to, '')::text AS sender_reply_to
FROM campaigns c
LEFT JOIN senders s ON s.id = c.sender_id
LEFT JOIN sending_domains d ON d.id = s.domain_id
LEFT JOIN users u ON u.id = c.updated_by
WHERE c.workspace_id = @workspace_id AND c.id = @id AND c.deleted_at IS NULL;

-- name: ReadinessRecipients :many
-- The audience before suppressions and exclusions by address: active
-- contacts subscribed to the lists (or in the segments), minus excluded lists
-- and segments, with what the counts need per contact. A deleted list
-- (restorable for 30 days) adds nobody; as an exclusion it still excludes.
-- Each contact's lists are read only with with_lists (routing needs them,
-- counts don't).
SELECT c.id, c.email, c.mailbox_provider, c.tier, c.role_address, c.language,
       (c.broadcasts_stopped_at IS NOT NULL)::boolean AS broadcasts_stopped,
       (btrim(c.first_name) = '')::boolean AS no_first_name,
       (CASE WHEN @with_lists::boolean
             THEN coalesce((SELECT array_agg(m.list_id) FROM list_memberships m WHERE m.contact_id = c.id AND m.status = 'subscribed'), '{}')
             ELSE '{}' END)::uuid[] AS list_ids,
       EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = c.workspace_id AND s.email = c.email) AS suppressed,
       EXISTS (SELECT 1 FROM delivery_events e WHERE e.contact_id = c.id AND e.type IN ('soft_bounce', 'hard_bounce') AND e.occurred_at > @bounced_since::timestamptz) AS bounced
FROM contacts c
WHERE c.workspace_id = @workspace_id AND c.status = 'active' AND c.erased_at IS NULL
  AND (EXISTS (SELECT 1 FROM list_memberships m WHERE m.contact_id = c.id AND m.status = 'subscribed' AND m.list_id = ANY(ARRAY(SELECT ll.id FROM lists ll WHERE ll.id = ANY(@lists::uuid[]) AND ll.deleted_at IS NULL)))
       OR c.id = ANY(@contacts::uuid[]))
  AND NOT EXISTS (SELECT 1 FROM list_memberships x WHERE x.contact_id = c.id AND x.status = 'subscribed' AND x.list_id = ANY(@exclude_lists::uuid[]))
  AND NOT (c.id = ANY(@exclude_contacts::uuid[]));

-- name: ReadinessListNames :many
SELECT id, name, requires_approval, public, origin FROM lists WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY created_at;

-- name: ReadinessSegmentNames :many
SELECT id, name FROM segments WHERE workspace_id = @workspace_id ORDER BY name;

-- name: ReadinessListCount :one
-- Lists of the workspace, to tell "All subscribers" from a few lists.
SELECT count(*)::bigint FROM lists WHERE workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: ReadinessInsertRun :exec
-- ran_by is the person who asked; an API key or agent is ran_by_actor_*.
INSERT INTO readiness_runs (id, campaign_id, trigger, verdict, checks, estimates, ran_at, duration_ms, ran_by, ran_by_actor_type, ran_by_actor_name,
                            campaign_version, fingerprint)
VALUES (@id, @campaign_id, @trigger, @verdict, @checks, @estimates, @ran_at, @duration_ms, sqlc.narg(ran_by), @ran_by_actor_type, @ran_by_actor_name,
        @campaign_version, @fingerprint);

-- name: ReadinessLatestRun :one
SELECT rr.id, rr.campaign_id, rr.trigger, rr.verdict, rr.checks, rr.estimates, rr.ran_at, rr.duration_ms, rr.ran_by, rr.campaign_version,
       rr.fingerprint, rr.unknown_accepted_at, rr.unknown_accepted_by,
       coalesce(u.name, nullif(rr.ran_by_actor_name, ''), '')::text AS ran_by_name, rr.ran_by_actor_type,
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
       coalesce(u.name, nullif(rr.ran_by_actor_name, ''), '')::text AS ran_by_name, rr.ran_by_actor_type,
       coalesce(a.name, '')::text AS accepted_by_name
FROM readiness_runs rr
LEFT JOIN users u ON u.id = rr.ran_by
LEFT JOIN users a ON a.id = rr.unknown_accepted_by
WHERE rr.id = @id AND rr.campaign_id = @campaign_id;

-- name: ReadinessAcceptUnknown :exec
UPDATE readiness_runs SET unknown_accepted_at = @accepted_at, unknown_accepted_by = @accepted_by WHERE id = @id;

-- name: ReadinessSetFingerprint :exec
-- A run confirmed for a new schedule at send time describes the campaign as scheduled.
UPDATE readiness_runs SET fingerprint = @fingerprint WHERE id = @id;

-- name: ReadinessPruneRuns :exec
-- Keeps the 20 latest runs of a campaign and any run an approval refers to.
DELETE FROM readiness_runs r
WHERE r.campaign_id = @campaign_id
  AND r.id NOT IN (SELECT x.id FROM readiness_runs x WHERE x.campaign_id = @campaign_id ORDER BY x.ran_at DESC LIMIT 20)
  AND NOT EXISTS (SELECT 1 FROM approval_requests a WHERE a.readiness_run_id = r.id);

-- name: ReadinessStartProgress :one
-- Starts a run unless one is already waiting or running (a run that stopped
-- updating for two minutes, or failed, is taken over).
INSERT INTO readiness_progress (campaign_id, run_id, trigger, started_by, started_by_actor_type, started_by_actor_name, started_at, updated_at, groups, done, total)
VALUES (@campaign_id, @run_id, @trigger, sqlc.narg(started_by), @started_by_actor_type, @started_by_actor_name, @now, @now, @groups, 0, @total)
ON CONFLICT (campaign_id) DO UPDATE
SET run_id = EXCLUDED.run_id, trigger = EXCLUDED.trigger, started_by = EXCLUDED.started_by, started_by_actor_type = EXCLUDED.started_by_actor_type,
    started_by_actor_name = EXCLUDED.started_by_actor_name, started_at = EXCLUDED.started_at,
    updated_at = EXCLUDED.updated_at, groups = EXCLUDED.groups, done = 0, total = EXCLUDED.total, failed_at = NULL, error = ''
WHERE readiness_progress.updated_at < @stale_before::timestamptz OR readiness_progress.failed_at IS NOT NULL
RETURNING run_id;

-- name: ReadinessClaimProgress :one
-- A worker takes up a queued run: it still owns the campaign's row.
UPDATE readiness_progress SET updated_at = @now
WHERE campaign_id = @campaign_id AND run_id = @run_id AND failed_at IS NULL
RETURNING run_id;

-- name: ReadinessFailProgress :exec
-- A run stopped on an error: the row says why until the next run.
UPDATE readiness_progress SET failed_at = @now, error = @error, updated_at = @now
WHERE campaign_id = @campaign_id AND run_id = @run_id;

-- name: ReadinessUpdateProgress :exec
UPDATE readiness_progress SET groups = @groups, done = @done, total = @total, updated_at = @now
WHERE campaign_id = @campaign_id AND run_id = @run_id;

-- name: ReadinessEndProgress :exec
DELETE FROM readiness_progress WHERE campaign_id = @campaign_id AND run_id = @run_id;

-- name: ReadinessGetProgress :one
SELECT p.campaign_id, p.run_id, p.trigger, p.started_by, p.started_at, p.updated_at, p.groups, p.done, p.total,
       p.failed_at, p.error,
       coalesce(u.name, nullif(p.started_by_actor_name, ''), '')::text AS started_by_name, p.started_by_actor_type
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
  AND c.channel = 'email'
  AND c.scheduled_at <= @until::timestamptz
  AND NOT EXISTS (SELECT 1 FROM readiness_runs r WHERE r.campaign_id = c.id AND r.ran_at >= c.scheduled_at - @lead::interval)
  -- A recurring digest checks its own issues before they start (internal/digests).
  AND NOT EXISTS (SELECT 1 FROM digest_issues di WHERE di.campaign_id = c.id AND di.mode = 'auto')
ORDER BY c.scheduled_at;

-- name: ReadinessCampaignWorkspace :one
-- The workspace of a campaign (the sending engine asks by campaign id).
SELECT workspace_id FROM campaigns WHERE id = @id AND deleted_at IS NULL;

-- name: ReadinessLatestRunRef :one
-- The latest run's id, time and verdict: what the review screen's poll
-- compares with what it shows.
SELECT id, ran_at, verdict FROM readiness_runs WHERE campaign_id = @campaign_id ORDER BY ran_at DESC LIMIT 1;

-- name: ReadinessPostmasterDays :many
-- Gmail Postmaster Tools' last days for a sending domain (or its parent):
-- the spam rate and the authentication and TLS pass rates Google reports.
SELECT domain, day, spam_rate, spf_rate, dkim_rate, dmarc_rate, tls_rate
FROM postmaster_daily
WHERE workspace_id = @workspace_id AND domain = ANY(@domains::text[]) AND day >= @since::date
ORDER BY day DESC, domain;

-- name: ReadinessMediaSizes :many
-- The sizes of the media library's images an email shows.
SELECT id, size_bytes FROM media_assets WHERE workspace_id = @workspace_id AND id = ANY(@ids::uuid[]);

-- name: ReadinessConfirmSend :exec
-- What a person confirmed as they scheduled or started a send (SR-3): the
-- send-time recheck holds the send on anything they didn't.
INSERT INTO campaign_send_confirmations (campaign_id, workspace_id, run_id, confirmed_by, confirmed_at, scheduled_at, recipients, acknowledged, unknown,
    content_digest, sender_id, route_digest, audience_digest)
VALUES (@campaign_id, @workspace_id, sqlc.narg(run_id), sqlc.narg(confirmed_by), @confirmed_at, sqlc.narg(scheduled_at), @recipients, @acknowledged::text[],
    @unknown::text[], @content_digest, sqlc.narg(sender_id), @route_digest, @audience_digest)
ON CONFLICT (campaign_id) DO UPDATE SET run_id = EXCLUDED.run_id, confirmed_by = EXCLUDED.confirmed_by, confirmed_at = EXCLUDED.confirmed_at,
    scheduled_at = EXCLUDED.scheduled_at, recipients = EXCLUDED.recipients, acknowledged = EXCLUDED.acknowledged, unknown = EXCLUDED.unknown,
    content_digest = EXCLUDED.content_digest, sender_id = EXCLUDED.sender_id, route_digest = EXCLUDED.route_digest, audience_digest = EXCLUDED.audience_digest;

-- name: ReadinessSendConfirmation :one
SELECT * FROM campaign_send_confirmations WHERE campaign_id = @campaign_id;

-- name: ReadinessForgetConfirmation :exec
DELETE FROM campaign_send_confirmations WHERE campaign_id = @campaign_id;

-- name: ReadinessPostmasterConnected :one
-- Whether Gmail Postmaster Tools data comes in for the workspace at all
-- (recent days for any domain).
SELECT EXISTS (SELECT 1 FROM postmaster_daily WHERE workspace_id = @workspace_id AND day >= @since::date)::bool;

-- name: ReadinessConfirmCanary :exec
-- The canary choice confirmed with a send (C1).
UPDATE campaign_send_confirmations SET canary = @canary WHERE campaign_id = @campaign_id;

-- name: ReadinessInsertFixBatch :exec
-- A batch of one-click fixes applied together (review R1.U3): the parts of
-- the campaign it changed, before and after, so Undo can put them back.
INSERT INTO readiness_fix_batches (id, workspace_id, campaign_id, applied_by, applied_at, applied, applied_lang, blocked,
    version_before, version_after, route_changed, route_before, route_after, audience_changed, audience_before, audience_after,
    settings_changed, settings_before, settings_after)
VALUES (@id, @workspace_id, @campaign_id, sqlc.narg(applied_by), @applied_at, @applied, @applied_lang, @blocked,
    sqlc.narg(version_before), sqlc.narg(version_after), @route_changed, sqlc.narg(route_before), sqlc.narg(route_after), @audience_changed,
    sqlc.narg(audience_before), sqlc.narg(audience_after), @settings_changed, sqlc.narg(settings_before), sqlc.narg(settings_after));

-- name: ReadinessFixBatch :one
SELECT b.id, b.workspace_id, b.campaign_id, b.applied_by, b.applied_at, b.applied, b.applied_lang, b.blocked, b.version_before, b.version_after,
       b.route_changed, b.route_before, b.route_after, b.audience_changed, b.audience_before, b.audience_after,
       b.settings_changed, b.settings_before, b.settings_after, b.undone_at, b.undone_by,
       coalesce(u.name, '')::text AS applied_by_name
FROM readiness_fix_batches b LEFT JOIN users u ON u.id = b.applied_by
WHERE b.workspace_id = @workspace_id AND b.campaign_id = @campaign_id AND b.id = @id;

-- name: ReadinessLatestFixBatch :one
-- The campaign's latest batch of fixes: the only one Undo can put back.
SELECT b.id, b.workspace_id, b.campaign_id, b.applied_by, b.applied_at, b.applied, b.applied_lang, b.blocked, b.version_before, b.version_after,
       b.route_changed, b.route_before, b.route_after, b.audience_changed, b.audience_before, b.audience_after,
       b.settings_changed, b.settings_before, b.settings_after, b.undone_at, b.undone_by,
       coalesce(u.name, '')::text AS applied_by_name
FROM readiness_fix_batches b LEFT JOIN users u ON u.id = b.applied_by
WHERE b.campaign_id = @campaign_id
ORDER BY b.applied_at DESC, b.id DESC
LIMIT 1;

-- name: ReadinessClaimFixUndo :execrows
-- Claims a batch for an undo: one at a time, never twice.
UPDATE readiness_fix_batches SET undone_at = sqlc.narg(undone_at), undone_by = sqlc.narg(undone_by) WHERE id = @id AND undone_at IS NULL;

-- name: ReadinessReleaseFixUndo :execrows
-- Gives a claimed batch back when its undo failed.
UPDATE readiness_fix_batches SET undone_at = NULL, undone_by = NULL WHERE id = @id;

-- name: ReadinessLockCampaign :one
-- The parts of a campaign an undo puts back, locked until it commits.
SELECT route_override, audience, content_settings FROM campaigns
WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL
FOR UPDATE;
