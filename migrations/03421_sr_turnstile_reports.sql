-- +goose Up
-- Errors the Turnstile widget showed visitors on a challenge page, as their
-- browsers reported them (a site key Cloudflare refuses, a domain it
-- doesn't allow): the last code, when, and how many reports since the keys
-- were saved, for the publisher to see beside the keys. Any browser can
-- send a report, so a report never turns Turnstile off: only Cloudflare
-- refusing the secret, asked from the server, does.
ALTER TABLE public_page_settings
    ADD COLUMN turnstile_reported_code  text    NOT NULL DEFAULT '',
    ADD COLUMN turnstile_reported_at    timestamptz,
    ADD COLUMN turnstile_reported_count integer NOT NULL DEFAULT 0;

-- Turnstile that a report turned off comes back on, the report kept to show.
UPDATE public_page_settings
SET turnstile_status = 'ok', turnstile_error = '', texts = texts - 'turnstile_error',
    turnstile_reported_code = coalesce(substring(turnstile_error FROM '[0-9]{3,12}'), ''),
    turnstile_reported_at = turnstile_checked_at, turnstile_reported_count = 1
WHERE turnstile_status = 'error' AND turnstile_site_key <> '' AND turnstile_secret_enc IS NOT NULL
  AND (texts->'turnstile_error'->>'id' IN ('forms.turnstile.siteRejected', 'forms.turnstile.siteUnknown', 'forms.turnstile.widget', 'forms.turnstile.reported')
       OR (texts->'turnstile_error' IS NULL
           AND (turnstile_error LIKE 'Cloudflare rejected the site key (error %' OR turnstile_error LIKE 'Cloudflare doesn''t know this site key (error %'
                OR turnstile_error LIKE 'The Turnstile widget couldn''t load (error %' OR turnstile_error LIKE 'Turnstile reported error %')));

-- +goose Down
-- Turnstile turned back on stays on: whether a report had turned it off
-- can't be told apart once it has run with the reports kept beside it.
ALTER TABLE public_page_settings
    DROP COLUMN turnstile_reported_count,
    DROP COLUMN turnstile_reported_at,
    DROP COLUMN turnstile_reported_code;
