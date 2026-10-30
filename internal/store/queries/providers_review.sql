-- Review 2026-10 (M35, R03): sending providers.

-- name: ProvEventsOverdue :one
-- PR-6: messages a provider accepted since a time (its last event) that
-- have no event yet, split at the 30-minute grace a message gets for its
-- first event: overdue (sent before grace_before) and still in grace.
SELECT count(*) FILTER (WHERE m.sent_at <= @grace_before::timestamptz)::bigint AS overdue,
       count(*) FILTER (WHERE m.sent_at > @grace_before::timestamptz)::bigint AS in_grace
FROM messages m
WHERE m.provider_id = @provider_id AND m.sent_at IS NOT NULL AND m.sent_at > @since::timestamptz
  AND m.status IN ('queued', 'sent');

-- name: ProvStreamSenderVerification :many
-- PR-7: the verification per provider of the senders a stream uses (its
-- default sender, and for Broadcasts the senders of the campaigns sending
-- now), to vet a failover target's domain.
SELECT s.verification FROM senders s
WHERE s.workspace_id = @workspace_id AND s.id IN (
    SELECT CASE @stream::text
             WHEN 'automation' THEN coalesce(w.default_automation_sender_id, w.default_sender_id)
             WHEN 'system' THEN coalesce(w.default_system_sender_id, w.default_sender_id)
             ELSE w.default_sender_id END
    FROM workspaces w WHERE w.id = @workspace_id
    UNION
    SELECT ts.default_sender_id FROM transactional_settings ts WHERE ts.workspace_id = @workspace_id AND @stream::text = 'transactional'
    UNION
    SELECT c.sender_id FROM campaigns c WHERE c.workspace_id = @workspace_id AND @stream::text = 'broadcast' AND c.status = 'sending'
);

-- name: ProvStreamRecentVolume :one
-- PR-7: messages of a stream a provider took in the last hour (what a
-- failover would move per hour).
SELECT count(*)::bigint FROM messages
WHERE provider_id = @provider_id AND stream = @stream AND sent_at >= @since;
