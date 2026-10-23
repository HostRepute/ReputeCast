-- +goose Up
-- A fix applied from Review & send keeps its summary's messages beside the
-- English summary, so Send Readiness writes it in each reader's language
-- (internal/accessibility, FixRecord.SummaryIn): a list of i18n.Text per
-- fix. Fixes recorded before keep reading in English.
ALTER TABLE accessibility_fixes ADD COLUMN summary_texts jsonb NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE accessibility_fixes DROP COLUMN summary_texts;
