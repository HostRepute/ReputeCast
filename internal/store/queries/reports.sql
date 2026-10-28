-- Campaign reports (spec 7.7, 5.5) and read-only report links.

-- name: ReportCampaign :one
-- send_ab is the send's A/B state (when the test is decided, the decision).
SELECT c.id, c.workspace_id, c.name, c.status, c.subject, c.audience, c.audience_count, c.ab_test, c.schedule,
       c.html, c.scheduled_at, c.started_at, c.completed_at, c.origin, c.note,
       coalesce(s.ab, '{}'::jsonb)::jsonb AS send_ab
FROM campaigns c
LEFT JOIN campaign_sends s ON s.campaign_id = c.id
WHERE c.workspace_id = @workspace_id AND c.id = @id AND c.deleted_at IS NULL;

-- name: ReportCounterCells :many
-- Final or live counters per sending provider × mailbox provider × A/B
-- variant, with the provider's name for the per-provider comparison.
SELECT cc.provider_id, cc.mailbox_provider, cc.variant,
       coalesce(p.name, '')::text AS provider_name, coalesce(p.adapter, '')::text AS adapter,
       cc.sent, cc.delivered, cc.deferred, cc.hard_bounced, cc.soft_bounced, cc.complained, cc.rejected,
       cc.opened_human, cc.opened_privacy, cc.opened_machine, cc.clicked, cc.unsubscribed
FROM campaign_counters cc
LEFT JOIN providers p ON p.id = cc.provider_id
WHERE cc.campaign_id = @campaign_id
ORDER BY cc.provider_id, cc.mailbox_provider, cc.variant;

-- name: ReportRoutes :many
-- Each sending provider's share of the send: what was planned, the cost and
-- the median time to delivery (stored at completion, else computed from the
-- route's messages; 0 before anything was delivered).
SELECT cr.provider_id, p.name AS provider_name, p.adapter, cr.planned, cr.status, cr.cost_micros,
       coalesce(cr.delivery_p50_ms,
                (SELECT (percentile_cont(0.5) WITHIN GROUP (ORDER BY extract(epoch FROM m.delivered_at - m.sent_at)) * 1000)::integer
                 FROM messages m
                 WHERE m.campaign_id = cr.campaign_id AND m.provider_id = cr.provider_id
                   AND m.delivered_at IS NOT NULL AND m.sent_at IS NOT NULL AND m.delivered_at >= m.sent_at),
                0)::integer AS delivery_p50_ms
FROM campaign_routes cr
JOIN providers p ON p.id = cr.provider_id
WHERE cr.campaign_id = @campaign_id
ORDER BY cr.planned DESC, p.name;

-- name: ReportHourly :many
SELECT hour, opened_human, opened_privacy, opened_machine, clicked
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

-- name: ReportRecentIssues :many
-- The last issues before a campaign to the same audience (sent email
-- campaigns, not deleted, to the same lists and segments in any order),
-- with their counters per mailbox provider, for the tiles' comparison with
-- the median of the last issues (review AN-11): another list's sends
-- aren't this newsletter's issues.
WITH this AS (
    SELECT coalesce(t.audience -> 'lists', '[]') AS lists, coalesce(t.audience -> 'segments', '[]') AS segments
    FROM campaigns t WHERE t.id = @campaign_id
), recent AS (
    SELECT c.id, c.started_at
    FROM campaigns c, this
    WHERE c.workspace_id = @workspace_id AND c.id <> @campaign_id AND c.deleted_at IS NULL AND c.channel = 'email'
      AND c.status = 'sent' AND c.started_at IS NOT NULL AND c.started_at < @before
      AND coalesce(c.audience -> 'lists', '[]') @> this.lists AND this.lists @> coalesce(c.audience -> 'lists', '[]')
      AND coalesce(c.audience -> 'segments', '[]') @> this.segments AND this.segments @> coalesce(c.audience -> 'segments', '[]')
      AND EXISTS (SELECT 1 FROM campaign_counters x WHERE x.campaign_id = c.id AND x.sent > 0)
    ORDER BY c.started_at DESC
    LIMIT @lim
)
SELECT r.id AS campaign_id, cc.mailbox_provider,
       sum(cc.sent)::bigint AS sent, sum(cc.delivered)::bigint AS delivered, sum(cc.hard_bounced)::bigint AS hard_bounced,
       sum(cc.soft_bounced)::bigint AS soft_bounced, sum(cc.complained)::bigint AS complained, sum(cc.opened_human)::bigint AS opened_human,
       sum(cc.opened_privacy)::bigint AS opened_privacy, sum(cc.opened_machine)::bigint AS opened_machine, sum(cc.clicked)::bigint AS clicked,
       sum(cc.unsubscribed)::bigint AS unsubscribed
FROM recent r
JOIN campaign_counters cc ON cc.campaign_id = r.id
GROUP BY r.id, r.started_at, cc.mailbox_provider
ORDER BY r.started_at DESC, cc.mailbox_provider;

-- name: ReportGrowthJoins :many
-- Contacts who joined since a campaign started through a form or the
-- subscribe page, with their first touch's page, referrer and UTM medium
-- and campaign: the issue's forwards and web-post signups are read from
-- them (review AN-12).
SELECT g.contact_id, g.page, g.referrer, g.utm_medium, g.utm_campaign
FROM contact_growth g
WHERE g.workspace_id = @workspace_id AND g.joined_at >= @since AND g.source_kind IN ('form', 'subscribe_page')
  AND (g.referrer LIKE '%/view/%' OR g.page LIKE '%/archive/%' OR g.referrer LIKE '%/archive/%' OR g.page LIKE @post_path OR g.referrer LIKE @post_path
       OR lower(g.utm_medium) = 'forward')
ORDER BY g.joined_at
LIMIT 20000;

-- name: ReportViewedMessages :many
-- Which of these messages are a campaign's (view-online pages a signup
-- came from).
SELECT id FROM messages WHERE campaign_id = @campaign_id AND id = ANY (@ids::uuid[]);

-- name: ReportGrowthReferrals :one
-- Referral signups since a campaign started credited to it: it was the
-- last email their referrer clicked, as a person, in the 7 days before the
-- signup (the attribution rule upgrades follow, review AN-12).
SELECT count(DISTINCT rf.id)::bigint
FROM referrals rf
JOIN messages m ON m.contact_id = rf.referrer_id AND m.campaign_id = @campaign_id
WHERE rf.workspace_id = @workspace_id AND rf.signed_up_at >= @since AND rf.status <> 'not_counted'
  AND m.clicked_at IS NOT NULL AND m.clicked_at <= rf.signed_up_at AND m.clicked_at > rf.signed_up_at - interval '7 days'
  AND NOT EXISTS (
      SELECT 1 FROM messages o
      WHERE o.contact_id = rf.referrer_id AND o.campaign_id IS DISTINCT FROM m.campaign_id
        AND o.clicked_at > m.clicked_at AND o.clicked_at <= rf.signed_up_at);

-- name: ReportIssuePages :one
-- Where a campaign is on the web: its archive page and its website post.
SELECT coalesce((SELECT ai.slug FROM archive_issues ai WHERE ai.workspace_id = @workspace_id AND ai.campaign_id = @campaign_id), '')::text AS archive_slug,
       coalesce((SELECT p.slug FROM website_posts p WHERE p.campaign_id = @campaign_id AND p.deleted_at IS NULL LIMIT 1), '')::text AS post_slug;

-- name: ReportWebViews :one
-- First-party, cookie-free views of a campaign on the web: its archive
-- page and its website post (review AN-13).
SELECT coalesce(sum(v.views), 0)::bigint
FROM website_view_days v
WHERE v.workspace_id = @workspace_id
  AND ((v.page_kind = 'issue' AND v.page_id = @campaign_id)
    OR (v.page_kind = 'post' AND v.page_id IN (SELECT p.id FROM website_posts p WHERE p.campaign_id = @campaign_id)));

-- name: ReportBlockedBounces :many
-- A campaign's bounces for the sender's reputation or policy by mailbox
-- provider and sending provider, with the provider's addresses (the
-- report's insights, review AN-15).
SELECT e.mailbox_provider, e.provider_id, coalesce(p.name, '')::text AS provider_name, coalesce(p.adapter, '')::text AS adapter,
       coalesce(p.ips, '{}')::text[] AS ips, count(*)::bigint AS n
FROM delivery_events e
LEFT JOIN providers p ON p.id = e.provider_id
WHERE e.campaign_id = @campaign_id AND e.workspace_id = @workspace_id AND e.type IN ('hard_bounce', 'soft_bounce')
  AND e.bounce_class IN ('policy_block', 'spam_block', 'reputation')
GROUP BY e.mailbox_provider, e.provider_id, p.name, p.adapter, p.ips
ORDER BY n DESC;

-- name: ReportLinkVariants :many
-- Clicks per link and A/B variant, for each variant's click map (review
-- AN-5).
SELECT position, variant, clicks, unique_clicks, bot_clicks
FROM campaign_link_variants
WHERE campaign_id = @campaign_id
ORDER BY variant, position;

-- name: ReportSendContent :one
-- The email as the send captured it: each content variant's HTML and the
-- first link position of each (internal/sending/engine sendContent).
SELECT coalesce(content, '{}'::jsonb)::jsonb AS content FROM campaign_sends WHERE campaign_id = @campaign_id;

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

-- name: ReportLeftOutInDoubt :one
-- A campaign's messages left out while in doubt (review GS-1): nobody knew
-- whether their provider took them, and a person chose not to send them
-- again. The report says so in its header ("not sent, in doubt").
SELECT count(*)::bigint FROM messages WHERE campaign_id = @campaign_id AND doubt_at IS NOT NULL AND status = 'cancelled';

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
INSERT INTO report_shares (id, campaign_id, token_hash, token_sealed, created_by, created_at, expires_at, include_cost, include_warmup, include_email)
VALUES (@id, @campaign_id, @token_hash, @token_sealed, @created_by, @created_at, @expires_at, @include_cost, @include_warmup, @include_email)
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
SELECT rs.id, rs.campaign_id, rs.expires_at, rs.revoked_at, rs.include_cost, rs.include_warmup, rs.include_email,
       c.workspace_id, w.slug AS workspace_slug, w.name AS workspace_name
FROM report_shares rs
JOIN campaigns c ON c.id = rs.campaign_id AND c.deleted_at IS NULL
JOIN workspaces w ON w.id = c.workspace_id AND w.deleted_at IS NULL
WHERE rs.token_hash = @token_hash;

-- name: ReportShareTouch :exec
UPDATE report_shares SET views = views + 1, last_viewed_at = @at WHERE id = @id;

-- name: ReportProviderTracked :one
-- Whether a sending provider reported opens or clicks of a campaign's
-- messages: its own tracking counted them (review AN-4).
SELECT EXISTS (
    SELECT 1 FROM delivery_events
    WHERE campaign_id = sqlc.arg(campaign_id)::uuid AND provider_id = sqlc.arg(provider_id)::uuid AND type IN ('open', 'click')
      AND original_type NOT IN ('', 'ReputeCast tracking')
)::boolean AS reported;

-- name: ReportFinalAlertSet :exec
-- Someone asked to be emailed when a report's results are final.
INSERT INTO report_final_alerts (campaign_id, user_id, workspace_id) VALUES (@campaign_id, @user_id, @workspace_id)
ON CONFLICT (campaign_id, user_id) DO NOTHING;

-- name: ReportFinalAlertRemove :execrows
DELETE FROM report_final_alerts WHERE campaign_id = @campaign_id AND user_id = @user_id;

-- name: ReportFinalAlertIs :one
SELECT EXISTS (SELECT 1 FROM report_final_alerts WHERE campaign_id = @campaign_id AND user_id = @user_id)::boolean AS asked;

-- name: ReportFinalAlertsDue :many
-- Requests whose campaign's results became final (72 hours after the send
-- ended), with what the notification says; the oldest first.
SELECT a.campaign_id, a.user_id, a.workspace_id, c.name AS campaign_name, w.slug AS workspace_slug, c.completed_at
FROM report_final_alerts a
JOIN campaigns c ON c.id = a.campaign_id
JOIN workspaces w ON w.id = a.workspace_id
WHERE c.completed_at IS NOT NULL AND c.completed_at <= sqlc.arg(final_before)::timestamptz
ORDER BY a.created_at
LIMIT 500;
