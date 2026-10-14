-- +goose Up
-- A campaign's messages go out in a shuffled order: each gets a
-- random send_order when the recipient list is built. The first minutes of
-- a send are then a fair sample of its audience, so the auto-pause
-- thresholds trip on what is representative instead of after one import,
-- signup source or engagement tier went out in full (the order contacts
-- were created in). Automation mail keeps 0 and goes out first come, first
-- served.
ALTER TABLE messages ADD COLUMN send_order integer NOT NULL DEFAULT 0;
DROP INDEX messages_send_queue_idx;
CREATE INDEX messages_send_queue_idx ON messages (campaign_id, provider_id, mailbox_provider, send_order, id) WHERE status = 'queued';

-- +goose Down
DROP INDEX messages_send_queue_idx;
CREATE INDEX messages_send_queue_idx ON messages (campaign_id, provider_id, mailbox_provider, id) WHERE status = 'queued';
ALTER TABLE messages DROP COLUMN send_order;
