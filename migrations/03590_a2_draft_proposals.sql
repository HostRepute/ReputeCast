-- +goose Up
-- One-shot generation has no product-resource side effects until acceptance.
-- Store the exact proposal and base, not a client assertion of AI provenance.
CREATE TABLE ai_draft_proposals (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind IN ('campaign', 'automation')),
    target_id uuid,
    base_revision bigint NOT NULL CHECK (base_revision >= 0),
    context jsonb NOT NULL CHECK (octet_length(context::text) <= 1500000),
    content jsonb NOT NULL CHECK (octet_length(content::text) <= 1500000),
    decision text NOT NULL DEFAULT '' CHECK (decision IN ('', 'accepted', 'rejected')),
    result_id uuid,
    result_revision bigint,
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    CHECK ((target_id IS NULL) = (base_revision = 0)),
    CHECK ((decision = 'accepted') = (result_id IS NOT NULL AND result_revision IS NOT NULL))
);
CREATE INDEX ai_draft_proposals_expiry_idx ON ai_draft_proposals(expires_at);

ALTER TABLE ai_requests DROP CONSTRAINT ai_requests_feature_check;
ALTER TABLE ai_requests ADD CONSTRAINT ai_requests_feature_check CHECK (feature IN
    ('subject_ideas','rewrite','translate','alt_text','accessibility_fix','segment_draft','report_summary','brand_voice','draft'));

-- +goose Down
DELETE FROM ai_requests WHERE feature = 'draft';
ALTER TABLE ai_requests DROP CONSTRAINT ai_requests_feature_check;
ALTER TABLE ai_requests ADD CONSTRAINT ai_requests_feature_check CHECK (feature IN
    ('subject_ideas','rewrite','translate','alt_text','accessibility_fix','segment_draft','report_summary','brand_voice'));
DROP TABLE ai_draft_proposals;
