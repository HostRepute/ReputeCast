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

-- name: TrackingFirstLinkClick :one
INSERT INTO tracking_link_clicks (message_id, link, clicked_at) VALUES (@message_id, @link, @at)
ON CONFLICT DO NOTHING
RETURNING link;

-- name: TrackingBumpAutomationLink :exec
UPDATE automation_email_links SET clicks = clicks + 1 WHERE email_id = @email_id AND version = @version AND language = @language AND position = @position;

-- name: TrackingInsertEvent :exec
-- class is how tracking classified the open or click (human, privacy,
-- machine; human, bot); link the clicked link's position.
INSERT INTO delivery_events (id, workspace_id, message_id, campaign_id, contact_id, provider_id, type, occurred_at, received_at,
                             mailbox_provider, original_type, diagnostic, machine, url, user_agent, ip, recipient, source, class, link)
VALUES (@id, @workspace_id, @message_id, sqlc.narg(campaign_id), sqlc.narg(contact_id), sqlc.narg(provider_id), @type, @occurred_at, @received_at,
        @mailbox_provider, @original_type, @diagnostic, @machine, @url, @user_agent, sqlc.narg(ip), @recipient, @source, @class, sqlc.narg(link));

-- name: TrackingSourceLabel :one
SELECT coalesce(c.name, a.name, '')::text
FROM messages m
LEFT JOIN campaigns c ON c.id = m.campaign_id
LEFT JOIN automations a ON a.id = m.automation_id
WHERE m.id = @id;

-- name: TrackingTouchContacts :exec
-- A batch's opens and clicks on contacts, one row per contact in id order
-- (the last lock a tracking batch takes). opened is the latest open, human
-- the latest person's open or click, clicked the latest click; machine says
-- the latest open was a machine's.
UPDATE contacts c
SET last_open_at = CASE WHEN x.opened IS NULL THEN c.last_open_at ELSE greatest(coalesce(c.last_open_at, x.opened), x.opened) END,
    last_human_open_at = CASE WHEN x.human IS NULL THEN c.last_human_open_at ELSE greatest(coalesce(c.last_human_open_at, x.human), x.human) END,
    last_click_at = CASE WHEN x.clicked IS NULL THEN c.last_click_at ELSE greatest(coalesce(c.last_click_at, x.clicked), x.clicked) END,
    last_open_machine = CASE WHEN x.opened IS NOT NULL AND (c.last_open_at IS NULL OR x.opened >= c.last_open_at) THEN x.machine ELSE c.last_open_machine END,
    last_activity_at = greatest(coalesce(c.last_activity_at, x.activity), x.activity),
    updated_at = now()
FROM (
    -- 'epoch' stands for "none" in the time arrays.
    SELECT unnest(@ids::uuid[]) AS id, nullif(unnest(@opened::timestamptz[]), 'epoch') AS opened, nullif(unnest(@human::timestamptz[]), 'epoch') AS human,
           nullif(unnest(@clicked::timestamptz[]), 'epoch') AS clicked, unnest(@machine::boolean[]) AS machine, unnest(@activity::timestamptz[]) AS activity
) x
WHERE c.id = x.id AND c.id IN (SELECT cc.id FROM contacts cc WHERE cc.id = ANY(@ids::uuid[]) ORDER BY cc.id FOR UPDATE);

-- name: TrackingRetryPark :exec
-- Opens and clicks a batch couldn't record, kept for the tracking.retry job.
INSERT INTO tracking_retry (message_id, event, next_at, error)
SELECT unnest(@message_ids::uuid[]), unnest(@events::jsonb[]), @next_at, @error;

-- name: TrackingRetryClaim :many
-- Takes due events for one retry: they are put off by the backoff first,
-- so a worker that dies leaves them for a later run.
UPDATE tracking_retry r
SET attempts = r.attempts + 1, next_at = @next_at
WHERE r.id IN (SELECT t.id FROM tracking_retry t WHERE t.next_at <= @now AND t.attempts < @max_attempts ORDER BY t.next_at, t.id LIMIT @lim FOR UPDATE SKIP LOCKED)
RETURNING r.id, r.message_id, r.event, r.attempts;

-- name: TrackingRetryDone :exec
DELETE FROM tracking_retry WHERE id = ANY(@ids::bigint[]);

-- name: TrackingRetryFailed :exec
UPDATE tracking_retry SET error = @error WHERE id = ANY(@ids::bigint[]);

-- name: TrackingRetryPurge :execrows
-- Events that kept failing for 30 days are dropped (the error stays in the
-- log of each attempt).
DELETE FROM tracking_retry WHERE created_at < @before;

-- name: TrackingRetryPending :one
SELECT count(*)::bigint FROM tracking_retry WHERE attempts < @max_attempts;

-- name: TrackingPendingClick :exec
-- A click that looks like a person's waits to settle (review AN-2).
INSERT INTO tracking_pending_clicks (event_id, message_id, link, url, clicked_at, settle_at)
VALUES (@event_id, @message_id, @link, @url, @clicked_at, @settle_at);

-- name: TrackingDueClicks :many
-- Pending clicks due to settle, each message's together and in message
-- order (the order writers lock messages in), with what the click sent;
-- reported: the sending provider reported it (review AN-4).
SELECT p.event_id, p.message_id, p.link, p.url, p.clicked_at,
       coalesce(e.user_agent, '')::text AS user_agent, e.ip,
       (coalesce(e.original_type, 'ReputeCast tracking') <> 'ReputeCast tracking')::boolean AS reported
FROM tracking_pending_clicks p
LEFT JOIN delivery_events e ON e.id = p.event_id
WHERE p.settle_at <= @now
ORDER BY p.message_id, p.clicked_at, p.event_id
LIMIT @lim
FOR UPDATE OF p SKIP LOCKED;

-- name: TrackingNextSettle :one
SELECT coalesce(min(settle_at), 'epoch'::timestamptz)::timestamptz FROM tracking_pending_clicks;

-- name: TrackingMessageClicks :many
-- A message's clicks between two times (bursts, the hidden link); link -1
-- is the hidden link, -2 a link that isn't one of the email's (a provider
-- reported it, review AN-4).
SELECT occurred_at, coalesce(link, -2)::integer AS link, class, url
FROM delivery_events
WHERE message_id = @message_id AND type = 'click' AND occurred_at BETWEEN @from_at AND @until_at
ORDER BY occurred_at;

-- name: TrackingSettleEvent :exec
UPDATE delivery_events SET class = @class, machine = @machine, diagnostic = @diagnostic WHERE id = @id;

-- name: TrackingClassifyEvent :exec
-- An open or click the sending provider reported (stored by ingest),
-- classified in place (review AN-4).
UPDATE delivery_events
SET class = @class, machine = @machine, diagnostic = @diagnostic, link = sqlc.narg(link), ip = sqlc.narg(ip)
WHERE id = @id;

-- name: TrackingSettled :exec
DELETE FROM tracking_pending_clicks WHERE event_id = ANY(@ids::uuid[]);

-- name: TrackingNetworkList :many
-- Networks of privacy proxies and mail scanners.
SELECT cidr::text AS cidr, name, class, source, note, created_at
FROM tracking_networks
ORDER BY class, name, cidr;

-- name: TrackingNetworkAdd :one
INSERT INTO tracking_networks (cidr, name, class, source, note, created_by)
VALUES (@cidr::cidr, @name, @class, 'admin', @note, sqlc.narg(created_by))
ON CONFLICT (cidr) DO NOTHING
RETURNING cidr::text AS cidr;

-- name: TrackingNetworkDelete :one
-- Only networks an admin added can be removed; the built-in ones come with
-- releases.
DELETE FROM tracking_networks WHERE cidr = @cidr::cidr AND source = 'admin'
RETURNING cidr::text AS cidr, name, class, note;
