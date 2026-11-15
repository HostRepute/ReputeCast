-- +goose Up
CREATE INDEX ai_draft_proposals_pending_idx
    ON ai_draft_proposals(workspace_id, user_id, kind, id DESC)
    WHERE decision = '';

-- +goose Down
DROP INDEX ai_draft_proposals_pending_idx;
