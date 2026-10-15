-- +goose Up
-- Welcome-series prompts (add-on transactional §6): List health, the form
-- builder and the import summary offer the Welcome series starter for a
-- list no live automation starts on. "Not now" dismisses it for that list,
-- for the whole workspace.
CREATE TABLE welcome_prompt_dismissals (
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    list_id      uuid        NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    dismissed_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    dismissed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, list_id)
);

-- +goose Down
DROP TABLE welcome_prompt_dismissals;
