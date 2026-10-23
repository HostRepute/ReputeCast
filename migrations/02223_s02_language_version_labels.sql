-- +goose Up
-- A language version's saved versions keep the message of their label
-- beside its English ({"id": "languages.label.copiedFrom", "args": {"code":
-- "en"}}), so the version history reads in each reader's language. Labels
-- saved before have none and keep reading in English.
ALTER TABLE content_translation_versions ADD COLUMN label_text jsonb;

-- +goose Down
ALTER TABLE content_translation_versions DROP COLUMN label_text;
