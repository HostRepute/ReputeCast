-- +goose Up
-- Lists, contacts, consent, custom fields, segments, saved views, imports.

CREATE TABLE lists (
    id                uuid PRIMARY KEY,
    workspace_id      uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name              text        NOT NULL,
    description       text        NOT NULL DEFAULT '',
    double_opt_in     boolean     NOT NULL DEFAULT true,
    requires_approval boolean     NOT NULL DEFAULT false,
    public            boolean     NOT NULL DEFAULT true,
    created_at        timestamptz NOT NULL DEFAULT now(),
    deleted_at        timestamptz
);
CREATE INDEX lists_workspace_idx ON lists (workspace_id) WHERE deleted_at IS NULL;

CREATE TABLE custom_fields (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    key          text        NOT NULL CHECK (key ~ '^[a-z][a-z0-9_]{0,62}$'),
    label        text        NOT NULL,
    type         text        NOT NULL CHECK (type IN ('text','number','date','boolean','select')),
    options      jsonb       NOT NULL DEFAULT '[]',
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX custom_fields_key ON custom_fields (workspace_id, key);

CREATE TABLE contacts (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email           text        NOT NULL CHECK (email = lower(email)),
    first_name      text        NOT NULL DEFAULT '',
    last_name       text        NOT NULL DEFAULT '',
    -- Overall status; per-list status lives in list_memberships.
    status          text        NOT NULL DEFAULT 'active' CHECK (status IN ('active','unsubscribed','bounced','complained','suppressed')),
    -- Engagement tier, recomputed nightly from last_open_at.
    tier            text        NOT NULL DEFAULT 'active' CHECK (tier IN ('active','cooling','at_risk','sunset')),
    source          text        NOT NULL DEFAULT '',
    timezone        text        NOT NULL DEFAULT '',
    country         text        NOT NULL DEFAULT '',
    mailbox_provider text       NOT NULL DEFAULT 'other',
    tags            text[]      NOT NULL DEFAULT '{}',
    attributes      jsonb       NOT NULL DEFAULT '{}',
    last_sent_at    timestamptz,
    last_open_at    timestamptz,
    last_click_at   timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    erased_at       timestamptz
);
CREATE UNIQUE INDEX contacts_email_key ON contacts (workspace_id, email);
CREATE INDEX contacts_tier_idx ON contacts (workspace_id, tier);
CREATE INDEX contacts_created_idx ON contacts (workspace_id, created_at DESC);
CREATE INDEX contacts_last_open_idx ON contacts (workspace_id, last_open_at);
CREATE INDEX contacts_tags_idx ON contacts USING gin (tags);
CREATE INDEX contacts_attributes_idx ON contacts USING gin (attributes jsonb_path_ops);

CREATE TABLE list_memberships (
    list_id         uuid        NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    contact_id      uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    status          text        NOT NULL DEFAULT 'subscribed' CHECK (status IN ('pending','subscribed','unsubscribed')),
    source          text        NOT NULL DEFAULT '',
    subscribed_at   timestamptz,
    unsubscribed_at timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (list_id, contact_id)
);
CREATE INDEX list_memberships_contact_idx ON list_memberships (contact_id);
CREATE INDEX list_memberships_status_idx ON list_memberships (list_id, status);

-- Consent history per contact and list (proof of opt-in, unsubscribes, imports).
CREATE TABLE consent_events (
    id          uuid PRIMARY KEY,
    contact_id  uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    list_id     uuid        REFERENCES lists(id) ON DELETE SET NULL,
    action      text        NOT NULL CHECK (action IN ('subscribed','confirmed','unsubscribed','resubscribed','imported','paused','frequency_changed','erased')),
    source      text        NOT NULL DEFAULT '',
    ip          inet,
    user_agent  text        NOT NULL DEFAULT '',
    detail      text        NOT NULL DEFAULT '',
    occurred_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX consent_events_contact_idx ON consent_events (contact_id, occurred_at DESC);

-- Subscriber preferences from the preference center.
CREATE TABLE contact_preferences (
    contact_id    uuid        PRIMARY KEY REFERENCES contacts(id) ON DELETE CASCADE,
    frequency     text        NOT NULL DEFAULT 'all' CHECK (frequency IN ('all','weekly','monthly')),
    paused_until  timestamptz,
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE segments (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name         text        NOT NULL,
    description  text        NOT NULL DEFAULT '',
    mode         text        NOT NULL DEFAULT 'builder' CHECK (mode IN ('builder','sql')),
    -- Builder definition: {"op":"and","groups":[{"op":"or","conditions":[{"field":"…","operator":"…","value":…}]}]}
    definition   jsonb       NOT NULL DEFAULT '{}',
    -- User-written SQL (mode = sql) runs read-only as the segments role.
    sql          text        NOT NULL DEFAULT '',
    cached_count bigint,
    counted_at   timestamptz,
    created_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX segments_workspace_idx ON segments (workspace_id);

CREATE TABLE saved_views (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id      uuid        REFERENCES users(id) ON DELETE CASCADE,
    name         text        NOT NULL,
    filters      jsonb       NOT NULL DEFAULT '{}',
    columns      text[]      NOT NULL DEFAULT '{}',
    position     integer     NOT NULL DEFAULT 0,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX saved_views_workspace_idx ON saved_views (workspace_id);

CREATE TABLE imports (
    id            uuid PRIMARY KEY,
    workspace_id  uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    source        text        NOT NULL CHECK (source IN ('brevo','mailchimp','mailerlite','listmonk','keila','csv','xlsx')),
    status        text        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','mapping','previewing','running','done','failed','cancelled')),
    file_name     text        NOT NULL DEFAULT '',
    file_data     bytea,
    connection    jsonb       NOT NULL DEFAULT '{}',
    credentials_enc bytea,
    field_map     jsonb       NOT NULL DEFAULT '{}',
    consent_map   jsonb       NOT NULL DEFAULT '{}',
    options       jsonb       NOT NULL DEFAULT '{}',
    preview       jsonb       NOT NULL DEFAULT '{}',
    stats         jsonb       NOT NULL DEFAULT '{}',
    report_csv    bytea,
    error         text        NOT NULL DEFAULT '',
    created_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    started_at    timestamptz,
    finished_at   timestamptz
);
CREATE INDEX imports_workspace_idx ON imports (workspace_id, created_at DESC);

-- +goose Down
DROP TABLE imports;
DROP TABLE saved_views;
DROP TABLE segments;
DROP TABLE contact_preferences;
DROP TABLE consent_events;
DROP TABLE list_memberships;
DROP TABLE contacts;
DROP TABLE custom_fields;
DROP TABLE lists;
