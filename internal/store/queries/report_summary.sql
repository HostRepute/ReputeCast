-- Sending summary by email (M57 G-14): members who can view reports and
-- asked for a weekly or monthly summary of a workspace (internal/reports,
-- summary.go).

-- name: ReportSummaryGet :one
SELECT frequency, last_period_start FROM report_summaries WHERE user_id = @user_id AND workspace_id = @workspace_id;

-- name: ReportSummarySet :exec
-- Turns the summary on or changes how often it comes. last_period_start is
-- the last period already behind the member, so the first email is the next
-- one, not a summary of the week they turned it on in.
INSERT INTO report_summaries (user_id, workspace_id, frequency, last_period_start)
VALUES (@user_id, @workspace_id, @frequency, @last_period_start)
ON CONFLICT (user_id, workspace_id) DO UPDATE SET frequency = excluded.frequency, last_period_start = excluded.last_period_start;

-- name: ReportSummaryRemove :execrows
DELETE FROM report_summaries WHERE user_id = @user_id AND workspace_id = @workspace_id;

-- name: ReportSummariesList :many
-- Every summary asked for in a workspace that isn't deleted, with the zones
-- its periods are counted in: the member's, else the workspace's.
SELECT r.user_id, r.workspace_id, r.frequency, r.last_period_start,
       u.timezone AS user_timezone, w.timezone AS workspace_timezone
FROM report_summaries r
JOIN workspaces w ON w.id = r.workspace_id AND w.deleted_at IS NULL
JOIN users u ON u.id = r.user_id
ORDER BY r.workspace_id, r.user_id;

-- name: ReportSummaryClaim :one
-- Claims a period for sending: only one tick gets the row back, however
-- many run at once.
UPDATE report_summaries SET last_period_start = @from_date::date
WHERE user_id = @user_id AND workspace_id = @workspace_id AND frequency = @frequency
  AND (last_period_start IS NULL OR last_period_start < @from_date::date)
RETURNING user_id;

-- name: ReportSummaryMember :one
-- Whether the person is still an active member of the workspace.
SELECT EXISTS (
    SELECT 1 FROM workspace_members m JOIN users u ON u.id = m.user_id AND u.status = 'active'
    WHERE m.workspace_id = @workspace_id AND m.user_id = @user_id AND m.status = 'active'
)::boolean AS active;

-- name: ReportSummaryCampaigns :many
-- The email campaigns started in a period (not history imported from
-- another platform), with their counters per mailbox provider; a campaign
-- without counters yet has one row with an empty provider.
SELECT c.id, c.name, c.started_at, c.completed_at,
       coalesce(cc.mailbox_provider, '')::text AS mailbox_provider,
       coalesce(sum(cc.sent), 0)::bigint AS sent,
       coalesce(sum(cc.delivered), 0)::bigint AS delivered,
       coalesce(sum(cc.opened_human), 0)::bigint AS opened_human,
       coalesce(sum(cc.clicked), 0)::bigint AS clicked,
       coalesce(sum(cc.unsubscribed), 0)::bigint AS unsubscribed,
       coalesce(sum(cc.hard_bounced + cc.soft_bounced), 0)::bigint AS bounced,
       coalesce(sum(cc.complained), 0)::bigint AS complained
FROM campaigns c
LEFT JOIN campaign_counters cc ON cc.campaign_id = c.id
WHERE c.workspace_id = @workspace_id AND c.deleted_at IS NULL AND c.channel = 'email' AND c.origin = ''
  AND c.started_at >= @period_from::timestamptz AND c.started_at < @period_to::timestamptz
GROUP BY c.id, cc.mailbox_provider
ORDER BY c.started_at, c.id, mailbox_provider;

-- name: ReportSummaryAudience :one
-- The newsletter's subscribers in a period, counted as the Overview's
-- growth counts them (OverviewListChanges) but between two moments: joins
-- of members still subscribed and departures, plus the subscribers now.
-- The lists a connected store fills aren't the newsletter's (X20).
SELECT
    (SELECT count(*) FROM list_memberships lm
     JOIN lists l ON l.id = lm.list_id AND l.workspace_id = @workspace_id AND l.deleted_at IS NULL AND l.origin <> 'store'
     WHERE lm.status = 'subscribed' AND lm.subscribed_at >= @period_from::timestamptz AND lm.subscribed_at < @period_to::timestamptz)::bigint AS joined,
    (SELECT count(*) FROM list_memberships lm
     JOIN lists l ON l.id = lm.list_id AND l.workspace_id = @workspace_id AND l.deleted_at IS NULL AND l.origin <> 'store'
     WHERE lm.unsubscribed_at >= @period_from::timestamptz AND lm.unsubscribed_at < @period_to::timestamptz)::bigint AS departed,
    (SELECT count(*) FROM list_memberships lm
     JOIN lists l ON l.id = lm.list_id AND l.workspace_id = @workspace_id AND l.deleted_at IS NULL AND l.origin <> 'store'
     WHERE lm.status = 'subscribed')::bigint AS subscribers;
