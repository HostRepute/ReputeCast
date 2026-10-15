-- +goose Up
-- Content languages (design expansion M17, phase 2 brief §3.4–3.5): the
-- workspace's languages with its default and fallback, language versions of
-- campaigns, automation emails and transactional templates, the block
-- tracking that tells a translation it is outdated, and the language each
-- message went out in.

-- The languages a workspace writes emails and public pages in. Codes are
-- base BCP 47 languages ("es"); a contact's regional code ("es-MX") matches
-- its base language. Removing a language keeps the row (removed_at) so its
-- page copy status and its email versions come back when it is added again.
-- A workspace without rows writes in English, its default and fallback.
CREATE TABLE workspace_languages (
    workspace_id      uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    code              text        NOT NULL CHECK (code ~ '^[a-z]{2,3}$'),
    is_default        boolean     NOT NULL DEFAULT false,
    is_fallback       boolean     NOT NULL DEFAULT false,
    added_by          uuid        REFERENCES users(id) ON DELETE SET NULL,
    added_at          timestamptz NOT NULL DEFAULT now(),
    removed_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    removed_at        timestamptz,
    -- Page copy of the public pages, forms and confirmation email in this
    -- language (kept up to date by the page-copy editor): live, incomplete
    -- (page_copy_missing strings to write) or not set up.
    page_copy_status  text        NOT NULL DEFAULT 'not_set_up' CHECK (page_copy_status IN ('live','incomplete','not_set_up')),
    page_copy_missing integer     NOT NULL DEFAULT 0,
    page_copy_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    page_copy_at      timestamptz,
    PRIMARY KEY (workspace_id, code),
    CHECK (removed_at IS NULL OR (NOT is_default AND NOT is_fallback))
);
CREATE UNIQUE INDEX workspace_languages_default_key ON workspace_languages (workspace_id) WHERE is_default;
CREATE UNIQUE INDEX workspace_languages_fallback_key ON workspace_languages (workspace_id) WHERE is_fallback;

INSERT INTO workspace_languages (workspace_id, code, is_default, is_fallback, added_at, page_copy_status)
SELECT id, 'en', true, true, created_at, 'live' FROM workspaces;

-- The source of an email that has (or had) language versions: the language
-- it is written in, and its top-level blocks (sections of <mj-body>) with
-- keys that stay with a block while it is edited or moved, so a translated
-- block can say which source block it came from.
--   blocks: [{"key":"b4","hash":"…","text":"…","changed":{"version":9,"by":"…","at":"…"}}]
-- owner_version is the version of an automation email or transactional
-- template the source belongs to (their versions are immutable); campaigns
-- use 0, their live draft.
CREATE TABLE content_sources (
    workspace_id  uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    owner_kind    text        NOT NULL CHECK (owner_kind IN ('campaign','automation_email','transactional_template')),
    owner_id      uuid        NOT NULL,
    owner_version integer     NOT NULL DEFAULT 0,
    language      text        NOT NULL,
    blocks        jsonb       NOT NULL DEFAULT '[]',
    subject_hash  text        NOT NULL DEFAULT '',
    preheader_hash text       NOT NULL DEFAULT '',
    -- The source content version the blocks describe (campaigns.version for
    -- a campaign), and the next block key to hand out.
    version       integer     NOT NULL DEFAULT 0,
    next_key      integer     NOT NULL DEFAULT 1,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (owner_kind, owner_id, owner_version)
);
CREATE INDEX content_sources_workspace_idx ON content_sources (workspace_id);

-- One language version of an email: its own subject, preheader and content.
-- A Draft is never sent; a translated version is Up to date or Outdated,
-- which is computed per block from `links` against the source.
--   links: {"subject":{…},"preheader":{…},"blocks":[{"src":"b4","hash":"…","text":"…","version":6,"own":"…"}]}
CREATE TABLE content_translations (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    owner_kind      text        NOT NULL CHECK (owner_kind IN ('campaign','automation_email','transactional_template')),
    owner_id        uuid        NOT NULL,
    owner_version   integer     NOT NULL DEFAULT 0,
    language        text        NOT NULL CHECK (language ~ '^[a-z]{2,3}$'),
    status          text        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','translated')),
    -- "Send English to them instead": readers of this version get the
    -- fallback until it is turned back on.
    held            boolean     NOT NULL DEFAULT false,
    subject         text        NOT NULL DEFAULT '',
    preheader       text        NOT NULL DEFAULT '',
    mjml            text        NOT NULL DEFAULT '',
    html            text        NOT NULL DEFAULT '',
    plain_text      text        NOT NULL DEFAULT '',
    ab_test         jsonb,
    links           jsonb       NOT NULL DEFAULT '{}',
    version         integer     NOT NULL DEFAULT 1,
    content_hash    text        NOT NULL DEFAULT '',
    -- The source version it was translated from (or last marked up to date with).
    translated_from integer     NOT NULL DEFAULT 0,
    translated_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    translated_at   timestamptz,
    created_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at      timestamptz NOT NULL DEFAULT now(),
    autosaved_at    timestamptz
);
CREATE UNIQUE INDEX content_translations_owner_key ON content_translations (owner_kind, owner_id, owner_version, language);
CREATE INDEX content_translations_workspace_idx ON content_translations (workspace_id, language);

-- Saved versions of a language version, newest last (autosave history and
-- conflicts are per language).
CREATE TABLE content_translation_versions (
    translation_id uuid        NOT NULL REFERENCES content_translations(id) ON DELETE CASCADE,
    version        integer     NOT NULL,
    subject        text        NOT NULL DEFAULT '',
    preheader      text        NOT NULL DEFAULT '',
    mjml           text        NOT NULL DEFAULT '',
    html           text        NOT NULL DEFAULT '',
    plain_text     text        NOT NULL DEFAULT '',
    ab_test        jsonb,
    links          jsonb       NOT NULL DEFAULT '{}',
    saved_by       uuid        REFERENCES users(id) ON DELETE SET NULL,
    saved_at       timestamptz NOT NULL DEFAULT now(),
    label          text        NOT NULL DEFAULT '',
    source         text        NOT NULL DEFAULT 'autosave' CHECK (source IN ('create','autosave','manual','restore','api','fix')),
    size_bytes     integer     NOT NULL DEFAULT 0,
    PRIMARY KEY (translation_id, version)
);

-- The language a message went out in and why: the version's language ('' is
-- the source of an email without versions), the contact's language at send
-- time (its base code), and the reason: their own language, no language, a
-- language the workspace doesn't offer, or no ready version (missing, a
-- draft, or held).
ALTER TABLE messages
    ADD COLUMN language         text NOT NULL DEFAULT '',
    ADD COLUMN contact_language text NOT NULL DEFAULT '',
    ADD COLUMN language_reason  text NOT NULL DEFAULT '' CHECK (language_reason IN ('','match','no_language','not_offered','missing','draft','held','requested'));

-- Links of an automation email's language versions.
ALTER TABLE automation_email_links ADD COLUMN language text NOT NULL DEFAULT '';
ALTER TABLE automation_email_links DROP CONSTRAINT automation_email_links_pkey;
ALTER TABLE automation_email_links ADD PRIMARY KEY (email_id, version, language, position);

-- +goose Down
DELETE FROM automation_email_links WHERE language <> '';
ALTER TABLE automation_email_links DROP CONSTRAINT automation_email_links_pkey;
ALTER TABLE automation_email_links ADD PRIMARY KEY (email_id, version, position);
ALTER TABLE automation_email_links DROP COLUMN language;
ALTER TABLE messages DROP COLUMN language_reason, DROP COLUMN contact_language, DROP COLUMN language;
DROP TABLE content_translation_versions;
DROP TABLE content_translations;
DROP TABLE content_sources;
DROP TABLE workspace_languages;
