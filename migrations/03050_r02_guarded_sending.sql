-- +goose Up
-- Guarded sending and the live monitor (review 2026-10, M34).

-- GS-1. A message whose provider call ended without a recorded answer (the
-- worker stopped in the middle of the call, the result couldn't be written)
-- is "in doubt": nobody knows whether the provider took it, so it is never
-- sent again on a guess. A provider that dedupes by idempotency key gets it
-- again with the same key; otherwise a person decides (send it again or
-- leave it out), and an event the provider reports for it settles it.
ALTER TABLE messages DROP CONSTRAINT messages_status_check;
ALTER TABLE messages ADD CONSTRAINT messages_status_check CHECK (status IN
    ('queued','held','sent','delivered','deferred','bounced','complained','rejected','failed','cancelled','in_doubt'));
CREATE INDEX messages_in_doubt_idx ON messages (campaign_id, provider_id) WHERE status = 'in_doubt';
-- Claims in flight, so a stale one is found without reading a whole queue.
CREATE INDEX messages_claimed_idx ON messages (claimed_at) WHERE status = 'queued' AND claimed_at IS NOT NULL;

-- GS-10. The canary: the messages that go first, and why the rest is held
-- ('canary': until the canary passes; '' for the A/B test's hold).
ALTER TABLE messages
    ADD COLUMN canary      boolean NOT NULL DEFAULT false,
    ADD COLUMN hold_reason text    NOT NULL DEFAULT '';

-- The guard's state of a send (campaign_sends.guard): where each scope's
-- thresholds re-arm after a resume (GS-2), slowdowns with their end (GS-11,
-- GS-12), the rates planned when the send started and the ones a person set
-- since (GS-13). canary is the canary's plan, phase and gates (GS-10);
-- watch_until ends the complaint watch after the send (GS-6).
ALTER TABLE campaign_sends
    ADD COLUMN guard       jsonb NOT NULL DEFAULT '{}',
    ADD COLUMN canary      jsonb NOT NULL DEFAULT '{}',
    ADD COLUMN watch_until timestamptz;
CREATE INDEX campaign_sends_watch_idx ON campaign_sends (watch_until) WHERE watch_until IS NOT NULL;

-- Incidents the guard raises now: a mailbox provider's reply rule (GS-12),
-- the canary's decision (GS-10), recipients the template failed for
-- (CO-4), the complaint watch after a send (GS-6), messages in doubt
-- waiting for a person's decision (GS-1). Timed holds end by
-- themselves ('expired'); a canary is continued or stopped; warnings and
-- alerts are acknowledged when raised; in-doubt messages sent again.
ALTER TABLE campaign_incidents DROP CONSTRAINT campaign_incidents_kind_check;
ALTER TABLE campaign_incidents ADD CONSTRAINT campaign_incidents_kind_check CHECK (kind IN
    ('threshold','provider_outage','throttled','recheck_blocked','reply_rule','canary','render_error','complaint_watch','in_doubt'));
ALTER TABLE campaign_incidents DROP CONSTRAINT campaign_incidents_resolution_check;
ALTER TABLE campaign_incidents ADD CONSTRAINT campaign_incidents_resolution_check CHECK (resolution IN
    ('','resumed','resumed_with_changes','cancelled','moved','retried','auto_recovered','expired','continued','stopped','acknowledged'));
-- Traffic rules' "Recent actions" read a workspace's incidents by time.
CREATE INDEX campaign_incidents_tripped_idx ON campaign_incidents (tripped_at DESC);

-- Traffic rules (GS-10, GS-6): the canary defaults and the complaint watch
-- after a send, per workspace. The thresholds by bounce type live with the
-- other auto-pause defaults in workspaces.send_thresholds.
CREATE TABLE traffic_rules (
    workspace_id uuid        PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    -- {"enabled":true,"newDomain":true,"neverMailed":true,"overUsual":true,"percent":5,"seeds":true,
    --  "watchMinutes":20,"gates":{"bounce":1.0,"blockBounce":0.2,"deferral":15,"complaint":0.10,"seedSpam":2}}
    canary       jsonb       NOT NULL DEFAULT '{}',
    -- {"hours":72,"alert":true,"postmaster":true}
    watch        jsonb       NOT NULL DEFAULT '{}',
    updated_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- Mailbox provider response rules (GS-12): when a mailbox provider's SMTP
-- replies contain a text (Yahoo's TSS04, Outlook's S3150) count times within
-- a window, the send pauses or slows that mailbox provider for a while.
CREATE TABLE traffic_reply_rules (
    id               uuid        PRIMARY KEY,
    workspace_id     uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    mailbox_provider text        NOT NULL CHECK (mailbox_provider IN ('gmail','outlook','yahoo','icloud','other')),
    contains         text        NOT NULL CHECK (length(contains) BETWEEN 2 AND 120),
    -- min_count replies within window_seconds (1 = the first reply).
    min_count        integer     NOT NULL DEFAULT 1 CHECK (min_count BETWEEN 1 AND 10000),
    window_seconds   integer     NOT NULL DEFAULT 600 CHECK (window_seconds BETWEEN 60 AND 86400),
    action           text        NOT NULL CHECK (action IN ('slow','pause','pause_route','alert')),
    -- How long the action lasts; NULL until someone resumes.
    duration_seconds integer     CHECK (duration_seconds IS NULL OR duration_seconds BETWEEN 60 AND 604800),
    enabled          boolean     NOT NULL DEFAULT true,
    position         integer     NOT NULL DEFAULT 0,
    note             text        NOT NULL DEFAULT '',
    last_matched_at  timestamptz,
    last_campaign_id uuid        REFERENCES campaigns(id) ON DELETE SET NULL,
    created_by       uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX traffic_reply_rules_workspace_idx ON traffic_reply_rules (workspace_id, position);

-- GS-8. Per-mailbox-provider send windows shared by a workspace's sends, so
-- two campaigns going out at once never pass a mailbox provider's rate
-- together. A slot is a second (or several, for rates under one a second);
-- rows older than a minute are purged, and losing them on a crash is
-- harmless.
CREATE UNLOGGED TABLE sending_mailbox_windows (
    workspace_id     uuid    NOT NULL,
    mailbox_provider text    NOT NULL,
    slot             bigint  NOT NULL,
    used             integer NOT NULL DEFAULT 0,
    last_grant       integer NOT NULL DEFAULT 0,
    PRIMARY KEY (workspace_id, mailbox_provider, slot)
);

-- +goose Down
DROP TABLE sending_mailbox_windows;
DROP TABLE traffic_reply_rules;
DROP TABLE traffic_rules;
DROP INDEX campaign_incidents_tripped_idx;
UPDATE campaign_incidents SET resolution = 'resumed' WHERE resolution IN ('expired','continued','stopped','acknowledged');
DELETE FROM campaign_incidents WHERE kind IN ('reply_rule','canary','render_error','complaint_watch','in_doubt');
ALTER TABLE campaign_incidents DROP CONSTRAINT campaign_incidents_resolution_check;
ALTER TABLE campaign_incidents ADD CONSTRAINT campaign_incidents_resolution_check CHECK (resolution IN
    ('','resumed','resumed_with_changes','cancelled','moved','retried','auto_recovered'));
ALTER TABLE campaign_incidents DROP CONSTRAINT campaign_incidents_kind_check;
ALTER TABLE campaign_incidents ADD CONSTRAINT campaign_incidents_kind_check CHECK (kind IN
    ('threshold','provider_outage','throttled','recheck_blocked'));
DROP INDEX campaign_sends_watch_idx;
ALTER TABLE campaign_sends DROP COLUMN watch_until, DROP COLUMN canary, DROP COLUMN guard;
ALTER TABLE messages DROP COLUMN hold_reason, DROP COLUMN canary;
DROP INDEX messages_claimed_idx;
DROP INDEX messages_in_doubt_idx;
UPDATE messages SET status = 'failed', error = 'In doubt when the migration was rolled back' WHERE status = 'in_doubt';
ALTER TABLE messages DROP CONSTRAINT messages_status_check;
ALTER TABLE messages ADD CONSTRAINT messages_status_check CHECK (status IN
    ('queued','held','sent','delivered','deferred','bounced','complained','rejected','failed','cancelled'));
