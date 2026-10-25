-- +goose Up
-- Contacts on company domains are classified from their MX records
-- (internal/mailbox): Google Workspace as Gmail, Microsoft 365 as Outlook,
-- Yahoo-hosted domains as Yahoo, iCloud+ domains as iCloud. The column says
-- when that happened; consumer domains are classified by name when the
-- contact is added and never wait for it.
ALTER TABLE contacts ADD COLUMN mailbox_checked_at timestamptz;
CREATE INDEX contacts_mailbox_pending_idx ON contacts (split_part(email, '@', 2))
    WHERE mailbox_checked_at IS NULL AND mailbox_provider = 'other' AND erased_at IS NULL;

-- +goose Down
DROP INDEX contacts_mailbox_pending_idx;
ALTER TABLE contacts DROP COLUMN mailbox_checked_at;
