-- Recurring and RSS digests (internal/digests): the digests, their feed's
-- items, their issues and the campaigns they build.

-- name: DigestInsert :exec
INSERT INTO digests (id, workspace_id, name, template_id, state, mode, source, feed_url, site_id, categories, tags, max_age_days,
                     start_mode, start_latest, schedule, prepare_hours, late_hours, archive, created_by, created_at, updated_by, updated_at)
VALUES (@id, @workspace_id, @name, sqlc.narg(template_id), 'off', @mode, @source, @feed_url, sqlc.narg(site_id), @categories, @tags,
        sqlc.narg(max_age_days), @start_mode, @start_latest, @schedule, @prepare_hours, @late_hours, @archive, sqlc.narg(created_by), @now::timestamptz, sqlc.narg(created_by), @now::timestamptz);

-- name: DigestGet :one
SELECT sqlc.embed(d),
       coalesce(o.name, '')::text AS owner_name, coalesce(o.avatar_color, '')::text AS owner_color,
       coalesce(pb.name, '')::text AS paused_by_name,
       coalesce(cb.name, '')::text AS created_by_name
FROM digests d
LEFT JOIN users o ON o.id = d.owner_id
LEFT JOIN users pb ON pb.id = d.paused_by
LEFT JOIN users cb ON cb.id = d.created_by
WHERE d.workspace_id = @workspace_id AND d.id = @id AND d.deleted_at IS NULL;

-- name: DigestByID :one
-- A digest of any workspace (jobs).
SELECT * FROM digests WHERE id = @id AND deleted_at IS NULL;

-- name: DigestLock :one
SELECT * FROM digests WHERE id = @id AND deleted_at IS NULL FOR UPDATE;

-- name: DigestList :many
SELECT d.*,
       coalesce(o.name, '')::text AS owner_name, coalesce(o.avatar_color, '')::text AS owner_color
FROM digests d
LEFT JOIN users o ON o.id = d.owner_id
WHERE d.workspace_id = @workspace_id AND d.deleted_at IS NULL
ORDER BY d.created_at, d.id;

-- name: DigestCount :one
SELECT count(*)::bigint FROM digests WHERE workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: DigestNameTaken :one
SELECT EXISTS (SELECT 1 FROM digests WHERE workspace_id = @workspace_id AND lower(name) = lower(@name) AND id <> @id AND deleted_at IS NULL);

-- name: DigestsActive :many
-- Digests the minute job looks after: on, or paused with an issue still
-- sending or waiting to be settled.
SELECT d.* FROM digests d
JOIN workspaces w ON w.id = d.workspace_id AND w.deleted_at IS NULL
WHERE d.deleted_at IS NULL
  AND (d.state = 'on' OR EXISTS (SELECT 1 FROM digest_issues i WHERE i.digest_id = d.id AND i.status IN ('preparing', 'prepared', 'held', 'draft', 'sending')))
ORDER BY d.id;

-- name: DigestsBySite :many
SELECT id, workspace_id, state FROM digests
WHERE site_id = @site_id AND deleted_at IS NULL;

-- name: DigestUpdateSettings :exec
UPDATE digests SET name = @name, mode = @mode, source = @source, feed_url = @feed_url, site_id = sqlc.narg(site_id), categories = @categories,
       tags = @tags, max_age_days = sqlc.narg(max_age_days), start_mode = @start_mode, start_latest = @start_latest, schedule = @schedule,
       prepare_hours = @prepare_hours, late_hours = @late_hours, archive = @archive, updated_by = sqlc.narg(updated_by), updated_at = @now::timestamptz
WHERE id = @id;

-- name: DigestSetTemplate :exec
UPDATE digests SET template_id = @template_id WHERE id = @id;

-- name: DigestTurnOn :exec
UPDATE digests SET state = 'on', mode = @mode, owner_id = @owner_id, owner_since = coalesce(CASE WHEN owner_id = @owner_id THEN owner_since END, @now::timestamptz),
       first_on_at = coalesce(first_on_at, @now::timestamptz), paused_at = NULL, paused_by = NULL, pause_reason = '', pause_note = '', pause_detail = '{}',
       feed_skips = 0, updated_by = @owner_id, updated_at = @now::timestamptz
WHERE id = @id;

-- name: DigestPause :exec
UPDATE digests SET state = 'paused', paused_at = @now::timestamptz, paused_by = sqlc.narg(paused_by), pause_reason = @pause_reason, pause_note = @pause_note,
       pause_detail = @pause_detail, updated_at = @now::timestamptz
WHERE id = @id;

-- name: DigestSoftDelete :exec
UPDATE digests SET deleted_at = @now::timestamptz, state = 'off', updated_by = sqlc.narg(updated_by), updated_at = @now::timestamptz WHERE id = @id;

-- name: DigestSetFeed :exec
-- What the last read found.
UPDATE digests SET feed_status = @feed_status, feed_kind = @feed_kind, feed_title = @feed_title, feed_items = @feed_items, feed_bytes = @feed_bytes,
       feed_etag = @feed_etag, feed_modified = @feed_modified, feed_error = @feed_error, feed_snippet = @feed_snippet, feed_moved_to = @feed_moved_to,
       last_read_at = @at, last_read_via = @via, last_ok_at = CASE WHEN @feed_status::text = 'ok' THEN @at ELSE last_ok_at END
WHERE id = @id;

-- name: DigestResetFeed :exec
-- The feed address changed: the next read starts afresh.
UPDATE digests SET feed_status = 'unread', feed_kind = '', feed_title = '', feed_items = 0, feed_bytes = 0, feed_etag = '', feed_modified = '',
       feed_error = '', feed_snippet = '', feed_moved_to = '', last_read_at = NULL, last_read_via = '', last_ok_at = NULL
WHERE id = @id;

-- name: DigestSetFeedSkips :exec
UPDATE digests SET feed_skips = @feed_skips WHERE id = @id;

-- name: DigestFeedReadInsert :exec
INSERT INTO digest_feed_reads (digest_id, at, via, ok, http_status, not_modified, kind, items, new_items, bytes, ms, error, hops)
VALUES (@digest_id, @at, @via, @ok, @http_status, @not_modified, @kind, @items, @new_items, @bytes, @ms, @error, @hops);

-- name: DigestFeedReadsPrune :exec
DELETE FROM digest_feed_reads WHERE at < @before;

-- name: DigestLastFeedRead :one
SELECT * FROM digest_feed_reads WHERE digest_id = @digest_id ORDER BY at DESC, id DESC LIMIT 1;

-- ---- Items ----

-- name: DigestItemUpsert :one
-- An item as the feed gives it now. Its first sighting is kept; a post the
-- feed edits keeps its id and gets the new text.
INSERT INTO digest_items (id, digest_id, key, url, title, author, published_at, categories, tags, excerpt, image_url, image_alt,
                          first_seen_at, first_seen_via, last_seen_at)
VALUES (@id, @digest_id, @key, @url, @title, @author, sqlc.narg(published_at), @categories, @tags, @excerpt, @image_url, @image_alt,
        @now::timestamptz, @via, @now::timestamptz)
ON CONFLICT (digest_id, key) DO UPDATE SET
    url = EXCLUDED.url, title = EXCLUDED.title, author = EXCLUDED.author,
    published_at = coalesce(EXCLUDED.published_at, digest_items.published_at),
    categories = EXCLUDED.categories, tags = EXCLUDED.tags, excerpt = EXCLUDED.excerpt,
    image_url = EXCLUDED.image_url, image_alt = EXCLUDED.image_alt,
    -- A new image is copied again.
    media_id = CASE WHEN digest_items.image_url = EXCLUDED.image_url THEN digest_items.media_id END,
    media_url = CASE WHEN digest_items.image_url = EXCLUDED.image_url THEN digest_items.media_url ELSE '' END,
    image_error = CASE WHEN digest_items.image_url = EXCLUDED.image_url THEN digest_items.image_error ELSE '' END,
    last_seen_at = EXCLUDED.last_seen_at, gone_at = NULL
RETURNING id, (xmax = 0) AS inserted;

-- name: DigestItemsMarkGone :exec
-- Items the feed no longer lists.
UPDATE digest_items SET gone_at = @now::timestamptz
WHERE digest_id = @digest_id AND gone_at IS NULL AND NOT (key = ANY(@keys::text[]));

-- name: DigestItems :many
-- The digest's items, newest first, with the sent issue each went in.
SELECT it.*,
       coalesce(sent.number, 0)::integer AS sent_in
FROM digest_items it
LEFT JOIN LATERAL (
    SELECT i.number FROM digest_issue_items ii JOIN digest_issues i ON i.id = ii.issue_id
    WHERE ii.item_id = it.id AND i.status IN ('sending', 'sent') AND i.number IS NOT NULL
    ORDER BY i.number LIMIT 1
) sent ON true
WHERE it.digest_id = @digest_id AND (it.gone_at IS NULL OR sent.number IS NOT NULL OR @with_gone::boolean)
ORDER BY it.published_at DESC NULLS LAST, it.first_seen_at DESC, it.id DESC
LIMIT @lim;

-- name: DigestItemsNeedingImage :many
SELECT id, image_url FROM digest_items
WHERE digest_id = @digest_id AND gone_at IS NULL AND image_url <> '' AND media_id IS NULL AND image_error = ''
ORDER BY published_at DESC NULLS LAST, first_seen_at DESC
LIMIT @lim;

-- name: DigestItemSetMedia :exec
UPDATE digest_items SET media_id = sqlc.narg(media_id), media_url = @media_url, media_width = @media_width, media_height = @media_height, image_error = @image_error
WHERE id = @id;

-- name: DigestItemsByID :many
SELECT * FROM digest_items WHERE id = ANY(@ids::uuid[]);

-- ---- Issues ----

-- name: DigestIssueInsert :exec
INSERT INTO digest_issues (id, digest_id, workspace_id, status, mode, send_at, prepare_at, deadline, created_at, updated_at)
VALUES (@id, @digest_id, @workspace_id, 'collecting', @mode, sqlc.narg(send_at), sqlc.narg(prepare_at), sqlc.narg(deadline), @now::timestamptz, @now::timestamptz);

-- name: DigestIssueOpen :one
-- The issue being collected, prepared or waiting.
SELECT * FROM digest_issues
WHERE digest_id = @digest_id AND status IN ('collecting', 'preparing', 'prepared', 'held', 'draft')
LIMIT 1;

-- name: DigestIssueGet :one
SELECT * FROM digest_issues WHERE id = @id AND digest_id = @digest_id;

-- name: DigestIssueLock :one
SELECT * FROM digest_issues WHERE id = @id FOR UPDATE;

-- name: DigestIssueByID :one
-- An issue for its job: the job is unique while it waits, so no lock.
SELECT * FROM digest_issues WHERE id = @id;

-- name: DigestIssueByCampaign :one
SELECT i.*, d.name AS digest_name, d.workspace_id AS digest_workspace_id
FROM digest_issues i JOIN digests d ON d.id = i.digest_id AND d.deleted_at IS NULL
WHERE i.campaign_id = @campaign_id
ORDER BY i.created_at DESC
LIMIT 1;

-- name: DigestIssues :many
-- A digest's issues, the newest send time first.
SELECT i.*,
       coalesce(c.status, '')::text AS campaign_status,
       c.started_at AS campaign_started_at, c.completed_at AS campaign_completed_at, c.audience_count AS campaign_audience,
       coalesce(sb.name, '')::text AS skipped_by_name
FROM digest_issues i
LEFT JOIN campaigns c ON c.id = i.campaign_id
LEFT JOIN users sb ON sb.id = i.skipped_by
WHERE i.digest_id = @digest_id
ORDER BY coalesce(i.send_at, i.prepared_at, i.created_at) DESC, i.created_at DESC
LIMIT @lim;

-- name: DigestIssuesInFlight :many
-- Issues the minute job moves along: being collected (their preparation
-- may be due), retried, prepared, held, in draft or sending.
SELECT i.* FROM digest_issues i
JOIN digests d ON d.id = i.digest_id AND d.deleted_at IS NULL
WHERE i.digest_id = @digest_id AND i.status IN ('collecting', 'preparing', 'prepared', 'held', 'draft', 'sending')
ORDER BY i.created_at;

-- name: DigestIssueSetTimes :exec
UPDATE digest_issues SET send_at = sqlc.narg(send_at), prepare_at = sqlc.narg(prepare_at), deadline = sqlc.narg(deadline), mode = @mode, updated_at = @now::timestamptz
WHERE id = @id;

-- name: DigestIssueSetCampaign :exec
UPDATE digest_issues SET campaign_id = sqlc.narg(campaign_id), updated_at = @now::timestamptz WHERE id = @id;

-- name: DigestIssuePreparing :exec
-- The feed failed at preparation: retried every 15 minutes until the send time.
UPDATE digest_issues SET status = 'preparing', feed_tries = @feed_tries, updated_at = @now::timestamptz WHERE id = @id AND status IN ('collecting', 'preparing');

-- name: DigestIssuePrepared :exec
UPDATE digest_issues SET status = @status, prepared_at = @now::timestamptz, held_at = sqlc.narg(held_at), hold_reason = @hold_reason, item_count = @item_count,
       send_at = sqlc.narg(send_at), deadline = sqlc.narg(deadline), campaign_id = sqlc.narg(campaign_id), updated_at = @now::timestamptz
WHERE id = @id AND status IN ('collecting', 'preparing', 'prepared', 'held', 'draft');

-- name: DigestIssueHold :exec
UPDATE digest_issues SET status = 'held', held_at = @now::timestamptz, hold_reason = @hold_reason, updated_at = @now::timestamptz WHERE id = @id;

-- name: DigestIssueRelease :exec
UPDATE digest_issues SET status = 'prepared', held_at = NULL, hold_reason = '', updated_at = @now::timestamptz WHERE id = @id;

-- name: DigestIssueSending :exec
UPDATE digest_issues SET status = 'sending', number = @number, started_at = coalesce(started_at, @now::timestamptz), updated_at = @now::timestamptz WHERE id = @id;

-- name: DigestIssueSent :exec
UPDATE digest_issues SET status = 'sent', sent_at = coalesce(sent_at, @now::timestamptz), updated_at = @now::timestamptz WHERE id = @id;

-- name: DigestIssueSkip :exec
UPDATE digest_issues SET status = @status, number = NULL, skipped_at = @now::timestamptz, skip_reason = @skip_reason, skip_note = @skip_note,
       skipped_by = sqlc.narg(skipped_by), updated_at = @now::timestamptz
WHERE id = @id;

-- name: DigestIssueNotified :exec
UPDATE digest_issues SET notified = array_append(notified, @kind::text) WHERE id = @id AND NOT (@kind::text = ANY(notified));

-- name: DigestIssueDelete :exec
DELETE FROM digest_issues WHERE id = @id AND status = 'collecting';

-- name: DigestNextNumber :one
SELECT (coalesce(max(number), 0) + 1)::integer FROM digest_issues WHERE digest_id = @digest_id AND number IS NOT NULL;

-- name: DigestLastSentIssue :one
SELECT * FROM digest_issues WHERE digest_id = @digest_id AND status IN ('sending', 'sent') AND number IS NOT NULL
ORDER BY number DESC LIMIT 1;

-- name: DigestRecentFeedSkips :many
-- The latest settled issues, to count feed skips in a row.
SELECT status, skip_reason FROM digest_issues
WHERE digest_id = @digest_id AND status IN ('sent', 'sending', 'skipped', 'not_sent')
ORDER BY coalesce(send_at, created_at) DESC
LIMIT 3;

-- name: DigestIssueItemsSet :exec
INSERT INTO digest_issue_items (issue_id, item_id, position)
SELECT @issue_id, unnest(@item_ids::uuid[]), generate_subscripts(@item_ids::uuid[], 1)
ON CONFLICT (issue_id, item_id) DO UPDATE SET position = EXCLUDED.position;

-- name: DigestIssueItemsClear :exec
DELETE FROM digest_issue_items WHERE issue_id = @issue_id;

-- name: DigestIssueItems :many
SELECT it.* FROM digest_issue_items ii JOIN digest_items it ON it.id = ii.item_id
WHERE ii.issue_id = @issue_id
ORDER BY ii.position;

-- name: DigestIssueItemCounts :many
SELECT issue_id, count(*)::integer AS n FROM digest_issue_items WHERE issue_id = ANY(@issue_ids::uuid[]) GROUP BY issue_id;

-- name: DigestUsedItems :many
-- Items already sent in an issue of the digest: they aren't new any more.
SELECT DISTINCT ii.item_id FROM digest_issue_items ii JOIN digest_issues i ON i.id = ii.issue_id
WHERE i.digest_id = @digest_id AND i.status IN ('sending', 'sent');

-- name: DigestIssueResults :many
-- Sent and delivered counts, human opens and clicks of issues' campaigns.
SELECT cc.campaign_id,
       coalesce(sum(cc.sent), 0)::bigint AS sent,
       coalesce(sum(cc.delivered), 0)::bigint AS delivered,
       coalesce(sum(cc.opened_human), 0)::bigint AS opened_human,
       coalesce(sum(cc.clicked), 0)::bigint AS clicked
FROM campaign_counters cc
WHERE cc.campaign_id = ANY(@campaign_ids::uuid[])
GROUP BY cc.campaign_id;

-- name: DigestIssueApprovals :many
-- The latest approval request of each issue's campaign, with who decided.
SELECT DISTINCT ON (r.campaign_id) r.campaign_id, r.id, r.status, r.requested_at, r.decided_at, r.matched_rules,
       coalesce(u.name, '')::text AS decided_by_name, coalesce(u.avatar_color, '')::text AS decided_by_color
FROM approval_requests r
LEFT JOIN users u ON u.id = r.decided_by
WHERE r.campaign_id = ANY(@campaign_ids::uuid[])
ORDER BY r.campaign_id, r.requested_at DESC;

-- ---- The campaigns a digest builds ----

-- name: DigestCampaignInsert :exec
-- A template, a next issue or an issue: content, audience, sender, route
-- and thresholds, as a draft.
INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, sender_id, reply_to, audience, audience_count, editor_mode, mjml, html, plain_text,
                       version, schedule, thresholds, route_override, tracking, content_hash, content_settings, digest_id, digest_role,
                       created_by, updated_by, created_at, updated_at, autosaved_at)
VALUES (@id, @workspace_id, @name, 'draft', @subject, @preheader, sqlc.narg(sender_id), @reply_to, @audience, sqlc.narg(audience_count), 'visual', @mjml, @html, @plain_text,
        1, @schedule, @thresholds, sqlc.narg(route_override), @tracking, @content_hash, @content_settings, @digest_id, @digest_role,
        sqlc.narg(created_by), sqlc.narg(created_by), @now::timestamptz, @now::timestamptz, @now::timestamptz);

-- name: DigestCampaignVersionInsert :exec
INSERT INTO campaign_versions (campaign_id, version, subject, preheader, mjml, html, plain_text, saved_by, saved_at, label, source, size_bytes)
VALUES (@campaign_id, @version, @subject, @preheader, @mjml, @html, @plain_text, sqlc.narg(saved_by), @saved_at, @label, @source, @size_bytes);

-- name: DigestCampaignRebuild :one
-- An issue built again: its content and send settings from the template,
-- as a new version.
UPDATE campaigns SET name = @name, subject = @subject, preheader = @preheader, sender_id = sqlc.narg(sender_id), reply_to = @reply_to, audience = @audience,
       mjml = @mjml, html = @html, plain_text = @plain_text, thresholds = @thresholds, route_override = sqlc.narg(route_override), tracking = @tracking,
       content_hash = @content_hash, content_settings = @content_settings, schedule = @schedule, digest_role = @digest_role,
       version = version + 1, updated_by = sqlc.narg(updated_by), updated_at = @now::timestamptz, autosaved_at = @now::timestamptz
WHERE id = @id AND deleted_at IS NULL
RETURNING version;

-- name: DigestCampaignTemplate :one
-- The template the next issue is built from.
SELECT c.*, coalesce(s.name, '')::text AS sender_name, coalesce(s.email, '')::text AS sender_email
FROM campaigns c LEFT JOIN senders s ON s.id = c.sender_id
WHERE c.id = @id AND c.deleted_at IS NULL;

-- name: DigestCampaignSchedule :exec
-- A prepared issue keeps its send time; the digest starts it then.
UPDATE campaigns SET status = 'scheduled', scheduled_at = @at::timestamptz, schedule = @schedule, updated_at = @now::timestamptz
WHERE id = @id AND status IN ('draft', 'approval', 'approved', 'scheduled');

-- name: DigestCampaignUnschedule :exec
-- A held issue waits for a person: it comes off the schedule.
UPDATE campaigns SET status = @status, scheduled_at = NULL, updated_at = @now::timestamptz WHERE id = @id AND status = 'scheduled';

-- name: DigestCampaignRemove :exec
-- A skipped or unsent issue's campaign leaves the campaigns list; its
-- history stays on the digest.
UPDATE campaigns SET deleted_at = @now::timestamptz, scheduled_at = NULL, status = CASE WHEN status = 'scheduled' THEN 'draft' ELSE status END
WHERE id = @id AND status IN ('draft', 'approval', 'approved', 'scheduled') AND deleted_at IS NULL;

-- name: DigestCampaignSetRole :exec
UPDATE campaigns SET digest_role = @digest_role, name = @name WHERE id = @id;

-- name: DigestCampaignState :one
SELECT id, status, name, scheduled_at, started_at, completed_at, version, content_hash, audience_count, deleted_at, digest_role
FROM campaigns WHERE id = @id;

-- name: DigestCampaignCancelApproval :exec
-- The late limit passed or the issue was skipped: its open request goes.
UPDATE approval_requests SET status = 'cancelled', decided_at = @now::timestamptz, invalidated_reason = @reason
WHERE campaign_id = @campaign_id AND status IN ('pending', 'approved');

-- name: DigestOpenApproval :one
SELECT * FROM approval_requests WHERE campaign_id = @campaign_id AND status IN ('pending', 'approved') ORDER BY requested_at DESC LIMIT 1;

-- name: DigestHeldIncident :one
-- A scheduled issue the send-time recheck held.
SELECT * FROM campaign_incidents WHERE campaign_id = @campaign_id AND kind = 'recheck_blocked' AND resolved_at IS NULL LIMIT 1;

-- name: DigestResolveHolds :exec
UPDATE campaign_incidents SET resolved_at = @now::timestamptz, resolution = 'cancelled'
WHERE campaign_id = @campaign_id AND kind = 'recheck_blocked' AND resolved_at IS NULL;

-- ---- Members ----

-- name: DigestMember :one
-- A member of the workspace with their role's permissions (the owner check).
SELECT u.id, u.name, u.email, u.avatar_color, u.status AS user_status, m.status AS member_status, r.key AS role_key, r.name AS role_name,
       r.builtin, r.permissions
FROM workspace_members m
JOIN users u ON u.id = m.user_id
JOIN roles r ON r.id = m.role_id
WHERE m.workspace_id = @workspace_id AND m.user_id = @user_id;

-- name: DigestSites :many
-- The workspace's connected WordPress sites, as feed sources.
SELECT i.id, i.label, i.connected_at, coalesce(u.name, '')::text AS connected_by_name,
       s.address, s.site_url, s.name, s.plugin_version, s.feed_url, s.categories, s.last_report_at, s.rule_enabled, s.rule
FROM integrations i
JOIN wordpress_sites s ON s.integration_id = i.id
LEFT JOIN users u ON u.id = i.connected_by
WHERE i.workspace_id = @workspace_id AND i.kind = 'wordpress' AND i.disconnected_at IS NULL
ORDER BY i.connected_at, i.id;

-- name: DigestAuditFor :many
-- The digest's own audit entries, for its history.
SELECT occurred_at, actor_type, actor_name, action, before, after, reason, metadata
FROM audit_events
WHERE workspace_id = @workspace_id AND target_type = 'digest' AND target_id = @target_id
ORDER BY occurred_at DESC
LIMIT @lim;

-- name: DigestTemplateSettings :exec
-- The settings page's audience, sender and limits live on the template.
UPDATE campaigns SET name = @name, audience = @audience, sender_id = @sender_id, thresholds = @thresholds, updated_at = @now::timestamptz
WHERE id = @id AND digest_role = 'template' AND deleted_at IS NULL;

-- name: DigestTemplateRemove :exec
UPDATE campaigns SET deleted_at = @now::timestamptz WHERE id = @id AND digest_role IN ('template', 'next') AND deleted_at IS NULL;

-- name: DigestSegmentExists :one
SELECT EXISTS (SELECT 1 FROM segments WHERE id = @id AND workspace_id = @workspace_id);

-- name: DigestPreviousRole :one
-- The role a member had before their last role change (team page or SSO).
SELECT coalesce(e.before ->> 'role', '')::text AS role
FROM audit_events e
WHERE e.workspace_id = @workspace_id AND e.action = 'member.role_changed'
  AND (e.target_id = @user_id::text OR e.metadata ->> 'user_id' = @user_id::text
       OR e.target_id IN (SELECT m.id::text FROM workspace_members m WHERE m.user_id::text = @user_id::text AND m.workspace_id = e.workspace_id))
ORDER BY e.occurred_at DESC LIMIT 1;

-- name: DigestFirstSent :one
-- When issue #1 went out.
SELECT sent_at, started_at, send_at FROM digest_issues WHERE digest_id = @digest_id AND number = 1;

-- name: DigestSiteFeedItems :one
-- The items in a connected site's feed, as its digests last read it.
SELECT coalesce(max(feed_items), 0)::int AS items FROM digests WHERE site_id = @site_id AND deleted_at IS NULL AND feed_status = 'ok';

-- name: DigestIssueSetBasis :exec
UPDATE digest_issues SET basis = @basis WHERE id = @id;

-- name: DigestIssuesByCampaigns :many
-- Digest issues of campaigns (the approvals inbox, Review & send, the report).
SELECT i.*, d.name AS digest_name, d.mode AS digest_mode, d.schedule AS digest_schedule, d.owner_id AS digest_owner_id,
       coalesce(o.name, '')::text AS owner_name, coalesce(o.avatar_color, '')::text AS owner_color
FROM digest_issues i
JOIN digests d ON d.id = i.digest_id AND d.deleted_at IS NULL
LEFT JOIN users o ON o.id = d.owner_id
WHERE i.workspace_id = @workspace_id AND i.campaign_id = ANY(@campaign_ids::uuid[]);

-- name: DigestLastApprovedIssue :one
-- The newest sent issue before this one, with when its approval was given.
SELECT i.id, i.number, i.basis, i.campaign_id, i.started_at,
       (SELECT ar.decided_at FROM approval_requests ar WHERE ar.campaign_id = i.campaign_id AND ar.status IN ('approved', 'sent') ORDER BY ar.decided_at DESC NULLS LAST LIMIT 1) AS approved_at
FROM digest_issues i
WHERE i.digest_id = @digest_id AND i.status IN ('sending', 'sent') AND i.id <> @id AND i.number IS NOT NULL
ORDER BY i.number DESC LIMIT 1;
