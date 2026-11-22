-- +goose Up
-- Keep the engagement timestamp separately from opens and clicks, so a
-- reply still counts after received-message detail is removed. No list
-- consent, paid access, or saved automation graph is changed here.
ALTER TABLE contacts ADD COLUMN last_human_reply_at timestamptz;

UPDATE contacts c SET last_human_reply_at = r.at
FROM (
    SELECT workspace_id, contact_id, max(received_at) AS at
    FROM inbound_messages
    WHERE kind = 'reply' AND NOT auto_reply AND NOT spam AND contact_id IS NOT NULL
    GROUP BY workspace_id, contact_id
) r
WHERE c.id = r.contact_id AND c.workspace_id = r.workspace_id AND c.erased_at IS NULL;

-- +goose Down
ALTER TABLE contacts DROP COLUMN last_human_reply_at;
