-- +goose Up
ALTER TABLE ai_requests DROP CONSTRAINT ai_requests_target_kind_check;
ALTER TABLE ai_requests ADD CONSTRAINT ai_requests_target_kind_check
    CHECK (target_kind IN ('','campaign','segment','report','automation'));

-- +goose Down
UPDATE ai_requests SET target_kind = '', target_id = NULL WHERE target_kind = 'automation';
ALTER TABLE ai_requests DROP CONSTRAINT ai_requests_target_kind_check;
ALTER TABLE ai_requests ADD CONSTRAINT ai_requests_target_kind_check
    CHECK (target_kind IN ('','campaign','segment','report'));
