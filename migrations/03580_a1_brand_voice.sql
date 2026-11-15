-- +goose Up
-- Voice revisions are separate from visual brand settings. Saving a voice
-- never propagates synced blocks or changes an existing email.
CREATE TABLE brand_voice_revisions (
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    revision integer NOT NULL CHECK (revision > 0),
    content jsonb NOT NULL CHECK (octet_length(content::text) <= 40000),
    source_campaigns uuid[] NOT NULL DEFAULT '{}' CHECK (cardinality(source_campaigns) <= 10),
    request_id uuid,
    created_by uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL,
    PRIMARY KEY (workspace_id, revision),
    UNIQUE (workspace_id, request_id)
);

-- A learned suggestion is bound to its author, sources and base revision.
-- The activity log keeps only its ID and outcome, never the content.
CREATE TABLE brand_voice_suggestions (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    base_revision integer NOT NULL CHECK (base_revision >= 0),
    content jsonb NOT NULL CHECK (octet_length(content::text) <= 40000),
    source_campaigns uuid[] NOT NULL CHECK (cardinality(source_campaigns) BETWEEN 1 AND 10),
    expires_at timestamptz NOT NULL
);
CREATE INDEX brand_voice_suggestions_expiry_idx ON brand_voice_suggestions (expires_at);

ALTER TABLE ai_requests DROP CONSTRAINT ai_requests_feature_check;
ALTER TABLE ai_requests ADD CONSTRAINT ai_requests_feature_check CHECK (feature IN
    ('subject_ideas','rewrite','translate','alt_text','accessibility_fix','segment_draft','report_summary','brand_voice'));
ALTER TABLE ai_requests ADD COLUMN voice_revision integer NOT NULL DEFAULT 0 CHECK (voice_revision >= 0);

-- +goose Down
ALTER TABLE ai_requests DROP COLUMN voice_revision;
DELETE FROM ai_requests WHERE feature = 'brand_voice';
ALTER TABLE ai_requests DROP CONSTRAINT ai_requests_feature_check;
ALTER TABLE ai_requests ADD CONSTRAINT ai_requests_feature_check CHECK (feature IN
    ('subject_ideas','rewrite','translate','alt_text','accessibility_fix','segment_draft','report_summary'));
DROP TABLE brand_voice_suggestions;
DROP TABLE brand_voice_revisions;
