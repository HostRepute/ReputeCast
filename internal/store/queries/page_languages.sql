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
