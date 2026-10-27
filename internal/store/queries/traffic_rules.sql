-- Traffic rules (Sending providers › Traffic rules, review 2026-10, M34):
-- the canary defaults, the complaint watch after a send and the mailbox
-- provider response rules of a workspace.

-- name: TrafficRulesGet :many
SELECT * FROM traffic_rules WHERE workspace_id = @workspace_id;

-- name: TrafficRulesSave :exec
INSERT INTO traffic_rules (workspace_id, canary, watch, updated_by, updated_at)
VALUES (@workspace_id, @canary, @watch, @updated_by, @updated_at)
ON CONFLICT (workspace_id) DO UPDATE SET canary = EXCLUDED.canary, watch = EXCLUDED.watch,
    updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at;

-- name: TrafficReplyRules :many
-- A workspace's mailbox provider response rules, in their order (GS-12).
SELECT * FROM traffic_reply_rules WHERE workspace_id = @workspace_id ORDER BY position, created_at;

-- name: TrafficReplyRulesEnabled :many
SELECT * FROM traffic_reply_rules WHERE workspace_id = @workspace_id AND enabled ORDER BY position, created_at;

-- name: TrafficReplyRuleInsert :exec
INSERT INTO traffic_reply_rules (id, workspace_id, mailbox_provider, contains, min_count, window_seconds, action, duration_seconds, enabled, position, note,
    created_by, created_at)
VALUES (@id, @workspace_id, @mailbox_provider, @contains, @min_count, @window_seconds, @action, sqlc.narg(duration_seconds), @enabled, @position, @note,
    sqlc.narg(created_by), @created_at);

-- name: TrafficReplyRuleUpdate :execrows
UPDATE traffic_reply_rules SET mailbox_provider = @mailbox_provider, contains = @contains, min_count = @min_count, window_seconds = @window_seconds,
    action = @action, duration_seconds = sqlc.narg(duration_seconds), enabled = @enabled, position = @position, note = @note
WHERE id = @id AND workspace_id = @workspace_id;

-- name: TrafficReplyRuleDelete :execrows
DELETE FROM traffic_reply_rules WHERE id = @id AND workspace_id = @workspace_id;

-- name: TrafficReplyRuleMatched :exec
-- A rule acted in a send.
UPDATE traffic_reply_rules SET last_matched_at = @at, last_campaign_id = @campaign_id WHERE id = @id;

-- name: TrafficRecentActions :many
-- What the traffic rules did lately in a workspace: auto-pauses,
-- slowdowns, reply rules, canaries and complaint alerts, newest first.
SELECT i.*, c.name AS campaign_name FROM campaign_incidents i JOIN campaigns c ON c.id = i.campaign_id
WHERE c.workspace_id = @workspace_id AND i.tripped_at >= @since AND i.kind IN ('threshold', 'reply_rule', 'canary', 'complaint_watch')
ORDER BY i.tripped_at DESC LIMIT @lim;

-- name: TrafficRecentAudiences :many
-- The audiences of a workspace's last 8 sends that went out: the canary's
-- usual size is their median.
SELECT recipients FROM campaign_sends
WHERE workspace_id = @workspace_id AND recipients > 0 AND phase IN ('sending', 'finished')
ORDER BY created_at DESC LIMIT 8;
