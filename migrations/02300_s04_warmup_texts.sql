-- +goose Up
-- Server text kept for later readers (internal/i18n Text, app languages):
-- beside the English a warm-up plan stores for people to read later (its
-- log's titles and bodies, the plan's status detail), the messages they
-- came from with their arguments, so the Warm-up tab reads in each
-- reader's language. Keys name the column the English is in: {"title":
-- {...}, "body": {...}} on a log entry, {"status_detail": {...}} on a plan.
-- Rows written before have none and keep reading in English.
ALTER TABLE warmup_events ADD COLUMN texts jsonb NOT NULL DEFAULT '{}';
ALTER TABLE warmup_plans ADD COLUMN texts jsonb NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE warmup_plans DROP COLUMN texts;
ALTER TABLE warmup_events DROP COLUMN texts;
