-- Referral programs (internal/referrals): programs, milestones and rewards,
-- links, referrals and their log, visits, codes, files and the leaderboard.

-- name: ReferralProgramInsert :exec
INSERT INTO referral_programs (id, workspace_id, list_id, status, setup_step, setup_saved_at, setup_saved_by, invite_headline, invite_text,
                               hub_intro, share_message, sender_id, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @list_id, 'off', 2, @now::timestamptz, sqlc.narg(created_by), @invite_headline, @invite_text, @hub_intro,
        @share_message, sqlc.narg(sender_id), sqlc.narg(created_by), @now::timestamptz, @now::timestamptz);

-- name: ReferralProgramGet :one
SELECT sqlc.embed(p),
       l.name AS list_name,
       coalesce(fo.name, '')::text AS first_on_name,
       coalesce(sb.name, '')::text AS status_by_name,
       coalesce(lb.name, '')::text AS leaderboard_by_name,
       coalesce(ss.name, '')::text AS setup_saved_name
FROM referral_programs p
JOIN lists l ON l.id = p.list_id
LEFT JOIN users fo ON fo.id = p.first_on_by
LEFT JOIN users sb ON sb.id = p.status_by
LEFT JOIN users lb ON lb.id = p.leaderboard_by
LEFT JOIN users ss ON ss.id = p.setup_saved_by
WHERE p.workspace_id = @workspace_id AND p.id = @id AND p.deleted_at IS NULL;

-- name: ReferralProgramByID :one
-- A program of any workspace (jobs, public pages).
SELECT * FROM referral_programs WHERE id = @id AND deleted_at IS NULL;

-- name: ReferralProgramLock :one
SELECT * FROM referral_programs WHERE id = @id AND deleted_at IS NULL FOR UPDATE;

-- name: ReferralProgramsOf :many
SELECT p.*, l.name AS list_name
FROM referral_programs p JOIN lists l ON l.id = p.list_id
WHERE p.workspace_id = @workspace_id AND p.deleted_at IS NULL
ORDER BY p.created_at, p.id;

-- name: ReferralProgramForList :one
SELECT * FROM referral_programs WHERE list_id = @list_id AND deleted_at IS NULL;

-- name: ReferralProgramsActive :many
-- Programs the minute job looks after: every program not deleted (counting
-- and rewards go on while a program is off).
SELECT p.* FROM referral_programs p
JOIN workspaces w ON w.id = p.workspace_id AND w.deleted_at IS NULL
WHERE p.deleted_at IS NULL
ORDER BY p.id;

-- name: ReferralProgramSetStatus :exec
UPDATE referral_programs SET status = @status, status_at = @now::timestamptz, status_by = sqlc.narg(by),
       first_on_at = coalesce(first_on_at, CASE WHEN @status = 'on' THEN @now::timestamptz END),
       first_on_by = CASE WHEN first_on_at IS NULL AND @status = 'on' THEN sqlc.narg(by) ELSE first_on_by END,
       setup_step = CASE WHEN @status = 'on' THEN 0 ELSE setup_step END,
       updated_at = @now::timestamptz
WHERE id = @id;

-- name: ReferralProgramSetList :exec
UPDATE referral_programs SET list_id = @list_id, updated_at = @now::timestamptz WHERE id = @id AND first_on_at IS NULL;

-- name: ReferralProgramSetStep :exec
UPDATE referral_programs SET setup_step = @setup_step, setup_saved_at = @now::timestamptz, setup_saved_by = sqlc.narg(by), updated_at = @now::timestamptz
WHERE id = @id AND first_on_at IS NULL;

-- name: ReferralProgramUpdate :exec
UPDATE referral_programs SET hold_days = @hold_days, rule_same_ip = @rule_same_ip, rule_disposable = @rule_disposable, rule_left = @rule_left,
       burst_on = @burst_on, burst_threshold = @burst_threshold, shared_ip_on = @shared_ip_on, shared_ip_threshold = @shared_ip_threshold,
       numbered_on = @numbered_on, invite_headline = @invite_headline, invite_text = @invite_text, hub_intro = @hub_intro,
       share_message = @share_message, share_buttons = @share_buttons, leaderboard_periods = @leaderboard_periods,
       leaderboard_rows = @leaderboard_rows, sender_id = sqlc.narg(sender_id), reply_to = @reply_to, updated_at = @now::timestamptz
WHERE id = @id;

-- name: ReferralProgramSetLeaderboard :exec
UPDATE referral_programs SET leaderboard_on = @leaderboard_on, leaderboard_at = @now::timestamptz, leaderboard_by = sqlc.narg(by), updated_at = @now::timestamptz
WHERE id = @id;

-- name: ReferralProgramDelete :exec
UPDATE referral_programs SET deleted_at = @now::timestamptz, deleted_by = sqlc.narg(by), updated_at = @now::timestamptz
WHERE id = @id AND status = 'off' AND deleted_at IS NULL;

-- name: ReferralProgramReviewNotified :exec
UPDATE referral_programs SET review_notified_at = sqlc.narg(at) WHERE id = @id;

-- name: ReferralListSubscribers :one
SELECT count(*)::bigint FROM list_memberships lm JOIN contacts c ON c.id = lm.contact_id
WHERE lm.list_id = @list_id AND lm.status = 'subscribed' AND c.erased_at IS NULL;

-- name: ReferralListsOffered :many
-- Lists a program can use, with their subscribers and whether a program
-- already has them.
SELECT l.id, l.name, l.description,
       (SELECT count(*) FROM list_memberships lm JOIN contacts c ON c.id = lm.contact_id
        WHERE lm.list_id = l.id AND lm.status = 'subscribed' AND c.erased_at IS NULL)::bigint AS subscribers,
       EXISTS (SELECT 1 FROM referral_programs p WHERE p.list_id = l.id AND p.deleted_at IS NULL) AS taken
FROM lists l
WHERE l.workspace_id = @workspace_id AND l.deleted_at IS NULL
ORDER BY subscribers DESC, l.name;

-- Milestones --------------------------------------------------------------

-- name: ReferralMilestoneInsert :exec
INSERT INTO referral_milestones (id, workspace_id, program_id, count, reward_name, reward_description, image_id, reward_type, file_id,
                                 download_url, codes_alert_below, ask_address, coupon_offer_id, coupon_offer_name, coupon_valid_days,
                                 webhook_url, webhook_secret, email_on, email_subject, email_preheader, email_mjml, email_html,
                                 created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @program_id, @count, @reward_name, @reward_description, sqlc.narg(image_id), @reward_type, sqlc.narg(file_id),
        @download_url, @codes_alert_below, @ask_address, @coupon_offer_id, @coupon_offer_name, @coupon_valid_days,
        @webhook_url, sqlc.narg(webhook_secret), @email_on, @email_subject, @email_preheader, @email_mjml, @email_html,
        sqlc.narg(created_by), @now::timestamptz, @now::timestamptz);

-- name: ReferralMilestones :many
SELECT * FROM referral_milestones WHERE program_id = @program_id AND deleted_at IS NULL ORDER BY count, id;

-- name: ReferralMilestoneGet :one
SELECT * FROM referral_milestones WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL;

-- name: ReferralMilestoneByID :one
SELECT * FROM referral_milestones WHERE id = @id;

-- name: ReferralMilestoneUpdate :exec
UPDATE referral_milestones SET count = @count, reward_name = @reward_name, reward_description = @reward_description, image_id = sqlc.narg(image_id),
       reward_type = @reward_type, file_id = sqlc.narg(file_id), download_url = @download_url, codes_alert_below = @codes_alert_below,
       ask_address = @ask_address, coupon_offer_id = @coupon_offer_id, coupon_offer_name = @coupon_offer_name,
       coupon_valid_days = @coupon_valid_days, webhook_url = @webhook_url, email_on = @email_on, updated_at = @now::timestamptz
WHERE id = @id;

-- name: ReferralMilestoneSetSecret :exec
UPDATE referral_milestones SET webhook_secret = sqlc.narg(webhook_secret), updated_at = @now::timestamptz WHERE id = @id;

-- name: ReferralMilestoneDelete :exec
UPDATE referral_milestones SET deleted_at = @now::timestamptz, updated_at = @now::timestamptz WHERE id = @id AND deleted_at IS NULL;

-- name: ReferralMilestoneSaveEmail :one
-- Autosave of the milestone email's draft, refused when someone saved since
-- base_revision.
UPDATE referral_milestones SET email_subject = @email_subject, email_preheader = @email_preheader, email_mjml = @email_mjml,
       email_html = @email_html, email_revision = email_revision + 1, email_saved_at = @now::timestamptz, email_saved_by = sqlc.narg(by),
       updated_at = @now::timestamptz
WHERE id = @id AND email_revision = @base_revision
RETURNING email_revision;

-- name: ReferralMilestoneEmailOn :exec
UPDATE referral_milestones SET email_on = @email_on, updated_at = @now::timestamptz WHERE id = @id;

-- name: ReferralMilestoneSetEmail :exec
UPDATE referral_milestones SET email_id = @email_id, updated_at = @now::timestamptz WHERE id = @id;

-- name: ReferralEmailCreate :exec
-- A milestone email is an automation email outside any flow.
INSERT INTO automation_emails (id, workspace_id, automation_id, name, latest_version, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, NULL, @name, 0, sqlc.narg(created_by), @now::timestamptz, @now::timestamptz);

-- name: ReferralEmailRename :exec
UPDATE automation_emails SET name = @name, updated_at = @now::timestamptz WHERE id = @id;

-- name: ReferralEmailVersionInsert :one
-- Saving the milestone email adds a version; sends use the latest one.
WITH v AS (
    UPDATE automation_emails SET latest_version = latest_version + 1, updated_at = @now::timestamptz
    WHERE id = @email_id RETURNING latest_version
)
INSERT INTO automation_email_versions (email_id, version, subject, preheader, mjml, html, plain_text, created_by, created_at)
SELECT @email_id, v.latest_version, @subject, @preheader, @mjml, @html, @plain_text, sqlc.narg(created_by), @now::timestamptz FROM v
RETURNING version;

-- name: ReferralEmailVersions :many
SELECT v.version, v.subject, v.created_at, coalesce(u.name, '')::text AS created_by_name
FROM automation_email_versions v LEFT JOIN users u ON u.id = v.created_by
WHERE v.email_id = @email_id ORDER BY v.version DESC;

-- name: ReferralEmailVersion :one
SELECT * FROM automation_email_versions WHERE email_id = @email_id AND version = @version;

-- name: ReferralEmailLatest :one
SELECT v.* FROM automation_email_versions v JOIN automation_emails e ON e.id = v.email_id AND v.version = e.latest_version
WHERE e.id = @email_id;

-- name: ReferralMilestoneWebhookResult :exec
UPDATE referral_milestones SET webhook_status = sqlc.narg(status), webhook_called_at = @now::timestamptz,
       webhook_failures = CASE WHEN @ok::boolean THEN 0 ELSE webhook_failures + 1 END
WHERE id = @id;

-- name: ReferralMilestoneWebhookAlerted :exec
UPDATE referral_milestones SET webhook_alerted_at = sqlc.narg(at) WHERE id = @id;

-- name: ReferralMilestoneCodesAlerted :exec
UPDATE referral_milestones SET codes_alerted = @codes_alerted WHERE id = @id;

-- name: ReferralMilestoneStats :many
-- Per milestone: reached (rewards given), sent, used, waiting and failed.
SELECT r.milestone_id,
       count(*)::bigint AS reached,
       count(*) FILTER (WHERE r.status IN ('sent', 'fulfilled', 'expired') OR r.sent_at IS NOT NULL)::bigint AS sent,
       count(*) FILTER (WHERE r.used_at IS NOT NULL)::bigint AS used,
       count(*) FILTER (WHERE r.status IN ('waiting_codes', 'pending', 'to_fulfill'))::bigint AS waiting,
       count(*) FILTER (WHERE r.status = 'failed')::bigint AS failed,
       count(*) FILTER (WHERE r.status = 'waiting_codes')::bigint AS waiting_codes,
       count(*) FILTER (WHERE r.status = 'to_fulfill')::bigint AS to_fulfill,
       count(*) FILTER (WHERE r.status = 'pending')::bigint AS pending
FROM referral_rewards r
WHERE r.program_id = @program_id AND r.milestone_id IS NOT NULL
GROUP BY r.milestone_id;

-- name: ReferralMilestoneReachedSince :one
-- How many referrers reached a milestone since a time (the pace of a code
-- reward).
SELECT count(*)::bigint FROM referral_rewards WHERE milestone_id = @milestone_id AND reached_at >= @since::timestamptz;

-- Files ---------------------------------------------------------------------

-- name: ReferralFileInsert :exec
INSERT INTO referral_files (id, workspace_id, program_id, file_name, content_type, size_bytes, sha256, uploaded_by, uploaded_at)
VALUES (@id, @workspace_id, @program_id, @file_name, @content_type, @size_bytes, @sha256, sqlc.narg(uploaded_by), @now::timestamptz);

-- name: ReferralFileChunkInsert :exec
INSERT INTO referral_file_chunks (file_id, seq, data) VALUES (@file_id, @seq, @data);

-- name: ReferralFiles :many
SELECT f.id, f.workspace_id, f.program_id, f.file_name, f.content_type, f.size_bytes, f.downloads, f.uploaded_at,
       coalesce(u.name, '')::text AS uploaded_by_name
FROM referral_files f LEFT JOIN users u ON u.id = f.uploaded_by
WHERE f.program_id = @program_id AND f.removed_at IS NULL
ORDER BY f.uploaded_at, f.id;

-- name: ReferralFileGet :one
SELECT f.id, f.workspace_id, f.program_id, f.file_name, f.content_type, f.size_bytes, f.downloads, f.uploaded_at, f.removed_at,
       coalesce(u.name, '')::text AS uploaded_by_name
FROM referral_files f LEFT JOIN users u ON u.id = f.uploaded_by
WHERE f.id = @id;

-- name: ReferralFileChunks :many
SELECT seq, data FROM referral_file_chunks WHERE file_id = @file_id AND seq >= @from_seq ORDER BY seq LIMIT @lim;

-- name: ReferralFileCounted :exec
UPDATE referral_files SET downloads = downloads + 1 WHERE id = @id;

-- name: ReferralFileRemove :exec
UPDATE referral_files SET removed_at = @now::timestamptz WHERE id = @id AND removed_at IS NULL;

-- Codes ---------------------------------------------------------------------

-- name: ReferralCodeInsert :one
INSERT INTO referral_reward_codes (id, workspace_id, milestone_id, code, code_hash, batch_id, uploaded_by, uploaded_at)
VALUES (@id, @workspace_id, @milestone_id, @code, @code_hash, @batch_id, sqlc.narg(uploaded_by), @now::timestamptz)
ON CONFLICT (milestone_id, code_hash) DO NOTHING
RETURNING id;

-- name: ReferralCodesLeft :one
SELECT count(*)::bigint FROM referral_reward_codes WHERE milestone_id = @milestone_id AND reward_id IS NULL AND removed_at IS NULL;

-- name: ReferralCodesUploaded :one
SELECT count(*)::bigint FROM referral_reward_codes WHERE milestone_id = @milestone_id AND removed_at IS NULL;

-- name: ReferralCodeTake :one
-- The oldest free code of a milestone, taken by a reward.
UPDATE referral_reward_codes SET reward_id = @reward_id, assigned_at = @now::timestamptz
WHERE id = (SELECT c.id FROM referral_reward_codes c WHERE c.milestone_id = @milestone_id AND c.reward_id IS NULL AND c.removed_at IS NULL
            ORDER BY c.uploaded_at, c.id LIMIT 1 FOR UPDATE SKIP LOCKED)
RETURNING id, code;

-- name: ReferralCodesRemoveFree :execrows
UPDATE referral_reward_codes SET removed_at = @now::timestamptz
WHERE milestone_id = @milestone_id AND reward_id IS NULL AND removed_at IS NULL;

-- name: ReferralCodeOf :one
SELECT code FROM referral_reward_codes WHERE id = @id;

-- Links ---------------------------------------------------------------------

-- name: ReferralLinkInsert :one
INSERT INTO referral_links (id, workspace_id, program_id, contact_id, code, created_at)
VALUES (@id, @workspace_id, @program_id, @contact_id, @code, @now::timestamptz)
ON CONFLICT DO NOTHING
RETURNING id;

-- name: ReferralLinksMissing :many
-- Confirmed subscribers of the program's list without a link yet.
SELECT lm.contact_id FROM list_memberships lm
JOIN contacts c ON c.id = lm.contact_id AND c.erased_at IS NULL
WHERE lm.list_id = @list_id AND lm.status = 'subscribed'
  AND NOT EXISTS (SELECT 1 FROM referral_links rl WHERE rl.program_id = @program_id AND rl.contact_id = lm.contact_id)
ORDER BY lm.contact_id
LIMIT @lim;

-- name: ReferralLinkCount :one
SELECT count(*)::bigint FROM referral_links WHERE program_id = @program_id;

-- name: ReferralLinkByCode :one
SELECT rl.*, c.email AS contact_email, c.first_name AS contact_first_name, c.last_name AS contact_last_name
FROM referral_links rl JOIN contacts c ON c.id = rl.contact_id
WHERE rl.workspace_id = @workspace_id AND rl.code = @code;

-- name: ReferralLinkGet :one
SELECT rl.*, c.email AS contact_email, c.first_name AS contact_first_name, c.last_name AS contact_last_name,
       coalesce(pb.name, '')::text AS paused_by_name
FROM referral_links rl JOIN contacts c ON c.id = rl.contact_id
LEFT JOIN users pb ON pb.id = rl.paused_by
WHERE rl.workspace_id = @workspace_id AND rl.id = @id;

-- name: ReferralLinkOfContact :one
SELECT rl.*, c.email AS contact_email, c.first_name AS contact_first_name, c.last_name AS contact_last_name
FROM referral_links rl JOIN contacts c ON c.id = rl.contact_id
WHERE rl.program_id = @program_id AND rl.contact_id = @contact_id;

-- name: ReferralLinksOfContact :many
SELECT rl.*, p.list_id, p.status AS program_status
FROM referral_links rl JOIN referral_programs p ON p.id = rl.program_id AND p.deleted_at IS NULL
WHERE rl.contact_id = @contact_id
ORDER BY rl.created_at;

-- name: ReferralLinkSetStatus :exec
UPDATE referral_links SET status = @status, paused_at = CASE WHEN @status = 'paused' THEN @now::timestamptz END,
       paused_by = CASE WHEN @status = 'paused' THEN sqlc.narg(by) END,
       pause_note = CASE WHEN @status = 'paused' THEN @note ELSE '' END
WHERE id = @id;

-- name: ReferralLinkSetChoices :exec
UPDATE referral_links SET show_name = @show_name, show_leaderboard = @show_leaderboard, choices_at = @now::timestamptz WHERE id = @id;

-- name: ReferralLinkBump :exec
UPDATE referral_links SET visits = visits + @visits, signups = signups + @signups WHERE id = @id;

-- name: ReferralLinkSeenIP :exec
INSERT INTO referral_referrer_ips (link_id, ip, seen_at) VALUES (@link_id, @ip, @now::timestamptz)
ON CONFLICT (link_id, ip) DO UPDATE SET seen_at = excluded.seen_at;

-- name: ReferralReferrerIPMatch :one
-- Whether ip is an address the referrer used in the last 30 days: to open
-- their hub or a page of their link, or to subscribe or confirm.
SELECT (EXISTS (SELECT 1 FROM referral_referrer_ips ri WHERE ri.link_id = @link_id AND ri.ip = @addr::inet AND ri.seen_at >= @since::timestamptz)
    OR EXISTS (SELECT 1 FROM consent_events ce WHERE ce.contact_id = @referrer_id AND ce.ip = @addr::inet AND ce.occurred_at >= @since::timestamptz
               AND ce.action IN ('subscribed', 'confirmed', 'resubscribed')))::boolean AS matched;

-- name: ReferralReferrerIPsPrune :exec
DELETE FROM referral_referrer_ips WHERE seen_at < @before::timestamptz;

-- Referrals -----------------------------------------------------------------

-- name: ReferralInsert :exec
INSERT INTO referrals (id, workspace_id, program_id, link_id, referrer_id, contact_id, seq, status, reason, via, ip, ip_network,
                       signed_up_at, not_counted_at, created_at, updated_at)
VALUES (@id, @workspace_id, @program_id, @link_id, @referrer_id, @contact_id,
        (SELECT coalesce(max(r.seq), 0) + 1 FROM referrals r WHERE r.link_id = @link_id),
        @status, @reason, @via, sqlc.narg(ip), @ip_network, @now::timestamptz, sqlc.narg(not_counted_at), @now::timestamptz, @now::timestamptz);

-- name: ReferralOfContact :one
SELECT * FROM referrals WHERE program_id = @program_id AND contact_id = @contact_id;

-- name: ReferralAwaitingOfContact :many
-- A contact's referrals waiting for their confirmation, with the program's
-- rules.
SELECT r.*, p.list_id, p.hold_days, p.rule_same_ip, p.rule_disposable, p.burst_on, p.burst_threshold, p.shared_ip_on,
       p.shared_ip_threshold, p.numbered_on
FROM referrals r JOIN referral_programs p ON p.id = r.program_id
WHERE r.contact_id = @contact_id AND r.status = 'awaiting'
FOR UPDATE OF r;

-- name: ReferralGet :one
SELECT * FROM referrals WHERE workspace_id = @workspace_id AND id = @id;

-- name: ReferralSetConfirmed :exec
UPDATE referrals SET confirmed_at = @confirmed_at::timestamptz, status = @status, reason = @reason, signals = @signals,
       hold_until = sqlc.narg(hold_until), held_at = CASE WHEN @status = 'review' THEN @confirmed_at::timestamptz END,
       not_counted_at = CASE WHEN @status = 'not_counted' THEN @confirmed_at::timestamptz END,
       counted_at = CASE WHEN @status = 'counted' THEN @confirmed_at::timestamptz END,
       updated_at = @confirmed_at::timestamptz
WHERE id = @id;

-- name: ReferralHoldToReview :many
-- Referrals of a link still in hold that a suspicion rule now covers:
-- confirmed since a time, or matching ids.
UPDATE referrals SET status = 'review', held_at = @now::timestamptz,
       signals = (SELECT array_agg(DISTINCT s) FROM unnest(signals || @signals::text[]) s),
       updated_at = @now::timestamptz
WHERE link_id = @link_id AND status = 'hold' AND (id = ANY (@ids::uuid[]))
RETURNING id;

-- name: ReferralAddSignals :exec
UPDATE referrals SET signals = (SELECT array_agg(DISTINCT s) FROM unnest(signals || @signals::text[]) s), updated_at = @now::timestamptz
WHERE id = ANY (@ids::uuid[]) AND status = 'review';

-- name: ReferralRecentConfirmed :many
-- A link's referrals confirmed since a time, still in hold or review (the
-- burst rule).
SELECT id, status, confirmed_at FROM referrals
WHERE link_id = @link_id AND confirmed_at >= @since::timestamptz AND status IN ('hold', 'review')
ORDER BY confirmed_at;

-- name: ReferralRecentFromIP :many
-- A link's referrals that signed up from an address since a time.
SELECT id, status FROM referrals
WHERE link_id = @link_id AND ip = @addr::inet AND signed_up_at >= @since::timestamptz AND status IN ('awaiting', 'hold', 'review')
ORDER BY signed_up_at;

-- name: ReferralNumberedPeers :many
-- A link's other referrals with their addresses (the numbered-address rule).
SELECT r.id, r.status, c.email FROM referrals r JOIN contacts c ON c.id = r.contact_id
WHERE r.link_id = @link_id AND r.id <> @id AND r.status IN ('awaiting', 'hold', 'review') AND r.signed_up_at >= @since::timestamptz;

-- name: ReferralsHoldDue :many
-- Referrals whose hold has ended, with what decides whether they count.
SELECT r.id, r.workspace_id, r.program_id, r.link_id, r.referrer_id, r.contact_id, p.rule_left, p.list_id,
       coalesce(lm.status, '')::text AS membership,
       EXISTS (SELECT 1 FROM suppressions s JOIN contacts c ON c.id = r.contact_id
               WHERE s.workspace_id = r.workspace_id AND s.email = c.email AND s.reason IN ('hard_bounce', 'complaint')) AS bounced,
       coalesce((SELECT s.reason FROM suppressions s JOIN contacts c ON c.id = r.contact_id
                 WHERE s.workspace_id = r.workspace_id AND s.email = c.email AND s.reason IN ('hard_bounce', 'complaint')
                 ORDER BY s.created_at LIMIT 1), '')::text AS bounce_reason
FROM referrals r
JOIN referral_programs p ON p.id = r.program_id
LEFT JOIN list_memberships lm ON lm.list_id = p.list_id AND lm.contact_id = r.contact_id
WHERE r.status = 'hold' AND r.hold_until <= @now::timestamptz
ORDER BY r.hold_until
LIMIT @lim
FOR UPDATE OF r SKIP LOCKED;

-- name: ReferralSetCounted :exec
UPDATE referrals SET status = 'counted', counted_at = @now::timestamptz, updated_at = @now::timestamptz WHERE id = @id AND status = 'hold';

-- name: ReferralSetNotCounted :exec
UPDATE referrals SET status = 'not_counted', reason = @reason, left_kind = @left_kind, not_counted_at = @now::timestamptz,
       updated_at = @now::timestamptz
WHERE id = @id;

-- name: ReferralsNeverConfirmed :many
-- Referrals still waiting for their confirmation after 7 days.
UPDATE referrals SET status = 'not_counted', reason = 'never_confirmed', not_counted_at = @now::timestamptz, updated_at = @now::timestamptz
WHERE status = 'awaiting' AND signed_up_at < @before::timestamptz
RETURNING id, workspace_id, program_id, link_id, referrer_id, contact_id;

-- name: ReferralCountedOf :one
-- A referrer's counted referrals in a program.
SELECT count(*)::bigint FROM referrals WHERE program_id = @program_id AND link_id = @link_id AND status = 'counted';

-- name: ReferralsReduceIPs :exec
-- After 90 days an address is reduced to its network.
UPDATE referrals SET ip = NULL, ip_reduced_at = @now::timestamptz WHERE ip IS NOT NULL AND signed_up_at < @before::timestamptz;

-- name: ReferralEventInsert :exec
INSERT INTO referral_events (id, workspace_id, program_id, referral_id, link_id, contact_id, kind, detail, occurred_at)
VALUES (@id, @workspace_id, @program_id, sqlc.narg(referral_id), sqlc.narg(link_id), sqlc.narg(contact_id), @kind, @detail, @occurred_at::timestamptz);

-- name: ReferralEventsOfContact :many
SELECT e.*, coalesce(m.reward_name, '')::text AS reward_name
FROM referral_events e
LEFT JOIN referral_milestones m ON m.id = (e.detail->>'milestone')::uuid
WHERE e.workspace_id = @workspace_id AND e.contact_id = @contact_id
ORDER BY e.occurred_at DESC
LIMIT 50;

-- Visits --------------------------------------------------------------------

-- name: ReferralVisitKey :one
-- Records the day's visit key; a row comes back only for the first visit.
INSERT INTO referral_visit_keys (key, seen_at) VALUES (@key, @now::timestamptz)
ON CONFLICT (key) DO NOTHING
RETURNING key;

-- name: ReferralVisitBump :exec
INSERT INTO referral_visits_daily (link_id, program_id, day, via, visits, signups)
VALUES (@link_id, @program_id, @day, @via, @visits, @signups)
ON CONFLICT (link_id, day, via) DO UPDATE SET visits = referral_visits_daily.visits + excluded.visits,
                                              signups = referral_visits_daily.signups + excluded.signups;

-- name: ReferralVisitKeysPrune :exec
DELETE FROM referral_visit_keys WHERE seen_at < @before::timestamptz;

-- Rewards -------------------------------------------------------------------

-- name: ReferralRewardInsert :one
INSERT INTO referral_rewards (id, workspace_id, program_id, milestone_id, contact_id, link_id, milestone_count, reward_name, reward_type,
                              status, reached_at, created_at, updated_at)
VALUES (@id, @workspace_id, @program_id, @milestone_id, @contact_id, @link_id, @milestone_count, @reward_name, @reward_type,
        @status, @now::timestamptz, @now::timestamptz, @now::timestamptz)
ON CONFLICT (milestone_id, contact_id) WHERE milestone_id IS NOT NULL DO NOTHING
RETURNING id;

-- name: ReferralRewardGet :one
SELECT * FROM referral_rewards WHERE workspace_id = @workspace_id AND id = @id;

-- name: ReferralRewardByID :one
SELECT * FROM referral_rewards WHERE id = @id;

-- name: ReferralRewardLock :one
SELECT * FROM referral_rewards WHERE id = @id FOR UPDATE;

-- name: ReferralRewardsOfContact :many
SELECT rw.*, coalesce(m.reward_description, '')::text AS reward_description, m.file_id, coalesce(m.download_url, '')::text AS download_url,
       coalesce(m.ask_address, false)::boolean AS ask_address, coalesce(f.file_name, '')::text AS file_name,
       coalesce(f.content_type, '')::text AS file_type, coalesce(f.size_bytes, 0)::bigint AS file_size, m.image_id
FROM referral_rewards rw
LEFT JOIN referral_milestones m ON m.id = rw.milestone_id
LEFT JOIN referral_files f ON f.id = m.file_id
WHERE rw.program_id = @program_id AND rw.contact_id = @contact_id
ORDER BY rw.milestone_count, rw.reached_at;

-- name: ReferralRewardSet :exec
UPDATE referral_rewards SET status = @status, sent_at = coalesce(sent_at, sqlc.narg(sent_at)), code_id = coalesce(sqlc.narg(code_id), code_id),
       claim_url = CASE WHEN @claim_url::text <> '' THEN @claim_url ELSE claim_url END, expires_at = coalesce(sqlc.narg(expires_at), expires_at),
       answer = coalesce(sqlc.narg(answer), answer), last_error = @last_error, attempts = @attempts,
       next_attempt_at = sqlc.narg(next_attempt_at), failed_at = sqlc.narg(failed_at), updated_at = @now::timestamptz
WHERE id = @id;

-- name: ReferralRewardEmailSent :exec
UPDATE referral_rewards SET email_sent_at = @now::timestamptz, email_message_id = sqlc.narg(message_id), updated_at = @now::timestamptz WHERE id = @id;

-- name: ReferralRewardResent :exec
UPDATE referral_rewards SET resent_at = @now::timestamptz, updated_at = @now::timestamptz WHERE id = @id;

-- name: ReferralRewardUsed :exec
UPDATE referral_rewards SET used_at = coalesce(used_at, @now::timestamptz), downloads = downloads + @downloads, updated_at = @now::timestamptz
WHERE id = @id;

-- name: ReferralRewardAddress :exec
UPDATE referral_rewards SET address = @address, address_at = @now::timestamptz, updated_at = @now::timestamptz
WHERE id = @id AND status = 'to_fulfill';

-- name: ReferralRewardFulfill :exec
UPDATE referral_rewards SET status = 'fulfilled', used_at = @now::timestamptz, fulfilled_by = sqlc.narg(by), fulfill_note = @note,
       address_purge_at = @purge_at::timestamptz, sent_at = coalesce(sent_at, @now::timestamptz), updated_at = @now::timestamptz
WHERE id = @id AND status = 'to_fulfill';

-- name: ReferralRewardsAddressPurge :exec
UPDATE referral_rewards SET address = NULL, address_purge_at = NULL WHERE address_purge_at IS NOT NULL AND address_purge_at < @now::timestamptz;

-- name: ReferralRewardsDue :many
-- Rewards to deliver: webhook calls due, codes that may have arrived,
-- coupons waiting for a payment provider.
SELECT * FROM referral_rewards
WHERE (status = 'pending' AND (next_attempt_at IS NULL OR next_attempt_at <= @now::timestamptz))
   OR status = 'waiting_codes'
ORDER BY reached_at
LIMIT @lim
FOR UPDATE SKIP LOCKED;

-- name: ReferralRewardsExpire :exec
UPDATE referral_rewards SET status = 'expired', updated_at = @now::timestamptz
WHERE reward_type = 'coupon' AND status = 'sent' AND used_at IS NULL AND expires_at IS NOT NULL AND expires_at < @now::timestamptz;

-- name: ReferralRewardsToFulfill :many
SELECT rw.id, rw.contact_id, rw.milestone_count, rw.reward_name, rw.reached_at, (rw.address IS NOT NULL)::boolean AS has_address, rw.address,
       rw.address_at, c.email, c.first_name, c.last_name
FROM referral_rewards rw JOIN contacts c ON c.id = rw.contact_id
WHERE rw.program_id = @program_id AND rw.status = 'to_fulfill'
ORDER BY rw.reached_at;

-- Leaderboard ---------------------------------------------------------------

-- name: ReferralLeaderboardClear :exec
DELETE FROM referral_leaderboard WHERE program_id = @program_id;

-- name: ReferralLeaderboardInsert :exec
INSERT INTO referral_leaderboard (program_id, period, link_id, rank, counted, computed_at)
VALUES (@program_id, @period, @link_id, @rank, @counted, @computed_at::timestamptz);

-- name: ReferralLeaderboardComputedAt :one
SELECT max(computed_at)::timestamptz FROM referral_leaderboard WHERE program_id = @program_id;

-- name: ReferralContactOnList :one
SELECT EXISTS (SELECT 1 FROM list_memberships WHERE list_id = @list_id AND contact_id = @contact_id AND status = 'subscribed');
