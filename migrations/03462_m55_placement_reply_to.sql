-- +goose Up
-- A placement test keeps its email's reply-to address (M55, G-1): when the
-- workspace includes a mailto: address in List-Unsubscribe, the copies
-- carry the reply-to, else the From, like a real send.

ALTER TABLE placement_tests ADD COLUMN reply_to text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE placement_tests DROP COLUMN reply_to;
