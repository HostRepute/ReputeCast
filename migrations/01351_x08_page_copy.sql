-- +goose Up
-- Page and form copy per language (design expansion M17, phase 2 brief
-- §3.6). The public pages and each form are written in one language (their
-- copy language, the workspace's default when they were made); the other
-- workspace languages get their own copy, and a language's pages and forms
-- are used only once that copy is complete ("Live").

-- The public pages' own text in a language other than their copy language
-- (headings, descriptions, the lines under each list, the fewer-emails
-- offer, list names and descriptions, the privacy policy link), as
-- {"subscribe.heading": "…", "list.<id>.description": "…"}; and the
-- built-in labels the workspace wrote for a language ReputeCast doesn't ship
-- them in ({"form.button": "Suscribirme"}).
CREATE TABLE page_copy (
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    language     text        NOT NULL CHECK (language ~ '^[a-z]{2,3}$'),
    pages        jsonb       NOT NULL DEFAULT '{}'::jsonb,
    pages_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    pages_at     timestamptz,
    labels       jsonb       NOT NULL DEFAULT '{}'::jsonb,
    labels_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    labels_at    timestamptz,
    PRIMARY KEY (workspace_id, language)
);

-- A form's copy in a language other than its copy language. The consent
-- text has its own versions in each language: consent_version counts this
-- language's texts, consent_for is the version of the form's own consent
-- text it was written (or last confirmed) for, so a newer one flags it
-- "Review the translation".
CREATE TABLE form_copy (
    form_id         uuid        NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    language        text        NOT NULL CHECK (language ~ '^[a-z]{2,3}$'),
    content         jsonb       NOT NULL DEFAULT '{}'::jsonb,
    consent_text    text        NOT NULL DEFAULT '',
    consent_version integer     NOT NULL DEFAULT 0,
    consent_for     integer     NOT NULL DEFAULT 0,
    updated_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (form_id, language)
);

-- Which language a form shows (language_mode): the page it's on (its lang
-- attribute), the visitor's browser, or always one language. Embedded and
-- popup forms match the page by default; hosted forms follow the browser.
-- consent_changed_* name who last changed the form's own consent text: the
-- other languages' consent texts are flagged for review against it.
ALTER TABLE forms
    ADD COLUMN copy_language      text NOT NULL DEFAULT 'en' CHECK (copy_language ~ '^[a-z]{2,3}$'),
    ADD COLUMN language_mode      text NOT NULL DEFAULT 'page' CHECK (language_mode IN ('page', 'browser', 'fixed')),
    ADD COLUMN fixed_language     text NOT NULL DEFAULT '' CHECK (fixed_language = '' OR fixed_language ~ '^[a-z]{2,3}$'),
    ADD COLUMN consent_changed_by uuid REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN consent_changed_at timestamptz;
UPDATE forms SET language_mode = 'browser' WHERE kind = 'hosted';

ALTER TABLE public_page_settings
    ADD COLUMN copy_language text NOT NULL DEFAULT 'en' CHECK (copy_language ~ '^[a-z]{2,3}$');

-- The language a form was shown in: its confirmation email and the
-- confirmation page follow it.
ALTER TABLE form_submissions ADD COLUMN language text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE form_submissions DROP COLUMN language;
ALTER TABLE public_page_settings DROP COLUMN copy_language;
ALTER TABLE forms DROP COLUMN consent_changed_at, DROP COLUMN consent_changed_by, DROP COLUMN fixed_language, DROP COLUMN language_mode, DROP COLUMN copy_language;
DROP TABLE form_copy;
DROP TABLE page_copy;
