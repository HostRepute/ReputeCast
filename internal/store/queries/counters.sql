-- Campaign counters kept in step with message states (review AN-3,
-- internal/counters). A writer moves a message (the statement returns its
-- state before and after), collects the counter changes in a ledger and
-- applies them at the end of its transaction, in key order: message rows
-- first (each writer locks them in id order), then campaign_counters, then
-- campaign_hourly, then campaign_links. Every writer takes its locks in
-- that one order, so concurrent writers wait for each other instead of
-- deadlocking.

-- name: CountersMoveDelivery :one
-- Moves a message on a delivery event: delivered and deferred are replaced
-- by bounces and complaints, never the other way round; a soft bounce is
-- remembered so a later delivery takes it back. kind is the event type
-- (delivered, deferred, hard_bounce, soft_bounce, complaint, rejected,
-- unsubscribe).
WITH old AS (
    SELECT m.id, m.campaign_id, m.status, m.sent_at, m.delivered_at, m.deferred_at, m.soft_bounced_at, m.opened_at, m.human_opened_at,
           m.privacy_opened_at, m.clicked_at, m.unsubscribed_at
    FROM messages m
    WHERE m.id = @id
    FOR UPDATE
)
UPDATE messages m
SET status = CASE
        WHEN @kind::text = 'complaint' THEN 'complained'
        WHEN @kind::text = 'hard_bounce' AND m.status <> 'complained' THEN 'bounced'
        WHEN @kind::text = 'rejected' AND m.status IN ('queued', 'sent', 'deferred', 'held') THEN 'rejected'
        WHEN @kind::text = 'delivered' AND m.status IN ('queued', 'sent', 'deferred', 'held') THEN 'delivered'
        WHEN @kind::text IN ('deferred', 'soft_bounce') AND m.status IN ('queued', 'sent', 'held') THEN 'deferred'
        ELSE m.status END,
    delivered_at = CASE WHEN @kind::text = 'delivered' THEN coalesce(m.delivered_at, @at::timestamptz) ELSE m.delivered_at END,
    deferred_at = CASE WHEN @kind::text = 'deferred' THEN coalesce(m.deferred_at, @at::timestamptz) ELSE m.deferred_at END,
    soft_bounced_at = CASE WHEN @kind::text = 'soft_bounce' THEN coalesce(m.soft_bounced_at, @at::timestamptz) ELSE m.soft_bounced_at END,
    unsubscribed_at = CASE WHEN @kind::text = 'unsubscribe' THEN coalesce(m.unsubscribed_at, @at::timestamptz) ELSE m.unsubscribed_at END,
    error = CASE WHEN @kind::text IN ('hard_bounce', 'soft_bounce', 'deferred', 'rejected') AND @error::text <> '' THEN @error::text ELSE m.error END
FROM old
LEFT JOIN campaigns c ON c.id = old.campaign_id
WHERE m.id = old.id
RETURNING m.campaign_id, m.provider_id, m.mailbox_provider, (CASE WHEN m.ab_rollout THEN '' ELSE m.variant END)::text AS variant, c.started_at AS campaign_started_at,
          old.status AS old_status, old.sent_at AS old_sent_at, old.delivered_at AS old_delivered_at, old.deferred_at AS old_deferred_at,
          old.soft_bounced_at AS old_soft_bounced_at, old.opened_at AS old_opened_at, old.human_opened_at AS old_human_opened_at,
          old.privacy_opened_at AS old_privacy_opened_at, old.clicked_at AS old_clicked_at, old.unsubscribed_at AS old_unsubscribed_at,
          m.status AS new_status, m.sent_at AS new_sent_at, m.delivered_at AS new_delivered_at, m.deferred_at AS new_deferred_at,
          m.soft_bounced_at AS new_soft_bounced_at, m.opened_at AS new_opened_at, m.human_opened_at AS new_human_opened_at,
          m.privacy_opened_at AS new_privacy_opened_at, m.clicked_at AS new_clicked_at, m.unsubscribed_at AS new_unsubscribed_at;

-- name: CountersLockMessage :exec
-- A writer locks the message before anything that refers to it (a
-- delivery event's foreign key takes a shared lock on it; two writers
-- holding that shared lock would both wait to move the message).
SELECT 1 FROM messages WHERE id = @id FOR UPDATE;

-- name: CountersMoveEngagement :one
-- Records an open or a person's click on a message. class is how tracking
-- classified it: human, privacy (Apple Mail Privacy Protection fetched the
-- images) or machine (a scanner). A click is always a person's (bot clicks
-- never get here) and is a human open too.
WITH old AS (
    SELECT m.id, m.campaign_id, m.status, m.sent_at, m.delivered_at, m.deferred_at, m.soft_bounced_at, m.opened_at, m.human_opened_at,
           m.privacy_opened_at, m.clicked_at, m.unsubscribed_at
    FROM messages m
    WHERE m.id = @id
    FOR UPDATE
)
UPDATE messages m
SET opened_at = coalesce(m.opened_at, @at::timestamptz),
    human_opened_at = CASE WHEN @click::boolean OR @class::text = 'human' THEN coalesce(m.human_opened_at, @at::timestamptz) ELSE m.human_opened_at END,
    privacy_opened_at = CASE WHEN NOT @click::boolean AND @class::text = 'privacy' THEN coalesce(m.privacy_opened_at, @at::timestamptz) ELSE m.privacy_opened_at END,
    clicked_at = CASE WHEN @click::boolean THEN coalesce(m.clicked_at, @at::timestamptz) ELSE m.clicked_at END
FROM old
LEFT JOIN campaigns c ON c.id = old.campaign_id
WHERE m.id = old.id
RETURNING m.campaign_id, m.provider_id, m.mailbox_provider, (CASE WHEN m.ab_rollout THEN '' ELSE m.variant END)::text AS variant, c.started_at AS campaign_started_at,
          old.status AS old_status, old.sent_at AS old_sent_at, old.delivered_at AS old_delivered_at, old.deferred_at AS old_deferred_at,
          old.soft_bounced_at AS old_soft_bounced_at, old.opened_at AS old_opened_at, old.human_opened_at AS old_human_opened_at,
          old.privacy_opened_at AS old_privacy_opened_at, old.clicked_at AS old_clicked_at, old.unsubscribed_at AS old_unsubscribed_at,
          m.status AS new_status, m.sent_at AS new_sent_at, m.delivered_at AS new_delivered_at, m.deferred_at AS new_deferred_at,
          m.soft_bounced_at AS new_soft_bounced_at, m.opened_at AS new_opened_at, m.human_opened_at AS new_human_opened_at,
          m.privacy_opened_at AS new_privacy_opened_at, m.clicked_at AS new_clicked_at, m.unsubscribed_at AS new_unsubscribed_at;

-- name: CountersBump :exec
-- Adds counter deltas, one row per cell, locking the rows in key order.
INSERT INTO campaign_counters (campaign_id, provider_id, mailbox_provider, variant, sent, delivered, deferred, hard_bounced, soft_bounced,
                               complained, rejected, opened_human, opened_privacy, opened_machine, clicked, unsubscribed)
SELECT x.campaign_id, x.provider_id, x.mailbox, x.variant, x.sent, x.delivered, x.deferred, x.hard, x.soft,
       x.complained, x.rejected, x.human, x.privacy, x.machine, x.clicked, x.unsubscribed
FROM (SELECT unnest(@campaign_ids::uuid[]) AS campaign_id, unnest(@provider_ids::uuid[]) AS provider_id,
             unnest(@mailboxes::text[]) AS mailbox, unnest(@variants::text[]) AS variant,
             unnest(@sent::bigint[]) AS sent, unnest(@delivered::bigint[]) AS delivered, unnest(@deferred::bigint[]) AS deferred,
             unnest(@hard::bigint[]) AS hard, unnest(@soft::bigint[]) AS soft, unnest(@complained::bigint[]) AS complained,
             unnest(@rejected::bigint[]) AS rejected, unnest(@human::bigint[]) AS human, unnest(@privacy::bigint[]) AS privacy,
             unnest(@machine::bigint[]) AS machine, unnest(@clicked::bigint[]) AS clicked, unnest(@unsubscribed::bigint[]) AS unsubscribed) AS x
ORDER BY x.campaign_id, x.provider_id, x.mailbox, x.variant
ON CONFLICT (campaign_id, provider_id, mailbox_provider, variant) DO UPDATE
SET sent = campaign_counters.sent + EXCLUDED.sent,
    delivered = campaign_counters.delivered + EXCLUDED.delivered,
    deferred = campaign_counters.deferred + EXCLUDED.deferred,
    hard_bounced = campaign_counters.hard_bounced + EXCLUDED.hard_bounced,
    soft_bounced = campaign_counters.soft_bounced + EXCLUDED.soft_bounced,
    complained = campaign_counters.complained + EXCLUDED.complained,
    rejected = campaign_counters.rejected + EXCLUDED.rejected,
    opened_human = campaign_counters.opened_human + EXCLUDED.opened_human,
    opened_privacy = campaign_counters.opened_privacy + EXCLUDED.opened_privacy,
    opened_machine = campaign_counters.opened_machine + EXCLUDED.opened_machine,
    clicked = campaign_counters.clicked + EXCLUDED.clicked,
    unsubscribed = campaign_counters.unsubscribed + EXCLUDED.unsubscribed;

-- name: CountersBumpHourly :exec
-- Adds hourly engagement deltas, locking the rows in key order.
INSERT INTO campaign_hourly (campaign_id, hour, opened_human, opened_privacy, opened_machine, clicked)
SELECT x.campaign_id, x.hour, x.human, x.privacy, x.machine, x.clicked
FROM (SELECT unnest(@campaign_ids::uuid[]) AS campaign_id, unnest(@hours::integer[]) AS hour,
             unnest(@human::bigint[]) AS human, unnest(@privacy::bigint[]) AS privacy, unnest(@machine::bigint[]) AS machine,
             unnest(@clicked::bigint[]) AS clicked) AS x
ORDER BY x.campaign_id, x.hour
ON CONFLICT (campaign_id, hour) DO UPDATE
SET opened_human = campaign_hourly.opened_human + EXCLUDED.opened_human,
    opened_privacy = campaign_hourly.opened_privacy + EXCLUDED.opened_privacy,
    opened_machine = campaign_hourly.opened_machine + EXCLUDED.opened_machine,
    clicked = campaign_hourly.clicked + EXCLUDED.clicked;

-- name: CountersBumpLinks :exec
-- Adds clicks per link (people's clicks, people's first click per link,
-- bot clicks), locking the links in key order.
WITH d AS (
    SELECT unnest(@campaign_ids::uuid[]) AS campaign_id, unnest(@positions::integer[]) AS position,
           unnest(@clicks::bigint[]) AS clicks, unnest(@unique_clicks::bigint[]) AS unique_clicks, unnest(@bot_clicks::bigint[]) AS bot_clicks
), locked AS (
    SELECT l.id FROM campaign_links l JOIN d ON d.campaign_id = l.campaign_id AND d.position = l.position
    ORDER BY l.campaign_id, l.position, l.id
    FOR UPDATE OF l
)
UPDATE campaign_links l
SET clicks = l.clicks + d.clicks, unique_clicks = l.unique_clicks + d.unique_clicks, bot_clicks = l.bot_clicks + d.bot_clicks
FROM d
WHERE l.campaign_id = d.campaign_id AND l.position = d.position AND l.id IN (SELECT id FROM locked);

-- name: CountersBumpLinkVariants :exec
-- Adds clicks per link and A/B variant (review AN-5), in key order, after
-- campaign_links.
WITH d AS (
    SELECT unnest(@campaign_ids::uuid[]) AS campaign_id, unnest(@positions::integer[]) AS position, unnest(@variants::text[]) AS variant,
           unnest(@clicks::bigint[]) AS clicks, unnest(@unique_clicks::bigint[]) AS unique_clicks, unnest(@bot_clicks::bigint[]) AS bot_clicks
)
INSERT INTO campaign_link_variants AS v (campaign_id, position, variant, clicks, unique_clicks, bot_clicks)
SELECT d.campaign_id, d.position, d.variant, d.clicks, d.unique_clicks, d.bot_clicks
FROM d
ORDER BY d.campaign_id, d.position, d.variant
ON CONFLICT (campaign_id, position, variant) DO UPDATE
SET clicks = v.clicks + EXCLUDED.clicks, unique_clicks = v.unique_clicks + EXCLUDED.unique_clicks, bot_clicks = v.bot_clicks + EXCLUDED.bot_clicks;

-- name: CountersCampaignHistory :one
-- Whether a campaign's per-message history is stored, so its counters can
-- be rebuilt from it: imported history and sample data have counters but
-- no message per recipient.
SELECT c.origin,
       c.started_at,
       (SELECT count(*) FROM messages m WHERE m.campaign_id = c.id AND m.sent_at IS NOT NULL)::bigint AS messages_sent,
       (SELECT coalesce(sum(cc.sent), 0) FROM campaign_counters cc WHERE cc.campaign_id = c.id)::bigint AS counted_sent
FROM campaigns c
WHERE c.id = @id;

-- name: CountersLockCampaign :exec
-- A rebuild takes the campaign's counter rows first, so writers wait for
-- it (they take the rows after the messages; the rebuild reads messages
-- without locking them).
SELECT 1 FROM campaign_counters WHERE campaign_id = @campaign_id ORDER BY provider_id, mailbox_provider, variant FOR UPDATE;

-- name: CountersRefreshStates :exec
-- Rebuilds the states the status can't tell apart from the stored events:
-- the first deferral, the first soft bounce, an unsubscribe, a privacy
-- prefetch. Only fills what is missing.
UPDATE messages m
SET deferred_at = coalesce(m.deferred_at, e.deferred_at),
    soft_bounced_at = coalesce(m.soft_bounced_at, e.soft_bounced_at),
    unsubscribed_at = coalesce(m.unsubscribed_at, e.unsubscribed_at),
    privacy_opened_at = coalesce(m.privacy_opened_at, e.privacy_opened_at)
FROM (
    SELECT de.message_id,
           min(de.occurred_at) FILTER (WHERE de.type = 'deferred') AS deferred_at,
           min(de.occurred_at) FILTER (WHERE de.type = 'soft_bounce') AS soft_bounced_at,
           min(de.occurred_at) FILTER (WHERE de.type = 'unsubscribe') AS unsubscribed_at,
           min(de.occurred_at) FILTER (WHERE de.type = 'open' AND (de.class = 'privacy'
               OR (de.class = '' AND de.machine AND de.diagnostic = 'Apple Mail Privacy Protection prefetch'))) AS privacy_opened_at
    FROM delivery_events de
    WHERE de.campaign_id = @campaign_id AND de.message_id IS NOT NULL
      AND de.type IN ('deferred', 'soft_bounce', 'unsubscribe', 'open')
    GROUP BY de.message_id
) e
WHERE m.id = e.message_id AND m.campaign_id = @campaign_id
  AND ((m.deferred_at IS NULL AND e.deferred_at IS NOT NULL) OR (m.soft_bounced_at IS NULL AND e.soft_bounced_at IS NOT NULL)
       OR (m.unsubscribed_at IS NULL AND e.unsubscribed_at IS NOT NULL) OR (m.privacy_opened_at IS NULL AND e.privacy_opened_at IS NOT NULL));

-- name: CountersRebuildCells :exec
-- Replaces a campaign's counters with what its messages' states add up to
-- (the same rules as internal/counters State.Flags).
WITH s AS (
    -- The recipients who got an A/B winner after the test count apart from
    -- the sample (variant '').
    SELECT m.provider_id, m.mailbox_provider, (CASE WHEN m.ab_rollout THEN '' ELSE m.variant END)::text AS variant,
           (m.status = 'rejected') AS rejected,
           (m.sent_at IS NOT NULL AND m.status <> 'rejected') AS sent,
           (m.status = 'bounced') AS hard,
           (m.status = 'complained') AS complained,
           (m.status = 'complained' OR (m.delivered_at IS NOT NULL AND m.status NOT IN ('bounced', 'rejected'))) AS delivered,
           (m.deferred_at IS NOT NULL) AS deferred,
           (m.human_opened_at IS NOT NULL) AS human,
           (m.human_opened_at IS NULL AND m.privacy_opened_at IS NOT NULL) AS privacy,
           (m.human_opened_at IS NULL AND m.privacy_opened_at IS NULL AND m.opened_at IS NOT NULL) AS machine,
           (m.clicked_at IS NOT NULL) AS clicked,
           (m.unsubscribed_at IS NOT NULL) AS unsubscribed,
           m.soft_bounced_at
    FROM messages m
    WHERE m.campaign_id = @campaign_id AND m.provider_id IS NOT NULL
), sums AS (
    SELECT provider_id, mailbox_provider, variant,
           count(*) FILTER (WHERE sent) AS sent,
           count(*) FILTER (WHERE delivered) AS delivered,
           count(*) FILTER (WHERE deferred) AS deferred,
           count(*) FILTER (WHERE hard) AS hard,
           count(*) FILTER (WHERE soft_bounced_at IS NOT NULL AND NOT delivered AND NOT hard AND NOT complained AND NOT rejected) AS soft,
           count(*) FILTER (WHERE complained) AS complained,
           count(*) FILTER (WHERE rejected) AS rejected,
           count(*) FILTER (WHERE human) AS human,
           count(*) FILTER (WHERE privacy) AS privacy,
           count(*) FILTER (WHERE machine) AS machine,
           count(*) FILTER (WHERE clicked) AS clicked,
           count(*) FILTER (WHERE unsubscribed) AS unsubscribed
    FROM s
    GROUP BY provider_id, mailbox_provider, variant
), gone AS (
    DELETE FROM campaign_counters cc WHERE cc.campaign_id = @campaign_id
      AND NOT EXISTS (SELECT 1 FROM sums WHERE sums.provider_id = cc.provider_id AND sums.mailbox_provider = cc.mailbox_provider AND sums.variant = cc.variant)
)
INSERT INTO campaign_counters (campaign_id, provider_id, mailbox_provider, variant, sent, delivered, deferred, hard_bounced, soft_bounced,
                               complained, rejected, opened_human, opened_privacy, opened_machine, clicked, unsubscribed)
SELECT @campaign_id, provider_id, mailbox_provider, variant, sent, delivered, deferred, hard, soft, complained, rejected, human, privacy, machine, clicked, unsubscribed
FROM sums
ORDER BY provider_id, mailbox_provider, variant
ON CONFLICT (campaign_id, provider_id, mailbox_provider, variant) DO UPDATE
SET sent = EXCLUDED.sent, delivered = EXCLUDED.delivered, deferred = EXCLUDED.deferred, hard_bounced = EXCLUDED.hard_bounced,
    soft_bounced = EXCLUDED.soft_bounced, complained = EXCLUDED.complained, rejected = EXCLUDED.rejected, opened_human = EXCLUDED.opened_human,
    opened_privacy = EXCLUDED.opened_privacy, opened_machine = EXCLUDED.opened_machine, clicked = EXCLUDED.clicked, unsubscribed = EXCLUDED.unsubscribed;

-- name: CountersRebuildHourly :exec
-- Replaces a campaign's hourly engagement with its messages' states: a
-- person's first open, a privacy prefetch or a scanner's first open, and
-- the first click, in the hour after the send started they happened in.
WITH b AS (
    SELECT least(greatest(floor(extract(epoch FROM (m.human_opened_at - @started_at::timestamptz)) / 3600), 0), 720)::integer AS hour, 1 AS human, 0 AS privacy, 0 AS machine, 0 AS clicked
    FROM messages m WHERE m.campaign_id = @campaign_id AND m.provider_id IS NOT NULL AND m.human_opened_at IS NOT NULL
    UNION ALL
    SELECT least(greatest(floor(extract(epoch FROM (m.privacy_opened_at - @started_at::timestamptz)) / 3600), 0), 720)::integer, 0, 1, 0, 0
    FROM messages m WHERE m.campaign_id = @campaign_id AND m.provider_id IS NOT NULL AND m.human_opened_at IS NULL AND m.privacy_opened_at IS NOT NULL
    UNION ALL
    SELECT least(greatest(floor(extract(epoch FROM (m.opened_at - @started_at::timestamptz)) / 3600), 0), 720)::integer, 0, 0, 1, 0
    FROM messages m WHERE m.campaign_id = @campaign_id AND m.provider_id IS NOT NULL AND m.human_opened_at IS NULL AND m.privacy_opened_at IS NULL AND m.opened_at IS NOT NULL
    UNION ALL
    SELECT least(greatest(floor(extract(epoch FROM (m.clicked_at - @started_at::timestamptz)) / 3600), 0), 720)::integer, 0, 0, 0, 1
    FROM messages m WHERE m.campaign_id = @campaign_id AND m.provider_id IS NOT NULL AND m.clicked_at IS NOT NULL
), sums AS (
    SELECT hour, sum(human)::bigint AS human, sum(privacy)::bigint AS privacy, sum(machine)::bigint AS machine, sum(clicked)::bigint AS clicked
    FROM b GROUP BY hour
), gone AS (
    DELETE FROM campaign_hourly h WHERE h.campaign_id = @campaign_id AND NOT EXISTS (SELECT 1 FROM sums WHERE sums.hour = h.hour)
)
INSERT INTO campaign_hourly (campaign_id, hour, opened_human, opened_privacy, opened_machine, clicked)
SELECT @campaign_id, hour, human, privacy, machine, clicked FROM sums
ORDER BY hour
ON CONFLICT (campaign_id, hour) DO UPDATE
SET opened_human = EXCLUDED.opened_human, opened_privacy = EXCLUDED.opened_privacy, opened_machine = EXCLUDED.opened_machine, clicked = EXCLUDED.clicked;

-- name: CountersRebuildLinks :exec
-- Replaces clicks per link with the stored clicks: people's clicks and
-- bot clicks (click events with the link's position and their class;
-- clicks still waiting to settle count as neither), people's first click
-- per link (tracking_link_clicks).
UPDATE campaign_links l
SET clicks = coalesce(x.clicks, 0), unique_clicks = coalesce(x.unique_clicks, 0), bot_clicks = coalesce(x.bot_clicks, 0)
FROM (
    SELECT cl.id,
           (SELECT count(*) FROM delivery_events e WHERE e.campaign_id = cl.campaign_id AND e.type = 'click' AND e.link = cl.position AND e.class = 'human') AS clicks,
           (SELECT count(*) FROM tracking_link_clicks t JOIN messages m ON m.id = t.message_id WHERE m.campaign_id = cl.campaign_id AND t.link = cl.position) AS unique_clicks,
           (SELECT count(*) FROM delivery_events e WHERE e.campaign_id = cl.campaign_id AND e.type = 'click' AND e.link = cl.position AND e.class = 'bot') AS bot_clicks
    FROM campaign_links cl
    WHERE cl.campaign_id = @campaign_id
) x
WHERE l.id = x.id AND l.campaign_id = @campaign_id
  -- Clicks recorded before link positions were stored can't be told apart:
  -- those links keep their counts.
  AND NOT EXISTS (SELECT 1 FROM delivery_events e WHERE e.campaign_id = @campaign_id AND e.type = 'click' AND e.link IS NULL);

-- name: CountersClearLinkVariants :exec
-- Before CountersRebuildLinkVariants; kept when clicks without link
-- positions are stored (as CountersRebuildLinks).
DELETE FROM campaign_link_variants v
WHERE v.campaign_id = sqlc.arg(campaign_id)::uuid
  AND NOT EXISTS (SELECT 1 FROM delivery_events e WHERE e.campaign_id = sqlc.arg(campaign_id)::uuid AND e.type = 'click' AND e.link IS NULL);

-- name: CountersRebuildLinkVariants :exec
-- Clicks per link and A/B variant from the stored clicks: people's and
-- bots' click events by the variant their message had, people's first
-- click per link (tracking_link_clicks).
INSERT INTO campaign_link_variants (campaign_id, position, variant, clicks, unique_clicks, bot_clicks)
SELECT @campaign_id::uuid, k.position, k.variant, coalesce(c.clicks, 0), coalesce(u.unique_clicks, 0), coalesce(c.bot_clicks, 0)
FROM (
    SELECT e.link AS position, m.variant
    FROM delivery_events e JOIN messages m ON m.id = e.message_id
    WHERE e.campaign_id = @campaign_id AND e.type = 'click' AND e.link >= 0 AND e.class IN ('human', 'bot') AND m.variant <> ''
    UNION
    SELECT t.link, m.variant
    FROM tracking_link_clicks t JOIN messages m ON m.id = t.message_id
    WHERE m.campaign_id = @campaign_id AND t.link >= 0 AND m.variant <> ''
) k
LEFT JOIN (
    SELECT e.link AS position, m.variant, count(*) FILTER (WHERE e.class = 'human') AS clicks, count(*) FILTER (WHERE e.class = 'bot') AS bot_clicks
    FROM delivery_events e JOIN messages m ON m.id = e.message_id
    WHERE e.campaign_id = @campaign_id AND e.type = 'click' AND e.link >= 0 AND m.variant <> ''
    GROUP BY e.link, m.variant
) c ON c.position = k.position AND c.variant = k.variant
LEFT JOIN (
    SELECT t.link AS position, m.variant, count(*) AS unique_clicks
    FROM tracking_link_clicks t JOIN messages m ON m.id = t.message_id
    WHERE m.campaign_id = @campaign_id AND m.variant <> ''
    GROUP BY t.link, m.variant
) u ON u.position = k.position AND u.variant = k.variant
WHERE NOT EXISTS (SELECT 1 FROM delivery_events e WHERE e.campaign_id = @campaign_id AND e.type = 'click' AND e.link IS NULL)
ORDER BY k.position, k.variant;

-- name: CountersBuildSave :exec
INSERT INTO campaign_counter_builds (campaign_id, version, built_at, skipped)
VALUES (@campaign_id, @version, @built_at, @skipped)
ON CONFLICT (campaign_id) DO UPDATE SET version = EXCLUDED.version, built_at = EXCLUDED.built_at, skipped = EXCLUDED.skipped;

-- name: CountersDueRebuilds :many
-- Campaigns to rebuild: sent from ReputeCast, their results final (72 h
-- after the send ended), and not rebuilt with the current rules yet.
SELECT c.id FROM campaigns c
LEFT JOIN campaign_counter_builds b ON b.campaign_id = c.id
WHERE c.deleted_at IS NULL AND c.origin = '' AND c.channel = 'email' AND c.started_at IS NOT NULL
  AND c.completed_at IS NOT NULL AND c.completed_at < @final_before
  AND (b.campaign_id IS NULL OR b.version < @version)
ORDER BY c.completed_at
LIMIT @lim;
