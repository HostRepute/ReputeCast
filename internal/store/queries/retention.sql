-- Data retention (review 2026-11 DP-4, DP-6).

-- name: RetentionWorkspaces :many
-- Workspaces with a retention setting on, and their settings.
SELECT id, unconfirmed_erase_days, event_detail_days FROM workspaces
WHERE deleted_at IS NULL AND (unconfirmed_erase_days > 0 OR event_detail_days > 0);

-- name: RetentionClearEventDetail :execrows
-- Clears the IP address and user agent of up to @lim opens and clicks older
-- than @cutoff in campaigns and automations (or of a message since deleted).
-- Counters, classes and links stay, so reports read the same.
UPDATE delivery_events d SET ip = NULL, user_agent = ''
WHERE d.id IN (
    SELECT e.id FROM delivery_events e LEFT JOIN messages m ON m.id = e.message_id
    WHERE e.workspace_id = @workspace_id AND e.occurred_at < @cutoff::timestamptz
      AND e.type IN ('open', 'click') AND (e.ip IS NOT NULL OR e.user_agent <> '')
      AND (e.message_id IS NULL OR m.id IS NULL OR m.stream IN ('broadcast', 'automation'))
    ORDER BY e.occurred_at
    LIMIT @lim);

-- name: RetentionUnconfirmedDue :many
-- Contacts a double opt-in signup created who never confirmed, older than
-- @before, with nothing else tying them to the workspace: never confirmed,
-- imported or resubscribed, nothing since @before, no subscribed list, no
-- campaign or automation email, no SMS or push subscription, no paid
-- access and no order. Contacts the team added or imported are never due.
SELECT c.id FROM contacts c
WHERE c.workspace_id = @workspace_id AND c.erased_at IS NULL AND c.consent = 'pending' AND c.created_at < @before::timestamptz
  AND c.source <> 'manual' AND c.source NOT LIKE 'import:%'
  AND NOT EXISTS (SELECT 1 FROM consent_events e WHERE e.contact_id = c.id
                  AND (e.action IN ('confirmed', 'imported', 'resubscribed') OR e.occurred_at >= @before::timestamptz))
  AND NOT EXISTS (SELECT 1 FROM form_submissions f WHERE f.contact_id = c.id AND f.created_at >= @before::timestamptz)
  AND NOT EXISTS (SELECT 1 FROM list_memberships m WHERE m.contact_id = c.id AND m.status = 'subscribed')
  AND NOT EXISTS (SELECT 1 FROM messages m WHERE m.contact_id = c.id AND m.stream IN ('broadcast', 'automation'))
  AND NOT EXISTS (SELECT 1 FROM channel_consents x WHERE x.contact_id = c.id AND x.status = 'subscribed')
  AND NOT EXISTS (SELECT 1 FROM push_subscriptions p WHERE p.contact_id = c.id AND p.removed_at IS NULL)
  AND NOT EXISTS (SELECT 1 FROM paid_subscriptions s WHERE s.contact_id = c.id)
  AND NOT EXISTS (SELECT 1 FROM product_orders o WHERE o.contact_id = c.id)
  AND NOT EXISTS (SELECT 1 FROM store_orders o WHERE o.contact_id = c.id)
ORDER BY c.created_at
LIMIT @lim;

-- name: RetentionOptInSendsPurge :execrows
-- Confirmation email claims older than a day no longer limit anything.
DELETE FROM optin_sends WHERE sent_at < @before::timestamptz;

-- name: WsUpdateRetention :exec
UPDATE workspaces SET unconfirmed_erase_days = @unconfirmed_erase_days, event_detail_days = @event_detail_days, updated_at = now()
WHERE id = @id AND deleted_at IS NULL;
