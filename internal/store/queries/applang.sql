-- App languages: a member's language and region format, the instance
-- defaults and an invitation's language (internal/applang, internal/i18n).

-- name: AppLangSetUser :one
UPDATE users
SET language = sqlc.narg(language), region_format = sqlc.narg(region_format),
    hour_cycle = sqlc.narg(hour_cycle), week_start = sqlc.narg(week_start), updated_at = now()
WHERE id = @id
RETURNING *;

-- name: AppLangSetPrompted :exec
UPDATE users SET language_prompted_at = coalesce(language_prompted_at, now()) WHERE id = @id;

-- name: AppLangInstanceDefaults :one
SELECT default_language, default_region_format FROM instance_settings WHERE id;

-- name: AppLangSetInstanceDefaults :exec
UPDATE instance_settings SET default_language = @default_language, default_region_format = @default_region_format WHERE id;

-- name: AppLangSetInvitationLanguage :exec
UPDATE invitations SET language = sqlc.narg(language) WHERE id = @id;

-- name: AppLangInvitationLanguage :one
SELECT coalesce(language, '')::text FROM invitations WHERE id = @id;

-- The language a member reads their email in: their choices, and the
-- instance defaults for what they haven't chosen.
-- name: AppLangForUser :one
SELECT coalesce(u.language, '')::text AS language, coalesce(u.region_format, '')::text AS region_format,
       s.default_language, s.default_region_format
FROM users u CROSS JOIN instance_settings s
WHERE u.id = @id AND s.id;

-- name: AppLangForEmail :one
SELECT coalesce(u.language, '')::text AS language, coalesce(u.region_format, '')::text AS region_format,
       s.default_language, s.default_region_format
FROM users u CROSS JOIN instance_settings s
WHERE u.email = lower(@email::text) AND s.id;
