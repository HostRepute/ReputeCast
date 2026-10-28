-- +goose Up
-- Synced blocks beyond the Header and Footer (review 2026-10, R4.A3,
-- CO-11): any section can be saved as a synced block, under a name the
-- person gives it. Its kind is a key, "b-" and 12 hex digits, that the
-- emails holding it carry in rc-synced, as they carry "header" and
-- "footer". The new column has a constant default (no table rewrite) and
-- the wider check is added NOT VALID, then validated.
ALTER TABLE synced_blocks ADD COLUMN name text NOT NULL DEFAULT '';
ALTER TABLE synced_blocks ADD CONSTRAINT synced_blocks_name_check CHECK (length(name) <= 80) NOT VALID;
ALTER TABLE synced_blocks VALIDATE CONSTRAINT synced_blocks_name_check;
ALTER TABLE synced_blocks DROP CONSTRAINT synced_blocks_kind_check;
ALTER TABLE synced_blocks ADD CONSTRAINT synced_blocks_kind_check
    CHECK (kind IN ('header', 'footer') OR kind ~ '^b-[0-9a-f]{12}$') NOT VALID;
ALTER TABLE synced_blocks VALIDATE CONSTRAINT synced_blocks_kind_check;

-- +goose Down
-- The named blocks go; emails holding one keep its content (rc-synced is
-- left out of the compiled HTML, and the composer offers to unlink it).
DELETE FROM synced_blocks WHERE kind NOT IN ('header', 'footer');
ALTER TABLE synced_blocks DROP CONSTRAINT synced_blocks_kind_check;
ALTER TABLE synced_blocks ADD CONSTRAINT synced_blocks_kind_check CHECK (kind IN ('header', 'footer'));
ALTER TABLE synced_blocks DROP CONSTRAINT synced_blocks_name_check;
ALTER TABLE synced_blocks DROP COLUMN name;
