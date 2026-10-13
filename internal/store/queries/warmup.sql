-- W01 · Warm-up plans for new providers and dedicated IPs.

-- name: WarmupListPlans :many
-- Plans of a workspace's providers, current first.
SELECT w.*, p.name AS provider_name, p.adapter, p.workspace_id, p.ips AS provider_ips, p.status AS provider_status,
       coalesce(u.name, '')::text AS created_by_name
FROM warmup_plans w
JOIN providers p ON p.id = w.provider_id AND p.disconnected_at IS NULL
LEFT JOIN users u ON u.id = w.created_by
WHERE p.workspace_id = @workspace_id
ORDER BY CASE w.status WHEN 'done' THEN 1 ELSE 0 END, w.created_at DESC;

-- name: WarmupGetPlan :one
SELECT w.*, p.name AS provider_name, p.adapter, p.workspace_id
FROM warmup_plans w JOIN providers p ON p.id = w.provider_id
WHERE w.id = @id AND p.workspace_id = @workspace_id;

-- name: WarmupActivePlans :many
-- Every plan that is still running (daily warm-up job).
SELECT w.*, p.name AS provider_name, p.adapter, p.workspace_id
FROM warmup_plans w JOIN providers p ON p.id = w.provider_id AND p.disconnected_at IS NULL
WHERE w.status IN ('active', 'slowed');

-- name: WarmupActiveForProvider :one
SELECT * FROM warmup_plans WHERE provider_id = @provider_id AND status <> 'done' ORDER BY created_at DESC LIMIT 1;

-- name: WarmupInsertPlan :exec
INSERT INTO warmup_plans (id, provider_id, ip, start_date, days, schedule, engaged_first, slowdown_bounce, slowdown_complaint,
                          status, status_detail, progress, guard_action, spread_mailbox, created_by, created_at, updated_at)
VALUES (@id, @provider_id, @ip, @start_date, @days, @schedule, @engaged_first, @slowdown_bounce, @slowdown_complaint,
        'active', '', '[]', @guard_action, @spread_mailbox, @created_by, @created_at, @created_at);

-- name: WarmupUpdateSettings :exec
UPDATE warmup_plans
SET days = @days, schedule = @schedule, engaged_first = @engaged_first, slowdown_bounce = @slowdown_bounce,
    slowdown_complaint = @slowdown_complaint, guard_action = @guard_action, spread_mailbox = @spread_mailbox, updated_at = now()
WHERE id = @id;

-- name: WarmupSetStatus :exec
UPDATE warmup_plans SET status = @status, status_detail = @status_detail, paused_at = @paused_at, updated_at = now() WHERE id = @id;

-- name: WarmupSetProgress :exec
UPDATE warmup_plans SET progress = @progress, status = @status, status_detail = @status_detail, updated_at = now() WHERE id = @id;

-- name: WarmupLog :many
SELECT * FROM warmup_events WHERE plan_id = @plan_id ORDER BY occurred_at DESC, id DESC LIMIT @lim;

-- name: WarmupAddEvent :exec
INSERT INTO warmup_events (id, plan_id, occurred_at, kind, title, body) VALUES (@id, @plan_id, @occurred_at, @kind, @title, @body);

-- name: WarmupDailyStats :many
SELECT day, sent, delivered, bounced, complained FROM provider_daily_stats
WHERE provider_id = @provider_id AND day BETWEEN @since::date AND @until::date
ORDER BY day;

-- name: WarmupPlannedCampaignUse :many
-- Scheduled campaigns with a share on the provider (the plan table shows
-- "Weekly Digest #142 uses 3,160").
SELECT c.id, c.name, c.scheduled_at, cr.planned
FROM campaign_routes cr JOIN campaigns c ON c.id = cr.campaign_id
WHERE cr.provider_id = @provider_id AND c.status IN ('scheduled', 'approved', 'approval') AND c.scheduled_at IS NOT NULL
  AND c.scheduled_at BETWEEN @since AND @until AND c.deleted_at IS NULL
ORDER BY c.scheduled_at;
