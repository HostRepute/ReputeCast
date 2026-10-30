-- +goose NO TRANSACTION
-- +goose Up
-- A campaign's messages that went in doubt (03310): the report counts those
-- left out. Few rows, so a small index; built concurrently, so an upgrade
-- doesn't hold up sends on a large messages table.
CREATE INDEX CONCURRENTLY IF NOT EXISTS messages_doubt_idx ON messages (campaign_id) WHERE doubt_at IS NOT NULL;

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS messages_doubt_idx;
