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
