-- +goose NO TRANSACTION
-- +goose Up
-- Every page counts the workspace's contacts (the shell's and the workspace
-- switcher's subscriber counts, GET /me and /workspaces/{workspace}/me): an
-- index-only scan of the contacts that aren't erased counts 48,000 of them
-- in 3 ms, where scanning the table took 20 ms. Built concurrently, so an
-- upgrade doesn't hold up imports and sign-ups on a large table.
CREATE INDEX CONCURRENTLY IF NOT EXISTS contacts_live_idx ON contacts (workspace_id) WHERE erased_at IS NULL;

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS contacts_live_idx;
