-- +goose Up
-- Who ran a Send Readiness run, or started the one in progress, when that
-- was an API key or an agent (security review). ran_by and started_by name
-- people only: a key's run used to name the person who made the key. The
-- key or agent is kept as the audit log names it.
ALTER TABLE readiness_runs
    ADD COLUMN ran_by_actor_type text NOT NULL DEFAULT '' CHECK (ran_by_actor_type IN ('', 'api_key', 'mcp_agent')),
    ADD COLUMN ran_by_actor_name text NOT NULL DEFAULT '';
ALTER TABLE readiness_progress
    ADD COLUMN started_by_actor_type text NOT NULL DEFAULT '' CHECK (started_by_actor_type IN ('', 'api_key', 'mcp_agent')),
    ADD COLUMN started_by_actor_name text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE readiness_progress DROP COLUMN started_by_actor_name, DROP COLUMN started_by_actor_type;
ALTER TABLE readiness_runs DROP COLUMN ran_by_actor_name, DROP COLUMN ran_by_actor_type;
