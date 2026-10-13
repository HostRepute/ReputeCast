-- name: ApprovalInsertRequest :exec
INSERT INTO approval_requests (id, workspace_id, subject_type, campaign_id, automation_id, requested_by, requested_at, note, status,
                               matched_rules, approvals_required, snapshot, campaign_version, readiness_run_id, previous_id, bypass_reason)
VALUES (@id, @workspace_id, @subject_type, sqlc.narg(campaign_id), sqlc.narg(automation_id), sqlc.narg(requested_by), @requested_at, @note, @status,
        @matched_rules, @approvals_required, @snapshot, sqlc.narg(campaign_version), sqlc.narg(readiness_run_id), sqlc.narg(previous_id), @bypass_reason);

-- name: ApprovalGet :one
SELECT r.*,
       coalesce(u.name, '')::text AS requested_by_name,
       coalesce(u.email, '')::text AS requested_by_email,
       coalesce(u.avatar_color, '')::text AS requested_by_color,
       coalesce(d.name, '')::text AS decided_by_name,
       coalesce(i.name, '')::text AS invalidated_by_name,
       coalesce(c.name, a.name, '')::text AS subject_name,
       coalesce(c.status, '')::text AS campaign_status,
       c.scheduled_at AS campaign_scheduled_at,
       c.started_at AS campaign_started_at
FROM approval_requests r
LEFT JOIN users u ON u.id = r.requested_by
LEFT JOIN users d ON d.id = r.decided_by
LEFT JOIN users i ON i.id = r.invalidated_by
LEFT JOIN campaigns c ON c.id = r.campaign_id
LEFT JOIN automations a ON a.id = r.automation_id
WHERE r.workspace_id = @workspace_id AND r.id = @id;

-- name: ApprovalLock :one
SELECT * FROM approval_requests WHERE workspace_id = @workspace_id AND id = @id FOR UPDATE;

-- name: ApprovalOpenForCampaign :one
-- The pending or approved request of a campaign.
SELECT * FROM approval_requests WHERE campaign_id = @campaign_id AND status IN ('pending', 'approved')
ORDER BY requested_at DESC LIMIT 1;

-- name: ApprovalLatestForCampaign :one
-- The most recent request of a campaign (bypass records aside).
SELECT r.*,
       coalesce(u.name, '')::text AS requested_by_name,
       coalesce(d.name, '')::text AS decided_by_name,
       coalesce(i.name, '')::text AS invalidated_by_name
FROM approval_requests r
LEFT JOIN users u ON u.id = r.requested_by
LEFT JOIN users d ON d.id = r.decided_by
LEFT JOIN users i ON i.id = r.invalidated_by
WHERE r.campaign_id = @campaign_id AND r.status <> 'bypassed'
ORDER BY r.requested_at DESC LIMIT 1;

-- name: ApprovalOpenRequests :many
-- Open campaign requests of every workspace (the revalidation job).
SELECT r.id, r.workspace_id, r.campaign_id
FROM approval_requests r
JOIN campaigns c ON c.id = r.campaign_id AND c.deleted_at IS NULL AND c.status IN ('draft', 'approval', 'approved', 'scheduled')
JOIN workspaces w ON w.id = r.workspace_id AND w.deleted_at IS NULL
WHERE r.status IN ('pending', 'approved') AND r.subject_type = 'campaign';

-- name: ApprovalSetStatus :exec
UPDATE approval_requests SET status = @status, decided_at = sqlc.narg(decided_at), decided_by = sqlc.narg(decided_by) WHERE id = @id;

-- name: ApprovalInvalidate :exec
UPDATE approval_requests
SET status = 'invalidated', invalidated_at = @invalidated_at, invalidated_reason = @invalidated_reason, changes = @changes,
    invalidated_by = sqlc.narg(invalidated_by), invalidated_from = @invalidated_from
WHERE id = @id;

-- name: ApprovalReopen :exec
-- "Restore … (keeps approval)": the campaign matches the snapshot again.
UPDATE approval_requests
SET status = @status, invalidated_at = NULL, invalidated_reason = '', changes = '[]', invalidated_by = NULL, invalidated_from = ''
WHERE id = @id;

-- name: ApprovalSetReminded :exec
UPDATE approval_requests SET reminded_at = @reminded_at WHERE id = @id;

-- name: ApprovalInsertDecision :exec
INSERT INTO approval_decisions (id, request_id, approver_id, decision, comment, created_at)
VALUES (@id, @request_id, sqlc.narg(approver_id), @decision, @comment, @created_at);

-- name: ApprovalDecisions :many
SELECT d.id, d.request_id, d.approver_id, d.decision, d.comment, d.created_at,
       coalesce(u.name, '')::text AS approver_name,
       coalesce(u.avatar_color, '')::text AS approver_color
FROM approval_decisions d
LEFT JOIN users u ON u.id = d.approver_id
WHERE d.request_id = ANY(@request_ids::uuid[])
ORDER BY d.created_at;

-- name: ApprovalInsertComment :exec
INSERT INTO approval_comments (id, request_id, author_id, body, created_at)
VALUES (@id, @request_id, sqlc.narg(author_id), @body, @created_at);

-- name: ApprovalComments :many
-- The discussion of a request and of the requests it follows.
SELECT c.id, c.request_id, c.author_id, c.body, c.created_at,
       coalesce(u.name, '')::text AS author_name,
       coalesce(u.avatar_color, '')::text AS author_color
FROM approval_comments c
LEFT JOIN users u ON u.id = c.author_id
WHERE c.request_id = ANY(@request_ids::uuid[])
ORDER BY c.created_at;

-- name: ApprovalChain :many
-- A request and the requests it follows, newest first.
WITH RECURSIVE chain AS (
    SELECT r.id, r.previous_id, 0 AS depth FROM approval_requests r WHERE r.id = @id
    UNION ALL
    SELECT p.id, p.previous_id, chain.depth + 1 FROM approval_requests p JOIN chain ON p.id = chain.previous_id WHERE chain.depth < 20
)
SELECT id FROM chain ORDER BY depth;

-- name: ApprovalList :many
-- The approval inbox. Approvers see every request; others only their own.
SELECT r.id, r.subject_type, r.campaign_id, r.automation_id, r.requested_by, r.requested_at, r.status, r.matched_rules, r.approvals_required,
       r.decided_at, r.invalidated_at, r.invalidated_reason, r.changes, r.bypass_reason,
       coalesce((r.snapshot ->> 'audience_count')::bigint, 0)::bigint AS audience_count,
       coalesce(r.snapshot -> 'route', '[]'::jsonb)::jsonb AS route,
       coalesce(r.snapshot -> 'schedule', '{}'::jsonb)::jsonb AS schedule,
       coalesce(r.snapshot -> 'rules', '[]'::jsonb)::jsonb AS rules,
       coalesce(r.snapshot ->> 'subject', '')::text AS subject,
       coalesce(u.name, '')::text AS requested_by_name,
       coalesce(u.email, '')::text AS requested_by_email,
       coalesce(u.avatar_color, '')::text AS requested_by_color,
       coalesce(d.name, '')::text AS decided_by_name,
       coalesce(c.name, a.name, '')::text AS subject_name,
       coalesce(c.status, '')::text AS campaign_status,
       c.scheduled_at AS campaign_scheduled_at,
       c.started_at AS campaign_started_at,
       coalesce((SELECT x.comment FROM approval_decisions x WHERE x.request_id = r.id ORDER BY x.created_at DESC LIMIT 1), '')::text AS last_comment
FROM approval_requests r
LEFT JOIN users u ON u.id = r.requested_by
LEFT JOIN users d ON d.id = r.decided_by
LEFT JOIN campaigns c ON c.id = r.campaign_id
LEFT JOIN automations a ON a.id = r.automation_id
WHERE r.workspace_id = @workspace_id AND r.status <> 'bypassed'
  AND (@all_requests::boolean OR r.requested_by = @viewer::uuid)
  AND r.requested_at >= @since::timestamptz
  AND (sqlc.narg(author)::uuid IS NULL OR r.requested_by = sqlc.narg(author)::uuid)
  AND (sqlc.narg(approver)::uuid IS NULL OR r.decided_by = sqlc.narg(approver)::uuid
       OR EXISTS (SELECT 1 FROM approval_decisions y WHERE y.request_id = r.id AND y.approver_id = sqlc.narg(approver)::uuid))
  AND NOT EXISTS (SELECT 1 FROM approval_requests n WHERE n.previous_id = r.id AND n.status <> 'bypassed' AND r.status IN ('invalidated') AND n.requested_at > r.requested_at
                  AND @hide_replaced::boolean)
ORDER BY CASE r.status WHEN 'pending' THEN 0 ELSE 1 END, r.requested_at DESC
LIMIT 200;

-- name: ApprovalPendingForWorkspace :many
SELECT r.id, r.requested_at, coalesce((r.snapshot ->> 'audience_count')::bigint, 0)::bigint AS audience_count,
       coalesce(c.name, a.name, '')::text AS subject_name,
       coalesce(u.name, '')::text AS requested_by_name,
       coalesce(u.email, '')::text AS requested_by_email,
       coalesce(u.avatar_color, '')::text AS requested_by_color
FROM approval_requests r
LEFT JOIN users u ON u.id = r.requested_by
LEFT JOIN campaigns c ON c.id = r.campaign_id
LEFT JOIN automations a ON a.id = r.automation_id
WHERE r.workspace_id = @workspace_id AND r.status = 'pending'
ORDER BY r.requested_at DESC;

-- name: ApprovalDecisionWaits :many
-- How long requests waited for their first decision (the median wait).
SELECT EXTRACT(EPOCH FROM (min(d.created_at) - r.requested_at))::bigint AS seconds
FROM approval_requests r
JOIN approval_decisions d ON d.request_id = r.id
WHERE r.workspace_id = @workspace_id AND r.requested_at >= @since::timestamptz
GROUP BY r.id, r.requested_at;

-- name: ApprovalBypassCount :one
SELECT count(*)::bigint FROM approval_requests
WHERE workspace_id = @workspace_id AND status = 'bypassed' AND requested_at >= @since::timestamptz;

-- name: ApprovalMembers :many
-- Active members with their role and user details (approvers, presence).
SELECT u.id, u.name, u.email, u.avatar_color, (u.password_hash IS NOT NULL)::boolean AS has_password,
       r.key AS role_key, r.name AS role_name, r.builtin, r.permissions, m.last_active_at
FROM workspace_members m
JOIN roles r ON r.id = m.role_id
JOIN users u ON u.id = m.user_id AND u.status = 'active'
WHERE m.workspace_id = @workspace_id AND m.status = 'active'
ORDER BY CASE r.key WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END, u.name;

-- name: ApprovalMember :one
SELECT u.id, u.name, u.email, u.avatar_color, r.key AS role_key, r.name AS role_name, r.builtin, r.permissions
FROM workspace_members m
JOIN roles r ON r.id = m.role_id
JOIN users u ON u.id = m.user_id
WHERE m.workspace_id = @workspace_id AND m.user_id = @user_id;

-- name: ApprovalSetCampaignStatus :exec
UPDATE campaigns SET status = @status, updated_at = @now
WHERE id = @id AND status = ANY(@from_statuses::text[]);

-- name: ApprovalScheduleCampaign :exec
UPDATE campaigns SET schedule = @schedule, scheduled_at = sqlc.narg(scheduled_at), updated_by = sqlc.narg(updated_by), updated_at = @now
WHERE workspace_id = @workspace_id AND id = @id;

-- name: ApprovalMarkScheduled :exec
UPDATE campaigns SET status = 'scheduled', scheduled_at = @scheduled_at, updated_at = @now
WHERE id = @id AND status IN ('draft', 'approval', 'approved', 'scheduled');

-- name: ApprovalMarkSending :exec
UPDATE campaigns SET status = 'sending', started_by = @started_by, started_at = coalesce(started_at, @now), updated_at = @now
WHERE id = @id AND status IN ('draft', 'approval', 'approved', 'scheduled');

-- name: ApprovalUnschedule :exec
UPDATE campaigns SET status = @status, scheduled_at = NULL, updated_by = sqlc.narg(updated_by), updated_at = @now
WHERE workspace_id = @workspace_id AND id = @id AND status = 'scheduled';

-- name: ApprovalOpenHold :one
SELECT * FROM campaign_incidents WHERE campaign_id = @campaign_id AND kind = 'recheck_blocked' AND resolved_at IS NULL
ORDER BY tripped_at DESC LIMIT 1;

-- name: ApprovalInsertHold :exec
INSERT INTO campaign_incidents (id, campaign_id, kind, scope, held, detail, tripped_at)
VALUES (@id, @campaign_id, 'recheck_blocked', 'campaign', @held, @detail, @tripped_at);

-- name: ApprovalResolveHold :exec
UPDATE campaign_incidents SET resolved_at = @resolved_at, resolution = @resolution, resolved_by = sqlc.narg(resolved_by)
WHERE campaign_id = @campaign_id AND kind = 'recheck_blocked' AND resolved_at IS NULL;

-- name: ApprovalHeldCampaigns :many
-- Scheduled campaigns held by a Blocked recheck (retried until a recheck passes).
SELECT c.id, c.workspace_id, c.scheduled_at, i.tripped_at
FROM campaign_incidents i
JOIN campaigns c ON c.id = i.campaign_id AND c.deleted_at IS NULL AND c.status = 'scheduled'
WHERE i.kind = 'recheck_blocked' AND i.resolved_at IS NULL;

-- name: ApprovalLastAudit :one
-- The latest audit entry of an action on a target (who scheduled, who approved, from where).
SELECT occurred_at, actor_name, ip, metadata FROM audit_events
WHERE workspace_id = @workspace_id AND action = @action AND target_id = @target_id
ORDER BY occurred_at DESC LIMIT 1;

-- name: ApprovalCampaignVersion :one
SELECT version, subject, preheader, mjml, html, saved_at, coalesce(u.name, '')::text AS saved_by_name
FROM campaign_versions v
LEFT JOIN users u ON u.id = v.saved_by
WHERE v.campaign_id = @campaign_id AND v.version = @version;

-- name: ApprovalAudienceLists :many
SELECT id, name, requires_approval FROM lists WHERE workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: ApprovalPauseAlertsSlack :one
-- Whether an enabled webhook posts campaign.paused to Slack (Review & send says who hears about a held send).
SELECT EXISTS (
    SELECT 1 FROM webhooks
    WHERE workspace_id = @workspace_id AND enabled AND 'campaign.paused' = ANY(events) AND url LIKE 'https://hooks.slack.com/%'
)::boolean;
