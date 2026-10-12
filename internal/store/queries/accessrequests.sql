-- Access requests from the "403 · no permission" screen (internal/accessrequests).

-- name: LastAccessRequest :one
-- The member's latest request for this role in the last day (one a day).
SELECT occurred_at
FROM audit_events
WHERE action = 'access.request'
  AND actor_id = @user_id
  AND workspace_id = @workspace_id
  AND target_id = @role_key::text
  AND occurred_at > now() - interval '24 hours'
ORDER BY occurred_at DESC
LIMIT 1;
