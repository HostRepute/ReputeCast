-- Guarded sending and the live monitor (review 2026-10, M34).

-- GS-1: messages in doubt -------------------------------------------------

-- name: SendingRequeueStale :execrows
-- Stale claims of providers that dedupe by message id, still inside the
-- provider's idempotency window: back in the queue, sent again with the
-- same key, so the provider drops the copy if it took the first one.
UPDATE messages m SET claimed_at = NULL
FROM (SELECT unnest(@provider_ids::uuid[]) AS provider_id, unnest(@since::timestamptz[]) AS since) AS w
WHERE m.status = 'queued' AND m.claimed_at IS NOT NULL AND m.claimed_at < @stale::timestamptz
  AND m.campaign_id = @campaign_id AND m.provider_id = w.provider_id AND m.claimed_at > w.since;

-- name: SendingMarkInDoubt :many
-- The other stale claims: the call that took them ended without a recorded
-- answer, so nobody knows whether the provider has them. They are in doubt
-- and never sent again on a guess.
UPDATE messages m SET status = 'in_doubt', error = @reason
WHERE m.status = 'queued' AND m.claimed_at IS NOT NULL AND m.claimed_at < @stale::timestamptz AND m.campaign_id = @campaign_id
RETURNING m.id, m.provider_id;

-- name: SendingOutboxRequeueStale :execrows
-- The automation outbox's stale claims of providers that dedupe (as
-- SendingRequeueStale).
UPDATE messages m SET claimed_at = NULL
FROM (SELECT unnest(@provider_ids::uuid[]) AS provider_id, unnest(@since::timestamptz[]) AS since) AS w
WHERE m.status = 'queued' AND m.claimed_at IS NOT NULL AND m.claimed_at < @stale::timestamptz
  AND m.campaign_id IS NULL AND m.stream = 'automation' AND m.provider_id = w.provider_id AND m.claimed_at > w.since;

-- name: SendingOutboxMarkInDoubt :execrows
-- The automation outbox's other stale claims become in doubt.
UPDATE messages m SET status = 'in_doubt', error = @reason
WHERE m.status = 'queued' AND m.claimed_at IS NOT NULL AND m.claimed_at < @stale::timestamptz
  AND m.campaign_id IS NULL AND m.stream = 'automation';

-- name: SendingOutboxStaleProviders :many
-- Providers with automation emails claimed before the stale time.
SELECT DISTINCT provider_id::uuid AS provider_id FROM messages
WHERE status = 'queued' AND claimed_at IS NOT NULL AND claimed_at < @stale::timestamptz
  AND campaign_id IS NULL AND stream = 'automation' AND provider_id IS NOT NULL;

-- name: SendingInDoubtByProvider :many
-- A campaign's messages in doubt per sending provider, with the oldest.
SELECT provider_id, count(*)::bigint AS n, min(coalesce(claimed_at, queued_at))::timestamptz AS since
FROM messages
WHERE campaign_id = @campaign_id AND status = 'in_doubt'
GROUP BY provider_id;

-- name: SendingResolveInDoubt :execrows
-- A person's decision on a campaign's messages in doubt: send them again
-- (back in the queue) or leave them out (cancelled, with why).
UPDATE messages
SET status = CASE WHEN @resend::boolean THEN 'queued' ELSE 'cancelled' END,
    claimed_at = NULL, next_attempt_at = NULL,
    error = CASE WHEN @resend::boolean THEN '' ELSE @reason::text END
WHERE campaign_id = @campaign_id AND status = 'in_doubt'
  AND (sqlc.narg(provider_id)::uuid IS NULL OR provider_id = sqlc.narg(provider_id)::uuid);

-- name: SendingSettleInDoubt :one
-- A provider reported an event for a message in doubt: it took the message
-- after all. The message counts as sent from the event on.
UPDATE messages
SET status = @status, sent_at = coalesce(sent_at, @at::timestamptz), claimed_at = NULL, error = '',
    delivered_at = CASE WHEN @status::text = 'delivered' THEN coalesce(delivered_at, @at::timestamptz) ELSE delivered_at END
WHERE id = @id AND status = 'in_doubt'
RETURNING campaign_id, provider_id, mailbox_provider, variant;

-- GS-4: the provider errors threshold ---------------------------------------

-- name: SendingCallWindow :one
-- A provider's API calls and failed calls within a time window (the API
-- error threshold reads the three minutes before now).
SELECT coalesce(sum(calls), 0)::bigint AS calls, coalesce(sum(errors), 0)::bigint AS errors
FROM provider_call_windows
WHERE provider_id = @provider_id AND minute >= @since AND minute <= @until;

-- GS-3: one outage path ---------------------------------------------------

-- name: SendingSharesOn :many
-- The running sends with an open or held share on a provider: an outage
-- the provider health job finds goes through each one's outage path.
SELECT cr.campaign_id
FROM campaign_routes cr
JOIN campaigns c ON c.id = cr.campaign_id
JOIN campaign_sends s ON s.campaign_id = cr.campaign_id
WHERE cr.provider_id = @provider_id AND cr.status IN ('pending', 'sending', 'held')
  AND c.status IN ('sending', 'paused') AND c.deleted_at IS NULL AND s.phase = 'sending'
ORDER BY cr.campaign_id;

-- name: SendingProbeSeed :many
-- A connected seed inbox of the workspace, for the test message a share
-- retried after an outage sends every minute.
SELECT email FROM placement_seeds
WHERE workspace_id = @workspace_id AND status = 'connected'
ORDER BY email
LIMIT 1;

-- name: SendingSetProbe :execrows
-- A retried share's test messages so far. Written only while the route is
-- as the send loop last read it: a person's decision in the meantime wins.
UPDATE campaign_routes SET status_detail = @new
WHERE campaign_id = @campaign_id AND provider_id = @provider_id AND status = 'held' AND status_detail = @old;

-- GS-5: every move is vetted ----------------------------------------------

-- name: SendingOutboxQueuedOn :one
-- How many automation emails wait on a provider: what its failover would
-- move, vetted as every move is.
SELECT count(*)::bigint FROM messages
WHERE provider_id = @provider_id AND status = 'queued' AND campaign_id IS NULL AND stream = 'automation';

-- GS-2: resuming re-arms; thresholds while paused -------------------------

-- name: SendingSetGuard :exec
-- The guard's state of a send (engine/guardstate.go).
UPDATE campaign_sends SET guard = @guard, updated_at = @at WHERE campaign_id = @campaign_id;

-- name: SendingSetSendThresholds :exec
-- A send's thresholds, changed by a person while it sends or waits.
UPDATE campaign_sends SET thresholds = @thresholds, updated_at = @at WHERE campaign_id = @campaign_id;

-- name: SendingExcludeGroupOn :many
-- Takes a group out of the rest of one sending provider's share (resuming a
-- route with changes); returns their contact ids for another flow.
UPDATE messages m SET status = 'cancelled', claimed_at = NULL, error = @reason
FROM contacts c
WHERE c.id = m.contact_id AND m.campaign_id = @campaign_id AND m.provider_id = @provider_id AND m.status IN ('queued', 'held')
  AND CASE @grp::text
        WHEN 'imported' THEN c.last_human_open_at IS NULL AND c.source LIKE 'import:%' AND (@source::text = '' OR c.source = @source::text)
        WHEN 'lapsed' THEN c.tier IN ('at_risk', 'sunset')
        ELSE c.last_human_open_at IS NULL
      END
RETURNING m.contact_id;

-- GS-6: the complaint watch after a send ----------------------------------

-- name: SendingSetWatch :exec
-- How long complaints are watched after the send ended.
UPDATE campaign_sends SET watch_until = @watch_until WHERE campaign_id = @campaign_id;

-- name: SendingWatchDue :many
-- The sends whose complaint watch runs.
SELECT campaign_id FROM campaign_sends WHERE watch_until IS NOT NULL AND watch_until > @now::timestamptz ORDER BY watch_until;

-- name: SendingEndWatches :exec
-- Watches that ended.
UPDATE campaign_sends SET watch_until = NULL WHERE watch_until IS NOT NULL AND watch_until <= @now::timestamptz;

-- name: SendingIncidentsOfKind :many
-- A campaign's incidents of one kind, open or not (an alert is raised once
-- per scope).
SELECT * FROM campaign_incidents WHERE campaign_id = @campaign_id AND kind = @kind ORDER BY tripped_at;

-- name: SendingPostmasterLatest :many
-- Gmail's latest daily spam rate for a sending domain (Postmaster Tools).
SELECT day, spam_rate FROM postmaster_daily
WHERE workspace_id = @workspace_id AND domain = @domain AND spam_rate IS NOT NULL AND day >= @since::date
ORDER BY day DESC LIMIT 1;

-- GS-8: throttles shared by a workspace's sends -----------------------------

-- name: SendingOpenMailboxWindow :exec
-- Opens a mailbox provider's window: a slot of the workspace's sends to it.
INSERT INTO sending_mailbox_windows (workspace_id, mailbox_provider, slot, used, last_grant)
VALUES (@workspace_id, @mailbox_provider, @slot, 0, 0)
ON CONFLICT (workspace_id, mailbox_provider, slot) DO NOTHING;

-- name: SendingMailboxSenders :one
-- How many sends took part of a mailbox provider's window in a slot.
SELECT coalesce(max(cardinality(senders)), 0)::int FROM sending_mailbox_windows
WHERE workspace_id = @workspace_id AND mailbox_provider = @mailbox_provider AND slot = @slot;

-- name: SendingMailboxReserve :one
-- Takes up to want sends from a mailbox provider's window of cap (open it
-- first), and no more than fair: every send going out at once shares it,
-- so together they keep to the throttle and each gets its part. The send
-- is recorded even when it gets nothing, so the next slot is shared.
UPDATE sending_mailbox_windows
SET senders = CASE WHEN sqlc.arg(campaign_id)::uuid = ANY(senders) THEN senders ELSE array_append(senders, sqlc.arg(campaign_id)::uuid) END,
    last_grant = greatest(least(sqlc.arg(want)::int, sqlc.arg(cap)::int - used, sqlc.arg(fair)::int), 0),
    used = used + greatest(least(sqlc.arg(want)::int, sqlc.arg(cap)::int - used, sqlc.arg(fair)::int), 0)
WHERE workspace_id = sqlc.arg(workspace_id) AND mailbox_provider = sqlc.arg(mailbox_provider) AND slot = sqlc.arg(slot)
RETURNING last_grant;

-- name: SendingMailboxRelease :exec
-- Gives back what a send took from a window and didn't send.
UPDATE sending_mailbox_windows SET used = greatest(used - @n::int, 0)
WHERE workspace_id = @workspace_id AND mailbox_provider = @mailbox_provider AND slot = @slot;

-- name: SendingPurgeMailboxWindows :exec
DELETE FROM sending_mailbox_windows WHERE slot < @before;

-- name: SendingClearMailboxWindows :exec
-- Frees a workspace's mailbox windows over a past interval (the demo
-- command's fast-forward).
DELETE FROM sending_mailbox_windows WHERE workspace_id = @workspace_id AND slot BETWEEN @from_sec AND @to_sec;

-- GS-11: thresholds by bounce type ------------------------------------------

-- name: SendingBounceTypes :many
-- A send's hard bounces by mailbox provider and class (block/policy or
-- unknown users).
SELECT mailbox_provider, bounce_class, count(*)::bigint AS n FROM delivery_events
WHERE campaign_id = @campaign_id AND type = 'hard_bounce'
GROUP BY mailbox_provider, bounce_class;

-- name: SendingDeferredSince :many
-- The messages each mailbox provider deferred, or bounced softly, since a
-- time (each message once).
SELECT mailbox_provider, count(DISTINCT message_id)::bigint AS n FROM delivery_events
WHERE campaign_id = @campaign_id AND type IN ('deferred', 'soft_bounce') AND occurred_at >= @since::timestamptz
GROUP BY mailbox_provider;

-- name: SendingSentSinceByMailbox :many
-- What a send sent to each mailbox provider since a time.
SELECT mailbox_provider, count(*)::bigint AS n FROM messages
WHERE campaign_id = @campaign_id AND sent_at >= @since
GROUP BY mailbox_provider;

-- GS-12: mailbox provider response rules ------------------------------------

-- name: SendingReplyMatches :many
-- How many of a mailbox provider's replies in a send contained a text since
-- a time, per sending provider: the deferrals and bounces the providers
-- report. The text is matched as plain text, not case-sensitive.
SELECT provider_id, count(*)::bigint AS n FROM delivery_events
WHERE campaign_id = @campaign_id AND type IN ('deferred', 'soft_bounce', 'hard_bounce', 'rejected')
  AND occurred_at >= @since::timestamptz AND mailbox_provider = @mailbox_provider
  AND position(lower(@contains::text) IN lower(original_code || ' ' || diagnostic || ' ' || original_type)) > 0
GROUP BY provider_id;

-- name: SendingSeedCount :one
-- How many seed inboxes a workspace connected.
SELECT count(*)::bigint FROM placement_seeds WHERE workspace_id = @workspace_id AND status = 'connected';
