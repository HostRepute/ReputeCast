-- Offers (X17): signup offers by code or link, the retention offer of the
-- cancel page and referral rewards, with their coupons in the provider.

-- name: OfferList :many
SELECT o.*, coalesce(u.name, '')::text AS created_by_name, coalesce(eb.name, '')::text AS ended_by_name
FROM earn_offers o
LEFT JOIN users u ON u.id = o.created_by
LEFT JOIN users eb ON eb.id = o.ended_by
WHERE o.workspace_id = @workspace_id AND o.deleted_at IS NULL
ORDER BY CASE o.kind WHEN 'signup' THEN 0 WHEN 'retention' THEN 1 ELSE 2 END, o.started_at, o.created_at;

-- name: OfferGet :one
SELECT o.*, coalesce(u.name, '')::text AS created_by_name, coalesce(eb.name, '')::text AS ended_by_name
FROM earn_offers o
LEFT JOIN users u ON u.id = o.created_by
LEFT JOIN users eb ON eb.id = o.ended_by
WHERE o.id = @id AND o.workspace_id = @workspace_id AND o.deleted_at IS NULL;

-- name: OfferByCode :one
-- A live signup offer by its code or link (case-insensitive).
SELECT * FROM earn_offers
WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND kind = 'signup' AND upper(code) = upper(@code::text)
ORDER BY (status = 'live') DESC, started_at DESC LIMIT 1;

-- name: OfferCodeTaken :one
SELECT EXISTS (
  SELECT 1 FROM earn_offers WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND code <> '' AND upper(code) = upper(@code::text) AND id <> @id
)::boolean;

-- name: OfferInsert :exec
INSERT INTO earn_offers (id, workspace_id, name, kind, discount_type, percent_off, amount_off, currency, duration, duration_months, applies,
  by_code, code, by_link, redeem_limit, ends_at, new_only, min_paid_months, code_days, status, sync_status, started_at, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @name, @kind, @discount_type, sqlc.narg(percent_off), sqlc.narg(amount_off), @currency, @duration, sqlc.narg(duration_months), @applies,
  @by_code, @code, @by_link, sqlc.narg(redeem_limit), sqlc.narg(ends_at), @new_only, @min_paid_months, @code_days, 'live', 'draft', @at, sqlc.narg(created_by), @at, @at);

-- name: OfferUpdate :exec
UPDATE earn_offers SET name = @name, discount_type = @discount_type, percent_off = sqlc.narg(percent_off), amount_off = sqlc.narg(amount_off),
  currency = @currency, duration = @duration, duration_months = sqlc.narg(duration_months), applies = @applies, by_code = @by_code, code = @code,
  by_link = @by_link, redeem_limit = sqlc.narg(redeem_limit), ends_at = sqlc.narg(ends_at), new_only = @new_only, min_paid_months = @min_paid_months,
  code_days = @code_days, updated_at = @at
WHERE id = @id AND workspace_id = @workspace_id;

-- name: OfferSetStripe :exec
UPDATE earn_offers SET stripe = @stripe, sync_status = @sync_status, sync_error = @sync_error, updated_at = @at WHERE id = @id;

-- name: OfferEnd :exec
UPDATE earn_offers SET status = 'ended', ended_at = @at::timestamptz, ended_reason = @reason, ended_by = sqlc.narg(ended_by), updated_at = @at::timestamptz
WHERE id = @id AND status = 'live';

-- name: OfferReopen :exec
-- Raising the limit or moving the date of an offer that ended by them
-- brings it back.
UPDATE earn_offers SET status = 'live', ended_at = NULL, ended_reason = '', ended_by = NULL, updated_at = @at WHERE id = @id;

-- name: OfferRedemptions :one
-- Paid members who took a signup offer: by its coupons, or the code it
-- carries; new ones started at or after the offer; and the MRR they bring
-- after the discount.
SELECT count(*)::bigint AS redemptions,
       count(*) FILTER (WHERE s.started_at >= @started_at)::bigint AS new_members,
       coalesce(sum(earn_arr_cents(s.kind, s.mode, s.status, s.ended_at, s.period, s.amount_cents, s.discount)), 0)::bigint AS arr
FROM paid_subscriptions s
WHERE s.workspace_id = @workspace_id AND s.mode = 'live' AND s.discount IS NOT NULL
  AND (s.discount->>'coupon' = ANY(@coupons::text[]) OR (@code::text <> '' AND upper(s.discount->>'label') = upper(@code::text)));

-- name: OfferRetentionResults :one
-- The retention offer on the cancel page: shown, and kept with it.
SELECT count(*)::bigint AS shown, count(*) FILTER (WHERE decision = 'kept')::bigint AS accepted
FROM earn_cancel_visits WHERE workspace_id = @workspace_id AND offer_id = @offer_id AND mode = 'live';

-- name: OfferCodes :many
SELECT c.*, coalesce(nullif(trim(ct.first_name || ' ' || ct.last_name), ''), ct.email, c.issued_to, '')::text AS issued_name
FROM earn_offer_codes c LEFT JOIN contacts ct ON ct.id = c.contact_id
WHERE c.offer_id = @offer_id ORDER BY c.issued_at;

-- name: OfferCodeInsert :exec
INSERT INTO earn_offer_codes (id, workspace_id, offer_id, code, contact_id, issued_to, issued_at, expires_at, stripe, source)
VALUES (@id, @workspace_id, @offer_id, @code, sqlc.narg(contact_id), @issued_to, @issued_at, @expires_at, @stripe, @source);

-- name: OffersLive :many
-- Live offers, for the job that ends them at their date or limit.
SELECT * FROM earn_offers WHERE status = 'live' AND deleted_at IS NULL AND kind = 'signup' AND (ends_at IS NOT NULL OR redeem_limit IS NOT NULL);

-- name: OfferRetentionOf :one
-- The workspace's live retention offer (one at most).
SELECT * FROM earn_offers WHERE workspace_id = @workspace_id AND kind = 'retention' AND status = 'live' AND deleted_at IS NULL
ORDER BY started_at DESC LIMIT 1;

-- name: OfferRetentionUsedSince :one
-- Whether a paid member took a retention offer since a moment (one in 12
-- months).
SELECT EXISTS (
  SELECT 1 FROM earn_offer_uses u JOIN earn_offers o ON o.id = u.offer_id
  WHERE u.contact_id = @contact_id AND o.kind = 'retention' AND u.used_at >= @since
)::boolean;

-- name: OfferUseInsert :exec
INSERT INTO earn_offer_uses (id, workspace_id, offer_id, contact_id, subscription_id, mode, used_at, source)
VALUES (@id, @workspace_id, @offer_id, @contact_id, sqlc.narg(subscription_id), @mode, @used_at, @source);

-- name: OfferTierPrices :many
-- The tiers and their current prices, for an offer's lines.
SELECT t.id AS tier_id, t.name AS tier, t.stripe_product_live, t.stripe_product_test, p.period, p.amount_cents, p.currency
FROM earn_tiers t JOIN earn_prices p ON p.tier_id = t.id AND p.status = 'current'
WHERE t.workspace_id = @workspace_id AND t.deleted_at IS NULL
ORDER BY t.access_rank, t.name, p.period;

-- name: CancelVisitRecent :one
-- A reader's undecided visit of the last half hour: a reload counts once.
SELECT * FROM earn_cancel_visits
WHERE workspace_id = @workspace_id AND contact_id = @contact_id AND decision = '' AND visited_at >= @since
ORDER BY visited_at DESC LIMIT 1;

-- name: CancelVisitInsert :exec
INSERT INTO earn_cancel_visits (id, workspace_id, contact_id, subscription_id, mode, visited_at, offer_id)
VALUES (@id, @workspace_id, @contact_id, @subscription_id, @mode, @visited_at, sqlc.narg(offer_id));

-- name: CancelVisitGet :one
SELECT * FROM earn_cancel_visits WHERE id = @id AND workspace_id = @workspace_id AND contact_id = @contact_id;

-- name: CancelVisitDecide :exec
UPDATE earn_cancel_visits SET reason = @reason, decision = @decision, decided_at = @decided_at WHERE id = @id;

-- name: CancelSetReason :exec
-- The reason given on the cancel page, for the paid member's timeline.
UPDATE paid_subscriptions SET cancel_reason = @reason, updated_at = @at WHERE id = @id;

-- name: OfferCodeByCode :one
-- A referral reward's single-use code, by the code a reader brings.
SELECT * FROM earn_offer_codes WHERE workspace_id = @workspace_id AND upper(code) = upper(@code::text);

-- name: OfferCodeRedeem :execrows
-- A subscription took a single-use code.
UPDATE earn_offer_codes SET redeemed_at = @at::timestamptz WHERE workspace_id = @workspace_id AND upper(code) = upper(@code::text) AND redeemed_at IS NULL;

-- name: OfferReferralOffers :many
-- Live "Referral reward" offers with the codes they issued and those
-- redeemed, for a milestone's coupon reward.
SELECT o.id, o.name, o.applies, o.discount_type, o.percent_off, o.amount_off, o.currency, o.duration, o.duration_months,
       (SELECT count(*) FROM earn_offer_codes c WHERE c.offer_id = o.id)::bigint AS issued,
       (SELECT count(*) FROM earn_offer_codes c WHERE c.offer_id = o.id AND c.redeemed_at IS NOT NULL)::bigint AS redeemed
FROM earn_offers o
WHERE o.workspace_id = @workspace_id AND o.kind = 'referral' AND o.status = 'live' AND o.deleted_at IS NULL
ORDER BY o.started_at, o.name;
