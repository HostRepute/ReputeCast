-- +goose Up
-- Several model accounts per workspace, several models per account and a
-- default model per feature (assistant brief D3). The workspace's one
-- connection (ai_connections) splits into its workspace settings, its
-- account and its model, and that model becomes every feature's default.
-- The activity log, the reservations, the caps and the prices stay.

-- The workspace's AI settings: the monthly cap, the features switched off
-- and the notices. Its row is the AI configuration lock: every change to
-- the accounts, models and defaults, and every reservation, takes it first.
CREATE TABLE ai_workspace_settings (
    workspace_id     uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    cap_kind         text          NOT NULL DEFAULT 'usd' CHECK (cap_kind IN ('usd','tokens','none')),
    cap_usd          numeric(10,2) NOT NULL DEFAULT 25,
    cap_tokens       bigint        NOT NULL DEFAULT 0,
    cap_set_by       uuid          REFERENCES users(id) ON DELETE SET NULL,
    cap_set_at       timestamptz,
    features_off     text[]        NOT NULL DEFAULT '{}',
    notice_month     text          NOT NULL DEFAULT '',
    cap_notice_month text          NOT NULL DEFAULT '',
    revision         bigint        NOT NULL DEFAULT 1,
    updated_by       uuid          REFERENCES users(id) ON DELETE SET NULL,
    updated_at       timestamptz   NOT NULL DEFAULT now()
);

-- A model account: a provider, its address and its key. Disconnecting
-- removes the key (key_sealed NULL, disconnected_at set) and keeps the
-- account's models for the next connection.
CREATE TABLE ai_accounts (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    provider        text        NOT NULL CHECK (provider IN ('anthropic','openai','google','azure','compatible')),
    label           text        NOT NULL DEFAULT '',
    base_url        text        NOT NULL DEFAULT '',
    api_version     text        NOT NULL DEFAULT '',
    -- The API key, sealed with internal/secrets ("ai:<account>:key").
    -- Write-only: never returned.
    key_sealed      bytea,
    key_added_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    key_added_at    timestamptz,
    connected_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    connected_at    timestamptz NOT NULL,
    disconnected_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    disconnected_at timestamptz,
    status          text        NOT NULL DEFAULT 'ok' CHECK (status IN ('ok','key_rejected')),
    status_detail   text        NOT NULL DEFAULT '',
    status_at       timestamptz,
    last_test_at    timestamptz,
    last_test_ok    boolean,
    last_test_ms    integer,
    demo            boolean     NOT NULL DEFAULT false,
    revision        bigint      NOT NULL DEFAULT 1,
    created_at      timestamptz NOT NULL,
    updated_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at      timestamptz NOT NULL,
    UNIQUE (workspace_id, id)
);
CREATE INDEX ai_accounts_workspace_idx ON ai_accounts (workspace_id, created_at, id);

-- A model of an account, with the prices its requests are estimated with.
CREATE TABLE ai_models (
    id           uuid PRIMARY KEY,
    workspace_id uuid          NOT NULL,
    account_id   uuid          NOT NULL,
    model        text          NOT NULL,
    model_name   text          NOT NULL DEFAULT '',
    -- Azure OpenAI: the deployment.
    deployment   text          NOT NULL DEFAULT '',
    image_input  boolean       NOT NULL DEFAULT true,
    input_price  numeric(10,4) NOT NULL DEFAULT 0,
    output_price numeric(10,4) NOT NULL DEFAULT 0,
    prices_from  date,
    price_known  boolean       NOT NULL DEFAULT false,
    revision     bigint        NOT NULL DEFAULT 1,
    created_at   timestamptz   NOT NULL,
    updated_by   uuid          REFERENCES users(id) ON DELETE SET NULL,
    updated_at   timestamptz   NOT NULL,
    UNIQUE (workspace_id, id),
    UNIQUE (account_id, model, deployment),
    FOREIGN KEY (workspace_id, account_id) REFERENCES ai_accounts(workspace_id, id) ON DELETE CASCADE
);

-- The model each feature uses. A model can't be removed while it is a
-- feature's default (the workspace's deletion removes both).
CREATE TABLE ai_feature_models (
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    feature      text NOT NULL CHECK (feature IN ('assistant','agent','subject_ideas','rewrite','translate','alt_text','accessibility_fix','segment_draft','report_summary','brand_voice','draft')),
    model_id     uuid NOT NULL,
    PRIMARY KEY (workspace_id, feature),
    FOREIGN KEY (workspace_id, model_id) REFERENCES ai_models(workspace_id, id)
);
CREATE INDEX ai_feature_models_model_idx ON ai_feature_models (workspace_id, model_id);

INSERT INTO ai_workspace_settings (workspace_id, cap_kind, cap_usd, cap_tokens, cap_set_by, cap_set_at, features_off, notice_month, cap_notice_month, updated_by, updated_at)
SELECT workspace_id, cap_kind, cap_usd, cap_tokens, cap_set_by, cap_set_at, features_off, notice_month, cap_notice_month, updated_by, updated_at
FROM ai_connections;

-- The first account keeps the workspace's id: its key was sealed with
-- "ai:<workspace>:key", which is "ai:<account>:key" for it. The database
-- can't open a key to seal it again.
INSERT INTO ai_accounts (id, workspace_id, provider, label, base_url, api_version, key_sealed, key_added_by, key_added_at,
    connected_by, connected_at, disconnected_by, disconnected_at, status, status_detail, status_at,
    last_test_at, last_test_ok, last_test_ms, demo, created_at, updated_by, updated_at)
SELECT workspace_id, workspace_id, provider, label, base_url, api_version, key_sealed, key_added_by, key_added_at,
    connected_by, connected_at, disconnected_by, disconnected_at, status, status_detail, status_at,
    last_test_at, last_test_ok, last_test_ms, demo, connected_at, updated_by, updated_at
FROM ai_connections;

-- Ids are UUIDv7 like the ones the app makes (the time in milliseconds,
-- then random bits); Postgres 16 and 17 have no uuidv7().
INSERT INTO ai_models (id, workspace_id, account_id, model, model_name, deployment, image_input, input_price, output_price,
    prices_from, price_known, created_at, updated_by, updated_at)
SELECT encode(set_bit(set_bit(overlay(uuid_send(gen_random_uuid())
           PLACING substring(int8send((extract(epoch FROM clock_timestamp()) * 1000)::bigint) FROM 3) FROM 1 FOR 6), 52, 1), 53, 1), 'hex')::uuid,
    workspace_id, workspace_id, model, model_name, deployment, image_input, input_price, output_price,
    prices_from, price_known, connected_at, updated_by, updated_at
FROM ai_connections;

INSERT INTO ai_feature_models (workspace_id, feature, model_id)
SELECT m.workspace_id, f.feature, m.id
FROM ai_models m
CROSS JOIN (VALUES ('assistant'),('agent'),('subject_ideas'),('rewrite'),('translate'),('alt_text'),('accessibility_fix'),
    ('segment_draft'),('report_summary'),('brand_voice'),('draft')) AS f (feature);

-- A run keeps the model it started with; a removed model pauses it.
ALTER TABLE ai_agent_runs ADD COLUMN model_id uuid REFERENCES ai_models(id) ON DELETE SET NULL;
UPDATE ai_agent_runs r SET model_id = m.id FROM ai_models m WHERE m.workspace_id = r.workspace_id;

-- Tool support is a model's.
ALTER TABLE ai_agent_capabilities ADD COLUMN model_id uuid REFERENCES ai_models(id) ON DELETE CASCADE;
UPDATE ai_agent_capabilities c SET model_id = m.id FROM ai_models m WHERE m.workspace_id = c.workspace_id;
DELETE FROM ai_agent_capabilities WHERE model_id IS NULL;
ALTER TABLE ai_agent_capabilities DROP CONSTRAINT ai_agent_capabilities_pkey,
    ALTER COLUMN model_id SET NOT NULL,
    ADD PRIMARY KEY (model_id);
CREATE INDEX ai_agent_capabilities_workspace_idx ON ai_agent_capabilities (workspace_id);

-- Which account and model a request used. The log and the reservations
-- outlive both, so these aren't references; earlier rows have none.
ALTER TABLE ai_requests ADD COLUMN account_id uuid, ADD COLUMN model_id uuid;
ALTER TABLE ai_budget_reservations ADD COLUMN account_id uuid, ADD COLUMN model_id uuid;

DROP TABLE ai_connections;

-- +goose Down
-- Each workspace's first account and its first model become its one
-- connection again; other accounts and models are dropped. A key sealed for
-- an account made after the upgrade can't open as the workspace's: that
-- connection comes back disconnected.
CREATE TABLE ai_connections (
    workspace_id        uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    provider            text        NOT NULL CHECK (provider IN ('anthropic','openai','google','azure','compatible')),
    label               text        NOT NULL DEFAULT '',
    base_url            text        NOT NULL DEFAULT '',
    deployment          text        NOT NULL DEFAULT '',
    api_version         text        NOT NULL DEFAULT '',
    model               text        NOT NULL,
    model_name          text        NOT NULL DEFAULT '',
    image_input         boolean     NOT NULL DEFAULT true,
    key_sealed          bytea,
    key_added_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    key_added_at        timestamptz,
    connected_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    connected_at        timestamptz NOT NULL DEFAULT now(),
    disconnected_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    disconnected_at     timestamptz,
    status              text        NOT NULL DEFAULT 'ok' CHECK (status IN ('ok','key_rejected')),
    status_detail       text        NOT NULL DEFAULT '',
    status_at           timestamptz,
    last_test_at        timestamptz,
    last_test_ok        boolean,
    last_test_ms        integer,
    cap_kind            text        NOT NULL DEFAULT 'usd' CHECK (cap_kind IN ('usd','tokens','none')),
    cap_usd             numeric(10,2) NOT NULL DEFAULT 25,
    cap_tokens          bigint      NOT NULL DEFAULT 0,
    cap_set_by          uuid        REFERENCES users(id) ON DELETE SET NULL,
    cap_set_at          timestamptz,
    input_price         numeric(10,4) NOT NULL DEFAULT 0,
    output_price        numeric(10,4) NOT NULL DEFAULT 0,
    prices_from         date,
    features_off        text[]      NOT NULL DEFAULT '{}',
    notice_month        text        NOT NULL DEFAULT '',
    cap_notice_month    text        NOT NULL DEFAULT '',
    demo                boolean     NOT NULL DEFAULT false,
    updated_by          uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at          timestamptz NOT NULL DEFAULT now(),
    price_known         boolean     NOT NULL DEFAULT false
);
INSERT INTO ai_connections (workspace_id, provider, label, base_url, deployment, api_version, model, model_name, image_input,
    key_sealed, key_added_by, key_added_at, connected_by, connected_at, disconnected_by, disconnected_at,
    status, status_detail, status_at, last_test_at, last_test_ok, last_test_ms,
    cap_kind, cap_usd, cap_tokens, cap_set_by, cap_set_at, input_price, output_price, prices_from,
    features_off, notice_month, cap_notice_month, demo, updated_by, updated_at, price_known)
SELECT s.workspace_id, a.provider, a.label, a.base_url, m.deployment, a.api_version, m.model, m.model_name, m.image_input,
    CASE WHEN a.id = a.workspace_id THEN a.key_sealed END,
    CASE WHEN a.id = a.workspace_id THEN a.key_added_by END,
    CASE WHEN a.id = a.workspace_id THEN a.key_added_at END,
    a.connected_by, a.connected_at,
    CASE WHEN a.id <> a.workspace_id AND a.key_sealed IS NOT NULL AND a.disconnected_at IS NULL THEN NULL ELSE a.disconnected_by END,
    CASE WHEN a.id <> a.workspace_id AND a.key_sealed IS NOT NULL AND a.disconnected_at IS NULL THEN now() ELSE a.disconnected_at END,
    a.status, a.status_detail, a.status_at, a.last_test_at, a.last_test_ok, a.last_test_ms,
    s.cap_kind, s.cap_usd, s.cap_tokens, s.cap_set_by, s.cap_set_at, m.input_price, m.output_price, m.prices_from,
    s.features_off, s.notice_month, s.cap_notice_month, a.demo, s.updated_by, greatest(s.updated_at, a.updated_at, m.updated_at), m.price_known
FROM ai_workspace_settings s
JOIN LATERAL (
    SELECT a.id AS account_id, m.id AS model_id
    FROM ai_accounts a JOIN ai_models m ON m.account_id = a.id
    WHERE a.workspace_id = s.workspace_id
    ORDER BY a.created_at, a.id, m.created_at, m.id LIMIT 1
) p ON true
JOIN ai_accounts a ON a.id = p.account_id
JOIN ai_models m ON m.id = p.model_id;

ALTER TABLE ai_budget_reservations DROP COLUMN model_id, DROP COLUMN account_id;
ALTER TABLE ai_requests DROP COLUMN model_id, DROP COLUMN account_id;

DROP INDEX ai_agent_capabilities_workspace_idx;
DELETE FROM ai_agent_capabilities c WHERE c.model_id IS DISTINCT FROM (
    SELECT m.id FROM ai_accounts a JOIN ai_models m ON m.account_id = a.id
    WHERE a.workspace_id = c.workspace_id
    ORDER BY a.created_at, a.id, m.created_at, m.id LIMIT 1);
ALTER TABLE ai_agent_capabilities DROP CONSTRAINT ai_agent_capabilities_pkey,
    DROP COLUMN model_id,
    ADD PRIMARY KEY (workspace_id);

ALTER TABLE ai_agent_runs DROP COLUMN model_id;

DROP TABLE ai_feature_models;
DROP TABLE ai_models;
DROP TABLE ai_accounts;
DROP TABLE ai_workspace_settings;
