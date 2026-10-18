-- +goose Up
-- AI assistant with the workspace's own model (design expansion M22, phase 3
-- brief §3.4): the workspace's one model connection, the AI activity log
-- (each request, never its content, 90 days), report summaries people kept,
-- and the "Translated by AI · not reviewed" mark on language versions.

-- The workspace's model. One connection per workspace; disconnecting
-- removes the key (key_sealed NULL, disconnected_at set) and keeps the
-- model, the cap and the prices for the next connection.
CREATE TABLE ai_connections (
    workspace_id        uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    provider            text        NOT NULL CHECK (provider IN ('anthropic','openai','google','azure','compatible')),
    -- A name for a server the workspace runs ("Ollama on the office network").
    label               text        NOT NULL DEFAULT '',
    -- The Azure endpoint or the OpenAI-compatible server's address; empty is
    -- the provider's own API.
    base_url            text        NOT NULL DEFAULT '',
    -- Azure OpenAI: the deployment and the API version.
    deployment          text        NOT NULL DEFAULT '',
    api_version         text        NOT NULL DEFAULT '',
    model               text        NOT NULL,
    model_name          text        NOT NULL DEFAULT '',
    -- Whether the model reads images (alt text needs it).
    image_input         boolean     NOT NULL DEFAULT true,
    -- The API key, sealed with internal/secrets ("ai:<workspace>:key").
    -- Write-only: never returned. NULL for a keyless local server and once
    -- disconnected.
    key_sealed          bytea,
    key_added_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    key_added_at        timestamptz,
    connected_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    connected_at        timestamptz NOT NULL DEFAULT now(),
    disconnected_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    disconnected_at     timestamptz,
    -- key_rejected: the provider answered 401/403 to the saved key; features
    -- show an error until it's replaced.
    status              text        NOT NULL DEFAULT 'ok' CHECK (status IN ('ok','key_rejected')),
    status_detail       text        NOT NULL DEFAULT '',
    status_at           timestamptz,
    last_test_at        timestamptz,
    last_test_ok        boolean,
    last_test_ms        integer,
    -- The monthly cap: dollars (estimated from the token counts and the
    -- prices), tokens (a local model), or none.
    cap_kind            text        NOT NULL DEFAULT 'usd' CHECK (cap_kind IN ('usd','tokens','none')),
    cap_usd             numeric(10,2) NOT NULL DEFAULT 25,
    cap_tokens          bigint      NOT NULL DEFAULT 0,
    cap_set_by          uuid        REFERENCES users(id) ON DELETE SET NULL,
    cap_set_at          timestamptz,
    -- Dollars per million tokens, pre-filled from ReputeCast's price list
    -- (prices_from is its date) and editable.
    input_price         numeric(10,4) NOT NULL DEFAULT 0,
    output_price        numeric(10,4) NOT NULL DEFAULT 0,
    prices_from         date,
    -- Features switched off (all on when connected).
    features_off        text[]      NOT NULL DEFAULT '{}',
    -- The month ("2026-10") whose 80% notice and cap notice went out.
    notice_month        text        NOT NULL DEFAULT '',
    cap_notice_month    text        NOT NULL DEFAULT '',
    -- The demo's sample connection answers from recordings and never calls
    -- the provider (development only).
    demo                boolean     NOT NULL DEFAULT false,
    updated_by          uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at          timestamptz NOT NULL DEFAULT now()
);

-- The AI activity log: one row per request to the model, kept 90 days. It
-- never stores what was sent or returned: who, when, which feature, on
-- what, the model, the tokens, the estimated cost and the outcome.
CREATE TABLE ai_requests (
    id             uuid PRIMARY KEY,
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id        uuid        REFERENCES users(id) ON DELETE SET NULL,
    feature        text        NOT NULL CHECK (feature IN ('subject_ideas','rewrite','translate','alt_text','accessibility_fix','segment_draft','report_summary')),
    -- The rewrite's action (shorten, friendlier, formal, direct, instruction)
    -- or the language translated to.
    action         text        NOT NULL DEFAULT '',
    target_kind    text        NOT NULL DEFAULT '' CHECK (target_kind IN ('','campaign','segment','report')),
    target_id      uuid,
    -- What the request was about, in words ("Weekly Digest #143, 2 images").
    target_label   text        NOT NULL DEFAULT '',
    provider       text        NOT NULL,
    model          text        NOT NULL,
    input_tokens   integer     NOT NULL DEFAULT 0,
    output_tokens  integer     NOT NULL DEFAULT 0,
    -- Estimated cost in millionths of a dollar.
    cost_micros    bigint      NOT NULL DEFAULT 0,
    outcome        text        NOT NULL DEFAULT 'pending' CHECK (outcome IN ('pending','accepted','accepted_edits','kept','rejected','discarded','refused','error','nothing_drafted','cancelled')),
    -- "1 of 5", "529 overloaded", "rate limited".
    outcome_detail text        NOT NULL DEFAULT '',
    duration_ms    integer     NOT NULL DEFAULT 0,
    created_at     timestamptz NOT NULL DEFAULT now(),
    decided_at     timestamptz
);
CREATE INDEX ai_requests_workspace_idx ON ai_requests (workspace_id, created_at DESC);
CREATE INDEX ai_requests_created_idx ON ai_requests (created_at);

-- A report summary someone kept ("Summary by AI, kept by Linh Nguyen"). It
-- sits at the top of the report for members and never on read-only links.
CREATE TABLE ai_report_summaries (
    campaign_id  uuid PRIMARY KEY REFERENCES campaigns(id) ON DELETE CASCADE,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    summary      text        NOT NULL,
    model        text        NOT NULL,
    model_name   text        NOT NULL DEFAULT '',
    request_id   uuid,
    kept_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    kept_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ai_report_summaries_workspace_idx ON ai_report_summaries (workspace_id);

-- A language version written from an accepted AI translation stays
-- "Translated by AI · not reviewed" while ai_version is its current
-- version: an edit (a new version) or "Mark as reviewed" ends it.
ALTER TABLE content_translations
    ADD COLUMN ai_model       text    NOT NULL DEFAULT '',
    ADD COLUMN ai_version     integer NOT NULL DEFAULT 0,
    ADD COLUMN ai_reviewed_by uuid    REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN ai_reviewed_at timestamptz;

-- +goose Down
ALTER TABLE content_translations
    DROP COLUMN ai_reviewed_at,
    DROP COLUMN ai_reviewed_by,
    DROP COLUMN ai_version,
    DROP COLUMN ai_model;
DROP TABLE ai_report_summaries;
DROP TABLE ai_requests;
DROP TABLE ai_connections;
