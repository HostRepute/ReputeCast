-- +goose NO TRANSACTION
-- +goose Up
-- The contacts table at a million rows (review 2026-11 AU-2, AU-3, AU-9).
-- Search matches part of an address or a name through trigram indexes;
-- each order of the table has an index that ends with the id its keyset
-- pages on; the Source filter has its own. Built concurrently, then the
-- activity index it replaces is dropped. pg_trgm is a trusted extension:
-- the database's owner can create it.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX CONCURRENTLY IF NOT EXISTS contacts_email_trgm_idx ON contacts USING gin (email gin_trgm_ops);
CREATE INDEX CONCURRENTLY IF NOT EXISTS contacts_name_trgm_idx ON contacts USING gin (lower(first_name || ' ' || last_name) gin_trgm_ops);
CREATE INDEX CONCURRENTLY IF NOT EXISTS contacts_source_idx ON contacts (workspace_id, source);
CREATE INDEX CONCURRENTLY IF NOT EXISTS contacts_activity_order_idx ON contacts (workspace_id, last_activity_at DESC NULLS LAST, id DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS contacts_joined_order_idx ON contacts (workspace_id, created_at DESC, id DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS contacts_last_open_order_idx ON contacts (workspace_id, last_open_at DESC NULLS LAST, id DESC);
DROP INDEX CONCURRENTLY IF EXISTS contacts_activity_idx;
DROP INDEX CONCURRENTLY IF EXISTS contacts_created_idx;

-- +goose Down
CREATE INDEX CONCURRENTLY IF NOT EXISTS contacts_created_idx ON contacts (workspace_id, created_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS contacts_activity_idx ON contacts (workspace_id, last_activity_at DESC NULLS LAST, id);
DROP INDEX CONCURRENTLY IF EXISTS contacts_last_open_order_idx;
DROP INDEX CONCURRENTLY IF EXISTS contacts_joined_order_idx;
DROP INDEX CONCURRENTLY IF EXISTS contacts_activity_order_idx;
DROP INDEX CONCURRENTLY IF EXISTS contacts_source_idx;
DROP INDEX CONCURRENTLY IF EXISTS contacts_name_trgm_idx;
DROP INDEX CONCURRENTLY IF EXISTS contacts_email_trgm_idx;
