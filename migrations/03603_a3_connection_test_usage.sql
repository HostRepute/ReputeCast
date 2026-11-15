-- +goose Up
ALTER TABLE ai_requests DROP CONSTRAINT ai_requests_feature_check;
ALTER TABLE ai_requests ADD CONSTRAINT ai_requests_feature_check CHECK (feature IN
    ('subject_ideas','rewrite','translate','alt_text','accessibility_fix','segment_draft','report_summary','brand_voice','draft','agent','connection_test'));

-- +goose Down
DELETE FROM ai_requests WHERE feature = 'connection_test';
ALTER TABLE ai_requests DROP CONSTRAINT ai_requests_feature_check;
ALTER TABLE ai_requests ADD CONSTRAINT ai_requests_feature_check CHECK (feature IN
    ('subject_ideas','rewrite','translate','alt_text','accessibility_fix','segment_draft','report_summary','brand_voice','draft','agent'));
