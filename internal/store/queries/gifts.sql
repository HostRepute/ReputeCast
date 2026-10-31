-- Gifts (X17): gift options, the gift page's settings, gifts bought on the
-- gift page, their delivery, redemption and reminders.

-- name: GiftOptions :many
SELECT o.id, o.tier_id, o.months, o.amount_cents, o.currency, o.position, o.created_at,
       t.name AS tier_name, t.list_ids, t.benefits, t.description AS tier_description
FROM earn_gift_options o JOIN earn_tiers t ON t.id = o.tier_id
WHERE o.workspace_id = @workspace_id AND o.deleted_at IS NULL AND t.deleted_at IS NULL
ORDER BY o.position, o.months, o.created_at;

-- name: GiftOptionGet :one
SELECT * FROM earn_gift_options WHERE id = @id AND workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: GiftOptionTaken :one
-- Another option of the same tier and period.
SELECT EXISTS (
  SELECT 1 FROM earn_gift_options WHERE workspace_id = @workspace_id AND tier_id = @tier_id AND months = @months AND deleted_at IS NULL AND id <> @id
)::boolean;

-- name: GiftOptionInsert :exec
INSERT INTO earn_gift_options (id, workspace_id, tier_id, months, amount_cents, currency, position, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @tier_id, @months, @amount_cents, @currency,
  coalesce((SELECT max(position) + 1 FROM earn_gift_options WHERE workspace_id = @workspace_id AND deleted_at IS NULL), 0), sqlc.narg(created_by), @at::timestamptz, @at::timestamptz);

-- name: GiftOptionUpdate :exec
UPDATE earn_gift_options SET tier_id = @tier_id, months = @months, amount_cents = @amount_cents, currency = @currency, updated_at = @at::timestamptz
WHERE id = @id AND workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: GiftOptionDelete :exec
UPDATE earn_gift_options SET deleted_at = @at::timestamptz WHERE id = @id AND workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: GiftOptionSales :many
-- Each option's gifts paid since a time, refunded ones left out.
SELECT g.option_id, count(*)::bigint AS sold, coalesce(sum(g.amount_cents), 0)::bigint AS cents
FROM earn_gifts g
WHERE g.workspace_id = @workspace_id AND g.mode = @mode AND g.paid_at >= @since::timestamptz AND g.status IN ('paid','delivered','redeemed','expired')
GROUP BY g.option_id;

-- name: GiftSoldByMonths :many
-- Gifts paid since a time by their period ("7 for a year, 4 for 3 months").
SELECT g.months, count(*)::bigint AS sold, coalesce(sum(g.amount_cents), 0)::bigint AS cents
FROM earn_gifts g
WHERE g.workspace_id = @workspace_id AND g.mode = @mode AND g.paid_at >= @since::timestamptz AND g.status IN ('paid','delivered','redeemed','expired')
GROUP BY g.months ORDER BY count(*) DESC, g.months DESC;

-- name: GiftNumbers :one
-- The Gifts tab's numbers: of the gifts paid since a time, those delivered
-- and redeemed; the gifts waiting for their date, and the next one.
SELECT count(*) FILTER (WHERE g.paid_at >= @since::timestamptz AND g.status IN ('delivered','redeemed','expired'))::bigint AS delivered,
       count(*) FILTER (WHERE g.paid_at >= @since::timestamptz AND g.status = 'redeemed')::bigint AS redeemed,
       count(*) FILTER (WHERE g.paid_at >= @since::timestamptz AND g.status = 'delivered')::bigint AS waiting,
       count(*) FILTER (WHERE g.status = 'paid')::bigint AS scheduled
FROM earn_gifts g
WHERE g.workspace_id = @workspace_id AND g.mode = @mode;

-- name: GiftNextScheduled :one
SELECT g.* FROM earn_gifts g
WHERE g.workspace_id = @workspace_id AND g.mode = @mode AND g.status = 'paid'
ORDER BY g.deliver_at NULLS FIRST, g.paid_at LIMIT 1;

-- name: GiftAccessNow :one
-- Paid members with gift access now.
SELECT count(*)::bigint FROM paid_subscriptions
WHERE workspace_id = @workspace_id AND mode = @mode AND kind = 'gift' AND ended_at IS NULL;

-- name: GiftCounts :one
-- The Gifts table's filters: how many gifts each shows.
SELECT count(*)::bigint AS all_gifts,
       count(*) FILTER (WHERE g.status = 'paid')::bigint AS scheduled,
       count(*) FILTER (WHERE g.status = 'delivered')::bigint AS not_redeemed,
       count(*) FILTER (WHERE g.status = 'redeemed')::bigint AS redeemed,
       count(*) FILTER (WHERE g.status = 'expired')::bigint AS expired,
       count(*) FILTER (WHERE g.status = 'refunded')::bigint AS refunded
FROM earn_gifts g
WHERE g.workspace_id = @workspace_id AND g.mode = @mode AND g.status IN ('paid','delivered','redeemed','expired','refunded')
  AND (@search::text = '' OR g.buyer_name ILIKE '%' || @search::text || '%' OR g.buyer_email ILIKE '%' || @search::text || '%'
    OR g.recipient_name ILIKE '%' || @search::text || '%' OR g.recipient_email ILIKE '%' || @search::text || '%' OR g.redeemed_email ILIKE '%' || @search::text || '%');

-- name: GiftList :many
-- A page of the Gifts table, the latest bought first.
SELECT sqlc.embed(g), coalesce(t.name, '')::text AS tier_name, coalesce(t.list_ids, '{}')::uuid[] AS list_ids,
       s.current_period_end AS access_ends
FROM earn_gifts g
LEFT JOIN earn_tiers t ON t.id = g.tier_id
LEFT JOIN paid_subscriptions s ON s.id = g.subscription_id
WHERE g.workspace_id = @workspace_id AND g.mode = @mode AND g.status IN ('paid','delivered','redeemed','expired','refunded')
  AND (@status::text = '' OR g.status = @status::text)
  AND (@search::text = '' OR g.buyer_name ILIKE '%' || @search::text || '%' OR g.buyer_email ILIKE '%' || @search::text || '%'
    OR g.recipient_name ILIKE '%' || @search::text || '%' OR g.recipient_email ILIKE '%' || @search::text || '%' OR g.redeemed_email ILIKE '%' || @search::text || '%')
ORDER BY g.paid_at DESC NULLS LAST, g.created_at DESC
LIMIT @lim OFFSET @off;

-- name: GiftNeedsLook :many
-- Gifts that need a look: delivered a week ago and not redeemed, or
-- refunded in the last 30 days.
SELECT sqlc.embed(g), coalesce(t.name, '')::text AS tier_name, coalesce(t.list_ids, '{}')::uuid[] AS list_ids
FROM earn_gifts g LEFT JOIN earn_tiers t ON t.id = g.tier_id
WHERE g.workspace_id = @workspace_id AND g.mode = @mode
  AND ((g.status = 'delivered' AND g.delivered_at <= @week_ago::timestamptz) OR (g.status = 'refunded' AND g.refunded_at >= @month_ago::timestamptz))
ORDER BY coalesce(g.refunded_at, g.delivered_at) DESC LIMIT 10;

-- name: GiftGet :one
SELECT sqlc.embed(g), coalesce(t.name, '')::text AS tier_name, coalesce(t.list_ids, '{}')::uuid[] AS list_ids,
       s.current_period_end AS access_ends
FROM earn_gifts g
LEFT JOIN earn_tiers t ON t.id = g.tier_id
LEFT JOIN paid_subscriptions s ON s.id = g.subscription_id
WHERE g.id = @id AND g.workspace_id = @workspace_id;

-- name: GiftForUpdate :one
SELECT * FROM earn_gifts WHERE id = @id FOR UPDATE;

-- name: GiftByCode :one
SELECT sqlc.embed(g), coalesce(t.name, '')::text AS tier_name, coalesce(t.list_ids, '{}')::uuid[] AS list_ids,
       coalesce(t.benefits, '{}')::text[] AS benefits, s.current_period_end AS access_ends
FROM earn_gifts g
LEFT JOIN earn_tiers t ON t.id = g.tier_id
LEFT JOIN paid_subscriptions s ON s.id = g.subscription_id
WHERE g.workspace_id = @workspace_id AND g.code = @code;

-- name: GiftBySession :one
SELECT sqlc.embed(g), coalesce(t.name, '')::text AS tier_name, coalesce(t.list_ids, '{}')::uuid[] AS list_ids
FROM earn_gifts g LEFT JOIN earn_tiers t ON t.id = g.tier_id
WHERE g.workspace_id = @workspace_id AND g.stripe_session_id = @session AND g.stripe_session_id <> '';

-- name: GiftBySessionForUpdate :one
SELECT * FROM earn_gifts WHERE workspace_id = @workspace_id AND mode = @mode AND stripe_session_id = @session AND stripe_session_id <> '' FOR UPDATE;

-- name: GiftByIntentForUpdate :one
SELECT * FROM earn_gifts WHERE workspace_id = @workspace_id AND mode = @mode AND payment_intent = @payment_intent AND payment_intent <> '' FOR UPDATE;

-- name: GiftCodeTaken :one
SELECT EXISTS (SELECT 1 FROM earn_gifts WHERE code = @code)::boolean;

-- name: GiftInsert :exec
INSERT INTO earn_gifts (id, workspace_id, mode, option_id, tier_id, months, amount_cents, currency, status, buyer_name, buyer_email, in_person,
  recipient_name, recipient_email, message, deliver_at, timezone, code, created_at, updated_at)
VALUES (@id, @workspace_id, @mode, sqlc.narg(option_id), sqlc.narg(tier_id), @months, @amount_cents, @currency, 'checkout', @buyer_name, @buyer_email, @in_person,
  @recipient_name, @recipient_email, @message, sqlc.narg(deliver_at), @timezone, @code, @at::timestamptz, @at::timestamptz);

-- name: GiftSetSession :exec
UPDATE earn_gifts SET stripe_session_id = @session, updated_at = @at::timestamptz WHERE id = @id;

-- name: GiftPaid :exec
-- The payment went through: the gift waits for its delivery date.
UPDATE earn_gifts SET status = 'paid', paid_at = @at::timestamptz, payment_intent = @payment_intent, stripe_customer = @stripe_customer,
  buyer_email = CASE WHEN buyer_email = '' THEN @buyer_email ELSE buyer_email END, updated_at = @at::timestamptz
WHERE id = @id AND status IN ('checkout','abandoned');

-- name: GiftDelivered :exec
UPDATE earn_gifts SET status = 'delivered', delivered_at = @at::timestamptz, link_expires_at = @link_expires_at::timestamptz, email_status = @email_status, updated_at = @at::timestamptz
WHERE id = @id AND status = 'paid';

-- name: GiftEmailed :exec
-- "Your gift" went again (a reminder, or resent by the team).
UPDATE earn_gifts SET email_status = @email_status, reminded_at = coalesce(sqlc.narg(reminded_at), reminded_at), updated_at = @at::timestamptz WHERE id = @id;

-- name: GiftReceiptSent :exec
UPDATE earn_gifts SET receipt_sent_at = @at::timestamptz, updated_at = @at::timestamptz WHERE id = @id;

-- name: GiftRedeemed :exec
UPDATE earn_gifts SET status = 'redeemed', redeemed_at = @at::timestamptz, redeemed_email = @email, redeemed_contact = @contact_id, subscription_id = @subscription_id,
  -- A gift redeemed before its date isn't delivered again.
  delivered_at = coalesce(delivered_at, @at::timestamptz), updated_at = @at::timestamptz
WHERE id = @id AND status IN ('paid','delivered');

-- name: GiftRefunded :exec
UPDATE earn_gifts SET status = 'refunded', refunded_at = @at::timestamptz, updated_at = @at::timestamptz WHERE id = @id AND status <> 'refunded';

-- name: GiftSetDelivery :exec
UPDATE earn_gifts SET deliver_at = @deliver_at::timestamptz, timezone = @timezone, updated_at = @at::timestamptz WHERE id = @id AND status = 'paid';

-- name: GiftEventInsert :exec
INSERT INTO earn_gift_events (id, gift_id, kind, occurred_at, actor_id, message_id, data)
VALUES (@id, @gift_id, @kind, @at::timestamptz, sqlc.narg(actor_id), sqlc.narg(message_id), @data);

-- name: GiftsToReceipt :many
-- Paid gifts whose receipt hasn't gone yet.
SELECT * FROM earn_gifts WHERE status IN ('paid','delivered','redeemed') AND receipt_sent_at IS NULL AND paid_at IS NOT NULL
  AND paid_at > @since::timestamptz
ORDER BY paid_at LIMIT 200;

-- name: GiftsDue :many
-- Paid gifts whose delivery time has come.
-- A gift whose buyer and recipient were both erased has no one to go to.
SELECT * FROM earn_gifts WHERE status = 'paid' AND (deliver_at IS NULL OR deliver_at <= @now::timestamptz)
  AND (recipient_email <> '' OR buyer_email <> '')
ORDER BY deliver_at NULLS FIRST LIMIT 200;

-- name: GiftsToRemind :many
-- Delivered a week ago, not redeemed, not reminded yet, the link still good.
SELECT * FROM earn_gifts WHERE status = 'delivered' AND reminded_at IS NULL AND delivered_at <= @week_ago::timestamptz
  AND (link_expires_at IS NULL OR link_expires_at > @now::timestamptz)
  AND (recipient_email <> '' OR buyer_email <> '')
ORDER BY delivered_at LIMIT 200;

-- name: GiftsToExpire :many
SELECT * FROM earn_gifts WHERE status = 'delivered' AND link_expires_at <= @now::timestamptz LIMIT 500;

-- name: GiftExpired :exec
UPDATE earn_gifts SET status = 'expired', updated_at = @at::timestamptz WHERE id = @id AND status = 'delivered';

-- name: GiftsEnding :many
-- Redeemed gifts whose access ends within a week, not told yet.
SELECT sqlc.embed(g), s.current_period_end AS access_ends FROM earn_gifts g JOIN paid_subscriptions s ON s.id = g.subscription_id
WHERE g.status = 'redeemed' AND g.ending_sent_at IS NULL AND s.ended_at IS NULL AND s.kind = 'gift'
  AND s.current_period_end > @now::timestamptz AND s.current_period_end <= @week_ahead::timestamptz
ORDER BY s.current_period_end LIMIT 200;

-- name: GiftEndingSent :exec
UPDATE earn_gifts SET ending_sent_at = @at::timestamptz, updated_at = @at::timestamptz WHERE id = @id;

-- name: GiftsPurgeMessages :execrows
-- Gift messages are kept until the gift ends, plus 30 days; a checkout
-- that was never paid loses its message once abandoned.
UPDATE earn_gifts g SET message = '', message_purged_at = @now::timestamptz, updated_at = @now::timestamptz
WHERE g.message <> '' AND (
     g.status = 'abandoned'
  OR (g.status = 'refunded' AND g.refunded_at <= @month_ago::timestamptz)
  OR (g.status = 'expired' AND g.link_expires_at <= @month_ago::timestamptz)
  OR (g.status = 'redeemed' AND EXISTS (SELECT 1 FROM paid_subscriptions s WHERE s.id = g.subscription_id
        AND coalesce(s.ended_at, s.current_period_end) <= @month_ago::timestamptz))
);

-- name: GiftsAbandon :execrows
-- Checkouts never paid within a day.
UPDATE earn_gifts SET status = 'abandoned', updated_at = @now::timestamptz WHERE status = 'checkout' AND created_at <= @day_ago::timestamptz;

-- name: GiftFirstEnding :one
-- The first gift-ending reminder due: the redeemed gift whose access ends
-- first, and who gave it.
SELECT g.buyer_name, g.in_person, g.recipient_name, g.redeemed_email, s.current_period_end AS access_ends
FROM earn_gifts g JOIN paid_subscriptions s ON s.id = g.subscription_id
WHERE g.workspace_id = @workspace_id AND g.mode = @mode AND g.status = 'redeemed' AND g.ending_sent_at IS NULL AND s.ended_at IS NULL
  AND s.current_period_end > @now::timestamptz
ORDER BY s.current_period_end LIMIT 1;

-- name: GiftPageSet :exec
UPDATE earn_settings SET gift_page_on = @gift_page_on, gift_page_at = @at::timestamptz, gift_page_by = sqlc.narg(by), gift_link_months = @gift_link_months
WHERE workspace_id = @workspace_id;

-- name: GiftPageByName :one
SELECT coalesce(u.name, '')::text FROM earn_settings st LEFT JOIN users u ON u.id = st.gift_page_by WHERE st.workspace_id = @workspace_id;
