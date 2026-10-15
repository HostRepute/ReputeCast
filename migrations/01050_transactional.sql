-- +goose Up
-- Transactional email (design expansion phase 1, M15): the fourth stream,
-- templates with variables, test data and published versions, the send
-- API's message log, categories and their opt-outs, settings, delivery
-- alerts, send-only and test-mode keys, and the SMTP relay.

-- The Transactional stream joins Broadcasts, Automations and System.
ALTER TABLE routing_rules DROP CONSTRAINT routing_rules_stream_check;
ALTER TABLE routing_rules ADD CONSTRAINT routing_rules_stream_check
    CHECK (stream IN ('broadcast','automation','system','transactional'));
ALTER TABLE stream_policies DROP CONSTRAINT stream_policies_stream_check;
ALTER TABLE stream_policies ADD CONSTRAINT stream_policies_stream_check
    CHECK (stream IN ('broadcast','automation','system','transactional'));
ALTER TABLE stream_failovers DROP CONSTRAINT stream_failovers_stream_check;
ALTER TABLE stream_failovers ADD CONSTRAINT stream_failovers_stream_check
    CHECK (stream IN ('broadcast','automation','system','transactional'));
ALTER TABLE messages DROP CONSTRAINT messages_stream_check;
ALTER TABLE messages ADD CONSTRAINT messages_stream_check
    CHECK (stream IN ('broadcast','automation','system','transactional'));

-- "First in line on shared providers": transactional messages take their
-- share of a provider's per-second rate before Broadcasts do. Only the
-- Transactional stream reads it; on by default.
ALTER TABLE stream_policies ADD COLUMN priority boolean NOT NULL DEFAULT true;

-- The transactional outbox walks queued transactional mail per provider.
CREATE INDEX messages_transactional_outbox_idx ON messages (provider_id, id)
    WHERE status = 'queued' AND stream = 'transactional';

-- A sending domain used for transactional mail: verified senders on it are
-- the workspace's Transactional senders (the API's "from", the SMTP relay's
-- allowed From addresses).
ALTER TABLE sending_domains ADD COLUMN transactional boolean NOT NULL DEFAULT false;

-- Settings › Transactional, one row per workspace (created on first use).
-- guide keeps the first-send guide's progress so a reload resumes the step.
CREATE TABLE transactional_settings (
    workspace_id           uuid        PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    link_contacts          boolean     NOT NULL DEFAULT true,
    content_retention_days integer     NOT NULL DEFAULT 30 CHECK (content_retention_days BETWEEN 0 AND 90),
    alert_bounce_rate      numeric(5,2) NOT NULL DEFAULT 2.0 CHECK (alert_bounce_rate > 0 AND alert_bounce_rate <= 100),
    alert_failed           integer     NOT NULL DEFAULT 10 CHECK (alert_failed BETWEEN 1 AND 100000),
    alert_median_seconds   integer     NOT NULL DEFAULT 30 CHECK (alert_median_seconds BETWEEN 1 AND 3600),
    default_sender_id      uuid        REFERENCES senders(id) ON DELETE SET NULL,
    default_reply_to       text        NOT NULL DEFAULT '',
    default_track_opens    boolean     NOT NULL DEFAULT false,
    default_track_clicks   boolean     NOT NULL DEFAULT false,
    guide                  jsonb       NOT NULL DEFAULT '{}',
    updated_by             uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at             timestamptz NOT NULL DEFAULT now()
);

-- Categories group templates. Essential ones are always sent; recipients
-- can turn Optional ones off in the preference center or with the
-- category's unsubscribe link.
CREATE TABLE transactional_categories (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name         text        NOT NULL,
    kind         text        NOT NULL CHECK (kind IN ('essential','optional')),
    description  text        NOT NULL DEFAULT '',
    position     integer     NOT NULL DEFAULT 0,
    created_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX transactional_categories_name_key ON transactional_categories (workspace_id, lower(name));

-- A template: its ID (key) is what the API sends ("password-reset"); it is
-- fixed once the template was first published. The draft lives on the row
-- and saves itself (draft_revision is the autosave counter the editor
-- sends back to detect a teammate's newer save); publishing copies the
-- draft into transactional_template_versions and points live_version at it.
-- variables: [{"name":"reset_url","type":"url","required":true,"sample":"…","default":"","sensitive":true}]
-- test_data: [{"name":"New user","data":{…}}]
CREATE TABLE transactional_templates (
    id                 uuid PRIMARY KEY,
    workspace_id       uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    key                text        NOT NULL CHECK (key ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(key) <= 64),
    name               text        NOT NULL,
    description        text        NOT NULL DEFAULT '',
    category_id        uuid        REFERENCES transactional_categories(id) ON DELETE RESTRICT,
    -- Draft content and settings.
    subject            text        NOT NULL DEFAULT '',
    preheader          text        NOT NULL DEFAULT '',
    editor_mode        text        NOT NULL DEFAULT 'visual' CHECK (editor_mode IN ('visual','mjml','html')),
    mjml               text        NOT NULL DEFAULT '',
    html               text        NOT NULL DEFAULT '',
    plain_text         text        NOT NULL DEFAULT '',
    variables          jsonb       NOT NULL DEFAULT '[]',
    test_data          jsonb       NOT NULL DEFAULT '[]',
    sender_id          uuid        REFERENCES senders(id) ON DELETE SET NULL,
    reply_to           text        NOT NULL DEFAULT '',
    track_opens        boolean     NOT NULL DEFAULT false,
    track_clicks       boolean     NOT NULL DEFAULT false,
    draft_revision     integer     NOT NULL DEFAULT 1,
    draft_hash         text        NOT NULL DEFAULT '',
    draft_saved_at     timestamptz NOT NULL DEFAULT now(),
    draft_saved_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    -- Publishing.
    live_version       integer,
    live_since         timestamptz,
    live_by            uuid        REFERENCES users(id) ON DELETE SET NULL,
    versions_count     integer     NOT NULL DEFAULT 0,
    first_published_at timestamptz,
    -- Where an imported template came from ({"source":"postmark","id":"…"}).
    origin             jsonb       NOT NULL DEFAULT '{}',
    archived_at        timestamptz,
    archived_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    deleted_at         timestamptz,
    created_by         uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX transactional_templates_key ON transactional_templates (workspace_id, key) WHERE deleted_at IS NULL;
CREATE INDEX transactional_templates_ws_idx ON transactional_templates (workspace_id, created_at) WHERE deleted_at IS NULL;

-- Earlier draft revisions, for the conflict diff and restoring an autosave.
CREATE TABLE transactional_template_drafts (
    template_id uuid        NOT NULL REFERENCES transactional_templates(id) ON DELETE CASCADE,
    revision    integer     NOT NULL,
    subject     text        NOT NULL,
    preheader   text        NOT NULL,
    mjml        text        NOT NULL,
    html        text        NOT NULL,
    variables   jsonb       NOT NULL,
    saved_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    saved_at    timestamptz NOT NULL DEFAULT now(),
    label       text        NOT NULL DEFAULT '',
    source      text        NOT NULL DEFAULT 'autosave',
    PRIMARY KEY (template_id, revision)
);

-- Published versions (v1, v2…). Rolling back points live_version at an
-- older one; nothing is copied. change is the summary shown in the history
-- ("trial_end_date replaces days_left · default “soon”").
CREATE TABLE transactional_template_versions (
    template_id    uuid        NOT NULL REFERENCES transactional_templates(id) ON DELETE CASCADE,
    version        integer     NOT NULL,
    subject        text        NOT NULL,
    preheader      text        NOT NULL,
    editor_mode    text        NOT NULL,
    mjml           text        NOT NULL,
    html           text        NOT NULL,
    plain_text     text        NOT NULL,
    variables      jsonb       NOT NULL,
    test_data      jsonb       NOT NULL,
    sender_id      uuid        REFERENCES senders(id) ON DELETE SET NULL,
    reply_to       text        NOT NULL DEFAULT '',
    track_opens    boolean     NOT NULL DEFAULT false,
    track_clicks   boolean     NOT NULL DEFAULT false,
    content_hash   text        NOT NULL,
    size_bytes     integer     NOT NULL DEFAULT 0,
    change         text        NOT NULL DEFAULT '',
    note           text        NOT NULL DEFAULT '',
    published_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    published_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (template_id, version)
);

-- One row per transactional message, next to its messages row (the
-- delivery queue the sending engine and webhook ingest share). It is the
-- message log's record: the request that produced it (key, address,
-- idempotency key, tags, metadata, the variables' names and their values
-- except sensitive ones), the template version, the route decision and
-- the provider attempts. Kept 90 days.
CREATE TABLE transactional_messages (
    message_id         uuid        PRIMARY KEY REFERENCES messages(id) ON DELETE CASCADE,
    workspace_id       uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    -- api, smtp, batch, resend, guide (sent from the first-send guide) or
    -- test (a test send from the editor).
    source             text        NOT NULL CHECK (source IN ('api','batch','smtp','resend','guide','test')),
    template_id        uuid        REFERENCES transactional_templates(id) ON DELETE SET NULL,
    template_key       text        NOT NULL DEFAULT '',
    template_name      text        NOT NULL DEFAULT '',
    template_version   integer,
    category_id        uuid        REFERENCES transactional_categories(id) ON DELETE SET NULL,
    category_name      text        NOT NULL DEFAULT '',
    category_kind      text        NOT NULL DEFAULT '',
    key_id             uuid        REFERENCES api_keys(id) ON DELETE SET NULL,
    key_name           text        NOT NULL DEFAULT '',
    user_id            uuid        REFERENCES users(id) ON DELETE SET NULL,
    ip                 inet,
    idempotency_key    text        NOT NULL DEFAULT '',
    subject            text        NOT NULL DEFAULT '',
    from_email         text        NOT NULL DEFAULT '',
    from_name          text        NOT NULL DEFAULT '',
    reply_to           text        NOT NULL DEFAULT '',
    to_name            text        NOT NULL DEFAULT '',
    cc                 text[]      NOT NULL DEFAULT '{}',
    bcc                text[]      NOT NULL DEFAULT '{}',
    tags               text[]      NOT NULL DEFAULT '{}',
    metadata           jsonb       NOT NULL DEFAULT '{}',
    -- The variables as sent, with sensitive values replaced by "••••••".
    data               jsonb       NOT NULL DEFAULT '{}',
    data_keys          text[]      NOT NULL DEFAULT '{}',
    sensitive          text[]      NOT NULL DEFAULT '{}',
    -- [{"filename":"INV-0418.pdf","content_type":"application/pdf","size":48211,"content_id":""}]
    attachments        jsonb       NOT NULL DEFAULT '[]',
    size_bytes         integer     NOT NULL DEFAULT 0,
    test               boolean     NOT NULL DEFAULT false,
    test_delivered     boolean     NOT NULL DEFAULT false,
    send_at            timestamptz,
    -- Why a message stopped before sending: suppressed, opted_out, invalid,
    -- test_not_allowed; detail holds what the log shows about it.
    reject_reason      text        NOT NULL DEFAULT '',
    reject_detail      jsonb       NOT NULL DEFAULT '{}',
    track_opens        boolean     NOT NULL DEFAULT false,
    track_clicks       boolean     NOT NULL DEFAULT false,
    -- Tracked links by position, as rendered for this message.
    links              text[]      NOT NULL DEFAULT '{}',
    -- {"rule":"Default","matched":"","value":"","primary":"…","fallback":"…","policy":"automatic","provider":"…"}
    route              jsonb       NOT NULL DEFAULT '{}',
    -- [{"provider_id":"…","provider":"Postmark","at":"…","error":"503 Service Unavailable","temporary":true}]
    attempts           jsonb       NOT NULL DEFAULT '[]',
    failed_over_from   uuid        REFERENCES providers(id) ON DELETE SET NULL,
    failed_over_at     timestamptz,
    -- When the message reached the current provider (retries end 10 minutes later).
    provider_since     timestamptz,
    accepted_at        timestamptz NOT NULL,
    rendered_at        timestamptz,
    content_expires_at timestamptz,
    content_removed_at timestamptz,
    resent_from        uuid,
    -- Webhook events emitted for the message ("message.failed").
    events_emitted     text[]      NOT NULL DEFAULT '{}'
);
CREATE INDEX transactional_messages_log_idx ON transactional_messages (workspace_id, accepted_at DESC, message_id DESC);
CREATE INDEX transactional_messages_template_idx ON transactional_messages (template_id, accepted_at DESC);
CREATE INDEX transactional_messages_key_idx ON transactional_messages (key_id, accepted_at DESC);
CREATE INDEX transactional_messages_idem_idx ON transactional_messages (workspace_id, idempotency_key) WHERE idempotency_key <> '';
CREATE INDEX transactional_messages_content_idx ON transactional_messages (content_expires_at) WHERE content_expires_at IS NOT NULL AND content_removed_at IS NULL;

-- The rendered content shown on the message detail, with sensitive values
-- masked. Kept as long as the workspace's retention says (0–90 days).
CREATE TABLE transactional_message_content (
    message_id uuid        PRIMARY KEY REFERENCES messages(id) ON DELETE CASCADE,
    html       text        NOT NULL DEFAULT '',
    text       text        NOT NULL DEFAULT '',
    headers    jsonb       NOT NULL DEFAULT '{}',
    expires_at timestamptz NOT NULL
);

-- What the sending engine hands to the provider: the content as sent (with
-- sensitive values and attachments), sealed with the instance key. A row
-- lives only until the provider accepted the message or every provider
-- failed, so sensitive values are never kept.
CREATE TABLE transactional_outbox (
    message_id uuid        PRIMARY KEY REFERENCES messages(id) ON DELETE CASCADE,
    sealed     bytea       NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Idempotency keys: a retry with the same key within 24 hours gets the
-- first answer and sends nothing.
CREATE TABLE transactional_idempotency (
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    key          text        NOT NULL,
    request_hash text        NOT NULL,
    status       integer     NOT NULL DEFAULT 0,
    response     jsonb       NOT NULL DEFAULT '{}',
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, key)
);
CREATE INDEX transactional_idempotency_created_idx ON transactional_idempotency (created_at);

-- Calls the API refused before a message existed (template_not_published,
-- missing_variables…): the first-send guide shows the last one, and the
-- breaking-change check counts them. Kept 90 days.
CREATE TABLE transactional_rejected_calls (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    key_id          uuid        REFERENCES api_keys(id) ON DELETE SET NULL,
    user_id         uuid        REFERENCES users(id) ON DELETE SET NULL,
    at              timestamptz NOT NULL,
    status          integer     NOT NULL,
    code            text        NOT NULL,
    detail          text        NOT NULL DEFAULT '',
    template_key    text        NOT NULL DEFAULT '',
    recipient       text        NOT NULL DEFAULT '',
    idempotency_key text        NOT NULL DEFAULT '',
    ip              inet
);
CREATE INDEX transactional_rejected_calls_ws_idx ON transactional_rejected_calls (workspace_id, at DESC);

-- Opt-outs of Optional categories, per address (recipients don't have to
-- be contacts).
CREATE TABLE transactional_opt_outs (
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email        text        NOT NULL CHECK (email = lower(email)),
    category_id  uuid        NOT NULL REFERENCES transactional_categories(id) ON DELETE CASCADE,
    opted_out_at timestamptz NOT NULL DEFAULT now(),
    -- preference_center, unsubscribe_link, one_click, api, member
    source       text        NOT NULL DEFAULT '',
    reason       text        NOT NULL DEFAULT '',
    message_id   uuid,
    PRIMARY KEY (workspace_id, email, category_id)
);
CREATE INDEX transactional_opt_outs_category_idx ON transactional_opt_outs (category_id);

-- Delivery alerts (bounce rate per template, failed messages, median
-- delivery time over the last hour) and failovers, for the overview's
-- "Incidents and alerts" and the notification history.
CREATE TABLE transactional_alerts (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    kind         text        NOT NULL CHECK (kind IN ('bounce_rate','failed','median_delivery','failover')),
    template_id  uuid        REFERENCES transactional_templates(id) ON DELETE SET NULL,
    provider_id  uuid        REFERENCES providers(id) ON DELETE SET NULL,
    value        numeric(12,4) NOT NULL DEFAULT 0,
    threshold    numeric(12,4) NOT NULL DEFAULT 0,
    -- {"sent":620,"bounced":15,"domains":{"gmial.com":9},"messages":37,"lost":0,"fallback":"…"}
    detail       jsonb       NOT NULL DEFAULT '{}',
    started_at   timestamptz NOT NULL,
    ended_at     timestamptz,
    seen_at      timestamptz,
    seen_by      uuid        REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX transactional_alerts_ws_idx ON transactional_alerts (workspace_id, started_at DESC);
CREATE UNIQUE INDEX transactional_alerts_open_key ON transactional_alerts (workspace_id, kind, coalesce(template_id, '00000000-0000-0000-0000-000000000000'::uuid))
    WHERE ended_at IS NULL;

-- Send-only, test-mode and SMTP keys: an API key whose only permission is
-- transactional:send. preset says which ("send_only", "smtp"); mode
-- "test" delivers only to the workspace's members and the allowed test
-- addresses (patterns such as qa+*@acme.io). templates limits the keys to
-- some template IDs, categories an SMTP credential to some categories.
-- rate_per_second is the key's send rate.
ALTER TABLE api_keys
    ADD COLUMN preset          text    NOT NULL DEFAULT '' CHECK (preset IN ('','send_only','smtp')),
    ADD COLUMN mode            text    NOT NULL DEFAULT 'live' CHECK (mode IN ('live','test')),
    ADD COLUMN templates       text[]  NOT NULL DEFAULT '{}',
    ADD COLUMN categories      uuid[]  NOT NULL DEFAULT '{}',
    ADD COLUMN test_addresses  text[]  NOT NULL DEFAULT '{}',
    ADD COLUMN rate_per_second integer NOT NULL DEFAULT 50 CHECK (rate_per_second BETWEEN 1 AND 10000);

-- The SMTP relay (Instance admin › Instance settings): off until an
-- instance admin turns it on. One row. The certificate comes from Let's
-- Encrypt (ACME, cached in smtp_relay_acme sealed) or from files.
-- status: the last check of DNS, the ports and the certificate.
CREATE TABLE smtp_relay (
    id             boolean     PRIMARY KEY DEFAULT true CHECK (id),
    enabled        boolean     NOT NULL DEFAULT false,
    hostname       text        NOT NULL DEFAULT '',
    starttls_port  integer     NOT NULL DEFAULT 587 CHECK (starttls_port BETWEEN 1 AND 65535),
    tls_port       integer     NOT NULL DEFAULT 465 CHECK (tls_port BETWEEN 1 AND 65535),
    cert_source    text        NOT NULL DEFAULT 'acme' CHECK (cert_source IN ('acme','files')),
    cert_file      text        NOT NULL DEFAULT '',
    key_file       text        NOT NULL DEFAULT '',
    -- {"checked_at":"…","dns":{"ok":true,"detail":"…"},"starttls":{…},"tls":{…},"cert":{"ok":true,"not_after":"…","issuer":"…"}}
    status         jsonb       NOT NULL DEFAULT '{}',
    cert_not_after timestamptz,
    cert_issuer    text        NOT NULL DEFAULT '',
    renew_error    text        NOT NULL DEFAULT '',
    renew_failed_at timestamptz,
    enabled_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    enabled_at     timestamptz,
    unreachable_since timestamptz,
    updated_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at     timestamptz NOT NULL DEFAULT now()
);
INSERT INTO smtp_relay (id) VALUES (true);

-- Let's Encrypt account key and certificates for the relay (autocert's
-- cache), sealed, shared by every process that runs the relay.
CREATE TABLE smtp_relay_acme (
    key        text        PRIMARY KEY,
    sealed     bytea       NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE smtp_relay_acme;
DROP TABLE smtp_relay;
ALTER TABLE api_keys
    DROP COLUMN rate_per_second,
    DROP COLUMN test_addresses,
    DROP COLUMN categories,
    DROP COLUMN templates,
    DROP COLUMN mode,
    DROP COLUMN preset;
DROP TABLE transactional_alerts;
DROP TABLE transactional_opt_outs;
DROP TABLE transactional_rejected_calls;
DROP TABLE transactional_idempotency;
DROP TABLE transactional_outbox;
DROP TABLE transactional_message_content;
DROP TABLE transactional_messages;
DROP TABLE transactional_template_versions;
DROP TABLE transactional_template_drafts;
DROP TABLE transactional_templates;
DROP TABLE transactional_categories;
DROP TABLE transactional_settings;
ALTER TABLE sending_domains DROP COLUMN transactional;
DROP INDEX messages_transactional_outbox_idx;
ALTER TABLE stream_policies DROP COLUMN priority;
DELETE FROM messages WHERE stream = 'transactional';
DELETE FROM stream_failovers WHERE stream = 'transactional';
DELETE FROM stream_policies WHERE stream = 'transactional';
DELETE FROM routing_rules WHERE stream = 'transactional';
ALTER TABLE messages DROP CONSTRAINT messages_stream_check;
ALTER TABLE messages ADD CONSTRAINT messages_stream_check CHECK (stream IN ('broadcast','automation','system'));
ALTER TABLE stream_failovers DROP CONSTRAINT stream_failovers_stream_check;
ALTER TABLE stream_failovers ADD CONSTRAINT stream_failovers_stream_check CHECK (stream IN ('broadcast','automation','system'));
ALTER TABLE stream_policies DROP CONSTRAINT stream_policies_stream_check;
ALTER TABLE stream_policies ADD CONSTRAINT stream_policies_stream_check CHECK (stream IN ('broadcast','automation','system'));
ALTER TABLE routing_rules DROP CONSTRAINT routing_rules_stream_check;
ALTER TABLE routing_rules ADD CONSTRAINT routing_rules_stream_check CHECK (stream IN ('broadcast','automation','system'));
