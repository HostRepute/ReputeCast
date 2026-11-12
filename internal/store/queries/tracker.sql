-- The site tracker (M62 G-28): sites, the collect writer's batches,
-- identification, the contact timeline, the hub card and retention.

-- name: TrackerSites :many
-- A workspace's sites, newest last, with the store each is linked to.
SELECT t.*, coalesce(st.name, '')::text AS store_name, coalesce(st.platform, '')::text AS store_platform,
       coalesce(st.status, '')::text AS store_status, coalesce(u.name, '')::text AS created_by_name
FROM tracker_sites t
LEFT JOIN stores st ON st.id = t.store_id
LEFT JOIN users u ON u.id = t.created_by
WHERE t.workspace_id = @workspace_id AND t.deleted_at IS NULL
ORDER BY t.created_at, t.id;

-- name: TrackerSite :one
SELECT t.*, coalesce(st.name, '')::text AS store_name, coalesce(st.platform, '')::text AS store_platform,
       coalesce(st.status, '')::text AS store_status, coalesce(u.name, '')::text AS created_by_name
FROM tracker_sites t
LEFT JOIN stores st ON st.id = t.store_id
LEFT JOIN users u ON u.id = t.created_by
WHERE t.id = @id AND t.workspace_id = @workspace_id AND t.deleted_at IS NULL;

-- name: TrackerSiteInsert :exec
INSERT INTO tracker_sites (id, workspace_id, name, origin, store_id, wait_consent, identify_secret_sealed, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @name, @origin, sqlc.narg(store_id), @wait_consent, @identify_secret_sealed, sqlc.narg(created_by), @at, @at);

-- name: TrackerSiteUpdate :exec
UPDATE tracker_sites SET name = @name, store_id = sqlc.narg(store_id), wait_consent = @wait_consent, updated_at = @at
WHERE id = @id AND workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: TrackerSiteDelete :execrows
-- A site stops collecting at once; its events go with the retention.
UPDATE tracker_sites SET deleted_at = @at, updated_at = @at, identify_secret_sealed = NULL
WHERE id = @id AND workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: TrackerSiteSetSecret :execrows
UPDATE tracker_sites SET identify_secret_sealed = @identify_secret_sealed, updated_at = @at
WHERE id = @id AND workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: TrackerSiteEventsSince :many
-- Events per site since a moment (the sites table's "events today").
SELECT e.site_id, count(*)::bigint AS events
FROM tracker_events e
WHERE e.site_id = ANY(@site_ids::uuid[]) AND e.occurred_at >= @since
GROUP BY e.site_id;

-- name: TrackerSiteRefusedSince :many
-- Origins refused per site since a moment, most refused first.
SELECT r.site_id, r.origin, r.count, r.last_at
FROM tracker_site_refusals r
WHERE r.site_id = ANY(@site_ids::uuid[]) AND r.last_at >= @since
ORDER BY r.site_id, r.count DESC, r.origin;

-- name: TrackerRecentEvents :many
-- The setup's live check: a site's last 20 events, without addresses.
SELECT e.kind, e.path, coalesce(p.title, '')::text AS product_title, e.name, (e.contact_id IS NOT NULL)::boolean AS known, e.occurred_at
FROM tracker_events e
LEFT JOIN store_products p ON p.id = e.product_id
WHERE e.site_id = @site_id AND e.workspace_id = @workspace_id AND e.occurred_at >= @since
ORDER BY e.occurred_at DESC, e.id DESC
LIMIT 20;

-- name: TrackerCatalogFacts :one
-- The hub card's facts: live sites, their events since a moment, the last
-- event ('epoch' when there's none).
SELECT count(*)::bigint AS sites,
       coalesce((SELECT count(*) FROM tracker_events e
                 WHERE e.site_id IN (SELECT s2.id FROM tracker_sites s2 WHERE s2.workspace_id = @workspace_id AND s2.deleted_at IS NULL)
                   AND e.occurred_at >= @since), 0)::bigint AS events,
       coalesce(max(s.last_event_at), 'epoch')::timestamptz AS last_event_at
FROM tracker_sites s
WHERE s.workspace_id = @workspace_id AND s.deleted_at IS NULL;

-- name: TrackerCollectSite :one
-- The site a collect call names, by the workspace's slug.
SELECT t.id, t.workspace_id, w.slug, t.origin, t.store_id, t.wait_consent, t.identify_secret_sealed
FROM tracker_sites t
JOIN workspaces w ON w.id = t.workspace_id
WHERE t.id = @id AND w.slug = @slug AND w.deleted_at IS NULL AND t.deleted_at IS NULL;

-- name: TrackerSiteOrigins :many
-- A workspace's live site origins (the click decorator's hosts).
SELECT origin FROM tracker_sites WHERE workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: TrackerProductsByExternal :many
-- A linked store's products by the ids its pages name.
SELECT id, external_id FROM store_products
WHERE store_id = @store_id AND external_id = ANY(@external_ids::text[]) AND removed_at IS NULL;

-- name: TrackerVisitorsUpsert :many
-- A batch's visitors, in (workspace, hash) order: new ones are added, the
-- others seen again; each comes back with the contact it is linked to.
INSERT INTO tracker_visitors (id, workspace_id, vid_hash, first_seen, last_seen)
SELECT unnest(@ids::uuid[]), unnest(@workspace_ids::uuid[]), unnest(@hashes::bytea[]), unnest(@seen::timestamptz[]), unnest(@seen::timestamptz[])
ON CONFLICT (workspace_id, vid_hash) DO UPDATE SET last_seen = greatest(tracker_visitors.last_seen, EXCLUDED.last_seen)
RETURNING id, workspace_id, vid_hash, contact_id;

-- name: TrackerContactLive :one
-- A contact of the workspace who hasn't been erased (identification).
SELECT id FROM contacts WHERE id = @id AND workspace_id = @workspace_id AND erased_at IS NULL;

-- name: TrackerContactByEmail :one
SELECT id FROM contacts WHERE workspace_id = @workspace_id AND email = lower(@email::text) AND erased_at IS NULL;

-- name: TrackerVisitorIdentify :exec
UPDATE tracker_visitors SET contact_id = @contact_id, identified_by = @identified_by, identified_at = @at
WHERE id = @id;

-- name: TrackerBackfillEvents :execrows
-- The visitor's anonymous events of the last 30 days join the contact
-- (events already linked to someone stay theirs).
UPDATE tracker_events SET contact_id = @contact_id
WHERE visitor_id = @visitor_id AND contact_id IS NULL AND occurred_at >= @since;

-- name: TrackerMergeVisitorProducts :exec
-- A newly linked visitor's product interest joins the contact's.
INSERT INTO tracker_product_views AS v (workspace_id, contact_id, product_id, views, carts, last_viewed_at, last_carted_at)
SELECT @workspace_id, @contact_id, vp.product_id, vp.views, vp.carts, vp.last_viewed_at, vp.last_carted_at
FROM tracker_visitor_products vp
WHERE vp.visitor_id = @visitor_id
ORDER BY vp.product_id
ON CONFLICT (contact_id, product_id) DO UPDATE SET views = v.views + EXCLUDED.views, carts = v.carts + EXCLUDED.carts,
    last_viewed_at = greatest(v.last_viewed_at, EXCLUDED.last_viewed_at), last_carted_at = greatest(v.last_carted_at, EXCLUDED.last_carted_at);

-- name: TrackerDropVisitorProducts :exec
DELETE FROM tracker_visitor_products WHERE visitor_id = @visitor_id;

-- name: TrackerProductViewsAdd :exec
-- A batch's product views and add-to-carts of known contacts, in
-- (contact, product) order. 'epoch' stands for "none" in the time arrays.
INSERT INTO tracker_product_views AS v (workspace_id, contact_id, product_id, views, carts, last_viewed_at, last_carted_at)
SELECT unnest(@workspace_ids::uuid[]), unnest(@contact_ids::uuid[]), unnest(@product_ids::uuid[]), unnest(@views::int[]), unnest(@carts::int[]),
       nullif(unnest(@viewed::timestamptz[]), 'epoch'), nullif(unnest(@carted::timestamptz[]), 'epoch')
ON CONFLICT (contact_id, product_id) DO UPDATE SET views = v.views + EXCLUDED.views, carts = v.carts + EXCLUDED.carts,
    last_viewed_at = greatest(v.last_viewed_at, EXCLUDED.last_viewed_at), last_carted_at = greatest(v.last_carted_at, EXCLUDED.last_carted_at);

-- name: TrackerVisitorProductsAdd :exec
-- The same for visitors not linked to a contact yet.
INSERT INTO tracker_visitor_products AS v (visitor_id, product_id, views, carts, last_viewed_at, last_carted_at)
SELECT unnest(@visitor_ids::uuid[]), unnest(@product_ids::uuid[]), unnest(@views::int[]), unnest(@carts::int[]),
       nullif(unnest(@viewed::timestamptz[]), 'epoch'), nullif(unnest(@carted::timestamptz[]), 'epoch')
ON CONFLICT (visitor_id, product_id) DO UPDATE SET views = v.views + EXCLUDED.views, carts = v.carts + EXCLUDED.carts,
    last_viewed_at = greatest(v.last_viewed_at, EXCLUDED.last_viewed_at), last_carted_at = greatest(v.last_carted_at, EXCLUDED.last_carted_at);

-- name: TrackerStockRequestsAdd :exec
-- "Email me when it's back" from known visitors: asking again starts over.
INSERT INTO store_stock_requests (workspace_id, contact_id, product_id, created_at)
SELECT unnest(@workspace_ids::uuid[]), unnest(@contact_ids::uuid[]), unnest(@product_ids::uuid[]), unnest(@at::timestamptz[])
ON CONFLICT (contact_id, product_id) DO UPDATE SET created_at = EXCLUDED.created_at, notified_at = NULL;

-- name: TrackerSitesTouch :exec
-- The last event of each site of a batch.
UPDATE tracker_sites s SET last_event_at = x.at
FROM (SELECT unnest(@ids::uuid[]) AS id, unnest(@at::timestamptz[]) AS at) x
WHERE s.id = x.id AND (s.last_event_at IS NULL OR s.last_event_at < x.at);

-- name: TrackerRefusalsAdd :exec
-- Calls refused for coming from another origin, counted in memory.
INSERT INTO tracker_site_refusals AS r (site_id, origin, count, first_at, last_at)
SELECT x.site_id, x.origin, x.n, x.first_at, x.last_at
FROM (SELECT unnest(@site_ids::uuid[]) AS site_id, unnest(@origins::text[]) AS origin, unnest(@counts::bigint[]) AS n,
             unnest(@first_at::timestamptz[]) AS first_at, unnest(@last_at::timestamptz[]) AS last_at) x
WHERE EXISTS (SELECT 1 FROM tracker_sites s WHERE s.id = x.site_id)
ON CONFLICT (site_id, origin) DO UPDATE SET count = r.count + EXCLUDED.count, last_at = greatest(r.last_at, EXCLUDED.last_at);

-- name: TrackerContactDays :many
-- The contact timeline: one line per day (in the workspace's time zone)
-- and site, newest first, before the cursor.
SELECT d.id, d.at, d.origin, d.products, d.carts, d.pages,
       coalesce((SELECT array_agg(p.title ORDER BY p.title) FROM store_products p WHERE p.id = ANY(d.product_ids[1:3])), '{}')::text[] AS titles
FROM (
    SELECT (array_agg(e.id ORDER BY e.occurred_at DESC, e.id DESC))[1]::uuid AS id, max(e.occurred_at)::timestamptz AS at, s.origin,
           (count(DISTINCT e.product_id) FILTER (WHERE e.kind = 2) + count(*) FILTER (WHERE e.kind = 2 AND e.product_id IS NULL))::bigint AS products,
           count(*) FILTER (WHERE e.kind = 3)::bigint AS carts,
           count(*) FILTER (WHERE e.kind = 1)::bigint AS pages,
           coalesce(array_agg(DISTINCT e.product_id) FILTER (WHERE e.kind = 2 AND e.product_id IS NOT NULL), '{}')::uuid[] AS product_ids
    FROM tracker_events e
    JOIN tracker_sites s ON s.id = e.site_id
    WHERE e.contact_id = @contact_id AND e.workspace_id = @workspace_id
    GROUP BY s.origin, e.site_id, (e.occurred_at AT TIME ZONE @tz::text)::date
) d
WHERE sqlc.narg(before)::timestamptz IS NULL OR (d.at, d.id) < (sqlc.narg(before)::timestamptz, @before_id::uuid)
ORDER BY d.at DESC, d.id DESC
LIMIT @lim;

-- name: TrackerContactDayCount :one
SELECT count(*)::bigint FROM (
    SELECT 1 FROM tracker_events e
    WHERE e.contact_id = @contact_id AND e.workspace_id = @workspace_id
    GROUP BY e.site_id, (e.occurred_at AT TIME ZONE @tz::text)::date
) d;

-- name: TrackerRetentionSites :many
-- Every site's events cutoff: 30 days where the workspace keeps event
-- detail 30 days, else 90.
SELECT s.id, (CASE WHEN w.event_detail_days = 30 THEN 30 ELSE 90 END)::int AS days
FROM tracker_sites s
JOIN workspaces w ON w.id = s.workspace_id
ORDER BY s.id;

-- name: TrackerDeleteSiteEvents :execrows
DELETE FROM tracker_events WHERE site_id = @site_id AND occurred_at < @before;

-- name: TrackerDeleteDefaultEvents :execrows
-- The default partition should stay empty; what it holds past the 90 days
-- goes.
DELETE FROM tracker_events_default WHERE occurred_at < @before;

-- name: TrackerStaleVisitors :many
-- Visitors never linked to a contact and unseen since a moment.
SELECT id FROM tracker_visitors WHERE contact_id IS NULL AND last_seen < @before ORDER BY last_seen LIMIT @lim;

-- name: TrackerDeleteVisitorEvents :execrows
DELETE FROM tracker_events WHERE visitor_id = ANY(@ids::uuid[]);

-- name: TrackerDeleteVisitors :execrows
DELETE FROM tracker_visitors WHERE id = ANY(@ids::uuid[]) AND contact_id IS NULL;

-- name: TrackerPurgeProductViews :execrows
-- A contact's interest in a product is kept 180 days after the last view
-- or add-to-cart.
DELETE FROM tracker_product_views WHERE greatest(last_viewed_at, last_carted_at) < @before;

-- name: TrackerPurgeRefusals :execrows
DELETE FROM tracker_site_refusals WHERE last_at < @before;
