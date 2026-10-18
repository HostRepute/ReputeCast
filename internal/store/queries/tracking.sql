-- Open and click tracking (spec 5.5, 7.6).

-- name: TrackingMessage :one
SELECT m.id, m.workspace_id, m.stream, m.campaign_id, m.automation_id, m.contact_id, m.email, m.variant, m.provider_id,
       m.mailbox_provider, m.sent_at, m.delivered_at, m.opened_at, m.human_opened_at, m.clicked_at,
       coalesce(m.content_ref, '{}'::jsonb)::jsonb AS content_ref,
       c.started_at AS campaign_started_at, m.kind
FROM messages m
LEFT JOIN campaigns c ON c.id = m.campaign_id
WHERE m.id = @id;

-- name: TrackingCampaignLinks :many
SELECT position, url FROM campaign_links WHERE campaign_id = @campaign_id ORDER BY position;

-- name: TrackingAutomationLinks :many
SELECT position, url FROM automation_email_links WHERE email_id = @email_id AND version = @version AND language = @language ORDER BY position;

-- name: TrackingContactVars :one
SELECT email, first_name, last_name, attributes FROM contacts WHERE id = @id;

-- name: TrackingMarkOpen :one
-- Records an open on a message and says what changed, so counters move
-- once per message: the first open, or the first human open after a
-- machine prefetch (prev_opened_at is when that prefetch was).
WITH old AS (SELECT opened_at, human_opened_at FROM messages WHERE id = @id FOR UPDATE)
UPDATE messages m
SET opened_at = coalesce(m.opened_at, @at::timestamptz),
    human_opened_at = CASE WHEN @human::boolean THEN coalesce(m.human_opened_at, @at::timestamptz) ELSE m.human_opened_at END
FROM old
WHERE m.id = @id
RETURNING (old.opened_at IS NULL)::boolean AS first_open,
          (old.human_opened_at IS NULL AND @human::boolean)::boolean AS first_human,
          old.opened_at AS prev_opened_at;

-- name: TrackingMarkClick :one
-- A click counts as a human open too.
WITH old AS (SELECT opened_at, human_opened_at, clicked_at FROM messages WHERE id = @id FOR UPDATE)
UPDATE messages m
SET clicked_at = coalesce(m.clicked_at, @at::timestamptz),
    opened_at = coalesce(m.opened_at, @at::timestamptz),
    human_opened_at = coalesce(m.human_opened_at, @at::timestamptz)
FROM old
WHERE m.id = @id
RETURNING (old.clicked_at IS NULL)::boolean AS first_click,
          (old.opened_at IS NULL)::boolean AS first_open,
          (old.human_opened_at IS NULL)::boolean AS first_human,
          old.opened_at AS prev_opened_at;

-- name: TrackingFirstLinkClick :one
INSERT INTO tracking_link_clicks (message_id, link, clicked_at) VALUES (@message_id, @link, @at)
ON CONFLICT DO NOTHING
RETURNING link;

-- name: TrackingBumpLink :exec
UPDATE campaign_links SET clicks = clicks + 1, unique_clicks = unique_clicks + @unique_clicks
WHERE campaign_id = @campaign_id AND position = @position;

-- name: TrackingBumpAutomationLink :exec
UPDATE automation_email_links SET clicks = clicks + 1 WHERE email_id = @email_id AND version = @version AND language = @language AND position = @position;

-- name: TrackingBumpCounters :exec
INSERT INTO campaign_counters (campaign_id, provider_id, mailbox_provider, variant, opened_human, opened_machine, clicked)
VALUES (@campaign_id, @provider_id, @mailbox_provider, @variant, @opened_human, @opened_machine, @clicked)
ON CONFLICT (campaign_id, provider_id, mailbox_provider, variant) DO UPDATE
SET opened_human = campaign_counters.opened_human + EXCLUDED.opened_human,
    opened_machine = campaign_counters.opened_machine + EXCLUDED.opened_machine,
    clicked = campaign_counters.clicked + EXCLUDED.clicked;

-- name: TrackingBumpHourly :exec
INSERT INTO campaign_hourly (campaign_id, hour, opened_human, opened_machine, clicked)
VALUES (@campaign_id, @hour, @opened_human, @opened_machine, @clicked)
ON CONFLICT (campaign_id, hour) DO UPDATE
SET opened_human = campaign_hourly.opened_human + EXCLUDED.opened_human,
    opened_machine = campaign_hourly.opened_machine + EXCLUDED.opened_machine,
    clicked = campaign_hourly.clicked + EXCLUDED.clicked;

-- name: TrackingInsertEvent :exec
INSERT INTO delivery_events (id, workspace_id, message_id, campaign_id, contact_id, provider_id, type, occurred_at, received_at,
                             mailbox_provider, original_type, diagnostic, machine, url, user_agent, ip, recipient, source)
VALUES (@id, @workspace_id, @message_id, sqlc.narg(campaign_id), sqlc.narg(contact_id), sqlc.narg(provider_id), @type, @occurred_at, @received_at,
        @mailbox_provider, @original_type, @diagnostic, @machine, @url, @user_agent, sqlc.narg(ip), @recipient, @source);

-- name: TrackingTouchContactOpen :exec
-- last_open_at counts every open, last_human_open_at only people's (tiers
-- and the sunset policy use it); last_open_machine says what the latest was.
UPDATE contacts
SET last_open_at = greatest(coalesce(last_open_at, @at::timestamptz), @at::timestamptz),
    last_human_open_at = CASE WHEN @human::boolean THEN greatest(coalesce(last_human_open_at, @at::timestamptz), @at::timestamptz) ELSE last_human_open_at END,
    last_open_machine = CASE WHEN last_open_at IS NULL OR @at::timestamptz >= last_open_at THEN NOT @human::boolean ELSE last_open_machine END,
    last_activity_at = greatest(coalesce(last_activity_at, @at::timestamptz), @at::timestamptz),
    updated_at = now()
WHERE id = @id;

-- name: TrackingTouchContactClick :exec
UPDATE contacts
SET last_click_at = greatest(coalesce(last_click_at, @at::timestamptz), @at::timestamptz),
    last_open_at = greatest(coalesce(last_open_at, @at::timestamptz), @at::timestamptz),
    last_human_open_at = greatest(coalesce(last_human_open_at, @at::timestamptz), @at::timestamptz),
    last_open_machine = false,
    last_activity_at = greatest(coalesce(last_activity_at, @at::timestamptz), @at::timestamptz),
    updated_at = now()
WHERE id = @id;

-- name: TrackingSourceLabel :one
SELECT coalesce(c.name, a.name, '')::text
FROM messages m
LEFT JOIN campaigns c ON c.id = m.campaign_id
LEFT JOIN automations a ON a.id = m.automation_id
WHERE m.id = @id;
