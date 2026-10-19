-- The publication website (internal/website, design expansion phase 7):
-- its settings, tags, posts and their earlier addresses, views counted
-- without cookies, members, sign-in links, sessions and the meter.

-- name: WebsiteSettingsEnsure :exec
INSERT INTO website_settings (workspace_id) VALUES (@workspace_id) ON CONFLICT (workspace_id) DO NOTHING;

-- name: WebsiteSettingsGet :one
SELECT * FROM website_settings WHERE workspace_id = @workspace_id;

-- name: WebsiteSettingsLock :one
SELECT * FROM website_settings WHERE workspace_id = @workspace_id FOR UPDATE;

-- name: WebsiteSettingsSave :exec
UPDATE website_settings SET
    title = @title, tagline = @tagline, description = @description, social_image = @social_image, index_site = @index_site,
    home_layout = @home_layout, featured = @featured, sections = @sections, about_title = @about_title, about_text = @about_text, about_link = @about_link,
    header_links = @header_links, footer = @footer,
    signin_on = @signin_on, link_minutes = @link_minutes, session_days = @session_days, signup_on = @signup_on, signup_lists = @signup_lists,
    default_visibility = @default_visibility, campaign_visibility = @campaign_visibility,
    meter_visitors_on = @meter_visitors_on, meter_visitors = @meter_visitors, meter_members_on = @meter_members_on, meter_members = @meter_members,
    paywall = @paywall, updated_at = @at, updated_by = sqlc.narg(updated_by)
WHERE workspace_id = @workspace_id;

-- name: WebsiteSetStatus :exec
UPDATE website_settings SET status = @status, progress = @progress,
    turned_on_at = CASE WHEN @status = 'turning_on' THEN @at::timestamptz ELSE turned_on_at END,
    turned_on_by = CASE WHEN @status = 'turning_on' THEN sqlc.narg(actor)::uuid ELSE turned_on_by END,
    turned_off_at = CASE WHEN @status = 'off' THEN @at::timestamptz ELSE turned_off_at END,
    turned_off_by = CASE WHEN @status = 'off' THEN sqlc.narg(actor)::uuid ELSE turned_off_by END,
    updated_at = @at
WHERE workspace_id = @workspace_id;

-- name: WebsiteSetProgress :exec
UPDATE website_settings SET progress = @progress WHERE workspace_id = @workspace_id;

-- name: WebsiteSetDomainNotified :exec
UPDATE website_settings SET domain_notified = @domain_notified WHERE workspace_id = @workspace_id;

-- name: WebsiteTurningOn :many
-- Websites whose turning-on job hasn't finished (the job resumes them).
SELECT workspace_id FROM website_settings WHERE status = 'turning_on';

-- name: WebsitesOn :many
SELECT s.workspace_id FROM website_settings s
JOIN workspaces w ON w.id = s.workspace_id AND w.deleted_at IS NULL
WHERE s.status = 'on';

-- name: WebsiteSettingsUpdatedBy :one
SELECT coalesce(u.name, '')::text FROM website_settings s LEFT JOIN users u ON u.id = s.updated_by WHERE s.workspace_id = @workspace_id;

-- Tags

-- name: WebsiteTags :many
-- Every tag with its published posts (and whether only unpublished ones are left).
SELECT t.*,
       count(p.id) FILTER (WHERE p.status = 'published')::bigint AS published,
       count(p.id) FILTER (WHERE p.status = 'unpublished')::bigint AS unpublished
FROM website_tags t
LEFT JOIN website_post_tags pt ON pt.tag_id = t.id
LEFT JOIN website_posts p ON p.id = pt.post_id AND p.deleted_at IS NULL
WHERE t.workspace_id = @workspace_id
GROUP BY t.id
ORDER BY t.created_at, t.name;

-- name: WebsiteTagGet :one
SELECT * FROM website_tags WHERE workspace_id = @workspace_id AND id = @id;

-- name: WebsiteTagBySlug :one
SELECT * FROM website_tags WHERE workspace_id = @workspace_id AND slug = @slug;

-- name: WebsiteTagByList :one
SELECT * FROM website_tags WHERE workspace_id = @workspace_id AND list_id = @list_id LIMIT 1;

-- name: WebsiteTagSlugTaken :one
SELECT EXISTS (SELECT 1 FROM website_tags WHERE workspace_id = @workspace_id AND slug = @slug AND id <> @id)::boolean;

-- name: WebsiteTagInsert :exec
INSERT INTO website_tags (id, workspace_id, name, slug, description, list_id, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @name, @slug, @description, sqlc.narg(list_id), sqlc.narg(created_by), @at, @at);

-- name: WebsiteTagUpdate :exec
UPDATE website_tags SET name = @name, slug = @slug, description = @description, updated_at = @at
WHERE workspace_id = @workspace_id AND id = @id;

-- name: WebsiteTagDelete :exec
DELETE FROM website_tags WHERE workspace_id = @workspace_id AND id = @id;

-- Posts

-- name: WebsitePostInsert :exec
INSERT INTO website_posts (id, workspace_id, kind, campaign_id, web_on, status, publish_mode, publish_at, publish_tz, title, title_edited, slug,
                           excerpt, excerpt_edited, cover_url, cover_alt, byline_user_id, byline_name, visibility, tiers, never_free,
                           mjml, created_by, created_at, updated_by, updated_at)
VALUES (@id, @workspace_id, @kind, sqlc.narg(campaign_id), @web_on, @status, @publish_mode, sqlc.narg(publish_at), @publish_tz, @title, @title_edited, @slug,
        @excerpt, @excerpt_edited, @cover_url, @cover_alt, sqlc.narg(byline_user_id), @byline_name, @visibility, @tiers, @never_free,
        @mjml, sqlc.narg(created_by), @at, sqlc.narg(created_by), @at);

-- name: WebsitePostGet :one
SELECT sqlc.embed(p),
       coalesce(bu.name, '')::text AS byline_user_name, coalesce(bu.avatar_color, '')::text AS byline_color,
       coalesce(c.name, '')::text AS campaign_name, coalesce(c.status, '')::text AS campaign_status,
       c.started_at AS campaign_started_at, c.scheduled_at AS campaign_scheduled_at,
       coalesce(c.subject, '')::text AS campaign_subject, coalesce(c.preheader, '')::text AS campaign_preheader,
       coalesce(c.audience, '{}'::jsonb)::jsonb AS campaign_audience,
       coalesce(pb.name, '')::text AS published_by_name, coalesce(ub.name, '')::text AS unpublished_by_name,
       coalesce(cb.name, '')::text AS created_by_name
FROM website_posts p
LEFT JOIN users bu ON bu.id = p.byline_user_id
LEFT JOIN campaigns c ON c.id = p.campaign_id
LEFT JOIN users pb ON pb.id = p.published_by
LEFT JOIN users ub ON ub.id = p.unpublished_by
LEFT JOIN users cb ON cb.id = p.created_by
WHERE p.workspace_id = @workspace_id AND p.id = @id AND p.deleted_at IS NULL;

-- name: WebsitePostLock :one
SELECT * FROM website_posts WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL FOR UPDATE;

-- name: WebsitePostBySlug :one
SELECT * FROM website_posts WHERE workspace_id = @workspace_id AND slug = @slug AND deleted_at IS NULL;

-- name: WebsitePostByCampaign :one
SELECT * FROM website_posts WHERE campaign_id = @campaign_id AND deleted_at IS NULL;

-- name: WebsitePostsByCampaigns :many
-- The posts of campaigns (the Campaigns list's Web column).
SELECT p.id, p.campaign_id, p.status, p.web_on, p.publish_mode, p.publish_at, p.published_at, p.unpublished_at, p.slug, p.visibility
FROM website_posts p
WHERE p.workspace_id = @workspace_id AND p.campaign_id = ANY (@campaigns::uuid[]) AND p.deleted_at IS NULL;

-- name: WebsitePostSlugTaken :one
-- A slug in use by another post, or kept as an earlier address of one.
SELECT (EXISTS (SELECT 1 FROM website_posts wp WHERE wp.workspace_id = @workspace_id::uuid AND wp.slug = @slug::text AND wp.id <> @id::uuid AND wp.deleted_at IS NULL)
    OR EXISTS (SELECT 1 FROM website_redirects wr WHERE wr.workspace_id = @workspace_id::uuid AND wr.path = 'posts/' || @slug::text AND wr.post_id <> @id::uuid))::boolean AS taken;

-- name: WebsitePostSlugOwner :one
-- The post that holds a slug, or once held it.
SELECT p.id, p.title, p.status, coalesce(p.first_published_at, p.created_at)::timestamptz AS since
FROM website_posts p
WHERE p.workspace_id = @workspace_id AND p.deleted_at IS NULL AND p.id <> @id
  AND (p.slug = @slug OR EXISTS (SELECT 1 FROM website_redirects r WHERE r.workspace_id = @workspace_id AND r.path = 'posts/' || @slug AND r.post_id = p.id))
LIMIT 1;

-- name: WebsitePostSaveFields :exec
UPDATE website_posts SET
    web_on = @web_on, publish_mode = @publish_mode, publish_at = sqlc.narg(publish_at), publish_tz = @publish_tz,
    title = @title, title_edited = @title_edited, slug = @slug, excerpt = @excerpt, excerpt_edited = @excerpt_edited,
    cover_url = @cover_url, cover_alt = @cover_alt, byline_user_id = sqlc.narg(byline_user_id), byline_name = @byline_name,
    visibility = @visibility, tiers = @tiers, never_free = @never_free,
    seo_title = @seo_title, seo_description = @seo_description, social_image = @social_image, canonical_url = @canonical_url,
    hide_from_search = @hide_from_search, updated_by = sqlc.narg(updated_by), updated_at = @at
WHERE workspace_id = @workspace_id AND id = @id;

-- name: WebsitePostSaveContent :exec
UPDATE website_posts SET mjml = @mjml, version = @version, autosaved_at = sqlc.narg(autosaved_at), updated_by = sqlc.narg(updated_by), updated_at = @at
WHERE workspace_id = @workspace_id AND id = @id;

-- name: WebsitePostVersionInsert :exec
INSERT INTO website_post_versions (post_id, version, title, mjml, saved_by, saved_at, label)
VALUES (@post_id, @version, @title, @mjml, sqlc.narg(saved_by), @at, @label)
ON CONFLICT (post_id, version) DO UPDATE SET title = EXCLUDED.title, mjml = EXCLUDED.mjml, saved_by = EXCLUDED.saved_by, saved_at = EXCLUDED.saved_at, label = EXCLUDED.label;

-- name: WebsitePostVersions :many
SELECT v.version, v.title, v.saved_at, v.label, coalesce(u.name, '')::text AS saved_by_name
FROM website_post_versions v LEFT JOIN users u ON u.id = v.saved_by
WHERE v.post_id = @post_id ORDER BY v.version DESC LIMIT 50;

-- name: WebsitePostVersion :one
SELECT * FROM website_post_versions WHERE post_id = @post_id AND version = @version;

-- name: WebsitePostSetRendered :exec
-- The web version of the content a post was published with.
UPDATE website_posts SET published_mjml = @published_mjml, published_version = @published_version,
    body_html = @body_html, preview_html = @preview_html, has_marker = @has_marker, marker_after = @marker_after,
    body_text = @body_text, preview_text = @preview_text, words = @words, render_notes = @render_notes, paid_blocks = @paid_blocks,
    rendered_hash = @rendered_hash
WHERE id = @id;

-- name: WebsitePostPublish :exec
UPDATE website_posts SET status = 'published', published_at = @at::timestamptz, published_by = sqlc.narg(published_by),
    first_published_at = coalesce(first_published_at, @at::timestamptz), unpublished_at = NULL, unpublished_by = NULL, unpublish_reason = '', redirect_to = '',
    updated_at = greatest(updated_at, @at::timestamptz)
WHERE workspace_id = @workspace_id AND id = @id;

-- name: WebsitePostSchedule :exec
UPDATE website_posts SET status = 'scheduled', publish_mode = @publish_mode, publish_at = sqlc.narg(publish_at), publish_tz = @publish_tz,
    updated_by = sqlc.narg(updated_by), updated_at = @at
WHERE workspace_id = @workspace_id AND id = @id;

-- name: WebsitePostToDraft :exec
UPDATE website_posts SET status = 'draft', updated_by = sqlc.narg(updated_by), updated_at = @at
WHERE workspace_id = @workspace_id AND id = @id;

-- name: WebsitePostUnpublish :exec
UPDATE website_posts SET status = 'unpublished', unpublished_at = @at::timestamptz, unpublished_by = sqlc.narg(unpublished_by), unpublish_reason = @reason,
    redirect_to = @redirect_to, updated_at = @at::timestamptz
WHERE workspace_id = @workspace_id AND id = @id;

-- name: WebsitePostSetRedirect :exec
UPDATE website_posts SET redirect_to = @redirect_to, updated_at = @at WHERE workspace_id = @workspace_id AND id = @id;

-- name: WebsitePostSetSentAs :exec
UPDATE website_posts SET sent_as_campaign = @campaign_id, updated_at = @at WHERE workspace_id = @workspace_id AND id = @id;

-- name: WebsitePostDelete :exec
UPDATE website_posts SET deleted_at = @at::timestamptz, updated_by = sqlc.narg(updated_by), updated_at = @at::timestamptz WHERE workspace_id = @workspace_id AND id = @id;

-- name: WebsitePostsDue :many
-- Posts whose time has come: scheduled at a time, or set to go live with
-- their campaign's send once it started.
SELECT p.id, p.workspace_id FROM website_posts p
LEFT JOIN campaigns c ON c.id = p.campaign_id
JOIN website_settings s ON s.workspace_id = p.workspace_id AND s.status = 'on'
WHERE p.deleted_at IS NULL AND p.web_on AND p.status IN ('draft','scheduled')
  AND ((p.status = 'scheduled' AND p.publish_mode IN ('at','now') AND p.publish_at <= @now::timestamptz)
       OR (p.publish_mode = 'with_send' AND c.started_at IS NOT NULL AND c.started_at <= @now::timestamptz AND c.status IN ('sending','paused','sent')))
  AND NOT EXISTS (SELECT 1 FROM approval_requests ar WHERE ar.website_post_id = p.id AND ar.status = 'pending')
ORDER BY p.publish_at NULLS LAST, p.id
LIMIT 200;

-- name: WebsitePostTags :many
SELECT pt.post_id, t.id, t.name, t.slug, pt.position
FROM website_post_tags pt JOIN website_tags t ON t.id = pt.tag_id
WHERE pt.post_id = ANY (@posts::uuid[])
ORDER BY pt.post_id, pt.position, t.name;

-- name: WebsitePostClearTags :exec
DELETE FROM website_post_tags WHERE post_id = @post_id;

-- name: WebsitePostAddTag :exec
INSERT INTO website_post_tags (post_id, tag_id, position) VALUES (@post_id, @tag_id, @position)
ON CONFLICT (post_id, tag_id) DO UPDATE SET position = EXCLUDED.position;

-- name: WebsiteRedirectInsert :exec
INSERT INTO website_redirects (workspace_id, path, post_id, created_at) VALUES (@workspace_id, @path, @post_id, @at)
ON CONFLICT (workspace_id, path) DO UPDATE SET post_id = EXCLUDED.post_id;

-- name: WebsiteRedirectGet :one
SELECT p.* FROM website_redirects r JOIN website_posts p ON p.id = r.post_id AND p.deleted_at IS NULL
WHERE r.workspace_id = @workspace_id AND r.path = @path;

-- name: WebsiteRedirectCount :one
SELECT count(*)::bigint FROM website_redirects WHERE workspace_id = @workspace_id AND path LIKE 'archive/%';

-- name: WebsitePostCounts :many
-- Posts by kind, status and visibility; a campaign's post set to go live
-- with a scheduled send counts as scheduled.
SELECT p.kind, p.status, p.web_on, p.publish_mode, p.visibility, coalesce(c.status, '')::text AS campaign_status, count(*)::bigint AS n
FROM website_posts p LEFT JOIN campaigns c ON c.id = p.campaign_id
WHERE p.workspace_id = @workspace_id AND p.deleted_at IS NULL
GROUP BY p.kind, p.status, p.web_on, p.publish_mode, p.visibility, c.status;

-- name: WebsiteArchiveCampaigns :many
-- Sent campaigns of the archived lists, as the archive files them: the
-- posts turning the website on creates.
SELECT ai.campaign_id, ai.slug, ai.list_id, ai.title, ai.excerpt, ai.hidden, c.name, c.subject, c.preheader, c.audience,
       coalesce(c.started_at, c.completed_at, c.created_at)::timestamptz AS sent_at
FROM archive_issues ai
JOIN campaigns c ON c.id = ai.campaign_id AND c.deleted_at IS NULL AND c.status = 'sent'
WHERE ai.workspace_id = @workspace_id AND ai.list_id = ANY (@lists::uuid[]) AND NOT ai.hidden
  AND NOT EXISTS (SELECT 1 FROM digests dg WHERE dg.id = c.digest_id AND NOT dg.archive)
ORDER BY sent_at, c.id;

-- name: WebsiteNewIssues :many
-- Campaigns of the archived lists that the archive filed while the website
-- was on and that have no post (not even a deleted one): they go on the
-- site with their send, as turning the website on did with older ones.
SELECT ai.campaign_id, ai.slug, ai.list_id, ai.title, ai.excerpt, ai.hidden, c.name, c.subject, c.preheader, c.audience,
       coalesce(c.started_at, c.completed_at, c.created_at)::timestamptz AS sent_at
FROM archive_issues ai
JOIN campaigns c ON c.id = ai.campaign_id AND c.deleted_at IS NULL AND c.status = 'sent'
WHERE ai.workspace_id = @workspace_id AND ai.list_id = ANY (@lists::uuid[]) AND NOT ai.hidden
  AND NOT EXISTS (SELECT 1 FROM digests dg WHERE dg.id = c.digest_id AND NOT dg.archive)
  AND NOT EXISTS (SELECT 1 FROM website_posts p WHERE p.campaign_id = c.id)
ORDER BY sent_at, c.id
LIMIT 100;

-- name: WebsiteSentWithoutPost :many
-- Sent campaigns that aren't posts yet ("Publish past campaigns").
SELECT c.id, c.name, c.subject, c.preheader, c.audience, coalesce(c.audience_count, 0)::bigint AS recipients,
       coalesce(c.started_at, c.completed_at, c.created_at)::timestamptz AS sent_at,
       coalesce(ai.hidden, false)::boolean AS archive_hidden, (ai.campaign_id IS NOT NULL)::boolean AS archived
FROM campaigns c
LEFT JOIN archive_issues ai ON ai.campaign_id = c.id
WHERE c.workspace_id = @workspace_id AND c.status = 'sent' AND c.deleted_at IS NULL AND coalesce(c.digest_role, '') IN ('', 'issue')
  AND NOT EXISTS (SELECT 1 FROM website_posts p WHERE p.campaign_id = c.id AND p.deleted_at IS NULL AND (p.web_on OR p.status <> 'draft'))
ORDER BY sent_at DESC, c.id DESC
LIMIT 100;

-- Views

-- name: WebsiteSalt :one
SELECT salt FROM website_view_salts WHERE day = @day;

-- name: WebsiteSaltInsert :one
INSERT INTO website_view_salts (day, salt) VALUES (@day, @salt)
ON CONFLICT (day) DO UPDATE SET day = EXCLUDED.day
RETURNING salt;

-- name: WebsiteSeen :one
-- True the first time a hash is seen on a day.
WITH ins AS (INSERT INTO website_view_seen (day, hash) VALUES (@day, @hash) ON CONFLICT DO NOTHING RETURNING 1)
SELECT EXISTS (SELECT 1 FROM ins)::boolean;

-- name: WebsiteViewAdd :exec
INSERT INTO website_view_days (workspace_id, day, page_kind, page_id, views) VALUES (@workspace_id, @day, @page_kind, @page_id, @views)
ON CONFLICT (workspace_id, day, page_kind, page_id) DO UPDATE SET views = website_view_days.views + EXCLUDED.views;

-- name: WebsiteVisitorAdd :exec
INSERT INTO website_visitor_days (workspace_id, day, visitors) VALUES (@workspace_id, @day, @visitors)
ON CONFLICT (workspace_id, day) DO UPDATE SET visitors = website_visitor_days.visitors + EXCLUDED.visitors;

-- name: WebsiteViewSourceAdd :exec
INSERT INTO website_view_sources (workspace_id, post_id, day, dim, key, views) VALUES (@workspace_id, @post_id, @day, @dim, @key, @views)
ON CONFLICT (post_id, day, dim, key) DO UPDATE SET views = website_view_sources.views + EXCLUDED.views;

-- name: WebsiteViewsPurge :exec
-- Salts and hashes of days that are over: nothing links a visitor across days.
DELETE FROM website_view_seen WHERE day < @day;

-- name: WebsiteSaltsPurge :exec
DELETE FROM website_view_salts WHERE day < @day;

-- name: WebsiteViewTotals :one
SELECT coalesce((SELECT sum(views) FROM website_view_days v WHERE v.workspace_id = @workspace_id AND v.day >= @from_day AND v.day <= @until_day), 0)::bigint AS views,
       coalesce((SELECT sum(visitors) FROM website_visitor_days d WHERE d.workspace_id = @workspace_id AND d.day >= @from_day AND d.day <= @until_day), 0)::bigint AS visitors;

-- name: WebsitePostViewCounts :many
SELECT page_id AS post_id, sum(views)::bigint AS views
FROM website_view_days
WHERE workspace_id = @workspace_id AND page_kind = 'post' AND page_id = ANY (@posts::uuid[]) AND day >= @from_day AND day <= @until_day
GROUP BY page_id;

-- name: WebsitePostSources :many
SELECT dim, key, sum(views)::bigint AS views
FROM website_view_sources
WHERE post_id = @post_id AND day >= @from_day AND day <= @until_day
GROUP BY dim, key
ORDER BY views DESC, key;

-- name: WebsiteConversionInsert :exec
INSERT INTO website_post_conversions (id, workspace_id, post_id, kind, contact_id, occurred_at)
VALUES (@id, @workspace_id, @post_id, @kind, sqlc.narg(contact_id), @at);

-- name: WebsiteConversionCounts :many
SELECT post_id, kind, count(*)::bigint AS n
FROM website_post_conversions
WHERE workspace_id = @workspace_id AND post_id = ANY (@posts::uuid[]) AND occurred_at >= @from_day AND occurred_at < @until_day
GROUP BY post_id, kind;

-- name: WebsiteConversionTotals :one
SELECT count(*) FILTER (WHERE kind = 'signup')::bigint AS signups, count(*) FILTER (WHERE kind = 'upgrade')::bigint AS upgrades
FROM website_post_conversions
WHERE workspace_id = @workspace_id AND occurred_at >= @from_day AND occurred_at < @until_day;

-- Members

-- name: WebsiteMemberSignedIn :exec
INSERT INTO website_members (contact_id, workspace_id, first_signed_in, last_signed_in, last_method, sign_ins)
VALUES (@contact_id, @workspace_id, @at, @at, @method, 1)
ON CONFLICT (contact_id) DO UPDATE SET last_signed_in = EXCLUDED.last_signed_in, last_method = EXCLUDED.last_method, sign_ins = website_members.sign_ins + 1;

-- name: WebsiteMemberGet :one
SELECT * FROM website_members WHERE contact_id = @contact_id AND workspace_id = @workspace_id;

-- name: WebsiteMemberCounts :one
SELECT count(*) FILTER (WHERE last_signed_in >= @since)::bigint AS since,
       count(*) FILTER (WHERE last_signed_in >= @week)::bigint AS week,
       count(*)::bigint AS total
FROM website_members WHERE workspace_id = @workspace_id;

-- name: WebsiteMemberEventInsert :exec
INSERT INTO website_member_events (id, workspace_id, contact_id, kind, method, same_browser, actor_id, detail, occurred_at)
VALUES (@id, @workspace_id, @contact_id, @kind, @method, sqlc.narg(same_browser), sqlc.narg(actor_id), @detail, @at);

-- name: WebsiteMemberEvents :many
SELECT e.*, coalesce(u.name, '')::text AS actor_name
FROM website_member_events e LEFT JOIN users u ON u.id = e.actor_id
WHERE e.workspace_id = @workspace_id AND e.contact_id = @contact_id
ORDER BY e.occurred_at DESC, e.id DESC
LIMIT @lim;

-- Sign-in links

-- name: WebsiteLinkInsert :exec
INSERT INTO website_signin_links (id, workspace_id, contact_id, purpose, token_hash, code_hash, browser_hash, return_to, new_email, lists, created_at, expires_at)
VALUES (@id, @workspace_id, @contact_id, @purpose, @token_hash, @code_hash, @browser_hash, @return_to, @new_email, @lists, @at, @expires_at);

-- name: WebsiteLinkReplaceOlder :exec
-- A newer link replaces the contact's earlier links of the same purpose.
UPDATE website_signin_links SET replaced_at = @at
WHERE contact_id = @contact_id AND purpose = @purpose AND used_at IS NULL AND replaced_at IS NULL AND id <> @id;

-- name: WebsiteLinkByToken :one
SELECT * FROM website_signin_links WHERE token_hash = @token_hash;

-- name: WebsiteLinkLatestForBrowser :one
-- The newest link this browser asked for (the code is typed where the link was asked for).
SELECT * FROM website_signin_links
WHERE workspace_id = @workspace_id AND browser_hash = @browser_hash AND purpose IN ('signin','signup')
ORDER BY created_at DESC LIMIT 1;

-- name: WebsiteLinksLastHour :one
SELECT count(*)::bigint FROM website_signin_links WHERE contact_id = @contact_id AND created_at > @since;

-- name: WebsiteLinkOldestInHour :one
SELECT min(created_at)::timestamptz FROM website_signin_links WHERE contact_id = @contact_id AND created_at > @since;

-- name: WebsiteLinkUse :one
-- Uses a link once: false when it was used, replaced or expired meanwhile.
UPDATE website_signin_links SET used_at = @at, used_same_browser = @same_browser
WHERE id = @id AND used_at IS NULL AND replaced_at IS NULL AND expires_at > @at
RETURNING id;

-- name: WebsiteLinkTry :one
UPDATE website_signin_links SET code_tries = code_tries + 1 WHERE id = @id RETURNING code_tries;

-- Sessions

-- name: WebsiteSessionInsert :exec
INSERT INTO website_sessions (id, workspace_id, contact_id, token_hash, method, created_at, last_seen_at, expires_at)
VALUES (@id, @workspace_id, @contact_id, @token_hash, @method, @at, @at, @expires_at);

-- name: WebsiteSessionByToken :one
SELECT s.*, c.email, c.first_name, c.last_name, c.status AS contact_status, c.erased_at
FROM website_sessions s JOIN contacts c ON c.id = s.contact_id
WHERE s.token_hash = @token_hash AND s.ended_at IS NULL AND s.expires_at > @now::timestamptz;

-- name: WebsiteSessionTouch :exec
UPDATE website_sessions SET last_seen_at = @at WHERE id = @id;

-- name: WebsiteSessionEnd :exec
UPDATE website_sessions SET ended_at = @at, end_reason = @reason WHERE id = @id AND ended_at IS NULL;

-- name: WebsiteSessionsEndAll :one
WITH ended AS (
    UPDATE website_sessions SET ended_at = @at, end_reason = @reason
    WHERE workspace_id = @workspace_id AND contact_id = @contact_id AND ended_at IS NULL
    RETURNING 1)
SELECT count(*)::bigint FROM ended;

-- name: WebsiteSessionsActive :many
SELECT * FROM website_sessions
WHERE workspace_id = @workspace_id AND contact_id = @contact_id AND ended_at IS NULL AND expires_at > @now::timestamptz
ORDER BY last_seen_at DESC;

-- The meter

-- name: WebsiteMeterState :one
SELECT count(*)::bigint AS used, coalesce(bool_or(post_id = @post_id), false)::boolean AS read
FROM website_meter_reads
WHERE workspace_id = @workspace_id AND month = @month AND reader = @reader AND rule = @rule;

-- name: WebsiteMeterRead :exec
INSERT INTO website_meter_reads (workspace_id, month, reader, post_id, rule, read_at)
VALUES (@workspace_id, @month, @reader, @post_id, @rule, @at)
ON CONFLICT DO NOTHING;

-- name: WebsiteMeterReads :many
SELECT r.post_id, r.read_at, p.title FROM website_meter_reads r JOIN website_posts p ON p.id = r.post_id
WHERE r.workspace_id = @workspace_id AND r.month = @month AND r.reader = @reader AND r.rule = @rule
ORDER BY r.read_at;

-- Paid tiers of readers

-- name: WebsiteReaderTiers :many
-- The paid tiers a contact holds now (internal/earn's access rule).
SELECT t.id, t.name, t.access_rank
FROM paid_subscriptions s JOIN earn_tiers t ON t.id = s.tier_id AND t.deleted_at IS NULL
WHERE s.workspace_id = @workspace_id AND s.contact_id = @contact_id AND s.mode = 'live' AND s.ended_at IS NULL
  AND (s.status IN ('trialing','active','complimentary','gift') OR (s.status = 'past_due' AND NOT s.access_paused));

-- name: WebsiteTiers :many
-- The workspace's paid tiers that can be sold or held (not archived).
SELECT t.id, t.name, t.slug, t.access_rank, t.visibility FROM earn_tiers t
WHERE t.workspace_id = @workspace_id AND t.deleted_at IS NULL AND t.archived_at IS NULL
ORDER BY t.access_rank, t.created_at;

-- name: WebsiteTierPrices :many
-- The paid tiers that can be bought, with their current prices: the
-- paywall card's rows and the account page's plan.
SELECT t.id, t.name, t.slug, t.access_rank, t.trial_days, t.visibility, t.benefits, t.description,
       p.period, p.amount_cents, p.currency
FROM earn_tiers t
JOIN earn_prices p ON p.tier_id = t.id AND p.status = 'current'
WHERE t.workspace_id = @workspace_id AND t.deleted_at IS NULL AND t.archived_at IS NULL
ORDER BY t.access_rank, t.created_at, p.period;

-- name: WebsitePayments :one
-- The payment connection the site sells through: its mode and account ("" when none is connected).
SELECT coalesce((SELECT c.mode FROM payment_connections c WHERE c.workspace_id = @workspace_id AND c.status = 'connected'
                 ORDER BY (c.mode = 'live') DESC LIMIT 1), '')::text AS mode,
       coalesce((SELECT c.account_name FROM payment_connections c WHERE c.workspace_id = @workspace_id AND c.status = 'connected'
                 ORDER BY (c.mode = 'live') DESC LIMIT 1), '')::text AS account;
