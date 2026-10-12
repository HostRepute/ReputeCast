-- name: ListCampaigns :many
-- One page of the campaigns list. Status groups: drafts = draft, approval,
-- approved; sending = sending, paused; sent = sent, cancelled, failed.
SELECT c.id, c.name, c.status, c.audience, c.audience_count, c.ab_test, c.scheduled_at, c.schedule,
       c.created_at, c.updated_at, c.started_at, c.completed_at, c.origin, c.note,
       coalesce(u.name, '')::text AS updated_by_name,
       coalesce(t.sent, 0)::bigint AS sent,
       coalesce(t.delivered, 0)::bigint AS delivered,
       coalesce(t.opened_human, 0)::bigint AS opened_human,
       coalesce(t.clicked, 0)::bigint AS clicked,
       coalesce(r.verdict, '')::text AS readiness_verdict,
       coalesce(r.issues, 0)::bigint AS readiness_issues
FROM campaigns c
LEFT JOIN users u ON u.id = c.updated_by
LEFT JOIN LATERAL (
    SELECT sum(cc.sent) AS sent, sum(cc.delivered) AS delivered, sum(cc.opened_human) AS opened_human, sum(cc.clicked) AS clicked
    FROM campaign_counters cc WHERE cc.campaign_id = c.id
) t ON true
LEFT JOIN LATERAL (
    SELECT rr.verdict,
           (SELECT count(*) FROM jsonb_array_elements(coalesce(rr.checks -> 'checks', '[]'::jsonb)) e WHERE e ->> 'status' IN ('warn', 'block')) AS issues
    FROM readiness_runs rr WHERE rr.campaign_id = c.id ORDER BY rr.ran_at DESC LIMIT 1
) r ON c.status IN ('draft', 'approval', 'approved', 'scheduled')
WHERE c.workspace_id = @workspace_id AND c.deleted_at IS NULL
  AND (@status_group::text = 'all'
       OR (@status_group = 'drafts' AND c.status IN ('draft', 'approval', 'approved'))
       OR (@status_group = 'scheduled' AND c.status = 'scheduled')
       OR (@status_group = 'sending' AND c.status IN ('sending', 'paused'))
       OR (@status_group = 'sent' AND c.status IN ('sent', 'cancelled', 'failed')))
  AND (@query::text = '' OR c.name ILIKE '%' || @query::text || '%')
ORDER BY CASE c.status WHEN 'sending' THEN 0 WHEN 'paused' THEN 0 WHEN 'scheduled' THEN 1 WHEN 'approval' THEN 2 WHEN 'approved' THEN 2 WHEN 'draft' THEN 3 ELSE 4 END,
         CASE WHEN c.status = 'scheduled' THEN c.scheduled_at END ASC,
         coalesce(c.started_at, c.updated_at) DESC,
         c.id DESC
LIMIT @lim OFFSET @off;

-- name: CountCampaigns :one
SELECT count(*) FROM campaigns c
WHERE c.workspace_id = @workspace_id AND c.deleted_at IS NULL
  AND (@status_group::text = 'all'
       OR (@status_group = 'drafts' AND c.status IN ('draft', 'approval', 'approved'))
       OR (@status_group = 'scheduled' AND c.status = 'scheduled')
       OR (@status_group = 'sending' AND c.status IN ('sending', 'paused'))
       OR (@status_group = 'sent' AND c.status IN ('sent', 'cancelled', 'failed')))
  AND (@query::text = '' OR c.name ILIKE '%' || @query::text || '%');

-- name: CountCampaignsByStatus :many
SELECT status, count(*)::bigint AS n FROM campaigns
WHERE workspace_id = @workspace_id AND deleted_at IS NULL
GROUP BY status;

-- name: ListCampaignRouteShares :many
-- Per-provider share of each campaign: planned recipients and what was sent.
SELECT cr.campaign_id, p.id AS provider_id, p.name AS provider_name, p.adapter, cr.planned,
       coalesce((SELECT sum(cc.sent) FROM campaign_counters cc WHERE cc.campaign_id = cr.campaign_id AND cc.provider_id = cr.provider_id), 0)::bigint AS sent
FROM campaign_routes cr
JOIN providers p ON p.id = cr.provider_id
WHERE cr.campaign_id = ANY(@campaign_ids::uuid[])
ORDER BY cr.campaign_id, cr.planned DESC;

-- name: ListCampaignIncidentSummaries :many
SELECT campaign_id, kind, mailbox_provider, metric, count(*)::bigint AS n
FROM campaign_incidents
WHERE campaign_id = ANY(@campaign_ids::uuid[])
GROUP BY campaign_id, kind, mailbox_provider, metric;

-- name: CampaignTotalsSince :one
SELECT count(DISTINCT c.id)::bigint AS campaigns,
       coalesce(sum(cc.sent), 0)::bigint AS sent,
       coalesce(sum(cc.delivered), 0)::bigint AS delivered,
       coalesce(sum(cc.hard_bounced + cc.soft_bounced), 0)::bigint AS bounced,
       coalesce(sum(cc.complained), 0)::bigint AS complained,
       coalesce(sum(cc.opened_human), 0)::bigint AS opened_human,
       coalesce(sum(cc.clicked), 0)::bigint AS clicked
FROM campaigns c
JOIN campaign_counters cc ON cc.campaign_id = c.id
WHERE c.workspace_id = @workspace_id AND c.deleted_at IS NULL AND c.origin = '' AND c.started_at >= @since;

-- name: CampaignCostSince :many
SELECT p.name AS provider_name, p.adapter, coalesce(sum(cr.cost_micros), 0)::bigint AS cost_micros
FROM campaign_routes cr
JOIN campaigns c ON c.id = cr.campaign_id
JOIN providers p ON p.id = cr.provider_id
WHERE c.workspace_id = @workspace_id AND c.deleted_at IS NULL AND c.origin = '' AND c.started_at >= @since
GROUP BY p.name, p.adapter
ORDER BY cost_micros DESC, p.name;

-- name: ListWorkspaceLists :many
SELECT id, name FROM lists WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY created_at;

-- name: BroadcastDefaultSplit :one
SELECT splits FROM routing_rules
WHERE workspace_id = @workspace_id AND stream = 'broadcast' AND enabled AND conditions = '{}'::jsonb
ORDER BY priority LIMIT 1;

-- name: ProviderNames :many
SELECT id, name, adapter FROM providers WHERE workspace_id = @workspace_id AND disconnected_at IS NULL;
