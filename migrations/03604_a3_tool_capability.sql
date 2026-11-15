-- +goose Up
CREATE TABLE ai_agent_capabilities (
    workspace_id uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    connection_hash text NOT NULL,
    supported boolean NOT NULL,
    checked_at timestamptz NOT NULL
);

-- +goose Down
DROP TABLE ai_agent_capabilities;
