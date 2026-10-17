-- +goose Up
-- The message detail lists the webhook deliveries of a transactional
-- message (message.* events carry its ID).
CREATE INDEX webhook_deliveries_message_idx ON webhook_deliveries ((payload->'data'->>'message_id'))
    WHERE event LIKE 'message.%';

-- +goose Down
DROP INDEX webhook_deliveries_message_idx;
