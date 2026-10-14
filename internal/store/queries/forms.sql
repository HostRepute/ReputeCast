-- Signup forms and public page settings (spec 5.12, 7.12).

-- name: FormsList :many
SELECT * FROM forms WHERE workspace_id = @workspace_id ORDER BY created_at, id;

-- name: FormGet :one
SELECT * FROM forms WHERE workspace_id = @workspace_id AND id = @id;

-- name: FormByPublicID :one
SELECT * FROM forms WHERE workspace_id = @workspace_id AND public_id = @public_id;

-- name: FormBySlug :one
SELECT * FROM forms WHERE workspace_id = @workspace_id AND slug = @slug;

-- name: FormSlugTaken :one
SELECT EXISTS (SELECT 1 FROM forms WHERE workspace_id = @workspace_id AND slug = @slug AND id <> @id)::boolean;

-- name: FormPublicIDTaken :one
SELECT EXISTS (SELECT 1 FROM forms WHERE workspace_id = @workspace_id AND public_id = @public_id)::boolean;

-- name: FormInsert :one
INSERT INTO forms (id, workspace_id, name, kind, slug, public_id, list_ids, default_list_ids, list_choice, double_opt_in,
                   fields, content, spam_protection, consent_text, consent_version, allowed_origins, after_submit, redirect_url,
                   status, published_at, created_by, updated_by, created_at, updated_at)
VALUES (@id, @workspace_id, @name, @kind, @slug, @public_id, @list_ids, @default_list_ids, @list_choice, @double_opt_in,
        @fields, @content, @spam_protection, @consent_text, @consent_version, @allowed_origins, @after_submit, @redirect_url,
        @status, @published_at, @created_by, @created_by, @at, @at)
RETURNING *;

-- name: FormUpdate :one
UPDATE forms
SET name = @name, kind = @kind, list_ids = @list_ids, default_list_ids = @default_list_ids, list_choice = @list_choice,
    double_opt_in = @double_opt_in, fields = @fields, content = @content, spam_protection = @spam_protection,
    consent_text = @consent_text, consent_version = @consent_version, allowed_origins = @allowed_origins,
    after_submit = @after_submit, redirect_url = @redirect_url, status = @status,
    published_at = CASE WHEN @status::text = 'live' THEN @at::timestamptz ELSE published_at END,
    updated_by = @updated_by, updated_at = @at
WHERE workspace_id = @workspace_id AND id = @id
RETURNING *;

-- name: FormSetStatus :one
UPDATE forms SET status = @status, updated_by = @updated_by, updated_at = @at
WHERE workspace_id = @workspace_id AND id = @id
RETURNING *;

-- name: FormRename :one
UPDATE forms SET name = @name, slug = @slug, updated_by = @updated_by, updated_at = @at
WHERE workspace_id = @workspace_id AND id = @id
RETURNING *;

-- name: FormDelete :execrows
DELETE FROM forms WHERE workspace_id = @workspace_id AND id = @id;

-- name: FormAllowOrigin :one
UPDATE forms SET allowed_origins = array_append(allowed_origins, @origin::text), updated_by = @updated_by, updated_at = @at
WHERE workspace_id = @workspace_id AND id = @id AND NOT (@origin::text = ANY (allowed_origins))
RETURNING *;

-- name: FormStatsSince :many
-- Per form: views, submissions, confirmations of those submissions and spam
-- blocked since a day (inclusive).
SELECT form_id,
       coalesce(sum(views), 0)::bigint AS views,
       coalesce(sum(submissions), 0)::bigint AS submissions,
       coalesce(sum(confirmed), 0)::bigint AS confirmed,
       coalesce(sum(blocked), 0)::bigint AS blocked
FROM form_daily
WHERE form_id = ANY (@form_ids::uuid[]) AND day >= @since::date
GROUP BY form_id;

-- name: FormBumpDaily :exec
INSERT INTO form_daily (form_id, day, views, submissions, confirmed, blocked)
VALUES (@form_id, @day, @views, @submissions, @confirmed, @blocked)
ON CONFLICT (form_id, day) DO UPDATE
SET views = form_daily.views + excluded.views,
    submissions = form_daily.submissions + excluded.submissions,
    confirmed = form_daily.confirmed + excluded.confirmed,
    blocked = form_daily.blocked + excluded.blocked;

-- name: FormBumpTotals :exec
UPDATE forms
SET views = views + @views, submissions = submissions + @submissions, confirmed = confirmed + @confirmed, blocked = blocked + @blocked
WHERE id = @id;

-- name: FormRecordRejection :exec
INSERT INTO form_rejections (form_id, day, reason, origin, count, last_at)
VALUES (@form_id, @day, @reason, @origin, 1, @at)
ON CONFLICT (form_id, day, reason, origin) DO UPDATE
SET count = form_rejections.count + 1, last_at = excluded.last_at;

-- name: FormBlockedOrigins :many
-- Origins rejected today that nobody chose to keep blocked yet.
SELECT origin, count, last_at FROM form_rejections
WHERE form_id = @form_id AND day = @day AND reason = 'origin' AND origin <> '' AND dismissed_at IS NULL
ORDER BY count DESC, origin
LIMIT 5;

-- name: FormDismissOrigin :execrows
UPDATE form_rejections SET dismissed_at = @at
WHERE form_id = @form_id AND reason = 'origin' AND origin = @origin AND dismissed_at IS NULL;

-- name: FormSubmissionInsert :exec
INSERT INTO form_submissions (id, workspace_id, form_id, contact_id, list_ids, double_opt_in, created_at, confirmed_at)
VALUES (@id, @workspace_id, @form_id, @contact_id, @list_ids, @double_opt_in, @at, @confirmed_at);

-- name: FormSubmissionsConfirm :many
-- Marks a contact's waiting submissions confirmed; returns the form and the
-- day each was submitted, so the day's confirmed count moves.
UPDATE form_submissions
SET confirmed_at = @at
WHERE contact_id = @contact_id AND double_opt_in AND confirmed_at IS NULL
RETURNING id, form_id, created_at;

-- name: FormSubmissionsToSend :many
-- Double opt-in confirmations that couldn't go out yet (no System provider,
-- a provider error): retried by the confirmations job for three days.
SELECT fs.id, fs.workspace_id, fs.form_id, fs.contact_id, fs.send_attempts
FROM form_submissions fs
WHERE fs.double_opt_in AND fs.confirmed_at IS NULL AND fs.confirm_sent_at IS NULL
  AND fs.created_at > @since AND fs.send_attempts < 20
ORDER BY fs.created_at
LIMIT @lim;

-- name: FormSubmissionsToRemind :many
-- Confirmations sent more than 24 hours ago without a click: one reminder.
SELECT fs.id, fs.workspace_id, fs.form_id, fs.contact_id
FROM form_submissions fs
WHERE fs.double_opt_in AND fs.confirmed_at IS NULL AND fs.reminded_at IS NULL
  AND fs.confirm_sent_at IS NOT NULL AND fs.confirm_sent_at < @before AND fs.created_at > @since
ORDER BY fs.created_at
LIMIT @lim;

-- name: FormSubmissionMarkSent :exec
UPDATE form_submissions SET confirm_sent_at = @at, send_attempts = send_attempts + 1 WHERE id = @id;

-- name: FormSubmissionMarkAttempt :exec
UPDATE form_submissions SET send_attempts = send_attempts + 1 WHERE id = @id;

-- name: FormSubmissionMarkReminded :exec
UPDATE form_submissions SET reminded_at = @at WHERE id = @id;

-- name: FormSubmissionSettle :exec
-- A contact confirmed some other way (re-permission, the team): nothing
-- to send or remind any more.
UPDATE form_submissions SET confirmed_at = @at WHERE id = @id AND confirmed_at IS NULL;

-- name: FormWaitingSubmissions :many
-- Double opt-in signups of a form whose confirmation email hasn't gone out
-- (nothing could send System mail, or the provider refused it), newest
-- first: the builder offers their links to send by hand.
SELECT id, contact_id, list_ids, created_at, send_attempts FROM form_submissions
WHERE form_id = @form_id AND double_opt_in AND confirmed_at IS NULL AND confirm_sent_at IS NULL AND created_at > @since
ORDER BY created_at DESC, id DESC
LIMIT @lim;

-- name: FormWaitingCount :one
SELECT count(*)::bigint FROM form_submissions
WHERE form_id = @form_id AND double_opt_in AND confirmed_at IS NULL AND confirm_sent_at IS NULL AND created_at > @since;

-- name: FormSubmissionByID :one
SELECT * FROM form_submissions WHERE id = @id AND workspace_id = @workspace_id AND form_id = @form_id;

-- name: FormLastSubmissionForContact :one
SELECT fs.form_id, fs.list_ids FROM form_submissions fs
WHERE fs.contact_id = @contact_id
ORDER BY fs.created_at DESC
LIMIT 1;

-- name: FormChallengeInsert :exec
INSERT INTO form_challenges (id, workspace_id, form_id, payload_enc, origin, created_at, expires_at)
VALUES (@id, @workspace_id, @form_id, @payload_enc, @origin, @at, @expires_at);

-- name: FormChallengeTake :one
DELETE FROM form_challenges WHERE id = @id AND workspace_id = @workspace_id AND expires_at > @at
RETURNING *;

-- name: FormChallengeLive :one
-- A held submission still waiting for its Turnstile check.
SELECT EXISTS (SELECT 1 FROM form_challenges WHERE id = @id AND workspace_id = @workspace_id AND expires_at > @at)::boolean;

-- name: FormChallengesPurge :execrows
DELETE FROM form_challenges WHERE expires_at <= @at;

-- name: PublicSettingsEnsure :exec
INSERT INTO public_page_settings (workspace_id) VALUES (@workspace_id) ON CONFLICT (workspace_id) DO NOTHING;

-- name: PublicSettingsGet :one
SELECT * FROM public_page_settings WHERE workspace_id = @workspace_id;

-- name: PublicSettingsByDomain :one
SELECT s.workspace_id, w.slug
FROM public_page_settings s
JOIN workspaces w ON w.id = s.workspace_id AND w.deleted_at IS NULL
WHERE s.custom_domain = @domain AND s.domain_status = 'verified';

-- name: PublicSettingsSetPages :exec
UPDATE public_page_settings SET pages = @pages, updated_by = @updated_by, updated_at = @at WHERE workspace_id = @workspace_id;

-- name: PublicSettingsTouch :exec
UPDATE public_page_settings SET updated_by = @updated_by, updated_at = @at WHERE workspace_id = @workspace_id;

-- name: PublicSettingsSetHostedForm :exec
UPDATE public_page_settings SET hosted_form_id = @hosted_form_id, updated_by = @updated_by, updated_at = @at WHERE workspace_id = @workspace_id;

-- name: PublicSettingsSetDomain :exec
UPDATE public_page_settings
SET custom_domain = @custom_domain, domain_status = @domain_status, domain_detail = @domain_detail,
    domain_checked_at = @domain_checked_at, domain_verified_at = @domain_verified_at, updated_by = @updated_by, updated_at = @at
WHERE workspace_id = @workspace_id;

-- name: PublicSettingsDomainTaken :one
SELECT EXISTS (SELECT 1 FROM public_page_settings WHERE custom_domain = @domain AND workspace_id <> @workspace_id)::boolean;

-- name: PublicSettingsDomainsToCheck :many
-- Domains waiting for DNS (every run) and verified ones (once a day).
SELECT workspace_id, custom_domain::text AS custom_domain, domain_status
FROM public_page_settings
WHERE custom_domain IS NOT NULL
  AND (domain_status IN ('pending','failed') OR domain_checked_at IS NULL OR domain_checked_at < @before);

-- name: PublicSettingsSetTurnstile :exec
UPDATE public_page_settings
SET turnstile_site_key = @site_key, turnstile_secret_enc = @secret_enc, turnstile_status = @status, turnstile_error = @error,
    turnstile_checked_at = @checked_at, updated_by = @updated_by, updated_at = @at
WHERE workspace_id = @workspace_id;

-- name: PublicSettingsTurnstileError :exec
-- The widget on a challenge page reported an error (wrong site key, domain
-- not allowed): spam protection falls back until someone fixes the key.
UPDATE public_page_settings SET turnstile_status = 'error', turnstile_error = @error, turnstile_checked_at = @at
WHERE workspace_id = @workspace_id AND turnstile_site_key <> '';

-- name: PublicSettingsSetLogo :exec
UPDATE public_page_settings SET logo = @logo, logo_type = @logo_type, updated_by = @updated_by, updated_at = @at
WHERE workspace_id = @workspace_id;

-- name: PublicSettingsLogo :one
SELECT logo, logo_type, updated_at FROM public_page_settings WHERE workspace_id = @workspace_id AND logo IS NOT NULL;

-- name: FormsWorkspaceBranding :one
SELECT branding FROM workspaces WHERE id = @id;

-- name: FormsUpdateBranding :one
-- Merges the public-page keys into workspaces.branding; keys other screens
-- own (the postal address of the email footer) stay as they are.
UPDATE workspaces SET branding = branding || @patch::jsonb, updated_at = now()
WHERE id = @id
RETURNING branding;

-- name: FormsRemoveBrandingKeys :exec
UPDATE workspaces SET branding = branding - @keys::text[], updated_at = now() WHERE id = @id;

-- name: FormsDefaultSender :one
SELECT s.email, s.name FROM workspaces w JOIN senders s ON s.id = w.default_sender_id WHERE w.id = @workspace_id;
