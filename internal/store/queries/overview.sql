-- W09 · Overview (spec 7.1).

-- name: OverviewNextScheduled :one
-- The next campaign due to go out.
SELECT c.id, c.name, c.scheduled_at, c.schedule, c.audience_count
FROM campaigns c
WHERE c.workspace_id = @workspace_id AND c.deleted_at IS NULL AND c.status = 'scheduled'
  AND c.scheduled_at IS NOT NULL AND c.scheduled_at >= @since
ORDER BY c.scheduled_at, c.id
LIMIT 1;

-- name: OverviewDraftCount :one
SELECT count(*)::bigint FROM campaigns
WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND status IN ('draft', 'approval', 'approved');

-- name: OverviewLatestReadiness :one
SELECT verdict, checks, ran_at FROM readiness_runs WHERE campaign_id = @campaign_id ORDER BY ran_at DESC LIMIT 1;

-- name: OverviewCampaignRoutes :many
SELECT cr.provider_id, p.name AS provider_name, p.adapter, cr.planned, cr.cost_micros
FROM campaign_routes cr
JOIN providers p ON p.id = cr.provider_id
WHERE cr.campaign_id = @campaign_id
ORDER BY cr.planned DESC, p.name;

-- name: OverviewAudienceTotals :one
-- Contacts (not erased) by engagement tier.
SELECT count(*)::bigint AS contacts,
       count(*) FILTER (WHERE tier = 'active')::bigint AS active,
       count(*) FILTER (WHERE tier = 'cooling')::bigint AS cooling,
       count(*) FILTER (WHERE tier = 'at_risk')::bigint AS at_risk,
       count(*) FILTER (WHERE tier = 'sunset')::bigint AS sunset
FROM contacts
WHERE workspace_id = @workspace_id AND erased_at IS NULL;

-- name: OverviewContactsByDay :many
-- Contacts that existed at the end of each day of the period (created by
-- then and not erased before then).
SELECT d::date AS day,
       (SELECT count(*) FROM contacts c
        WHERE c.workspace_id = @workspace_id AND c.created_at < d + interval '1 day'
          AND (c.erased_at IS NULL OR c.erased_at >= d + interval '1 day'))::bigint AS contacts
FROM generate_series(@since::date, @until::date, interval '1 day') AS d
ORDER BY d;

-- name: OverviewListGrowth :many
-- Each list's subscribers and its joins and departures since a date.
SELECT l.id, l.name,
       count(lm.contact_id) FILTER (WHERE lm.status = 'subscribed')::bigint AS subscribed,
       count(lm.contact_id) FILTER (WHERE lm.status = 'subscribed' AND lm.subscribed_at >= @since)::bigint AS joined,
       count(lm.contact_id) FILTER (WHERE lm.unsubscribed_at >= @since)::bigint AS departed
FROM lists l
LEFT JOIN list_memberships lm ON lm.list_id = l.id
WHERE l.workspace_id = @workspace_id AND l.deleted_at IS NULL
GROUP BY l.id, l.name, l.created_at
ORDER BY l.created_at, l.id;
