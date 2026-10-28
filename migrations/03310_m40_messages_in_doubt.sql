-- +goose Up
-- When a message went in doubt (review GS-1): the time of the provider call
-- whose answer was lost. It stays after a person decides, so the live
-- monitor can say how long a provider's idempotency key still covers it,
-- the report can count the messages left out while in doubt ("not sent, in
-- doubt"), and a provider event for one of those still counts it as sent.
-- A nullable column without a default: no table rewrite.
ALTER TABLE messages ADD COLUMN doubt_at timestamptz;
-- The messages in doubt now (few, read through messages_in_doubt_idx):
-- their claim when it was kept, otherwise when they were queued.
UPDATE messages SET doubt_at = coalesce(claimed_at, queued_at) WHERE status = 'in_doubt';

-- +goose Down
ALTER TABLE messages DROP COLUMN doubt_at;
