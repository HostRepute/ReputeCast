-- +goose Up
-- Agency accounts (design expansion phase 8, M32; channels and agencies
-- brief §3.10): one agency per instance, its staff and their levels, the
-- client workspaces with their account manager and rebill rates, workspace
-- templates, the shared library with its versions and the copies made of
-- it, usage per client and day, and the daily sender-health snapshots the
-- agency overview draws its trend from.
--
-- Assignments are ordinary workspace memberships (source 'agency'), so they
-- need no table of their own. Client roles are built-in roles that only
-- client workspaces get (internal/authz/agency.go).

-- Agency mode: one row, like instance_settings. Off by default.
CREATE TABLE agency_settings (
    id                  boolean PRIMARY KEY DEFAULT true CHECK (id),
    enabled             boolean     NOT NULL DEFAULT false,
    -- The agency's name; the instance name when empty.
    name                text        NOT NULL DEFAULT '',
    -- The template a new client starts from unless another is picked.
    default_template_id uuid,
    enabled_by          uuid        REFERENCES users(id) ON DELETE SET NULL,
    enabled_at          timestamptz,
    updated_by          uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at          timestamptz NOT NULL DEFAULT now()
);
INSERT INTO agency_settings (id) VALUES (true);

-- The agency's staff. Instance admins are agency managers without a row.
-- A person invited by email has no account yet: the row keeps the email
-- and is bound to the account when they first sign in.
CREATE TABLE agency_staff (
    id       uuid PRIMARY KEY,
    user_id  uuid        UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    email    text        NOT NULL CHECK (email = lower(email)),
    level    text        NOT NULL CHECK (level IN ('agency_manager','account_manager')),
    -- What they do at the agency ("designer"), shown under the level.
    title    text        NOT NULL DEFAULT '',
    -- Narrows the level's agency permissions (an account manager without
    -- agency:usage); NULL holds all of the level's.
    permissions text[],
    added_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    added_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX agency_staff_email_key ON agency_staff (email);

-- Workspace templates: blueprints for setting up a client (what is copied
-- is in content; providers, domains, contacts, suppressions, consent, keys,
-- integrations and members never are).
CREATE TABLE agency_templates (
    id            uuid PRIMARY KEY,
    name          text        NOT NULL,
    description   text        NOT NULL DEFAULT '',
    content       jsonb       NOT NULL DEFAULT '{}',
    -- The client it was saved from ("Save as workspace template").
    created_from  uuid        REFERENCES workspaces(id) ON DELETE SET NULL,
    created_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz
);
CREATE UNIQUE INDEX agency_templates_name_key ON agency_templates (lower(name)) WHERE deleted_at IS NULL;
ALTER TABLE agency_settings ADD CONSTRAINT agency_settings_default_template_fk
    FOREIGN KEY (default_template_id) REFERENCES agency_templates(id) ON DELETE SET NULL;

-- The agency's workspaces: its clients and its own (kind 'agency', at most
-- one). Each keeps its own providers, domains, contacts, suppressions,
-- consent, keys and audit log; nothing here is shared between them.
CREATE TABLE agency_workspaces (
    workspace_id       uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    kind               text        NOT NULL DEFAULT 'client' CHECK (kind IN ('client','agency')),
    -- The client's contact at the agency ("your contact: Mei Lin").
    account_manager_id uuid        REFERENCES users(id) ON DELETE SET NULL,
    client_since       date        NOT NULL,
    -- The template it was set up from, and the setup steps it asked for
    -- (["provider","domain","invite","shopify"]).
    template_id        uuid        REFERENCES agency_templates(id) ON DELETE SET NULL,
    setup_steps        text[]      NOT NULL DEFAULT '{}',
    -- The colour of the client's initials tile, kept so it never changes.
    color              text        NOT NULL DEFAULT '',
    added_by           uuid        REFERENCES users(id) ON DELETE SET NULL,
    added_at           timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX agency_workspaces_agency_key ON agency_workspaces (kind) WHERE kind = 'agency';

-- Rebill rates. A rate applies from the first day of a month until the next
-- one; an empty rate ({}) means the client isn't billed.
-- {"costPlusPct":20,"per1000Emails":0.25,"perSegment":0.02,"monthlyFee":150,"flat":120}
CREATE TABLE agency_rates (
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    effective_from date        NOT NULL,
    rate           jsonb       NOT NULL DEFAULT '{}',
    set_by         uuid        REFERENCES users(id) ON DELETE SET NULL,
    set_at         timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, effective_from)
);

-- Every time a template was applied, with what it added and renamed.
CREATE TABLE agency_template_applications (
    id           uuid PRIMARY KEY,
    template_id  uuid        NOT NULL REFERENCES agency_templates(id) ON DELETE CASCADE,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    applied_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    applied_at   timestamptz NOT NULL DEFAULT now(),
    result       jsonb       NOT NULL DEFAULT '{}'
);
CREATE INDEX agency_template_applications_template_idx ON agency_template_applications (template_id, applied_at DESC);
CREATE INDEX agency_template_applications_workspace_idx ON agency_template_applications (workspace_id, applied_at DESC);

-- The shared library: saved sections, campaign templates, automation
-- templates, transactional templates and SMS texts, each versioned.
CREATE TABLE agency_library_items (
    id         uuid PRIMARY KEY,
    kind       text        NOT NULL CHECK (kind IN ('section','template','automation','transactional','sms')),
    name       text        NOT NULL,
    version    integer     NOT NULL DEFAULT 1,
    -- The Liquid fields it needs, with their fallbacks: [{"key":"first_name","fallback":"there"}].
    fields     jsonb       NOT NULL DEFAULT '[]',
    created_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE INDEX agency_library_items_live_idx ON agency_library_items (kind, updated_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE agency_library_versions (
    item_id             uuid        NOT NULL REFERENCES agency_library_items(id) ON DELETE CASCADE,
    version             integer     NOT NULL,
    -- By kind: {"mjml","html","thumb"} for sections and templates; {"graph",
    -- "emails",…} for automations; {"subject","mjml",…} for transactional
    -- templates; {"body"} for SMS texts.
    content             jsonb       NOT NULL,
    -- The lines the diff between versions compares ("Button: Shop gifts").
    summary             text[]      NOT NULL DEFAULT '{}',
    note                text        NOT NULL DEFAULT '',
    source_workspace_id uuid        REFERENCES workspaces(id) ON DELETE SET NULL,
    created_by          uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (item_id, version)
);

-- Copies of library items in clients. Copies aren't linked: this only
-- remembers which version a copy came from, for "Library has a newer
-- version" and "Update this copy".
CREATE TABLE agency_library_copies (
    id           uuid PRIMARY KEY,
    item_id      uuid        NOT NULL REFERENCES agency_library_items(id) ON DELETE CASCADE,
    version      integer     NOT NULL,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    -- template (sections and campaign templates), automation, transactional, sms.
    target_kind  text        NOT NULL,
    target_id    uuid        NOT NULL,
    copied_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    copied_at    timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX agency_library_copies_item_idx ON agency_library_copies (item_id);
CREATE INDEX agency_library_copies_workspace_idx ON agency_library_copies (workspace_id);
CREATE UNIQUE INDEX agency_library_copies_target_key ON agency_library_copies (target_kind, target_id);

-- Usage per client and day, rolled up every hour from the message tables
-- (emails by stream), each provider's daily statistics (emails and their
-- cost) and the other channels' logs (SMS segments, push notifications and
-- WhatsApp messages). Rows the demo seeded (demo) are never recomputed.
CREATE TABLE agency_usage (
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    day          date        NOT NULL,
    channel      text        NOT NULL CHECK (channel IN ('email','sms','push','whatsapp')),
    -- stream: emails per stream (key broadcast, automation, transactional,
    -- system); provider: units and cost per provider (key its id); country:
    -- SMS segments per country (key the ISO code).
    dimension    text        NOT NULL CHECK (dimension IN ('stream','provider','country')),
    key          text        NOT NULL,
    label        text        NOT NULL DEFAULT '',
    units        bigint      NOT NULL DEFAULT 0,
    cost_micros  bigint      NOT NULL DEFAULT 0,
    demo         boolean     NOT NULL DEFAULT false,
    computed_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, day, channel, dimension, key)
);
CREATE INDEX agency_usage_day_idx ON agency_usage (day);

-- The Overview's sender health per client and day: the agency overview's
-- score and its 30-day trend. Refreshed every hour for today.
CREATE TABLE agency_health (
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    day          date        NOT NULL,
    -- ready, or pending while the client has no sending domain yet.
    state        text        NOT NULL DEFAULT 'ready' CHECK (state IN ('ready','pending')),
    score        integer     NOT NULL DEFAULT 0,
    tone         text        NOT NULL DEFAULT 'neutral',
    -- '' when HostRepute isn't connected; clean, listed or stale.
    hostrepute   text        NOT NULL DEFAULT '' CHECK (hostrepute IN ('','clean','listed','stale')),
    listings     integer     NOT NULL DEFAULT 0,
    demo         boolean     NOT NULL DEFAULT false,
    computed_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, day)
);

-- +goose Down
DROP TABLE agency_health;
DROP TABLE agency_usage;
DROP TABLE agency_library_copies;
DROP TABLE agency_library_versions;
DROP TABLE agency_library_items;
DROP TABLE agency_template_applications;
DROP TABLE agency_rates;
DROP TABLE agency_workspaces;
ALTER TABLE agency_settings DROP CONSTRAINT agency_settings_default_template_fk;
DROP TABLE agency_templates;
DROP TABLE agency_staff;
DROP TABLE agency_settings;
