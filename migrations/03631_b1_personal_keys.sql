-- +goose Up
-- Personal keys (assistant brief D4): a member's own model accounts, when
-- an integration manager turns them on. Their use has its own monthly cap
-- per person and never counts against the workspace's.

ALTER TABLE ai_workspace_settings
    ADD COLUMN personal_keys       boolean       NOT NULL DEFAULT false,
    ADD COLUMN personal_cap_usd    numeric(10,2) NOT NULL DEFAULT 10,
    ADD COLUMN personal_cap_set_by uuid          REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN personal_cap_set_at timestamptz;

-- The person a personal account belongs to; NULL is the workspace's.
ALTER TABLE ai_accounts ADD COLUMN user_id uuid REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX ai_accounts_user_idx ON ai_accounts (workspace_id, user_id) WHERE user_id IS NOT NULL;

-- The model of their own a person picked for a feature, for their own
-- requests. The workspace's defaults (ai_feature_models) are never personal.
CREATE TABLE ai_member_feature_models (
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    feature      text NOT NULL CHECK (feature IN ('assistant','agent','subject_ideas','rewrite','translate','alt_text','accessibility_fix','segment_draft','report_summary','brand_voice','draft')),
    model_id     uuid NOT NULL,
    PRIMARY KEY (workspace_id, user_id, feature),
    FOREIGN KEY (workspace_id, model_id) REFERENCES ai_models(workspace_id, id) ON DELETE CASCADE
);
CREATE INDEX ai_member_feature_models_model_idx ON ai_member_feature_models (workspace_id, model_id);

-- A request on a personal key: its user_id is the key's owner, and it
-- counts against that person's cap only.
ALTER TABLE ai_requests ADD COLUMN personal boolean NOT NULL DEFAULT false;
ALTER TABLE ai_budget_reservations ADD COLUMN personal boolean NOT NULL DEFAULT false;
CREATE INDEX ai_requests_personal_idx ON ai_requests (workspace_id, user_id, created_at) WHERE personal;
CREATE INDEX ai_budget_reservations_personal_idx ON ai_budget_reservations (workspace_id, user_id, created_at) WHERE personal AND state <> 'settled';

-- +goose Down
DROP INDEX ai_budget_reservations_personal_idx;
DROP INDEX ai_requests_personal_idx;
ALTER TABLE ai_budget_reservations DROP COLUMN personal;
ALTER TABLE ai_requests DROP COLUMN personal;
DROP TABLE ai_member_feature_models;
DELETE FROM ai_accounts WHERE user_id IS NOT NULL;
DROP INDEX ai_accounts_user_idx;
ALTER TABLE ai_accounts DROP COLUMN user_id;
ALTER TABLE ai_workspace_settings
    DROP COLUMN personal_cap_set_at,
    DROP COLUMN personal_cap_set_by,
    DROP COLUMN personal_cap_usd,
    DROP COLUMN personal_keys;
