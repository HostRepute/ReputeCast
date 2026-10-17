-- Paid members (internal/earn): paid subscriptions, complimentary access
-- and gifts, and each paid member's timeline.

-- name: EarnSubscription :one
SELECT * FROM paid_subscriptions WHERE id = @id AND workspace_id = @workspace_id;

-- name: EarnSubscriptionForUpdate :one
SELECT * FROM paid_subscriptions WHERE id = @id FOR UPDATE;

-- name: EarnSubscriptionByStripeID :one
SELECT * FROM paid_subscriptions WHERE workspace_id = @workspace_id AND mode = @mode AND stripe_subscription_id = @stripe_subscription_id;

-- name: EarnSubscriptionsOfContact :many
-- Every paid subscription of a contact, the ones with access first.
SELECT sqlc.embed(s), coalesce(t.name, '')::text AS tier_name, coalesce(t.access_rank, 0)::int AS tier_rank
FROM paid_subscriptions s
LEFT JOIN earn_tiers t ON t.id = s.tier_id
WHERE s.contact_id = @contact_id AND s.workspace_id = @workspace_id AND s.status <> 'incomplete'
ORDER BY (s.ended_at IS NULL) DESC, coalesce(t.access_rank, 0) DESC, s.started_at DESC;

-- name: EarnSubscriptionsOfCustomer :many
SELECT * FROM paid_subscriptions WHERE workspace_id = @workspace_id AND mode = @mode AND stripe_customer_id = @stripe_customer_id;

-- name: EarnSubscriptionInsert :exec
INSERT INTO paid_subscriptions (id, workspace_id, contact_id, mode, kind, tier_id, price_id, stripe_price_id, status, period, amount_cents, currency,
    discount, started_at, trial_end, current_period_end, cancel_at, canceled_at, ended_at, end_reason, paid_cents, refunded_cents,
    stripe_customer_id, stripe_subscription_id, billing_email, source_kind, source_campaign_id, source_label, source_detail,
    grant_reason, grant_note, granted_by, grant_until, import_id, stripe_synced_at, created_at, updated_at)
VALUES (@id, @workspace_id, @contact_id, @mode, @kind, sqlc.narg(tier_id), sqlc.narg(price_id), @stripe_price_id, @status, @period, @amount_cents, @currency,
    sqlc.narg(discount), @started_at, sqlc.narg(trial_end), sqlc.narg(current_period_end), sqlc.narg(cancel_at), sqlc.narg(canceled_at), sqlc.narg(ended_at), @end_reason,
    @paid_cents, @refunded_cents, @stripe_customer_id, @stripe_subscription_id, @billing_email, @source_kind, sqlc.narg(source_campaign_id), @source_label, @source_detail,
    @grant_reason, @grant_note, sqlc.narg(granted_by), sqlc.narg(grant_until), sqlc.narg(import_id), sqlc.narg(stripe_synced_at), @at, @at);

-- name: EarnSubscriptionSetState :exec
-- A subscription's state as the provider last reported it.
UPDATE paid_subscriptions SET tier_id = sqlc.narg(tier_id), price_id = sqlc.narg(price_id), stripe_price_id = @stripe_price_id, status = @status,
    period = @period, amount_cents = @amount_cents, currency = @currency, discount = sqlc.narg(discount), trial_end = sqlc.narg(trial_end),
    current_period_end = sqlc.narg(current_period_end), cancel_at = sqlc.narg(cancel_at), canceled_at = sqlc.narg(canceled_at),
    ended_at = sqlc.narg(ended_at), end_reason = @end_reason, stripe_customer_id = @stripe_customer_id, stripe_synced_at = @synced_at, updated_at = @at
WHERE id = @id;

-- name: EarnSubscriptionPaid :exec
-- A payment: lifetime value grows, any failure is cleared.
UPDATE paid_subscriptions SET paid_cents = paid_cents + @amount, failed_at = NULL, failure_reason = '', next_retry_at = NULL, updated_at = @at
WHERE id = @id;

-- name: EarnSubscriptionRefunded :exec
UPDATE paid_subscriptions SET refunded_cents = refunded_cents + @amount, updated_at = @at WHERE id = @id;

-- name: EarnSubscriptionFailed :exec
UPDATE paid_subscriptions SET failed_at = @failed_at, failure_reason = @failure_reason, next_retry_at = sqlc.narg(next_retry_at), updated_at = @at
WHERE id = @id;

-- name: EarnSubscriptionEmailed :exec
UPDATE paid_subscriptions SET emailed = array_append(emailed, @email::text) WHERE id = @id AND NOT (@email::text = ANY(emailed));

-- name: EarnSubscriptionSetBillingEmail :exec
UPDATE paid_subscriptions SET billing_email = @billing_email, updated_at = @at
WHERE workspace_id = @workspace_id AND mode = @mode AND stripe_customer_id = @stripe_customer_id;

-- name: EarnSubscriptionCancelRequested :exec
UPDATE paid_subscriptions SET canceled_at = coalesce(canceled_at, @at::timestamptz), canceled_by = sqlc.narg(canceled_by),
    cancel_reason = CASE WHEN @reason::text = '' THEN cancel_reason ELSE @reason END, updated_at = @at
WHERE id = @id;

-- name: EarnSubscriptionEnd :exec
UPDATE paid_subscriptions SET status = 'canceled', ended_at = coalesce(ended_at, @at::timestamptz), end_reason = @end_reason, updated_at = @at
WHERE id = @id;

-- name: EarnSubscriptionSetGrant :exec
UPDATE paid_subscriptions SET tier_id = @tier_id, grant_until = sqlc.narg(grant_until), grant_reason = @grant_reason, grant_note = @grant_note, updated_at = @at
WHERE id = @id;

-- name: EarnSubscriptionSetSource :exec
UPDATE paid_subscriptions SET source_kind = @source_kind, source_campaign_id = sqlc.narg(source_campaign_id), source_label = @source_label,
    source_detail = @source_detail, updated_at = @at
WHERE id = @id AND source_kind = '';

-- name: EarnContactHasHadTrial :one
-- Trials are for people who haven't had one before.
SELECT EXISTS (SELECT 1 FROM paid_subscriptions WHERE contact_id = @contact_id AND trial_end IS NOT NULL)::boolean;

-- name: EarnGrantsToEnd :many
-- Complimentary access whose date has passed, and paid access a
-- disconnected account can no longer renew, past its period.
SELECT s.* FROM paid_subscriptions s
WHERE s.ended_at IS NULL AND (
    (s.kind = 'complimentary' AND s.grant_until IS NOT NULL AND s.grant_until <= @now::timestamptz)
 OR (s.kind = 'gift' AND s.current_period_end IS NOT NULL AND s.current_period_end <= @now::timestamptz)
 OR (s.kind = 'stripe' AND s.current_period_end IS NOT NULL AND s.current_period_end <= @grace_before::timestamptz
     AND NOT EXISTS (SELECT 1 FROM payment_connections pc WHERE pc.workspace_id = s.workspace_id AND pc.mode = s.mode AND pc.status = 'connected'))
)
LIMIT 500;

-- name: EarnEventInsert :exec
INSERT INTO paid_events (id, workspace_id, contact_id, subscription_id, mode, kind, occurred_at, amount_cents, currency, payment_event_id, stripe_event,
    message_id, actor_id, data)
VALUES (@id, @workspace_id, @contact_id, sqlc.narg(subscription_id), @mode, @kind, @occurred_at, sqlc.narg(amount_cents), @currency,
    sqlc.narg(payment_event_id), @stripe_event, sqlc.narg(message_id), sqlc.narg(actor_id), @data);

-- name: EarnTimeline :many
-- A paid member's timeline, newest first, with the person who acted.
SELECT e.*, coalesce(u.name, '')::text AS actor_name
FROM paid_events e
LEFT JOIN users u ON u.id = e.actor_id
WHERE e.contact_id = @contact_id AND e.workspace_id = @workspace_id AND (@mode::text = '' OR e.mode = @mode)
ORDER BY e.occurred_at DESC, e.id DESC
LIMIT @lim;

-- name: EarnContactBrief :one
SELECT id, email, first_name, last_name, status, language FROM contacts
WHERE id = @id AND workspace_id = @workspace_id AND erased_at IS NULL;

-- name: EarnContactByEmail :one
SELECT id, email, first_name, last_name, status, language FROM contacts
WHERE workspace_id = @workspace_id AND email = lower(@email) AND erased_at IS NULL;

-- name: EarnClearTestData :execrows
-- Test paid members and their timelines go; the contacts stay.
DELETE FROM paid_subscriptions WHERE workspace_id = @workspace_id AND mode = 'test';

-- name: EarnClearTestEvents :exec
DELETE FROM paid_events WHERE workspace_id = @workspace_id AND mode = 'test';

-- name: EarnClearTestCheckouts :exec
DELETE FROM earn_checkouts WHERE workspace_id = @workspace_id AND mode = 'test';

-- name: EarnErasePaid :exec
-- Erasing a contact removes its paid status and timeline from ReputeCast.
DELETE FROM paid_subscriptions WHERE contact_id = @contact_id;

-- name: EarnEraseTimeline :exec
DELETE FROM paid_events WHERE contact_id = @contact_id;

-- name: EarnSubscriptionSetAccessPaused :exec
UPDATE paid_subscriptions SET access_paused = @access_paused, updated_at = @at WHERE id = @id;

-- name: EarnContactInsert :one
-- A contact a paid flow creates without joining a list (a tier with no
-- lists, or an address suppressed for marketing).
INSERT INTO contacts (id, workspace_id, email, first_name, last_name, mailbox_provider, consent, source, role_address, created_at, updated_at, last_activity_at)
VALUES (@id, @workspace_id, @email, @first_name, @last_name, @mailbox_provider, 'single', @source, @role_address, @at, @at, @at)
ON CONFLICT (workspace_id, email) DO UPDATE SET updated_at = contacts.updated_at
RETURNING id;
