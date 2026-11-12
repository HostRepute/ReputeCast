-- Holdout group (M63 G-33): periods, who is held out, the measurement.

-- name: HoldoutOpenPeriod :one
-- The workspace's open period: while it's open its contacts in the buckets
-- under basis_points are held out.
SELECT * FROM holdout_periods WHERE workspace_id = @workspace_id AND ended_at IS NULL;

-- name: HoldoutLatestPeriod :one
SELECT * FROM holdout_periods WHERE workspace_id = @workspace_id ORDER BY started_at DESC, id DESC LIMIT 1;

-- name: HoldoutRecentPeriods :many
-- The newest periods first: the open one and the last closed ones.
SELECT * FROM holdout_periods WHERE workspace_id = @workspace_id ORDER BY started_at DESC, id DESC LIMIT @lim;

-- name: HoldoutGetPeriod :one
SELECT * FROM holdout_periods WHERE workspace_id = @workspace_id AND id = @id;

-- name: HoldoutExists :one
-- Whether the workspace ever had a holdout group (Audience shows the tab).
SELECT EXISTS (SELECT 1 FROM holdout_periods WHERE workspace_id = @workspace_id)::boolean;

-- name: HoldoutLockWorkspace :exec
-- Serializes changes of a workspace's holdout settings.
SELECT 1 FROM workspaces WHERE id = @workspace_id FOR UPDATE;

-- name: HoldoutEndPeriod :exec
UPDATE holdout_periods SET ended_at = @at::timestamptz WHERE workspace_id = @workspace_id AND ended_at IS NULL;

-- name: HoldoutInsertPeriod :exec
INSERT INTO holdout_periods (id, workspace_id, seed, basis_points, campaigns, automations, started_at, started_by, reason)
VALUES (@id, @workspace_id, @seed, @basis_points, @campaigns, @automations, @started_at, sqlc.narg(started_by), @reason);

-- name: HoldoutHeldOutCount :one
-- The active contacts the open period holds out now.
SELECT count(*)::bigint FROM contacts c
WHERE c.workspace_id = @workspace_id AND c.erased_at IS NULL AND c.status = 'active'
  AND rc_holdout_bucket(c.id, @seed::bigint) < @basis_points::int;

-- name: HoldoutContactHeldOut :one
-- Whether the open period holds a contact out of automations (or, with
-- automations false, out of campaigns).
SELECT EXISTS (SELECT 1 FROM holdout_periods p
    WHERE p.workspace_id = @workspace_id AND p.ended_at IS NULL
      AND (CASE WHEN @automations::boolean THEN p.automations ELSE p.campaigns END)
      AND rc_holdout_bucket(@contact_id::uuid, p.seed) < p.basis_points)::boolean;

-- name: HoldoutHeldContacts :many
-- Which of the contacts are in the buckets under basis_points.
SELECT x.id::uuid AS id FROM unnest(@contact_ids::uuid[]) AS x(id)
WHERE rc_holdout_bucket(x.id, @seed::bigint) < @basis_points::int;

-- name: HoldoutRequestMeasure :exec
UPDATE holdout_periods SET measure_requested_at = @at WHERE id = @id;

-- name: HoldoutMeasureDue :many
-- Workspaces with a period the nightly measurement should compute (an
-- open one, or one that ended since it was last computed), with their
-- time zone and when they were last computed (the epoch for never).
SELECT w.id AS workspace_id, w.timezone,
       coalesce((SELECT max(r.computed_at) FROM holdout_results r JOIN holdout_periods pp ON pp.id = r.period_id WHERE pp.workspace_id = w.id),
                'epoch'::timestamptz)::timestamptz AS last_computed
FROM workspaces w
WHERE EXISTS (SELECT 1 FROM holdout_periods p WHERE p.workspace_id = w.id
    AND (p.ended_at IS NULL OR NOT EXISTS (SELECT 1 FROM holdout_results r WHERE r.period_id = p.id AND r.computed_at >= p.ended_at)));

-- name: HoldoutCurrency :one
-- The workspace's main currency (Earn › Settings), USD by default.
SELECT coalesce((SELECT lower(e.currency) FROM earn_settings e WHERE e.workspace_id = @workspace_id AND e.currency <> ''), 'usd')::text;

-- name: HoldoutMeasureGroups :many
-- The measurement of one window, per group (held out or not): the
-- population is the contacts that weren't erased, existed before the
-- window's end and weren't on the suppression list before its start. A
-- converter placed a store order (not a test, not cancelled), recorded a
-- conversion or started a live paid subscription in the window; revenue
-- is orders minus refunds plus conversion values in the main currency, in
-- cents, with the sums of squares the variance needs; a visitor was seen
-- on one of the workspace's tracked sites (counted only when it has one).
WITH pop AS (
    SELECT c.id, (rc_holdout_bucket(c.id, @seed::bigint) < @basis_points::int) AS held
    FROM contacts c
    WHERE c.workspace_id = @workspace_id AND c.erased_at IS NULL AND c.created_at < @until::timestamptz
      AND NOT EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = c.workspace_id AND s.email = c.email AND s.created_at < @since::timestamptz)
), ord AS (
    SELECT o.contact_id, count(*) AS orders,
           coalesce(sum(o.total_cents - o.refunded_cents) FILTER (WHERE lower(o.currency) = @currency::text), 0) AS cents
    FROM store_orders o
    WHERE o.workspace_id = @workspace_id AND o.contact_id IS NOT NULL AND NOT o.test AND NOT o.cancelled
      AND o.placed_at >= @since::timestamptz AND o.placed_at < @until::timestamptz
    GROUP BY o.contact_id
), conv AS (
    SELECT v.contact_id, coalesce(sum(v.value_cents) FILTER (WHERE lower(v.currency) = @currency::text), 0) AS cents
    FROM conversions v
    WHERE v.workspace_id = @workspace_id AND v.contact_id IS NOT NULL AND NOT v.test
      AND v.occurred_at >= @since::timestamptz AND v.occurred_at < @until::timestamptz
    GROUP BY v.contact_id
), paid AS (
    SELECT DISTINCT p.contact_id FROM paid_subscriptions p
    WHERE p.workspace_id = @workspace_id AND p.mode = 'live' AND p.kind = 'stripe' AND p.status <> 'incomplete'
      AND p.started_at >= @since::timestamptz AND p.started_at < @until::timestamptz
), vis AS (
    -- Site visitors (M62's tracker): contacts seen on a tracked site.
    SELECT DISTINCT e.contact_id FROM tracker_events e
    WHERE e.workspace_id = @workspace_id AND e.contact_id IS NOT NULL
      AND e.occurred_at >= @since::timestamptz AND e.occurred_at < @until::timestamptz
), per AS (
    SELECT pop.held,
           (coalesce(ord.cents, 0) + coalesce(conv.cents, 0))::float8 AS revenue,
           coalesce(ord.orders, 0)::float8 AS orders,
           (ord.contact_id IS NOT NULL OR conv.contact_id IS NOT NULL OR paid.contact_id IS NOT NULL) AS converted,
           (vis.contact_id IS NOT NULL) AS visited
    FROM pop
    LEFT JOIN ord ON ord.contact_id = pop.id
    LEFT JOIN conv ON conv.contact_id = pop.id
    LEFT JOIN paid ON paid.contact_id = pop.id
    LEFT JOIN vis ON vis.contact_id = pop.id
)
SELECT per.held::boolean AS held,
       count(*)::bigint AS contacts,
       (count(*) FILTER (WHERE per.converted))::bigint AS converters,
       (count(*) FILTER (WHERE per.visited))::bigint AS visitors,
       coalesce(sum(per.orders), 0)::float8 AS orders,
       coalesce(sum(per.orders * per.orders), 0)::float8 AS orders_sq,
       coalesce(sum(per.revenue), 0)::float8 AS revenue,
       coalesce(sum(per.revenue * per.revenue), 0)::float8 AS revenue_sq
FROM per
GROUP BY per.held;

-- name: HoldoutTracking :one
-- Whether the workspace tracks a site (Website tracking, M62): the
-- measurement then counts site visitors.
SELECT EXISTS (SELECT 1 FROM tracker_sites WHERE workspace_id = @workspace_id AND deleted_at IS NULL)::boolean;

-- name: HoldoutOtherCurrencies :many
-- What the window recorded in other currencies, per group: counted and
-- named apart, never converted.
WITH pop AS (
    SELECT c.id, (rc_holdout_bucket(c.id, @seed::bigint) < @basis_points::int) AS held
    FROM contacts c
    WHERE c.workspace_id = @workspace_id AND c.erased_at IS NULL AND c.created_at < @until::timestamptz
      AND NOT EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = c.workspace_id AND s.email = c.email AND s.created_at < @since::timestamptz)
), amounts AS (
    SELECT o.contact_id, lower(o.currency) AS currency, (o.total_cents - o.refunded_cents) AS cents
    FROM store_orders o
    WHERE o.workspace_id = @workspace_id AND o.contact_id IS NOT NULL AND NOT o.test AND NOT o.cancelled
      AND o.placed_at >= @since::timestamptz AND o.placed_at < @until::timestamptz AND lower(o.currency) <> @currency::text
    UNION ALL
    SELECT v.contact_id, lower(v.currency), v.value_cents
    FROM conversions v
    WHERE v.workspace_id = @workspace_id AND v.contact_id IS NOT NULL AND NOT v.test
      AND v.occurred_at >= @since::timestamptz AND v.occurred_at < @until::timestamptz AND lower(v.currency) <> @currency::text
)
SELECT a.currency::text AS currency,
       coalesce(sum(a.cents) FILTER (WHERE pop.held), 0)::bigint AS holdout_cents,
       coalesce(sum(a.cents) FILTER (WHERE NOT pop.held), 0)::bigint AS marketing_cents
FROM amounts a
JOIN pop ON pop.id = a.contact_id
GROUP BY a.currency
ORDER BY a.currency;

-- name: HoldoutCampaignsIncluded :one
-- Email campaigns started in the window that included the holdout group.
SELECT count(*)::bigint FROM campaigns
WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND started_at >= @since::timestamptz AND started_at < @until::timestamptz
  AND audience @> '{"include_holdout": true}'::jsonb;

-- name: HoldoutSaveResult :exec
INSERT INTO holdout_results (period_id, span, computed_at, data) VALUES (@period_id, @span, @computed_at, @data)
ON CONFLICT (period_id, span) DO UPDATE SET computed_at = EXCLUDED.computed_at, data = EXCLUDED.data;

-- name: HoldoutResult :one
SELECT * FROM holdout_results WHERE period_id = @period_id AND span = @span;
