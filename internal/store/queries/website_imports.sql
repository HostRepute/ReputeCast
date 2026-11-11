-- Importing posts from Substack, Ghost and WordPress (M61, G-26):
-- internal/website/imports*.go.

-- name: WebsiteImportInsert :exec
INSERT INTO website_imports (id, workspace_id, source, status, file_name, file_size, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @source, 'uploaded', @file_name, @file_size, sqlc.narg(created_by), @at, @at);

-- name: WebsiteImportFileInsert :exec
INSERT INTO website_import_files (import_id, data) VALUES (@import_id, @data);

-- name: WebsiteImportFile :one
SELECT data FROM website_import_files WHERE import_id = @import_id;

-- name: WebsiteImportFileDelete :exec
DELETE FROM website_import_files WHERE import_id = @import_id;

-- name: WebsiteImportGet :one
-- An import of a workspace, with who uploaded it.
SELECT i.id, i.workspace_id, i.source, i.status, i.options, i.stats, i.error, (i.report_csv IS NOT NULL)::boolean AS has_report,
       i.file_name, i.file_size, i.created_by, i.created_at, i.updated_at, i.finished_at, coalesce(u.name, '')::text AS created_by_name
FROM website_imports i LEFT JOIN users u ON u.id = i.created_by
WHERE i.workspace_id = @workspace_id AND i.id = @id;

-- name: WebsiteImportByID :one
-- An import for its background jobs.
SELECT id, workspace_id, source, status, options, stats, error, file_name, file_size, created_by, created_at, updated_at, finished_at
FROM website_imports WHERE id = @id;

-- name: WebsiteImportLock :one
SELECT id, workspace_id, source, status, options, stats, error, file_name, file_size, created_by, created_at, updated_at, finished_at
FROM website_imports WHERE workspace_id = @workspace_id AND id = @id FOR UPDATE;

-- name: WebsiteImportStatus :one
SELECT status FROM website_imports WHERE id = @id;

-- name: WebsiteImportMove :execrows
-- Moves an import on from one of the statuses it may move from.
UPDATE website_imports SET status = @status, updated_at = @at
WHERE id = @id AND status = ANY (@from_statuses::text[]);

-- name: WebsiteImportResume :execrows
-- Try again after a run stopped: it carries on from its cursor.
UPDATE website_imports SET status = 'importing', stats = @stats, error = '', finished_at = NULL, updated_at = @at
WHERE id = @id AND status = 'failed';

-- name: WebsiteImportSetOptions :exec
UPDATE website_imports SET options = @options, updated_at = @at WHERE id = @id;

-- name: WebsiteImportSetStats :exec
UPDATE website_imports SET stats = @stats, updated_at = @at WHERE id = @id;

-- name: WebsiteImportReady :exec
-- The analysis is done: the posts are staged.
UPDATE website_imports SET status = 'ready', options = @options, stats = @stats, updated_at = @at WHERE id = @id AND status = 'analysing';

-- name: WebsiteImportFinish :exec
UPDATE website_imports SET status = @status, stats = @stats, error = @error, report_csv = sqlc.narg(report_csv), finished_at = @at, updated_at = @at
WHERE id = @id;

-- name: WebsiteImportReport :one
SELECT report_csv, file_name, finished_at FROM website_imports
WHERE workspace_id = @workspace_id AND id = @id AND report_csv IS NOT NULL;

-- name: WebsiteImportTray :many
-- The imports the jobs tray shows: running ones, and ones that ended since.
SELECT id, source, status, stats, error, file_name, created_at, updated_at, finished_at
FROM website_imports
WHERE workspace_id = @workspace_id
  AND (status IN ('uploaded','analysing','importing') OR (status = 'ready' AND updated_at > @ended_after::timestamptz)
       OR (status IN ('done','failed','canceled') AND finished_at > @ended_after::timestamptz))
ORDER BY created_at DESC
LIMIT 20;

-- name: WebsiteImportsPurgeReports :exec
-- Reports are kept 30 days.
UPDATE website_imports SET report_csv = NULL WHERE report_csv IS NOT NULL AND finished_at < @before::timestamptz;

-- name: WebsiteImportsPurgeFiles :exec
-- An ended import's file goes (the run drops it; this catches the rest).
DELETE FROM website_import_files f USING website_imports i
WHERE f.import_id = i.id AND i.status IN ('done','failed','canceled');

-- Staged posts

-- name: WebsiteImportPostsInsert :copyfrom
INSERT INTO website_import_posts (id, import_id, source_id, title, slug, published_at, status, visibility, tier_refs, tags, author, excerpt,
                                  cover_url, seo, legacy_paths, html, images, problems, decision, post_id)
VALUES (@id, @import_id, @source_id, @title, @slug, @published_at, @status, @visibility, @tier_refs, @tags, @author, @excerpt,
        @cover_url, @seo, @legacy_paths, @html, @images, @problems, @decision, @post_id);

-- name: WebsiteImportPostsClear :exec
DELETE FROM website_import_posts WHERE import_id = @import_id;

-- name: WebsiteImportPostList :many
-- A page of the staged posts, in the export's order (newest first, drafts
-- last), with the status of the post each became.
SELECT ip.id, ip.source_id, ip.title, ip.slug, ip.published_at, ip.status, ip.visibility, ip.tier_refs, ip.tags, ip.author, ip.images, ip.problems,
       ip.decision, ip.post_id, coalesce(ip.seo->>'sourceUrl', '')::text AS source_url, coalesce(wp.status, '')::text AS post_status
FROM website_import_posts ip
LEFT JOIN website_posts wp ON wp.id = ip.post_id
WHERE ip.import_id = @import_id AND ip.id > @after::uuid
ORDER BY ip.id
LIMIT @lim;

-- name: WebsiteImportPostsDropHTML :exec
-- An ended import's staged posts keep their outcome, not their bodies.
UPDATE website_import_posts SET html = '' WHERE import_id = @import_id;

-- name: WebsiteImportPostsToRun :many
-- The next staged posts of a run, after its cursor.
SELECT * FROM website_import_posts
WHERE import_id = @import_id AND id > @after::uuid
ORDER BY id
LIMIT @lim;

-- name: WebsiteImportPostRefs :many
-- The staged posts' decisions, for checking new ones.
SELECT id, decision, post_id FROM website_import_posts WHERE import_id = @import_id AND id = ANY (@ids::uuid[]);

-- name: WebsiteImportPostSetDecision :exec
UPDATE website_import_posts SET decision = @decision WHERE import_id = @import_id AND id = @id;

-- name: WebsiteImportPostDone :exec
-- A staged post after its run: the post it became, its address and problems.
UPDATE website_import_posts SET post_id = sqlc.narg(post_id), slug = @slug, problems = @problems WHERE id = @id;

-- name: WebsiteImportCounts :one
-- The staged posts in numbers (the Review step's).
SELECT count(*)::bigint AS found,
       count(*) FILTER (WHERE decision = 'import')::bigint AS to_import,
       count(*) FILTER (WHERE decision = 'update')::bigint AS to_update,
       count(*) FILTER (WHERE decision = 'skip')::bigint AS to_skip,
       count(*) FILTER (WHERE status = 'draft')::bigint AS drafts,
       count(*) FILTER (WHERE status = 'draft' AND decision = 'skip')::bigint AS drafts_skipped,
       count(*) FILTER (WHERE problems @> '[{"code":"already_imported"}]'::jsonb)::bigint AS already_imported,
       count(*) FILTER (WHERE visibility = 'paid' AND decision <> 'skip')::bigint AS paid,
       count(*) FILTER (WHERE visibility = 'members' AND decision <> 'skip')::bigint AS members,
       coalesce(sum(images), 0)::bigint AS images,
       count(*) FILTER (WHERE jsonb_array_length(problems) > 0 AND NOT problems @> '[{"code":"already_imported"}]'::jsonb)::bigint AS with_problems
FROM website_import_posts WHERE import_id = @import_id;

-- name: WebsiteImportTierRefs :many
-- The old site's tiers the staged posts to import name, with how many posts each.
SELECT ref::text AS ref, count(*)::bigint AS posts
FROM website_import_posts, unnest(tier_refs) AS ref
WHERE import_id = @import_id AND decision <> 'skip'
GROUP BY ref ORDER BY count(*) DESC, ref;

-- name: WebsiteImportAuthors :many
-- The authors of the staged posts to import, with how many posts each.
SELECT author, count(*)::bigint AS posts
FROM website_import_posts
WHERE import_id = @import_id AND author <> '' AND decision <> 'skip'
GROUP BY author ORDER BY count(*) DESC, author;

-- Imported posts

-- name: WebsiteImportedPosts :many
-- The posts already imported from a platform, by their source ids.
SELECT id, import_source_id::text AS source_id FROM website_posts
WHERE workspace_id = @workspace_id AND import_source = @source AND import_source_id = ANY (@source_ids::text[]) AND deleted_at IS NULL;

-- name: WebsiteImportedPost :one
SELECT * FROM website_posts
WHERE workspace_id = @workspace_id AND import_source = @source AND import_source_id = @source_id AND deleted_at IS NULL;

-- name: WebsitePostImportInsert :exec
-- An imported post: a web-only draft with its settings and content.
INSERT INTO website_posts (id, workspace_id, kind, web_on, status, publish_mode, title, title_edited, slug, excerpt, excerpt_edited,
                           cover_url, cover_alt, byline_user_id, byline_name, visibility, tiers, seo_title, seo_description, canonical_url,
                           mjml, import_source, import_source_id, created_by, created_at, updated_by, updated_at)
VALUES (@id, @workspace_id, 'web', true, 'draft', 'now', @title, true, @slug, @excerpt, true,
        @cover_url, @cover_alt, sqlc.narg(byline_user_id), @byline_name, @visibility, @tiers, @seo_title, @seo_description, @canonical_url,
        @mjml, @import_source, @import_source_id, sqlc.narg(created_by), @at, sqlc.narg(created_by), @at);

-- name: WebsitePostImportUpdate :exec
-- An imported post imported again ("Update"): its settings and content as
-- the export has them now. Its address stays.
UPDATE website_posts SET title = @title, excerpt = @excerpt, cover_url = @cover_url, cover_alt = @cover_alt,
    byline_user_id = sqlc.narg(byline_user_id), byline_name = @byline_name, visibility = @visibility, tiers = @tiers,
    seo_title = @seo_title, seo_description = @seo_description, canonical_url = @canonical_url,
    mjml = @mjml, version = version + 1, updated_by = sqlc.narg(updated_by), updated_at = @at
WHERE workspace_id = @workspace_id AND id = @id;

-- name: WebsiteLegacyPathInsert :exec
INSERT INTO website_legacy_paths (workspace_id, path, post_id, created_at) VALUES (@workspace_id, @path, @post_id, @at)
ON CONFLICT (workspace_id, path) DO UPDATE SET post_id = EXCLUDED.post_id;

-- name: WebsiteLegacyPathPost :one
-- The post an old address of the site leads to.
SELECT p.slug FROM website_legacy_paths l JOIN website_posts p ON p.id = l.post_id AND p.deleted_at IS NULL
WHERE l.workspace_id = @workspace_id AND l.path = @path;
