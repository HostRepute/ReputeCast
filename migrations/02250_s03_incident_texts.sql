-- +goose Up
-- Server text kept for later readers (internal/i18n Text, app languages):
-- beside the English title and detail an incident stores for people to read
-- later, the messages they came from with their arguments, so the incident
-- timeline reads in each reader's language. Keys name the column the
-- English is in: {"title": {...}, "detail": {...}}. Incidents written before
-- have none and keep reading in English.
ALTER TABLE incidents ADD COLUMN IF NOT EXISTS texts jsonb NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE incidents DROP COLUMN IF EXISTS texts;
