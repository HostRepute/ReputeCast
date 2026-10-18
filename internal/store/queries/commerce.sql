-- Shopify and WooCommerce stores (internal/commerce): connections, what
-- they sync, webhooks and raw events, the activity log, attribution and
-- the abandoned checkouts the automations watch.

-- ── Stores ──────────────────────────────────────────────────────────────

-- name: CommerceStores :many
SELECT * FROM stores WHERE workspace_id = @workspace_id ORDER BY created_at;

-- name: CommerceStore :one
SELECT * FROM stores WHERE id = @id AND workspace_id = @workspace_id;

-- name: CommerceStoreByID :one
SELECT * FROM stores WHERE id = @id;

-- name: CommerceStoreForUpdate :one
SELECT * FROM stores WHERE id = @id AND workspace_id = @workspace_id FOR UPDATE;

-- name: CommerceStoreByAddress :one
SELECT * FROM stores WHERE workspace_id = @workspace_id AND platform = @platform AND address = @address;

-- name: CommerceStoreByToken :one
SELECT * FROM stores WHERE webhook_token = @webhook_token;

-- name: CommerceStoreByState :one
SELECT * FROM stores WHERE oauth_state = @oauth_state;

-- name: CommerceStoresByDomain :many
-- Stores installing or installed on a Shopify domain (the app URL).
SELECT * FROM stores WHERE platform = 'shopify' AND domain = @domain AND client_id <> '' ORDER BY updated_at DESC;

-- name: CommerceStoresActive :many
-- Stores to watch: connected (or importing), not the demo's.
SELECT * FROM stores WHERE status IN ('connected', 'importing') AND NOT demo ORDER BY id;

-- name: CommerceStoresConnected :many
-- Connected stores of a workspace (links to tag, triggers, blocks).
SELECT * FROM stores WHERE workspace_id = @workspace_id AND status IN ('connected', 'importing') ORDER BY created_at;

-- name: CommerceStoreInsert :one
INSERT INTO stores (id, workspace_id, platform, status, step, name, address, domain, public_domain, currency, api_version, app_name,
    webhook_token, demo, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @platform, 'connecting', @step, @name, @address, @domain, @public_domain, @currency, @api_version, @app_name,
    @webhook_token, @demo, sqlc.narg(created_by), @at, @at)
RETURNING *;

-- name: CommerceStoreRestart :one
-- Connecting a store again (after a disconnect): its data is kept.
UPDATE stores SET step = @step, name = CASE WHEN @name::text <> '' THEN @name ELSE name END, domain = @domain, public_domain = @public_domain,
    currency = CASE WHEN @currency::text <> '' THEN @currency ELSE currency END, connect_error = '', oauth_state = NULL, oauth_state_expires_at = NULL,
    status = CASE WHEN status = 'disconnected' THEN 'disconnected' ELSE 'connecting' END, updated_at = @at
WHERE id = @id
RETURNING *;

-- name: CommerceStoreSetStep :exec
UPDATE stores SET step = @step, connect_error = @connect_error, updated_at = @at WHERE id = @id;

-- name: CommerceStoreSetKeys :exec
UPDATE stores SET client_id = @client_id, credentials_sealed = @credentials_sealed, key_hint = @key_hint, step = @step,
    connect_error = '', updated_at = @at
WHERE id = @id;

-- name: CommerceStoreSetCredentials :exec
UPDATE stores SET credentials_sealed = @credentials_sealed, updated_at = @at WHERE id = @id;

-- name: CommerceStoreSetState :exec
UPDATE stores SET oauth_state = @oauth_state, oauth_state_expires_at = @expires_at, updated_at = @at WHERE id = @id;

-- name: CommerceStoreInstalled :exec
UPDATE stores SET credentials_sealed = @credentials_sealed, scopes = @scopes, installed_at = @at, oauth_state = NULL, oauth_state_expires_at = NULL,
    step = @step, connect_error = '', name = CASE WHEN @name::text <> '' THEN @name ELSE name END,
    currency = CASE WHEN @currency::text <> '' THEN @currency ELSE currency END,
    public_domain = CASE WHEN @public_domain::text <> '' THEN @public_domain ELSE public_domain END,
    revoked_at = NULL, revoked_reason = '', updated_at = @at
WHERE id = @id;

-- name: CommerceStoreSetWebhookSecret :exec
UPDATE stores SET webhook_secret_sealed = @webhook_secret_sealed, updated_at = @at WHERE id = @id;

-- name: CommerceStoreSetPlugin :exec
UPDATE stores SET plugin_found = @plugin_found, plugin_checked_at = @at, updated_at = @at WHERE id = @id;

-- name: CommerceStoreStartSync :one
-- Step 5 / finishing: what syncs, then the first import starts.
UPDATE stores SET sync_products = @sync_products, sync_customers = @sync_customers, sync_orders = @sync_orders, sync_checkouts = @sync_checkouts,
    import_months = @import_months, list_id = sqlc.narg(list_id), write_back = @write_back, plugin_skipped = @plugin_skipped,
    status = 'importing', step = 'done', import_progress = @import_progress, import_started_at = @at, import_started_by = sqlc.narg(by),
    import_finished_at = NULL, connected_by = coalesce(connected_by, sqlc.narg(by)), connected_at = coalesce(connected_at, @at),
    disconnected_at = NULL, disconnected_by = NULL, revoked_at = NULL, revoked_reason = '', notified_at = NULL, updated_at = @at
WHERE id = @id
RETURNING *;

-- name: CommerceStoreReconnected :one
-- Reconnecting a store whose import was done before: no new first import;
-- orders placed in the meantime are fetched by the reconciliation.
UPDATE stores SET status = 'connected', step = 'done', disconnected_at = NULL, disconnected_by = NULL, revoked_at = NULL, revoked_reason = '',
    notified_at = NULL, updated_at = @at
WHERE id = @id
RETURNING *;

-- name: CommerceStoreProgress :exec
UPDATE stores SET import_progress = @import_progress, updated_at = @at WHERE id = @id;

-- name: CommerceStoreImportDone :exec
UPDATE stores SET status = 'connected', import_finished_at = @at, import_progress = @import_progress, reconciled_at = coalesce(reconciled_at, @at),
    updated_at = @at
WHERE id = @id AND status = 'importing';

-- name: CommerceStoreSetSettings :one
UPDATE stores SET list_id = sqlc.narg(list_id), write_back = @write_back, tag_links = @tag_links, attribution_days = @attribution_days,
    attribution_set_by = CASE WHEN attribution_days <> @attribution_days THEN sqlc.narg(by) ELSE attribution_set_by END,
    attribution_set_at = CASE WHEN attribution_days <> @attribution_days THEN @at ELSE attribution_set_at END,
    sync_products = @sync_products, sync_customers = @sync_customers, sync_orders = @sync_orders, sync_checkouts = @sync_checkouts,
    updated_at = @at
WHERE id = @id
RETURNING *;

-- name: CommerceStoreTouchEvent :exec
UPDATE stores SET last_event_at = greatest(coalesce(last_event_at, @at), @at) WHERE id = @id;

-- name: CommerceStoreHealthChecked :exec
UPDATE stores SET health_checked_at = @at WHERE id = @id;

-- name: CommerceStoreRateLimited :exec
UPDATE stores SET rate_limited_until = @until, updated_at = @at WHERE id = @id;

-- name: CommerceStoreReconciled :exec
UPDATE stores SET reconciled_at = @at, reconcile_missed = @missed, reconcile_products = @products, updated_at = @at WHERE id = @id;

-- name: CommerceStoreRevoked :one
-- The store refused the credentials: Disconnected, with Reconnect.
UPDATE stores SET status = 'disconnected', revoked_at = coalesce(revoked_at, @at), revoked_reason = @reason, disconnected_at = coalesce(disconnected_at, @at),
    updated_at = @at
WHERE id = @id AND status <> 'disconnected'
RETURNING *;

-- name: CommerceStoreDisconnected :exec
UPDATE stores SET status = 'disconnected', disconnected_at = @at, disconnected_by = sqlc.narg(by), credentials_sealed = NULL, oauth_state = NULL,
    updated_at = @at
WHERE id = @id;

-- name: CommerceStoreNotified :exec
UPDATE stores SET notified_at = @at WHERE id = @id;

-- name: CommerceStoreDelete :exec
DELETE FROM stores WHERE id = @id;

-- ── Webhooks ────────────────────────────────────────────────────────────

-- name: CommerceWebhooks :many
SELECT * FROM store_webhooks WHERE store_id = @store_id ORDER BY position, topic;

-- name: CommerceWebhookUpsert :exec
INSERT INTO store_webhooks (store_id, topic, external_id, status, registered_at, registered_by, missing_since, error, position)
VALUES (@store_id, @topic, @external_id, @status, sqlc.narg(registered_at), sqlc.narg(registered_by), NULL, @error, @position)
ON CONFLICT (store_id, topic) DO UPDATE SET external_id = EXCLUDED.external_id, status = EXCLUDED.status,
    registered_at = coalesce(EXCLUDED.registered_at, store_webhooks.registered_at), registered_by = coalesce(EXCLUDED.registered_by, store_webhooks.registered_by),
    missing_since = CASE WHEN EXCLUDED.status = 'registered' THEN NULL ELSE store_webhooks.missing_since END,
    error = EXCLUDED.error, position = EXCLUDED.position;

-- name: CommerceWebhookMissing :exec
UPDATE store_webhooks SET status = 'missing', missing_since = coalesce(missing_since, @at), external_id = ''
WHERE store_id = @store_id AND topic = @topic;

-- name: CommerceWebhookPolled :exec
UPDATE store_webhooks SET last_polled_at = @at WHERE store_id = @store_id AND topic = @topic;

-- name: CommerceWebhookEvent :exec
UPDATE store_webhooks SET last_event_at = greatest(coalesce(last_event_at, @at), @at) WHERE store_id = @store_id AND topic = @topic;

-- name: CommerceWebhooksClear :exec
UPDATE store_webhooks SET status = 'pending', external_id = '', missing_since = NULL WHERE store_id = @store_id;

-- ── Raw events ──────────────────────────────────────────────────────────

-- name: CommerceEventInsert :one
INSERT INTO store_events (id, store_id, workspace_id, topic, delivery_id, triggered_at, received_at, headers, body, signature_ok, status)
VALUES (@id, @store_id, @workspace_id, @topic, @delivery_id, sqlc.narg(triggered_at), @received_at, @headers, @body, @signature_ok, @status)
ON CONFLICT (store_id, delivery_id) WHERE delivery_id <> '' DO NOTHING
RETURNING id;

-- name: CommerceEvent :one
SELECT * FROM store_events WHERE id = @id;

-- name: CommerceEventDone :exec
UPDATE store_events SET status = @status, outcome = @outcome, applied_at = @at WHERE id = @id;

-- name: CommerceEventsPending :many
-- Events not applied two minutes after they arrived (the recovery sweep).
SELECT id FROM store_events WHERE status = 'pending' AND received_at < @before ORDER BY received_at LIMIT 200;

-- name: CommerceEventsPurge :execrows
DELETE FROM store_events WHERE received_at < @before;

-- name: CommerceEventsRejected :one
SELECT count(*) FROM store_events WHERE store_id = @store_id AND status = 'rejected' AND received_at >= @since;

-- ── Activity ────────────────────────────────────────────────────────────

-- name: CommerceActivityInsert :exec
INSERT INTO store_activity (id, store_id, workspace_id, at, kind, tone, data, order_id, checkout_id, contact_id)
VALUES (@id, @store_id, @workspace_id, @at, @kind, @tone, @data, sqlc.narg(order_id), sqlc.narg(checkout_id), sqlc.narg(contact_id));

-- name: CommerceActivity :many
SELECT * FROM store_activity
WHERE store_id = @store_id
  AND (sqlc.narg(before)::timestamptz IS NULL OR (at, id) < (sqlc.narg(before)::timestamptz, sqlc.narg(before_id)::uuid))
  AND (@kinds::text[] = '{}' OR kind = ANY(@kinds::text[]))
ORDER BY at DESC, id DESC
LIMIT @lim;

-- name: CommerceActivityPurge :execrows
DELETE FROM store_activity WHERE at < @before;

-- ── Collections and products ────────────────────────────────────────────

-- name: CommerceCollectionUpsert :exec
INSERT INTO store_collections (store_id, external_id, title, handle, products, position)
VALUES (@store_id, @external_id, @title, @handle, @products, @position)
ON CONFLICT (store_id, external_id) DO UPDATE SET title = EXCLUDED.title, handle = EXCLUDED.handle, products = EXCLUDED.products,
    position = EXCLUDED.position;

-- name: CommerceCollections :many
-- Collections with their live product counts (products not removed).
SELECT c.external_id, c.title, c.handle, c.position,
    coalesce((SELECT count(*) FROM store_products p WHERE p.store_id = c.store_id AND p.removed_at IS NULL AND p.status = 'active'
        AND c.external_id = ANY(p.collections)), 0)::bigint AS products
FROM store_collections c
WHERE c.store_id = @store_id
ORDER BY c.position, c.title;

-- name: CommerceProductUpsert :one
-- An older version of a product than the one stored is ignored (events
-- that arrive out of order).
INSERT INTO store_products (id, store_id, workspace_id, external_id, title, handle, url, image_url, image_alt, price_cents, compare_at_cents, currency,
    available, inventory, status, collections, created_at_store, updated_at_store, removed_at, synced_at)
VALUES (@id, @store_id, @workspace_id, @external_id, @title, @handle, @url, @image_url, @image_alt, @price_cents, @compare_at_cents, @currency,
    @available, sqlc.narg(inventory), @status, @collections, sqlc.narg(created_at_store), sqlc.narg(updated_at_store), NULL, @at)
ON CONFLICT (store_id, external_id) DO UPDATE SET title = EXCLUDED.title, handle = EXCLUDED.handle,
    url = CASE WHEN EXCLUDED.url <> '' THEN EXCLUDED.url ELSE store_products.url END,
    image_url = EXCLUDED.image_url, image_alt = EXCLUDED.image_alt, price_cents = EXCLUDED.price_cents, compare_at_cents = EXCLUDED.compare_at_cents,
    currency = EXCLUDED.currency, available = EXCLUDED.available, inventory = EXCLUDED.inventory, status = EXCLUDED.status,
    collections = CASE WHEN @keep_collections::boolean THEN store_products.collections ELSE EXCLUDED.collections END,
    created_at_store = coalesce(store_products.created_at_store, EXCLUDED.created_at_store), updated_at_store = EXCLUDED.updated_at_store,
    removed_at = NULL, synced_at = EXCLUDED.synced_at
WHERE store_products.updated_at_store IS NULL OR EXCLUDED.updated_at_store IS NULL OR EXCLUDED.updated_at_store >= store_products.updated_at_store
RETURNING id;

-- name: CommerceProductRemoved :execrows
UPDATE store_products SET removed_at = coalesce(removed_at, @at) WHERE store_id = @store_id AND external_id = @external_id;

-- name: CommerceProductsRemoveMissing :many
-- After a full product read: products the store no longer has.
UPDATE store_products SET removed_at = @at
WHERE store_id = @store_id AND removed_at IS NULL AND NOT (external_id = ANY(@seen::text[]))
RETURNING external_id, title;

-- name: CommerceProducts :many
-- The catalog for the picker: products not removed, newest first,
-- optionally one collection's, optionally matching a search.
SELECT * FROM store_products
WHERE store_id = @store_id AND removed_at IS NULL AND status = 'active'
  AND (@collection::text = '' OR @collection::text = ANY(collections))
  AND (@search::text = '' OR title ILIKE '%' || @search::text || '%')
ORDER BY created_at_store DESC NULLS LAST, title
LIMIT @lim OFFSET @off;

-- name: CommerceProductCount :one
SELECT count(*) FROM store_products WHERE store_id = @store_id AND removed_at IS NULL AND status = 'active';

-- name: CommerceProductsByExternal :many
-- Products by id, removed ones included (blocks flag those).
SELECT * FROM store_products WHERE store_id = @store_id AND external_id = ANY(@ids::text[]);

-- name: CommerceProductsNewest :many
-- The newest products of a collection (the product grid's "Newest from a
-- collection").
SELECT * FROM store_products
WHERE store_id = @store_id AND removed_at IS NULL AND status = 'active' AND @collection::text = ANY(collections)
ORDER BY created_at_store DESC NULLS LAST, title
LIMIT @lim;

-- ── Customers ───────────────────────────────────────────────────────────

-- name: CommerceCustomer :one
SELECT * FROM store_customers WHERE store_id = @store_id AND external_id = @external_id;

-- name: CommerceCustomerUpsert :one
INSERT INTO store_customers (id, store_id, workspace_id, external_id, contact_id, email, first_name, last_name, country, marketing,
    opt_in_level, consent_at, joined_list_at, updated_at_store, created_at)
VALUES (@id, @store_id, @workspace_id, @external_id, sqlc.narg(contact_id), @email, @first_name, @last_name, @country, @marketing,
    @opt_in_level, sqlc.narg(consent_at), sqlc.narg(joined_list_at), sqlc.narg(updated_at_store), @at)
ON CONFLICT (store_id, external_id) DO UPDATE SET contact_id = coalesce(EXCLUDED.contact_id, store_customers.contact_id),
    email = EXCLUDED.email, first_name = EXCLUDED.first_name, last_name = EXCLUDED.last_name, country = EXCLUDED.country,
    marketing = EXCLUDED.marketing, opt_in_level = EXCLUDED.opt_in_level, consent_at = EXCLUDED.consent_at,
    joined_list_at = coalesce(store_customers.joined_list_at, EXCLUDED.joined_list_at),
    updated_at_store = greatest(store_customers.updated_at_store, EXCLUDED.updated_at_store)
RETURNING *;

-- name: CommerceCustomerByContact :many
SELECT * FROM store_customers WHERE contact_id = @contact_id;

-- name: CommerceCustomerToldStore :exec
UPDATE store_customers SET told_store_at = @at, marketing = 'unsubscribed' WHERE id = @id;

-- name: CommerceCustomerStats :one
-- The store page's consent numbers.
SELECT count(*) FILTER (WHERE marketing = 'subscribed')::bigint AS subscribed,
       count(*) FILTER (WHERE joined_list_at IS NOT NULL)::bigint AS joined
FROM store_customers WHERE store_id = @store_id;

-- ── Contacts the sync needs ─────────────────────────────────────────────

-- name: CommerceContactByEmail :one
SELECT id, status, email, first_name, last_name, erased_at FROM contacts WHERE workspace_id = @workspace_id AND email = lower(@email::text);

-- name: CommerceContactInsert :one
INSERT INTO contacts (id, workspace_id, email, first_name, last_name, status, consent, source, country, mailbox_provider, consent_evidence, created_at, updated_at)
VALUES (@id, @workspace_id, @email, @first_name, @last_name, 'active', @consent, @source, @country, @mailbox_provider, @consent_evidence, @at, @at)
ON CONFLICT (workspace_id, email) DO NOTHING
RETURNING id;

-- name: CommerceMembership :one
SELECT status, unsubscribed_at FROM list_memberships WHERE list_id = @list_id AND contact_id = @contact_id;

-- name: CommerceJoinList :execrows
-- Joins the store's list. A contact who left the list stays out, unless
-- the store's consent is newer than their leaving (a new opt-in).
INSERT INTO list_memberships (list_id, contact_id, status, source, subscribed_at, created_at)
VALUES (@list_id, @contact_id, 'subscribed', @source, @subscribed_at, @at)
ON CONFLICT (list_id, contact_id) DO UPDATE SET status = 'subscribed', source = EXCLUDED.source, subscribed_at = EXCLUDED.subscribed_at,
    unsubscribed_at = NULL
WHERE list_memberships.status = 'unsubscribed' AND list_memberships.unsubscribed_at < @consent_at::timestamptz;

-- name: CommerceLeaveList :execrows
UPDATE list_memberships SET status = 'unsubscribed', unsubscribed_at = @at
WHERE list_id = @list_id AND contact_id = @contact_id AND status <> 'unsubscribed';

-- name: CommerceConsentInsert :exec
INSERT INTO consent_events (id, contact_id, list_id, action, source, detail, note, consent_text, occurred_at)
VALUES (@id, @contact_id, sqlc.narg(list_id), @action, @source, @detail, @note, @consent_text, @at);

-- name: CommerceListCreate :exec
INSERT INTO lists (id, workspace_id, name, description, double_opt_in, requires_approval, origin, created_at)
VALUES (@id, @workspace_id, @name, @description, false, false, 'store', @at);

-- name: CommerceListByName :one
SELECT id FROM lists WHERE workspace_id = @workspace_id AND lower(name) = lower(@name::text) AND deleted_at IS NULL;

-- name: CommerceLists :many
-- The lists the customers can join, with their subscribed count.
SELECT l.id, l.name, l.origin,
    coalesce((SELECT count(*) FROM list_memberships m WHERE m.list_id = l.id AND m.status = 'subscribed'), 0)::bigint AS subscribed
FROM lists l
WHERE l.workspace_id = @workspace_id AND l.deleted_at IS NULL
ORDER BY l.created_at;

-- name: CommerceListName :one
SELECT name FROM lists WHERE id = @id;

-- name: CommerceUnsubscribesSince :many
-- Contacts who left the store's list, or every list, since a time: the
-- unsubscribes to tell the store about (write-back). Leaving that the
-- store itself caused (source store:…) is left out.
SELECT DISTINCT ON (sc.id) sc.id, sc.external_id, sc.contact_id, ce.occurred_at
FROM consent_events ce
JOIN store_customers sc ON sc.contact_id = ce.contact_id AND sc.store_id = @store_id
WHERE ce.action = 'unsubscribed' AND ce.occurred_at > @since AND ce.occurred_at <= @until
  AND (ce.list_id = @list_id OR ce.list_id IS NULL OR ce.source IN ('one_click', 'unsubscribe_page'))
  AND ce.source NOT LIKE 'store:%'
  AND sc.marketing = 'subscribed'
ORDER BY sc.id, ce.occurred_at DESC;

-- ── Orders ──────────────────────────────────────────────────────────────

-- name: CommerceOrder :one
SELECT * FROM store_orders WHERE store_id = @store_id AND external_id = @external_id;

-- name: CommerceOrderByID :one
SELECT * FROM store_orders WHERE id = @id;

-- name: CommerceOrderUpsert :one
-- Inserts an order, or updates it when the version is newer. inserted
-- says which (the first time an order is seen starts its triggers).
INSERT INTO store_orders (id, store_id, workspace_id, external_id, name, contact_id, customer_external_id, accepted_marketing, placed_at, currency,
    subtotal_cents, total_cents, refunded_cents, discount_codes, items, landing_page, utm_source, utm_medium, utm_campaign, utm_content, utm_term,
    checkout_external_id, cancelled, test, first_order, source, updated_at_store, created_at, updated_at)
VALUES (@id, @store_id, @workspace_id, @external_id, @name, sqlc.narg(contact_id), @customer_external_id, @accepted_marketing, @placed_at, @currency,
    @subtotal_cents, @total_cents, @refunded_cents, @discount_codes, @items, @landing_page, @utm_source, @utm_medium, @utm_campaign, @utm_content, @utm_term,
    @checkout_external_id, @cancelled, @test, @first_order, @source, sqlc.narg(updated_at_store), @at, @at)
ON CONFLICT (store_id, external_id) DO UPDATE SET contact_id = coalesce(EXCLUDED.contact_id, store_orders.contact_id),
    accepted_marketing = EXCLUDED.accepted_marketing OR store_orders.accepted_marketing, currency = EXCLUDED.currency,
    subtotal_cents = EXCLUDED.subtotal_cents, total_cents = EXCLUDED.total_cents, refunded_cents = greatest(EXCLUDED.refunded_cents, store_orders.refunded_cents),
    discount_codes = EXCLUDED.discount_codes, items = EXCLUDED.items,
    landing_page = CASE WHEN EXCLUDED.landing_page <> '' THEN EXCLUDED.landing_page ELSE store_orders.landing_page END,
    utm_source = CASE WHEN EXCLUDED.landing_page <> '' THEN EXCLUDED.utm_source ELSE store_orders.utm_source END,
    utm_medium = CASE WHEN EXCLUDED.landing_page <> '' THEN EXCLUDED.utm_medium ELSE store_orders.utm_medium END,
    utm_campaign = CASE WHEN EXCLUDED.landing_page <> '' THEN EXCLUDED.utm_campaign ELSE store_orders.utm_campaign END,
    utm_content = CASE WHEN EXCLUDED.landing_page <> '' THEN EXCLUDED.utm_content ELSE store_orders.utm_content END,
    utm_term = CASE WHEN EXCLUDED.landing_page <> '' THEN EXCLUDED.utm_term ELSE store_orders.utm_term END,
    checkout_external_id = CASE WHEN EXCLUDED.checkout_external_id <> '' THEN EXCLUDED.checkout_external_id ELSE store_orders.checkout_external_id END,
    cancelled = EXCLUDED.cancelled, test = EXCLUDED.test,
    updated_at_store = EXCLUDED.updated_at_store, updated_at = EXCLUDED.updated_at
WHERE store_orders.updated_at_store IS NULL OR EXCLUDED.updated_at_store IS NULL OR EXCLUDED.updated_at_store >= store_orders.updated_at_store
RETURNING id, (xmax = 0)::boolean AS inserted, contact_id, matched, placed_at;

-- name: CommerceOrderRefund :exec
UPDATE store_orders SET refunded_cents = refunded_cents + @amount, updated_at = @at WHERE store_id = @store_id AND external_id = @external_id;

-- name: CommerceOrderDelete :exec
DELETE FROM store_orders WHERE store_id = @store_id AND external_id = @external_id;

-- name: CommerceOrderAttribute :exec
UPDATE store_orders SET matched = @matched, message_id = sqlc.narg(message_id), campaign_id = sqlc.narg(campaign_id),
    automation_id = sqlc.narg(automation_id), automation_node = @automation_node, clicked_at = sqlc.narg(clicked_at)
WHERE id = @id;

-- name: CommerceContactHasOrder :one
-- Whether the contact ordered in the store before this order.
SELECT EXISTS (SELECT 1 FROM store_orders WHERE store_id = @store_id AND contact_id = @contact_id AND placed_at < @placed_at AND NOT test AND NOT cancelled);

-- name: CommerceOrdersToMatch :many
-- Imported orders not matched yet (the first import's last phase).
SELECT id, contact_id, placed_at FROM store_orders
WHERE store_id = @store_id AND matched = 'none' AND source = 'import' AND id > @after AND contact_id IS NOT NULL
ORDER BY id LIMIT @lim;

-- name: CommerceOrderCount :one
SELECT count(*) FROM store_orders WHERE store_id = @store_id;

-- name: CommerceContactOrders :many
-- A contact's store orders, newest first, with the email they were
-- credited to.
SELECT o.*, s.name AS store_name, s.platform, s.list_id AS store_list_id,
    coalesce(c.name, '')::text AS campaign_name, coalesce(a.name, '')::text AS automation_name
FROM store_orders o
JOIN stores s ON s.id = o.store_id
LEFT JOIN campaigns c ON c.id = o.campaign_id
LEFT JOIN automations a ON a.id = o.automation_id
WHERE o.contact_id = @contact_id AND o.workspace_id = @workspace_id
ORDER BY o.placed_at DESC
LIMIT @lim;

-- name: CommerceContactStoreCustomers :many
SELECT sc.*, s.name AS store_name, s.platform, s.list_id AS store_list_id
FROM store_customers sc
JOIN stores s ON s.id = sc.store_id
WHERE sc.contact_id = @contact_id AND sc.workspace_id = @workspace_id
ORDER BY sc.created_at;

-- ── Checkouts ───────────────────────────────────────────────────────────

-- name: CommerceCheckoutUpsert :one
INSERT INTO store_checkouts (id, store_id, workspace_id, external_id, contact_id, customer_external_id, has_email, accepted_marketing, items,
    subtotal_cents, total_cents, currency, recovery_url, created_at_store, updated_at_store, completed_at, source, created_at, updated_at)
VALUES (@id, @store_id, @workspace_id, @external_id, sqlc.narg(contact_id), @customer_external_id, @has_email, @accepted_marketing, @items,
    @subtotal_cents, @total_cents, @currency, @recovery_url, @created_at_store, @updated_at_store, sqlc.narg(completed_at), @source, @at, @at)
ON CONFLICT (store_id, external_id) DO UPDATE SET contact_id = coalesce(EXCLUDED.contact_id, store_checkouts.contact_id),
    customer_external_id = CASE WHEN EXCLUDED.customer_external_id <> '' THEN EXCLUDED.customer_external_id ELSE store_checkouts.customer_external_id END,
    has_email = EXCLUDED.has_email OR store_checkouts.has_email, accepted_marketing = EXCLUDED.accepted_marketing,
    items = EXCLUDED.items, subtotal_cents = EXCLUDED.subtotal_cents, total_cents = EXCLUDED.total_cents, currency = EXCLUDED.currency,
    recovery_url = CASE WHEN EXCLUDED.recovery_url <> '' THEN EXCLUDED.recovery_url ELSE store_checkouts.recovery_url END,
    updated_at_store = EXCLUDED.updated_at_store, completed_at = coalesce(store_checkouts.completed_at, EXCLUDED.completed_at), updated_at = EXCLUDED.updated_at
WHERE EXCLUDED.updated_at_store >= store_checkouts.updated_at_store
RETURNING *;

-- name: CommerceCheckoutComplete :many
-- An order completed its checkout: by its token, or (WooCommerce, Shopify
-- without the token) the contact's open checkouts from before it.
UPDATE store_checkouts SET completed_at = coalesce(completed_at, @at), order_id = @order_id, updated_at = @at
WHERE store_id = @store_id AND completed_at IS NULL
  AND (external_id = @external_id OR (sqlc.narg(contact_id)::uuid IS NOT NULL AND contact_id = sqlc.narg(contact_id)::uuid AND created_at_store <= @placed_at))
RETURNING id;

-- name: CommerceCheckoutByID :one
SELECT * FROM store_checkouts WHERE id = @id;

-- name: CommerceCheckoutsAbandoned :many
-- Checkouts of a store with no order after the delay, created since a
-- time, that an automation hasn't looked at yet.
SELECT ch.* FROM store_checkouts ch
WHERE ch.store_id = @store_id AND ch.completed_at IS NULL AND ch.updated_at_store <= @before AND ch.created_at_store >= @since
  AND NOT EXISTS (SELECT 1 FROM store_checkout_entries e WHERE e.automation_id = @automation_id AND e.checkout_id = ch.id)
ORDER BY ch.updated_at_store
LIMIT 500;

-- name: CommerceCheckoutEntry :exec
INSERT INTO store_checkout_entries (automation_id, checkout_id, outcome, at) VALUES (@automation_id, @checkout_id, @outcome, @at)
ON CONFLICT DO NOTHING;

-- name: CommerceCheckoutEntries :many
-- An automation's abandoned-checkout numbers since it went live.
SELECT outcome, count(*)::bigint AS n FROM store_checkout_entries WHERE automation_id = @automation_id GROUP BY outcome;

-- name: CommerceCheckoutStats :one
-- The store's abandoned checkouts since a time: all, with an email.
SELECT count(*)::bigint AS abandoned, count(*) FILTER (WHERE has_email)::bigint AS with_email
FROM store_checkouts WHERE store_id = @store_id AND completed_at IS NULL AND created_at_store >= @since AND updated_at_store <= @before;

-- name: CommerceAbandonedCartEmailedSince :one
-- The skip rule: the contact got an email from an abandoned-cart flow
-- (one started by abandoned checkouts) since a time.
SELECT EXISTS (
    SELECT 1 FROM messages m JOIN automations a ON a.id = m.automation_id
    WHERE m.contact_id = @contact_id AND m.queued_at >= @since AND a.workspace_id = @workspace_id
      AND a.trigger ->> 'kind' = 'abandoned_checkout');

-- name: CommerceAbandonedFlows :many
-- Live automations started by abandoned checkouts.
SELECT id, workspace_id, trigger, live_version, updated_at FROM automations
WHERE status = 'live' AND trigger ->> 'kind' = 'abandoned_checkout';

-- name: CommerceFlowLiveSince :one
-- When an automation's live version was published.
SELECT published_at FROM automation_versions WHERE automation_id = @automation_id AND version = @version;

-- name: CommerceStoreFlows :many
-- Live and paused automations started by a store's triggers.
SELECT id, name, status, trigger, pause_reason FROM automations
WHERE workspace_id = @workspace_id AND trigger ->> 'kind' IN ('order_placed', 'first_order', 'abandoned_checkout') AND trigger ->> 'store' = @store::text
ORDER BY created_at;

-- name: CommercePauseFlow :execrows
UPDATE automations SET status = 'paused', paused_at = @at, paused_by = NULL, pause_reason = @reason WHERE id = @id AND status = 'live';

-- name: CommerceEmitEvent :exec
INSERT INTO automation_events (workspace_id, kind, contact_id, ref, payload, occurred_at)
VALUES (@workspace_id, @kind, @contact_id, @ref, @payload, @occurred_at);

-- ── Attribution ─────────────────────────────────────────────────────────

-- name: CommerceClickMessage :one
-- The email a click token names: its contact and campaign or automation step.
SELECT id, workspace_id, contact_id, campaign_id, automation_id, coalesce(automation_node, '')::text AS automation_node, stream
FROM messages WHERE id = @id;

-- name: CommerceLastClick :one
-- The contact's last human click on a campaign or automation email within
-- the window before the order.
SELECT e.message_id, m.campaign_id, m.automation_id, coalesce(m.automation_node, '')::text AS automation_node, e.occurred_at
FROM delivery_events e JOIN messages m ON m.id = e.message_id
WHERE e.contact_id = @contact_id AND e.type = 'click' AND NOT e.machine
  AND e.occurred_at <= @at::timestamptz AND e.occurred_at >= @since::timestamptz
  AND m.stream IN ('broadcast', 'automation')
ORDER BY e.occurred_at DESC
LIMIT 1;

-- name: CommerceClickTime :one
-- When a message's link was clicked last before a time (a click token's
-- click, when the token carries no time).
SELECT max(occurred_at)::timestamptz FROM delivery_events WHERE message_id = @message_id AND type = 'click' AND NOT machine AND occurred_at <= @at;

-- ── Numbers ─────────────────────────────────────────────────────────────

-- name: CommerceStoreTotals :one
-- A store's orders and revenue in a period, and the part credited to email.
SELECT count(*)::bigint AS orders, coalesce(sum(total_cents - refunded_cents), 0)::bigint AS revenue,
    count(*) FILTER (WHERE matched <> 'none')::bigint AS credited, coalesce(sum(total_cents - refunded_cents) FILTER (WHERE matched <> 'none'), 0)::bigint AS credited_revenue,
    count(*) FILTER (WHERE matched = 'click_token')::bigint AS by_token, count(*) FILTER (WHERE matched = 'email')::bigint AS by_email
FROM store_orders
WHERE store_id = @store_id AND placed_at >= @since AND placed_at <= @until AND NOT test AND NOT cancelled;

-- name: CommerceStoreMatchedAll :one
-- How the orders were matched, every order.
SELECT count(*) FILTER (WHERE matched = 'click_token')::bigint AS by_token, count(*) FILTER (WHERE matched = 'email')::bigint AS by_email
FROM store_orders WHERE store_id = @store_id AND NOT test;

-- name: CommerceRevenueByEmail :many
-- A store's credited orders in a period by campaign or automation.
SELECT o.campaign_id, o.automation_id, coalesce(c.name, a.name, '')::text AS name, (o.automation_id IS NOT NULL)::boolean AS automation,
    count(*)::bigint AS orders, coalesce(sum(o.total_cents - o.refunded_cents), 0)::bigint AS revenue
FROM store_orders o
LEFT JOIN campaigns c ON c.id = o.campaign_id
LEFT JOIN automations a ON a.id = o.automation_id
WHERE o.store_id = @store_id AND o.placed_at >= @since AND o.placed_at <= @until AND o.matched <> 'none' AND NOT o.test AND NOT o.cancelled
GROUP BY o.campaign_id, o.automation_id, c.name, a.name
ORDER BY revenue DESC, orders DESC;

-- name: CommerceCampaignRevenue :many
-- A campaign's credited store orders, per store (the report's rows).
SELECT s.id AS store_id, s.name, s.platform, s.attribution_days, count(*)::bigint AS orders,
    coalesce(sum(o.total_cents - o.refunded_cents), 0)::bigint AS revenue, max(o.currency)::text AS currency,
    count(*) FILTER (WHERE o.matched = 'click_token')::bigint AS by_token, count(*) FILTER (WHERE o.matched = 'email')::bigint AS by_email
FROM store_orders o JOIN stores s ON s.id = o.store_id
WHERE o.campaign_id = @campaign_id AND o.workspace_id = @workspace_id AND o.matched <> 'none' AND NOT o.test AND NOT o.cancelled
GROUP BY s.id, s.name, s.platform, s.attribution_days
ORDER BY revenue DESC;

-- name: CommerceAutomationRevenue :many
-- Credited store orders per automation since a time (the automations list).
SELECT automation_id, count(*)::bigint AS orders, coalesce(sum(total_cents - refunded_cents), 0)::bigint AS revenue, max(currency)::text AS currency
FROM store_orders
WHERE workspace_id = @workspace_id AND automation_id = ANY(@ids::uuid[]) AND placed_at >= @since AND matched <> 'none' AND NOT test AND NOT cancelled
GROUP BY automation_id;

-- name: CommerceAutomationGoalRevenue :one
-- One automation's credited orders since a time (the goal node's numbers).
SELECT count(*)::bigint AS orders, coalesce(sum(total_cents - refunded_cents), 0)::bigint AS revenue
FROM store_orders WHERE automation_id = @automation_id AND placed_at >= @since AND matched <> 'none' AND NOT test AND NOT cancelled;

-- name: CommerceWorkspaceRevenue :one
-- Every connected store's orders and credited revenue in a period (the
-- Earn overview's card).
SELECT count(*)::bigint AS orders, coalesce(sum(o.total_cents - o.refunded_cents), 0)::bigint AS revenue,
    count(*) FILTER (WHERE o.matched <> 'none')::bigint AS credited,
    coalesce(sum(o.total_cents - o.refunded_cents) FILTER (WHERE o.matched <> 'none'), 0)::bigint AS credited_revenue,
    coalesce(max(o.currency), '')::text AS currency
FROM store_orders o JOIN stores s ON s.id = o.store_id
WHERE o.workspace_id = @workspace_id AND o.placed_at >= @since AND o.placed_at <= @until AND NOT o.test AND NOT o.cancelled;

-- ── Purges and data deletion ────────────────────────────────────────────

-- name: CommerceDeleteStoreData :exec
-- "Delete store data": products, orders, checkouts and customers go; the
-- contacts the store added stay, with their consent.
DELETE FROM store_products WHERE store_id = @store_id;

-- name: CommerceDeleteStoreOrders :exec
DELETE FROM store_orders WHERE store_id = @store_id;

-- name: CommerceDeleteStoreCheckouts :exec
DELETE FROM store_checkouts WHERE store_id = @store_id;

-- name: CommerceDeleteStoreCustomers :exec
DELETE FROM store_customers WHERE store_id = @store_id;

-- name: CommerceDeleteStoreCollections :exec
DELETE FROM store_collections WHERE store_id = @store_id;

-- name: CommerceMembers :many
-- Active members with their role's permissions (who can help, who to tell).
SELECT u.id AS user_id, u.name, r.key AS role_key, r.permissions
FROM workspace_members m
JOIN roles r ON r.id = m.role_id
JOIN users u ON u.id = m.user_id AND u.status = 'active'
WHERE m.workspace_id = @workspace_id AND m.status = 'active'
ORDER BY CASE r.key WHEN 'admin' THEN 0 WHEN 'owner' THEN 1 ELSE 2 END, u.name;

-- name: CommerceUserName :one
SELECT name FROM users WHERE id = @id;
