-- +goose Up
-- M58 G-15: receiving mail. A workspace receives through one inbound
-- route: a connected provider's receiving (the provider posts what it
-- receives to /hooks/inbound/{token}) or a mailbox ReputeCast polls over
-- IMAP or POP3. Replies, mailto: unsubscribes, DMARC aggregate reports and
-- feedback-loop reports arrive at addresses on the route's domain.
CREATE TABLE inbound_routes (
    id                       uuid PRIMARY KEY,
    workspace_id             uuid        NOT NULL UNIQUE REFERENCES workspaces(id) ON DELETE CASCADE,
    kind                     text        NOT NULL CHECK (kind IN ('provider', 'mailbox')),
    provider_id              uuid        REFERENCES providers(id) ON DELETE SET NULL,
    domain                   text        NOT NULL,
    -- A mailbox route's address is <mailbox_local>@<domain>; tags go after
    -- a plus (replies+r-…@acme.io).
    mailbox_local            text        NOT NULL DEFAULT '',
    -- protocol (imap|pop3), host, port, security (tls|starttls), username.
    mailbox_settings         jsonb       NOT NULL DEFAULT '{}',
    mailbox_password_sealed  bytea,
    -- The adapter's settings for the route (an SNS topic ARN).
    settings                 jsonb       NOT NULL DEFAULT '{}',
    -- The hook's basic-auth password or signing key, sealed.
    secret_sealed            bytea       NOT NULL,
    hook_token_hash          bytea       NOT NULL UNIQUE,
    hook_token_sealed        bytea       NOT NULL,
    status                   text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'receiving', 'failing')),
    status_detail            text        NOT NULL DEFAULT '',
    registered_at            timestamptz,
    verified_at              timestamptz,
    last_received_at         timestamptz,
    last_polled_at           timestamptz,
    -- When the route's members were last told: replies received (at most
    -- every 15 minutes) or receiving failing (once until it recovers).
    notified_at              timestamptz,
    failing_notified_at      timestamptz,
    test_sent_at             timestamptz,
    test_received_at         timestamptz,
    created_by               uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at               timestamptz NOT NULL,
    updated_at               timestamptz NOT NULL
);
-- A domain receives for one workspace.
CREATE UNIQUE INDEX inbound_routes_domain_idx ON inbound_routes (lower(domain)) WHERE kind = 'provider';

-- What arrived, as it arrived: verified hook requests and mailbox messages,
-- kept 7 days, parsed by the inbound.process job.
CREATE TABLE inbound_raw (
    id             uuid PRIMARY KEY,
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    route_id       uuid        NOT NULL REFERENCES inbound_routes(id) ON DELETE CASCADE,
    source         text        NOT NULL CHECK (source IN ('hook', 'mailbox', 'test')),
    received_at    timestamptz NOT NULL,
    content_type   text        NOT NULL DEFAULT '',
    headers        jsonb       NOT NULL DEFAULT '{}',
    body           bytea       NOT NULL,
    signature_ok   boolean     NOT NULL,
    processed_at   timestamptz,
    process_error  text        NOT NULL DEFAULT ''
);
CREATE INDEX inbound_raw_unprocessed_idx ON inbound_raw (received_at) WHERE processed_at IS NULL;
CREATE INDEX inbound_raw_route_idx ON inbound_raw (route_id, received_at);

-- Each received message, classified. Bodies are kept as text (64 KiB at
-- most); attachments only by name, type and size.
CREATE TABLE inbound_messages (
    id                 uuid PRIMARY KEY,
    workspace_id       uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    route_id           uuid        REFERENCES inbound_routes(id) ON DELETE SET NULL,
    raw_id             uuid        REFERENCES inbound_raw(id) ON DELETE SET NULL,
    kind               text        NOT NULL CHECK (kind IN ('reply', 'unsubscribe', 'dmarc_report', 'feedback_report', 'test', 'other')),
    -- The message of ours it answers (messages.id), when the address says.
    message_id         uuid,
    contact_id         uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    campaign_id        uuid        REFERENCES campaigns(id) ON DELETE SET NULL,
    automation_id      uuid        REFERENCES automations(id) ON DELETE SET NULL,
    from_email         text        NOT NULL DEFAULT '',
    from_name          text        NOT NULL DEFAULT '',
    to_address         text        NOT NULL DEFAULT '',
    subject            text        NOT NULL DEFAULT '',
    text_body          text        NOT NULL DEFAULT '',
    has_html           boolean     NOT NULL DEFAULT false,
    attachments        jsonb       NOT NULL DEFAULT '[]',
    auto_reply         boolean     NOT NULL DEFAULT false,
    spam               boolean     NOT NULL DEFAULT false,
    asks_unsubscribe   boolean     NOT NULL DEFAULT false,
    -- The provider's verdicts: {"spf", "dkim", "dmarc"}.
    auth               jsonb       NOT NULL DEFAULT '{}',
    status             text        NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'done')),
    handled_by         uuid        REFERENCES users(id) ON DELETE SET NULL,
    handled_at         timestamptz,
    header_message_id  text        NOT NULL DEFAULT '',
    received_at        timestamptz NOT NULL
);
CREATE UNIQUE INDEX inbound_messages_header_idx ON inbound_messages (workspace_id, header_message_id) WHERE header_message_id <> '';
CREATE INDEX inbound_messages_list_idx ON inbound_messages (workspace_id, kind, status, received_at DESC, id DESC);
CREATE INDEX inbound_messages_message_idx ON inbound_messages (message_id) WHERE message_id IS NOT NULL;
CREATE INDEX inbound_messages_contact_idx ON inbound_messages (contact_id) WHERE contact_id IS NOT NULL;
CREATE INDEX inbound_messages_automation_idx ON inbound_messages (automation_id, contact_id) WHERE automation_id IS NOT NULL;
CREATE INDEX inbound_messages_search_idx ON inbound_messages
    USING gin ((subject || ' ' || from_email || ' ' || from_name) gin_trgm_ops);

-- DMARC aggregate reports can arrive by email at dmarc@<route domain>.
ALTER TABLE dmarc_reports DROP CONSTRAINT IF EXISTS dmarc_reports_via_check;
ALTER TABLE dmarc_reports ADD CONSTRAINT dmarc_reports_via_check CHECK (via IN ('hostrepute', 'upload', 'inbound'));

-- Feedback-loop (ARF) reports received at fbl@<route domain> are stored as
-- provider events, with their source.
ALTER TABLE provider_raw_events DROP CONSTRAINT IF EXISTS provider_raw_events_source_check;
ALTER TABLE provider_raw_events ADD CONSTRAINT provider_raw_events_source_check
    CHECK (source IN ('webhook', 'bounce_mailbox', 'backfill', 'test', 'queue', 'inbound'));

-- +goose Down
DELETE FROM provider_raw_events WHERE source = 'inbound';
ALTER TABLE provider_raw_events DROP CONSTRAINT IF EXISTS provider_raw_events_source_check;
ALTER TABLE provider_raw_events ADD CONSTRAINT provider_raw_events_source_check
    CHECK (source IN ('webhook', 'bounce_mailbox', 'backfill', 'test', 'queue'));
DELETE FROM dmarc_reports WHERE via = 'inbound';
ALTER TABLE dmarc_reports DROP CONSTRAINT IF EXISTS dmarc_reports_via_check;
ALTER TABLE dmarc_reports ADD CONSTRAINT dmarc_reports_via_check CHECK (via IN ('hostrepute', 'upload'));
DROP TABLE inbound_messages;
DROP TABLE inbound_raw;
DROP TABLE inbound_routes;
