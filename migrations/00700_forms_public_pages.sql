-- +goose Up
-- W13 · Signup forms and public pages (spec 5.12, 7.12): what the form
-- builder needs beyond the base forms table, the per-workspace settings of
-- the public pages (hosted subscribe page, preference center, unsubscribe
-- page, archive, custom domain, Cloudflare Turnstile, logo), daily form
-- numbers, rejected submissions, accepted submissions waiting for double
-- opt-in, submissions held for a Turnstile check, and the public archive.
-- Contacts, list memberships, consent, preferences and suppressions stay
-- in the audience tables and are written through internal/audience.

-- Forms ------------------------------------------------------------------
ALTER TABLE forms
    -- The form's id in embed code, script URLs and the plain HTML form ("frm_8c2e1a").
    ADD COLUMN public_id        text        NOT NULL DEFAULT ('frm_' || substr(md5(gen_random_uuid()::text), 1, 6)),
    -- Offered lists (list_ids) that start checked, and whether visitors pick
    -- lists at all (off: they join every offered list).
    ADD COLUMN default_list_ids uuid[]      NOT NULL DEFAULT '{}',
    ADD COLUMN list_choice      boolean     NOT NULL DEFAULT true,
    -- The sentence next to the consent checkbox; the exact text and its
    -- version go with every consent event the form records.
    ADD COLUMN consent_text     text        NOT NULL DEFAULT '',
    ADD COLUMN consent_version  integer     NOT NULL DEFAULT 1,
    -- Hosts allowed to submit the form (embeds) and to frame its hosted page.
    ADD COLUMN allowed_origins  text[]      NOT NULL DEFAULT '{}',
    ADD COLUMN after_submit     text        NOT NULL DEFAULT 'message' CHECK (after_submit IN ('message','redirect')),
    ADD COLUMN redirect_url     text        NOT NULL DEFAULT '',
    ADD COLUMN published_at     timestamptz,
    ADD COLUMN updated_by       uuid        REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE forms ADD CONSTRAINT forms_public_id_check CHECK (public_id ~ '^frm_[a-z0-9]{6,16}$');
CREATE UNIQUE INDEX forms_public_id_key ON forms (workspace_id, public_id);

-- Public page settings ----------------------------------------------------
-- Branding (publisher name, logo metadata, colours, font, footer) lives in
-- workspaces.branding; everything else about the public pages is here.
CREATE TABLE public_page_settings (
    workspace_id         uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    -- The form behind the hosted page (/subscribe); null uses the built-in form.
    hosted_form_id       uuid        REFERENCES forms(id) ON DELETE SET NULL,
    -- Options and per-list copy of each page:
    -- {"subscribe":{…},"confirm":{…},"preferences":{…},"unsubscribe":{…},"archive":{…}}
    pages                jsonb       NOT NULL DEFAULT '{}',
    -- A domain that serves the pages at its root (news.acme.io/subscribe),
    -- verified by DNS before ReputeCast answers for it.
    custom_domain        text        CHECK (custom_domain = lower(custom_domain)),
    domain_status        text        NOT NULL DEFAULT 'none' CHECK (domain_status IN ('none','pending','verified','failed')),
    domain_detail        text        NOT NULL DEFAULT '',
    domain_checked_at    timestamptz,
    domain_verified_at   timestamptz,
    -- Cloudflare Turnstile: the site key is public; the secret is write-only,
    -- sealed with the instance key ("turnstile:<workspace>:secret").
    turnstile_site_key   text        NOT NULL DEFAULT '',
    turnstile_secret_enc bytea,
    turnstile_status     text        NOT NULL DEFAULT 'off' CHECK (turnstile_status IN ('off','ok','error')),
    turnstile_error      text        NOT NULL DEFAULT '',
    turnstile_checked_at timestamptz,
    -- The uploaded logo (PNG, JPEG, GIF, WebP or SVG, 256 KB at most).
    logo                 bytea,
    logo_type            text        NOT NULL DEFAULT '',
    updated_by           uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at           timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX public_page_settings_domain_key ON public_page_settings (custom_domain) WHERE custom_domain IS NOT NULL;

-- Form numbers ------------------------------------------------------------
-- Per form and day of submission: views, accepted submissions, how many of
-- that day's submissions are confirmed (so pending = submissions - confirmed
-- for any window) and spam blocked.
CREATE TABLE form_daily (
    form_id     uuid   NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    day         date   NOT NULL,
    views       bigint NOT NULL DEFAULT 0,
    submissions bigint NOT NULL DEFAULT 0,
    confirmed   bigint NOT NULL DEFAULT 0,
    blocked     bigint NOT NULL DEFAULT 0,
    PRIMARY KEY (form_id, day)
);

-- Rejected submissions by day, reason and origin: the "blocked origin"
-- alert in the builder and the spam totals.
CREATE TABLE form_rejections (
    form_id      uuid        NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    day          date        NOT NULL,
    reason       text        NOT NULL CHECK (reason IN ('origin','honeypot','timing','turnstile','disposable','role','rate_limit')),
    origin       text        NOT NULL DEFAULT '',
    count        bigint      NOT NULL DEFAULT 0,
    last_at      timestamptz NOT NULL,
    dismissed_at timestamptz,
    PRIMARY KEY (form_id, day, reason, origin)
);

-- Accepted submissions: who signed up through which form. Double opt-in
-- submissions wait here for the confirmation click; the confirmation email
-- is (re)sent and reminded from these rows.
CREATE TABLE form_submissions (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    form_id         uuid        REFERENCES forms(id) ON DELETE SET NULL,
    contact_id      uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    list_ids        uuid[]      NOT NULL DEFAULT '{}',
    double_opt_in   boolean     NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    confirm_sent_at timestamptz,
    send_attempts   integer     NOT NULL DEFAULT 0,
    reminded_at     timestamptz,
    confirmed_at    timestamptz
);
CREATE INDEX form_submissions_pending_idx ON form_submissions (created_at) WHERE double_opt_in AND confirmed_at IS NULL;
CREATE INDEX form_submissions_contact_idx ON form_submissions (contact_id) WHERE confirmed_at IS NULL;
CREATE INDEX form_submissions_form_idx ON form_submissions (form_id, created_at);

-- Submissions that looked automated, held for a Turnstile check for 30
-- minutes. The fields are sealed ("form_challenge:<id>"): they hold
-- personal data and never travel in a URL.
CREATE TABLE form_challenges (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    form_id      uuid        REFERENCES forms(id) ON DELETE CASCADE,
    payload_enc  bytea       NOT NULL,
    origin       text        NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL
);
CREATE INDEX form_challenges_expiry_idx ON form_challenges (expires_at);

-- Public archive ----------------------------------------------------------
-- Sent campaigns of the archived lists, with the slug of their page, the
-- list they are filed under and optional overrides (title, excerpt, label).
CREATE TABLE archive_issues (
    campaign_id  uuid PRIMARY KEY REFERENCES campaigns(id) ON DELETE CASCADE,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    slug         text        NOT NULL CHECK (slug ~ '^[a-z0-9][a-z0-9-]{0,99}$'),
    list_id      uuid        REFERENCES lists(id) ON DELETE SET NULL,
    title        text        NOT NULL DEFAULT '',
    excerpt      text        NOT NULL DEFAULT '',
    label        text        NOT NULL DEFAULT '',
    hidden       boolean     NOT NULL DEFAULT false,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    updated_by   uuid        REFERENCES users(id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX archive_issues_slug_key ON archive_issues (workspace_id, slug);

-- Consent evidence: the exact sentence a contact agreed to, next to the IP,
-- time and version already on the event.
ALTER TABLE consent_events ADD COLUMN consent_text text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE consent_events DROP COLUMN consent_text;
DROP TABLE archive_issues;
DROP TABLE form_challenges;
DROP TABLE form_submissions;
DROP TABLE form_rejections;
DROP TABLE form_daily;
DROP TABLE public_page_settings;
DROP INDEX forms_public_id_key;
ALTER TABLE forms DROP CONSTRAINT forms_public_id_check;
ALTER TABLE forms DROP COLUMN updated_by, DROP COLUMN published_at, DROP COLUMN redirect_url, DROP COLUMN after_submit,
    DROP COLUMN allowed_origins, DROP COLUMN consent_version, DROP COLUMN consent_text, DROP COLUMN list_choice,
    DROP COLUMN default_list_ids, DROP COLUMN public_id;
