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

-- name: OverviewListChanges :many
-- Joins and departures across the workspace's lists in each 24-hour step
-- since a time, counted the way List health counts a list's net growth
-- (ListGrowthSince): joins of members still subscribed, and departures.
SELECT floor(extract(epoch FROM (x.at - @since::timestamptz)) / 86400)::int AS step,
       count(*) FILTER (WHERE x.joined)::bigint AS joined,
       count(*) FILTER (WHERE NOT x.joined)::bigint AS departed
FROM (
    SELECT lm.subscribed_at AS at, true AS joined
    FROM list_memberships lm
    JOIN lists l ON l.id = lm.list_id AND l.workspace_id = @workspace_id AND l.deleted_at IS NULL
    WHERE lm.status = 'subscribed' AND lm.subscribed_at >= @since::timestamptz
    UNION ALL
    SELECT lm.unsubscribed_at, false
    FROM list_memberships lm
    JOIN lists l ON l.id = lm.list_id AND l.workspace_id = @workspace_id AND l.deleted_at IS NULL
    WHERE lm.unsubscribed_at >= @since::timestamptz
) x
GROUP BY 1
ORDER BY 1;
