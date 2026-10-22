-- +goose Up
-- A version's summary for readers in other languages (app languages,
-- server text): summary keeps the English it has always had, and
-- summary_text the message it was written from, with its arguments
-- (internal/automations/lang.go), so the version history reads in each
-- reader's language. Versions written before have none and read in
-- English.
ALTER TABLE automation_versions ADD COLUMN summary_text jsonb;

-- +goose Down
ALTER TABLE automation_versions DROP COLUMN summary_text;
