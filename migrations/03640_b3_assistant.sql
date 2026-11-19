-- +goose Up
-- The AI assistant's threads (assistant brief D1, D7; B3). A thread is one
-- person's: nobody else reads it, managers included. Its content (title,
-- messages, attached snapshots, stream events) is sealed with the
-- instance's key; the usage ledger (ai_requests, ai_budget_reservations)
-- holds none of it.

CREATE TABLE ai_assistant_settings (
    workspace_id      uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    enabled           boolean NOT NULL DEFAULT false,
    tool_limit        integer NOT NULL DEFAULT 40 CHECK (tool_limit BETWEEN 1 AND 200),
    token_limit       bigint  NOT NULL DEFAULT 300000 CHECK (token_limit BETWEEN 1000 AND 2000000),
    cost_limit_micros bigint  NOT NULL DEFAULT 2000000 CHECK (cost_limit_micros BETWEEN 1 AND 100000000),
    -- Answering turns at once: per person and per workspace (one per thread).
    per_person        integer NOT NULL DEFAULT 2 CHECK (per_person BETWEEN 1 AND 10),
    per_workspace     integer NOT NULL DEFAULT 10 CHECK (per_workspace BETWEEN 1 AND 100),
    revision          bigint  NOT NULL DEFAULT 1,
    updated_by        uuid REFERENCES users(id) ON DELETE SET NULL,
    updated_at        timestamptz NOT NULL
);

CREATE TABLE ai_assistant_threads (
    id                uuid PRIMARY KEY,
    workspace_id      uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id           uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- The model the thread's turns use; NULL once that model was removed.
    model_id          uuid REFERENCES ai_models(id) ON DELETE SET NULL,
    title_sealed      bytea,
    -- expired: retention removed the content; the row stays as a tombstone.
    state             text NOT NULL DEFAULT 'open' CHECK (state IN ('open', 'expired')),
    -- The screen the thread was opened on (a route key, never content).
    screen            text NOT NULL DEFAULT '' CHECK (screen ~ '^[a-z0-9_.-]{0,80}$'),
    -- The last stream event's number: events are numbered per thread
    -- without gaps, in commit order.
    event_seq         bigint  NOT NULL DEFAULT 0 CHECK (event_seq >= 0),
    tool_calls        integer NOT NULL DEFAULT 0 CHECK (tool_calls >= 0),
    tool_limit        integer NOT NULL CHECK (tool_limit BETWEEN 1 AND 200),
    token_limit       bigint  NOT NULL CHECK (token_limit BETWEEN 1000 AND 2000000),
    cost_limit_micros bigint  NOT NULL CHECK (cost_limit_micros BETWEEN 1 AND 100000000),
    -- A retry with the same key returns this thread. No digest of the
    -- request is kept: it would confirm guesses about the sealed title.
    request_key       text CHECK (request_key ~ '^[A-Za-z0-9_-]{1,80}$'),
    created_at        timestamptz NOT NULL,
    updated_at        timestamptz NOT NULL,
    last_activity_at  timestamptz NOT NULL,
    -- Content is kept 30 days from the last activity.
    expires_at        timestamptz NOT NULL,
    UNIQUE (workspace_id, id),
    CHECK (state = 'open' OR title_sealed IS NULL)
);
CREATE INDEX ai_assistant_threads_owner_idx ON ai_assistant_threads (workspace_id, user_id, last_activity_at DESC, id DESC);
CREATE INDEX ai_assistant_threads_expiry_idx ON ai_assistant_threads (expires_at) WHERE state = 'open';
CREATE INDEX ai_assistant_threads_model_idx ON ai_assistant_threads (model_id) WHERE model_id IS NOT NULL;
CREATE UNIQUE INDEX ai_assistant_threads_request_key_idx ON ai_assistant_threads (workspace_id, user_id, request_key) WHERE request_key IS NOT NULL;

-- A turn is one message of the person and the assistant's answer. Its
-- version fences a worker: Stop, a lost lease or retention bumps it.
CREATE TABLE ai_assistant_turns (
    id           uuid PRIMARY KEY,
    thread_id    uuid NOT NULL REFERENCES ai_assistant_threads(id) ON DELETE CASCADE,
    sequence     integer NOT NULL CHECK (sequence >= 1),
    model_id     uuid REFERENCES ai_models(id) ON DELETE SET NULL,
    version      bigint NOT NULL DEFAULT 1,
    state        text NOT NULL CHECK (state IN ('queued', 'answering', 'waiting', 'stopped', 'failed', 'done')),
    reason       text NOT NULL DEFAULT '' CHECK (reason ~ '^[a-z_]{0,40}$'),
    -- A retry with the same key is the same turn; it is compared with the
    -- sealed message, never with a digest of it.
    request_key  text NOT NULL CHECK (request_key ~ '^[A-Za-z0-9_-]{1,80}$'),
    retry_of     uuid REFERENCES ai_assistant_turns(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL,
    started_at   timestamptz,
    ended_at     timestamptz,
    UNIQUE (thread_id, sequence),
    UNIQUE (thread_id, request_key),
    CHECK ((ended_at IS NOT NULL) = (state IN ('stopped', 'failed', 'done')))
);
-- One answering turn per thread.
CREATE UNIQUE INDEX ai_assistant_turns_active_idx ON ai_assistant_turns (thread_id) WHERE state IN ('queued', 'answering', 'waiting');
CREATE INDEX ai_assistant_turns_ended_idx ON ai_assistant_turns (ended_at) WHERE ended_at IS NOT NULL;

-- One model request of a turn (an answer or a summary). Its id is its
-- budget reservation's and its activity log row's. The request itself is
-- never stored, nor a digest of it: it is assembled again from the sealed
-- messages.
CREATE TABLE ai_assistant_steps (
    id             uuid PRIMARY KEY,
    turn_id        uuid NOT NULL REFERENCES ai_assistant_turns(id) ON DELETE CASCADE,
    sequence       integer NOT NULL CHECK (sequence BETWEEN 1 AND 250),
    turn_version   bigint NOT NULL,
    kind           text NOT NULL CHECK (kind IN ('answer', 'summary')),
    state          text NOT NULL CHECK (state IN ('requested', 'responded', 'done', 'uncertain', 'awaiting_confirmation')),
    created_at     timestamptz NOT NULL,
    updated_at     timestamptz NOT NULL,
    UNIQUE (turn_id, sequence)
);

CREATE TABLE ai_assistant_messages (
    id             uuid PRIMARY KEY,
    thread_id      uuid NOT NULL REFERENCES ai_assistant_threads(id) ON DELETE CASCADE,
    turn_id        uuid REFERENCES ai_assistant_turns(id) ON DELETE CASCADE,
    sequence       integer NOT NULL CHECK (sequence >= 1),
    role           text NOT NULL CHECK (role IN ('user', 'assistant', 'tool', 'summary')),
    -- The model that wrote it: provider context is never replayed to another.
    model_id       uuid,
    content_sealed bytea NOT NULL,
    bytes          integer NOT NULL CHECK (bytes BETWEEN 0 AND 2000000),
    created_at     timestamptz NOT NULL,
    UNIQUE (thread_id, sequence)
);
CREATE INDEX ai_assistant_messages_turn_idx ON ai_assistant_messages (turn_id) WHERE turn_id IS NOT NULL;

-- What the person attached and confirmed sharing. Each read checks their
-- access again; a report's numbers are its fixed snapshot.
CREATE TABLE ai_assistant_contexts (
    id              uuid PRIMARY KEY,
    thread_id       uuid NOT NULL REFERENCES ai_assistant_threads(id) ON DELETE CASCADE,
    kind            text NOT NULL CHECK (kind IN ('campaign', 'post', 'report', 'segment', 'automation')),
    target_id       uuid NOT NULL,
    revision        bigint NOT NULL DEFAULT 0 CHECK (revision >= 0),
    snapshot_sealed bytea,
    shared_at       timestamptz NOT NULL,
    UNIQUE (thread_id, kind, target_id)
);

-- Stream events, kept 24 hours after their turn ends. Losing them in a
-- crash is harmless: a reader that finds a gap gets the thread's snapshot.
CREATE UNLOGGED TABLE ai_assistant_events (
    thread_id      uuid NOT NULL REFERENCES ai_assistant_threads(id) ON DELETE CASCADE,
    seq            bigint NOT NULL CHECK (seq >= 1),
    turn_id        uuid REFERENCES ai_assistant_turns(id) ON DELETE CASCADE,
    kind           text NOT NULL CHECK (kind IN ('turn', 'delta', 'tool', 'message', 'usage', 'confirmation')),
    payload_sealed bytea NOT NULL,
    created_at     timestamptz NOT NULL,
    PRIMARY KEY (thread_id, seq)
);
CREATE INDEX ai_assistant_events_turn_idx ON ai_assistant_events (turn_id);

-- An answering turn's slot: a lease, renewed while it streams, that fences
-- a second worker and counts against the person's and the workspace's
-- limits. Separate from the agent's slots.
CREATE TABLE ai_assistant_slots (
    thread_id    uuid PRIMARY KEY REFERENCES ai_assistant_threads(id) ON DELETE CASCADE,
    turn_id      uuid NOT NULL UNIQUE REFERENCES ai_assistant_turns(id) ON DELETE CASCADE,
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ws_slot      integer NOT NULL CHECK (ws_slot BETWEEN 1 AND 100),
    user_slot    integer NOT NULL CHECK (user_slot BETWEEN 1 AND 10),
    lease_id     uuid NOT NULL,
    expires_at   timestamptz NOT NULL,
    UNIQUE (workspace_id, ws_slot),
    UNIQUE (workspace_id, user_id, user_slot)
);

-- A thread's cumulative ceilings count its requests' reservations. The
-- ledger outlives a deleted thread.
ALTER TABLE ai_budget_reservations ADD COLUMN thread_id uuid REFERENCES ai_assistant_threads(id) ON DELETE SET NULL;
CREATE INDEX ai_budget_reservations_thread_idx ON ai_budget_reservations (thread_id) WHERE thread_id IS NOT NULL;

ALTER TABLE ai_requests DROP CONSTRAINT ai_requests_feature_check;
ALTER TABLE ai_requests ADD CONSTRAINT ai_requests_feature_check CHECK (feature IN
    ('subject_ideas','rewrite','translate','alt_text','accessibility_fix','segment_draft','report_summary','brand_voice','draft','agent','connection_test','assistant'));
ALTER TABLE ai_requests DROP CONSTRAINT ai_requests_outcome_check;
ALTER TABLE ai_requests ADD CONSTRAINT ai_requests_outcome_check CHECK (outcome IN
    ('pending','accepted','accepted_edits','kept','rejected','discarded','refused','error','nothing_drafted','cancelled','answered'));
-- An assistant request is on its thread (target_label stays empty: a
-- thread's title is content).
ALTER TABLE ai_requests DROP CONSTRAINT ai_requests_target_kind_check;
ALTER TABLE ai_requests ADD CONSTRAINT ai_requests_target_kind_check
    CHECK (target_kind IN ('','campaign','segment','report','automation','assistant_thread'));

-- +goose Down
DELETE FROM ai_requests WHERE feature = 'assistant' OR outcome = 'answered' OR target_kind = 'assistant_thread';
ALTER TABLE ai_requests DROP CONSTRAINT ai_requests_target_kind_check;
ALTER TABLE ai_requests ADD CONSTRAINT ai_requests_target_kind_check
    CHECK (target_kind IN ('','campaign','segment','report','automation'));
ALTER TABLE ai_requests DROP CONSTRAINT ai_requests_outcome_check;
ALTER TABLE ai_requests ADD CONSTRAINT ai_requests_outcome_check CHECK (outcome IN
    ('pending','accepted','accepted_edits','kept','rejected','discarded','refused','error','nothing_drafted','cancelled'));
ALTER TABLE ai_requests DROP CONSTRAINT ai_requests_feature_check;
ALTER TABLE ai_requests ADD CONSTRAINT ai_requests_feature_check CHECK (feature IN
    ('subject_ideas','rewrite','translate','alt_text','accessibility_fix','segment_draft','report_summary','brand_voice','draft','agent','connection_test'));
DELETE FROM ai_budget_reservations WHERE feature = 'assistant';
DROP INDEX ai_budget_reservations_thread_idx;
ALTER TABLE ai_budget_reservations DROP COLUMN thread_id;
DROP TABLE ai_assistant_slots;
DROP TABLE ai_assistant_events;
DROP TABLE ai_assistant_contexts;
DROP TABLE ai_assistant_messages;
DROP TABLE ai_assistant_steps;
DROP TABLE ai_assistant_turns;
DROP TABLE ai_assistant_threads;
DROP TABLE ai_assistant_settings;
