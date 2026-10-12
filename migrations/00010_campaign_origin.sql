-- +goose Up
-- origin: where a campaign came from ('' = written here, or the importer
-- source such as 'brevo' for history pulled in by the migration importer).
-- note: a short annotation shown under the name in lists.
ALTER TABLE campaigns ADD COLUMN origin text NOT NULL DEFAULT '';
ALTER TABLE campaigns ADD COLUMN note text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE campaigns DROP COLUMN note;
ALTER TABLE campaigns DROP COLUMN origin;
