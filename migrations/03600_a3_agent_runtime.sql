-- +goose Up
ALTER TABLE ai_connections ADD COLUMN price_known boolean NOT NULL DEFAULT false;
UPDATE ai_connections SET price_known = true WHERE prices_from IS NOT NULL OR input_price > 0 OR output_price > 0;
ALTER TABLE ai_requests ADD COLUMN usage_known boolean NOT NULL DEFAULT true;

CREATE TABLE ai_agent_settings (
    workspace_id uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    enabled boolean NOT NULL DEFAULT false,
    tool_limit integer NOT NULL DEFAULT 40 CHECK (tool_limit BETWEEN 1 AND 200),
    token_limit bigint NOT NULL DEFAULT 300000 CHECK (token_limit BETWEEN 1000 AND 2000000),
    cost_limit_micros bigint NOT NULL DEFAULT 2000000 CHECK (cost_limit_micros BETWEEN 1 AND 100000000),
    revision bigint NOT NULL DEFAULT 1,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE ai_agent_runs (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    state text NOT NULL CHECK (state IN ('brief','planning','needs_answers','ready','queued','building','paused','review','completed','cancelled','expired')),
    version bigint NOT NULL DEFAULT 1,
    plan_revision bigint NOT NULL DEFAULT 0,
    accepted_revision bigint NOT NULL DEFAULT 0,
    tool_limit integer NOT NULL CHECK (tool_limit BETWEEN 1 AND 200),
    token_limit bigint NOT NULL CHECK (token_limit BETWEEN 1000 AND 2000000),
    cost_limit_micros bigint NOT NULL CHECK (cost_limit_micros BETWEEN 1 AND 100000000),
    tool_calls integer NOT NULL DEFAULT 0 CHECK (tool_calls >= 0),
    pause_reason text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    closed_at timestamptz,
    UNIQUE (workspace_id, id),
    CHECK (accepted_revision <= plan_revision)
);
CREATE INDEX ai_agent_runs_owner_idx ON ai_agent_runs(workspace_id, user_id, updated_at DESC, id DESC);
CREATE INDEX ai_agent_runs_expiry_idx ON ai_agent_runs(expires_at) WHERE state <> 'expired';
CREATE INDEX ai_agent_runs_queue_idx ON ai_agent_runs(created_at, id) WHERE state IN ('planning','queued','building');

-- Content has a shorter lifetime than the content-free usage ledger.
CREATE TABLE ai_agent_contexts (
    run_id uuid PRIMARY KEY REFERENCES ai_agent_runs(id) ON DELETE CASCADE,
    brief jsonb NOT NULL CHECK (octet_length(brief::text) <= 200000),
    plan jsonb NOT NULL DEFAULT '{}' CHECK (octet_length(plan::text) <= 200000),
    bindings jsonb NOT NULL DEFAULT '{}' CHECK (octet_length(bindings::text) <= 2000000),
    messages jsonb NOT NULL DEFAULT '[]' CHECK (octet_length(messages::text) <= 2000000)
);

CREATE TABLE ai_agent_steps (
    id uuid PRIMARY KEY,
    run_id uuid NOT NULL REFERENCES ai_agent_runs(id) ON DELETE CASCADE,
    sequence integer NOT NULL CHECK (sequence BETWEEN 1 AND 250),
    run_version bigint NOT NULL,
    state text NOT NULL CHECK (state IN ('prepared','requested','responded','done','uncertain')),
    request jsonb CHECK (octet_length(request::text) <= 1000000),
    response jsonb CHECK (octet_length(response::text) <= 1000000),
    results jsonb NOT NULL DEFAULT '[]' CHECK (octet_length(results::text) <= 1000000),
    next_call integer NOT NULL DEFAULT 0 CHECK (next_call BETWEEN 0 AND 40),
    summary jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (run_id, sequence)
);

-- Only these staging rows are mutable by tools. Product resources are created
-- by the human Keep/Accept transaction, never visible to live engines early.
CREATE TABLE ai_agent_artifacts (
    id uuid PRIMARY KEY,
    run_id uuid NOT NULL REFERENCES ai_agent_runs(id) ON DELETE CASCADE,
    handle text NOT NULL,
    kind text NOT NULL CHECK (kind IN ('campaign','automation','segment','post')),
    bundle text NOT NULL,
    revision bigint NOT NULL DEFAULT 1,
    target_id uuid,
    base_revision bigint NOT NULL DEFAULT 0,
    content jsonb CHECK (octet_length(content::text) <= 2000000),
    context jsonb CHECK (octet_length(context::text) <= 2000000),
    decision text NOT NULL DEFAULT '' CHECK (decision IN ('','kept','discarded','accepted','rejected','expired')),
    result_id uuid,
    updated_at timestamptz NOT NULL,
    UNIQUE (run_id, handle),
    CHECK ((target_id IS NULL) = (base_revision = 0))
);

-- A lease is a fence, not permission to repeat an uncertain paid request.
CREATE TABLE ai_agent_slots (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    slot integer NOT NULL CHECK (slot IN (1, 2)),
    run_id uuid NOT NULL UNIQUE REFERENCES ai_agent_runs(id) ON DELETE CASCADE,
    lease_id uuid NOT NULL,
    expires_at timestamptz NOT NULL,
    UNIQUE (workspace_id, slot)
);

-- Covers one-shot requests too, so they cannot race an agent past the monthly
-- cap. Unknown usage keeps its full reservation until explicit reconciliation.
CREATE TABLE ai_budget_reservations (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    run_id uuid,
    provider text NOT NULL,
    model text NOT NULL,
    feature text NOT NULL,
    pricing_revision text NOT NULL,
    input_price numeric NOT NULL CHECK (input_price >= 0),
    output_price numeric NOT NULL CHECK (output_price >= 0),
    state text NOT NULL CHECK (state IN ('reserved','uncertain','settled')),
    input_tokens bigint NOT NULL CHECK (input_tokens >= 0),
    output_tokens bigint NOT NULL CHECK (output_tokens >= 0),
    cost_micros bigint NOT NULL CHECK (cost_micros >= 0),
    created_at timestamptz NOT NULL,
    settled_at timestamptz,
    FOREIGN KEY (workspace_id, run_id) REFERENCES ai_agent_runs(workspace_id, id) ON DELETE CASCADE
);
CREATE INDEX ai_budget_reservations_month_idx ON ai_budget_reservations(workspace_id, created_at);
CREATE INDEX ai_budget_reservations_run_idx ON ai_budget_reservations(run_id) WHERE run_id IS NOT NULL;
CREATE INDEX ai_budget_reservations_pending_idx ON ai_budget_reservations(workspace_id, created_at, id) WHERE state <> 'settled';

-- +goose Down
DROP TABLE ai_budget_reservations;
DROP TABLE ai_agent_slots;
DROP TABLE ai_agent_artifacts;
DROP TABLE ai_agent_steps;
DROP TABLE ai_agent_contexts;
DROP TABLE ai_agent_runs;
DROP TABLE ai_agent_settings;
ALTER TABLE ai_connections DROP COLUMN price_known;
ALTER TABLE ai_requests DROP COLUMN usage_known;
