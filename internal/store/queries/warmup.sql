-- Warm-up plans for new providers and dedicated IPs.

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
INSERT INTO warmup_plans (id, provider_id, ip, start_date, started_on, days, schedule, engaged_first, slowdown_bounce, slowdown_complaint,
                          status, status_detail, progress, guard_action, spread_mailbox, overflow, created_by, created_at, updated_at)
VALUES (@id, @provider_id, @ip, @start_date, @start_date, @days, @schedule, @engaged_first, @slowdown_bounce, @slowdown_complaint,
        'active', '', '[]', @guard_action, @spread_mailbox, @overflow, @created_by, @created_at, @created_at);

-- name: WarmupUpdateSettings :exec
UPDATE warmup_plans
SET days = @days, schedule = @schedule, engaged_first = @engaged_first, slowdown_bounce = @slowdown_bounce,
    slowdown_complaint = @slowdown_complaint, guard_action = @guard_action, spread_mailbox = @spread_mailbox,
    overflow = @overflow, updated_at = now()
WHERE id = @id;

-- name: WarmupMove :exec
-- PR-8: where the plan stands after the daily job judged its days: the
-- day that counts as day 1 now (start_date), the last day judged and a
-- slowdown's hold.
UPDATE warmup_plans
SET start_date = @start_date, evaluated_through = @evaluated_through, held_cap = sqlc.narg(held_cap), healthy_streak = @healthy_streak, updated_at = now()
WHERE id = @id;

-- name: WarmupResume :exec
-- PR-8: a paused plan resumes on the day it stopped: its days move on by
-- the days it was paused, which are never judged.
UPDATE warmup_plans
SET status = @status, status_detail = @status_detail, paused_at = NULL, start_date = @start_date, evaluated_through = @evaluated_through,
    texts = coalesce(sqlc.narg(texts)::jsonb, '{}'), updated_at = now()
WHERE id = @id;

-- name: WarmupAllowancePlan :one
-- The running plan of a provider with its workspace's time zone: warm-up
-- caps count the workspace's day, whatever the caller's clock (PR-8).
SELECT w.*, ws.timezone AS workspace_timezone
FROM warmup_plans w
JOIN providers p ON p.id = w.provider_id
JOIN workspaces ws ON ws.id = p.workspace_id
WHERE w.provider_id = @provider_id AND w.status <> 'done'
ORDER BY w.created_at DESC
LIMIT 1;

-- name: WarmupSentSince :many
-- What a provider sent since a time (the start of the workspace's day),
-- per mailbox provider: every stream counts toward the warm-up cap.
SELECT coalesce(nullif(m.mailbox_provider, ''), 'other')::text AS mailbox_provider, count(*)::bigint AS n
FROM messages m
WHERE m.provider_id = @provider_id AND m.sent_at >= @since::timestamptz
GROUP BY 1;

-- name: WarmupDayStats :many
-- One day of a warming provider (the workspace's day): sent, bounced,
-- complained and delivered per mailbox provider.
SELECT coalesce(nullif(m.mailbox_provider, ''), 'other')::text AS mailbox_provider,
       count(*)::bigint AS sent,
       count(*) FILTER (WHERE m.status = 'bounced')::bigint AS bounced,
       count(*) FILTER (WHERE m.status = 'complained')::bigint AS complained,
       count(*) FILTER (WHERE m.delivered_at IS NOT NULL OR m.status IN ('delivered', 'complained'))::bigint AS delivered
FROM messages m
WHERE m.provider_id = @provider_id AND m.sent_at >= @since::timestamptz AND m.sent_at < @until::timestamptz
GROUP BY 1;

-- name: WarmupSetStatus :exec
-- texts keeps the status detail's message for later readers ({} without one).
UPDATE warmup_plans SET status = @status, status_detail = @status_detail, paused_at = @paused_at,
    texts = coalesce(sqlc.narg(texts)::jsonb, '{}'), updated_at = now() WHERE id = @id;

-- name: WarmupSetProgress :exec
-- texts keeps the status detail's message for later readers ({} without one).
UPDATE warmup_plans SET progress = @progress, status = @status, status_detail = @status_detail,
    texts = coalesce(sqlc.narg(texts)::jsonb, '{}'), updated_at = now() WHERE id = @id;

-- name: WarmupLog :many
SELECT * FROM warmup_events WHERE plan_id = @plan_id ORDER BY occurred_at DESC, id DESC LIMIT @lim;

-- name: WarmupAddEvent :exec
-- texts keeps the title's and body's messages for later readers.
INSERT INTO warmup_events (id, plan_id, occurred_at, kind, title, body, texts)
VALUES (@id, @plan_id, @occurred_at, @kind, @title, @body, coalesce(sqlc.narg(texts)::jsonb, '{}'));

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

-- name: WarmupAddOverflow :exec
-- Messages over today's cap that went through a warm provider (PR-8).
UPDATE warmup_plans
SET overflowed = CASE WHEN overflowed_on = @day::date THEN overflowed + @n ELSE @n END, overflowed_on = @day::date
WHERE id = @id;

-- name: WarmupAddOverCap :exec
-- Transactional messages a warming provider sent over today's cap, or while
-- its plan was paused, because no warm provider of their route could take
-- them: transactional mail never waits (review R3.L6b).
UPDATE warmup_plans
SET over_cap = CASE WHEN over_cap_on = @day::date THEN over_cap + @n ELSE @n END, over_cap_on = @day::date
WHERE id = @id;

-- name: WarmupCapHolds :many
-- PR-8: campaign shares today's warm-up cap holds on a provider (route
-- held with reason warmup_cap), with what still waits.
SELECT cr.campaign_id, c.name AS campaign_name, c.sender_id, cr.provider_id, cr.status_detail, cr.planned,
       (SELECT count(*) FROM messages m WHERE m.campaign_id = cr.campaign_id AND m.provider_id = cr.provider_id AND m.status = 'queued')::bigint AS held
FROM campaign_routes cr JOIN campaigns c ON c.id = cr.campaign_id
WHERE cr.provider_id = @provider_id AND cr.status = 'held' AND cr.status_detail LIKE '%"reason":"warmup_cap"%'
ORDER BY c.name;

-- name: WarmupCapHoldsDue :many
-- Shares a warm-up cap held whose provider's new day has started (the
-- daily job sends them on), across workspaces.
SELECT cr.campaign_id, cr.provider_id, cr.status_detail
FROM campaign_routes cr
WHERE cr.status = 'held' AND cr.status_detail LIKE '%"reason":"warmup_cap"%';

-- name: WarmupResolveCapHold :exec
UPDATE campaign_incidents SET resolved_at = @at, resolution = @resolution, resolved_by = sqlc.narg(resolved_by)
WHERE campaign_id = @campaign_id AND provider_id = @provider_id AND kind = 'throttled' AND metric = 'warmup_cap' AND resolved_at IS NULL;
