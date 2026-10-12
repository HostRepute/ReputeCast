-- name: GetInstanceSettings :one
SELECT * FROM instance_settings WHERE id;

-- name: CompleteInstanceSetup :one
UPDATE instance_settings
SET name = @name, public_url = @public_url, default_timezone = @default_timezone,
    update_check = @update_check, setup_completed_at = now(), updated_at = now()
WHERE id
RETURNING *;

-- name: UpdateInstanceBasics :one
UPDATE instance_settings
SET name = @name, logo_svg = @logo_svg, public_url = @public_url,
    default_timezone = @default_timezone, update_check = @update_check, updated_at = now()
WHERE id
RETURNING *;

-- name: UpdateAuthPolicy :one
UPDATE instance_settings
SET allowed_methods = @allowed_methods, password_min_length = @password_min_length,
    breached_check = @breached_check, twofa_enforcement = @twofa_enforcement,
    twofa_grace_days = @twofa_grace_days, session_lifetime = @session_lifetime,
    session_idle = @session_idle, lockout_attempts = @lockout_attempts,
    lockout_minutes = @lockout_minutes, jit_domain_allowlist = @jit_domain_allowlist,
    updated_at = now()
WHERE id
RETURNING *;

-- name: UpdateSocialProviders :one
UPDATE instance_settings
SET social_providers = @social_providers, social_secrets_enc = @social_secrets_enc, updated_at = now()
WHERE id
RETURNING *;

-- name: SetLatestRelease :exec
UPDATE instance_settings SET latest_release = @latest_release, latest_release_at = now() WHERE id;
