-- +goose Up
-- WhatsApp (design expansion phase 8, M32, optional): one WhatsApp Business
-- Account per workspace through Meta's Cloud API, its message templates as
-- Meta approved them, Meta's rate card, WhatsApp campaigns, the messages
-- people send (keywords, replies, quick replies) that open the 24-hour
-- customer service window, and Meta's raw webhook events.
--
-- WhatsApp builds on X21's channel model: consent per list is
-- channel_consents with channel 'whatsapp' (and its evidence in
-- channel_consent_events), suppressions are channel_suppressions, and every
-- message is a row of the message log (channel_messages).

-- The account, with its credentials sealed: the business account and
-- phone number ids, a system user's access token and the app secret.
CREATE TABLE whatsapp_accounts (
    id                  uuid PRIMARY KEY,
    workspace_id        uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    credentials_enc     bytea       NOT NULL,
    -- The safe parts, shown in place of the credentials.
    business_hint       text        NOT NULL DEFAULT '',
    -- As Meta reports them, refreshed every hour and on demand.
    business_name       text        NOT NULL DEFAULT '',
    phone_number        text        NOT NULL DEFAULT '',
    display_name        text        NOT NULL DEFAULT '',
    -- approved, pending, declined, available or none.
    name_status         text        NOT NULL DEFAULT '',
    -- connected, pending, flagged, restricted, disconnected.
    phone_status        text        NOT NULL DEFAULT '',
    quality             text        NOT NULL DEFAULT 'unknown' CHECK (quality IN ('high','medium','low','unknown')),
    previous_quality    text        NOT NULL DEFAULT '',
    quality_changed_at  timestamptz,
    -- People a day the business can start conversations with (0: unlimited).
    messaging_limit     integer     NOT NULL DEFAULT 0,
    previous_limit      integer     NOT NULL DEFAULT 0,
    limit_changed_at    timestamptz,
    -- The webhook: Meta's handshake proves the address and verify token
    -- were saved in Meta's app settings.
    verify_token        text        NOT NULL,
    verified_at         timestamptz,
    last_event_at       timestamptz,
    -- waiting: for Meta's verification request; connected; token_rejected:
    -- Meta refused the access token (sends are held until it's replaced).
    status              text        NOT NULL DEFAULT 'waiting' CHECK (status IN ('waiting','connected','token_rejected')),
    status_detail       text        NOT NULL DEFAULT '',
    status_since        timestamptz,
    templates_synced_at timestamptz,
    checked_at          timestamptz,
    -- The demo's account answers from an in-process Graph API.
    demo                boolean     NOT NULL DEFAULT false,
    connected_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    connected_at        timestamptz NOT NULL,
    removed_at          timestamptz,
    removed_by          uuid        REFERENCES users(id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX whatsapp_accounts_live_idx ON whatsapp_accounts (workspace_id) WHERE removed_at IS NULL;

-- Message templates, synced from Meta every hour and on demand. They're
-- created and edited in Meta's WhatsApp Manager; ReputeCast reads them.
CREATE TABLE whatsapp_templates (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    account_id   uuid        NOT NULL REFERENCES whatsapp_accounts(id) ON DELETE CASCADE,
    meta_id      text        NOT NULL DEFAULT '',
    name         text        NOT NULL,
    -- Meta's locale: en_US.
    language     text        NOT NULL,
    category     text        NOT NULL CHECK (category IN ('marketing','utility','authentication')),
    status       text        NOT NULL CHECK (status IN ('approved','pending','rejected','paused','disabled','in_appeal','other')),
    status_since timestamptz,
    -- Meta's reason for a rejection: its code and its words.
    reason_code  text        NOT NULL DEFAULT '',
    reason       text        NOT NULL DEFAULT '',
    body         text        NOT NULL DEFAULT '',
    -- [{"key":"1","label":"first name","example":"Sam"}]
    variables    jsonb       NOT NULL DEFAULT '[]',
    -- [{"type":"URL","text":"View the agenda","url":"…"}]
    buttons      jsonb       NOT NULL DEFAULT '[]',
    synced_at    timestamptz NOT NULL,
    removed_at   timestamptz
);
CREATE UNIQUE INDEX whatsapp_templates_key ON whatsapp_templates (account_id, name, language) WHERE removed_at IS NULL;

-- Meta charges per delivered template message, by category and the
-- recipient's country. ReputeCast estimates costs from Meta's rate card
-- and the workspace can edit it; country '*' is every other country.
CREATE TABLE whatsapp_rates (
    workspace_id uuid          NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    country      text          NOT NULL,
    category     text          NOT NULL CHECK (category IN ('marketing','utility','authentication')),
    price        numeric(10,4) NOT NULL CHECK (price >= 0),
    PRIMARY KEY (workspace_id, country, category)
);

-- The rate card's date and who last edited it. Whether WhatsApp keeps to
-- the quiet hours is sms_settings.whatsapp_same_window once the workspace
-- has SMS rules; quiet_hours holds the choice until then.
CREATE TABLE whatsapp_settings (
    workspace_id    uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    rates_as_of     date        NOT NULL,
    quiet_hours     boolean     NOT NULL DEFAULT true,
    rates_edited_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    rates_edited_at timestamptz,
    updated_at      timestamptz NOT NULL
);

-- What a WhatsApp campaign holds besides the campaign row: the approved
-- template, its variables mapped to contact fields with fallbacks, and the
-- schedule.
CREATE TABLE whatsapp_campaigns (
    campaign_id       uuid PRIMARY KEY REFERENCES campaigns(id) ON DELETE CASCADE,
    workspace_id      uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    template_id       uuid        REFERENCES whatsapp_templates(id) ON DELETE SET NULL,
    template_name     text        NOT NULL DEFAULT '',
    template_language text        NOT NULL DEFAULT '',
    -- [{"key":"1","source":"field","field":"first_name","fallback":"there"},
    --  {"key":"2","source":"text","text":"8:30"}]
    variables         jsonb       NOT NULL DEFAULT '[]',
    -- now: at once; at: one moment; local: a time in each recipient's zone.
    schedule_mode     text        NOT NULL DEFAULT 'local' CHECK (schedule_mode IN ('now','at','local')),
    send_at           timestamptz,
    local_date        date,
    local_minute      integer     CHECK (local_minute BETWEEN 0 AND 1439),
    -- People over the messaging limit: the next day from the window's
    -- opening, or skipped.
    over_limit        text        NOT NULL DEFAULT 'next_day' CHECK (over_limit IN ('next_day','skip')),
    version           integer     NOT NULL DEFAULT 1
);

-- A WhatsApp campaign's send.
CREATE TABLE whatsapp_sends (
    campaign_id   uuid PRIMARY KEY REFERENCES campaigns(id) ON DELETE CASCADE,
    workspace_id  uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    phase         text        NOT NULL CHECK (phase IN ('scheduled','sending','completed','cancelled')),
    recipients    integer     NOT NULL DEFAULT 0,
    left_out      jsonb       NOT NULL DEFAULT '{}',
    started_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    approved_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    scheduled_at  timestamptz NOT NULL,
    first_send_at timestamptz,
    last_send_at  timestamptz,
    completed_at  timestamptz
);

-- What only WhatsApp has, per message of the message log.
CREATE TABLE whatsapp_messages (
    message_id   uuid PRIMARY KEY REFERENCES channel_messages(id) ON DELETE CASCADE,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    account_id   uuid        REFERENCES whatsapp_accounts(id) ON DELETE SET NULL,
    -- template or text (free-form, inside the customer service window)
    kind         text        NOT NULL CHECK (kind IN ('template','text')),
    template     text        NOT NULL DEFAULT '',
    language     text        NOT NULL DEFAULT '',
    category     text        NOT NULL DEFAULT '',
    params       jsonb       NOT NULL DEFAULT '[]',
    read_at      timestamptz,
    -- Meta's billing for it, when its status reported one.
    billable     boolean
);
CREATE INDEX channel_messages_wamid_idx ON channel_messages (provider_message_id) WHERE channel = 'whatsapp' AND provider_message_id <> '';

-- What people send: keywords (the SMS keywords work on WhatsApp too),
-- quick replies and other replies. Each opens the 24-hour customer
-- service window; replies go on the contact's timeline (there's no inbox).
CREATE TABLE whatsapp_inbound (
    id               uuid PRIMARY KEY,
    workspace_id     uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    account_id       uuid        REFERENCES whatsapp_accounts(id) ON DELETE SET NULL,
    phone            text        NOT NULL,
    -- Their WhatsApp profile name.
    name             text        NOT NULL DEFAULT '',
    body             text        NOT NULL DEFAULT '',
    kind             text        NOT NULL CHECK (kind IN ('opt_out','opt_in','help','reply','button')),
    keyword          text        NOT NULL DEFAULT '',
    -- The message it answers (a quick reply's template, a reply's quote,
    -- else the last message the number got).
    message_id       uuid        REFERENCES channel_messages(id) ON DELETE SET NULL,
    campaign_id      uuid        REFERENCES campaigns(id) ON DELETE SET NULL,
    automation_id    uuid        REFERENCES automations(id) ON DELETE SET NULL,
    contact_ids      uuid[]      NOT NULL DEFAULT '{}',
    effect           jsonb       NOT NULL DEFAULT '{}',
    reply_message_id uuid        REFERENCES channel_messages(id) ON DELETE SET NULL,
    meta_id          text        NOT NULL DEFAULT '',
    raw_event_id     uuid,
    received_at      timestamptz NOT NULL
);
CREATE INDEX whatsapp_inbound_phone_idx ON whatsapp_inbound (workspace_id, phone, received_at DESC);
CREATE INDEX whatsapp_inbound_campaign_idx ON whatsapp_inbound (campaign_id, received_at) WHERE campaign_id IS NOT NULL;
CREATE UNIQUE INDEX whatsapp_inbound_meta_idx ON whatsapp_inbound (workspace_id, meta_id) WHERE meta_id <> '';

-- Meta's webhooks: verified with the app secret, stored as Meta sent them,
-- answered 200 at once and normalized by a River job. Kept 90 days.
CREATE TABLE whatsapp_raw_events (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    account_id      uuid        REFERENCES whatsapp_accounts(id) ON DELETE CASCADE,
    received_at     timestamptz NOT NULL,
    headers         jsonb       NOT NULL DEFAULT '{}',
    body            bytea       NOT NULL,
    signature_ok    boolean     NOT NULL,
    -- The raw event as the Events tab shows it ("statuses[0].status=delivered").
    summary         text        NOT NULL DEFAULT '',
    normalized_at   timestamptz,
    normalize_error text        NOT NULL DEFAULT '',
    events          integer     NOT NULL DEFAULT 0
);
CREATE INDEX whatsapp_raw_events_workspace_idx ON whatsapp_raw_events (workspace_id, received_at DESC);
CREATE INDEX whatsapp_raw_events_pending_idx ON whatsapp_raw_events (received_at) WHERE normalized_at IS NULL;

-- +goose Down
DROP TABLE whatsapp_raw_events;
DROP TABLE whatsapp_inbound;
DROP INDEX channel_messages_wamid_idx;
DROP TABLE whatsapp_messages;
DROP TABLE whatsapp_sends;
DROP TABLE whatsapp_campaigns;
DROP TABLE whatsapp_settings;
DROP TABLE whatsapp_rates;
DROP TABLE whatsapp_templates;
DROP TABLE whatsapp_accounts;
