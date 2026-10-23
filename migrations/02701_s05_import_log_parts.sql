-- +goose Up
-- An import's live log keeps, beside the English line it stores, the
-- messages the line came from (internal/i18n Text, app languages), so each
-- reader sees it in their own language: [{"id": "imports.log.batch",
-- "args": {...}}, ...], joined with " · " as the English is. Lines written
-- before keep reading in their English.
ALTER TABLE import_log ADD COLUMN message_parts jsonb NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE import_log DROP COLUMN message_parts;
