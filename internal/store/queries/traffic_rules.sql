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
