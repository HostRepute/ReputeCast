-- +goose Up
-- Saved views keep their counts (review 2026-11 AU-3): the Contacts chips
-- and Audience › Health read them, and they are counted again once ten
-- minutes old or when the view's filters change. stats holds the count by
-- tier and the contacts waiting for their opt-in. updated_at dates a
-- rename or a change of filters or columns (AU-8, AU-9).
ALTER TABLE saved_views
    ADD COLUMN stats      jsonb,
    ADD COLUMN counted_at timestamptz,
    ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();

-- +goose Down
ALTER TABLE saved_views DROP COLUMN updated_at, DROP COLUMN counted_at, DROP COLUMN stats;
