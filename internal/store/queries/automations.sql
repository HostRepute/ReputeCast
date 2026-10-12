-- Automations (W06): flows, versions, runs, step stats, emails, trigger
-- events, webhook calls, step secrets and alerts.

-- name: ListAutomations :many
-- Every flow of a workspace with its live and draft graphs: live first, then
-- drafts, then paused, most recently changed first.
SELECT a.id, a.name, a.description, a.status, a.template_key, a.live_version, a.draft_version,
       a.trigger, a.created_at, a.updated_at, a.paused_at, a.pause_reason,
       coalesce(pu.name, '')::text AS paused_by_name,
       lv.published_at AS live_published_at,
       coalesce(lv.graph, 'null'::jsonb)::jsonb AS live_graph,
       coalesce(dv.graph, 'null'::jsonb)::jsonb AS draft_graph
FROM automations a
LEFT JOIN users pu ON pu.id = a.paused_by
LEFT JOIN automation_versions lv ON lv.automation_id = a.id AND lv.version = a.live_version
LEFT JOIN automation_versions dv ON dv.automation_id = a.id AND dv.version = a.draft_version
WHERE a.workspace_id = @workspace_id AND a.deleted_at IS NULL
ORDER BY CASE a.status WHEN 'live' THEN 0 WHEN 'draft' THEN 1 ELSE 2 END, a.updated_at DESC, a.id;

-- name: GetAutomation :one
SELECT a.*, coalesce(pu.name, '')::text AS paused_by_name
FROM automations a
LEFT JOIN users pu ON pu.id = a.paused_by
WHERE a.workspace_id = @workspace_id AND a.id = @id AND a.deleted_at IS NULL;

-- name: LockAutomation :one
SELECT * FROM automations WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL FOR UPDATE;

-- name: CreateAutomation :one
INSERT INTO automations (id, workspace_id, name, description, status, template_key, draft_version, trigger, created_by, updated_by)
VALUES (@id, @workspace_id, @name, @description, 'draft', @template_key, 1, @trigger, @created_by, @created_by)
RETURNING *;

-- name: UpdateAutomationMeta :one
UPDATE automations SET name = @name, description = @description, updated_by = @updated_by, updated_at = now()
WHERE id = @id
RETURNING *;

-- name: SetAutomationDraft :exec
-- A draft was saved: bump the revision and remember the trigger of a flow
-- that was never published (live flows keep the live trigger).
UPDATE automations
SET draft_version = @draft_version, draft_revision = draft_revision + 1, updated_by = @updated_by, updated_at = now(),
    trigger = CASE WHEN live_version IS NULL THEN @trigger::jsonb ELSE trigger END
WHERE id = @id;

-- name: SetAutomationLive :exec
UPDATE automations
SET status = 'live', live_version = @live_version, draft_version = @live_version, trigger = @trigger,
    paused_at = NULL, paused_by = NULL, pause_reason = '', draft_revision = draft_revision + 1,
    updated_by = @updated_by, updated_at = now()
WHERE id = @id;

-- name: PauseAutomation :exec
UPDATE automations SET status = 'paused', paused_at = now(), paused_by = @paused_by, pause_reason = @pause_reason,
    updated_by = @paused_by, updated_at = now()
WHERE id = @id;

-- name: ResumeAutomation :exec
UPDATE automations SET status = 'live', paused_at = NULL, paused_by = NULL, pause_reason = '', updated_by = @updated_by, updated_at = now()
WHERE id = @id;

-- name: SoftDeleteAutomation :exec
UPDATE automations SET deleted_at = now(), status = CASE WHEN status = 'live' THEN 'paused' ELSE status END, updated_by = @updated_by
WHERE id = @id;

-- name: CountAutomationsByStatus :many
SELECT status, count(*)::bigint AS n FROM automations
WHERE workspace_id = @workspace_id AND deleted_at IS NULL
GROUP BY status;

-- name: InsertAutomationVersion :exec
INSERT INTO automation_versions (automation_id, version, graph, created_by, summary)
VALUES (@automation_id, @version, @graph, @created_by, @summary);

-- name: UpdateDraftGraph :execrows
UPDATE automation_versions SET graph = @graph
WHERE automation_id = @automation_id AND version = @version AND published_at IS NULL;

-- name: GetAutomationVersion :one
SELECT v.*, coalesce(cu.name, '')::text AS created_by_name, coalesce(pu.name, '')::text AS published_by_name
FROM automation_versions v
LEFT JOIN users cu ON cu.id = v.created_by
LEFT JOIN users pu ON pu.id = v.published_by
WHERE v.automation_id = @automation_id AND v.version = @version;

-- name: ListAutomationVersions :many
SELECT v.automation_id, v.version, v.created_at, v.published_at, v.retired_at, v.summary, v.migration,
       coalesce(cu.name, '')::text AS created_by_name, coalesce(pu.name, '')::text AS published_by_name
FROM automation_versions v
LEFT JOIN users cu ON cu.id = v.created_by
LEFT JOIN users pu ON pu.id = v.published_by
WHERE v.automation_id = @automation_id
ORDER BY v.version DESC;

-- name: ListPublishedGraphs :many
SELECT version, graph FROM automation_versions
WHERE automation_id = @automation_id AND published_at IS NOT NULL
ORDER BY version;

-- name: PublishAutomationVersion :exec
UPDATE automation_versions SET published_at = now(), published_by = @published_by, summary = @summary, migration = @migration
WHERE automation_id = @automation_id AND version = @version;

-- name: RetireAutomationVersion :exec
UPDATE automation_versions SET retired_at = now()
WHERE automation_id = @automation_id AND version = @version AND retired_at IS NULL;

-- name: ListRetiringVersions :many
-- Published versions that are no longer live but not yet retired: they
-- retire once their last contact leaves.
SELECT v.automation_id, v.version
FROM automation_versions v
JOIN automations a ON a.id = v.automation_id
WHERE v.published_at IS NOT NULL AND v.retired_at IS NULL AND v.version <> coalesce(a.live_version, 0)
  AND NOT EXISTS (SELECT 1 FROM automation_runs r WHERE r.automation_id = v.automation_id AND r.version = v.version AND r.status IN ('active', 'waiting'));

-- name: CountInFlow :many
SELECT automation_id, count(*)::bigint AS n
FROM automation_runs
WHERE automation_id = ANY(@automation_ids::uuid[]) AND status IN ('active', 'waiting')
GROUP BY automation_id;

-- name: CountInFlowByNode :many
SELECT version, node_id, count(*)::bigint AS n
FROM automation_runs
WHERE automation_id = @automation_id AND status IN ('active', 'waiting')
GROUP BY version, node_id;

-- name: StepStatsSince :many
-- Per-step totals since a day, across versions (steps keep their ids).
SELECT automation_id, node_id,
       sum(entered)::bigint AS entered, sum(completed)::bigint AS completed, sum(waiting)::bigint AS waiting,
       sum(exited)::bigint AS exited, sum(dropped)::bigint AS dropped, sum(goal_reached)::bigint AS goal_reached,
       sum(skipped)::bigint AS skipped, sum(failed)::bigint AS failed,
       sum(sent)::bigint AS sent, sum(delivered)::bigint AS delivered, sum(bounced)::bigint AS bounced,
       sum(complained)::bigint AS complained, sum(opened)::bigint AS opened, sum(opened_human)::bigint AS opened_human,
       sum(clicked)::bigint AS clicked, sum(unsubscribed)::bigint AS unsubscribed,
       sum(branch_yes)::bigint AS branch_yes, sum(branch_no)::bigint AS branch_no
FROM automation_step_stats
WHERE automation_id = ANY(@automation_ids::uuid[]) AND day >= @since::date
GROUP BY automation_id, node_id;

-- name: StepPathsSince :many
-- Paths taken out of branches and splits.
SELECT s.automation_id, s.node_id, p.k::text AS path, sum(p.v::bigint)::bigint AS n
FROM automation_step_stats s, jsonb_each_text(s.paths) AS p(k, v)
WHERE s.automation_id = ANY(@automation_ids::uuid[]) AND s.day >= @since::date
GROUP BY s.automation_id, s.node_id, p.k;

-- name: StepProvidersSince :many
-- Emails per sending provider, per step.
SELECT s.automation_id, s.node_id, p.k::text AS provider_id, sum(p.v::bigint)::bigint AS n, max(s.day)::date AS last_day
FROM automation_step_stats s, jsonb_each_text(s.providers) AS p(k, v)
WHERE s.automation_id = ANY(@automation_ids::uuid[]) AND s.day >= @since::date
GROUP BY s.automation_id, s.node_id, p.k;

-- name: WorkspaceProviderSends :many
-- Automation emails per sending provider since a day, and the last day each sent.
SELECT p.k::text AS provider_id, sum(p.v::bigint)::bigint AS n, max(s.day)::date AS last_day
FROM automation_step_stats s
JOIN automations a ON a.id = s.automation_id
CROSS JOIN LATERAL jsonb_each_text(s.providers) AS p(k, v)
WHERE a.workspace_id = @workspace_id AND a.deleted_at IS NULL AND s.day >= @since::date
GROUP BY p.k;

-- name: WorkspaceStatsSince :many
-- Totals per automation since a day, for the KPI row.
SELECT s.automation_id,
       sum(s.sent)::bigint AS sent, sum(s.delivered)::bigint AS delivered,
       sum(s.opened_human)::bigint AS opened_human, sum(s.clicked)::bigint AS clicked,
       sum(s.goal_reached)::bigint AS goal_reached
FROM automation_step_stats s
JOIN automations a ON a.id = s.automation_id
WHERE a.workspace_id = @workspace_id AND a.deleted_at IS NULL AND s.day >= @since::date
GROUP BY s.automation_id;

-- name: BumpStepStats :exec
-- Adds to one step's counters for a day. Path and provider counts are merged
-- key by key.
INSERT INTO automation_step_stats AS s (automation_id, version, node_id, day,
    entered, completed, waiting, exited, dropped, goal_reached, skipped, failed, sent,
    delivered, bounced, complained, opened, opened_human, clicked, unsubscribed, branch_yes, branch_no, paths, providers)
VALUES (@automation_id, @version, @node_id, @day,
    @entered, @completed, @waiting, @exited, @dropped, @goal_reached, @skipped, @failed, @sent,
    @delivered, @bounced, @complained, @opened, @opened_human, @clicked, @unsubscribed, @branch_yes, @branch_no,
    @paths, @providers)
ON CONFLICT (automation_id, version, node_id, day) DO UPDATE SET
    entered = s.entered + EXCLUDED.entered,
    completed = s.completed + EXCLUDED.completed,
    waiting = s.waiting + EXCLUDED.waiting,
    exited = s.exited + EXCLUDED.exited,
    dropped = s.dropped + EXCLUDED.dropped,
    goal_reached = s.goal_reached + EXCLUDED.goal_reached,
    skipped = s.skipped + EXCLUDED.skipped,
    failed = s.failed + EXCLUDED.failed,
    sent = s.sent + EXCLUDED.sent,
    delivered = s.delivered + EXCLUDED.delivered,
    bounced = s.bounced + EXCLUDED.bounced,
    complained = s.complained + EXCLUDED.complained,
    opened = s.opened + EXCLUDED.opened,
    opened_human = s.opened_human + EXCLUDED.opened_human,
    clicked = s.clicked + EXCLUDED.clicked,
    unsubscribed = s.unsubscribed + EXCLUDED.unsubscribed,
    branch_yes = s.branch_yes + EXCLUDED.branch_yes,
    branch_no = s.branch_no + EXCLUDED.branch_no,
    paths = (SELECT coalesce(jsonb_object_agg(k, v), '{}'::jsonb) FROM (
        SELECT k, sum(v)::bigint AS v FROM (
            SELECT a.k, a.v::bigint AS v FROM jsonb_each_text(s.paths) AS a(k, v)
            UNION ALL SELECT b.k, b.v::bigint FROM jsonb_each_text(EXCLUDED.paths) AS b(k, v)) x GROUP BY k) y),
    providers = (SELECT coalesce(jsonb_object_agg(k, v), '{}'::jsonb) FROM (
        SELECT k, sum(v)::bigint AS v FROM (
            SELECT a.k, a.v::bigint AS v FROM jsonb_each_text(s.providers) AS a(k, v)
            UNION ALL SELECT b.k, b.v::bigint FROM jsonb_each_text(EXCLUDED.providers) AS b(k, v)) x GROUP BY k) y);

-- name: AutomationRoute :one
-- The default Automations route: the weighted split and the fallback.
SELECT splits, fallback_provider_id FROM routing_rules
WHERE workspace_id = @workspace_id AND stream = 'automation' AND enabled AND conditions = '{}'::jsonb
ORDER BY priority LIMIT 1;

-- name: WorkspaceProviders :many
SELECT id, name, adapter, status, status_detail, connected_at FROM providers
WHERE workspace_id = @workspace_id AND disconnected_at IS NULL
ORDER BY connected_at;

-- name: WorkspaceForms :many
SELECT id, name, slug FROM forms WHERE workspace_id = @workspace_id ORDER BY created_at;

-- name: WorkspaceDateFields :many
SELECT key, label, type FROM custom_fields WHERE workspace_id = @workspace_id ORDER BY label;

-- name: WorkspaceTopTags :many
SELECT t::text AS tag, count(*)::bigint AS n
FROM contacts c, unnest(c.tags) t
WHERE c.workspace_id = @workspace_id AND c.erased_at IS NULL
GROUP BY t ORDER BY n DESC, t LIMIT 50;

-- name: WorkspaceEventNames :many
SELECT DISTINCT ref::text AS event FROM automation_events
WHERE workspace_id = @workspace_id AND kind = 'api_event' AND occurred_at > now() - interval '90 days'
ORDER BY 1 LIMIT 50;

-- name: CreateAutomationEmail :exec
INSERT INTO automation_emails (id, workspace_id, automation_id, name, latest_version, created_by)
VALUES (@id, @workspace_id, @automation_id, @name, 1, @created_by);

-- name: InsertAutomationEmailVersion :exec
INSERT INTO automation_email_versions (email_id, version, subject, preheader, mjml, html, plain_text, created_by)
VALUES (@email_id, @version, @subject, @preheader, @mjml, @html, @plain_text, @created_by);

-- name: ListAutomationEmails :many
SELECT e.id, e.name, e.latest_version, e.automation_id, e.updated_at, v.subject
FROM automation_emails e
JOIN automation_email_versions v ON v.email_id = e.id AND v.version = e.latest_version
WHERE e.workspace_id = @workspace_id AND e.deleted_at IS NULL
ORDER BY e.created_at;

-- name: GetAutomationEmailVersion :one
SELECT e.id, e.name, e.latest_version, v.version, v.subject, v.preheader, v.mjml, v.html, v.plain_text, v.created_at
FROM automation_emails e
JOIN automation_email_versions v ON v.email_id = e.id
WHERE e.workspace_id = @workspace_id AND e.id = @id AND v.version = @version;

-- name: InsertAutomationEvent :exec
INSERT INTO automation_events (workspace_id, kind, contact_id, ref, payload, occurred_at)
VALUES (@workspace_id, @kind, @contact_id, @ref, @payload, @occurred_at);

-- name: ClaimPendingEvents :many
SELECT id, workspace_id, kind, contact_id, ref, payload, occurred_at
FROM automation_events
WHERE processed_at IS NULL
ORDER BY id
LIMIT @lim
FOR UPDATE SKIP LOCKED;

-- name: MarkEventProcessed :exec
UPDATE automation_events SET processed_at = now(), enrolled = @enrolled WHERE id = @id;

-- name: PurgeProcessedEvents :execrows
DELETE FROM automation_events WHERE processed_at < @before;

-- name: LiveAutomationsForTrigger :many
-- Live flows a trigger event enrolls into.
SELECT a.id, a.workspace_id, a.live_version, a.trigger, v.graph
FROM automations a
JOIN automation_versions v ON v.automation_id = a.id AND v.version = a.live_version
WHERE a.workspace_id = @workspace_id AND a.status = 'live' AND a.deleted_at IS NULL
  AND a.trigger ->> 'kind' = @kind::text;

-- name: RunExistsForContact :one
SELECT EXISTS (SELECT 1 FROM automation_runs WHERE automation_id = @automation_id AND contact_id = @contact_id);

-- name: InsertRun :execrows
INSERT INTO automation_runs (id, automation_id, version, contact_id, node_id, status, state, entered_at, node_entered_at, next_at)
VALUES (@id, @automation_id, @version, @contact_id, @node_id, 'active', @state, @entered_at, @entered_at, @entered_at)
ON CONFLICT DO NOTHING;

-- name: ClaimDueRuns :many
-- Runs ready to move on, in flows that are live (paused flows hold their runs).
SELECT r.id, r.automation_id, r.version, r.contact_id, r.node_id, r.status, r.state, r.entered_at, r.node_entered_at, r.next_at,
       a.workspace_id
FROM automation_runs r
JOIN automations a ON a.id = r.automation_id
WHERE r.status IN ('active', 'waiting') AND r.next_at <= @now AND a.status = 'live' AND a.deleted_at IS NULL
ORDER BY r.next_at
LIMIT @lim
FOR UPDATE OF r SKIP LOCKED;

-- name: GetRunForUpdate :one
SELECT r.id, r.automation_id, r.version, r.contact_id, r.node_id, r.status, r.state, r.entered_at, r.node_entered_at, r.next_at,
       a.workspace_id
FROM automation_runs r
JOIN automations a ON a.id = r.automation_id
WHERE r.id = @id
FOR UPDATE OF r;

-- name: UpdateRun :exec
UPDATE automation_runs
SET version = @version, node_id = @node_id, status = @status, state = @state, node_entered_at = @node_entered_at,
    next_at = sqlc.narg(next_at), finished_at = sqlc.narg(finished_at), exit_reason = @exit_reason
WHERE id = @id;

-- name: ListActiveRunsForMigration :many
SELECT id, version, node_id, status, state, node_entered_at, next_at
FROM automation_runs
WHERE automation_id = @automation_id AND status IN ('active', 'waiting') AND version <> @version
FOR UPDATE;

-- name: ExitActiveRuns :execrows
UPDATE automation_runs SET status = 'exited', finished_at = now(), next_at = NULL, exit_reason = @exit_reason
WHERE automation_id = @automation_id AND status IN ('active', 'waiting');

-- name: WakeEngagedBranchRuns :execrows
-- Runs waiting at a branch for an open or click: wake them as soon as the
-- email they wait on has one, instead of at the end of the window.
UPDATE automation_runs r SET next_at = @now
WHERE r.status = 'waiting' AND r.next_at > @now AND r.state ->> 'wait' = 'branch'
  AND (r.state ->> 'message') IS NOT NULL
  AND EXISTS (
    SELECT 1 FROM delivery_events de
    WHERE de.message_id = (r.state ->> 'message')::uuid
      AND ((r.state ->> 'want' = 'opened' AND de.type = 'open' AND (NOT coalesce((r.state ->> 'human')::boolean, false) OR NOT de.machine))
        OR (r.state ->> 'want' = 'clicked' AND de.type = 'click')));

-- name: ContactForRun :one
SELECT id, email, first_name, last_name, status, timezone, tags, attributes, erased_at
FROM contacts WHERE id = @id;

-- name: ContactInList :one
SELECT EXISTS (SELECT 1 FROM list_memberships WHERE list_id = @list_id AND contact_id = @contact_id AND status = 'subscribed');

-- name: ContactGotEmailSince :one
SELECT EXISTS (SELECT 1 FROM messages WHERE contact_id = @contact_id AND queued_at >= @since);

-- name: ContactSuppressed :one
SELECT EXISTS (SELECT 1 FROM suppressions WHERE workspace_id = @workspace_id AND email = @email);

-- name: RunMessage :one
-- The message a run's email step queued (for branch checks).
SELECT id, queued_at, status FROM messages
WHERE automation_id = @automation_id AND automation_node = @node_id AND contact_id = @contact_id
ORDER BY queued_at DESC LIMIT 1;

-- name: MessageEngaged :one
SELECT EXISTS (
    SELECT 1 FROM delivery_events de
    WHERE de.message_id = @message_id
      AND ((@want::text = 'opened' AND de.type = 'open' AND (NOT @human_only::boolean OR NOT de.machine))
        OR (@want::text = 'clicked' AND de.type = 'click' AND (@url_contains::text = '' OR de.url ILIKE '%' || @url_contains::text || '%'))));

-- name: AutomationEngaged :one
-- Whether the contact opened or clicked any email of this flow since a time.
SELECT EXISTS (
    SELECT 1 FROM delivery_events de
    JOIN messages m ON m.id = de.message_id
    WHERE de.contact_id = @contact_id AND m.automation_id = @automation_id AND de.occurred_at >= @since
      AND ((@want::text = 'opened' AND de.type = 'open' AND (NOT @human_only::boolean OR NOT de.machine))
        OR (@want::text = 'clicked' AND de.type = 'click' AND (@url_contains::text = '' OR de.url ILIKE '%' || @url_contains::text || '%'))));

-- name: ContactSentEvent :one
SELECT EXISTS (
    SELECT 1 FROM automation_events
    WHERE workspace_id = @workspace_id AND contact_id = @contact_id AND kind = 'api_event' AND ref = @event AND occurred_at >= @since);

-- name: InsertAutomationMessage :one
INSERT INTO messages (id, workspace_id, stream, automation_id, automation_node, contact_id, email, provider_id, mailbox_provider, status, queued_at, error)
VALUES (@id, @workspace_id, 'automation', @automation_id, @automation_node, @contact_id, @email, @provider_id, @mailbox_provider, @status, @queued_at, @error)
RETURNING id;

-- name: CountHeldMessages :one
SELECT count(*)::bigint FROM messages WHERE automation_id = @automation_id AND status = 'held';

-- name: ReleaseHeldMessages :execrows
UPDATE messages SET status = 'queued', provider_id = @provider_id, error = ''
WHERE automation_id = @automation_id AND status = 'held';

-- name: AddListMembership :exec
INSERT INTO list_memberships (list_id, contact_id, status, source, subscribed_at)
VALUES (@list_id, @contact_id, 'subscribed', @source, now())
ON CONFLICT (list_id, contact_id) DO UPDATE SET status = 'subscribed', source = EXCLUDED.source, subscribed_at = now(), unsubscribed_at = NULL
WHERE list_memberships.status <> 'subscribed';

-- name: RemoveListMembership :exec
UPDATE list_memberships SET status = 'unsubscribed', unsubscribed_at = now()
WHERE list_id = @list_id AND contact_id = @contact_id AND status <> 'unsubscribed';

-- name: UnsubscribeAllLists :exec
UPDATE list_memberships SET status = 'unsubscribed', unsubscribed_at = now()
WHERE contact_id = @contact_id AND status <> 'unsubscribed';

-- name: AddContactTag :exec
UPDATE contacts SET tags = array_append(tags, @tag::text), updated_at = now()
WHERE id = @id AND NOT (@tag::text = ANY (tags));

-- name: RemoveContactTag :exec
UPDATE contacts SET tags = array_remove(tags, @tag::text), updated_at = now()
WHERE id = @id AND @tag::text = ANY (tags);

-- name: GetStepSecret :one
SELECT secret_enc FROM automation_step_secrets WHERE automation_id = @automation_id AND node_id = @node_id;

-- name: UpsertStepSecret :exec
INSERT INTO automation_step_secrets (automation_id, node_id, secret_enc, updated_by)
VALUES (@automation_id, @node_id, @secret_enc, @updated_by)
ON CONFLICT (automation_id, node_id) DO UPDATE SET secret_enc = EXCLUDED.secret_enc, updated_by = EXCLUDED.updated_by, updated_at = now();

-- name: DeleteStepSecret :execrows
DELETE FROM automation_step_secrets WHERE automation_id = @automation_id AND node_id = @node_id;

-- name: StepSecretNodes :many
SELECT node_id FROM automation_step_secrets WHERE automation_id = @automation_id;

-- name: InsertWebhookCall :exec
INSERT INTO automation_webhook_calls (id, automation_id, version, node_id, run_id, url, status_code, error, duration_ms, attempt, called_at)
VALUES (@id, @automation_id, @version, @node_id, @run_id, @url, @status_code, @error, @duration_ms, @attempt, @called_at);

-- name: ListWebhookCalls :many
SELECT id, version, run_id, url, status_code, error, duration_ms, attempt, called_at
FROM automation_webhook_calls
WHERE automation_id = @automation_id AND node_id = @node_id
ORDER BY called_at DESC, id DESC
LIMIT @lim;

-- name: PurgeWebhookCalls :execrows
DELETE FROM automation_webhook_calls WHERE called_at < @before;

-- name: OpenAlert :exec
-- Records a failure: opens the alert or adds to the open one.
INSERT INTO automation_alerts (id, automation_id, kind, node_id, detail, failures, first_at, last_at)
VALUES (@id, @automation_id, @kind, @node_id, @detail, @failures, @at, @at)
ON CONFLICT (automation_id, kind, node_id) WHERE resolved_at IS NULL
DO UPDATE SET failures = automation_alerts.failures + EXCLUDED.failures, detail = EXCLUDED.detail, last_at = EXCLUDED.last_at;

-- name: ResolveAlert :execrows
UPDATE automation_alerts SET resolved_at = now()
WHERE automation_id = @automation_id AND kind = @kind AND node_id = @node_id AND resolved_at IS NULL;

-- name: ListOpenAlerts :many
SELECT al.id, al.automation_id, al.kind, al.node_id, al.detail, al.failures, al.first_at, al.last_at,
       a.name AS automation_name, a.draft_version, a.live_version
FROM automation_alerts al
JOIN automations a ON a.id = al.automation_id
WHERE a.workspace_id = @workspace_id AND a.deleted_at IS NULL AND al.resolved_at IS NULL
ORDER BY al.kind DESC, al.first_at;

-- name: StepFailures :many
-- The latest alert per webhook step, open or resolved (for the publish diff).
SELECT DISTINCT ON (node_id) node_id, detail, failures, first_at, last_at, resolved_at
FROM automation_alerts
WHERE automation_id = @automation_id AND kind = 'step_failing'
ORDER BY node_id, first_at DESC;

-- name: OpenSendingHeldAlerts :many
SELECT al.automation_id, a.workspace_id FROM automation_alerts al
JOIN automations a ON a.id = al.automation_id
WHERE al.kind = 'sending_held' AND al.resolved_at IS NULL;

-- name: DateFieldMatches :many
-- Contacts whose date field (plus the offset) is today in the workspace's
-- timezone; yearly triggers match the day and month every year.
SELECT c.id FROM contacts c
WHERE c.workspace_id = @workspace_id AND c.status = 'active' AND c.erased_at IS NULL
  AND CASE WHEN @field::text = 'created_at' THEN (c.created_at AT TIME ZONE @tz::text)::date
           ELSE CASE WHEN (c.attributes ->> @field::text) ~ '^\d{4}-\d{2}-\d{2}' THEN substr(c.attributes ->> @field::text, 1, 10)::date END
      END IS NOT NULL
  AND (
    (NOT @yearly::boolean AND
       CASE WHEN @field::text = 'created_at' THEN (c.created_at AT TIME ZONE @tz::text)::date
            ELSE substr(c.attributes ->> @field::text, 1, 10)::date END + @offset_days::int = @today::date)
    OR (@yearly::boolean AND to_char(
       CASE WHEN @field::text = 'created_at' THEN (c.created_at AT TIME ZONE @tz::text)::date
            ELSE substr(c.attributes ->> @field::text, 1, 10)::date END + @offset_days::int, 'MM-DD') = to_char(@today::date, 'MM-DD')))
  AND NOT EXISTS (SELECT 1 FROM automation_events e
                  WHERE e.workspace_id = c.workspace_id AND e.contact_id = c.id AND e.kind = 'date_field'
                    AND e.ref = @field::text AND e.occurred_at >= @today::date)
LIMIT 5000;

-- name: ContactByEmail :one
SELECT id, status FROM contacts WHERE workspace_id = @workspace_id AND email = @email AND erased_at IS NULL;

-- name: ContactInWorkspace :one
SELECT id, status FROM contacts WHERE workspace_id = @workspace_id AND id = @id AND erased_at IS NULL;

-- name: LookupWorkspaceLists :many
SELECT id, name FROM lists WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY created_at;

-- name: ActiveRunNodes :many
-- Where contacts are, per version and step (publish impact).
SELECT r.version, r.node_id, r.status, r.state, count(*)::bigint AS n
FROM automation_runs r
WHERE r.automation_id = @automation_id AND r.status IN ('active', 'waiting')
GROUP BY r.version, r.node_id, r.status, r.state;

-- name: ActiveRunVersions :many
-- Live flows' versions that still have contacts in them (goal detection).
SELECT DISTINCT r.automation_id, r.version, a.workspace_id, v.graph
FROM automation_runs r
JOIN automations a ON a.id = r.automation_id
JOIN automation_versions v ON v.automation_id = r.automation_id AND v.version = r.version
WHERE r.status IN ('active', 'waiting') AND a.status = 'live' AND a.deleted_at IS NULL;

-- name: GoalRunsEngaged :many
-- Runs whose contact opened or clicked an email of this flow since entering.
SELECT r.id, r.node_id FROM automation_runs r
WHERE r.automation_id = @automation_id AND r.version = @version AND r.status IN ('active', 'waiting')
  AND EXISTS (
    SELECT 1 FROM delivery_events de
    JOIN messages m ON m.id = de.message_id
    WHERE de.contact_id = r.contact_id AND m.automation_id = r.automation_id AND de.occurred_at >= r.entered_at
      AND ((@want::text = 'opened' AND de.type = 'open' AND (NOT @human_only::boolean OR NOT de.machine))
        OR (@want::text = 'clicked' AND de.type = 'click' AND (@url_contains::text = '' OR de.url ILIKE '%' || @url_contains::text || '%'))))
LIMIT 1000
FOR UPDATE OF r SKIP LOCKED;

-- name: GoalRunsTagged :many
SELECT r.id, r.node_id FROM automation_runs r
JOIN contacts c ON c.id = r.contact_id
WHERE r.automation_id = @automation_id AND r.version = @version AND r.status IN ('active', 'waiting')
  AND @tag::text = ANY (c.tags)
LIMIT 1000
FOR UPDATE OF r SKIP LOCKED;

-- name: GoalRunsInList :many
SELECT r.id, r.node_id FROM automation_runs r
WHERE r.automation_id = @automation_id AND r.version = @version AND r.status IN ('active', 'waiting')
  AND EXISTS (SELECT 1 FROM list_memberships lm WHERE lm.list_id = @list_id AND lm.contact_id = r.contact_id AND lm.status = 'subscribed')
LIMIT 1000
FOR UPDATE OF r SKIP LOCKED;

-- name: GoalRunsEvent :many
SELECT r.id, r.node_id FROM automation_runs r
JOIN automations a ON a.id = r.automation_id
WHERE r.automation_id = @automation_id AND r.version = @version AND r.status IN ('active', 'waiting')
  AND EXISTS (SELECT 1 FROM automation_events e
              WHERE e.workspace_id = a.workspace_id AND e.contact_id = r.contact_id AND e.kind = 'api_event'
                AND e.ref = @event::text AND e.occurred_at >= r.entered_at)
LIMIT 1000
FOR UPDATE OF r SKIP LOCKED;

-- name: LiveDateTriggers :many
SELECT a.id, a.workspace_id, a.trigger, w.timezone
FROM automations a
JOIN workspaces w ON w.id = a.workspace_id
WHERE a.status = 'live' AND a.deleted_at IS NULL AND a.trigger ->> 'kind' = 'date_field';

-- name: LostWebhookRuns :many
-- Runs waiting on a webhook call whose job never ran (lost or not enqueued).
SELECT id, automation_id, version, node_id FROM automation_runs
WHERE status = 'waiting' AND next_at IS NULL AND state ->> 'wait' = 'webhook' AND node_entered_at < @before
LIMIT 500;

-- name: TouchRunNextAt :exec
UPDATE automation_runs SET next_at = @next_at WHERE id = @id;

-- name: MessageForStats :one
SELECT m.automation_id, m.automation_node, a.live_version, a.draft_version
FROM messages m
JOIN automations a ON a.id = m.automation_id
WHERE m.id = @id;
