-- name: ListBlocklistResults :many
-- Every blocklist result of a workspace, most important first: listings,
-- then by the source's rank.
SELECT * FROM blocklist_results
WHERE workspace_id = @workspace_id
ORDER BY via, listed DESC, rank, source, asset;

-- name: GetBlocklistListing :one
SELECT * FROM blocklist_results
WHERE workspace_id = @workspace_id AND via = @via AND asset = @asset AND source = @source;

-- name: SetBlocklistDelisting :exec
UPDATE blocklist_results SET delisting = @delisting, delisting_requested_at = @requested_at
WHERE workspace_id = @workspace_id AND via = @via AND asset = @asset AND source = @source;

-- name: ListPostmasterDays :many
SELECT * FROM postmaster_daily
WHERE workspace_id = @workspace_id AND day >= @since::date
ORDER BY domain, day;

-- name: ListSNDSDays :many
SELECT * FROM snds_daily
WHERE workspace_id = @workspace_id AND day >= @since::date
ORDER BY ip, day;

-- name: ListDMARCReports :many
SELECT * FROM dmarc_reports
WHERE workspace_id = @workspace_id AND date_end >= @since
ORDER BY date_end DESC, org_name;

-- name: ListDeliverabilityIncidents :many
-- The incident timeline. An incident without an explicit campaign is linked
-- to the campaign that was sending through its provider when it happened.
SELECT i.id, i.occurred_at, i.kind, i.severity, i.title, i.detail, i.source, i.asset, i.resolved_at,
       i.provider_id, coalesce(p.name, '')::text AS provider_name, coalesce(p.adapter, '')::text AS provider_adapter,
       coalesce(i.campaign_id, lc.id, '00000000-0000-0000-0000-000000000000'::uuid) AS campaign_id,
       coalesce(c.name, lc.name, '')::text AS campaign_name,
       coalesce(c.status, lc.status, '')::text AS campaign_status
FROM incidents i
LEFT JOIN providers p ON p.id = i.provider_id
LEFT JOIN campaigns c ON c.id = i.campaign_id
LEFT JOIN LATERAL (
    SELECT c2.id, c2.name, c2.status FROM campaigns c2
    JOIN campaign_routes cr ON cr.campaign_id = c2.id AND cr.provider_id = i.provider_id
    WHERE i.campaign_id IS NULL AND i.provider_id IS NOT NULL
      AND i.kind IN ('auto_pause', 'warmup_slowdown', 'provider_outage')
      AND c2.workspace_id = i.workspace_id AND c2.deleted_at IS NULL
      AND c2.started_at <= i.occurred_at
      AND coalesce(c2.completed_at, i.occurred_at) >= i.occurred_at - interval '30 minutes'
    ORDER BY c2.started_at DESC LIMIT 1
) lc ON true
WHERE i.workspace_id = @workspace_id AND i.occurred_at >= @since
ORDER BY i.occurred_at DESC
LIMIT @lim;

-- name: CountDeliverabilityIncidents :one
SELECT count(*) FROM incidents WHERE workspace_id = @workspace_id AND occurred_at >= @since;

-- name: CreateCheckRun :one
INSERT INTO deliverability_check_runs (id, workspace_id, via, trigger, status, assets, total, requested_by, started_at)
VALUES (@id, @workspace_id, @via, @trigger, 'running', @assets, @total, @requested_by, @started_at)
RETURNING *;

-- name: GetCheckRun :one
SELECT * FROM deliverability_check_runs WHERE id = @id AND workspace_id = @workspace_id;

-- name: LatestCheckRun :one
SELECT * FROM deliverability_check_runs WHERE workspace_id = @workspace_id ORDER BY started_at DESC LIMIT 1;

-- name: LatestFinishedCheckRun :one
SELECT * FROM deliverability_check_runs
WHERE workspace_id = @workspace_id AND via = @via AND status = 'done'
ORDER BY finished_at DESC LIMIT 1;

-- name: RunningCheckRun :one
-- A run still in progress (runs older than ten minutes are considered dead).
SELECT * FROM deliverability_check_runs
WHERE workspace_id = @workspace_id AND status = 'running' AND started_at > @alive_after
ORDER BY started_at DESC LIMIT 1;

-- name: UpdateCheckRunProgress :exec
UPDATE deliverability_check_runs SET total = @total, answered = @answered WHERE id = @id;

-- name: FinishCheckRun :exec
UPDATE deliverability_check_runs SET status = @status, answered = @answered, total = @total, listed = @listed, error = @error, finished_at = @finished_at
WHERE id = @id;

-- name: FailStaleCheckRuns :execrows
UPDATE deliverability_check_runs SET status = 'failed', error = 'The check stopped answering.', finished_at = now()
WHERE status = 'running' AND started_at < @alive_after;

-- name: UpsertAssetCheck :exec
INSERT INTO deliverability_asset_checks (workspace_id, asset, asset_type, result, checked_at)
VALUES (@workspace_id, @asset, @asset_type, @result, @checked_at)
ON CONFLICT (workspace_id, asset) DO UPDATE SET asset_type = EXCLUDED.asset_type, result = EXCLUDED.result, checked_at = EXCLUDED.checked_at;

-- name: ListAssetChecks :many
SELECT * FROM deliverability_asset_checks WHERE workspace_id = @workspace_id ORDER BY asset;

-- name: ListActiveWarmupPlans :many
SELECT w.provider_id, w.ip, w.start_date, w.days, w.status FROM warmup_plans w
JOIN providers p ON p.id = w.provider_id AND p.workspace_id = @workspace_id AND p.disconnected_at IS NULL
WHERE w.status <> 'done';

-- name: ListProviderStreams :many
-- Which streams each provider serves, as primary (in a split) or fallback.
SELECT DISTINCT r.stream, (s ->> 'provider_id')::uuid AS provider_id, false AS fallback
FROM routing_rules r, jsonb_array_elements(r.splits) s
WHERE r.workspace_id = @workspace_id AND r.enabled
UNION
SELECT DISTINCT r.stream, r.fallback_provider_id AS provider_id, true AS fallback
FROM routing_rules r
WHERE r.workspace_id = @workspace_id AND r.enabled AND r.fallback_provider_id IS NOT NULL;

-- name: WorkspacesWithSendingSetup :many
-- Workspaces the daily checks look at: they send through something.
SELECT w.id FROM workspaces w
WHERE w.deleted_at IS NULL
  AND (EXISTS (SELECT 1 FROM sending_domains d WHERE d.workspace_id = w.id)
       OR EXISTS (SELECT 1 FROM providers p WHERE p.workspace_id = w.id AND p.disconnected_at IS NULL));

-- name: LastAssetCheckAt :one
SELECT coalesce(max(checked_at), 'epoch'::timestamptz)::timestamptz AS checked_at FROM deliverability_asset_checks WHERE workspace_id = @workspace_id;
