-- Recommendations and partner links (internal/recommendations): the
-- newsletters a workspace recommends, its partner profile, partner codes,
-- partner links in both directions and the partner request logs.

-- name: RecSettingsGet :one
SELECT * FROM recommendation_settings WHERE workspace_id = @workspace_id;

-- name: RecSettingsUpsert :one
INSERT INTO recommendation_settings (workspace_id, confirm_page, profile_name, profile_description, profile_logo_id, profile_list_id, updated_by, updated_at)
VALUES (@workspace_id, @confirm_page, @profile_name, @profile_description, sqlc.narg(profile_logo_id), sqlc.narg(profile_list_id), sqlc.narg(updated_by), @now::timestamptz)
ON CONFLICT (workspace_id) DO UPDATE SET confirm_page = excluded.confirm_page, profile_name = excluded.profile_name,
    profile_description = excluded.profile_description, profile_logo_id = excluded.profile_logo_id, profile_list_id = excluded.profile_list_id,
    updated_by = excluded.updated_by, updated_at = excluded.updated_at
RETURNING *;

-- name: RecInsert :exec
INSERT INTO recommendations (id, workspace_id, kind, name, description, logo_id, url, partner_link_id, show_confirm, show_block, position, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @kind, @name, @description, sqlc.narg(logo_id), @url, sqlc.narg(partner_link_id), @show_confirm, @show_block,
        coalesce((SELECT max(position) + 1 FROM recommendations WHERE workspace_id = @workspace_id AND removed_at IS NULL), 0),
        sqlc.narg(created_by), @now::timestamptz, @now::timestamptz);

-- name: RecGet :one
SELECT * FROM recommendations WHERE workspace_id = @workspace_id AND id = @id AND removed_at IS NULL;

-- name: RecByID :one
SELECT * FROM recommendations WHERE id = @id;

-- name: RecList :many
SELECT * FROM recommendations WHERE workspace_id = @workspace_id AND removed_at IS NULL ORDER BY position, created_at;

-- name: RecUpdate :exec
UPDATE recommendations SET name = @name, description = @description, logo_id = sqlc.narg(logo_id), url = @url,
       show_confirm = @show_confirm, show_block = @show_block, updated_at = @now::timestamptz
WHERE id = @id;

-- name: RecSetStatus :exec
UPDATE recommendations SET status = @status, paused_at = CASE WHEN @status = 'paused' THEN @now::timestamptz END,
       paused_by = CASE WHEN @status = 'paused' THEN sqlc.narg(by)::uuid END, updated_at = @now::timestamptz
WHERE id = @id;

-- name: RecRemove :exec
UPDATE recommendations SET removed_at = @now::timestamptz, updated_at = @now::timestamptz WHERE id = @id;

-- name: RecSetPosition :exec
UPDATE recommendations SET position = @position, updated_at = @now::timestamptz WHERE id = @id AND workspace_id = @workspace_id;

-- name: RecOfLink :one
SELECT * FROM recommendations WHERE partner_link_id = @link_id AND removed_at IS NULL;

-- name: RecDailyBump :exec
INSERT INTO recommendation_daily (recommendation_id, day, shown_confirm, shown_email, clicks, requests, confirmed)
VALUES (@recommendation_id, @day, @shown_confirm, @shown_email, @clicks, @requests, @confirmed)
ON CONFLICT (recommendation_id, day) DO UPDATE SET shown_confirm = recommendation_daily.shown_confirm + excluded.shown_confirm,
    shown_email = recommendation_daily.shown_email + excluded.shown_email, clicks = recommendation_daily.clicks + excluded.clicks,
    requests = recommendation_daily.requests + excluded.requests, confirmed = recommendation_daily.confirmed + excluded.confirmed;

-- name: RecTotals :many
SELECT recommendation_id, sum(shown_confirm)::bigint AS shown_confirm, sum(shown_email)::bigint AS shown_email, sum(clicks)::bigint AS clicks,
       sum(requests)::bigint AS requests, sum(confirmed)::bigint AS confirmed
FROM recommendation_daily
WHERE recommendation_id = ANY (@ids::uuid[]) AND day >= @since::date
GROUP BY recommendation_id;

-- name: PartnerCodeInsert :exec
INSERT INTO partner_codes (id, workspace_id, code_hash, label, list_id, daily_limit, created_by, created_at, expires_at)
VALUES (@id, @workspace_id, @code_hash, @label, @list_id, @daily_limit, sqlc.narg(created_by), @now::timestamptz, @expires_at::timestamptz);

-- name: PartnerCodeByHash :one
SELECT * FROM partner_codes WHERE code_hash = @code_hash;

-- name: PartnerCodeLock :one
SELECT * FROM partner_codes WHERE id = @id FOR UPDATE;

-- name: PartnerCodeUse :exec
UPDATE partner_codes SET used_at = @now::timestamptz, used_link_id = @link_id WHERE id = @id;

-- name: PartnerCodeRevoke :exec
UPDATE partner_codes SET revoked_at = @now::timestamptz, revoked_by = sqlc.narg(by) WHERE id = @id AND used_at IS NULL AND revoked_at IS NULL;

-- name: PartnerCodesOpen :many
-- Codes not used, revoked or expired yet ("Waiting for …").
SELECT c.*, l.name AS list_name, coalesce(u.name, '')::text AS created_by_name
FROM partner_codes c JOIN lists l ON l.id = c.list_id LEFT JOIN users u ON u.id = c.created_by
WHERE c.workspace_id = @workspace_id AND c.used_at IS NULL AND c.revoked_at IS NULL AND c.expires_at > @now::timestamptz
ORDER BY c.created_at DESC;

-- name: PartnerLinkInsert :exec
INSERT INTO partner_links (id, workspace_id, direction, remote_url, remote_link_id, protocol_version, remote_name, remote_description, remote_logo_url,
                           remote_list_name, daily_limit, list_id, label, code_id, code_hash, secret, connected_at, connected_by, created_at, updated_at)
VALUES (@id, @workspace_id, @direction, @remote_url, @remote_link_id, @protocol_version, @remote_name, @remote_description, @remote_logo_url,
        @remote_list_name, @daily_limit, sqlc.narg(list_id), @label, sqlc.narg(code_id), sqlc.narg(code_hash), @secret, @now::timestamptz,
        sqlc.narg(connected_by), @now::timestamptz, @now::timestamptz);

-- name: PartnerLinkGet :one
SELECT * FROM partner_links WHERE workspace_id = @workspace_id AND id = @id;

-- name: PartnerLinkByID :one
SELECT * FROM partner_links WHERE id = @id;

-- name: PartnerLinkLock :one
SELECT * FROM partner_links WHERE id = @id FOR UPDATE;

-- name: PartnerLinks :many
SELECT pl.*, coalesce(u.name, '')::text AS connected_by_name, coalesce(pu.name, '')::text AS paused_by_name, coalesce(l.name, '')::text AS list_name
FROM partner_links pl
LEFT JOIN users u ON u.id = pl.connected_by
LEFT JOIN users pu ON pu.id = pl.paused_by
LEFT JOIN lists l ON l.id = pl.list_id
WHERE pl.workspace_id = @workspace_id AND pl.direction = @direction AND pl.status <> 'ended'
ORDER BY pl.connected_at;

-- name: PartnerLinkByCodeHash :one
SELECT pl.*, coalesce(u.name, '')::text AS connected_by_name
FROM partner_links pl LEFT JOIN users u ON u.id = pl.connected_by
WHERE pl.workspace_id = @workspace_id AND pl.code_hash = @code_hash
ORDER BY pl.connected_at DESC LIMIT 1;

-- name: PartnerLinkSetStatus :exec
UPDATE partner_links SET status = @status, paused_side = @paused_side, paused_at = sqlc.narg(paused_at), paused_by = sqlc.narg(paused_by),
       updated_at = @now::timestamptz
WHERE id = @id;

-- name: PartnerLinkEnd :exec
UPDATE partner_links SET status = 'ended', ended_at = @now::timestamptz, ended_by = sqlc.narg(by), ended_side = @side, updated_at = @now::timestamptz
WHERE id = @id;

-- name: PartnerLinkInvalid :exec
UPDATE partner_links SET status = 'invalid', invalid_since = coalesce(invalid_since, @now::timestamptz), invalid_reason = @reason, updated_at = @now::timestamptz
WHERE id = @id AND status <> 'ended';

-- name: PartnerLinkLimit :exec
UPDATE partner_links SET limit_until = sqlc.narg(until), updated_at = @now::timestamptz WHERE id = @id;

-- name: PartnerLinkFailing :exec
UPDATE partner_links SET failing_since = CASE WHEN @failing::boolean THEN coalesce(failing_since, @now::timestamptz) END,
       notified_at = CASE WHEN @failing::boolean THEN notified_at END, updated_at = @now::timestamptz
WHERE id = @id;

-- name: PartnerLinkNotified :exec
UPDATE partner_links SET notified_at = @now::timestamptz WHERE id = @id;

-- name: PartnerLinkSetLimit :exec
UPDATE partner_links SET daily_limit = @daily_limit, updated_at = @now::timestamptz WHERE id = @id;

-- name: PartnerLinkSetProfile :exec
UPDATE partner_links SET remote_name = @remote_name, remote_description = @remote_description, remote_logo_url = @remote_logo_url,
       remote_list_name = @remote_list_name, daily_limit = @daily_limit, updated_at = @now::timestamptz
WHERE id = @id;

-- name: PartnerLinkRotate :exec
UPDATE partner_links SET prev_secret = @prev_secret, prev_secret_until = @until::timestamptz, secret = @secret, secret_rotated_at = @now::timestamptz,
       last_rotation = @rotation, updated_at = @now::timestamptz
WHERE id = @id;

-- name: PartnerRequestOutInsert :exec
INSERT INTO partner_requests_out (id, workspace_id, link_id, recommendation_id, email, origin, status, next_attempt_at, created_at)
VALUES (@id, @workspace_id, @link_id, sqlc.narg(recommendation_id), @email, @origin, 'queued', @now::timestamptz, @now::timestamptz);

-- name: PartnerRequestOutGet :one
SELECT * FROM partner_requests_out WHERE id = @id;

-- name: PartnerRequestsOutDue :many
SELECT * FROM partner_requests_out WHERE status = 'queued' AND next_attempt_at <= @now::timestamptz
ORDER BY next_attempt_at LIMIT @lim FOR UPDATE SKIP LOCKED;

-- name: PartnerRequestOutResult :exec
UPDATE partner_requests_out SET status = @status, attempts = attempts + 1, next_attempt_at = sqlc.narg(next_attempt_at), last_error = @last_error,
       sent_at = CASE WHEN @status IN ('accepted', 'limit', 'rejected') THEN @now::timestamptz ELSE sent_at END,
       email = CASE WHEN @status = 'queued' THEN email END
WHERE id = @id;

-- name: PartnerRequestOutConfirmed :one
UPDATE partner_requests_out SET confirmed_at = @now::timestamptz
WHERE id = @id AND link_id = @link_id AND confirmed_at IS NULL AND status = 'accepted'
RETURNING recommendation_id;

-- name: PartnerRequestInInsert :one
INSERT INTO partner_requests_in (id, workspace_id, link_id, request_id, status, contact_id, asked, received_at)
VALUES (@id, @workspace_id, @link_id, @request_id, @status, sqlc.narg(contact_id), @asked, @now::timestamptz)
ON CONFLICT (link_id, request_id) DO NOTHING
RETURNING id;

-- name: PartnerRequestsInToday :one
SELECT count(*) FROM partner_requests_in WHERE link_id = @link_id AND status = 'accepted' AND received_at >= @since::timestamptz;

-- name: PartnerRequestsInPending :many
-- Accepted requests of a contact waiting for their confirmation.
SELECT ri.*, pl.list_id FROM partner_requests_in ri JOIN partner_links pl ON pl.id = ri.link_id
WHERE ri.contact_id = @contact_id AND ri.confirmed_at IS NULL AND ri.status = 'accepted' AND ri.asked;

-- name: PartnerRequestInConfirmed :exec
UPDATE partner_requests_in SET confirmed_at = @now::timestamptz, notice_status = 'queued', notice_next_at = @now::timestamptz WHERE id = @id;

-- name: PartnerNoticesDue :many
SELECT * FROM partner_requests_in WHERE notice_status = 'queued' AND notice_next_at <= @now::timestamptz
ORDER BY notice_next_at LIMIT @lim FOR UPDATE SKIP LOCKED;

-- name: PartnerNoticeResult :exec
UPDATE partner_requests_in SET notice_status = @notice_status, notice_attempts = notice_attempts + 1, notice_next_at = sqlc.narg(notice_next_at),
       notice_sent_at = CASE WHEN @notice_status = 'sent' THEN @now::timestamptz END
WHERE id = @id;

-- name: PartnerRequestsPrune :exec
-- The partner request logs keep 90 days.
WITH a AS (DELETE FROM partner_requests_out WHERE created_at < @before::timestamptz)
DELETE FROM partner_requests_in WHERE received_at < @before::timestamptz;

-- name: PartnerLinkNames :many
-- The partners that recommend a workspace, for its growth sources
-- ("Partner · Ops Weekly").
SELECT id, remote_name FROM partner_links WHERE workspace_id = @workspace_id AND direction = 'in';
