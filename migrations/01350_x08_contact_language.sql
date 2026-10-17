-- +goose Up
-- A contact's language and where it came from (design expansion M17, phase
-- 2 brief §3.4): signup forms set it from the browser's first language when
-- the contact has none, imports from a mapped column, the API and
-- integrations, the preference center's "Email language", and the team on
-- the profile. Codes are stored normalized (BCP 47, "es-MX").
ALTER TABLE contacts
    ADD COLUMN language_source      text        NOT NULL DEFAULT ''
        CHECK (language_source IN ('', 'import', 'form', 'preference_center', 'api', 'team')),
    -- The importer ("Brevo"), the form ("Newsletter signup"), the integration
    -- ("Zapier") or the API key that set it, as it was named then.
    ADD COLUMN language_source_name text        NOT NULL DEFAULT '',
    -- A signup form read it from the browser's languages.
    ADD COLUMN language_browser     boolean     NOT NULL DEFAULT false,
    ADD COLUMN language_set_at      timestamptz,
    -- The team member who chose it on the profile.
    ADD COLUMN language_set_by      uuid        REFERENCES users(id) ON DELETE SET NULL;

-- Consent evidence keeps the language the consent text was shown in, next
-- to its version ("v1" of the Spanish text).
ALTER TABLE consent_events ADD COLUMN language text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE consent_events DROP COLUMN language;
ALTER TABLE contacts DROP COLUMN language_set_by, DROP COLUMN language_set_at, DROP COLUMN language_browser,
    DROP COLUMN language_source_name, DROP COLUMN language_source;
