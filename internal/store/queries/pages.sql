-- Landing and sales pages (internal/pages, design expansion phase 7, M29):
-- the pages, their saved and published versions, earlier addresses, what
-- each page brought in, and the shop's switch.

-- name: PagesList :many
-- Every page with who edited it last.
SELECT sqlc.embed(p), coalesce(u.name, '')::text AS saved_by_name, coalesce(pb.name, '')::text AS published_by_name
FROM pages p
LEFT JOIN users u ON u.id = p.saved_by
LEFT JOIN users pb ON pb.id = p.published_by
WHERE p.workspace_id = @workspace_id AND p.deleted_at IS NULL
ORDER BY coalesce(p.first_published_at, p.created_at), p.created_at;

-- name: PageGet :one
SELECT * FROM pages WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL;

-- name: PageLock :one
SELECT * FROM pages WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL FOR UPDATE;

-- name: PageNames :one
-- The names of who saved and who published a page.
SELECT coalesce((SELECT name FROM users WHERE id = sqlc.narg(saved_by)::uuid), '')::text AS saved_by_name,
       coalesce((SELECT name FROM users WHERE id = sqlc.narg(published_by)::uuid), '')::text AS published_by_name;

-- name: PageByLiveSlug :one
-- The published (or unpublished) page answering at an address.
SELECT * FROM pages WHERE workspace_id = @workspace_id AND live_slug = @slug AND deleted_at IS NULL;

-- name: PageEverLiveBySlug :one
-- A page deleted after being published still answers 410 at its address.
SELECT * FROM pages WHERE workspace_id = @workspace_id AND live_slug = @slug AND deleted_at IS NOT NULL
ORDER BY deleted_at DESC LIMIT 1;

-- name: PageSlugTaken :one
-- Whether another page holds the address (as its draft's or its live one),
-- or an earlier address of another page redirects from it.
SELECT (EXISTS (
    SELECT 1 FROM pages pg WHERE pg.workspace_id = @workspace_id AND pg.id <> @id AND pg.deleted_at IS NULL AND (pg.slug = @slug OR pg.live_slug = @slug)
) OR EXISTS (
    SELECT 1 FROM page_redirects pr WHERE pr.workspace_id = @workspace_id AND pr.slug = @slug AND pr.page_id <> @id
))::boolean AS taken;

-- name: PageInsert :exec
INSERT INTO pages (id, workspace_id, name, slug, template, visibility, show_in_nav, nav_label, project, html,
                   saved_at, saved_by, created_by, created_at, updated_by, updated_at)
VALUES (@id, @workspace_id, @name, @slug, @template, @visibility, @show_in_nav, @nav_label, @project, @html,
        @at, sqlc.narg(actor), sqlc.narg(actor), @at, sqlc.narg(actor), @at);

-- name: PageSaveContent :exec
-- Saves the draft's content; the version goes up by one.
UPDATE pages SET project = @project, html = @html, version = version + 1, saved_at = @at, saved_by = sqlc.narg(actor),
    autosaves = CASE WHEN @manual::boolean THEN 0 ELSE autosaves + 1 END, updated_at = @at, updated_by = sqlc.narg(actor)
WHERE id = @id;

-- name: PageSaveSettings :exec
UPDATE pages SET name = @name, slug = @slug, seo_title = @seo_title, seo_description = @seo_description, social_image = @social_image,
    visibility = @visibility, tiers = @tiers, hide_header = @hide_header, show_in_nav = @show_in_nav, nav_label = @nav_label,
    version = version + 1, saved_at = @at, saved_by = sqlc.narg(actor), updated_at = @at, updated_by = sqlc.narg(actor)
WHERE id = @id;

-- name: PageSchedule :exec
UPDATE pages SET status = @status, publish_at = sqlc.narg(publish_at), publish_tz = @publish_tz, updated_at = @at, updated_by = sqlc.narg(actor)
WHERE id = @id;

-- name: PagePublish :exec
-- Puts a version live at the page's address.
UPDATE pages SET status = 'published', live_version = @live_version, live_slug = @live_slug, publish_at = NULL, publish_tz = '',
    published_at = @at::timestamptz, published_by = sqlc.narg(actor), first_published_at = coalesce(first_published_at, @at::timestamptz),
    unpublished_at = NULL, unpublished_by = NULL, unpublish_reason = '', redirect_to = '', updated_at = @at
WHERE id = @id;

-- name: PageUnpublish :exec
UPDATE pages SET status = 'unpublished', unpublished_at = @at::timestamptz, unpublished_by = sqlc.narg(actor), unpublish_reason = @reason,
    redirect_to = @redirect_to, publish_at = NULL, publish_tz = '', updated_at = @at
WHERE id = @id;

-- name: PageRestoreLive :exec
-- Undo of a publish: the version that was live before, or none.
UPDATE pages SET status = @status, live_version = sqlc.narg(live_version), live_slug = sqlc.narg(live_slug),
    published_at = sqlc.narg(published_at), published_by = sqlc.narg(published_by),
    first_published_at = sqlc.narg(first_published_at), updated_at = @at
WHERE id = @id;

-- name: PageSoftDelete :exec
UPDATE pages SET deleted_at = @at::timestamptz, updated_at = @at, updated_by = sqlc.narg(actor) WHERE id = @id;

-- name: PagesDue :many
-- Scheduled pages whose time has come and that wait for no approval.
SELECT p.id, p.workspace_id FROM pages p
WHERE p.status = 'scheduled' AND p.deleted_at IS NULL AND p.publish_at <= @now
  AND NOT EXISTS (SELECT 1 FROM approval_requests a WHERE a.website_page_id = p.id AND a.status = 'pending')
ORDER BY p.publish_at
LIMIT 100;

-- Versions

-- name: PageVersionInsert :exec
INSERT INTO page_versions (page_id, version, name, slug, settings, project, html, label, saved_by, saved_at)
VALUES (@page_id, @version, @name, @slug, @settings, @project, @html, @label, sqlc.narg(saved_by), @saved_at)
ON CONFLICT (page_id, version) DO UPDATE SET name = EXCLUDED.name, slug = EXCLUDED.slug, settings = EXCLUDED.settings,
    project = EXCLUDED.project, html = EXCLUDED.html, label = EXCLUDED.label;

-- name: PageVersionPublished :exec
UPDATE page_versions SET published_at = @at::timestamptz, published_by = sqlc.narg(actor) WHERE page_id = @page_id AND version = @version;

-- name: PageVersionGet :one
SELECT sqlc.embed(v), coalesce(u.name, '')::text AS saved_by_name, coalesce(pb.name, '')::text AS published_by_name
FROM page_versions v
LEFT JOIN users u ON u.id = v.saved_by
LEFT JOIN users pb ON pb.id = v.published_by
WHERE v.page_id = @page_id AND v.version = @version;

-- name: PageVersions :many
SELECT v.version, v.name, v.label, v.saved_at, v.published_at, coalesce(u.name, '')::text AS saved_by_name,
       coalesce(pb.name, '')::text AS published_by_name, length(v.html)::bigint AS size
FROM page_versions v
LEFT JOIN users u ON u.id = v.saved_by
LEFT JOIN users pb ON pb.id = v.published_by
WHERE v.page_id = @page_id
ORDER BY v.version DESC
LIMIT 200;

-- name: PageVersionCount :one
SELECT count(*)::bigint FROM page_versions WHERE page_id = @page_id;

-- name: PagePreviousPublished :one
-- The version published before the live one (Undo).
SELECT * FROM page_versions WHERE page_id = @page_id AND published_at IS NOT NULL AND version < @version
ORDER BY version DESC LIMIT 1;

-- Earlier addresses

-- name: PageRedirectInsert :exec
INSERT INTO page_redirects (workspace_id, slug, page_id, created_at) VALUES (@workspace_id, @slug, @page_id, @at)
ON CONFLICT (workspace_id, slug) DO UPDATE SET page_id = EXCLUDED.page_id, created_at = EXCLUDED.created_at;

-- name: PageRedirectDelete :exec
DELETE FROM page_redirects WHERE workspace_id = @workspace_id AND slug = @slug;

-- name: PageRedirectGet :one
SELECT r.page_id FROM page_redirects r JOIN pages p ON p.id = r.page_id AND p.deleted_at IS NULL
WHERE r.workspace_id = @workspace_id AND r.slug = @slug;

-- Numbers

-- name: PageViewCounts :many
-- Views per page between two days (website_view_days, kind 'page'). The
-- shop counts as the nil page.
SELECT page_id, coalesce(sum(views), 0)::bigint AS views
FROM website_view_days
WHERE workspace_id = @workspace_id AND page_kind = 'page' AND day >= @since AND day <= @until
GROUP BY page_id;

-- name: PageConversionCounts :many
-- Conversions per page and kind between two times.
SELECT page_id, source, kind, count(*)::bigint AS n, coalesce(sum(amount_cents), 0)::bigint AS amount
FROM page_conversions
WHERE workspace_id = @workspace_id AND occurred_at >= @since AND occurred_at < @until
GROUP BY page_id, source, kind;

-- name: PageConversionInsert :exec
INSERT INTO page_conversions (id, workspace_id, page_id, source, kind, contact_id, order_id, amount_cents, occurred_at)
VALUES (@id, @workspace_id, sqlc.narg(page_id), @source, @kind, sqlc.narg(contact_id), sqlc.narg(order_id), @amount_cents, @occurred_at)
ON CONFLICT DO NOTHING;

-- name: PageConversionDeleteOrder :exec
-- A refunded sale leaves the page's numbers.
DELETE FROM page_conversions WHERE order_id = @order_id;

-- Header links (internal/website's navigation names custom pages by slug)

-- name: PageTitlesBySlug :many
-- The published pages among slugs, with the label their header link uses.
SELECT live_slug::text AS slug, CASE WHEN v.settings->>'navLabel' <> '' THEN v.settings->>'navLabel' ELSE v.name END::text AS title
FROM pages p
JOIN page_versions v ON v.page_id = p.id AND v.version = p.live_version
WHERE p.workspace_id = @workspace_id AND p.deleted_at IS NULL AND p.status = 'published' AND p.live_slug = ANY(@slugs::text[]);

-- name: PagesPublishedCount :one
SELECT count(*)::bigint FROM pages WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND status = 'published' AND visibility = 'public';

-- name: PagesPublished :many
-- Every published page (the sitemap, the navigation picker).
SELECT p.id, p.live_slug::text AS slug, v.name, v.settings, p.published_at
FROM pages p
JOIN page_versions v ON v.page_id = p.id AND v.version = p.live_version
WHERE p.workspace_id = @workspace_id AND p.deleted_at IS NULL AND p.status = 'published'
ORDER BY v.name;

-- The shop

-- name: ShopSettingsEnsure :exec
INSERT INTO shop_settings (workspace_id) VALUES (@workspace_id) ON CONFLICT (workspace_id) DO NOTHING;

-- name: ShopSettingsGet :one
SELECT * FROM shop_settings WHERE workspace_id = @workspace_id;

-- name: ShopSettingsSave :exec
UPDATE shop_settings SET shop_on = @shop_on, updated_at = @at, updated_by = sqlc.narg(actor) WHERE workspace_id = @workspace_id;
