-- Campaign reports (spec 7.7, 5.5) and read-only report links.

-- name: ReportCampaign :one
SELECT c.id, c.workspace_id, c.name, c.status, c.subject, c.audience, c.audience_count, c.ab_test, c.schedule,
       c.html, c.scheduled_at, c.started_at, c.completed_at, c.origin, c.note
FROM campaigns c
WHERE c.workspace_id = @workspace_id AND c.id = @id AND c.deleted_at IS NULL;

-- name: ReportCounterCells :many
-- Final or live counters per sending provider × mailbox provider × A/B
-- variant, with the provider's name for the per-provider comparison.
SELECT cc.provider_id, cc.mailbox_provider, cc.variant,
       coalesce(p.name, '')::text AS provider_name, coalesce(p.adapter, '')::text AS adapter,
       cc.sent, cc.delivered, cc.deferred, cc.hard_bounced, cc.soft_bounced, cc.complained, cc.rejected,
       cc.opened_human, cc.opened_machine, cc.clicked, cc.unsubscribed
FROM campaign_counters cc
LEFT JOIN providers p ON p.id = cc.provider_id
WHERE cc.campaign_id = @campaign_id
ORDER BY cc.provider_id, cc.mailbox_provider, cc.variant;

-- name: ReportRoutes :many
-- Each sending provider's share of the send: what was planned, the cost and
-- the median time to delivery (stored at completion, else computed from the
-- route's messages).
SELECT cr.provider_id, p.name AS provider_name, p.adapter, cr.planned, cr.status, cr.cost_micros,
       coalesce(cr.delivery_p50_ms,
                (SELECT (percentile_cont(0.5) WITHIN GROUP (ORDER BY extract(epoch FROM m.delivered_at - m.sent_at)) * 1000)::integer
                 FROM messages m
                 WHERE m.campaign_id = cr.campaign_id AND m.provider_id = cr.provider_id
                   AND m.delivered_at IS NOT NULL AND m.sent_at IS NOT NULL AND m.delivered_at >= m.sent_at))::integer AS delivery_p50_ms
FROM campaign_routes cr
JOIN providers p ON p.id = cr.provider_id
WHERE cr.campaign_id = @campaign_id
ORDER BY cr.planned DESC, p.name;

-- name: ReportHourly :many
SELECT hour, opened_human, opened_machine, clicked
FROM campaign_hourly
WHERE campaign_id = @campaign_id AND hour <= @max_hour
ORDER BY hour;

-- name: ReportLinks :many
-- Clicks per link: people's clicks, people's first click per link, and
-- the clicks the bot filter left out.
SELECT id, url, label, position, clicks, unique_clicks, bot_clicks
FROM campaign_links
WHERE campaign_id = @campaign_id
ORDER BY position, url;

-- name: ReportBounceCodes :many
-- Bounces of a campaign by taxonomy class, sending provider and the
-- provider's original event and code, with how many of the bounced
-- addresses are on the shared suppression list now.
SELECT e.bounce_class, e.type, e.provider_id,
       coalesce(p.name, '')::text AS provider_name, coalesce(p.adapter, '')::text AS adapter,
       e.original_type, e.original_code,
       count(*)::bigint AS n,
       count(DISTINCT e.recipient) FILTER (WHERE e.recipient <> '')::bigint AS recipients,
       count(DISTINCT e.recipient) FILTER (WHERE e.recipient <> '' AND EXISTS (
           SELECT 1 FROM suppressions s WHERE s.workspace_id = e.workspace_id AND s.email = e.recipient))::bigint AS suppressed
FROM delivery_events e
LEFT JOIN providers p ON p.id = e.provider_id
WHERE e.campaign_id = @campaign_id AND e.workspace_id = @workspace_id AND e.type IN ('hard_bounce', 'soft_bounce')
GROUP BY e.bounce_class, e.type, e.provider_id, p.name, p.adapter, e.original_type, e.original_code
ORDER BY n DESC, provider_name, e.original_code;

-- name: ReportBounceDiagnostics :many
-- The diagnostics behind one group of bounce classes (the Review dialog).
-- Addresses are never returned: an address inside a provider's words reads
-- "<address>", so the same answer for many recipients is one line.
SELECT d.provider_name, d.adapter, d.original_type, d.original_code, d.diagnostic, count(*)::bigint AS n
FROM (
    SELECT coalesce(p.name, '')::text AS provider_name, coalesce(p.adapter, '')::text AS adapter,
           e.original_type, e.original_code,
           regexp_replace(e.diagnostic, '[^[:space:]<>()"'',;:]+@[^[:space:]<>()"'',;:]+', '<address>', 'g')::text AS diagnostic
    FROM delivery_events e
    LEFT JOIN providers p ON p.id = e.provider_id
    WHERE e.campaign_id = @campaign_id AND e.workspace_id = @workspace_id AND e.type IN ('hard_bounce', 'soft_bounce')
      AND e.bounce_class = ANY(@classes::text[])
) d
GROUP BY d.provider_name, d.adapter, d.original_type, d.original_code, d.diagnostic
ORDER BY n DESC, d.provider_name
LIMIT 20;

-- name: ReportBounceRecipients :many
-- Bounced addresses of some classes that aren't suppressed yet (for "Add
-- to suppression"). Never sent to the browser.
SELECT DISTINCT e.recipient
FROM delivery_events e
WHERE e.campaign_id = @campaign_id AND e.workspace_id = @workspace_id AND e.type IN ('hard_bounce', 'soft_bounce')
  AND e.bounce_class = ANY(@classes::text[]) AND e.recipient <> ''
  AND NOT EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = e.workspace_id AND s.email = e.recipient)
ORDER BY e.recipient
LIMIT 5000;

-- name: ReportUnsubscribeReasons :many
SELECT reason, chose_fewer, count(*)::bigint AS n
FROM unsubscribe_feedback
WHERE campaign_id = @campaign_id AND workspace_id = @workspace_id
GROUP BY reason, chose_fewer;

-- name: ReportIngestBacklog :many
-- Provider events received but not normalized yet, per provider: when the
-- normalization job falls behind, live counts lag by that much.
SELECT r.provider_id, coalesce(p.name, '')::text AS provider_name, coalesce(p.adapter, '')::text AS adapter,
       count(*)::bigint AS waiting, min(r.received_at)::timestamptz AS oldest
FROM provider_raw_events r
LEFT JOIN providers p ON p.id = r.provider_id
WHERE r.workspace_id = @workspace_id AND r.normalized_at IS NULL AND r.signature_ok
  AND r.provider_id = ANY(@provider_ids::uuid[])
GROUP BY r.provider_id, p.name, p.adapter;

-- name: ReportWarmupAt :many
-- Warm-up plans of the campaign's providers (the day each was on when the
-- campaign was sent).
SELECT w.provider_id, w.start_date, w.days, w.engaged_first, w.status
FROM warmup_plans w
WHERE w.provider_id = ANY(@provider_ids::uuid[])
ORDER BY w.created_at DESC;

-- name: ReportLatestSent :one
-- The most recent campaign with results, for "See the last report instead".
SELECT c.id, c.name
FROM campaigns c
WHERE c.workspace_id = @workspace_id AND c.deleted_at IS NULL AND c.id <> @except_id
  AND c.status IN ('sent', 'sending', 'paused', 'cancelled', 'failed') AND c.started_at IS NOT NULL AND c.channel = 'email'
ORDER BY c.started_at DESC
LIMIT 1;

-- name: ReportLatestReadiness :one
SELECT verdict, ran_at FROM readiness_runs WHERE campaign_id = @campaign_id ORDER BY ran_at DESC LIMIT 1;

-- name: ReportShareCreate :one
INSERT INTO report_shares (id, campaign_id, token_hash, token_sealed, created_by, created_at, expires_at)
VALUES (@id, @campaign_id, @token_hash, @token_sealed, @created_by, @created_at, @expires_at)
RETURNING *;

-- name: ReportShareList :many
-- Active links of a campaign, newest first, with who made them.
SELECT rs.*, coalesce(u.name, '')::text AS created_by_name
FROM report_shares rs
LEFT JOIN users u ON u.id = rs.created_by
WHERE rs.campaign_id = @campaign_id AND rs.revoked_at IS NULL AND (rs.expires_at IS NULL OR rs.expires_at > @now)
ORDER BY rs.created_at DESC;

-- name: ReportShareGet :one
SELECT * FROM report_shares WHERE id = @id AND campaign_id = @campaign_id;

-- name: ReportShareRevoke :one
UPDATE report_shares SET revoked_at = @revoked_at, revoked_by = @revoked_by
WHERE id = @id AND campaign_id = @campaign_id AND revoked_at IS NULL
RETURNING *;

-- name: ReportShareByToken :one
-- A link by its token hash with its campaign and workspace (the public page).
SELECT rs.id, rs.campaign_id, rs.expires_at, rs.revoked_at, c.workspace_id, w.slug AS workspace_slug, w.name AS workspace_name
FROM report_shares rs
JOIN campaigns c ON c.id = rs.campaign_id AND c.deleted_at IS NULL
JOIN workspaces w ON w.id = c.workspace_id AND w.deleted_at IS NULL
WHERE rs.token_hash = @token_hash;

-- name: ReportShareTouch :exec
UPDATE report_shares SET views = views + 1, last_viewed_at = @at WHERE id = @id;
