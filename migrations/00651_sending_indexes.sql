-- +goose Up
-- Indexes for deleting a send's messages.
--
-- delivery_events.message_id references messages (ON DELETE SET NULL)
-- without an index, so every message deleted (a deleted campaign, a demo
-- send started again) scanned the whole event log. The tracking endpoints
-- and the contact timeline look a message's events up by it too.
CREATE INDEX delivery_events_message_idx ON delivery_events (message_id) WHERE message_id IS NOT NULL;

-- The demo transport's pending events of a campaign (deleted with it).
CREATE INDEX sending_sim_events_campaign_idx ON sending_sim_events (campaign_id) WHERE campaign_id IS NOT NULL;

-- +goose Down
DROP INDEX sending_sim_events_campaign_idx;
DROP INDEX delivery_events_message_idx;
