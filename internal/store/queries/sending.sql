-- W12 · Sending engine and live monitor (spec 5.3, 7.6).

-- Campaigns -------------------------------------------------------------

-- name: SendingCampaignForUpdate :one
SELECT * FROM campaigns WHERE id = @id AND deleted_at IS NULL FOR UPDATE;

-- name: SendingCampaignContext :one
-- A campaign with its sender and workspace, for building messages.
SELECT c.id, c.workspace_id, c.name, c.status, c.subject, c.preheader, c.sender_id, c.reply_to, c.audience, c.audience_count,
       c.html, c.plain_text, c.ab_test, c.schedule, c.thresholds, c.route_override, c.tracking, c.created_by, c.updated_by,
       c.scheduled_at, c.started_by, c.started_at, c.completed_at, c.version,
       coalesce(s.email, '')::text    AS sender_email,
       coalesce(s.name, '')::text     AS sender_name,
       coalesce(s.reply_to, '')::text AS sender_reply_to,
       w.slug AS workspace_slug, w.timezone AS workspace_timezone, w.name AS workspace_name
FROM campaigns c
JOIN workspaces w ON w.id = c.workspace_id
LEFT JOIN senders s ON s.id = c.sender_id
WHERE c.id = @id;

-- name: SendingCampaign :one
SELECT * FROM campaigns WHERE id = @id AND deleted_at IS NULL;

-- name: SendingCampaignInWorkspace :one
SELECT * FROM campaigns WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL;

-- name: SendingMarkScheduled :exec
UPDATE campaigns SET status = 'scheduled', scheduled_at = @at::timestamptz, updated_at = now() WHERE id = @id;

-- name: SendingMarkStarted :exec
UPDATE campaigns
SET status = 'sending', started_at = @at::timestamptz, started_by = sqlc.narg(started_by), scheduled_at = coalesce(scheduled_at, @at::timestamptz), updated_at = @at::timestamptz
WHERE id = @id;

-- name: SendingSetCampaignStatus :exec
UPDATE campaigns SET status = @status, updated_at = @at WHERE id = @id;

-- name: SendingMarkCompleted :exec
UPDATE campaigns SET status = @status, completed_at = @at::timestamptz, updated_at = @at::timestamptz WHERE id = @id;

-- name: SendingSetAudienceCount :exec
UPDATE campaigns SET audience_count = @n WHERE id = @id;

-- name: SendingDueScheduled :many
-- Scheduled campaigns whose time has come (the watchdog starts any whose
-- start job was lost).
SELECT id FROM campaigns
WHERE status = 'scheduled' AND scheduled_at <= @now AND deleted_at IS NULL
ORDER BY scheduled_at
LIMIT 50;

-- Send state ------------------------------------------------------------

-- name: SendingInsertSend :exec
INSERT INTO campaign_sends (campaign_id, workspace_id, phase, started_by, started_by_name, approved_by, approved_at, scheduled_by, throttles, thresholds, scenario, created_at, updated_at)
VALUES (@campaign_id, @workspace_id, 'preparing', sqlc.narg(started_by), @started_by_name, sqlc.narg(approved_by), sqlc.narg(approved_at), sqlc.narg(scheduled_by), @throttles, @thresholds, @scenario, @at, @at)
ON CONFLICT (campaign_id) DO UPDATE
SET phase = 'preparing', prepare_cursor = NULL, prepared_at = NULL, recipients = 0, excluded = '{}', started_by = EXCLUDED.started_by,
    started_by_name = EXCLUDED.started_by_name, approved_by = EXCLUDED.approved_by, approved_at = EXCLUDED.approved_at,
    scheduled_by = EXCLUDED.scheduled_by, throttles = EXCLUDED.throttles, thresholds = EXCLUDED.thresholds, holds = '{}', changes = '{}',
    ab = '{}', scenario = EXCLUDED.scenario, error = '', finished_at = NULL, updated_at = EXCLUDED.updated_at;

-- name: SendingGetSend :one
SELECT * FROM campaign_sends WHERE campaign_id = @campaign_id;

-- name: SendingGetSendForUpdate :one
SELECT * FROM campaign_sends WHERE campaign_id = @campaign_id FOR UPDATE;

-- name: SendingSavePrepare :exec
UPDATE campaign_sends
SET prepare_cursor = @cursor, recipients = recipients + @added, excluded = @excluded, updated_at = @at
WHERE campaign_id = @campaign_id;

-- name: SendingFinishPrepare :exec
UPDATE campaign_sends SET phase = 'sending', prepared_at = @at::timestamptz, ab = @ab, updated_at = @at::timestamptz WHERE campaign_id = @campaign_id;

-- name: SendingSetContent :exec
UPDATE campaign_sends SET content = @content, updated_at = @at WHERE campaign_id = @campaign_id;

-- name: SendingSetPhase :exec
UPDATE campaign_sends
SET phase = @phase, error = @error, finished_at = sqlc.narg(finished_at), updated_at = @at
WHERE campaign_id = @campaign_id;

-- name: SendingSetPaused :exec
UPDATE campaign_sends SET paused_at = sqlc.narg(paused_at), paused_by_name = @paused_by_name, pause_reason = @pause_reason, updated_at = now()
WHERE campaign_id = @campaign_id;

-- name: SendingSetHolds :exec
UPDATE campaign_sends SET holds = @holds, updated_at = @at WHERE campaign_id = @campaign_id;

-- name: SendingSetThrottles :exec
UPDATE campaign_sends SET throttles = @throttles, updated_at = @at WHERE campaign_id = @campaign_id;

-- name: SendingSetChanges :exec
UPDATE campaign_sends SET changes = @changes, updated_at = @at WHERE campaign_id = @campaign_id;

-- name: SendingSetAB :exec
UPDATE campaign_sends SET ab = @ab, updated_at = @at WHERE campaign_id = @campaign_id;

-- name: SendingActiveSends :many
SELECT s.campaign_id, s.phase, c.status
FROM campaign_sends s JOIN campaigns c ON c.id = s.campaign_id
WHERE s.phase IN ('preparing', 'sending') AND c.deleted_at IS NULL;

-- Audience --------------------------------------------------------------

-- name: SendingStageAdd :exec
INSERT INTO campaign_audience_stage (campaign_id, contact_id, exclude)
SELECT @campaign_id::uuid, unnest(@contact_ids::uuid[]), @exclude::boolean
ON CONFLICT DO NOTHING;

-- name: SendingStageClear :exec
DELETE FROM campaign_audience_stage WHERE campaign_id = @campaign_id;

-- name: SendingStageCount :one
SELECT count(*)::bigint FROM campaign_audience_stage WHERE campaign_id = @campaign_id;

-- name: SendingAudiencePage :many
-- One page of candidate recipients (members of the included lists or
-- segments) after the cursor, with everything that decides whether they
-- receive the campaign. The caller filters and counts what is left out.
SELECT c.id, c.email, c.first_name, c.last_name, c.attributes, c.mailbox_provider, c.tier, c.status, c.source,
       c.last_open_at, c.last_human_open_at, c.consent, c.role_address,
       (c.broadcasts_stopped_at IS NOT NULL)::boolean AS sunset_stopped,
       EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = c.workspace_id AND s.email = c.email)::boolean AS suppressed,
       coalesce((SELECT p.paused_until > @now::timestamptz FROM contact_preferences p WHERE p.contact_id = c.id), false)::boolean AS paused,
       EXISTS (SELECT 1 FROM list_memberships x WHERE x.contact_id = c.id AND x.status = 'subscribed' AND x.list_id = ANY(@exclude_lists::uuid[]))::boolean AS excluded_list,
       EXISTS (SELECT 1 FROM campaign_audience_stage st WHERE st.campaign_id = @campaign_id AND st.exclude AND st.contact_id = c.id)::boolean AS excluded_segment,
       ARRAY(SELECT m.list_id FROM list_memberships m WHERE m.contact_id = c.id AND m.status = 'subscribed' AND m.list_id = ANY(@lists::uuid[]) ORDER BY m.list_id)::uuid[] AS lists
FROM contacts c
WHERE c.workspace_id = @workspace_id AND c.id > @after::uuid AND c.erased_at IS NULL
  AND (EXISTS (SELECT 1 FROM list_memberships m WHERE m.contact_id = c.id AND m.status = 'subscribed' AND m.list_id = ANY(@lists::uuid[]))
       OR EXISTS (SELECT 1 FROM campaign_audience_stage st WHERE st.campaign_id = @campaign_id AND NOT st.exclude AND st.contact_id = c.id))
ORDER BY c.id
LIMIT @lim;

-- name: SendingClearRoutes :exec
DELETE FROM campaign_routes WHERE campaign_id = @campaign_id;

-- name: SendingAddPlanned :exec
INSERT INTO campaign_routes (campaign_id, provider_id, planned, status)
VALUES (@campaign_id, @provider_id, @planned, 'pending')
ON CONFLICT (campaign_id, provider_id) DO UPDATE SET planned = campaign_routes.planned + EXCLUDED.planned;

-- name: SendingClearLinks :exec
DELETE FROM campaign_links WHERE campaign_id = @campaign_id;

-- name: SendingInsertLink :exec
INSERT INTO campaign_links (id, campaign_id, url, label, position) VALUES (@id, @campaign_id, @url, @label, @position);

-- name: SendingClearMessages :exec
-- A send that is prepared again (after a failed preparation) starts clean.
DELETE FROM messages WHERE campaign_id = @campaign_id AND status IN ('queued', 'held') AND sent_at IS NULL;

-- Routes ----------------------------------------------------------------

-- name: SendingRoutes :many
SELECT cr.campaign_id, cr.provider_id, cr.planned, cr.status, cr.status_detail, cr.cost_micros
FROM campaign_routes cr
JOIN providers p ON p.id = cr.provider_id
WHERE cr.campaign_id = @campaign_id
ORDER BY cr.planned DESC, p.name;

-- name: SendingSetRouteStatus :exec
UPDATE campaign_routes SET status = @status, status_detail = @status_detail WHERE campaign_id = @campaign_id AND provider_id = @provider_id;

-- name: SendingAddRouteCost :exec
UPDATE campaign_routes
SET cost_micros = cost_micros + @cost_micros,
    status = CASE WHEN status = 'pending' THEN 'sending' ELSE status END
WHERE campaign_id = @campaign_id AND provider_id = @provider_id;

-- name: SendingFinishRoutes :exec
UPDATE campaign_routes SET status = @status WHERE campaign_id = @campaign_id AND status IN ('pending', 'sending', 'held', 'paused');

-- Send loop -------------------------------------------------------------

-- name: SendingClaim :many
-- Claims up to lim queued messages of one sending provider and mailbox
-- provider, oldest first, skipping rows another worker holds. A claim older
-- than the stale time belonged to a worker that died.
WITH picked AS (
    SELECT m.id FROM messages m
    WHERE m.campaign_id = @campaign_id AND m.provider_id = @provider_id AND m.mailbox_provider = @mailbox_provider
      AND m.status = 'queued'
      AND (m.claimed_at IS NULL OR m.claimed_at < @stale::timestamptz)
      AND (m.next_attempt_at IS NULL OR m.next_attempt_at <= @now::timestamptz)
    ORDER BY m.id
    LIMIT @lim
    FOR UPDATE SKIP LOCKED
), upd AS (
    UPDATE messages m SET claimed_at = @now::timestamptz, attempts = m.attempts + 1
    FROM picked WHERE m.id = picked.id
    RETURNING m.id, m.workspace_id, m.contact_id, m.email, m.variant, m.mailbox_provider, m.attempts
)
SELECT upd.id, upd.contact_id, upd.email, upd.variant, upd.mailbox_provider, upd.attempts,
       coalesce(c.first_name, '')::text AS first_name, coalesce(c.last_name, '')::text AS last_name,
       coalesce(c.attributes, '{}'::jsonb)::jsonb AS attributes, coalesce(c.tier, '')::text AS tier,
       coalesce(c.source, '')::text AS source, (c.last_human_open_at IS NOT NULL)::boolean AS engaged,
       (c.id IS NULL OR c.erased_at IS NOT NULL OR c.status <> 'active')::boolean AS gone,
       EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = upd.workspace_id AND s.email = upd.email)::boolean AS suppressed
FROM upd
LEFT JOIN contacts c ON c.id = upd.contact_id
ORDER BY upd.id;

-- name: SendingMarkSent :exec
UPDATE messages m
SET status = CASE WHEN m.status = 'queued' THEN 'sent' ELSE m.status END,
    sent_at = @at, provider_message_id = x.pmid, claimed_at = NULL, next_attempt_at = NULL, error = ''
FROM (SELECT unnest(@ids::uuid[]) AS id, unnest(@provider_message_ids::text[]) AS pmid) AS x
WHERE m.id = x.id;

-- name: SendingRetryLater :exec
UPDATE messages m SET claimed_at = NULL, next_attempt_at = x.at, error = x.err
FROM (SELECT unnest(@ids::uuid[]) AS id, unnest(@ats::timestamptz[]) AS at, unnest(@errors::text[]) AS err) AS x
WHERE m.id = x.id;

-- name: SendingMarkUnsent :exec
-- Messages that will never be sent: rejected by the provider, failed after
-- every retry, or dropped (suppressed during the send, contact erased).
UPDATE messages m SET status = x.status, claimed_at = NULL, next_attempt_at = NULL, error = x.err
FROM (SELECT unnest(@ids::uuid[]) AS id, unnest(@statuses::text[]) AS status, unnest(@errors::text[]) AS err) AS x
WHERE m.id = x.id;

-- name: SendingRelease :exec
UPDATE messages SET claimed_at = NULL, attempts = greatest(attempts - 1, 0) WHERE id = ANY(@ids::uuid[]);

-- name: SendingBumpCounters :exec
INSERT INTO campaign_counters (campaign_id, provider_id, mailbox_provider, variant, sent, rejected)
SELECT @campaign_id::uuid, x.provider_id, x.mailbox, x.variant, x.sent, x.rejected
FROM (SELECT unnest(@provider_ids::uuid[]) AS provider_id, unnest(@mailboxes::text[]) AS mailbox, unnest(@variants::text[]) AS variant,
             unnest(@sent::bigint[]) AS sent, unnest(@rejected::bigint[]) AS rejected) AS x
ON CONFLICT (campaign_id, provider_id, mailbox_provider, variant) DO UPDATE
SET sent = campaign_counters.sent + EXCLUDED.sent, rejected = campaign_counters.rejected + EXCLUDED.rejected;

-- name: SendingOpenWindow :exec
INSERT INTO sending_rate_windows (provider_id, sec, used, last_grant) VALUES (@provider_id, @sec, 0, 0)
ON CONFLICT (provider_id, sec) DO NOTHING;

-- name: SendingReserve :one
-- Reserves up to want sends in a provider's one-second window of cap (open
-- it first). No row comes back when the window is full.
UPDATE sending_rate_windows
SET last_grant = least(sqlc.arg(want)::int, sqlc.arg(cap)::int - used),
    used = used + least(sqlc.arg(want)::int, sqlc.arg(cap)::int - used)
WHERE provider_id = sqlc.arg(provider_id) AND sec = sqlc.arg(sec) AND used < sqlc.arg(cap)::int
RETURNING last_grant;

-- name: SendingPurgeWindows :exec
DELETE FROM sending_rate_windows WHERE sec < @before;

-- name: SendingQueueState :many
-- What is left of a campaign per sending and mailbox provider: queued (and
-- when the next retry is due) and held for the A/B winner.
SELECT provider_id, mailbox_provider, status,
       count(*)::bigint AS n,
       count(*) FILTER (WHERE next_attempt_at IS NOT NULL AND next_attempt_at > @now::timestamptz)::bigint AS waiting,
       coalesce(min(next_attempt_at), 'epoch'::timestamptz)::timestamptz AS next_at
FROM messages
WHERE campaign_id = @campaign_id AND status IN ('queued', 'held')
GROUP BY provider_id, mailbox_provider, status;

-- name: SendingCounters :many
SELECT provider_id, mailbox_provider, variant, sent, delivered, deferred, hard_bounced, soft_bounced, complained, rejected,
       opened_human, opened_machine, clicked, unsubscribed
FROM campaign_counters WHERE campaign_id = @campaign_id;

-- name: SendingSentSince :many
SELECT provider_id, count(*)::bigint AS n
FROM messages
WHERE campaign_id = @campaign_id AND sent_at >= @since
GROUP BY provider_id;

-- name: SendingUnsentCounts :many
-- Messages that will never be sent, per sending and mailbox provider.
SELECT provider_id, mailbox_provider, status, count(*)::bigint AS n
FROM messages
WHERE campaign_id = @campaign_id AND status IN ('failed', 'cancelled', 'rejected') AND sent_at IS NULL
GROUP BY provider_id, mailbox_provider, status;

-- name: SendingReleaseHeld :execrows
-- The A/B winner goes to everyone who waited for it.
UPDATE messages SET status = 'queued', variant = @variant WHERE campaign_id = @campaign_id AND status = 'held' AND variant = '';

-- name: SendingCancelRemaining :execrows
-- Cancels what is left of a campaign, or of one sending or mailbox provider.
UPDATE messages SET status = 'cancelled', claimed_at = NULL, error = @reason
WHERE campaign_id = @campaign_id AND status IN ('queued', 'held')
  AND (sqlc.narg(provider_id)::uuid IS NULL OR provider_id = sqlc.narg(provider_id)::uuid)
  AND (sqlc.narg(mailbox_provider)::text IS NULL OR mailbox_provider = sqlc.narg(mailbox_provider)::text);

-- name: SendingMoveRemaining :execrows
-- Moves what is left of one provider's share to another provider.
UPDATE messages SET provider_id = @to_provider, claimed_at = NULL, next_attempt_at = NULL
WHERE campaign_id = @campaign_id AND provider_id = @from_provider AND status IN ('queued', 'held');

-- name: SendingRemainingOn :one
SELECT count(*)::bigint FROM messages WHERE campaign_id = @campaign_id AND provider_id = @provider_id AND status IN ('queued', 'held');

-- name: SendingRetryNow :exec
UPDATE messages SET next_attempt_at = NULL WHERE campaign_id = @campaign_id AND provider_id = @provider_id AND status = 'queued';

-- name: SendingGroupCount :one
-- Recipients still to send in a group: never engaged, or imported and
-- never engaged, optionally for one mailbox provider.
SELECT count(*)::bigint
FROM messages m JOIN contacts c ON c.id = m.contact_id
WHERE m.campaign_id = @campaign_id AND m.status IN ('queued', 'held')
  AND (@mailbox_provider::text = '' OR m.mailbox_provider = @mailbox_provider::text)
  AND c.last_human_open_at IS NULL
  AND (NOT @imported_only::boolean OR c.source LIKE 'import:%');

-- name: SendingExcludeGroup :many
-- Takes a group out of the rest of the send; returns their contact ids so
-- they can be added to another flow.
UPDATE messages m SET status = 'cancelled', claimed_at = NULL, error = @reason
FROM contacts c
WHERE c.id = m.contact_id AND m.campaign_id = @campaign_id AND m.status IN ('queued', 'held')
  AND (@mailbox_provider::text = '' OR m.mailbox_provider = @mailbox_provider::text)
  AND c.last_human_open_at IS NULL
  AND (NOT @imported_only::boolean OR c.source LIKE 'import:%')
RETURNING m.contact_id;

-- name: SendingABTotals :many
SELECT variant, coalesce(sum(sent), 0)::bigint AS sent, coalesce(sum(delivered), 0)::bigint AS delivered,
       coalesce(sum(opened_human), 0)::bigint AS opened_human, coalesce(sum(clicked), 0)::bigint AS clicked
FROM campaign_counters WHERE campaign_id = @campaign_id AND variant <> ''
GROUP BY variant ORDER BY variant;

-- Incidents -------------------------------------------------------------

-- name: SendingInsertIncident :exec
INSERT INTO campaign_incidents (id, campaign_id, kind, scope, provider_id, mailbox_provider, metric, value, threshold, held, detail, tripped_at)
VALUES (@id, @campaign_id, @kind, @scope, sqlc.narg(provider_id), @mailbox_provider, @metric, sqlc.narg(value), sqlc.narg(threshold), @held, @detail, @tripped_at);

-- name: SendingIncidents :many
SELECT * FROM campaign_incidents WHERE campaign_id = @campaign_id ORDER BY tripped_at DESC;

-- name: SendingOpenIncidents :many
SELECT * FROM campaign_incidents WHERE campaign_id = @campaign_id AND resolved_at IS NULL ORDER BY tripped_at DESC;

-- name: SendingResolveIncident :exec
UPDATE campaign_incidents SET resolved_at = @at, resolution = @resolution, resolved_by = sqlc.narg(resolved_by)
WHERE id = @id AND resolved_at IS NULL;

-- name: SendingUpdateIncidentDetail :exec
UPDATE campaign_incidents SET detail = @detail, held = @held WHERE id = @id;

-- name: SendingComplaintCohorts :many
-- Complaints of a campaign (optionally one mailbox provider) by how the
-- complaining contacts joined and whether they ever engaged: the evidence
-- behind "likely causes".
SELECT coalesce(c.source, '')::text AS source, (c.last_human_open_at IS NULL)::boolean AS never_engaged,
       coalesce(c.consent, '')::text AS consent,
       count(*)::bigint AS n
FROM delivery_events e
JOIN messages m ON m.id = e.message_id
LEFT JOIN contacts c ON c.id = m.contact_id
WHERE e.campaign_id = @campaign_id AND e.type = 'complaint'
  AND (@mailbox_provider::text = '' OR e.mailbox_provider = @mailbox_provider::text)
GROUP BY 1, 2, 3
ORDER BY n DESC;

-- name: SendingCohortSize :one
-- Recipients of a mailbox provider in this send who joined from a source
-- and never engaged, and how many of them are still to send.
SELECT count(*)::bigint AS total,
       count(*) FILTER (WHERE m.status IN ('queued', 'held'))::bigint AS remaining
FROM messages m JOIN contacts c ON c.id = m.contact_id
WHERE m.campaign_id = @campaign_id
  AND (@mailbox_provider::text = '' OR m.mailbox_provider = @mailbox_provider::text)
  AND c.source = @source AND c.last_human_open_at IS NULL;

-- name: SendingImportOf :one
-- The most recent import from a source ("the Mailchimp import on 18 Sep").
SELECT source, created_at, coalesce(finished_at, created_at)::timestamptz AS finished_at
FROM imports WHERE workspace_id = @workspace_id AND source = @source
ORDER BY created_at DESC LIMIT 1;

-- name: SendingPriorSends :one
-- How many of a cohort's contacts received anything from this workspace
-- before this campaign.
SELECT count(DISTINCT m.contact_id)::bigint
FROM messages m
WHERE m.workspace_id = @workspace_id AND m.contact_id = ANY(@contact_ids::uuid[]) AND m.sent_at IS NOT NULL
  AND (m.campaign_id IS NULL OR m.campaign_id <> @campaign_id);

-- name: SendingCohortContacts :many
SELECT m.contact_id
FROM messages m JOIN contacts c ON c.id = m.contact_id
WHERE m.campaign_id = @campaign_id
  AND (@mailbox_provider::text = '' OR m.mailbox_provider = @mailbox_provider::text)
  AND c.source = @source AND c.last_open_at IS NULL
LIMIT 2000;

-- name: SendingDeferralsSince :one
SELECT count(*)::bigint FROM delivery_events
WHERE campaign_id = @campaign_id AND type = 'deferred' AND (@mailbox_provider::text = '' OR mailbox_provider = @mailbox_provider::text);

-- Live monitor ----------------------------------------------------------

-- name: SendingFeed :many
-- The newest delivery and engagement events of a campaign.
SELECT e.id, e.type, e.occurred_at, e.received_at, e.recipient, e.provider_id, e.original_code, e.diagnostic,
       e.machine, e.url, e.user_agent, e.mailbox_provider,
       coalesce(m.variant, '')::text AS variant,
       (e.type = 'hard_bounce' OR e.type = 'complaint')::boolean AS suppresses
FROM delivery_events e
LEFT JOIN messages m ON m.id = e.message_id
WHERE e.campaign_id = @campaign_id AND e.received_at > @since
ORDER BY e.received_at DESC, e.id DESC
LIMIT @lim;

-- name: SendingApproval :one
-- The approval a campaign's send went out under, with the approver.
SELECT r.id, r.requested_by, coalesce(u.name, '')::text AS approver_name, d.approver_id, d.created_at AS approved_at
FROM approval_requests r
JOIN approval_decisions d ON d.request_id = r.id AND d.decision = 'approve'
LEFT JOIN users u ON u.id = d.approver_id
WHERE r.campaign_id = @campaign_id AND r.status IN ('approved', 'sent')
ORDER BY d.created_at DESC
LIMIT 1;

-- name: SendingUserNames :many
SELECT id, name FROM users WHERE id = ANY(@ids::uuid[]);

-- name: SendingWorkspaceProviders :many
SELECT * FROM providers WHERE workspace_id = @workspace_id AND disconnected_at IS NULL ORDER BY connected_at, name;

-- name: SendingListNames :many
SELECT id, name FROM lists WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY created_at;

-- name: SendingSegmentNames :many
SELECT id, name FROM segments WHERE workspace_id = @workspace_id ORDER BY name;

-- name: SendingAutomationByName :one
-- A flow whose name matches, for "move them into the Re-permission automation".
SELECT id, name, status FROM automations
WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND name ILIKE @pattern
ORDER BY (status = 'live') DESC, updated_at DESC
LIMIT 1;

-- Demo transport --------------------------------------------------------

-- name: SendingDueSimEvents :many
SELECT * FROM sending_sim_events WHERE due_at <= @now ORDER BY due_at, id LIMIT @lim FOR UPDATE SKIP LOCKED;

-- name: SendingDeleteSimEvents :exec
DELETE FROM sending_sim_events WHERE id = ANY(@ids::bigint[]);

-- name: SendingNextSimEvent :one
SELECT coalesce(min(due_at), 'epoch'::timestamptz)::timestamptz FROM sending_sim_events;

-- name: SendingSimMessages :many
SELECT id, workspace_id, campaign_id, automation_id, contact_id, email, provider_id, provider_message_id, mailbox_provider, variant, sent_at, stream
FROM messages WHERE id = ANY(@ids::uuid[]);

-- Automation outbox -----------------------------------------------------

-- name: SendingInsertAutomationMessage :exec
INSERT INTO messages (id, workspace_id, stream, automation_id, automation_node, contact_id, email, provider_id, mailbox_provider, status, queued_at, content_ref)
VALUES (@id, @workspace_id, 'automation', @automation_id, @automation_node, @contact_id, @email, sqlc.narg(provider_id), @mailbox_provider, @status, @queued_at, @content_ref);

-- name: SendingOutboxProviders :many
SELECT DISTINCT provider_id::uuid AS provider_id FROM messages
WHERE status = 'queued' AND campaign_id IS NULL AND stream = 'automation' AND provider_id IS NOT NULL;

-- name: SendingOutboxClaim :many
WITH picked AS (
    SELECT m.id FROM messages m
    WHERE m.provider_id = @provider_id AND m.status = 'queued' AND m.campaign_id IS NULL AND m.stream = 'automation'
      AND (m.claimed_at IS NULL OR m.claimed_at < @stale::timestamptz)
      AND (m.next_attempt_at IS NULL OR m.next_attempt_at <= @now::timestamptz)
    ORDER BY m.id
    LIMIT @lim
    FOR UPDATE SKIP LOCKED
), upd AS (
    UPDATE messages m SET claimed_at = @now::timestamptz, attempts = m.attempts + 1
    FROM picked WHERE m.id = picked.id
    RETURNING m.id, m.workspace_id, m.automation_id, m.automation_node, m.contact_id, m.email, m.mailbox_provider, m.attempts, m.content_ref
)
SELECT upd.id, upd.workspace_id, upd.automation_id, upd.automation_node, upd.contact_id, upd.email, upd.mailbox_provider, upd.attempts,
       coalesce(upd.content_ref, '{}'::jsonb)::jsonb AS content_ref,
       coalesce(c.first_name, '')::text AS first_name, coalesce(c.last_name, '')::text AS last_name,
       coalesce(c.attributes, '{}'::jsonb)::jsonb AS attributes, coalesce(c.tier, '')::text AS tier,
       coalesce(c.source, '')::text AS source, (c.last_human_open_at IS NOT NULL)::boolean AS engaged,
       (c.id IS NULL OR c.erased_at IS NOT NULL OR c.status <> 'active')::boolean AS gone,
       EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = upd.workspace_id AND s.email = upd.email)::boolean AS suppressed
FROM upd
LEFT JOIN contacts c ON c.id = upd.contact_id
ORDER BY upd.id;

-- name: SendingAutomationEmail :one
SELECT v.email_id, v.version, v.subject, v.preheader, v.html, v.plain_text, e.name, e.workspace_id
FROM automation_email_versions v
JOIN automation_emails e ON e.id = v.email_id
WHERE v.email_id = @email_id AND v.version = @version;

-- name: SendingAutomationEmailLatest :one
SELECT latest_version FROM automation_emails WHERE id = @id;

-- name: SendingAutomationLinkCount :one
SELECT count(*)::bigint FROM automation_email_links WHERE email_id = @email_id AND version = @version;

-- name: SendingInsertAutomationLink :exec
INSERT INTO automation_email_links (email_id, version, position, url, label) VALUES (@email_id, @version, @position, @url, @label)
ON CONFLICT DO NOTHING;

-- name: SendingAutomationSender :one
-- The workspace's default sender (automation emails come from it).
SELECT coalesce(s.email, '')::text AS email, coalesce(s.name, '')::text AS name, coalesce(s.reply_to, '')::text AS reply_to, w.slug, w.timezone
FROM workspaces w LEFT JOIN senders s ON s.id = w.default_sender_id
WHERE w.id = @workspace_id;

-- name: SendingContactForAutomation :one
SELECT id, email, mailbox_provider, status, erased_at FROM contacts WHERE id = @id;

-- name: SendingCampaignWorkspace :one
SELECT workspace_id FROM campaigns WHERE id = $1;
