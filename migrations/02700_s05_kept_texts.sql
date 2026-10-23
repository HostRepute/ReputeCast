-- +goose Up
-- Server text kept for later readers (internal/i18n Text, app languages):
-- beside the English a row stores for people to read later, the message it
-- came from with its arguments, so each reader sees it in their own
-- language. Keys name the column the English is in: {"detail": {...},
-- "note": {...}} on a consent event, {"context": {...}, "original_code":
-- {...}} on a suppression. Rows without one keep reading in their English.
ALTER TABLE consent_events ADD COLUMN texts jsonb NOT NULL DEFAULT '{}';
ALTER TABLE suppressions ADD COLUMN texts jsonb NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE suppressions DROP COLUMN texts;
ALTER TABLE consent_events DROP COLUMN texts;
