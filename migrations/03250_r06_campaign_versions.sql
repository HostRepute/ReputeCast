-- +goose Up
-- Composer versions (review 2026-10, CO-2). A person can name a version;
-- named versions are listed on their own and never pruned. "Load theirs"
-- after a conflict keeps your edits in the history as a version of their
-- own (conflict); an automatic section merge (merge), edits recovered from
-- a browser's journal (recovered) and a synced block's update (synced) say
-- so too.
ALTER TABLE campaign_versions ADD COLUMN name text NOT NULL DEFAULT '' CHECK (length(name) <= 120);
ALTER TABLE campaign_versions DROP CONSTRAINT campaign_versions_source_check;
ALTER TABLE campaign_versions ADD CONSTRAINT campaign_versions_source_check
    CHECK (source IN ('create','autosave','manual','restore','template','import','duplicate','api','conflict','merge','recovered','synced'));
CREATE INDEX campaign_versions_named_idx ON campaign_versions (campaign_id, version DESC) WHERE name <> '';

-- +goose Down
DROP INDEX campaign_versions_named_idx;
UPDATE campaign_versions SET source = 'manual' WHERE source IN ('conflict','merge','recovered','synced');
ALTER TABLE campaign_versions DROP CONSTRAINT campaign_versions_source_check;
ALTER TABLE campaign_versions ADD CONSTRAINT campaign_versions_source_check
    CHECK (source IN ('create','autosave','manual','restore','template','import','duplicate','api'));
ALTER TABLE campaign_versions DROP COLUMN name;
