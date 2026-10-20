-- +goose Up
-- SMS (design expansion phase 8, M31): SMS providers and their senders with
-- the carriers' registration, routing by country, the SMS rules (keywords,
-- replies, quiet hours, the brand prefix and opt-out text, the budget and
-- auto-pause thresholds, the short-link domain, the text message terms),
-- phone numbers on contacts, consent per list and per channel with its
-- evidence, the SMS suppression list, SMS campaigns, the message log with
-- its events, inbound replies and keywords, short links and their clicks,
-- and the raw provider events.
--
-- The channel model is shared: campaigns get a channel, and consent,
-- suppressions and the message log carry one, so push and WhatsApp (X22)
-- add rows rather than tables. Email keeps its own tables untouched.
--
-- Texts, receipts, replies and clicks stay in their own logs for 90 days
-- (the ROADMAP's default), never in the audit log; everything a person or a
-- key changes is audited.

-- A phone number per contact, in E.164, with the country it belongs to and
-- what the line is known to be. The time zone stays the contact's own field
-- (contacts.timezone); without one, quiet hours read the number's.
ALTER TABLE contacts
    ADD COLUMN phone             text NOT NULL DEFAULT '',
    ADD COLUMN phone_country     text NOT NULL DEFAULT '',
    ADD COLUMN phone_line_type   text NOT NULL DEFAULT 'unknown'
        CHECK (phone_line_type IN ('mobile','landline','toll_free','voip','unknown')),
    -- number_plan: libphonenumber; provider: a provider's error (landline);
    -- receipt: a text was delivered to it.
    ADD COLUMN phone_line_source text NOT NULL DEFAULT ''
        CHECK (phone_line_source IN ('','number_plan','provider','receipt')),
    ADD COLUMN phone_set_at      timestamptz;
-- Two contacts can share a number: a STOP applies to both.
CREATE INDEX contacts_phone_idx ON contacts (workspace_id, phone) WHERE phone <> '';

-- The channel of a campaign. SMS campaigns are Regular only; push and
-- WhatsApp campaigns (X22) use the same column.
ALTER TABLE campaigns
    ADD COLUMN channel text NOT NULL DEFAULT 'email' CHECK (channel IN ('email','sms','push','whatsapp'));
CREATE INDEX campaigns_channel_idx ON campaigns (workspace_id, channel) WHERE deleted_at IS NULL AND channel <> 'email';

-- Consent per list and per channel. Being on a list by email says nothing
-- about texting the same person: SMS (and WhatsApp) consent is its own row.
-- pending: the person has to reply YES first (off by default).
CREATE TABLE channel_consents (
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    list_id      uuid        NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    contact_id   uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    channel      text        NOT NULL CHECK (channel IN ('sms','whatsapp')),
    status       text        NOT NULL CHECK (status IN ('subscribed','pending','opted_out','not_subscribed')),
    -- How the last change happened (form:<slug>, preference_center,
    -- account_page, import:<source>, api, team, keyword).
    source       text        NOT NULL DEFAULT '',
    -- The word that opted the number out ("STOP").
    keyword      text        NOT NULL DEFAULT '',
    -- Set when a STOP withdrew this consent: START brings back exactly these.
    stopped_at   timestamptz,
    changed_at   timestamptz NOT NULL,
    PRIMARY KEY (list_id, contact_id, channel)
);
CREATE INDEX channel_consents_contact_idx ON channel_consents (contact_id, channel);
CREATE INDEX channel_consents_list_idx ON channel_consents (list_id, channel, status);

-- The evidence of every consent change, kept like email consent: the
-- number, the list, the exact consent text and its version, the page or
-- source, the IP address and the time.
CREATE TABLE channel_consent_events (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    contact_id      uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    list_id         uuid        REFERENCES lists(id) ON DELETE SET NULL,
    channel         text        NOT NULL CHECK (channel IN ('sms','whatsapp')),
    action          text        NOT NULL CHECK (action IN ('subscribed','pending','confirmed','opted_out','opted_in','unsubscribed','imported','confirmation_sent','confirmation_failed')),
    phone           text        NOT NULL DEFAULT '',
    source          text        NOT NULL DEFAULT '',
    -- The page (news.acme.io/summit) or what stood for it ("Opt-in confirmation").
    page            text        NOT NULL DEFAULT '',
    consent_text    text        NOT NULL DEFAULT '',
    consent_version text        NOT NULL DEFAULT '',
    keyword         text        NOT NULL DEFAULT '',
    ip              inet,
    user_agent      text        NOT NULL DEFAULT '',
    actor_id        uuid        REFERENCES users(id) ON DELETE SET NULL,
    import_id       uuid,
    note            text        NOT NULL DEFAULT '',
    occurred_at     timestamptz NOT NULL
);
CREATE INDEX channel_consent_events_contact_idx ON channel_consent_events (contact_id, occurred_at DESC);
CREATE INDEX channel_consent_events_workspace_idx ON channel_consent_events (workspace_id, occurred_at DESC);

-- SMS providers: one connection per account, with write-only credentials.
-- Kept apart from email providers so no email screen or job ever sees them.
CREATE TABLE sms_providers (
    id                  uuid PRIMARY KEY,
    workspace_id        uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    adapter             text        NOT NULL,
    name                text        NOT NULL,
    -- "Acme Inc, live" and the safe part of the account id or key ("AC…3f9").
    account_label       text        NOT NULL DEFAULT '',
    account_hint        text        NOT NULL DEFAULT '',
    -- Non-secret settings (region, account id, live or test).
    config              jsonb       NOT NULL DEFAULT '{}',
    credentials_enc     bytea,
    -- /hooks/sms/{adapter}/{token}; the secret that signs the provider's
    -- requests where the provider signs with one we hold (sealed).
    webhook_token       text        NOT NULL UNIQUE,
    webhook_secret_enc  bytea,
    -- registered: through the provider's API; manual: an address and a
    -- secret pasted in the provider's console.
    webhooks            text        NOT NULL DEFAULT 'registered' CHECK (webhooks IN ('registered','manual')),
    status              text        NOT NULL DEFAULT 'healthy' CHECK (status IN ('healthy','degraded','down','credentials_rejected')),
    status_detail       text        NOT NULL DEFAULT '',
    status_since        timestamptz,
    -- {"lastError":…,"consecutiveFailures":n,"answer":"401 Unauthorized · invalid credentials"}
    health              jsonb       NOT NULL DEFAULT '{}',
    last_receipt_at     timestamptz,
    last_inbound_at     timestamptz,
    last_checked_at     timestamptz,
    registrations_read_at timestamptz,
    -- The demo's sample providers answer from an in-process fake.
    demo                boolean     NOT NULL DEFAULT false,
    connected_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    connected_at        timestamptz NOT NULL,
    removed_at          timestamptz
);
CREATE INDEX sms_providers_workspace_idx ON sms_providers (workspace_id) WHERE removed_at IS NULL;

-- The wizard's SMS path: provider, connection, senders, test text, countries.
CREATE TABLE sms_setups (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    adapter      text        NOT NULL,
    provider_id  uuid        REFERENCES sms_providers(id) ON DELETE SET NULL,
    step         text        NOT NULL DEFAULT 'connection' CHECK (step IN ('connection','senders','test','countries','done')),
    status       text        NOT NULL DEFAULT 'open' CHECK (status IN ('open','finished','abandoned')),
    state        jsonb       NOT NULL DEFAULT '{}',
    created_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL,
    updated_at   timestamptz NOT NULL
);

-- What a text appears to come from: a toll-free or 10DLC number, another
-- number, or an alphanumeric sender ID.
CREATE TABLE sms_senders (
    id           uuid PRIMARY KEY,
    workspace_id uuid          NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    provider_id  uuid          NOT NULL REFERENCES sms_providers(id) ON DELETE CASCADE,
    kind         text          NOT NULL CHECK (kind IN ('toll_free','10dlc','local','mobile','sender_id')),
    -- E.164 for numbers, the ID itself for sender IDs ("ACME").
    value        text          NOT NULL,
    countries    text[]        NOT NULL DEFAULT '{}',
    -- Texts per second, as the provider reports it.
    throughput   numeric(8,2)  NOT NULL DEFAULT 1,
    -- The provider's own id for the number (Twilio's PN…).
    provider_ref text          NOT NULL DEFAULT '',
    added_by     uuid          REFERENCES users(id) ON DELETE SET NULL,
    added_at     timestamptz   NOT NULL,
    removed_at   timestamptz
);
CREATE INDEX sms_senders_workspace_idx ON sms_senders (workspace_id) WHERE removed_at IS NULL;
CREATE UNIQUE INDEX sms_senders_key ON sms_senders (provider_id, value) WHERE removed_at IS NULL;

-- The carriers' approval of a sender: toll-free verification, a 10DLC brand
-- and campaign, or a country's sender ID registration. country '' covers
-- the whole sender. Read from the provider every hour where its API allows,
-- otherwise set by hand with a date.
CREATE TABLE sms_registrations (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    sender_id    uuid        NOT NULL REFERENCES sms_senders(id) ON DELETE CASCADE,
    kind         text        NOT NULL CHECK (kind IN ('toll_free','brand','campaign','sender_id')),
    country      text        NOT NULL DEFAULT '',
    -- The brand ("Acme Inc") or the campaign ("Acme Summit attendee updates").
    name         text        NOT NULL DEFAULT '',
    status       text        NOT NULL CHECK (status IN ('not_registered','pending','verified','approved','rejected','allowed','not_supported')),
    -- The carrier's reason for a rejection, as the provider reports it.
    reason       text        NOT NULL DEFAULT '',
    status_on    date,
    source       text        NOT NULL DEFAULT 'api' CHECK (source IN ('api','manual','rule')),
    -- What was submitted: use case, monthly volume, opt-in, terms, by whom.
    details      jsonb       NOT NULL DEFAULT '{}',
    set_by       uuid        REFERENCES users(id) ON DELETE SET NULL,
    read_at      timestamptz,
    updated_at   timestamptz NOT NULL,
    UNIQUE (sender_id, kind, country)
);

-- Routes from countries to a sender, with an optional fallback sender that
-- is registered for the same countries. The "every other country" rule is
-- in sms_settings.
CREATE TABLE sms_routes (
    id                 uuid PRIMARY KEY,
    workspace_id       uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    position           integer     NOT NULL,
    countries          text[]      NOT NULL,
    sender_id          uuid        NOT NULL REFERENCES sms_senders(id) ON DELETE CASCADE,
    fallback_sender_id uuid        REFERENCES sms_senders(id) ON DELETE SET NULL,
    created_at         timestamptz NOT NULL,
    updated_at         timestamptz NOT NULL
);
CREATE INDEX sms_routes_workspace_idx ON sms_routes (workspace_id, position);

-- The price of one segment per provider and country: read from the
-- provider's pricing where it has an API, otherwise entered by hand.
CREATE TABLE sms_prices (
    workspace_id uuid          NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    provider_id  uuid          NOT NULL REFERENCES sms_providers(id) ON DELETE CASCADE,
    country      text          NOT NULL,
    price        numeric(10,4) NOT NULL,
    source       text          NOT NULL DEFAULT 'api' CHECK (source IN ('api','manual')),
    as_of        date          NOT NULL,
    PRIMARY KEY (provider_id, country)
);

-- The workspace's SMS rules (one row; created with the first provider).
CREATE TABLE sms_settings (
    workspace_id          uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    brand_prefix          text          NOT NULL DEFAULT '',
    opt_out_text          text          NOT NULL DEFAULT 'Reply STOP to opt out',
    -- Always on campaigns; on by default in automation steps.
    opt_out_in_steps      boolean       NOT NULL DEFAULT true,
    confirmation_on       boolean       NOT NULL DEFAULT true,
    confirmation_text     text          NOT NULL DEFAULT '',
    require_yes           boolean       NOT NULL DEFAULT false,
    -- Quiet hours, minutes after local midnight; never outside 08:00–21:00.
    quiet_from            integer       NOT NULL DEFAULT 540 CHECK (quiet_from >= 480),
    quiet_to              integer       NOT NULL DEFAULT 1200 CHECK (quiet_to <= 1260),
    push_same_window      boolean       NOT NULL DEFAULT true,
    whatsapp_same_window  boolean       NOT NULL DEFAULT true,
    opt_out_keywords      text[]        NOT NULL DEFAULT '{STOP,STOPALL,UNSUBSCRIBE,CANCEL,END,QUIT,OPTOUT,REVOKE}',
    opt_in_keywords       text[]        NOT NULL DEFAULT '{START,UNSTOP}',
    help_keywords         text[]        NOT NULL DEFAULT '{HELP,INFO}',
    help_reply            text          NOT NULL DEFAULT '',
    opt_out_reply         text          NOT NULL DEFAULT '',
    opt_in_reply          text          NOT NULL DEFAULT '',
    -- reputecast: ReputeCast answers keywords; provider: the provider does
    -- (its opt-out list is synced and ReputeCast sends no reply of its own).
    keywords_by           text          NOT NULL DEFAULT 'reputecast' CHECK (keywords_by IN ('reputecast','provider')),
    -- The month's budget in the workspace's currency (USD), the alert
    -- level and who set it.
    budget                numeric(12,2) NOT NULL DEFAULT 0,
    budget_alert_pct      integer       NOT NULL DEFAULT 80 CHECK (budget_alert_pct BETWEEN 1 AND 100),
    budget_set_by         uuid          REFERENCES users(id) ON DELETE SET NULL,
    budget_set_at         timestamptz,
    pause_failed_pct      numeric(5,2)  NOT NULL DEFAULT 5,
    pause_filtered_pct    numeric(5,2)  NOT NULL DEFAULT 2,
    pause_opt_out_pct     numeric(5,2)  NOT NULL DEFAULT 3,
    -- Every other country: don't send, or a route's sender.
    other_countries       text          NOT NULL DEFAULT 'dont_send' CHECK (other_countries IN ('dont_send','route')),
    other_route_id        uuid          REFERENCES sms_routes(id) ON DELETE SET NULL,
    -- How long the text of each message is kept (0 to 90 days).
    retention_days        integer       NOT NULL DEFAULT 30 CHECK (retention_days BETWEEN 0 AND 90),
    -- The short-link domain (go.acme.io): a CNAME to the instance.
    short_domain          text          NOT NULL DEFAULT '',
    short_domain_status   text          NOT NULL DEFAULT 'none' CHECK (short_domain_status IN ('none','pending','verified','failed')),
    short_domain_target   text          NOT NULL DEFAULT '',
    short_domain_checked_at  timestamptz,
    short_domain_verified_at timestamptz,
    short_domain_detail   text          NOT NULL DEFAULT '',
    updated_by            uuid          REFERENCES users(id) ON DELETE SET NULL,
    updated_at            timestamptz   NOT NULL
);

-- Members' phones for test texts.
CREATE TABLE sms_test_numbers (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id      uuid        REFERENCES users(id) ON DELETE CASCADE,
    phone        text        NOT NULL,
    added_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    added_at     timestamptz NOT NULL,
    UNIQUE (workspace_id, phone)
);

-- The text message terms page, generated from the SMS rules and editable.
-- Each published version is kept: consent records name the version that
-- was live.
CREATE TABLE sms_terms_versions (
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    version      integer     NOT NULL,
    content      jsonb       NOT NULL,
    published_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    published_at timestamptz NOT NULL,
    PRIMARY KEY (workspace_id, version)
);

-- Numbers that must get no texts (or WhatsApp messages), shared by every
-- provider of the channel. A number that opted out by text can't be taken
-- off by hand: only the person can come back, by texting START.
CREATE TABLE channel_suppressions (
    id                 uuid PRIMARY KEY,
    workspace_id       uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    channel            text        NOT NULL CHECK (channel IN ('sms','whatsapp')),
    phone              text        NOT NULL,
    reason             text        NOT NULL CHECK (reason IN ('opted_out','landline','not_in_service','invalid','filtered','provider_list','manual')),
    keyword            text        NOT NULL DEFAULT '',
    -- The provider's own code ("Twilio 30005") or the word ("STOP").
    original_code      text        NOT NULL DEFAULT '',
    source             text        NOT NULL DEFAULT '',
    source_provider_id uuid        REFERENCES sms_providers(id) ON DELETE SET NULL,
    campaign_id        uuid        REFERENCES campaigns(id) ON DELETE SET NULL,
    automation_id      uuid        REFERENCES automations(id) ON DELETE SET NULL,
    -- What the number was sent when it happened ("Opt-in confirmation").
    context            text        NOT NULL DEFAULT '',
    created_by         uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at         timestamptz NOT NULL,
    removed_at         timestamptz,
    removed_by         uuid        REFERENCES users(id) ON DELETE SET NULL,
    removed_reason     text        NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX channel_suppressions_key ON channel_suppressions (workspace_id, channel, phone) WHERE removed_at IS NULL;
CREATE INDEX channel_suppressions_list_idx ON channel_suppressions (workspace_id, channel, created_at DESC) WHERE removed_at IS NULL;

-- Opt-out sync with each provider's own opt-out list.
CREATE TABLE sms_opt_out_sync (
    provider_id  uuid PRIMARY KEY REFERENCES sms_providers(id) ON DELETE CASCADE,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    mode         text        NOT NULL CHECK (mode IN ('two_way','read_only','off')),
    pulled       integer     NOT NULL DEFAULT 0,
    pushed       integer     NOT NULL DEFAULT 0,
    status       text        NOT NULL DEFAULT 'in_sync' CHECK (status IN ('in_sync','pending','failing')),
    detail       text        NOT NULL DEFAULT '',
    last_sync_at timestamptz
);

-- What an SMS campaign holds besides the campaign row: the text between the
-- locked parts, when it goes out, and its own auto-pause thresholds.
CREATE TABLE sms_campaigns (
    campaign_id    uuid PRIMARY KEY REFERENCES campaigns(id) ON DELETE CASCADE,
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    body           text        NOT NULL DEFAULT '',
    -- now: at once, holding whoever is outside their window; at: one moment
    -- for everyone; local: a time in each recipient's time zone.
    schedule_mode  text        NOT NULL DEFAULT 'local' CHECK (schedule_mode IN ('now','at','local')),
    send_at        timestamptz,
    local_date     date,
    local_minute   integer CHECK (local_minute BETWEEN 0 AND 1439),
    -- Recipients whose sender is pending in their country: skip them, or
    -- hold them for up to 7 days until it's approved.
    pending_policy text        NOT NULL DEFAULT 'skip' CHECK (pending_policy IN ('skip','wait')),
    -- {"failed":5,"filtered":2,"optOuts":3}; empty uses the workspace's.
    thresholds     jsonb       NOT NULL DEFAULT '{}'
);

CREATE TABLE sms_campaign_versions (
    campaign_id uuid        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    version     integer     NOT NULL,
    body        text        NOT NULL,
    saved_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    saved_at    timestamptz NOT NULL,
    PRIMARY KEY (campaign_id, version)
);

-- A campaign's send: its state, what tripped an auto-pause, and holds.
CREATE TABLE sms_sends (
    campaign_id   uuid PRIMARY KEY REFERENCES campaigns(id) ON DELETE CASCADE,
    workspace_id  uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    phase         text        NOT NULL CHECK (phase IN ('scheduled','sending','paused','completed','cancelled')),
    recipients    integer     NOT NULL DEFAULT 0,
    skipped       integer     NOT NULL DEFAULT 0,
    left_out      jsonb       NOT NULL DEFAULT '{}',
    started_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    approved_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    scheduled_at  timestamptz NOT NULL,
    first_send_at timestamptz,
    last_send_at  timestamptz,
    -- auto: the thresholds; person; outage; budget.
    paused_kind   text        NOT NULL DEFAULT '' CHECK (paused_kind IN ('','auto','person','outage','budget')),
    paused_at     timestamptz,
    pause         jsonb       NOT NULL DEFAULT '{}',
    thresholds    jsonb       NOT NULL DEFAULT '{}',
    completed_at  timestamptz
);

-- The message log: every text (and later every notification and WhatsApp
-- message) with its route, cost and status. Metadata stays 90 days; the
-- text itself goes after the workspace's retention (30 days by default).
CREATE TABLE channel_messages (
    id                  uuid PRIMARY KEY,
    workspace_id        uuid          NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    channel             text          NOT NULL CHECK (channel IN ('sms','push','whatsapp')),
    kind                text          NOT NULL CHECK (kind IN ('campaign','automation','test','confirmation','reply')),
    campaign_id         uuid          REFERENCES campaigns(id) ON DELETE CASCADE,
    automation_id       uuid          REFERENCES automations(id) ON DELETE CASCADE,
    automation_step     text          NOT NULL DEFAULT '',
    run_id              uuid,
    contact_id          uuid          REFERENCES contacts(id) ON DELETE SET NULL,
    list_id             uuid          REFERENCES lists(id) ON DELETE SET NULL,
    -- E.164 for texts and WhatsApp; a subscription for push.
    recipient           text          NOT NULL,
    country             text          NOT NULL DEFAULT '',
    time_zone           text          NOT NULL DEFAULT '',
    zone_source         text          NOT NULL DEFAULT '',
    provider_id         uuid          REFERENCES sms_providers(id) ON DELETE SET NULL,
    sender_id           uuid          REFERENCES sms_senders(id) ON DELETE SET NULL,
    sender              text          NOT NULL DEFAULT '',
    body                text,
    -- Segments for texts.
    units               integer       NOT NULL DEFAULT 1,
    encoding            text          NOT NULL DEFAULT '',
    price               numeric(10,4),
    cost                numeric(12,4) NOT NULL DEFAULT 0,
    status              text          NOT NULL CHECK (status IN ('queued','held','skipped','sending','sent','delivered','undelivered','failed','cancelled')),
    -- Why it's held, skipped or failed: unreachable, filtered, not_in_service,
    -- landline, invalid, quiet_hours, no_sender, pending_sender, provider_down,
    -- budget, suppressed, no_consent, recent_text…
    reason              text          NOT NULL DEFAULT '',
    provider_code       text          NOT NULL DEFAULT '',
    provider_message_id text          NOT NULL DEFAULT '',
    -- Quiet hours and local times: not before; held texts expire at the
    -- end of the recipient's window.
    send_after          timestamptz,
    expires_at          timestamptz,
    attempts            integer       NOT NULL DEFAULT 0,
    created_at          timestamptz   NOT NULL,
    sent_at             timestamptz,
    delivered_at        timestamptz,
    failed_at           timestamptz,
    updated_at          timestamptz   NOT NULL
);
CREATE INDEX channel_messages_campaign_idx ON channel_messages (campaign_id, status) WHERE campaign_id IS NOT NULL;
CREATE INDEX channel_messages_automation_idx ON channel_messages (automation_id, automation_step, created_at) WHERE automation_id IS NOT NULL;
CREATE INDEX channel_messages_due_idx ON channel_messages (send_after) WHERE status = 'queued';
CREATE INDEX channel_messages_contact_idx ON channel_messages (contact_id, created_at DESC) WHERE contact_id IS NOT NULL;
CREATE INDEX channel_messages_recipient_idx ON channel_messages (workspace_id, recipient, created_at DESC);
CREATE INDEX channel_messages_month_idx ON channel_messages (workspace_id, channel, created_at);
CREATE UNIQUE INDEX channel_messages_provider_idx ON channel_messages (provider_id, provider_message_id) WHERE provider_message_id <> '';

-- What happened to each message: receipts, clicks, replies and keywords.
CREATE TABLE channel_message_events (
    id            uuid PRIMARY KEY,
    workspace_id  uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    message_id    uuid        NOT NULL REFERENCES channel_messages(id) ON DELETE CASCADE,
    kind          text        NOT NULL CHECK (kind IN ('queued','sent','delivered','undelivered','failed','clicked','replied','opted_out','opted_in','help')),
    provider_code text        NOT NULL DEFAULT '',
    detail        jsonb       NOT NULL DEFAULT '{}',
    raw_event_id  uuid,
    machine       boolean     NOT NULL DEFAULT false,
    occurred_at   timestamptz NOT NULL
);
CREATE INDEX channel_message_events_message_idx ON channel_message_events (message_id, occurred_at);

-- Texts people send back: keywords and other replies. Other replies go on
-- the contact's timeline and into the campaign report; there's no inbox.
CREATE TABLE sms_inbound (
    id               uuid PRIMARY KEY,
    workspace_id     uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    provider_id      uuid        REFERENCES sms_providers(id) ON DELETE SET NULL,
    phone            text        NOT NULL,
    sender           text        NOT NULL DEFAULT '',
    body             text        NOT NULL DEFAULT '',
    kind             text        NOT NULL CHECK (kind IN ('opt_out','opt_in','help','yes','reply')),
    keyword          text        NOT NULL DEFAULT '',
    -- The last text this number got, and its campaign or flow.
    message_id       uuid        REFERENCES channel_messages(id) ON DELETE SET NULL,
    campaign_id      uuid        REFERENCES campaigns(id) ON DELETE SET NULL,
    automation_id    uuid        REFERENCES automations(id) ON DELETE SET NULL,
    contact_ids      uuid[]      NOT NULL DEFAULT '{}',
    -- What ReputeCast did: lists whose consent changed, the reply it sent.
    effect           jsonb       NOT NULL DEFAULT '{}',
    reply_message_id uuid        REFERENCES channel_messages(id) ON DELETE SET NULL,
    raw_event_id     uuid,
    received_at      timestamptz NOT NULL
);
CREATE INDEX sms_inbound_workspace_idx ON sms_inbound (workspace_id, received_at DESC);
CREATE INDEX sms_inbound_campaign_idx ON sms_inbound (campaign_id, received_at) WHERE campaign_id IS NOT NULL;
CREATE INDEX sms_inbound_phone_idx ON sms_inbound (workspace_id, phone, received_at DESC);

-- Short links on the workspace's own domain: a code per recipient, kept 12
-- months, redirecting with UTM parameters.
CREATE TABLE sms_links (
    code          text PRIMARY KEY,
    workspace_id  uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    message_id    uuid        REFERENCES channel_messages(id) ON DELETE SET NULL,
    campaign_id   uuid        REFERENCES campaigns(id) ON DELETE CASCADE,
    automation_id uuid        REFERENCES automations(id) ON DELETE CASCADE,
    contact_id    uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    -- The address as written and where the link leads (with UTM parameters).
    original      text        NOT NULL,
    target        text        NOT NULL,
    created_at    timestamptz NOT NULL,
    expires_at    timestamptz NOT NULL
);
CREATE INDEX sms_links_campaign_idx ON sms_links (campaign_id) WHERE campaign_id IS NOT NULL;

-- Clicks: people apart from link previews (messaging apps fetch links to
-- draw a preview; those are machine clicks, never counted as people).
CREATE TABLE sms_link_clicks (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    code         text        NOT NULL REFERENCES sms_links(code) ON DELETE CASCADE,
    campaign_id  uuid,
    message_id   uuid,
    contact_id   uuid,
    machine      boolean     NOT NULL,
    user_agent   text        NOT NULL DEFAULT '',
    occurred_at  timestamptz NOT NULL
);
CREATE INDEX sms_link_clicks_campaign_idx ON sms_link_clicks (campaign_id, occurred_at) WHERE campaign_id IS NOT NULL;

-- Provider webhooks: verified, stored raw, answered at once and normalized
-- later by a River job. Kept 90 days, like email provider events.
CREATE TABLE sms_raw_events (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    provider_id     uuid        REFERENCES sms_providers(id) ON DELETE CASCADE,
    kind            text        NOT NULL DEFAULT 'receipt' CHECK (kind IN ('receipt','inbound','test','other')),
    received_at     timestamptz NOT NULL,
    content_type    text        NOT NULL DEFAULT '',
    headers         jsonb       NOT NULL DEFAULT '{}',
    body            bytea       NOT NULL,
    signature_ok    boolean     NOT NULL,
    -- The raw event as the Events tab shows it ("MessageStatus=delivered").
    summary         text        NOT NULL DEFAULT '',
    normalized_at   timestamptz,
    normalize_error text        NOT NULL DEFAULT '',
    events          integer     NOT NULL DEFAULT 0
);
CREATE INDEX sms_raw_events_workspace_idx ON sms_raw_events (workspace_id, received_at DESC);
CREATE INDEX sms_raw_events_pending_idx ON sms_raw_events (received_at) WHERE normalized_at IS NULL;

-- The budget alerts sent this month (80% and 100%), so each goes out once.
CREATE TABLE sms_budget_alerts (
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    month        date        NOT NULL,
    level        integer     NOT NULL,
    at           timestamptz NOT NULL,
    PRIMARY KEY (workspace_id, month, level)
);

-- +goose Down
DROP TABLE sms_budget_alerts;
DROP TABLE sms_raw_events;
DROP TABLE sms_link_clicks;
DROP TABLE sms_links;
DROP TABLE sms_inbound;
DROP TABLE channel_message_events;
DROP TABLE channel_messages;
DROP TABLE sms_sends;
DROP TABLE sms_campaign_versions;
DROP TABLE sms_campaigns;
DROP TABLE sms_opt_out_sync;
DROP TABLE channel_suppressions;
DROP TABLE sms_terms_versions;
DROP TABLE sms_test_numbers;
DROP TABLE sms_settings;
DROP TABLE sms_prices;
DROP TABLE sms_routes;
DROP TABLE sms_registrations;
DROP TABLE sms_senders;
DROP TABLE sms_setups;
DROP TABLE sms_providers;
DROP TABLE channel_consent_events;
DROP TABLE channel_consents;
DROP INDEX campaigns_channel_idx;
ALTER TABLE campaigns DROP COLUMN channel;
DROP INDEX contacts_phone_idx;
ALTER TABLE contacts
    DROP COLUMN phone,
    DROP COLUMN phone_country,
    DROP COLUMN phone_line_type,
    DROP COLUMN phone_line_source,
    DROP COLUMN phone_set_at;
