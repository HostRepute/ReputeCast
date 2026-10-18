-- Digital products (internal/products, design expansion phase 7, M29):
-- products and their files, checkouts, orders with their download links,
-- downloads and activity, and lead magnets' links.

-- name: ProductsList :many
SELECT sqlc.embed(p), coalesce(pb.name, '')::text AS published_by_name, coalesce(ub.name, cb.name, '')::text AS edited_by_name
FROM products p
LEFT JOIN users pb ON pb.id = p.published_by
LEFT JOIN users ub ON ub.id = p.updated_by
LEFT JOIN users cb ON cb.id = p.created_by
WHERE p.workspace_id = @workspace_id AND p.deleted_at IS NULL
ORDER BY coalesce(p.first_published_at, p.created_at), p.created_at;

-- name: ProductGet :one
SELECT * FROM products WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL;

-- name: ProductLock :one
SELECT * FROM products WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL FOR UPDATE;

-- name: ProductAny :one
-- A product, deleted or not (an order's product).
SELECT * FROM products WHERE id = @id;

-- name: ProductByLiveSlug :one
SELECT * FROM products WHERE workspace_id = @workspace_id AND live_slug = @slug AND deleted_at IS NULL;

-- name: ProductRedirectGet :one
SELECT r.product_id FROM product_redirects r JOIN products p ON p.id = r.product_id AND p.deleted_at IS NULL
WHERE r.workspace_id = @workspace_id AND r.slug = @slug;

-- name: ProductRedirectInsert :exec
INSERT INTO product_redirects (workspace_id, slug, product_id, created_at) VALUES (@workspace_id, @slug, @product_id, @at)
ON CONFLICT (workspace_id, slug) DO UPDATE SET product_id = EXCLUDED.product_id, created_at = EXCLUDED.created_at;

-- name: ProductRedirectDelete :exec
DELETE FROM product_redirects WHERE workspace_id = @workspace_id AND slug = @slug;

-- name: ProductSlugTaken :one
SELECT (EXISTS (
    SELECT 1 FROM products pr WHERE pr.workspace_id = @workspace_id AND pr.id <> @id AND pr.deleted_at IS NULL AND (pr.slug = @slug OR pr.live_slug = @slug)
) OR EXISTS (
    SELECT 1 FROM product_redirects rd WHERE rd.workspace_id = @workspace_id AND rd.slug = @slug AND rd.product_id <> @id
))::boolean AS taken;

-- name: ProductInsert :exec
INSERT INTO products (id, workspace_id, name, slug, short_desc, long_desc, price_kind, price_cents, min_cents, suggested_cents,
                      show_in_shop, offer_text, delivery_template_id, created_by, created_at, updated_by, updated_at)
VALUES (@id, @workspace_id, @name, @slug, @short_desc, @long_desc, @price_kind, @price_cents, @min_cents, @suggested_cents,
        @show_in_shop, @offer_text, sqlc.narg(delivery_template_id), sqlc.narg(actor), @at, sqlc.narg(actor), @at);

-- name: ProductSave :exec
-- Saves the draft.
UPDATE products SET name = @name, slug = @slug, short_desc = @short_desc, long_desc = @long_desc, cover_url = @cover_url, cover_alt = @cover_alt,
    price_kind = @price_kind, price_cents = @price_cents, min_cents = @min_cents, suggested_cents = @suggested_cents,
    sales_page_id = sqlc.narg(sales_page_id), show_in_shop = @show_in_shop, offer_on = @offer_on, offer_list_id = sqlc.narg(offer_list_id),
    offer_text = @offer_text, link_days = @link_days, link_downloads = @link_downloads, delivery_template_id = sqlc.narg(delivery_template_id),
    revoke_on_refund = @revoke_on_refund, updated_at = @at, updated_by = sqlc.narg(actor)
WHERE id = @id;

-- name: ProductPublish :exec
-- Makes the draft live.
UPDATE products SET status = 'published', live = @live, live_slug = @live_slug, currency = @currency, stripe = @stripe,
    published_at = @at::timestamptz, published_by = sqlc.narg(actor), first_published_at = coalesce(first_published_at, @at::timestamptz),
    archived_at = NULL, archived_by = NULL, updated_at = @at, updated_by = sqlc.narg(actor)
WHERE id = @id;

-- name: ProductSetStripe :exec
UPDATE products SET stripe = @stripe WHERE id = @id;

-- name: ProductArchive :exec
UPDATE products SET status = 'archived', archived_at = @at::timestamptz, archived_by = sqlc.narg(actor), updated_at = @at, updated_by = sqlc.narg(actor) WHERE id = @id;

-- name: ProductUnarchive :exec
UPDATE products SET status = 'published', archived_at = NULL, archived_by = NULL, updated_at = @at, updated_by = sqlc.narg(actor) WHERE id = @id;

-- name: ProductSoftDelete :exec
UPDATE products SET deleted_at = @at::timestamptz, updated_at = @at, updated_by = sqlc.narg(actor) WHERE id = @id;

-- name: ProductsShop :many
-- Published products shown in the shop, oldest first.
SELECT * FROM products WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND status = 'published' AND (live->>'showInShop')::boolean
ORDER BY first_published_at, created_at;

-- name: ProductsByIDs :many
SELECT * FROM products WHERE workspace_id = @workspace_id AND id = ANY(@ids::uuid[]) AND deleted_at IS NULL;

-- name: ProductsPublished :many
SELECT * FROM products WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND status = 'published' ORDER BY first_published_at, created_at;

-- name: ProductsUsingTemplate :many
-- Products whose delivery uses a transactional template ("Used by 2 products").
SELECT id, name FROM products WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND delivery_template_id = @template_id ORDER BY name;

-- name: ProductsUsingPage :many
SELECT id, name, status FROM products WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND sales_page_id = @page_id ORDER BY name;

-- name: ProductDeliveryTemplates :many
-- The transactional templates a product can deliver with, product-delivery first.
SELECT t.id, t.key, t.name, coalesce(t.live_version, 0)::integer AS version, coalesce(c.name, '')::text AS category, coalesce(c.kind, '')::text AS kind
FROM transactional_templates t LEFT JOIN transactional_categories c ON c.id = t.category_id
WHERE t.workspace_id = @workspace_id AND t.archived_at IS NULL AND t.deleted_at IS NULL
ORDER BY t.key = 'product-delivery' DESC, t.key;

-- name: ProductTemplateByKey :one
SELECT id, key, coalesce(live_version, 0)::integer AS version FROM transactional_templates
WHERE workspace_id = @workspace_id AND key = @key AND deleted_at IS NULL;

-- name: ProductTemplateExists :one
SELECT id, key, coalesce(live_version, 0)::integer AS version FROM transactional_templates
WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL;

-- name: ProductListOptions :many
SELECT id, name, double_opt_in FROM lists WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY created_at, name;

-- Files

-- name: DownloadFilesOf :many
SELECT sqlc.embed(f), coalesce(u.name, '')::text AS uploaded_by_name
FROM download_files f LEFT JOIN users u ON u.id = f.uploaded_by
WHERE f.owner_kind = @owner_kind AND f.owner_id = @owner_id AND f.removed_at IS NULL
ORDER BY f.position, f.uploaded_at;

-- name: DownloadFilesOfOwners :many
SELECT * FROM download_files WHERE owner_kind = @owner_kind AND owner_id = ANY(@owner_ids::uuid[]) AND removed_at IS NULL ORDER BY position, uploaded_at;

-- name: DownloadFileGet :one
SELECT * FROM download_files WHERE workspace_id = @workspace_id AND id = @id;

-- name: DownloadFileInsert :exec
INSERT INTO download_files (id, workspace_id, owner_kind, owner_id, name, size_bytes, kind, content_type, pages, sha256, storage_key, position, uploaded_by, uploaded_at)
VALUES (@id, @workspace_id, @owner_kind, @owner_id, @name, @size_bytes, @kind, @content_type, sqlc.narg(pages), @sha256, @storage_key, @position, sqlc.narg(uploaded_by), @at);

-- name: DownloadFileRemove :exec
UPDATE download_files SET removed_at = @at::timestamptz WHERE id = @id;

-- name: DownloadFilesSize :one
SELECT coalesce(sum(size_bytes), 0)::bigint FROM download_files WHERE owner_kind = @owner_kind AND owner_id = @owner_id AND removed_at IS NULL;

-- name: DownloadFilesNextPosition :one
SELECT coalesce(max(position) + 1, 0)::integer FROM download_files WHERE owner_kind = @owner_kind AND owner_id = @owner_id;

-- name: DownloadFileInUse :one
-- Whether any file, removed or not, still keeps these bytes on disk.
SELECT EXISTS (SELECT 1 FROM download_files WHERE storage_key = @storage_key AND id <> @id)::boolean;

-- Checkouts

-- name: ProductCheckoutInsert :exec
INSERT INTO product_checkouts (id, workspace_id, product_id, mode, connection_id, email, contact_id, consent, source, source_page_id, source_url, created_at)
VALUES (@id, @workspace_id, @product_id, @mode, sqlc.narg(connection_id), @email, sqlc.narg(contact_id), @consent, @source, sqlc.narg(source_page_id), @source_url, @at);

-- name: ProductCheckoutSetSession :exec
UPDATE product_checkouts SET session_id = @session_id WHERE id = @id;

-- name: ProductCheckoutFailed :exec
UPDATE product_checkouts SET status = 'failed' WHERE id = @id;

-- name: ProductCheckoutGet :one
SELECT * FROM product_checkouts WHERE id = @id;

-- name: ProductCheckoutBySession :one
SELECT * FROM product_checkouts WHERE session_id = @session_id;

-- name: ProductCheckoutCompleted :exec
UPDATE product_checkouts SET status = 'completed', order_id = @order_id, completed_at = @at::timestamptz WHERE id = @id;

-- name: ProductCheckoutsExpire :exec
UPDATE product_checkouts SET status = 'expired' WHERE status = 'open' AND created_at < @before;

-- Orders

-- name: ProductOrderNextNumber :one
INSERT INTO product_order_numbers (workspace_id, last) VALUES (@workspace_id, 1001)
ON CONFLICT (workspace_id) DO UPDATE SET last = product_order_numbers.last + 1
RETURNING last;

-- name: ProductOrderInsert :exec
INSERT INTO product_orders (id, workspace_id, number, product_id, product_name, mode, connection_id, checkout_id, buyer_email, buyer_name, contact_id, buyer,
    amount_cents, tax_cents, currency, pay_what_you_want, provider, session_id, payment_intent, charge_id, source, source_page_id, source_url, consent,
    campaign_id, automation_id, attributed_by, checkout_started_at, paid_at, created_at)
VALUES (@id, @workspace_id, @number, @product_id, @product_name, @mode, sqlc.narg(connection_id), sqlc.narg(checkout_id), @buyer_email, @buyer_name,
    sqlc.narg(contact_id), @buyer, @amount_cents, @tax_cents, @currency, @pay_what_you_want, @provider, @session_id, @payment_intent, @charge_id,
    @source, sqlc.narg(source_page_id), @source_url, @consent, sqlc.narg(campaign_id), sqlc.narg(automation_id), @attributed_by,
    sqlc.narg(checkout_started_at), @paid_at, @paid_at);

-- name: ProductOrderGet :one
SELECT * FROM product_orders WHERE workspace_id = @workspace_id AND id = @id;

-- name: ProductOrderLock :one
SELECT * FROM product_orders WHERE workspace_id = @workspace_id AND id = @id FOR UPDATE;

-- name: ProductOrderByID :one
SELECT * FROM product_orders WHERE id = @id;

-- name: ProductOrderBySession :one
SELECT * FROM product_orders WHERE session_id = @session_id AND session_id <> '';

-- name: ProductOrderByIntent :one
SELECT * FROM product_orders WHERE workspace_id = @workspace_id AND payment_intent = @payment_intent AND payment_intent <> '' FOR UPDATE;

-- name: ProductOrderByCharge :one
SELECT * FROM product_orders WHERE workspace_id = @workspace_id AND charge_id = @charge_id AND charge_id <> '' FOR UPDATE;

-- name: ProductOrdersList :many
-- Orders newest first, with the latest link's state. q matches the order
-- number or the buyer's email.
SELECT sqlc.embed(o), coalesce(p.name, o.product_name)::text AS current_product_name
FROM product_orders o
LEFT JOIN products p ON p.id = o.product_id
WHERE o.workspace_id = @workspace_id
  AND (@q::text = '' OR o.number::text = @q OR lower(o.buyer_email) LIKE '%' || lower(@q) || '%')
  AND (sqlc.narg(product_id)::uuid IS NULL OR o.product_id = sqlc.narg(product_id))
  AND (@status::text = '' OR (@status = 'test' AND o.mode = 'test') OR (@status <> 'test' AND o.status = @status AND o.mode = 'live'))
  AND o.paid_at >= @since
ORDER BY o.paid_at DESC, o.number DESC
LIMIT @lim OFFSET @off;

-- name: ProductOrdersCount :one
SELECT count(*)::bigint FROM product_orders o
WHERE o.workspace_id = @workspace_id
  AND (@q::text = '' OR o.number::text = @q OR lower(o.buyer_email) LIKE '%' || lower(@q) || '%')
  AND (sqlc.narg(product_id)::uuid IS NULL OR o.product_id = sqlc.narg(product_id))
  AND (@status::text = '' OR (@status = 'test' AND o.mode = 'test') OR (@status <> 'test' AND o.status = @status AND o.mode = 'live'))
  AND o.paid_at >= @since;

-- name: ProductOrdersExport :many
SELECT sqlc.embed(o), coalesce(p.name, o.product_name)::text AS current_product_name
FROM product_orders o
LEFT JOIN products p ON p.id = o.product_id
WHERE o.workspace_id = @workspace_id
  AND (@q::text = '' OR o.number::text = @q OR lower(o.buyer_email) LIKE '%' || lower(@q) || '%')
  AND (sqlc.narg(product_id)::uuid IS NULL OR o.product_id = sqlc.narg(product_id))
  AND (@status::text = '' OR (@status = 'test' AND o.mode = 'test') OR (@status <> 'test' AND o.status = @status AND o.mode = 'live'))
  AND o.paid_at >= @since
ORDER BY o.paid_at DESC, o.number DESC
LIMIT 100000;

-- name: ProductOrdersOfContact :many
SELECT sqlc.embed(o), coalesce(p.name, o.product_name)::text AS current_product_name, coalesce(p.cover_url, '')::text AS cover_url, coalesce(p.cover_alt, '')::text AS cover_alt
FROM product_orders o
LEFT JOIN products p ON p.id = o.product_id
WHERE o.workspace_id = @workspace_id AND (o.contact_id = @contact_id OR lower(o.buyer_email) = lower(@email))
ORDER BY o.paid_at DESC;

-- name: ProductOrderSetContact :exec
UPDATE product_orders SET contact_id = @contact_id, buyer = @buyer WHERE id = @id;

-- name: ProductOrderSetCharge :exec
UPDATE product_orders SET charge_id = @charge_id WHERE id = @id AND charge_id = '';

-- name: ProductOrderRefund :exec
UPDATE product_orders SET refunded_cents = @refunded_cents, refunded_at = @at::timestamptz,
    status = CASE WHEN dispute_status = 'open' THEN 'disputed' WHEN @refunded_cents::bigint >= amount_cents THEN 'refunded' ELSE 'partially_refunded' END
WHERE id = @id;

-- name: ProductOrderDispute :exec
UPDATE product_orders SET dispute_id = @dispute_id, dispute_status = @dispute_status, dispute_reason = @dispute_reason,
    dispute_fee_cents = @dispute_fee_cents, disputed_at = coalesce(disputed_at, @at::timestamptz), dispute_due_by = sqlc.narg(dispute_due_by),
    status = CASE WHEN @dispute_status = 'open' THEN 'disputed'
                  WHEN refunded_cents >= amount_cents AND refunded_cents > 0 THEN 'refunded'
                  WHEN refunded_cents > 0 THEN 'partially_refunded'
                  WHEN @dispute_status = 'lost' THEN 'disputed'
                  ELSE 'paid' END
WHERE id = @id;

-- name: ProductOrderRevoke :exec
UPDATE product_orders SET access_revoked_at = @at::timestamptz, access_revoked_by = sqlc.narg(actor), access_revoke_reason = @reason WHERE id = @id;

-- name: ProductOrderRestore :exec
UPDATE product_orders SET access_revoked_at = NULL, access_revoked_by = NULL, access_revoke_reason = '' WHERE id = @id;

-- name: ProductOrderNames :one
SELECT coalesce((SELECT name FROM users WHERE id = sqlc.narg(revoked_by)::uuid), '')::text AS revoked_by_name;

-- Numbers

-- name: ProductOrderStats :many
-- Per product, in one mode, between two times: orders, gross, refunds.
SELECT product_id, count(*)::bigint AS orders, coalesce(sum(amount_cents), 0)::bigint AS gross,
       count(*) FILTER (WHERE refunded_cents > 0)::bigint AS refunds, coalesce(sum(refunded_cents), 0)::bigint AS refunded,
       count(*) FILTER (WHERE dispute_status = 'open')::bigint AS disputes
FROM product_orders
WHERE workspace_id = @workspace_id AND mode = @mode AND paid_at >= @since AND paid_at < @until
GROUP BY product_id;

-- name: ProductOrderSources :many
-- Where sales came from, per product and source.
SELECT product_id, source, count(*)::bigint AS orders, coalesce(sum(amount_cents), 0)::bigint AS gross
FROM product_orders
WHERE workspace_id = @workspace_id AND mode = @mode AND paid_at >= @since AND paid_at < @until
GROUP BY product_id, source;

-- name: ProductOrderBuyers :many
-- Buyers and contacts, per product, and the list offers ticked.
SELECT product_id, buyer, count(*)::bigint AS orders, count(*) FILTER (WHERE (consent->>'ticked')::boolean)::bigint AS ticked
FROM product_orders
WHERE workspace_id = @workspace_id AND mode = @mode AND paid_at >= @since AND paid_at < @until
GROUP BY product_id, buyer;

-- name: ProductOrdersByCampaign :one
-- Sales credited to a campaign (the report's Revenue rows).
SELECT count(*)::bigint AS orders, coalesce(sum(amount_cents - refunded_cents), 0)::bigint AS revenue,
       count(*) FILTER (WHERE attributed_by = 'token')::bigint AS by_token, count(*) FILTER (WHERE attributed_by = 'email')::bigint AS by_email,
       coalesce(max(currency), '')::text AS currency
FROM product_orders
WHERE workspace_id = @workspace_id AND campaign_id = @campaign_id AND mode = 'live';

-- Links

-- name: ProductOrderLinkInsert :exec
INSERT INTO product_order_links (id, order_id, reason, created_by, created_at, expires_at, downloads_allowed)
VALUES (@id, @order_id, @reason, sqlc.narg(created_by), @at, @expires_at, @downloads_allowed);

-- name: ProductOrderLinkGet :one
SELECT * FROM product_order_links WHERE id = @id;

-- name: ProductOrderLinkLock :one
SELECT * FROM product_order_links WHERE id = @id FOR UPDATE;

-- name: ProductOrderLinks :many
SELECT sqlc.embed(l), coalesce(u.name, '')::text AS created_by_name
FROM product_order_links l LEFT JOIN users u ON u.id = l.created_by
WHERE l.order_id = @order_id
ORDER BY l.created_at DESC;

-- name: ProductOrderLatestLinks :many
-- Each order's current link.
SELECT DISTINCT ON (l.order_id) sqlc.embed(l), coalesce(u.name, '')::text AS created_by_name
FROM product_order_links l LEFT JOIN users u ON u.id = l.created_by
WHERE l.order_id = ANY(@order_ids::uuid[])
ORDER BY l.order_id, l.created_at DESC;

-- name: ProductOrderLinksSupersede :exec
UPDATE product_order_links SET superseded_at = @at::timestamptz WHERE order_id = @order_id AND superseded_at IS NULL AND id <> @keep;

-- name: ProductOrderLinkDelivery :exec
UPDATE product_order_links SET message_id = sqlc.narg(message_id), delivery = @delivery, delivery_error = @delivery_error WHERE id = @id;

-- name: ProductOrderLinkUse :exec
UPDATE product_order_links SET downloads_used = downloads_used + 1 WHERE id = @id;

-- name: ProductOrderDownloadInsert :exec
INSERT INTO product_order_downloads (id, order_id, link_id, file_id, ip_hash, at) VALUES (@id, @order_id, @link_id, @file_id, @ip_hash, @at);

-- name: ProductOrderDownloads :one
SELECT count(*)::bigint AS downloads, count(DISTINCT ip_hash)::bigint AS addresses, max(at) AS last_at, min(at) AS first_at
FROM product_order_downloads WHERE order_id = @order_id;

-- name: ProductDeliveryMessages :many
-- The delivery emails of orders' links (Transactional › Messages).
SELECT m.id, m.status, m.queued_at, m.sent_at, m.delivered_at, m.error, coalesce(p.name, '')::text AS provider,
       coalesce(t.template_key, '')::text AS template_key, coalesce(t.template_version, 0)::integer AS template_version
FROM messages m
LEFT JOIN providers p ON p.id = m.provider_id
LEFT JOIN transactional_messages t ON t.message_id = m.id
WHERE m.workspace_id = @workspace_id AND m.id = ANY(@ids::uuid[]);

-- name: ProductLastClick :one
-- The email a buyer clicked last within the attribution window.
SELECT m.campaign_id, m.automation_id, c.clicked_at
FROM tracking_link_clicks c JOIN messages m ON m.id = c.message_id
WHERE m.workspace_id = @workspace_id AND m.contact_id = @contact_id AND m.stream IN ('broadcast','automation')
  AND c.clicked_at >= @since AND c.clicked_at <= @until
ORDER BY c.clicked_at DESC
LIMIT 1;

-- name: ProductBuyerContact :one
SELECT id, email, first_name, last_name, created_at, status FROM contacts WHERE workspace_id = @workspace_id AND id = @id;

-- name: ProductBuyerByEmail :one
SELECT id, email, first_name, last_name, created_at, status FROM contacts WHERE workspace_id = @workspace_id AND email = lower(@email);

-- name: ProductBuyerMembership :one
SELECT status, subscribed_at FROM list_memberships WHERE list_id = @list_id AND contact_id = @contact_id;

-- Activity

-- name: ProductOrderEventInsert :exec
INSERT INTO product_order_events (id, order_id, kind, at, actor_id, detail) VALUES (@id, @order_id, @kind, @at, sqlc.narg(actor_id), @detail);

-- name: ProductOrderEvents :many
SELECT sqlc.embed(e), coalesce(u.name, '')::text AS actor_name
FROM product_order_events e LEFT JOIN users u ON u.id = e.actor_id
WHERE e.order_id = @order_id
ORDER BY e.at, e.id;

-- Lead magnets

-- name: FormDownloadSet :exec
UPDATE forms SET download_file_id = sqlc.narg(file_id), download_days = @days WHERE id = @id;

-- name: FormDownloadGet :one
SELECT f.download_file_id, f.download_days FROM forms f WHERE f.id = @id;

-- name: LeadMagnetLinkInsert :exec
INSERT INTO lead_magnet_links (id, workspace_id, form_id, file_id, contact_id, created_at, expires_at)
VALUES (@id, @workspace_id, @form_id, @file_id, sqlc.narg(contact_id), @at, @expires_at);

-- name: LeadMagnetLinkGet :one
SELECT * FROM lead_magnet_links WHERE id = @id;

-- name: LeadMagnetLinkUse :exec
UPDATE lead_magnet_links SET downloads = downloads + 1 WHERE id = @id;

-- name: LeadMagnetLinkFor :one
-- The link a contact got for a form's download (one per signup).
SELECT * FROM lead_magnet_links WHERE form_id = @form_id AND contact_id = @contact_id ORDER BY created_at DESC LIMIT 1;

-- name: LeadMagnetStats :one
SELECT count(*)::bigint AS links, coalesce(sum(CASE WHEN downloads > 0 THEN 1 ELSE 0 END), 0)::bigint AS downloaded
FROM lead_magnet_links WHERE form_id = @form_id;
