-- +goose Up
-- A key the client generates once per brief makes creating a run safe to
-- retry after a lost response: the retry returns the same run instead of
-- paying for a second plan.
ALTER TABLE ai_agent_runs
    ADD COLUMN request_key text CHECK (request_key ~ '^[A-Za-z0-9_-]{1,80}$'),
    ADD COLUMN request_hash bytea CHECK (octet_length(request_hash) = 32),
    ADD CONSTRAINT ai_agent_runs_request_check CHECK ((request_key IS NULL) = (request_hash IS NULL));
CREATE UNIQUE INDEX ai_agent_runs_request_key_idx ON ai_agent_runs(workspace_id, user_id, request_key) WHERE request_key IS NOT NULL;

-- +goose Down
DROP INDEX ai_agent_runs_request_key_idx;
ALTER TABLE ai_agent_runs DROP CONSTRAINT ai_agent_runs_request_check, DROP COLUMN request_hash, DROP COLUMN request_key;
