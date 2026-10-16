-- Content languages of forms, public pages and contacts (design expansion
-- M17, workstream X08): a contact's language and where it came from, page
-- and form copy per language, and ReputeCast's built-in page labels.

-- name: ContactSetLanguage :exec
-- Sets a contact's language and records where it came from.
UPDATE contacts
SET language = @language, language_source = @source, language_source_name = @source_name, language_browser = @browser,
    language_set_at = @at::timestamptz, language_set_by = sqlc.narg(set_by), updated_at = @at::timestamptz
WHERE id = @id;

-- name: ContactSetLanguageSource :exec
-- Records where a contact's language came from, after another query set it.
UPDATE contacts
SET language_source = @source, language_source_name = @source_name, language_browser = @browser,
    language_set_at = @at::timestamptz, language_set_by = sqlc.narg(set_by)
WHERE id = @id;

-- name: ContactFillLanguage :execrows
-- A signup form sets the language from the browser's first language when
-- the contact has none. It never overwrites one.
UPDATE contacts
SET language = @language, language_source = 'form', language_source_name = @source_name, language_browser = true,
    language_set_at = @at::timestamptz, language_set_by = NULL
WHERE id = @id AND language = '';

-- name: ContactLanguageSetBy :one
SELECT coalesce(u.name, '')::text AS name FROM users u WHERE u.id = @id;

-- name: PageCopyList :many
-- The page copy a workspace wrote, per language, with who wrote it.
SELECT pc.language, pc.pages, pc.pages_by, pc.pages_at, pc.labels, pc.labels_by, pc.labels_at,
       coalesce(pu.name, '')::text AS pages_by_name, coalesce(lu.name, '')::text AS labels_by_name
FROM page_copy pc
LEFT JOIN users pu ON pu.id = pc.pages_by
LEFT JOIN users lu ON lu.id = pc.labels_by
WHERE pc.workspace_id = @workspace_id
ORDER BY pc.language;

-- name: PageCopySetPages :exec
INSERT INTO page_copy (workspace_id, language, pages, pages_by, pages_at)
VALUES (@workspace_id, @language, @pages, sqlc.narg(by), @at::timestamptz)
ON CONFLICT (workspace_id, language) DO UPDATE
SET pages = EXCLUDED.pages, pages_by = EXCLUDED.pages_by, pages_at = EXCLUDED.pages_at;

-- name: PageCopySetLabels :exec
INSERT INTO page_copy (workspace_id, language, labels, labels_by, labels_at)
VALUES (@workspace_id, @language, @labels, sqlc.narg(by), @at::timestamptz)
ON CONFLICT (workspace_id, language) DO UPDATE
SET labels = EXCLUDED.labels, labels_by = EXCLUDED.labels_by, labels_at = EXCLUDED.labels_at;

-- name: PageCopyLock :exec
-- Serializes changes to a workspace's page copy (the statuses are computed
-- from all of it).
SELECT pg_advisory_xact_lock(hashtextextended('page_copy:' || (@workspace_id::uuid)::text, 0));

-- name: FormCopyList :many
-- A form's copy in its other languages, with who wrote it last.
SELECT fc.language, fc.content, fc.consent_text, fc.consent_version, fc.consent_for, fc.updated_by, fc.updated_at,
       coalesce(u.name, '')::text AS updated_by_name
FROM form_copy fc
LEFT JOIN users u ON u.id = fc.updated_by
WHERE fc.form_id = @form_id
ORDER BY fc.language;

-- name: FormCopyListForms :many
-- The copy of several forms (the overview's statuses).
SELECT fc.form_id, fc.language, fc.content, fc.consent_text, fc.consent_version, fc.consent_for
FROM form_copy fc
WHERE fc.form_id = ANY (@form_ids::uuid[]);

-- name: FormCopyUpsert :exec
INSERT INTO form_copy (form_id, language, content, consent_text, consent_version, consent_for, updated_by, updated_at)
VALUES (@form_id, @language, @content, @consent_text, @consent_version, @consent_for, sqlc.narg(by), @at::timestamptz)
ON CONFLICT (form_id, language) DO UPDATE
SET content = EXCLUDED.content, consent_text = EXCLUDED.consent_text, consent_version = EXCLUDED.consent_version,
    consent_for = EXCLUDED.consent_for, updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at;

-- name: FormCopyConfirmConsent :execrows
-- "Spanish is still right": the translation now stands for the form's
-- current consent text.
UPDATE form_copy SET consent_for = @consent_for, updated_by = sqlc.narg(by), updated_at = @at::timestamptz
WHERE form_id = @form_id AND language = @language AND consent_text <> '' AND consent_for < @consent_for;

-- name: FormSetLanguage :one
UPDATE forms SET language_mode = @language_mode, fixed_language = @fixed_language, updated_by = sqlc.narg(by), updated_at = @at::timestamptz
WHERE workspace_id = @workspace_id AND id = @id
RETURNING *;

-- name: FormSubmissionLatestLanguage :one
-- The language the contact's latest signup form was shown in (the
-- confirmation page follows it).
SELECT language FROM form_submissions
WHERE contact_id = @contact_id AND language <> ''
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: FormConsentEvidence :one
-- The latest signup through a form whose consent was given in a language
-- (the builder's "Consent evidence" example).
SELECT c.id AS contact_id, coalesce(nullif(trim(concat_ws(' ', c.first_name, c.last_name)), ''), c.email)::text AS name,
       ce.occurred_at, ce.consent_version
FROM consent_events ce
JOIN contacts c ON c.id = ce.contact_id AND c.workspace_id = @workspace_id
WHERE ce.source = @source AND ce.language = @language AND ce.consent_text <> '' AND ce.action IN ('subscribed', 'resubscribed')
ORDER BY ce.occurred_at DESC, ce.id DESC
LIMIT 1;

-- name: FormSetCopy :one
-- Saves a form's own text from its language view: the copy, the field
-- labels and the consent text, whose version moves when the words change.
UPDATE forms
SET content = @content, fields = @fields, consent_text = @consent_text, consent_version = @consent_version,
    consent_changed_by = CASE WHEN @consent_version::int <> consent_version THEN sqlc.narg(by)::uuid ELSE consent_changed_by END,
    consent_changed_at = CASE WHEN @consent_version::int <> consent_version THEN @at::timestamptz ELSE consent_changed_at END,
    updated_by = sqlc.narg(by), updated_at = @at::timestamptz
WHERE workspace_id = @workspace_id AND id = @id
RETURNING *;

-- name: PageCopyUserName :one
SELECT coalesce(name, '')::text FROM users WHERE id = @id;
