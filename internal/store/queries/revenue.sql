-- Revenue and attribution (internal/revenue): the MRR ledger, the Earn
-- overview's numbers, the click token's settings and the conversions log.

-- name: RevenueMRRNow :one
-- The MRR definition: Active and Past due subscriptions, live only, after
-- discounts, a yearly price as a twelfth (twelve times MRR, in minor units).
SELECT coalesce(sum(earn_arr_cents(kind, mode, status, ended_at, period, amount_cents, discount)), 0)::bigint AS arr,
       count(*) FILTER (WHERE earn_arr_cents(kind, mode, status, ended_at, period, amount_cents, discount) > 0)::bigint AS members
FROM paid_subscriptions WHERE workspace_id = @workspace_id AND mode = 'live';

-- name: RevenueMovement :many
-- The ledger's changes after a moment: each kind's subscriptions and the
-- change of their yearly value.
SELECT kind, count(DISTINCT coalesce(subscription_id::text, id::text))::bigint AS subscriptions, coalesce(sum(arr_after - arr_before), 0)::bigint AS arr
FROM earn_mrr_changes
WHERE workspace_id = @workspace_id AND at > @from_at::timestamptz AND at <= @to_at::timestamptz
GROUP BY kind;

-- name: RevenueDeltaSince :many
-- The ledger's change after each of several moments (the MRR line).
SELECT p.at::timestamptz AS at, coalesce(sum(c.arr_after - c.arr_before), 0)::bigint AS arr,
       (count(*) FILTER (WHERE c.kind IN ('new','reactivation')) - count(*) FILTER (WHERE c.kind = 'churn'))::bigint AS members
FROM unnest(@points::timestamptz[]) AS p(at)
LEFT JOIN earn_mrr_changes c ON c.workspace_id = @workspace_id AND c.at > p.at
GROUP BY p.at ORDER BY p.at;

-- name: RevenueByTier :many
-- Paying subscriptions by tier, period and discount, today.
SELECT s.tier_id, coalesce(t.name, '')::text AS tier_name, coalesce(t.access_rank, 0)::integer AS access_rank, s.period, s.amount_cents,
       coalesce(s.discount->>'label', '')::text AS offer, coalesce((s.discount->>'amountCents')::bigint, s.amount_cents)::bigint AS paid_cents,
       count(*)::bigint AS members, coalesce(sum(earn_arr_cents(s.kind, s.mode, s.status, s.ended_at, s.period, s.amount_cents, s.discount)), 0)::bigint AS arr
FROM paid_subscriptions s LEFT JOIN earn_tiers t ON t.id = s.tier_id
WHERE s.workspace_id = @workspace_id AND s.mode = 'live' AND earn_arr_cents(s.kind, s.mode, s.status, s.ended_at, s.period, s.amount_cents, s.discount) > 0
GROUP BY 1, 2, 3, 4, 5, 6, 7
ORDER BY 3, 2, 4, 7 DESC;

-- name: RevenueAttributionSettings :one
SELECT attribution_days, click_token_on, click_token_domains, currency FROM earn_settings WHERE workspace_id = @workspace_id;

-- name: RevenueSetAttribution :exec
UPDATE earn_settings SET attribution_days = @attribution_days, click_token_on = @click_token_on, click_token_domains = @click_token_domains,
    updated_at = @at, updated_by = sqlc.narg(updated_by)
WHERE workspace_id = @workspace_id;

-- name: RevenueConversionInsert :exec
INSERT INTO conversions (id, workspace_id, api_key_id, key_name, name, value, value_cents, currency, occurred_at, received_at, contact_id, matched,
    message_id, campaign_id, automation_id, clicked_at, metadata, idempotency_key, request_hash, test)
VALUES (@id, @workspace_id, sqlc.narg(api_key_id), @key_name, @name, @value, @value_cents, @currency, @occurred_at, @received_at, sqlc.narg(contact_id), @matched,
    sqlc.narg(message_id), sqlc.narg(campaign_id), sqlc.narg(automation_id), sqlc.narg(clicked_at), @metadata, @idempotency_key, @request_hash, @test);

-- name: RevenueConversionByKey :one
SELECT * FROM conversions WHERE workspace_id = @workspace_id AND api_key_id = @api_key_id AND idempotency_key = @idempotency_key;

-- name: RevenueConversions :many
-- The conversions log, newest first, with what each was credited to.
SELECT c.*, coalesce(cp.name, a.name, '')::text AS credited_name
FROM conversions c
LEFT JOIN campaigns cp ON cp.id = c.campaign_id
LEFT JOIN automations a ON a.id = c.automation_id
WHERE c.workspace_id = @workspace_id AND (sqlc.narg(before)::timestamptz IS NULL OR c.received_at < sqlc.narg(before)::timestamptz)
ORDER BY c.received_at DESC, c.id DESC
LIMIT @lim;

-- name: RevenueConversionKeys :many
-- The keys that can record conversions, with who made them and their last use.
SELECT k.id, k.name, k.prefix, k.permissions, k.created_at, k.last_used_at, k.last_used_ip, k.expires_at, coalesce(u.name, '')::text AS created_by_name, k.mode
FROM api_keys k LEFT JOIN users u ON u.id = k.created_by
WHERE k.workspace_id = @workspace_id AND k.kind = 'api_key' AND k.revoked_at IS NULL AND 'conversions:create' = ANY(k.permissions)
ORDER BY k.created_at;

-- name: RevenueCampaignClickBump :exec
INSERT INTO earn_campaign_clicks (campaign_id, workspace_id, upgrade_clicks) VALUES (@campaign_id, @workspace_id, 1)
ON CONFLICT (campaign_id) DO UPDATE SET upgrade_clicks = earn_campaign_clicks.upgrade_clicks + 1;

-- name: RevenueCampaignClicks :one
SELECT coalesce((SELECT upgrade_clicks FROM earn_campaign_clicks WHERE campaign_id = @campaign_id), 0)::bigint;

-- name: RevenueLastEmailClick :one
-- A contact's last human click on a campaign or automation email within a
-- window before a moment.
SELECT e.message_id, m.campaign_id, m.automation_id, e.occurred_at
FROM delivery_events e JOIN messages m ON m.id = e.message_id
WHERE e.contact_id = @contact_id AND e.type = 'click' AND NOT e.machine AND e.occurred_at <= @at::timestamptz AND e.occurred_at >= @since::timestamptz
  AND m.stream IN ('broadcast','automation')
ORDER BY e.occurred_at DESC LIMIT 1;

-- name: RevenueContactByEmail :one
SELECT id FROM contacts WHERE workspace_id = @workspace_id AND lower(email) = lower(@email::text) AND erased_at IS NULL LIMIT 1;

-- name: RevenueMessageOf :one
SELECT m.id, m.workspace_id, m.campaign_id, m.automation_id, m.contact_id FROM messages m WHERE m.id = @id;
