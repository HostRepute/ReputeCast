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
       coalesce(s.discount->>'duration', '')::text AS duration, coalesce((s.discount->>'months')::integer, 0)::integer AS months,
       count(*)::bigint AS members, coalesce(sum(earn_arr_cents(s.kind, s.mode, s.status, s.ended_at, s.period, s.amount_cents, s.discount)), 0)::bigint AS arr
FROM paid_subscriptions s LEFT JOIN earn_tiers t ON t.id = s.tier_id
WHERE s.workspace_id = @workspace_id AND s.mode = 'live' AND earn_arr_cents(s.kind, s.mode, s.status, s.ended_at, s.period, s.amount_cents, s.discount) > 0
GROUP BY 1, 2, 3, 4, 5, 6, 7, 8, 9
ORDER BY 3, 2, 4, 6;

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

-- name: RevenueLiveConnection :one
-- The connected live payment account, if there is one.
SELECT id, account_name, connected_at, last_event_at, demo, created_at FROM payment_connections
WHERE workspace_id = @workspace_id AND mode = 'live' AND status = 'connected' ORDER BY created_at DESC LIMIT 1;

-- name: RevenueLedgerSince :one
-- How many changes the MRR ledger has, and when it starts.
SELECT count(*)::bigint AS changes, coalesce(min(at), now())::timestamptz AS first_at FROM earn_mrr_changes WHERE workspace_id = @workspace_id;

-- name: RevenueMoved :one
-- The first finished import of paid members: the move from another platform.
SELECT source, coalesce(finished_at, created_at)::timestamptz AS at FROM earn_imports
WHERE workspace_id = @workspace_id AND mode = 'live' AND status = 'done' ORDER BY coalesce(finished_at, created_at) LIMIT 1;

-- name: RevenueOfferStarts :many
-- Signup offers that started in a window (marks on the MRR line).
SELECT id, name, started_at FROM earn_offers
WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND kind = 'signup' AND started_at >= @from_at::timestamptz AND started_at <= @to_at::timestamptz
ORDER BY started_at;

-- name: RevenueTrials :one
-- Trials started in a window and what became of them; trialing now and
-- ending within a week.
SELECT
    count(*) FILTER (WHERE trial_end IS NOT NULL AND started_at >= @from_at::timestamptz AND started_at < @to_at::timestamptz)::bigint AS started,
    count(*) FILTER (WHERE status = 'trialing')::bigint AS trialing,
    count(*) FILTER (WHERE trial_end IS NOT NULL AND started_at >= @from_at::timestamptz AND started_at < @to_at::timestamptz AND trial_end <= @to_at::timestamptz
        AND (status IN ('active','past_due') OR (ended_at IS NOT NULL AND end_reason <> 'trial_ended' AND ended_at > trial_end)))::bigint AS became_paid,
    count(*) FILTER (WHERE trial_end IS NOT NULL AND started_at >= @from_at::timestamptz AND started_at < @to_at::timestamptz
        AND ended_at IS NOT NULL AND (end_reason = 'trial_ended' OR ended_at <= trial_end))::bigint AS canceled,
    count(*) FILTER (WHERE status = 'trialing' AND trial_end <= @soon::timestamptz)::bigint AS ending_soon
FROM paid_subscriptions WHERE workspace_id = @workspace_id AND mode = 'live' AND kind = 'stripe';

-- name: RevenueTrialsEndingTop :many
-- The email most of the trials ending soon came from.
SELECT s.source_campaign_id, coalesce(cp.name, '')::text AS name, count(*)::bigint AS trials
FROM paid_subscriptions s JOIN campaigns cp ON cp.id = s.source_campaign_id
WHERE s.workspace_id = @workspace_id AND s.mode = 'live' AND s.status = 'trialing' AND s.trial_end <= @soon::timestamptz
GROUP BY 1, 2 ORDER BY 3 DESC LIMIT 1;

-- name: RevenueCancelSummary :one
-- Visits to the cancel page in a window and what readers decided.
SELECT count(*)::bigint AS visits,
       count(*) FILTER (WHERE decision = 'kept')::bigint AS kept,
       count(*) FILTER (WHERE decision = 'continued')::bigint AS continued,
       count(*) FILTER (WHERE decision = '')::bigint AS undecided
FROM earn_cancel_visits WHERE workspace_id = @workspace_id AND mode = 'live' AND visited_at >= @from_at::timestamptz AND visited_at < @to_at::timestamptz;

-- name: RevenueCancelReasons :many
SELECT reason, count(*)::bigint AS visits FROM earn_cancel_visits
WHERE workspace_id = @workspace_id AND mode = 'live' AND visited_at >= @from_at::timestamptz AND visited_at < @to_at::timestamptz
GROUP BY reason;

-- name: RevenueKeptWith :many
-- The offer and tier most readers kept their membership with.
SELECT coalesce(o.name, '')::text AS offer, coalesce(t.name, '')::text AS tier, count(*)::bigint AS kept
FROM earn_cancel_visits v
LEFT JOIN earn_offers o ON o.id = v.offer_id
LEFT JOIN paid_subscriptions s ON s.id = v.subscription_id
LEFT JOIN earn_tiers t ON t.id = s.tier_id
WHERE v.workspace_id = @workspace_id AND v.mode = 'live' AND v.decision = 'kept' AND v.visited_at >= @from_at::timestamptz AND v.visited_at < @to_at::timestamptz
GROUP BY 1, 2 ORDER BY 3 DESC LIMIT 1;

-- name: RevenueRetentionOffer :many
-- The live retention offer, for the Cancellations card's link.
SELECT id, name FROM earn_offers WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND kind = 'retention' AND status = 'live'
ORDER BY created_at LIMIT 1;

-- name: RevenueUpgradesByEmail :many
-- New paid members of a window by the email they're credited to, with
-- their new MRR (twelve times, in minor units).
SELECT s.source_kind, s.source_campaign_id, m.automation_id, coalesce(cp.name, a.name, s.source_label, '')::text AS name,
       coalesce(cp.started_at, cp.scheduled_at) AS sent_at,
       count(DISTINCT c.subscription_id)::bigint AS members, coalesce(sum(c.arr_after), 0)::bigint AS arr
FROM earn_mrr_changes c
JOIN paid_subscriptions s ON s.id = c.subscription_id
LEFT JOIN campaigns cp ON cp.id = s.source_campaign_id
LEFT JOIN messages m ON s.source_kind = 'automation' AND s.source_detail->>'message_id' ~ '^[0-9a-f-]{36}$' AND m.id = (s.source_detail->>'message_id')::uuid
LEFT JOIN automations a ON a.id = m.automation_id
WHERE c.workspace_id = @workspace_id AND c.kind IN ('new','reactivation') AND c.at >= @from_at::timestamptz AND c.at < @to_at::timestamptz
GROUP BY 1, 2, 3, 4, 5;

-- name: RevenueFreeToPaid :many
-- Free subscribers by how they first joined: those at the start of a
-- window, those who joined during it, and those who became paid members in
-- it. A contact who paid within an hour of joining paid straight away and
-- is counted apart.
WITH paid_at_start AS (
    SELECT DISTINCT contact_id FROM paid_subscriptions
    WHERE workspace_id = @workspace_id AND mode = 'live' AND status <> 'incomplete'
      AND started_at < @from_at::timestamptz AND (ended_at IS NULL OR ended_at >= @from_at::timestamptz)
), first_paid AS (
    SELECT contact_id, min(at) AS at FROM earn_mrr_changes
    WHERE workspace_id = @workspace_id AND kind = 'new' AND at >= @from_at::timestamptz AND at < @to_at::timestamptz
    GROUP BY contact_id
)
SELECT g.source_kind, g.source_id,
       count(*) FILTER (WHERE g.joined_at < @from_at::timestamptz AND (g.left_at IS NULL OR g.left_at >= @from_at::timestamptz) AND ps.contact_id IS NULL)::bigint AS at_start,
       count(*) FILTER (WHERE g.joined_at >= @from_at::timestamptz AND g.joined_at < @to_at::timestamptz
           AND (fp.contact_id IS NULL OR fp.at > g.joined_at + interval '1 hour'))::bigint AS joined,
       count(*) FILTER (WHERE fp.contact_id IS NOT NULL AND fp.at > g.joined_at + interval '1 hour' AND ps.contact_id IS NULL)::bigint AS paid
FROM contact_growth g
LEFT JOIN paid_at_start ps ON ps.contact_id = g.contact_id
LEFT JOIN first_paid fp ON fp.contact_id = g.contact_id
WHERE g.workspace_id = @workspace_id
GROUP BY 1, 2;

-- name: RevenueNewContacts :one
-- Contacts who became paid members in a window.
SELECT count(DISTINCT contact_id)::bigint FROM earn_mrr_changes
WHERE workspace_id = @workspace_id AND kind = 'new' AND at >= @from_at::timestamptz AND at < @to_at::timestamptz;

-- name: RevenueConversionsSummary :many
-- Conversions of a window by currency, the email credited and how each matched.
SELECT c.currency, c.campaign_id, c.automation_id, coalesce(cp.name, a.name, '')::text AS credited_name, c.matched,
       count(*)::bigint AS conversions, coalesce(sum(c.value_cents), 0)::bigint AS value_cents
FROM conversions c
LEFT JOIN campaigns cp ON cp.id = c.campaign_id
LEFT JOIN automations a ON a.id = c.automation_id
WHERE c.workspace_id = @workspace_id AND NOT c.test AND c.received_at >= @from_at::timestamptz AND c.received_at < @to_at::timestamptz
GROUP BY 1, 2, 3, 4, 5;

-- name: RevenueConversionKinds :many
-- What the conversions of a window are: their names, values and keys.
SELECT name, value_cents, currency, key_name, count(*)::bigint AS conversions FROM conversions
WHERE workspace_id = @workspace_id AND NOT test AND received_at >= @from_at::timestamptz AND received_at < @to_at::timestamptz
GROUP BY 1, 2, 3, 4 ORDER BY 5 DESC LIMIT 5;

-- name: RevenueConversionsFirst :one
SELECT count(*)::bigint AS conversions, coalesce(min(received_at), now())::timestamptz AS first_at FROM conversions
WHERE workspace_id = @workspace_id AND NOT test;

-- name: RevenueGiftIncome :one
-- Gifts paid in a window (refunded ones left out).
SELECT coalesce(sum(amount_cents), 0)::bigint AS cents, count(*)::bigint AS gifts FROM earn_gifts
WHERE workspace_id = @workspace_id AND mode = 'live' AND paid_at >= @from_at::timestamptz AND paid_at < @to_at::timestamptz
  AND status IN ('paid','delivered','redeemed','expired');

-- name: RevenueSponsorIncome :one
-- Sponsored issues of a window, at their price.
SELECT coalesce(sum(price_cents), 0)::bigint AS cents, count(*)::bigint AS issues FROM sponsor_bookings
WHERE workspace_id = @workspace_id AND status IN ('booked','sent') AND issue_on >= @from_day::date AND issue_on <= @to_day::date;

-- name: RevenuePastDue :one
SELECT count(*)::bigint FROM paid_subscriptions WHERE workspace_id = @workspace_id AND mode = 'live' AND status = 'past_due';

-- name: RevenueOffersToWatch :many
-- Live signup offers with a limit or an end date, and how often they were used.
SELECT o.id, o.name, o.code, o.redeem_limit, o.ends_at,
       (SELECT count(*) FROM paid_subscriptions s WHERE s.workspace_id = o.workspace_id AND s.mode = 'live' AND s.discount IS NOT NULL
          AND (s.discount->>'offer' = o.id::text OR (o.code <> '' AND upper(s.discount->>'label') = upper(o.code)) OR s.discount->>'label' = o.name))::bigint AS used
FROM earn_offers o
WHERE o.workspace_id = @workspace_id AND o.deleted_at IS NULL AND o.status = 'live' AND o.kind = 'signup' AND (o.redeem_limit IS NOT NULL OR o.ends_at IS NOT NULL)
ORDER BY o.ends_at NULLS LAST, o.created_at;

-- name: RevenueSponsorNeeds :many
-- Bookings that need a person soon: a creative still waiting, or a
-- sponsor's payment not recorded for an issue that went out.
SELECT b.id, b.issue_number, b.issue_on, b.creative_status, b.creative_due, b.payment_status, b.payment_due, b.status,
       coalesce(sp.name, '')::text AS sponsor, l.name AS list_name
FROM sponsor_bookings b
JOIN sponsor_slots sl ON sl.id = b.slot_id
JOIN lists l ON l.id = sl.list_id
LEFT JOIN sponsors sp ON sp.id = b.sponsor_id
WHERE b.workspace_id = @workspace_id AND (
    (b.status = 'booked' AND b.creative_status = 'waiting' AND b.creative_due <= @creative_until::date)
    OR (b.status IN ('booked','sent') AND b.payment_status = 'unpaid' AND b.issue_on < @today::date))
ORDER BY b.issue_on, b.issue_number;

-- name: RevenueUnpublishedMemberEmails :many
-- Member emails whose template isn't published.
SELECT e.email, t.id AS template_id, t.key FROM earn_member_emails e JOIN transactional_templates t ON t.id = e.template_id
WHERE e.workspace_id = @workspace_id AND t.deleted_at IS NULL AND t.live_version IS NULL
ORDER BY e.email;

-- name: RevenueGiftPageOn :one
SELECT coalesce((SELECT gift_page_on FROM earn_settings WHERE workspace_id = @workspace_id), false)::boolean;

-- name: RevenueActivity :many
-- The latest paid-member event of each kind the overview lists. A paid
-- start is "subscribed" or a completed checkout without a trial (trials
-- have their own event), whichever came last. With it: what the line says
-- about the subscription (its discount, when access ends, who canceled it
-- at once) and the retention offer the member saw on the cancel page.
SELECT DISTINCT ON (grp) grp::text AS grp, e.id, e.kind, e.occurred_at, e.amount_cents, e.currency, e.data, e.contact_id,
       c.email, c.first_name, c.last_name, coalesce(t.name, '')::text AS tier, coalesce(s.period, '')::text AS period,
       coalesce(s.source_kind, '')::text AS source_kind, coalesce(s.source_label, '')::text AS source_label,
       coalesce(s.amount_cents, 0)::bigint AS sub_amount_cents, s.discount, s.current_period_end, s.cancel_at, s.ended_at,
       coalesce(s.access_paused, false)::boolean AS access_paused,
       (CASE WHEN s.canceled_at IS NOT NULL AND s.ended_at IS NOT NULL AND s.ended_at - s.canceled_at < interval '1 hour'
             THEN coalesce(u.name, '') ELSE '' END)::text AS canceled_at_once_by,
       coalesce((SELECT o.name FROM earn_cancel_visits v JOIN earn_offers o ON o.id = v.offer_id
                 WHERE v.contact_id = e.contact_id AND v.mode = 'live' AND v.visited_at <= e.occurred_at + interval '1 hour'
                 ORDER BY v.visited_at DESC LIMIT 1), '')::text AS saw_offer
FROM paid_events e
CROSS JOIN LATERAL (SELECT CASE WHEN e.kind IN ('subscribed', 'checkout_completed') THEN 'subscribed' ELSE e.kind END AS grp) g
JOIN contacts c ON c.id = e.contact_id
LEFT JOIN paid_subscriptions s ON s.id = e.subscription_id
LEFT JOIN earn_tiers t ON t.id = s.tier_id
LEFT JOIN users u ON u.id = s.canceled_by
WHERE e.workspace_id = @workspace_id AND e.mode = 'live' AND c.erased_at IS NULL
  AND (e.kind = ANY(@kinds::text[]) OR (e.kind = 'checkout_completed' AND s.trial_end IS NULL))
ORDER BY grp, e.occurred_at DESC;

-- name: RevenueTiers :many
-- The workspace's tiers by access rank (a higher tier includes the lower).
SELECT id, name, access_rank, visibility FROM earn_tiers WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY access_rank, name;

-- name: RevenueTierPrices :many
SELECT tier_id, period, amount_cents, currency FROM earn_prices WHERE workspace_id = @workspace_id AND status = 'current';

-- name: RevenueSponsorLinks :many
-- The sponsor links of a campaign's booking: the click token never goes on them.
SELECT coalesce(creative->>'link', '')::text AS link FROM sponsor_bookings WHERE campaign_id = @campaign_id AND status IN ('booked','sent');

-- name: RevenueCampaignInfo :one
SELECT c.id, c.name, c.status, c.started_at, c.scheduled_at, c.completed_at, c.mjml,
       coalesce((SELECT sum(cc.delivered) FROM campaign_counters cc WHERE cc.campaign_id = c.id), 0)::bigint AS delivered
FROM campaigns c WHERE c.id = @id AND c.workspace_id = @workspace_id;

-- name: RevenueCampaignUpgrades :many
-- Paid members credited to a campaign: what they pay first and whether
-- they're still trialing.
SELECT s.id, s.status, s.period, s.amount_cents, s.trial_end, s.started_at, coalesce(t.name, '')::text AS tier, coalesce(s.discount->>'label', '')::text AS offer,
       coalesce((s.discount->>'amountCents')::bigint, s.amount_cents)::bigint AS first_cents,
       earn_arr_cents(s.kind, s.mode, s.status, s.ended_at, s.period, s.amount_cents, s.discount)::bigint AS arr,
       (CASE WHEN s.period = 'year' THEN coalesce((s.discount->>'amountCents')::bigint, s.amount_cents) ELSE coalesce((s.discount->>'amountCents')::bigint, s.amount_cents) * 12 END)::bigint AS arr_if_paid
FROM paid_subscriptions s LEFT JOIN earn_tiers t ON t.id = s.tier_id
WHERE s.workspace_id = @workspace_id AND s.mode = 'live' AND s.kind = 'stripe' AND s.source_kind = 'campaign' AND s.source_campaign_id = @campaign_id
  AND s.status <> 'incomplete';

-- name: RevenueCampaignCheckouts :one
SELECT count(*)::bigint FROM earn_checkouts WHERE workspace_id = @workspace_id AND mode = 'live' AND attribution->>'campaign_id' = @campaign_id::text;

-- name: RevenueCampaignConversions :many
SELECT matched, name, value_cents, currency, key_name, count(*)::bigint AS conversions FROM conversions
WHERE workspace_id = @workspace_id AND campaign_id = @campaign_id AND NOT test
GROUP BY 1, 2, 3, 4, 5;

-- name: RevenueCampaignLinkUrls :many
SELECT url FROM campaign_links WHERE campaign_id = @campaign_id;

-- name: RevenueNextSponsor :many
-- The next sponsored issue of the workspace.
SELECT b.id, b.issue_number, b.creative_status, b.creative_due, coalesce(sp.name, '')::text AS sponsor
FROM sponsor_bookings b LEFT JOIN sponsors sp ON sp.id = b.sponsor_id
WHERE b.workspace_id = @workspace_id AND b.status = 'booked' AND b.issue_on >= @today::date
ORDER BY b.issue_on, b.issue_number LIMIT 1;

-- name: RevenueConnections :many
-- The workspace's payment connections (any mode), newest first.
SELECT id, provider, mode, status, account_name, last_event_at, connected_at FROM payment_connections
WHERE workspace_id = @workspace_id AND status <> 'disconnected' ORDER BY created_at DESC;

-- name: RevenueActiveSubscriptions :one
SELECT count(*)::bigint FROM paid_subscriptions WHERE workspace_id = @workspace_id AND mode = @mode AND kind = 'stripe' AND status IN ('active','past_due','trialing');

-- name: RevenueConversionCounts :one
-- The conversions log's totals: credited or not, in the workspace currency.
SELECT count(*)::bigint AS conversions, coalesce(sum(value_cents), 0)::bigint AS value_cents,
       count(*) FILTER (WHERE campaign_id IS NOT NULL OR automation_id IS NOT NULL)::bigint AS credited,
       coalesce(sum(value_cents) FILTER (WHERE campaign_id IS NOT NULL OR automation_id IS NOT NULL), 0)::bigint AS credited_cents,
       coalesce(min(received_at), now())::timestamptz AS first_at
FROM conversions WHERE workspace_id = @workspace_id AND NOT test AND lower(currency) = lower(@currency::text);

-- name: RevenueUsesEarn :one
-- Whether a campaign report shows a Revenue section: the workspace takes
-- payments or records conversions.
SELECT (EXISTS (SELECT 1 FROM payment_connections pc WHERE pc.workspace_id = @workspace_id::uuid AND pc.status = 'connected')
     OR EXISTS (SELECT 1 FROM conversions cv WHERE cv.workspace_id = @workspace_id::uuid AND NOT cv.test))::boolean AS uses;
