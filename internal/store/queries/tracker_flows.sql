-- Store flows from the site tracker (M62, G-29 and G-30): who enters
-- "Viewed a product and didn't buy" and "Added to cart and didn't check
-- out" (internal/tracker's tracker.tick), and the 7-day re-entry rule the
-- stock flows share (internal/commerce's commerce.product_changed).

-- name: TrackerFlowsListening :one
-- Whether any live flow starts on a view or an add-to-cart (the minute's
-- job does nothing otherwise).
SELECT EXISTS (SELECT 1 FROM automations
               WHERE status = 'live' AND deleted_at IS NULL
                 AND (triggers @> '[{"kind": "viewed_product"}]'::jsonb OR triggers @> '[{"kind": "added_to_cart"}]'::jsonb));

-- name: TrackerBrowseFlows :many
-- Live flows started by a view or an add-to-cart: one row per such
-- trigger, whichever of the flow's triggers it is, with its connected store.
SELECT a.id, a.workspace_id, ac.t::jsonb AS trigger, s.id AS store_id
FROM automations a
CROSS JOIN LATERAL jsonb_array_elements(a.triggers) AS ac (t)
JOIN stores s ON s.id::text = ac.t -> 'browse' ->> 'storeId' AND s.workspace_id = a.workspace_id AND s.status = 'connected'
WHERE a.status = 'live' AND a.deleted_at IS NULL
  AND (a.triggers @> '[{"kind": "viewed_product"}]'::jsonb OR a.triggers @> '[{"kind": "added_to_cart"}]'::jsonb)
  AND ac.t ->> 'kind' IN ('viewed_product', 'added_to_cart')
ORDER BY a.id;

-- name: TrackerBrowseCandidates :many
-- A page of contacts (by id, after @after) whose view of one of the store's
-- products (or add-to-cart, with @carts) last happened in [@window_start,
-- @window_end): products in @collections when set, viewed (carted)
-- @min_views times or more. For each: the products they viewed (carted) in
-- [@products_since, @window_end), newest first, 6 at most, and whether
-- they enter: active and not erased, nothing on the store's products since
-- @window_end (they're still browsing otherwise), not taken by the flow
-- since @reentry_since, and no order or checkout in the store since their
-- last view.
WITH slice AS (
    SELECT v.contact_id, max(CASE WHEN @carts::boolean THEN v.last_carted_at ELSE v.last_viewed_at END)::timestamptz AS last_at
    FROM tracker_product_views v
    JOIN store_products p ON p.id = v.product_id
    WHERE v.workspace_id = @workspace_id AND v.contact_id > @after::uuid
      AND ((NOT @carts::boolean AND v.last_viewed_at >= @window_start::timestamptz AND v.last_viewed_at < @window_end::timestamptz AND v.views >= @min_views::int)
        OR (@carts::boolean AND v.last_carted_at >= @window_start::timestamptz AND v.last_carted_at < @window_end::timestamptz AND v.carts >= @min_views::int))
      AND p.store_id = @store_id AND p.removed_at IS NULL AND p.status = 'active'
      AND (cardinality(@collections::text[]) = 0 OR p.collections && @collections::text[])
    GROUP BY v.contact_id
    ORDER BY v.contact_id
    LIMIT @lim
)
SELECT s.contact_id,
    coalesce((SELECT array_agg(x.product_id ORDER BY x.at DESC, x.product_id)
              FROM (SELECT v.product_id, CASE WHEN @carts::boolean THEN v.last_carted_at ELSE v.last_viewed_at END AS at
                    FROM tracker_product_views v
                    JOIN store_products p ON p.id = v.product_id
                    WHERE v.contact_id = s.contact_id AND p.store_id = @store_id AND p.removed_at IS NULL AND p.status = 'active'
                      AND (cardinality(@collections::text[]) = 0 OR p.collections && @collections::text[])
                      AND (CASE WHEN @carts::boolean THEN v.carts ELSE v.views END) >= @min_views::int
                      AND (CASE WHEN @carts::boolean THEN v.last_carted_at ELSE v.last_viewed_at END) >= @products_since::timestamptz
                      AND (CASE WHEN @carts::boolean THEN v.last_carted_at ELSE v.last_viewed_at END) < @window_end::timestamptz
                    ORDER BY 2 DESC, 1
                    LIMIT 6) x), '{}')::uuid[] AS products,
    (c.status = 'active' AND c.erased_at IS NULL
     AND NOT EXISTS (SELECT 1 FROM tracker_product_views v
                     JOIN store_products p ON p.id = v.product_id
                     WHERE v.contact_id = s.contact_id AND p.store_id = @store_id
                       AND (v.last_viewed_at >= @window_end::timestamptz OR v.last_carted_at >= @window_end::timestamptz))
     AND NOT EXISTS (SELECT 1 FROM tracker_trigger_entries e
                     WHERE e.automation_id = @automation_id AND e.contact_id = s.contact_id AND e.kind = @kind::text AND e.entered_at > @reentry_since::timestamptz)
     AND NOT EXISTS (SELECT 1 FROM store_orders o
                     WHERE o.store_id = @store_id AND o.contact_id = s.contact_id AND o.placed_at >= s.last_at AND NOT o.test AND NOT o.cancelled)
     AND NOT EXISTS (SELECT 1 FROM store_checkouts ch
                     WHERE ch.store_id = @store_id AND ch.contact_id = s.contact_id AND ch.updated_at_store >= s.last_at))::boolean AS eligible
FROM slice s
JOIN contacts c ON c.id = s.contact_id
ORDER BY s.contact_id;

-- name: TrackerFlowsProducts :many
-- Products an event names, for its {{ browse.products }}.
SELECT id, title, url, image_url, price_cents, currency, available FROM store_products WHERE id = ANY(@ids::uuid[]);

-- name: TrackerFlowsEnter :many
-- Takes contacts into a store flow: those it didn't take since
-- @reentry_since (the 7-day rule) are entered now and returned; the others
-- are left as they are.
INSERT INTO tracker_trigger_entries (automation_id, contact_id, kind, entered_at)
SELECT @automation_id::uuid, c, @kind::text, @at::timestamptz FROM unnest(@contacts::uuid[]) AS c
ON CONFLICT (automation_id, contact_id, kind) DO UPDATE SET entered_at = EXCLUDED.entered_at
WHERE tracker_trigger_entries.entered_at <= @reentry_since::timestamptz
RETURNING contact_id;

-- name: TrackerFlowsEmit :exec
-- One automation event per contact, in one statement.
INSERT INTO automation_events (workspace_id, kind, contact_id, ref, payload, occurred_at)
SELECT @workspace_id::uuid, @kind::text, e.contact_id, (@refs::text[])[e.n], (@payloads::jsonb[])[e.n], @at::timestamptz
FROM unnest(@contacts::uuid[]) WITH ORDINALITY AS e (contact_id, n);
