-- Default mailbox provider response rules (review GS-12): every workspace
-- starts with the rules for the replies mailbox providers are known for
-- (rules.DefaultReplyRules, which new workspaces get when they're made).
-- Workspaces made before them get them here, unless they already have a
-- rule of their own: those keep exactly what they set up.
--
-- Ids are UUIDv7 like the ones the app makes (the time in milliseconds,
-- then random bits); Postgres 16 and 17 have no uuidv7().

-- +goose Up
INSERT INTO traffic_reply_rules (id, workspace_id, mailbox_provider, contains, min_count, window_seconds, action, duration_seconds, enabled, position, note, created_at)
SELECT encode(set_bit(set_bit(overlay(uuid_send(gen_random_uuid())
           PLACING substring(int8send((extract(epoch FROM clock_timestamp()) * 1000)::bigint) FROM 3) FROM 1 FOR 6), 52, 1), 53, 1), 'hex')::uuid,
       w.id, d.mailbox_provider, d.contains, d.min_count, d.window_seconds, d.action, d.duration_seconds, true, d.position, '', now()
FROM workspaces w
CROSS JOIN (VALUES
    ('yahoo', 'TSS04', 10, 600, 'pause', 7200, 0),
    ('yahoo', 'TS04', 10, 600, 'pause', 7200, 1),
    ('outlook', 'S3150', 1, 600, 'pause_route', NULL, 2),
    ('gmail', '421-4.7.28', 5, 600, 'slow', 1800, 3)
) AS d (mailbox_provider, contains, min_count, window_seconds, action, duration_seconds, position)
WHERE NOT EXISTS (SELECT 1 FROM traffic_reply_rules r WHERE r.workspace_id = w.id);

-- +goose Down
-- Only the defaults nobody changed, made by no one and never matched.
DELETE FROM traffic_reply_rules r
USING (VALUES
    ('yahoo', 'TSS04', 10, 600, 'pause', 7200, 0),
    ('yahoo', 'TS04', 10, 600, 'pause', 7200, 1),
    ('outlook', 'S3150', 1, 600, 'pause_route', NULL, 2),
    ('gmail', '421-4.7.28', 5, 600, 'slow', 1800, 3)
) AS d (mailbox_provider, contains, min_count, window_seconds, action, duration_seconds, position)
WHERE r.mailbox_provider = d.mailbox_provider AND r.contains = d.contains AND r.min_count = d.min_count AND r.window_seconds = d.window_seconds
  AND r.action = d.action AND r.duration_seconds IS NOT DISTINCT FROM d.duration_seconds AND r.position = d.position
  AND r.enabled AND r.note = '' AND r.created_by IS NULL AND r.last_matched_at IS NULL;
