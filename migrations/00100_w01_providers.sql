-- +goose Up
-- Sending providers & routing: raw and normalized event columns, per-minute
-- API call windows, failover state per stream, warm-up settings and log.

-- Raw provider events. headers keeps the request headers verification and
-- parsing need (signature, authorization, content type) so the job can rebuild
-- the request; source says how the payload arrived; events counts what the
-- job normalized from it.
ALTER TABLE provider_raw_events ADD COLUMN headers jsonb   NOT NULL DEFAULT '{}';
ALTER TABLE provider_raw_events ADD COLUMN source  text    NOT NULL DEFAULT 'webhook' CHECK (source IN ('webhook','bounce_mailbox','backfill','test'));
ALTER TABLE provider_raw_events ADD COLUMN events  integer NOT NULL DEFAULT 0;

-- Normalized events also keep what the provider told us when no message
-- matches (recipient, provider message id), when ReputeCast learned about
-- the event, and a label of what sent the message, captured at normalization
-- ("Weekly Digest #142", "Welcome series · step 1", "Double opt-in
-- confirmation"). dedupe_key drops provider redeliveries of the same event.
ALTER TABLE delivery_events ADD COLUMN recipient           text        NOT NULL DEFAULT '';
ALTER TABLE delivery_events ADD COLUMN provider_message_id text        NOT NULL DEFAULT '';
ALTER TABLE delivery_events ADD COLUMN source              text        NOT NULL DEFAULT '';
ALTER TABLE delivery_events ADD COLUMN received_at         timestamptz NOT NULL DEFAULT now();
ALTER TABLE delivery_events ADD COLUMN dedupe_key          text        NOT NULL DEFAULT '';
CREATE UNIQUE INDEX delivery_events_dedupe_key ON delivery_events (provider_id, dedupe_key) WHERE dedupe_key <> '';
CREATE INDEX delivery_events_log_idx ON delivery_events (workspace_id, received_at DESC, id DESC);
CREATE INDEX delivery_events_provider_msg_idx ON delivery_events (provider_id, provider_message_id) WHERE provider_message_id <> '';

-- What produced a message that has no campaign or automation: system mail
-- ('double_opt_in', 'preference_link', 'invite', 'password_reset') or the
-- API ('api'). Used for failover summaries and the event log.
ALTER TABLE messages ADD COLUMN kind text NOT NULL DEFAULT '';
CREATE INDEX messages_provider_sent_idx ON messages (provider_id, sent_at DESC) WHERE sent_at IS NOT NULL;

-- Per-minute API call counters per provider, written by every adapter call
-- (sends, health checks): calls and failed calls ("API error rate · 3 min",
-- outage detection) and messages accepted ("0 /s now · peak today 14 /s").
-- Rows older than two days are purged.
CREATE TABLE provider_call_windows (
    provider_id uuid        NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    minute      timestamptz NOT NULL,
    calls       integer     NOT NULL DEFAULT 0,
    errors      integer     NOT NULL DEFAULT 0,
    messages    integer     NOT NULL DEFAULT 0,
    PRIMARY KEY (provider_id, minute)
);

-- Failover state per stream: while a provider is down, mail of a stream it
-- serves either goes to the stream's fallback (mode automatic) or is held
-- for a decision (mode held, Broadcasts by default).
CREATE TABLE stream_failovers (
    id                   uuid PRIMARY KEY,
    workspace_id         uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    stream               text        NOT NULL CHECK (stream IN ('broadcast','automation','system')),
    provider_id          uuid        NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    fallback_provider_id uuid        REFERENCES providers(id) ON DELETE SET NULL,
    mode                 text        NOT NULL CHECK (mode IN ('automatic','held')),
    started_at           timestamptz NOT NULL DEFAULT now(),
    ended_at             timestamptz,
    end_reason           text        NOT NULL DEFAULT '',
    detail               jsonb       NOT NULL DEFAULT '{}'
);
CREATE UNIQUE INDEX stream_failovers_active_key ON stream_failovers (workspace_id, stream, provider_id) WHERE ended_at IS NULL;

-- Keeping a Broadcast route that the provider's terms don't allow is a
-- recorded decision; Send Readiness blocks broadcasts while it stands.
ALTER TABLE stream_policies ADD COLUMN bulk_terms_ack_at timestamptz;
ALTER TABLE stream_policies ADD COLUMN bulk_terms_ack_by uuid REFERENCES users(id) ON DELETE SET NULL;

-- Warm-up settings: what to do when a guard trips, spreading the daily cap
-- across mailbox providers, who started the plan and when it was paused.
ALTER TABLE warmup_plans ADD COLUMN guard_action   text    NOT NULL DEFAULT 'hold' CHECK (guard_action IN ('hold','step_back','pause'));
ALTER TABLE warmup_plans ADD COLUMN spread_mailbox boolean NOT NULL DEFAULT true;
ALTER TABLE warmup_plans ADD COLUMN created_by     uuid    REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE warmup_plans ADD COLUMN paused_at      timestamptz;
ALTER TABLE warmup_plans ADD COLUMN updated_at     timestamptz NOT NULL DEFAULT now();

-- The warm-up log: day results, automatic slowdowns, share increases.
CREATE TABLE warmup_events (
    id          uuid PRIMARY KEY,
    plan_id     uuid        NOT NULL REFERENCES warmup_plans(id) ON DELETE CASCADE,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    kind        text        NOT NULL CHECK (kind IN ('started','day_complete','slowdown','resumed','paused','share_raised','settings','done')),
    -- Bold lead of the entry ("Automatic slowdown."), may be empty.
    title       text        NOT NULL DEFAULT '',
    body        text        NOT NULL
);
CREATE INDEX warmup_events_plan_idx ON warmup_events (plan_id, occurred_at DESC);

-- +goose Down
DROP TABLE warmup_events;
ALTER TABLE warmup_plans DROP COLUMN updated_at;
ALTER TABLE warmup_plans DROP COLUMN paused_at;
ALTER TABLE warmup_plans DROP COLUMN created_by;
ALTER TABLE warmup_plans DROP COLUMN spread_mailbox;
ALTER TABLE warmup_plans DROP COLUMN guard_action;
ALTER TABLE stream_policies DROP COLUMN bulk_terms_ack_by;
ALTER TABLE stream_policies DROP COLUMN bulk_terms_ack_at;
DROP TABLE stream_failovers;
DROP TABLE provider_call_windows;
DROP INDEX messages_provider_sent_idx;
ALTER TABLE messages DROP COLUMN kind;
DROP INDEX delivery_events_provider_msg_idx;
DROP INDEX delivery_events_log_idx;
DROP INDEX delivery_events_dedupe_key;
ALTER TABLE delivery_events DROP COLUMN dedupe_key;
ALTER TABLE delivery_events DROP COLUMN received_at;
ALTER TABLE delivery_events DROP COLUMN source;
ALTER TABLE delivery_events DROP COLUMN provider_message_id;
ALTER TABLE delivery_events DROP COLUMN recipient;
ALTER TABLE provider_raw_events DROP COLUMN events;
ALTER TABLE provider_raw_events DROP COLUMN source;
ALTER TABLE provider_raw_events DROP COLUMN headers;
