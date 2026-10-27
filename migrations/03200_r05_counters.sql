-- +goose Up
-- Honest counters (review AN-3, AN-9, AN-10). A campaign's counters
-- (campaign_counters, campaign_hourly) are a function of its messages'
-- states, and they change only when a message's state changes: a soft
-- bounce followed by a delivery counts once, as delivered; a deferral
-- counts once per message however often the provider retries
-- (internal/counters says which state counts where). The new columns are
-- the states the status alone can't tell apart:
--   deferred_at      the first deferral (counted once per message)
--   soft_bounced_at  the first soft bounce; it stops counting as one once
--                    the message is delivered or bounces hard
--   unsubscribed_at  the reader unsubscribed from this message (page,
--                    one-click or a provider's unsubscribe event)
--   privacy_opened_at a privacy proxy (Apple Mail Privacy Protection)
--                    fetched the images: whether a person read it can't be
--                    measured, so it isn't a human or a machine open
ALTER TABLE messages
    ADD COLUMN deferred_at       timestamptz,
    ADD COLUMN soft_bounced_at   timestamptz,
    ADD COLUMN unsubscribed_at   timestamptz,
    ADD COLUMN privacy_opened_at timestamptz;

-- Opens in three buckets: people, privacy-protected (Apple Mail opened it
-- on delivery) and machines (security scanners, proxies prefetching on
-- arrival). opened_machine keeps the scanners only from now on.
ALTER TABLE campaign_counters ADD COLUMN opened_privacy bigint NOT NULL DEFAULT 0;
ALTER TABLE campaign_hourly ADD COLUMN opened_privacy bigint NOT NULL DEFAULT 0;

-- Clicks per link that the bot filter left out (where scanners clicked).
ALTER TABLE campaign_links ADD COLUMN bot_clicks bigint NOT NULL DEFAULT 0;

-- An open's or a click's class as tracking decided it (human, privacy,
-- machine for opens; human, bot for clicks) and the position of the link
-- clicked, so counters can be rebuilt from the events and the click map
-- can show where bot clicks landed.
ALTER TABLE delivery_events
    ADD COLUMN class text NOT NULL DEFAULT '',
    ADD COLUMN link  integer;

-- Opens and clicks the tracking writer couldn't record (the database was
-- unreachable, a batch kept failing): kept here and applied again by the
-- tracking.retry job, so a pixel or a redirect never loses its event.
CREATE TABLE tracking_retry (
    id          bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    message_id  uuid        NOT NULL,
    event       jsonb       NOT NULL,
    attempts    integer     NOT NULL DEFAULT 0,
    next_at     timestamptz NOT NULL DEFAULT now(),
    error       text        NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX tracking_retry_due_idx ON tracking_retry (next_at);

-- When a campaign's counters were last rebuilt from its messages and
-- events (the reports.rebuild_counters job), with the counting rules'
-- version; skipped says why a campaign can't be rebuilt (its per-message
-- history isn't stored: imported history).
CREATE TABLE campaign_counter_builds (
    campaign_id uuid        PRIMARY KEY REFERENCES campaigns(id) ON DELETE CASCADE,
    version     integer     NOT NULL,
    built_at    timestamptz NOT NULL,
    skipped     text        NOT NULL DEFAULT ''
);

-- +goose Down
DROP TABLE campaign_counter_builds;
DROP TABLE tracking_retry;
ALTER TABLE delivery_events DROP COLUMN link, DROP COLUMN class;
ALTER TABLE campaign_links DROP COLUMN bot_clicks;
ALTER TABLE campaign_hourly DROP COLUMN opened_privacy;
ALTER TABLE campaign_counters DROP COLUMN opened_privacy;
ALTER TABLE messages
    DROP COLUMN privacy_opened_at,
    DROP COLUMN unsubscribed_at,
    DROP COLUMN soft_bounced_at,
    DROP COLUMN deferred_at;
