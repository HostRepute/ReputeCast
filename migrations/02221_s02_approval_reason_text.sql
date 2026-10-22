-- +goose Up
-- Why an approval was invalidated, kept as a message beside its English
-- (invalidated_reason) so each reader reads it in their own language
-- (internal/approvals). Requests invalidated before have none and read in
-- English.
ALTER TABLE approval_requests ADD COLUMN invalidated_reason_text jsonb;

-- +goose Down
ALTER TABLE approval_requests DROP COLUMN invalidated_reason_text;
