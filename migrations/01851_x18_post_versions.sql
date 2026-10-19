-- +goose Up
-- A published post keeps the content it was published with: edits after
-- publishing stay a draft until they're published too. Blocks for a paid
-- tier are drawn apart, for the readers who hold it.
ALTER TABLE website_posts
    ADD COLUMN published_mjml    text    NOT NULL DEFAULT '',
    ADD COLUMN published_version integer NOT NULL DEFAULT 0,
    ADD COLUMN paid_blocks       jsonb   NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE website_posts DROP COLUMN paid_blocks, DROP COLUMN published_version, DROP COLUMN published_mjml;
