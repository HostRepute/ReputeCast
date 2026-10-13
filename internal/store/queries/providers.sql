-- W01 · Sending providers: connected providers, health inputs, actions.
-- Query names start with Prov so they never collide with other domains.

-- name: ProvListActive :many
SELECT * FROM providers
WHERE workspace_id = @workspace_id AND disconnected_at IS NULL
ORDER BY connected_at, name;

-- name: ProvListAllActive :many
-- Every connected provider of every workspace (health and sync jobs).
SELECT p.* FROM providers p
JOIN workspaces w ON w.id = p.workspace_id AND w.deleted_at IS NULL
WHERE p.disconnected_at IS NULL
ORDER BY p.workspace_id, p.connected_at;

-- name: ProvGet :one
SELECT * FROM providers
WHERE workspace_id = @workspace_id AND id = @id AND disconnected_at IS NULL;

-- name: ProvGetAny :one
SELECT * FROM providers WHERE id = @id;

-- name: ProvGetByWebhookToken :one
SELECT * FROM providers
WHERE webhook_token = @webhook_token AND adapter = @adapter AND disconnected_at IS NULL;

-- name: ProvDailyTotals :many
-- Totals per provider over [since, until] (dates).
SELECT provider_id,
       coalesce(sum(sent), 0)::bigint        AS sent,
       coalesce(sum(delivered), 0)::bigint   AS delivered,
       coalesce(sum(bounced), 0)::bigint     AS bounced,
       coalesce(sum(complained), 0)::bigint  AS complained,
       coalesce(sum(api_calls), 0)::bigint   AS api_calls,
       coalesce(sum(api_errors), 0)::bigint  AS api_errors,
       coalesce(sum(cost_micros), 0)::bigint AS cost_micros
FROM provider_daily_stats
WHERE provider_id = ANY(@provider_ids::uuid[]) AND day BETWEEN @since::date AND @until::date
GROUP BY provider_id;

-- name: ProvCallWindowTotals :many
-- API calls, failures and messages per provider since a time (rolling windows).
SELECT provider_id,
       coalesce(sum(calls), 0)::bigint    AS calls,
       coalesce(sum(errors), 0)::bigint   AS errors,
       coalesce(sum(messages), 0)::bigint AS messages,
       coalesce(max(messages), 0)::bigint AS peak_minute
FROM provider_call_windows
WHERE provider_id = ANY(@provider_ids::uuid[]) AND minute >= @since
GROUP BY provider_id;

-- name: ProvRecordCall :exec
INSERT INTO provider_call_windows (provider_id, minute, calls, errors, messages)
VALUES (@provider_id, date_trunc('minute', @at::timestamptz), 1, @errors, @messages)
ON CONFLICT (provider_id, minute) DO UPDATE
SET calls = provider_call_windows.calls + 1,
    errors = provider_call_windows.errors + EXCLUDED.errors,
    messages = provider_call_windows.messages + EXCLUDED.messages;

-- name: ProvRecordDaily :exec
INSERT INTO provider_daily_stats (provider_id, day, sent, api_calls, api_errors, cost_micros)
VALUES (@provider_id, @day::date, @sent, 1, @errors, @cost_micros)
ON CONFLICT (provider_id, day) DO UPDATE
SET sent = provider_daily_stats.sent + EXCLUDED.sent,
    api_calls = provider_daily_stats.api_calls + 1,
    api_errors = provider_daily_stats.api_errors + EXCLUDED.api_errors,
    cost_micros = provider_daily_stats.cost_micros + EXCLUDED.cost_micros;

-- name: ProvRecordOutcome :exec
-- Delivery outcomes counted from normalized events (bounce and complaint rates).
INSERT INTO provider_daily_stats (provider_id, day, delivered, bounced, complained)
VALUES (@provider_id, @day::date, @delivered, @bounced, @complained)
ON CONFLICT (provider_id, day) DO UPDATE
SET delivered = provider_daily_stats.delivered + EXCLUDED.delivered,
    bounced = provider_daily_stats.bounced + EXCLUDED.bounced,
    complained = provider_daily_stats.complained + EXCLUDED.complained;

-- name: ProvPurgeCallWindows :execrows
DELETE FROM provider_call_windows WHERE minute < @before;

-- name: ProvUpdateHealth :exec
UPDATE providers
SET status = @status, status_detail = @status_detail, health = @health, last_checked_at = @checked_at
WHERE id = @id;

-- name: ProvUpdateCapabilities :exec
UPDATE providers SET capabilities = @capabilities WHERE id = @id;

-- name: ProvUpdateCredentials :exec
UPDATE providers SET credentials_enc = @credentials_enc WHERE id = @id;

-- name: ProvUpdateConfig :exec
UPDATE providers SET config = @config WHERE id = @id;

-- name: ProvUpdateSettings :exec
UPDATE providers SET label = @label, pricing = @pricing, limits = @limits WHERE id = @id;

-- name: ProvUpdateTracking :exec
UPDATE providers SET tracking = @tracking WHERE id = @id;

-- name: ProvDisconnect :exec
UPDATE providers SET disconnected_at = now(), status = 'paused', status_detail = 'Disconnected' WHERE id = @id;

-- name: ProvTouchLastEvent :exec
UPDATE providers SET last_event_at = greatest(coalesce(last_event_at, @at::timestamptz), @at::timestamptz) WHERE id = @id;

-- name: ProvSentSince :many
-- Messages handed to each provider since a time, and since the provider's
-- last webhook event (webhook freshness: mail went out but nothing came back).
SELECT m.provider_id,
       count(*)::bigint AS sent,
       count(*) FILTER (WHERE p.last_event_at IS NULL OR m.sent_at > p.last_event_at)::bigint AS since_last_event
FROM messages m
JOIN providers p ON p.id = m.provider_id
WHERE m.provider_id = ANY(@provider_ids::uuid[]) AND m.sent_at >= @since
GROUP BY m.provider_id;

-- name: ProvRawEventsByHour :many
-- Webhook events received per hour (webhook freshness chart).
SELECT date_trunc('hour', received_at)::timestamptz AS hour, count(*)::bigint AS n
FROM provider_raw_events
WHERE provider_id = @provider_id AND received_at >= @since AND source = 'webhook'
GROUP BY 1 ORDER BY 1;

-- name: ProvAwaitingStatus :many
-- Messages a provider accepted with no event since (newest first).
SELECT m.id, m.email, m.provider_message_id, m.sent_at, m.status, m.delivered_at, m.kind,
       coalesce(c.name, '')::text AS campaign_name,
       coalesce(a.name, '')::text AS automation_name,
       m.automation_node
FROM messages m
LEFT JOIN campaigns c ON c.id = m.campaign_id
LEFT JOIN automations a ON a.id = m.automation_id
WHERE m.provider_id = @provider_id AND m.sent_at IS NOT NULL AND m.sent_at >= @since
ORDER BY m.sent_at DESC, m.id DESC
LIMIT @lim;

-- name: ProvAwaitingCount :one
SELECT count(*)::bigint AS total,
       count(DISTINCT m.campaign_id)::bigint AS campaigns
FROM messages m
WHERE m.provider_id = @provider_id AND m.sent_at IS NOT NULL AND m.sent_at > @since
  AND m.status IN ('queued', 'sent');

-- name: ProvAwaitingCampaigns :many
SELECT DISTINCT c.id, c.name
FROM messages m JOIN campaigns c ON c.id = m.campaign_id
WHERE m.provider_id = @provider_id AND m.sent_at > @since AND m.status IN ('queued', 'sent')
ORDER BY c.name
LIMIT 5;

-- name: ProvAwaitingProviderIDs :many
SELECT provider_message_id::text AS provider_message_id
FROM messages
WHERE provider_id = @provider_id AND sent_at > @since AND status IN ('queued', 'sent') AND provider_message_id IS NOT NULL
ORDER BY sent_at
LIMIT @lim;

-- name: ProvReroutedByAutomation :many
-- Automation mail sent through a fallback since a time, per automation.
SELECT coalesce(a.name, 'other flows')::text AS name, count(*)::bigint AS n
FROM messages m
LEFT JOIN automations a ON a.id = m.automation_id
WHERE m.workspace_id = @workspace_id AND m.stream = 'automation' AND m.provider_id = @provider_id AND m.sent_at >= @since
GROUP BY 1 ORDER BY 2 DESC, 1;

-- name: ProvSentByKind :many
-- System mail sent through a provider since a time, per kind.
SELECT kind, count(*)::bigint AS n
FROM messages
WHERE workspace_id = @workspace_id AND stream = @stream AND provider_id = @provider_id AND sent_at >= @since
GROUP BY kind ORDER BY 2 DESC, 1;

-- name: ProvStreamVolume :many
-- Messages per stream over a window (volume estimates on the routing screen).
SELECT stream, count(*)::bigint AS n
FROM messages
WHERE workspace_id = @workspace_id AND sent_at >= @since AND sent_at < @until
GROUP BY stream;

-- name: ProvBroadcastWeekly :one
-- Broadcast recipients sent over the last weeks (campaign counters).
SELECT coalesce(sum(cc.sent), 0)::bigint AS sent
FROM campaign_counters cc
JOIN campaigns c ON c.id = cc.campaign_id
WHERE c.workspace_id = @workspace_id AND c.deleted_at IS NULL AND c.origin = '' AND c.started_at >= @since;

-- name: ProvHeldRoutes :many
-- Campaign shares held on a provider (Broadcasts waiting for a decision).
SELECT cr.campaign_id, cr.provider_id, cr.planned, cr.status, cr.status_detail,
       c.name AS campaign_name, c.status AS campaign_status, c.audience_count, c.started_at,
       coalesce((SELECT sum(cc.sent) FROM campaign_counters cc WHERE cc.campaign_id = cr.campaign_id AND cc.provider_id = cr.provider_id), 0)::bigint AS sent
FROM campaign_routes cr
JOIN campaigns c ON c.id = cr.campaign_id
WHERE c.workspace_id = @workspace_id AND cr.provider_id = @provider_id AND cr.status = 'held' AND c.deleted_at IS NULL
ORDER BY c.started_at DESC NULLS LAST;

-- name: ProvCampaignRoutes :many
SELECT cr.campaign_id, cr.provider_id, cr.planned, cr.status, cr.status_detail, p.name AS provider_name, p.adapter,
       coalesce((SELECT sum(cc.sent) FROM campaign_counters cc WHERE cc.campaign_id = cr.campaign_id AND cc.provider_id = cr.provider_id), 0)::bigint AS sent
FROM campaign_routes cr
JOIN providers p ON p.id = cr.provider_id
WHERE cr.campaign_id = @campaign_id
ORDER BY cr.planned DESC;

-- name: ProvSendingCampaignsOn :many
-- Campaigns sending through a provider right now.
SELECT DISTINCT c.id, c.name
FROM campaign_routes cr JOIN campaigns c ON c.id = cr.campaign_id
WHERE c.workspace_id = @workspace_id AND cr.provider_id = @provider_id AND c.status IN ('sending', 'paused')
  AND cr.status IN ('pending', 'sending') AND c.deleted_at IS NULL;

-- name: ProvSetRouteStatus :execrows
UPDATE campaign_routes SET status = @status, status_detail = @status_detail
WHERE campaign_id = @campaign_id AND provider_id = @provider_id;

-- name: ProvIncidentsSince :many
SELECT * FROM incidents
WHERE workspace_id = @workspace_id AND occurred_at >= @since
  AND (provider_id = ANY(@provider_ids::uuid[]) OR campaign_id = ANY(@campaign_ids::uuid[]))
ORDER BY occurred_at DESC, id DESC
LIMIT 12;

-- name: ProvInsertIncident :exec
INSERT INTO incidents (id, workspace_id, occurred_at, kind, severity, title, detail, campaign_id, provider_id)
VALUES (@id, @workspace_id, @occurred_at, @kind, @severity, @title, @detail, @campaign_id, @provider_id);

-- name: ProvResolveIncidents :exec
UPDATE incidents SET resolved_at = @resolved_at
WHERE provider_id = @provider_id AND kind = ANY(@kinds::text[]) AND resolved_at IS NULL;

-- name: ProvActiveFailovers :many
SELECT * FROM stream_failovers WHERE workspace_id = @workspace_id AND ended_at IS NULL ORDER BY started_at;

-- name: ProvStartFailover :exec
INSERT INTO stream_failovers (id, workspace_id, stream, provider_id, fallback_provider_id, mode, started_at, detail)
VALUES (@id, @workspace_id, @stream, @provider_id, @fallback_provider_id, @mode, @started_at, @detail)
ON CONFLICT (workspace_id, stream, provider_id) WHERE ended_at IS NULL DO NOTHING;

-- name: ProvEndFailovers :many
UPDATE stream_failovers SET ended_at = @ended_at, end_reason = @end_reason
WHERE provider_id = @provider_id AND ended_at IS NULL
RETURNING *;

-- name: ProvMembersWithNames :many
SELECT u.id, u.name FROM users u WHERE u.id = ANY(@ids::uuid[]);
