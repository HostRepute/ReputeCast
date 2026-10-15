-- Accessibility checks (X06): the workspace policy, linked pages' titles
-- and the fixes applied from Review & send.

-- name: AccessibilityPolicy :one
SELECT p.on_serious, p.min_text_px, p.updated_by, p.updated_at, coalesce(u.name, '')::text AS updated_by_name
FROM accessibility_policies p
LEFT JOIN users u ON u.id = p.updated_by
WHERE p.workspace_id = @workspace_id;

-- name: AccessibilityUpsertPolicy :exec
INSERT INTO accessibility_policies (workspace_id, on_serious, min_text_px, updated_by, updated_at)
VALUES (@workspace_id, @on_serious, @min_text_px, @updated_by, @now)
ON CONFLICT (workspace_id) DO UPDATE
SET on_serious = EXCLUDED.on_serious, min_text_px = EXCLUDED.min_text_px, updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at;

-- name: AccessibilityLinkTitles :many
SELECT url, title, lang, fetched_at FROM accessibility_link_titles WHERE url = ANY(@urls::text[]);

-- name: AccessibilityUpsertLinkTitle :exec
INSERT INTO accessibility_link_titles (url, title, lang, status, fetched_at)
VALUES (@url, @title, @lang, @status, @fetched_at)
ON CONFLICT (url) DO UPDATE SET title = EXCLUDED.title, lang = EXCLUDED.lang, status = EXCLUDED.status, fetched_at = EXCLUDED.fetched_at;

-- name: AccessibilityInsertFix :exec
INSERT INTO accessibility_fixes (id, workspace_id, campaign_id, content_hash, kinds, summary, languages, fixed_by, fixed_at)
VALUES (@id, @workspace_id, @campaign_id, @content_hash, @kinds::text[], @summary, @languages::text[], @fixed_by, @fixed_at);

-- name: AccessibilityLatestFix :one
SELECT f.content_hash, f.kinds, f.summary, f.languages, f.fixed_at, coalesce(u.name, '')::text AS fixed_by_name
FROM accessibility_fixes f
LEFT JOIN users u ON u.id = f.fixed_by
WHERE f.campaign_id = @campaign_id
ORDER BY f.fixed_at DESC
LIMIT 1;

-- name: AccessibilityCampaign :one
SELECT id, name, status, subject, preheader, mjml, html, content_hash, version, scheduled_at, editor_mode
FROM campaigns
WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL;

-- name: AccessibilityPendingCampaigns :many
-- Campaigns that haven't started: drafts, waiting for or holding an
-- approval, and scheduled sends.
SELECT id, name, status, mjml, html, scheduled_at
FROM campaigns
WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND status IN ('draft', 'approval', 'approved', 'scheduled')
  AND (mjml <> '' OR html <> '')
ORDER BY (status = 'scheduled') DESC, coalesce(scheduled_at, updated_at) DESC
LIMIT 200;

-- name: AccessibilityDraftAutomations :many
-- Automations with a draft version that isn't live yet.
SELECT id, name, status, live_version, draft_version
FROM automations
WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND (live_version IS NULL OR draft_version <> live_version)
ORDER BY updated_at DESC
LIMIT 100;

-- name: AccessibilityTemplates :many
SELECT id, name, mjml, html FROM templates WHERE workspace_id = @workspace_id AND kind = 'template' ORDER BY updated_at DESC;
