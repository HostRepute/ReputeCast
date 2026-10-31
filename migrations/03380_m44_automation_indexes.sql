-- +goose NO TRANSACTION
-- +goose Up
-- Automations, complete (review 2026-11 M44), built without locking writes.

-- AT-2: the contacts at a step, the latest to arrive first, a page at a time.
CREATE INDEX CONCURRENTLY IF NOT EXISTS automation_runs_at_node_idx
    ON automation_runs (automation_id, node_id, node_entered_at DESC, id DESC) WHERE status IN ('active', 'waiting');
-- AT-2: the flows a contact is in, on their profile.
CREATE INDEX CONCURRENTLY IF NOT EXISTS automation_runs_by_contact_idx ON automation_runs (contact_id, entered_at DESC);
-- AT-8: finished runs whose trigger data is still kept.
CREATE INDEX CONCURRENTLY IF NOT EXISTS automation_runs_finished_state_idx
    ON automation_runs (finished_at) WHERE finished_at IS NOT NULL AND state <> '{}'::jsonb;
-- AT-11: "does a live flow listen to this?"
CREATE INDEX CONCURRENTLY IF NOT EXISTS automations_live_triggers_idx
    ON automations USING gin (triggers jsonb_path_ops) WHERE status = 'live' AND deleted_at IS NULL;

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS automations_live_triggers_idx;
DROP INDEX CONCURRENTLY IF EXISTS automation_runs_finished_state_idx;
DROP INDEX CONCURRENTLY IF EXISTS automation_runs_by_contact_idx;
DROP INDEX CONCURRENTLY IF EXISTS automation_runs_at_node_idx;
