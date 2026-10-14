-- +goose Up
-- W12 · Sending engine, live monitor and tracking (spec 5.3, 7.6, add-on §6).

-- Messages: provider calls made for a message, the retry back-off and the
-- in-flight claim. A send worker claims a batch (claimed_at) while it hands
-- it to the provider; a claim older than five minutes belonged to a worker
-- that died and is taken over (the message id is the idempotency key the
-- adapters pass on, so a provider that dedupes never sends it twice).
-- content_ref says what an automation email renders ({"email":"…","version":3,
-- "subject":"…"}); human_opened_at separates human opens from machine opens
-- (Apple Mail Privacy Protection, image proxies, scanners).
ALTER TABLE messages
    ADD COLUMN attempts        smallint    NOT NULL DEFAULT 0,
    ADD COLUMN next_attempt_at timestamptz,
    ADD COLUMN claimed_at      timestamptz,
    ADD COLUMN content_ref     jsonb,
    ADD COLUMN human_opened_at timestamptz;
-- The send loop walks a campaign's queue per sending provider and mailbox
-- provider in id order; the automation outbox walks queued automation mail
-- per provider.
CREATE INDEX messages_send_queue_idx ON messages (campaign_id, provider_id, mailbox_provider, id) WHERE status = 'queued';
CREATE INDEX messages_outbox_idx ON messages (provider_id, id) WHERE status = 'queued' AND campaign_id IS NULL AND stream = 'automation';
CREATE INDEX messages_campaign_sent_idx ON messages (campaign_id, sent_at) WHERE sent_at IS NOT NULL;

-- The live event feed reads a campaign's newest events.
CREATE INDEX delivery_events_campaign_feed_idx ON delivery_events (campaign_id, received_at DESC);

-- One row per campaign send: how far the audience build got (it pages
-- through contacts with a cursor so a 50,000-recipient send never holds a
-- long transaction), what it left out, who started and approved it, the
-- per-mailbox throttles and auto-pause thresholds in force, mailbox
-- providers on hold, the A/B test's phase and, in the demo only, the
-- simulator scenario.
CREATE TABLE campaign_sends (
    campaign_id    uuid        PRIMARY KEY REFERENCES campaigns(id) ON DELETE CASCADE,
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    phase          text        NOT NULL DEFAULT 'preparing' CHECK (phase IN ('preparing','sending','finished','cancelled','failed')),
    prepare_cursor uuid,
    prepared_at    timestamptz,
    recipients     bigint      NOT NULL DEFAULT 0,
    -- {"suppressed":412,"unsubscribed":38,"bounced":12,"paused":6,"tiers":0,"segments":0}
    excluded       jsonb       NOT NULL DEFAULT '{}',
    started_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    started_by_name text       NOT NULL DEFAULT '',
    approved_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    approved_at    timestamptz,
    scheduled_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    -- Messages per second per mailbox provider ({"outlook":5,"yahoo":4}); absent = no limit.
    throttles      jsonb       NOT NULL DEFAULT '{}',
    -- Auto-pause thresholds in force, resolved from campaigns.thresholds.
    thresholds     jsonb       NOT NULL DEFAULT '{}',
    -- Mailbox providers on hold: {"outlook":{"incident":"…","since":"…"}}.
    holds          jsonb       NOT NULL DEFAULT '{}',
    -- Contacts taken out of the rest of the send ("Resume with changes").
    -- {"groups":[{"key":"never_engaged_imports","mailbox":"outlook","count":1420}]}
    changes        jsonb       NOT NULL DEFAULT '{}',
    -- A/B test: {"phase":"testing","decide_at":"…","winner":"B","decided_at":"…"}
    ab             jsonb       NOT NULL DEFAULT '{}',
    -- What the send renders, captured when it starts so edits during the
    -- send never change it: subject, preheader, HTML, text, A/B variants,
    -- sender and tracking ({"subject":"…","html":"…","variants":[…],"from":{…}}).
    content        jsonb       NOT NULL DEFAULT '{}',
    scenario       text        NOT NULL DEFAULT '',
    -- Who paused the whole send by hand, when and why.
    paused_at      timestamptz,
    paused_by_name text        NOT NULL DEFAULT '',
    pause_reason   text        NOT NULL DEFAULT '',
    error          text        NOT NULL DEFAULT '',
    finished_at    timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX campaign_sends_active_idx ON campaign_sends (phase) WHERE phase IN ('preparing','sending');

-- Segment members staged while the audience build pages through contacts
-- (included and excluded segments), deleted once the build is done.
CREATE TABLE campaign_audience_stage (
    campaign_id uuid    NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    contact_id  uuid    NOT NULL,
    exclude     boolean NOT NULL DEFAULT false,
    PRIMARY KEY (campaign_id, exclude, contact_id)
);

-- Per-second send windows per sending provider, shared by every worker so
-- concurrent campaigns and automations never exceed a provider's rate.
-- Rows older than a minute are purged; losing them on a crash is harmless.
CREATE UNLOGGED TABLE sending_rate_windows (
    provider_id uuid    NOT NULL,
    sec         bigint  NOT NULL,
    used        integer NOT NULL DEFAULT 0,
    last_grant  integer NOT NULL DEFAULT 0,
    PRIMARY KEY (provider_id, sec)
);

-- Events the demo transport will report for messages it accepted
-- (delivered, deferred, bounces, complaints, opens, clicks), applied when
-- due. Only providers marked "demo": true ever produce them.
CREATE TABLE sending_sim_events (
    id          bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    workspace_id uuid       NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    campaign_id uuid        REFERENCES campaigns(id) ON DELETE CASCADE,
    message_id  uuid        NOT NULL,
    provider_id uuid        NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    due_at      timestamptz NOT NULL,
    kind        text        NOT NULL CHECK (kind IN ('delivered','deferred','hard_bounce','soft_bounce','complaint','open','click')),
    data        jsonb       NOT NULL DEFAULT '{}'
);
CREATE INDEX sending_sim_events_due_idx ON sending_sim_events (due_at);

-- First click of a message on each link (unique clicks of the click map).
CREATE TABLE tracking_link_clicks (
    message_id uuid        NOT NULL,
    link       integer     NOT NULL,
    clicked_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (message_id, link)
);

-- Links of automation emails, per email version, for click tracking (the
-- campaign equivalent is campaign_links).
CREATE TABLE automation_email_links (
    email_id uuid    NOT NULL REFERENCES automation_emails(id) ON DELETE CASCADE,
    version  integer NOT NULL,
    position integer NOT NULL,
    url      text    NOT NULL,
    label    text    NOT NULL DEFAULT '',
    clicks   bigint  NOT NULL DEFAULT 0,
    PRIMARY KEY (email_id, version, position)
);

-- +goose Down
DROP TABLE automation_email_links;
DROP TABLE tracking_link_clicks;
DROP TABLE sending_sim_events;
DROP TABLE sending_rate_windows;
DROP TABLE campaign_audience_stage;
DROP TABLE campaign_sends;
DROP INDEX delivery_events_campaign_feed_idx;
DROP INDEX messages_campaign_sent_idx;
DROP INDEX messages_outbox_idx;
DROP INDEX messages_send_queue_idx;
ALTER TABLE messages
    DROP COLUMN human_opened_at,
    DROP COLUMN content_ref,
    DROP COLUMN claimed_at,
    DROP COLUMN next_attempt_at,
    DROP COLUMN attempts;
