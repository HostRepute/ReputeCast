-- +goose Up
-- Audience (spec 5.6, 5.8, 5.10, 7.9, 7.11): what the contacts table, the
-- suppression list, segments and imports need beyond the base schema, plus
-- daily audience rollups for list health and the import staging area.

-- Contacts ---------------------------------------------------------------
ALTER TABLE contacts
    ADD COLUMN company               text        NOT NULL DEFAULT '',
    ADD COLUMN language              text        NOT NULL DEFAULT '',
    -- The hosting behind an "Other" mailbox ("Google Workspace", "Fastmail"),
    -- found from MX records at import or signup.
    ADD COLUMN mailbox_detail        text        NOT NULL DEFAULT '',
    -- How consent was given: double opt-in, single opt-in, or pending
    -- (imported without proof, waiting for re-permission).
    ADD COLUMN consent               text        NOT NULL DEFAULT 'single' CHECK (consent IN ('double','single','pending')),
    -- Evidence carried over from imports: {"at":"…","ip":"…","source":"Mailchimp OPTIN_TIME"}.
    ADD COLUMN consent_evidence      jsonb       NOT NULL DEFAULT '{}',
    ADD COLUMN import_id             uuid        REFERENCES imports(id) ON DELETE SET NULL,
    -- Role addresses (info@, sales@…) are kept but never get Broadcasts.
    ADD COLUMN role_address          boolean     NOT NULL DEFAULT false,
    -- last_open_at includes machine opens (Apple Mail Privacy Protection);
    -- tiers use last_human_open_at unless the sunset policy counts machine opens.
    ADD COLUMN last_human_open_at    timestamptz,
    ADD COLUMN last_open_machine     boolean     NOT NULL DEFAULT false,
    -- Latest delivery, engagement or consent event; the default sort.
    ADD COLUMN last_activity_at      timestamptz,
    ADD COLUMN tier_changed_at       timestamptz,
    -- Sunset policy bookkeeping: when the contact entered Sunset and when
    -- Broadcasts stopped because there was still no open after that.
    ADD COLUMN sunset_at             timestamptz,
    ADD COLUMN broadcasts_stopped_at timestamptz;
CREATE INDEX contacts_activity_idx ON contacts (workspace_id, last_activity_at DESC NULLS LAST, id);
CREATE INDEX contacts_mailbox_idx ON contacts (workspace_id, mailbox_provider);
CREATE INDEX contacts_import_idx ON contacts (import_id) WHERE import_id IS NOT NULL;
CREATE INDEX list_memberships_subscribed_idx ON list_memberships (list_id, subscribed_at);

ALTER TABLE consent_events
    ADD COLUMN note            text NOT NULL DEFAULT '',
    ADD COLUMN consent_version text NOT NULL DEFAULT '',
    ADD COLUMN actor_id        uuid REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN import_id       uuid REFERENCES imports(id) ON DELETE SET NULL;
ALTER TABLE consent_events DROP CONSTRAINT consent_events_action_check;
ALTER TABLE consent_events ADD CONSTRAINT consent_events_action_check
    CHECK (action IN ('subscribed','confirmed','unsubscribed','resubscribed','imported','paused','frequency_changed','erased','optin_sent','suppressed'));

-- Number fields can be shown as money ("$29").
ALTER TABLE custom_fields
    ADD COLUMN display  text    NOT NULL DEFAULT '' CHECK (display IN ('','currency')),
    ADD COLUMN position integer NOT NULL DEFAULT 0;

-- Which provider delivered a message and why: the activity timeline shows
-- "via Postmark · failover" and the route note ("Postal share of the 90/10
-- split, warm-up day 6"). Filled by the sending pipeline.
ALTER TABLE messages
    ADD COLUMN failover   boolean NOT NULL DEFAULT false,
    ADD COLUMN route_note text    NOT NULL DEFAULT '';

-- Suppressions -----------------------------------------------------------
-- scope: 'all' blocks every stream (bounces, complaints, manual);
-- 'marketing' blocks Broadcasts and Automations but allows System mail
-- (unsubscribes). The campaign or flow that caused it, or a free-text
-- context ("Migration", "Legacy SMTP relay"); imports group their rows.
ALTER TABLE suppressions
    ADD COLUMN scope           text NOT NULL DEFAULT 'all' CHECK (scope IN ('all','marketing')),
    ADD COLUMN campaign_id     uuid REFERENCES campaigns(id) ON DELETE SET NULL,
    ADD COLUMN automation_id   uuid REFERENCES automations(id) ON DELETE SET NULL,
    ADD COLUMN automation_step text NOT NULL DEFAULT '',
    ADD COLUMN context         text NOT NULL DEFAULT '',
    ADD COLUMN import_id       uuid REFERENCES imports(id) ON DELETE SET NULL;
CREATE INDEX suppressions_import_idx ON suppressions (workspace_id, import_id);
CREATE INDEX suppressions_reason_idx ON suppressions (workspace_id, reason);

-- Segments ---------------------------------------------------------------
ALTER TABLE segments
    ADD COLUMN refresh     text    NOT NULL DEFAULT 'dynamic' CHECK (refresh IN ('dynamic','static')),
    ADD COLUMN count_ms    integer,
    -- Tier mix of the last count: {"active":1821,"cooling":297,"at_risk":0,"sunset":0}
    ADD COLUMN tier_counts jsonb   NOT NULL DEFAULT '{}',
    ADD COLUMN updated_by  uuid    REFERENCES users(id) ON DELETE SET NULL;

-- Static segments keep the contacts that matched when they were saved.
CREATE TABLE segment_members (
    segment_id uuid NOT NULL REFERENCES segments(id) ON DELETE CASCADE,
    contact_id uuid NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    PRIMARY KEY (segment_id, contact_id)
);

-- Saved views can be followed in List health (a cohort such as a
-- consent-pending import that needs watching).
ALTER TABLE saved_views ADD COLUMN show_in_health boolean NOT NULL DEFAULT false;

-- Daily audience rollups --------------------------------------------------
-- Written by the nightly tier job: the tier mix per scope (the workspace,
-- each list, each followed saved view) and the day's joins, departures and
-- Broadcast engagement. List health reads 30-day sums; the tier trend reads
-- weekly samples.
CREATE TABLE audience_daily (
    workspace_id uuid   NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    scope        text   NOT NULL CHECK (scope IN ('workspace','list','view')),
    scope_id     uuid   NOT NULL,
    day          date   NOT NULL,
    contacts     bigint NOT NULL DEFAULT 0,
    active       bigint NOT NULL DEFAULT 0,
    cooling      bigint NOT NULL DEFAULT 0,
    at_risk      bigint NOT NULL DEFAULT 0,
    sunset       bigint NOT NULL DEFAULT 0,
    joined       bigint NOT NULL DEFAULT 0,
    departed     bigint NOT NULL DEFAULT 0,
    sent         bigint NOT NULL DEFAULT 0,
    delivered    bigint NOT NULL DEFAULT 0,
    opened_human bigint NOT NULL DEFAULT 0,
    unsubscribed bigint NOT NULL DEFAULT 0,
    complained   bigint NOT NULL DEFAULT 0,
    bounced      bigint NOT NULL DEFAULT 0,
    PRIMARY KEY (workspace_id, scope, scope_id, day)
);

-- Per-workspace state of the tier job and the sunset policy's counters.
CREATE TABLE audience_state (
    workspace_id      uuid        PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    tiers_computed_at timestamptz,
    -- Contacts sent the re-engagement email and how many came back.
    reengaged         bigint      NOT NULL DEFAULT 0,
    won_back          bigint      NOT NULL DEFAULT 0,
    updated_at        timestamptz NOT NULL DEFAULT now()
);

-- Imports ----------------------------------------------------------------
ALTER TABLE imports DROP CONSTRAINT imports_status_check;
ALTER TABLE imports ADD CONSTRAINT imports_status_check
    CHECK (status IN ('draft','staging','mapping','previewing','running','paused','done','failed','cancelled'));
ALTER TABLE imports
    -- The wizard step the person last reached.
    ADD COLUMN step        text        NOT NULL DEFAULT 'source' CHECK (step IN ('source','fields','consent','preview','import','done')),
    ADD COLUMN staged_at   timestamptz,
    -- {"total":13739,"processed":9312,"batch":19,"batches":28}
    ADD COLUMN progress    jsonb       NOT NULL DEFAULT '{}',
    ADD COLUMN skipped_csv bytea,
    ADD COLUMN resume_at   timestamptz,
    ADD COLUMN updated_at  timestamptz NOT NULL DEFAULT now();

-- Rows read from the source, waiting for mapping, preview and import.
CREATE TABLE import_rows (
    import_id  uuid    NOT NULL REFERENCES imports(id) ON DELETE CASCADE,
    row_no     integer NOT NULL,
    email      text    NOT NULL DEFAULT '',
    -- Source consent status: subscribed, unsubscribed, cleaned, pending,
    -- transactional, complained, blocklisted…
    status     text    NOT NULL DEFAULT '',
    list_ref   text    NOT NULL DEFAULT '',
    data       jsonb   NOT NULL DEFAULT '{}',
    tags       text[]  NOT NULL DEFAULT '{}',
    -- {"last_open":"…","last_click":"…","rating":1,"changed":"…","optin_at":"…","optin_ip":"…","signup_at":"…"}
    engagement jsonb   NOT NULL DEFAULT '{}',
    -- Filled by the import: imported, merged, suppressed, pending, skipped.
    outcome    text    NOT NULL DEFAULT '',
    reason     text    NOT NULL DEFAULT '',
    PRIMARY KEY (import_id, row_no)
);

-- The live log of an import (batches, retries, provider pushes).
CREATE TABLE import_log (
    id          uuid        PRIMARY KEY,
    import_id   uuid        NOT NULL REFERENCES imports(id) ON DELETE CASCADE,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    kind        text        NOT NULL CHECK (kind IN ('start','running','done','sync','retry','error','info')),
    message     text        NOT NULL
);
CREATE INDEX import_log_import_idx ON import_log (import_id, occurred_at DESC);

-- MX lookups made while checking imported addresses, reused for a week.
CREATE TABLE mx_cache (
    domain     text        PRIMARY KEY,
    has_mx     boolean     NOT NULL,
    provider   text        NOT NULL DEFAULT '',
    checked_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE mx_cache;
DROP TABLE import_log;
DROP TABLE import_rows;
ALTER TABLE imports DROP COLUMN updated_at, DROP COLUMN resume_at, DROP COLUMN skipped_csv,
    DROP COLUMN progress, DROP COLUMN staged_at, DROP COLUMN step;
ALTER TABLE imports DROP CONSTRAINT imports_status_check;
ALTER TABLE imports ADD CONSTRAINT imports_status_check
    CHECK (status IN ('draft','mapping','previewing','running','done','failed','cancelled'));
DROP TABLE audience_state;
DROP TABLE audience_daily;
ALTER TABLE saved_views DROP COLUMN show_in_health;
DROP TABLE segment_members;
ALTER TABLE segments DROP COLUMN updated_by, DROP COLUMN tier_counts, DROP COLUMN count_ms, DROP COLUMN refresh;
DROP INDEX suppressions_reason_idx;
DROP INDEX suppressions_import_idx;
ALTER TABLE suppressions DROP COLUMN import_id, DROP COLUMN context, DROP COLUMN automation_step,
    DROP COLUMN automation_id, DROP COLUMN campaign_id, DROP COLUMN scope;
ALTER TABLE messages DROP COLUMN route_note, DROP COLUMN failover;
ALTER TABLE custom_fields DROP COLUMN position, DROP COLUMN display;
ALTER TABLE consent_events DROP CONSTRAINT consent_events_action_check;
ALTER TABLE consent_events ADD CONSTRAINT consent_events_action_check
    CHECK (action IN ('subscribed','confirmed','unsubscribed','resubscribed','imported','paused','frequency_changed','erased'));
ALTER TABLE consent_events DROP COLUMN import_id, DROP COLUMN actor_id, DROP COLUMN consent_version, DROP COLUMN note;
DROP INDEX list_memberships_subscribed_idx;
DROP INDEX contacts_import_idx;
DROP INDEX contacts_mailbox_idx;
DROP INDEX contacts_activity_idx;
ALTER TABLE contacts DROP COLUMN broadcasts_stopped_at, DROP COLUMN sunset_at, DROP COLUMN tier_changed_at,
    DROP COLUMN last_activity_at, DROP COLUMN last_open_machine, DROP COLUMN last_human_open_at,
    DROP COLUMN role_address, DROP COLUMN import_id, DROP COLUMN consent_evidence, DROP COLUMN consent,
    DROP COLUMN mailbox_detail, DROP COLUMN language, DROP COLUMN company;
