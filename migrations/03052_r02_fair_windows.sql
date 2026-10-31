-- +goose Up
-- GS-8. The sends that took part of a mailbox provider's window: a slot is
-- shared fairly among the sends going out at once, each taking at most the
-- cap divided among the sends of the slot before.
ALTER TABLE sending_mailbox_windows ADD COLUMN senders uuid[] NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE sending_mailbox_windows DROP COLUMN senders;
