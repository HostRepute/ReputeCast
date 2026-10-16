-- +goose Up
-- Settings › Approvals: the optional rule "Publishing a transactional template
-- needs approval" (add-on transactional §3.6). A request covers one draft
-- of a template (its draft hash in the snapshot); once approved, publishing
-- that draft goes ahead.
ALTER TABLE approval_requests DROP CONSTRAINT approval_requests_subject_type_check;
ALTER TABLE approval_requests ADD CONSTRAINT approval_requests_subject_type_check
    CHECK (subject_type IN ('campaign','automation','transactional_template'));
ALTER TABLE approval_requests
    ADD COLUMN transactional_template_id uuid REFERENCES transactional_templates(id) ON DELETE CASCADE;
-- One pending request per template.
CREATE UNIQUE INDEX approval_requests_open_transactional ON approval_requests (transactional_template_id)
    WHERE status = 'pending' AND transactional_template_id IS NOT NULL;
CREATE INDEX approval_requests_transactional_idx ON approval_requests (transactional_template_id)
    WHERE transactional_template_id IS NOT NULL;

-- +goose Down
DELETE FROM approval_requests WHERE subject_type = 'transactional_template';
DROP INDEX approval_requests_transactional_idx;
DROP INDEX approval_requests_open_transactional;
ALTER TABLE approval_requests DROP COLUMN transactional_template_id;
ALTER TABLE approval_requests DROP CONSTRAINT approval_requests_subject_type_check;
ALTER TABLE approval_requests ADD CONSTRAINT approval_requests_subject_type_check
    CHECK (subject_type IN ('campaign','automation'));
