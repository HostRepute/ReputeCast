-- +goose Up
-- Contact scoring (audience-insights brief §3.1, M23): rules, decay and
-- bands per workspace, every contact's score with what it is made of, the
-- compact event log the scorer replays, and background recalculations.

-- One row per workspace that set up scoring. `version` is the saved
-- configuration; `applied_version` is the one live scores use. They differ
-- while a recalculation runs: until it finishes, contacts keep their
-- previous score and segments and triggers use it.
CREATE TABLE scoring_configs (
    workspace_id     uuid        PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    version          integer     NOT NULL DEFAULT 0,
    applied_version  integer     NOT NULL DEFAULT 0,
    -- Source events (opens, clicks, forms, API events, soft bounces,
    -- unsubscribes) that arrived up to here are in score_events.
    ingested_through timestamptz,
    -- The last nightly run (decay, field conditions, events leaving the
    -- 180-day window), at 03:00 workspace time like the tier job.
    nightly_at       timestamptz,
    set_up_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    set_up_at        timestamptz,
    updated_at       timestamptz NOT NULL DEFAULT now()
);

-- Every saved configuration: {"rules":[…],"decay":{…},"bands":{…}}.
-- Rules keep their id across versions, so a breakdown can name them.
CREATE TABLE scoring_versions (
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    version      integer     NOT NULL,
    config       jsonb       NOT NULL,
    created_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, version)
);

-- A contact's score (0–100) under the applied configuration: the events
-- part (opens, clicks, forms, API events, bounces and unsubscribes after
-- caps and decay) plus the fields part (field conditions that are true).
-- The band follows from the workspace's two thresholds, so moving a
-- threshold never rewrites these rows. A narrow table of its own: scores
-- change with every open and click, and contacts carry wide indexes.
CREATE TABLE contact_scores (
    contact_id    uuid        PRIMARY KEY REFERENCES contacts(id) ON DELETE CASCADE,
    workspace_id  uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    score         smallint    NOT NULL CHECK (score BETWEEN 0 AND 100),
    events_part   smallint    NOT NULL,
    fields_part   smallint    NOT NULL,
    -- The last human open or click that counted (decay starts from it).
    active_at     timestamptz,
    -- When the score last changed.
    changed_at    timestamptz,
    -- When the score changes next without new activity: the next decay
    -- step, or the oldest counted event leaving the 180-day window.
    next_check_at timestamptz,
    version       integer     NOT NULL,
    computed_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX contact_scores_score_idx ON contact_scores (workspace_id, score DESC, contact_id);
CREATE INDEX contact_scores_check_idx ON contact_scores (workspace_id, next_check_at) WHERE next_check_at IS NOT NULL;

-- What the scorer counts, copied from where it happened (delivery events,
-- form submissions, API events, consent events) once per source row. A
-- score is a replay of these events under the rules, so a rule change
-- recalculates from here and the profile's 90-day line and breakdown come
-- from the same replay. Kept 270 days: 180 count, 90 more draw the line.
CREATE TABLE score_events (
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    contact_id   uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    -- A UUIDv5 of the source row (message, message + link, submission,
    -- API event, consent event): the same row is never counted twice.
    source_id    uuid        NOT NULL,
    occurred_at  timestamptz NOT NULL,
    kind         text        NOT NULL CHECK (kind IN ('open','click','form','api_event','soft_bounce','unsubscribe')),
    -- The link (clicks), the form id, the API event's name or the list id
    -- (unsubscribes).
    ref          text        NOT NULL DEFAULT '',
    -- The lists the email went to (opens and clicks).
    lists        uuid[]      NOT NULL DEFAULT '{}',
    message_id   uuid,
    -- A soft bounce delivered after a retry is kept, uncounted: the
    -- breakdown explains its 0.
    counted      boolean     NOT NULL DEFAULT true,
    PRIMARY KEY (contact_id, source_id)
);
CREATE INDEX score_events_contact_idx ON score_events (contact_id, occurred_at);
CREATE INDEX score_events_time_idx ON score_events (workspace_id, occurred_at);

-- A recalculation of every contact after a rule change (or the first
-- set-up). Scores are written to scoring_recalc_scores and applied at the
-- end, so contacts keep their previous score until it finishes.
CREATE TABLE scoring_recalcs (
    id                uuid        PRIMARY KEY,
    workspace_id      uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    version           integer     NOT NULL,
    reason            text        NOT NULL CHECK (reason IN ('setup','rules','retry')),
    -- superseded: a newer save started its own recalculation.
    status            text        NOT NULL DEFAULT 'running' CHECK (status IN ('running','finished','failed','superseded')),
    -- The save dialog's answer: start "Score crosses…" automations for
    -- contacts who cross only because the rules changed.
    start_automations boolean     NOT NULL DEFAULT false,
    total             bigint      NOT NULL DEFAULT 0,
    done              bigint      NOT NULL DEFAULT 0,
    -- Contacts are recalculated in id order; a retry resumes after this one.
    cursor            uuid,
    -- What changed, in the words of the save ("Rule 3: +15 → +25"), and
    -- as data for the app's own sentence: {"rules":[3],"decay":false,"bands":false}.
    label             text        NOT NULL DEFAULT '',
    changes           jsonb       NOT NULL DEFAULT '{}',
    started_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    started_at        timestamptz NOT NULL DEFAULT now(),
    finished_at       timestamptz,
    failed_at         timestamptz,
    error             text        NOT NULL DEFAULT '',
    -- After it finishes: {"changed":1312,"moves":[{"from":"warm","to":"hot","count":1108}],
    -- "automations":[{"id":"…","name":"Sales-ready handoff","crossed":1108,"started":0}]}
    summary           jsonb       NOT NULL DEFAULT '{}'
);
CREATE INDEX scoring_recalcs_workspace_idx ON scoring_recalcs (workspace_id, started_at DESC);
CREATE UNIQUE INDEX scoring_recalcs_running_key ON scoring_recalcs (workspace_id) WHERE status = 'running';

-- Scores a running recalculation computed, applied when it finishes.
-- Unlogged: after a crash the job starts over.
CREATE UNLOGGED TABLE scoring_recalc_scores (
    recalc_id     uuid        NOT NULL,
    contact_id    uuid        NOT NULL,
    score         smallint    NOT NULL,
    events_part   smallint    NOT NULL,
    fields_part   smallint    NOT NULL,
    active_at     timestamptz,
    changed_at    timestamptz,
    next_check_at timestamptz,
    PRIMARY KEY (recalc_id, contact_id)
);

-- Contacts to score again on the next tick: those with new activity while
-- a recalculation computed their score.
CREATE TABLE scoring_dirty (
    contact_id   uuid        PRIMARY KEY,
    workspace_id uuid        NOT NULL,
    queued_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX scoring_dirty_workspace_idx ON scoring_dirty (workspace_id);

-- "Score crosses…" starts automations: add the kind to the trigger events,
-- keeping every kind another migration may have added before this one.
-- +goose StatementBegin
DO $$
DECLARE
    def text;
BEGIN
    SELECT pg_get_constraintdef(oid) INTO def FROM pg_constraint
    WHERE conrelid = 'automation_events'::regclass AND conname = 'automation_events_kind_check';
    IF def IS NOT NULL AND position('score_crosses' IN def) = 0 THEN
        ALTER TABLE automation_events DROP CONSTRAINT automation_events_kind_check;
        EXECUTE 'ALTER TABLE automation_events ADD CONSTRAINT automation_events_kind_check '
            || replace(def, 'ARRAY[', 'ARRAY[''score_crosses''::text, ');
    END IF;
END;
$$;
-- +goose StatementEnd

-- The incremental scorer reads each workspace's new opens, clicks and soft
-- bounces by arrival, its form submissions and API events by time, and list
-- unsubscribes by time.
CREATE INDEX delivery_events_engagement_idx ON delivery_events (workspace_id, received_at) WHERE type IN ('open','click','soft_bounce');
CREATE INDEX form_submissions_workspace_time_idx ON form_submissions (workspace_id, created_at);
CREATE INDEX automation_events_api_time_idx ON automation_events (workspace_id, occurred_at) WHERE kind = 'api_event';
CREATE INDEX consent_events_unsubscribed_idx ON consent_events (occurred_at) WHERE action = 'unsubscribed';

-- +goose Down
DROP INDEX consent_events_unsubscribed_idx;
DROP INDEX automation_events_api_time_idx;
DROP INDEX form_submissions_workspace_time_idx;
DROP INDEX delivery_events_engagement_idx;
DELETE FROM automation_events WHERE kind = 'score_crosses';
-- +goose StatementBegin
DO $$
DECLARE
    def text;
BEGIN
    SELECT pg_get_constraintdef(oid) INTO def FROM pg_constraint
    WHERE conrelid = 'automation_events'::regclass AND conname = 'automation_events_kind_check';
    IF def IS NOT NULL AND position('score_crosses' IN def) > 0 THEN
        ALTER TABLE automation_events DROP CONSTRAINT automation_events_kind_check;
        EXECUTE 'ALTER TABLE automation_events ADD CONSTRAINT automation_events_kind_check '
            || replace(replace(def, '''score_crosses''::text, ', ''), ', ''score_crosses''::text', '');
    END IF;
END;
$$;
-- +goose StatementEnd
DROP TABLE scoring_dirty;
DROP TABLE scoring_recalc_scores;
DROP TABLE scoring_recalcs;
DROP TABLE score_events;
DROP TABLE contact_scores;
DROP TABLE scoring_versions;
DROP TABLE scoring_configs;
