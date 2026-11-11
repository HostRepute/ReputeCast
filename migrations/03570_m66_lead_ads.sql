-- +goose Up
-- M66 G-41: lead ads from Meta (Facebook and Instagram Lead Ads). A
-- workspace connects its own Meta app with a system user's token (sealed),
-- chooses the Pages whose lead forms it collects, and maps each form's
-- questions to contact fields, lists, tags and a consent mode. Meta posts
-- each new lead's ids to /hooks/meta-leads/{token}; the delivery is stored
-- as it came (only ids, never answers) and a job fetches the lead, signs the
-- person up and records what happened, once per lead.
--
-- Never stored: a lead's answers (they go to the contact's fields, or are
-- dropped), and the tokens in clear.

CREATE TABLE lead_connections (
    id                    uuid PRIMARY KEY,
    workspace_id          uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    platform              text        NOT NULL DEFAULT 'meta' CHECK (platform IN ('meta')),
    name                  text        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 80),
    -- The demo's connection answers from the in-process Graph API and is
    -- never polled.
    demo                  boolean     NOT NULL DEFAULT false,
    -- The Meta app's id: not a secret (it's in every Meta URL of the app),
    -- shown on the page for troubleshooting.
    app_id                text        NOT NULL DEFAULT '',
    -- Sealed JSON {appId, appSecret, token} (AD lead_connection:<id>:credentials);
    -- NULL once disconnected.
    credentials_sealed    bytea,
    -- Meta posts to /hooks/meta-leads/{token}: the token's SHA-256, checked in
    -- constant time, and the token sealed for the callback URL.
    callback_token_hash   bytea       UNIQUE,
    callback_token_sealed bytea,
    -- hub.verify_token of Meta's verification request, sealed.
    verify_token_sealed   bytea,
    -- connected; attention (status_detail says why: token_rejected,
    -- token_expired, app_rejected, permission_missing); disconnected.
    status                text        NOT NULL DEFAULT 'connected' CHECK (status IN ('connected', 'attention', 'disconnected')),
    status_detail         text        NOT NULL DEFAULT '',
    -- Why leads are polled every 15 minutes instead of pushed: '' (the
    -- webhook works), not_subscribed (Meta refused the app subscription),
    -- quiet (no delivery for 24 hours while forms are active), unreachable
    -- (the instance isn't reachable from the internet).
    polling_reason        text        NOT NULL DEFAULT '' CHECK (polling_reason IN ('', 'not_subscribed', 'quiet', 'unreachable')),
    rate_limited_until    timestamptz,
    -- The last delivery with a valid signature, and Meta's verification
    -- request (answered) and app subscription (accepted).
    last_event_at         timestamptz,
    verified_at           timestamptz,
    subscribed_at         timestamptz,
    -- The token's last check, and when its problem was told about (once).
    checked_at            timestamptz,
    notified_at           timestamptz,
    created_by            uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at            timestamptz NOT NULL,
    updated_at            timestamptz NOT NULL,
    disconnected_at       timestamptz,
    disconnected_by       uuid        REFERENCES users(id) ON DELETE SET NULL
);
-- One Meta connection at a time per workspace; disconnected ones stay with
-- their history.
CREATE UNIQUE INDEX lead_connections_live ON lead_connections (workspace_id, platform) WHERE status <> 'disconnected';

-- The Pages the token reaches that were chosen (chosen = false: left out
-- later, its mapping kept, unsubscribed).
CREATE TABLE lead_pages (
    id                uuid PRIMARY KEY,
    workspace_id      uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    connection_id     uuid        NOT NULL REFERENCES lead_connections(id) ON DELETE CASCADE,
    page_id           text        NOT NULL,
    name              text        NOT NULL DEFAULT '',
    -- The Page's own token (from the system user's), sealed
    -- (AD lead_page:<id>:token); NULL once left out or disconnected.
    page_token_sealed bytea,
    chosen            boolean     NOT NULL DEFAULT true,
    subscribed_at     timestamptz,
    created_at        timestamptz NOT NULL,
    updated_at        timestamptz NOT NULL,
    UNIQUE (connection_id, page_id)
);

-- A Page's lead forms, as Meta lists them, and how each one is collected.
CREATE TABLE lead_forms (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    page_id         uuid        NOT NULL REFERENCES lead_pages(id) ON DELETE CASCADE,
    form_id         text        NOT NULL,
    name            text        NOT NULL DEFAULT '',
    -- Meta's status: ACTIVE, ARCHIVED, DELETED, DRAFT.
    meta_status     text        NOT NULL DEFAULT 'ACTIVE',
    -- The questions [{key, label, type}] and the custom disclaimer's
    -- checkboxes [{key, text, required}] as Meta returned them last.
    questions       jsonb       NOT NULL DEFAULT '[]',
    checkboxes      jsonb       NOT NULL DEFAULT '[]',
    disclaimer      text        NOT NULL DEFAULT '',
    -- active or paused; a form is collected once mapped (mapped_at).
    status          text        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused')),
    mapped_at       timestamptz,
    -- question key → email, first_name, last_name, full_name, company,
    -- phone or field:<custom field key>.
    mapping         jsonb       NOT NULL DEFAULT '{}',
    consent_mode    text        NOT NULL DEFAULT 'double' CHECK (consent_mode IN ('double', 'single')),
    -- Single opt-in: the checkbox, its text as Meta returned it and that
    -- text's hash.
    consent_key     text        NOT NULL DEFAULT '',
    consent_text    text        NOT NULL DEFAULT '',
    consent_version text        NOT NULL DEFAULT '',
    list_ids        uuid[]      NOT NULL DEFAULT '{}',
    tags            text[]      NOT NULL DEFAULT '{}',
    -- Leads that arrived while the form wasn't mapped or was paused:
    -- counted, not stored.
    waiting         integer     NOT NULL DEFAULT 0,
    -- Import past leads (the last 90 days).
    import_state    text        NOT NULL DEFAULT '' CHECK (import_state IN ('', 'running', 'done', 'failed')),
    import_count    integer     NOT NULL DEFAULT 0,
    imported_at     timestamptz,
    last_polled_at  timestamptz,
    last_lead_at    timestamptz,
    created_at      timestamptz NOT NULL,
    updated_at      timestamptz NOT NULL,
    UNIQUE (page_id, form_id)
);

-- Meta's deliveries as they came, kept 30 days. A bad signature keeps 2 KB
-- for diagnosis and is never applied.
CREATE TABLE lead_events (
    id            uuid PRIMARY KEY,
    workspace_id  uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    connection_id uuid        NOT NULL REFERENCES lead_connections(id) ON DELETE CASCADE,
    received_at   timestamptz NOT NULL,
    body          bytea       NOT NULL,
    signature_ok  boolean     NOT NULL,
    status        text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'applied', 'rejected', 'failed')),
    error         text        NOT NULL DEFAULT '',
    applied_at    timestamptz
);
CREATE INDEX lead_events_pending ON lead_events (received_at) WHERE status = 'pending';
CREATE INDEX lead_events_received ON lead_events (received_at);

-- Each lead once (by Meta's id per form), what it did and the contact.
CREATE TABLE lead_entries (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    form_id      uuid        NOT NULL REFERENCES lead_forms(id) ON DELETE CASCADE,
    leadgen_id   text        NOT NULL,
    contact_id   uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    -- created, updated (an existing contact), pending (double opt-in),
    -- blocked (a suppressed address), invalid (no valid email), declined
    -- (single opt-in, the consent box not ticked), test (Meta's test lead).
    outcome      text        NOT NULL CHECK (outcome IN ('created', 'updated', 'pending', 'blocked', 'invalid', 'declined', 'test')),
    source       text        NOT NULL DEFAULT 'webhook' CHECK (source IN ('webhook', 'poll', 'import')),
    -- Meta's created_time, and when ReputeCast applied it.
    lead_at      timestamptz NOT NULL,
    created_at   timestamptz NOT NULL,
    -- A double opt-in lead's confirmation, which starts its flows.
    confirmed_at timestamptz,
    UNIQUE (form_id, leadgen_id)
);
CREATE INDEX lead_entries_recent ON lead_entries (workspace_id, created_at DESC);
CREATE INDEX lead_entries_form ON lead_entries (form_id, lead_at);
CREATE INDEX lead_entries_contact ON lead_entries (contact_id) WHERE contact_id IS NOT NULL;

-- +goose Down
DROP TABLE lead_entries;
DROP TABLE lead_events;
DROP TABLE lead_forms;
DROP TABLE lead_pages;
DROP TABLE lead_connections;
