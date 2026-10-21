-- Web push (internal/push): VAPID keys, settings, sites, subscriptions,
-- push prompts, push campaigns and their sends, and the notifications of
-- the message log.

-- name: PushKeyLive :one
SELECT k.*, coalesce(u.name, '')::text AS created_by_name
FROM push_keys k LEFT JOIN users u ON u.id = k.created_by
WHERE k.workspace_id = @workspace_id AND k.rotated_at IS NULL;

-- name: PushKeyByID :one
SELECT * FROM push_keys WHERE id = @id;

-- name: PushKeyInsert :exec
INSERT INTO push_keys (id, workspace_id, public_key, private_enc, contact, created_by, created_at)
VALUES (@id, @workspace_id, @public_key, @private_enc, @contact, sqlc.narg(created_by), @created_at);

-- name: PushKeyRotate :exec
UPDATE push_keys SET rotated_at = @at, rotated_by = sqlc.narg(rotated_by) WHERE id = @id AND rotated_at IS NULL;

-- name: PushKeySetContact :exec
UPDATE push_keys SET contact = @contact WHERE id = @id;

-- name: PushKeyAccepted :exec
UPDATE push_keys SET last_accepted_at = greatest(coalesce(last_accepted_at, @at::timestamptz), @at::timestamptz) WHERE id = @id;

-- name: PushSettings :one
SELECT * FROM push_settings WHERE workspace_id = @workspace_id;

-- name: PushSettingsUpsert :exec
INSERT INTO push_settings (workspace_id, frequency_cap, quiet_hours, updated_by, updated_at)
VALUES (@workspace_id, @frequency_cap, @quiet_hours, sqlc.narg(updated_by), @updated_at)
ON CONFLICT (workspace_id) DO UPDATE SET frequency_cap = EXCLUDED.frequency_cap, quiet_hours = EXCLUDED.quiet_hours, updated_by = EXCLUDED.updated_by,
    updated_at = EXCLUDED.updated_at;

-- name: PushChannelWindow :one
-- The window texts, notifications and WhatsApp messages share (the SMS
-- rules), and whether push and WhatsApp keep to it.
SELECT quiet_from, quiet_to, push_same_window, whatsapp_same_window, opt_out_keywords, opt_in_keywords, help_keywords,
    help_reply, opt_out_reply, opt_in_reply
FROM sms_settings WHERE workspace_id = @workspace_id;

-- name: PushSetSameWindow :execrows
-- Push's quiet hours, kept with the SMS rules once the workspace has them
-- (sms_settings.push_same_window). Never creates the SMS rules: they start
-- with their own defaults when SMS is set up.
UPDATE sms_settings SET push_same_window = @same_window WHERE workspace_id = @workspace_id;

-- name: WhatsAppSetSameWindow :execrows
UPDATE sms_settings SET whatsapp_same_window = @same_window WHERE workspace_id = @workspace_id;

-- name: PushSites :many
SELECT s.*, coalesce(u.name, '')::text AS added_by_name
FROM push_sites s LEFT JOIN users u ON u.id = s.added_by
WHERE s.workspace_id = @workspace_id AND s.removed_at IS NULL
ORDER BY CASE s.kind WHEN 'publication' THEN 0 ELSE 1 END, s.added_at, s.id;

-- name: PushSite :one
SELECT * FROM push_sites WHERE id = @id AND workspace_id = @workspace_id AND removed_at IS NULL;

-- name: PushSiteAny :one
SELECT * FROM push_sites WHERE id = @id;

-- name: PushSiteByOrigin :one
SELECT * FROM push_sites WHERE workspace_id = @workspace_id AND origin = @origin AND removed_at IS NULL;

-- name: PushSitesAll :many
-- Every listed site of the instance, for the daily service worker check.
SELECT s.* FROM push_sites s
JOIN workspaces w ON w.id = s.workspace_id AND w.deleted_at IS NULL
WHERE s.removed_at IS NULL AND s.kind = 'snippet'
ORDER BY s.id;

-- name: PushSiteInsert :exec
INSERT INTO push_sites (id, workspace_id, origin, host, kind, label, worker_path, worker_mode, check_status, demo, added_by, added_at)
VALUES (@id, @workspace_id, @origin, @host, @kind, @label, @worker_path, @worker_mode, @check_status, @demo, sqlc.narg(added_by), @added_at);

-- name: PushSiteUpdate :exec
UPDATE push_sites SET label = @label, worker_path = @worker_path, worker_mode = @worker_mode, icon_media_id = sqlc.narg(icon_media_id)
WHERE id = @id AND workspace_id = @workspace_id;

-- name: PushSiteSetCheck :exec
UPDATE push_sites SET check_status = @check_status, check_version = sqlc.narg(check_version), check_scope = @check_scope,
    check_http_status = sqlc.narg(check_http_status), check_detail = @check_detail, checked_at = @checked_at,
    missing_since = CASE WHEN @check_status::text IN ('missing','unreachable','not_a_worker') THEN coalesce(missing_since, @checked_at::timestamptz) ELSE NULL END
WHERE id = @id;

-- name: PushSiteSnippetSeen :exec
UPDATE push_sites SET snippet_seen_at = coalesce(snippet_seen_at, @at::timestamptz) WHERE id = @id;

-- name: PushSiteRemove :exec
UPDATE push_sites SET removed_at = @at, removed_by = sqlc.narg(removed_by) WHERE id = @id AND workspace_id = @workspace_id AND removed_at IS NULL;

-- name: PushSubscriptionsBySite :many
-- Live subscriptions per site.
SELECT site_id, count(*)::bigint AS n, count(*) FILTER (WHERE contact_id IS NOT NULL)::bigint AS linked
FROM push_subscriptions WHERE workspace_id = @workspace_id AND removed_at IS NULL
GROUP BY site_id;

-- name: PushSubscriptionsByBrowser :many
SELECT browser, count(*)::bigint AS n
FROM push_subscriptions WHERE workspace_id = @workspace_id AND removed_at IS NULL
GROUP BY browser;

-- name: PushSubscription :one
SELECT * FROM push_subscriptions WHERE id = @id AND workspace_id = @workspace_id;

-- name: PushSubscriptionByEndpoint :one
SELECT * FROM push_subscriptions WHERE workspace_id = @workspace_id AND endpoint = @endpoint AND removed_at IS NULL;

-- name: PushSubscriptionInsert :exec
INSERT INTO push_subscriptions (id, workspace_id, site_id, key_id, endpoint, p256dh, auth, browser, platform, time_zone, contact_id, linked_at, linked_by,
    prompt_form_id, prompt_version, prompt_text, page, ip, user_agent, created_at, last_seen_at, demo)
VALUES (@id, @workspace_id, @site_id, @key_id, @endpoint, @p256dh, @auth, @browser, @platform, @time_zone, sqlc.narg(contact_id), sqlc.narg(linked_at),
    @linked_by, sqlc.narg(prompt_form_id), sqlc.narg(prompt_version), @prompt_text, @page, sqlc.narg(ip), @user_agent, @created_at, @created_at, @demo);

-- name: PushSubscriptionRefresh :exec
-- The same browser subscribed again (new keys, or the same subscription
-- sent from another page): keep the record, refresh what changed.
UPDATE push_subscriptions SET p256dh = @p256dh, auth = @auth, time_zone = CASE WHEN @time_zone::text = '' THEN time_zone ELSE @time_zone END,
    last_seen_at = @at, failures = 0
WHERE id = @id;

-- name: PushSubscriptionLink :exec
UPDATE push_subscriptions SET contact_id = @contact_id, linked_at = @at, linked_by = @linked_by
WHERE id = @id AND (contact_id IS NULL OR contact_id = @contact_id);

-- name: PushSubscriptionRemove :exec
-- A removed subscription loses its endpoint and keys; its site, browser
-- and contact stay for the counts.
UPDATE push_subscriptions SET removed_at = @at, removed_reason = @reason, endpoint = '', p256dh = NULL, auth = NULL
WHERE id = @id AND removed_at IS NULL;

-- name: PushSubscriptionTouch :exec
UPDATE push_subscriptions SET last_seen_at = @at WHERE id = @id;

-- name: PushSubscriptionFailure :one
UPDATE push_subscriptions SET failures = failures + 1 WHERE id = @id RETURNING failures;

-- name: PushSubscriptionsOfContact :many
SELECT s.*, st.host AS site_host
FROM push_subscriptions s JOIN push_sites st ON st.id = s.site_id
WHERE s.workspace_id = @workspace_id AND s.contact_id = @contact_id AND s.removed_at IS NULL
ORDER BY s.created_at DESC;

-- name: PushTestSubscription :one
SELECT * FROM push_test_subscriptions WHERE user_id = @user_id AND workspace_id = @workspace_id;

-- name: PushTestSubscriptionByID :one
SELECT * FROM push_test_subscriptions WHERE id = @id;

-- name: PushTestSubscriptionUpsert :exec
INSERT INTO push_test_subscriptions (user_id, workspace_id, key_id, id, endpoint, p256dh, auth, browser, platform, created_at)
VALUES (@user_id, @workspace_id, @key_id, @id, @endpoint, @p256dh, @auth, @browser, @platform, @created_at)
ON CONFLICT (user_id, workspace_id) DO UPDATE SET key_id = EXCLUDED.key_id, id = EXCLUDED.id, endpoint = EXCLUDED.endpoint, p256dh = EXCLUDED.p256dh,
    auth = EXCLUDED.auth, browser = EXCLUDED.browser, platform = EXCLUDED.platform, created_at = EXCLUDED.created_at;

-- name: PushPrompt :one
SELECT p.*, f.name, f.status, f.slug, f.created_at AS form_created_at, coalesce(u.name, '')::text AS updated_by_name
FROM push_prompts p JOIN forms f ON f.id = p.form_id
LEFT JOIN users u ON u.id = p.updated_by
WHERE p.form_id = @form_id AND p.workspace_id = @workspace_id;

-- name: PushPrompts :many
SELECT p.*, f.name, f.status, f.slug
FROM push_prompts p JOIN forms f ON f.id = p.form_id
WHERE p.workspace_id = @workspace_id
ORDER BY f.created_at, f.id;

-- name: PushPromptInsert :exec
INSERT INTO push_prompts (form_id, workspace_id, site_id, draft, updated_by, updated_at)
VALUES (@form_id, @workspace_id, sqlc.narg(site_id), @draft, sqlc.narg(updated_by), @updated_at);

-- name: PushPromptSaveDraft :exec
UPDATE push_prompts SET draft = @draft, site_id = sqlc.narg(site_id), updated_by = sqlc.narg(updated_by), updated_at = @updated_at
WHERE form_id = @form_id AND workspace_id = @workspace_id;

-- name: PushPromptPublish :exec
UPDATE push_prompts SET live_version = @live_version, live_since = coalesce(live_since, @at::timestamptz), updated_by = sqlc.narg(updated_by), updated_at = @at
WHERE form_id = @form_id;

-- name: PushPromptVersion :one
SELECT v.*, coalesce(u.name, '')::text AS published_by_name
FROM push_prompt_versions v LEFT JOIN users u ON u.id = v.published_by
WHERE v.form_id = @form_id AND v.version = @version;

-- name: PushPromptVersionInsert :exec
INSERT INTO push_prompt_versions (form_id, version, config, published_by, published_at)
VALUES (@form_id, @version, @config, sqlc.narg(published_by), @published_at);

-- name: PushPromptDayAdd :exec
-- Adds to a prompt's counts of the day.
INSERT INTO push_prompt_days (form_id, workspace_id, day, shown, allowed, granted, not_now, closed, bell_clicked, blocked, ios_steps, ios_added)
VALUES (@form_id, @workspace_id, @day, @shown, @allowed, @granted, @not_now, @closed, @bell_clicked, @blocked, @ios_steps, @ios_added)
ON CONFLICT (form_id, day) DO UPDATE SET shown = push_prompt_days.shown + EXCLUDED.shown, allowed = push_prompt_days.allowed + EXCLUDED.allowed,
    granted = push_prompt_days.granted + EXCLUDED.granted, not_now = push_prompt_days.not_now + EXCLUDED.not_now,
    closed = push_prompt_days.closed + EXCLUDED.closed, bell_clicked = push_prompt_days.bell_clicked + EXCLUDED.bell_clicked,
    blocked = push_prompt_days.blocked + EXCLUDED.blocked, ios_steps = push_prompt_days.ios_steps + EXCLUDED.ios_steps,
    ios_added = push_prompt_days.ios_added + EXCLUDED.ios_added;

-- name: PushPromptTotals :one
-- A prompt's counts since a day.
SELECT coalesce(sum(shown), 0)::bigint AS shown, coalesce(sum(allowed), 0)::bigint AS allowed, coalesce(sum(granted), 0)::bigint AS granted,
    coalesce(sum(not_now), 0)::bigint AS not_now, coalesce(sum(closed), 0)::bigint AS closed, coalesce(sum(bell_clicked), 0)::bigint AS bell_clicked,
    coalesce(sum(blocked), 0)::bigint AS blocked, coalesce(sum(ios_steps), 0)::bigint AS ios_steps, coalesce(sum(ios_added), 0)::bigint AS ios_added
FROM push_prompt_days WHERE form_id = @form_id AND day >= @since;

-- name: PushCampaign :one
SELECT c.id, c.workspace_id, c.name, c.status, c.audience, c.version, c.content_hash, c.created_by, c.updated_by, c.created_at, c.updated_at,
    c.autosaved_at, c.scheduled_at, c.started_at, c.completed_at, c.started_by, c.channel, c.audience_count,
    p.title, p.message, p.link, p.image_media_id, p.buttons, p.ttl_minutes, p.tag, p.replace_tag, p.site_ids, p.lists, p.segments,
    p.schedule_mode, p.send_at, p.outside,
    coalesce(cu.name, '')::text AS created_by_name, coalesce(uu.name, '')::text AS updated_by_name, coalesce(su.name, '')::text AS started_by_name
FROM campaigns c
JOIN push_campaigns p ON p.campaign_id = c.id
LEFT JOIN users cu ON cu.id = c.created_by
LEFT JOIN users uu ON uu.id = c.updated_by
LEFT JOIN users su ON su.id = c.started_by
WHERE c.id = @id AND c.workspace_id = @workspace_id AND c.deleted_at IS NULL;

-- name: PushCampaignInsert :exec
INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, audience, editor_mode, mjml, html, plain_text, version, channel,
    content_hash, created_by, updated_by, created_at, updated_at)
VALUES (@id, @workspace_id, @name, 'draft', '', '', @audience, 'html', '', '', '', 1, 'push', @content_hash, sqlc.narg(by), sqlc.narg(by), @at, @at);

-- name: PushCampaignDetailInsert :exec
INSERT INTO push_campaigns (campaign_id, workspace_id, title, message, link, image_media_id, buttons, ttl_minutes, tag, replace_tag, site_ids, lists, segments,
    schedule_mode, send_at, outside)
VALUES (@campaign_id, @workspace_id, @title, @message, @link, sqlc.narg(image_media_id), @buttons, @ttl_minutes, @tag, @replace_tag, @site_ids, @lists, @segments,
    @schedule_mode, sqlc.narg(send_at), @outside);

-- name: PushCampaignSave :exec
-- An autosave: the campaign row's name, audience and version.
UPDATE campaigns SET name = @name, audience = @audience, content_hash = @content_hash, version = @version, updated_by = sqlc.narg(by),
    updated_at = @at, autosaved_at = @at
WHERE id = @id AND workspace_id = @workspace_id;

-- name: PushCampaignDetailSave :exec
UPDATE push_campaigns SET title = @title, message = @message, link = @link, image_media_id = sqlc.narg(image_media_id), buttons = @buttons,
    ttl_minutes = @ttl_minutes, tag = @tag, replace_tag = @replace_tag, site_ids = @site_ids, lists = @lists, segments = @segments
WHERE campaign_id = @campaign_id;

-- name: PushCampaignScheduleSave :exec
UPDATE push_campaigns SET schedule_mode = @schedule_mode, send_at = sqlc.narg(send_at), outside = @outside WHERE campaign_id = @campaign_id;

-- name: PushCampaignSetStatus :exec
UPDATE campaigns SET status = @status, scheduled_at = sqlc.narg(scheduled_at), started_at = coalesce(sqlc.narg(started_at), started_at),
    started_by = coalesce(sqlc.narg(started_by), started_by), completed_at = sqlc.narg(completed_at), audience_count = coalesce(sqlc.narg(audience_count), audience_count),
    updated_at = @at
WHERE id = @id;

-- name: PushSend :one
SELECT * FROM push_sends WHERE campaign_id = @campaign_id;

-- name: PushSendUpsert :exec
INSERT INTO push_sends (campaign_id, workspace_id, phase, recipients, left_out, started_by, approved_by, scheduled_at)
VALUES (@campaign_id, @workspace_id, @phase, @recipients, @left_out, sqlc.narg(started_by), sqlc.narg(approved_by), @scheduled_at)
ON CONFLICT (campaign_id) DO UPDATE SET phase = EXCLUDED.phase, recipients = EXCLUDED.recipients, left_out = EXCLUDED.left_out,
    started_by = EXCLUDED.started_by, approved_by = EXCLUDED.approved_by, scheduled_at = EXCLUDED.scheduled_at, first_send_at = NULL,
    last_send_at = NULL, completed_at = NULL;

-- name: PushSendSetPhase :exec
UPDATE push_sends SET phase = @phase, completed_at = sqlc.narg(completed_at) WHERE campaign_id = @campaign_id;

-- name: PushSendsDue :many
-- Scheduled push sends whose time has come.
SELECT * FROM push_sends WHERE phase = 'scheduled' AND scheduled_at <= @now ORDER BY scheduled_at LIMIT 20;

-- name: PushMessageInsert :exec
INSERT INTO push_messages (message_id, workspace_id, subscription_id, site_id, browser, token)
VALUES (@message_id, @workspace_id, sqlc.narg(subscription_id), sqlc.narg(site_id), @browser, @token);

-- name: PushMessage :one
SELECT m.*, p.subscription_id, p.site_id, p.browser, p.token, p.answer, p.shown_at, p.clicked_at, p.closed_at
FROM channel_messages m JOIN push_messages p ON p.message_id = m.id
WHERE m.id = @id;

-- name: PushMessageAnswer :exec
UPDATE push_messages SET answer = @answer WHERE message_id = @message_id;

-- name: PushMessageShown :execrows
UPDATE push_messages SET shown_at = @at WHERE message_id = @message_id AND shown_at IS NULL;

-- name: PushMessageClicked :execrows
UPDATE push_messages SET clicked_at = @at WHERE message_id = @message_id AND clicked_at IS NULL;

-- name: PushMessageClosed :execrows
UPDATE push_messages SET closed_at = @at WHERE message_id = @message_id AND closed_at IS NULL;

-- name: PushClickInsert :exec
INSERT INTO push_clicks (id, workspace_id, message_id, action, target, occurred_at)
VALUES (@id, @workspace_id, @message_id, @action, @target, @occurred_at);
