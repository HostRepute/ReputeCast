-- +goose Up
-- Model identity is operational metadata, independent of expiring content.
ALTER TABLE ai_agent_runs ADD COLUMN model text NOT NULL DEFAULT '';
UPDATE ai_agent_runs r SET model = coalesce(c.bindings->'Model'->>'Model', '')
FROM ai_agent_contexts c WHERE c.run_id = r.id;

-- +goose Down
ALTER TABLE ai_agent_runs DROP COLUMN model;
