-- Content languages (design expansion M17): workspace languages, language
-- versions of emails and their sources.

-- name: LanguagesList :many
-- Every language the workspace has had, removed ones included (they keep
-- their page copy status), with who added it.
SELECT wl.code, wl.is_default, wl.is_fallback, wl.added_at, wl.removed_at,
       wl.page_copy_status, wl.page_copy_missing, wl.page_copy_at,
       coalesce(au.name, '')::text AS added_by_name,
       coalesce(pu.name, '')::text AS page_copy_by_name
FROM workspace_languages wl
LEFT JOIN users au ON au.id = wl.added_by
LEFT JOIN users pu ON pu.id = wl.page_copy_by
WHERE wl.workspace_id = @workspace_id
ORDER BY wl.is_default DESC, wl.added_at, wl.code;

-- name: LanguagesEnsureDefault :exec
-- A workspace that never chose its default writes in English: English
-- becomes its default (and its fallback unless it has one).
INSERT INTO workspace_languages (workspace_id, code, is_default, is_fallback, added_at, page_copy_status)
SELECT @workspace_id, 'en', true, NOT EXISTS (SELECT 1 FROM workspace_languages f WHERE f.workspace_id = @workspace_id AND f.is_fallback), w.created_at, 'live'
FROM workspaces w
WHERE w.id = @workspace_id AND NOT EXISTS (SELECT 1 FROM workspace_languages d WHERE d.workspace_id = @workspace_id AND d.is_default)
ON CONFLICT (workspace_id, code) DO UPDATE SET is_default = true, is_fallback = EXCLUDED.is_fallback OR workspace_languages.is_fallback, removed_at = NULL, removed_by = NULL;

-- name: LanguagesLock :many
SELECT code FROM workspace_languages WHERE workspace_id = @workspace_id FOR UPDATE;

-- name: LanguagesAdd :exec
-- Adding a language that was removed brings it back with its page copy.
INSERT INTO workspace_languages (workspace_id, code, added_by, added_at, page_copy_status)
VALUES (@workspace_id, @code, @added_by, @at, @page_copy_status)
ON CONFLICT (workspace_id, code) DO UPDATE
SET removed_at = NULL, removed_by = NULL, added_by = EXCLUDED.added_by, added_at = EXCLUDED.added_at;

-- name: LanguagesRemove :execrows
UPDATE workspace_languages SET removed_at = @at, removed_by = @removed_by
WHERE workspace_id = @workspace_id AND code = @code AND removed_at IS NULL AND NOT is_default AND NOT is_fallback;

-- name: LanguagesClearDefault :exec
UPDATE workspace_languages SET is_default = false WHERE workspace_id = @workspace_id AND is_default;

-- name: LanguagesSetDefault :execrows
UPDATE workspace_languages SET is_default = true WHERE workspace_id = @workspace_id AND code = @code AND removed_at IS NULL;

-- name: LanguagesClearFallback :exec
UPDATE workspace_languages SET is_fallback = false WHERE workspace_id = @workspace_id AND is_fallback;

-- name: LanguagesSetFallback :execrows
UPDATE workspace_languages SET is_fallback = true WHERE workspace_id = @workspace_id AND code = @code AND removed_at IS NULL;

-- name: LanguagesSetPageCopy :execrows
UPDATE workspace_languages
SET page_copy_status = @status, page_copy_missing = @missing, page_copy_by = sqlc.narg(by), page_copy_at = @at
WHERE workspace_id = @workspace_id AND code = @code;

-- name: LanguagesContactCodes :many
-- Contacts per language code as stored ("es-MX", "" for none), erased
-- contacts left out, as on Contacts.
SELECT lower(replace(language, '_', '-'))::text AS code, count(*)::bigint AS n
FROM contacts
WHERE workspace_id = @workspace_id AND erased_at IS NULL
GROUP BY 1;

-- name: LanguagesVersionsIn :many
-- What has a version in a language (the Remove dialog): unsent campaigns,
-- automation emails used by a flow, transactional templates.
SELECT t.owner_kind, t.owner_id, t.owner_version, t.status, t.held,
       coalesce(c.name, ae.name, tt.key, '')::text AS name,
       coalesce(c.status, '')::text AS campaign_status,
       coalesce(a.name, '')::text AS automation_name,
       coalesce(a.id, '00000000-0000-0000-0000-000000000000'::uuid)::uuid AS automation_id,
       coalesce(tt.live_version, 0)::int AS live_version
FROM content_translations t
LEFT JOIN campaigns c ON t.owner_kind = 'campaign' AND c.id = t.owner_id AND c.deleted_at IS NULL
LEFT JOIN automation_emails ae ON t.owner_kind = 'automation_email' AND ae.id = t.owner_id AND ae.deleted_at IS NULL
LEFT JOIN automations a ON a.id = ae.automation_id
LEFT JOIN transactional_templates tt ON t.owner_kind = 'transactional_template' AND tt.id = t.owner_id AND tt.deleted_at IS NULL
WHERE t.workspace_id = @workspace_id AND t.language = @language
  AND (t.owner_kind <> 'campaign' OR c.status IN ('draft','approval','approved','scheduled','sending','paused'))
  AND (t.owner_kind <> 'automation_email' OR ae.id IS NOT NULL)
  AND (t.owner_kind <> 'transactional_template' OR tt.id IS NOT NULL)
ORDER BY t.owner_kind, t.updated_at DESC;

-- name: ContentSource :one
SELECT * FROM content_sources
WHERE owner_kind = @owner_kind AND owner_id = @owner_id AND owner_version = @owner_version;

-- name: ContentSourceForUpdate :one
SELECT * FROM content_sources
WHERE owner_kind = @owner_kind AND owner_id = @owner_id AND owner_version = @owner_version
FOR UPDATE;

-- name: ContentSourceUpsert :exec
INSERT INTO content_sources (workspace_id, owner_kind, owner_id, owner_version, language, blocks, subject_hash, preheader_hash, version, next_key, updated_at)
VALUES (@workspace_id, @owner_kind, @owner_id, @owner_version, @language, @blocks, @subject_hash, @preheader_hash, @version, @next_key, @at)
ON CONFLICT (owner_kind, owner_id, owner_version) DO UPDATE
SET language = EXCLUDED.language, blocks = EXCLUDED.blocks, subject_hash = EXCLUDED.subject_hash, preheader_hash = EXCLUDED.preheader_hash,
    version = EXCLUDED.version, next_key = EXCLUDED.next_key, updated_at = EXCLUDED.updated_at;

-- name: ContentSourceSetLanguage :exec
UPDATE content_sources SET language = @language, updated_at = @at
WHERE owner_kind = @owner_kind AND owner_id = @owner_id AND owner_version = @owner_version;

-- name: ContentTranslations :many
-- The language versions of an email, in the order of the workspace's
-- languages ("English, Spanish and German"), with who translated and last
-- saved them.
SELECT t.*,
       coalesce(tu.name, '')::text AS translated_by_name,
       coalesce(uu.name, '')::text AS updated_by_name
FROM content_translations t
LEFT JOIN users tu ON tu.id = t.translated_by
LEFT JOIN users uu ON uu.id = t.updated_by
LEFT JOIN workspace_languages wl ON wl.workspace_id = t.workspace_id AND wl.code = t.language
WHERE t.owner_kind = @owner_kind AND t.owner_id = @owner_id AND t.owner_version = @owner_version
ORDER BY wl.is_default DESC NULLS LAST, wl.added_at NULLS LAST, t.language;

-- name: ContentTranslation :one
SELECT t.*,
       coalesce(tu.name, '')::text AS translated_by_name,
       coalesce(uu.name, '')::text AS updated_by_name
FROM content_translations t
LEFT JOIN users tu ON tu.id = t.translated_by
LEFT JOIN users uu ON uu.id = t.updated_by
WHERE t.owner_kind = @owner_kind AND t.owner_id = @owner_id AND t.owner_version = @owner_version AND t.language = @language;

-- name: ContentTranslationForUpdate :one
SELECT * FROM content_translations
WHERE owner_kind = @owner_kind AND owner_id = @owner_id AND owner_version = @owner_version AND language = @language
FOR UPDATE;

-- name: ContentTranslationInsert :exec
INSERT INTO content_translations (id, workspace_id, owner_kind, owner_id, owner_version, language, status, held, subject, preheader, mjml, html, plain_text,
    ab_test, links, version, content_hash, translated_from, translated_by, translated_at, created_by, created_at, updated_by, updated_at, autosaved_at)
VALUES (@id, @workspace_id, @owner_kind, @owner_id, @owner_version, @language, @status, @held, @subject, @preheader, @mjml, @html, @plain_text,
    @ab_test, @links, @version, @content_hash, @translated_from, sqlc.narg(translated_by), sqlc.narg(translated_at), sqlc.narg(created_by), @created_at,
    sqlc.narg(updated_by), @updated_at, sqlc.narg(autosaved_at));

-- name: ContentTranslationSave :exec
UPDATE content_translations
SET subject = @subject, preheader = @preheader, mjml = @mjml, html = @html, plain_text = @plain_text, ab_test = @ab_test, links = @links,
    version = @version, content_hash = @content_hash, updated_by = sqlc.narg(updated_by), updated_at = @at, autosaved_at = @at
WHERE id = @id;

-- name: ContentTranslationSetLinks :exec
UPDATE content_translations SET links = @links WHERE id = @id;

-- name: ContentTranslationMark :exec
-- Marked translated or up to date: the links point at the current source.
UPDATE content_translations
SET status = @status, links = @links, translated_from = @translated_from, translated_by = sqlc.narg(translated_by), translated_at = @at,
    held = CASE WHEN @status::text = 'translated' THEN held ELSE false END
WHERE id = @id;

-- name: ContentTranslationSetHeld :exec
UPDATE content_translations SET held = @held, updated_at = @at WHERE id = @id;

-- name: ContentTranslationDelete :execrows
DELETE FROM content_translations
WHERE owner_kind = @owner_kind AND owner_id = @owner_id AND owner_version = @owner_version AND language = @language;

-- name: ContentTranslationVersionInsert :exec
INSERT INTO content_translation_versions (translation_id, version, subject, preheader, mjml, html, plain_text, ab_test, links, saved_by, saved_at, label, source, size_bytes, label_text)
VALUES (@translation_id, @version, @subject, @preheader, @mjml, @html, @plain_text, @ab_test, @links, sqlc.narg(saved_by), @saved_at, @label, @source, @size_bytes, sqlc.narg(label_text));

-- name: ContentTranslationVersions :many
SELECT v.version, v.saved_at, v.label, v.label_text, v.source, v.size_bytes, coalesce(u.name, '')::text AS saved_by_name
FROM content_translation_versions v
LEFT JOIN users u ON u.id = v.saved_by
WHERE v.translation_id = @translation_id
ORDER BY v.version DESC
LIMIT @lim;

-- name: ContentTranslationVersion :one
SELECT v.*, coalesce(u.name, '')::text AS saved_by_name
FROM content_translation_versions v
LEFT JOIN users u ON u.id = v.saved_by
WHERE v.translation_id = @translation_id AND v.version = @version;

-- name: ContentTranslationPruneVersions :execrows
-- Keeps the newest autosaves; the first version and named saves stay.
DELETE FROM content_translation_versions v
WHERE v.translation_id = @translation_id AND v.version > 1 AND v.label = '' AND v.source = 'autosave'
  AND v.version NOT IN (SELECT version FROM content_translation_versions WHERE translation_id = @translation_id ORDER BY version DESC LIMIT @keep);

-- name: ContentTranslationsCopy :exec
-- Carries an automation email's or template's language versions to its
-- next version (their versions are immutable), with ids made in Go: one
-- per language.
INSERT INTO content_translations (id, workspace_id, owner_kind, owner_id, owner_version, language, status, held, subject, preheader, mjml, html, plain_text,
    ab_test, links, version, content_hash, translated_from, translated_by, translated_at, created_by, created_at, updated_by, updated_at, autosaved_at)
SELECT (@ids::uuid[])[array_position(@languages::text[], src.language)], src.workspace_id, src.owner_kind, src.owner_id, @to_version, src.language,
    src.status, src.held, src.subject, src.preheader,
    src.mjml, src.html, src.plain_text, src.ab_test, src.links, src.version, src.content_hash, src.translated_from, src.translated_by, src.translated_at,
    src.created_by, src.created_at, src.updated_by, src.updated_at, src.autosaved_at
FROM content_translations src
WHERE src.owner_kind = @owner_kind AND src.owner_id = @owner_id AND src.owner_version = @from_version AND src.language = ANY(@languages::text[])
ON CONFLICT (owner_kind, owner_id, owner_version, language) DO NOTHING;

-- name: ContentSourceCopy :exec
INSERT INTO content_sources (workspace_id, owner_kind, owner_id, owner_version, language, blocks, subject_hash, preheader_hash, version, next_key, updated_at)
SELECT src.workspace_id, src.owner_kind, src.owner_id, @to_version, src.language, src.blocks, src.subject_hash, src.preheader_hash, src.version,
    src.next_key, src.updated_at
FROM content_sources src
WHERE src.owner_kind = @owner_kind AND src.owner_id = @owner_id AND src.owner_version = @from_version
ON CONFLICT (owner_kind, owner_id, owner_version) DO NOTHING;

-- name: LanguagesCampaignSource :one
-- What a campaign's source is: its content and who saved it last.
SELECT c.id, c.workspace_id, c.name, c.status, c.subject, c.preheader, c.mjml, c.html, c.plain_text, c.ab_test, c.version,
       c.content_hash, c.audience, c.updated_at, c.autosaved_at, c.started_at,
       coalesce(u.name, '')::text AS updated_by_name
FROM campaigns c
LEFT JOIN users u ON u.id = c.updated_by
WHERE c.workspace_id = @workspace_id AND c.id = @id AND c.deleted_at IS NULL;

-- name: LanguagesCampaignVersionAt :one
-- The source version saved last at or before a time (who changed a block).
SELECT v.version, v.saved_at, coalesce(u.name, '')::text AS saved_by_name
FROM campaign_versions v
LEFT JOIN users u ON u.id = v.saved_by
WHERE v.campaign_id = @campaign_id AND v.version = @version;

-- name: LanguagesMessageResults :many
-- Results of a sent campaign per language version and why readers got it.
WITH unsub AS (
    SELECT DISTINCT e.message_id FROM delivery_events e
    WHERE e.campaign_id = @campaign_id AND e.type = 'unsubscribe' AND e.message_id IS NOT NULL
)
SELECT m.language, m.language_reason, m.contact_language,
       count(*) FILTER (WHERE m.sent_at IS NOT NULL)::bigint AS sent,
       count(*) FILTER (WHERE m.delivered_at IS NOT NULL OR m.status = 'complained')::bigint AS delivered,
       count(*) FILTER (WHERE m.human_opened_at IS NOT NULL)::bigint AS human_opens,
       count(*) FILTER (WHERE m.clicked_at IS NOT NULL)::bigint AS clicks,
       count(*) FILTER (WHERE m.status = 'complained')::bigint AS complaints,
       count(*) FILTER (WHERE unsub.message_id IS NOT NULL)::bigint AS unsubscribes
FROM messages m
LEFT JOIN unsub ON unsub.message_id = m.id
WHERE m.campaign_id = @campaign_id AND m.status <> 'cancelled'
GROUP BY 1, 2, 3;

-- name: LanguagesLatestDraftFor :one
-- The newest unsent campaign with the same audience (the report's "Add a
-- Vietnamese version to …").
SELECT c.id, c.name FROM campaigns c
WHERE c.workspace_id = @workspace_id AND c.id <> @except_id AND c.deleted_at IS NULL
  AND c.status IN ('draft','approval','approved','scheduled') AND c.audience = @audience
ORDER BY c.created_at DESC
LIMIT 1;

-- name: LanguagesCampaignSendContent :one
SELECT content FROM campaign_sends WHERE campaign_id = @campaign_id;

-- name: LanguagesAutomationEmailSource :one
-- An automation email version as the source of its language versions.
SELECT e.id, e.name, e.latest_version, v.version, v.subject, v.preheader, v.mjml, v.html, v.created_at,
       coalesce(u.name, '')::text AS created_by_name
FROM automation_emails e
JOIN automation_email_versions v ON v.email_id = e.id
LEFT JOIN users u ON u.id = v.created_by
WHERE e.workspace_id = @workspace_id AND e.id = @id AND v.version = @version AND e.deleted_at IS NULL;

-- name: LanguagesAutomationEmailLock :one
-- Locks an automation email to give it its next version.
SELECT latest_version FROM automation_emails
WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL
FOR UPDATE;

-- name: LanguagesAutomationEmailCopyVersion :exec
-- The email's next version: a copy of one of its versions, its translations
-- to be written.
INSERT INTO automation_email_versions (email_id, version, subject, preheader, mjml, html, plain_text, created_by)
SELECT src.email_id, @to_version, src.subject, src.preheader, src.mjml, src.html, src.plain_text, sqlc.narg(created_by)
FROM automation_email_versions src
WHERE src.email_id = @email_id AND src.version = @from_version;

-- name: LanguagesAutomationEmailSetLatest :exec
UPDATE automation_emails SET latest_version = @version, updated_at = now() WHERE id = @id;

-- name: LanguagesPreviewContacts :many
-- "Preview as" in the composer: one active contact per contact language
-- (its base code), from the given lists when there are any: people with a
-- first name first, then the one who opened last, then by email. Each
-- language's contact is found with aggregates, so the audience isn't sorted.
WITH audience AS MATERIALIZED (
  SELECT c.id, lower(split_part(replace(c.language, '_', '-'), '-', 1)) AS base,
         c.first_name <> '' AS named, c.last_open_at, c.email
  FROM contacts c
  WHERE c.workspace_id = @workspace_id AND c.erased_at IS NULL AND c.status = 'active' AND c.language <> ''
    AND (cardinality(@lists::uuid[]) = 0 OR EXISTS (
      SELECT 1 FROM list_memberships m
      WHERE m.contact_id = c.id AND m.status = 'subscribed'
        AND m.list_id = ANY(ARRAY(SELECT ll.id FROM lists ll WHERE ll.id = ANY(@lists::uuid[]) AND ll.deleted_at IS NULL))))
    AND (cardinality(@scope_lists::uuid[]) = 0 OR EXISTS (SELECT 1 FROM list_memberships sm WHERE sm.contact_id = c.id AND sm.list_id = ANY(@scope_lists::uuid[])))
), named AS (
  -- Whether a language has anyone with a first name.
  SELECT base, bool_or(named) AS named FROM audience GROUP BY base
), latest AS (
  -- The last open among them (NULL when none of them opened).
  SELECT a.base, a.named, max(a.last_open_at) AS last_open_at
  FROM audience a JOIN named n ON n.base = a.base AND n.named = a.named
  GROUP BY a.base, a.named
)
SELECT DISTINCT ON (a.base) c.id, c.email, c.first_name, c.last_name, c.language
FROM audience a
JOIN latest l ON l.base = a.base AND l.named = a.named AND a.last_open_at IS NOT DISTINCT FROM l.last_open_at
JOIN contacts c ON c.id = a.id
ORDER BY a.base, a.email
LIMIT 200;

-- name: LanguagesTemplateSource :one
-- A transactional template's draft as the source of its language versions,
-- by its id or its key.
SELECT t.id, t.workspace_id, t.key, t.name, t.subject, t.preheader, t.mjml, t.html, t.plain_text, t.draft_revision, t.draft_saved_at,
       t.archived_at, t.live_version, coalesce(u.name, '')::text AS saved_by_name
FROM transactional_templates t
LEFT JOIN users u ON u.id = t.draft_saved_by
WHERE t.workspace_id = @workspace_id AND (t.id = @id OR (@key::text <> '' AND t.key = @key::text)) AND t.deleted_at IS NULL;

-- name: LanguagesContactLanguage :one
-- A contact's language code ("" for none), for a transactional send.
SELECT language FROM contacts WHERE workspace_id = @workspace_id AND id = @id AND erased_at IS NULL;

-- name: LanguagesSetMessageLanguage :exec
-- The language version a transactional message was sent in, and why.
UPDATE messages SET language = @language, contact_language = @contact_language, language_reason = @language_reason
WHERE id = @id;
