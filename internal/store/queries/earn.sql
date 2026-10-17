-- Earn (internal/earn): settings, tiers and prices, paid status and its
-- timeline, checkouts, the upgrade page's visits and the member emails.

-- name: EarnTeam :many
-- Active members, Admins first then Owners, with their role (who can help).
SELECT u.id AS user_id, u.name, u.email, r.key AS role_key, r.builtin, r.permissions
FROM workspace_members m
JOIN roles r ON r.id = m.role_id
JOIN users u ON u.id = m.user_id AND u.status = 'active'
WHERE m.workspace_id = @workspace_id AND m.status = 'active'
ORDER BY CASE r.key WHEN 'admin' THEN 0 WHEN 'owner' THEN 1 ELSE 2 END, u.name;

-- name: EarnUserName :one
SELECT name FROM users WHERE id = @id;

-- name: EarnSettings :one
SELECT * FROM earn_settings WHERE workspace_id = @workspace_id;

-- name: EarnSettingsEnsure :exec
INSERT INTO earn_settings (workspace_id) VALUES (@workspace_id) ON CONFLICT DO NOTHING;

-- name: EarnSettingsSetCheckout :exec
UPDATE earn_settings SET tax_at_checkout = @tax_at_checkout, prices_include_tax = @prices_include_tax, portal = @portal,
    currency = CASE WHEN currency = '' THEN @currency ELSE currency END, updated_at = @at, updated_by = sqlc.narg(updated_by)
WHERE workspace_id = @workspace_id;

-- name: EarnSettingsSetCurrency :exec
UPDATE earn_settings SET currency = @currency WHERE workspace_id = @workspace_id AND currency = '';

-- name: EarnSettingsSetAccess :exec
UPDATE earn_settings SET on_payment_failure = @on_payment_failure, updated_at = @at, updated_by = sqlc.narg(updated_by)
WHERE workspace_id = @workspace_id;

-- name: EarnSettingsSetPages :exec
UPDATE earn_settings SET terms_url = @terms_url, support_email = @support_email, updated_at = @at, updated_by = sqlc.narg(updated_by)
WHERE workspace_id = @workspace_id;

-- name: EarnSettingsSetUpgradePage :exec
UPDATE earn_settings SET upgrade_page = @upgrade_page, upgrade_published_at = @at, upgrade_published_by = sqlc.narg(by), updated_at = @at
WHERE workspace_id = @workspace_id;

-- name: EarnTiers :many
SELECT * FROM earn_tiers WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY access_rank, created_at;

-- name: EarnTier :one
SELECT * FROM earn_tiers WHERE id = @id AND workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: EarnTierBySlug :one
SELECT * FROM earn_tiers WHERE workspace_id = @workspace_id AND slug = @slug AND deleted_at IS NULL;

-- name: EarnTierSlugTaken :one
SELECT EXISTS (SELECT 1 FROM earn_tiers WHERE workspace_id = @workspace_id AND slug = @slug AND id <> @id AND deleted_at IS NULL)::boolean;

-- name: EarnTierInsert :exec
INSERT INTO earn_tiers (id, workspace_id, name, slug, description, benefits, trial_days, list_ids, access_rank, visibility, origin, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @name, @slug, @description, @benefits, @trial_days, @list_ids, @access_rank, @visibility, @origin, @created_by, @at, @at);

-- name: EarnTierUpdate :exec
UPDATE earn_tiers SET name = @name, slug = @slug, description = @description, benefits = @benefits, trial_days = @trial_days, list_ids = @list_ids,
    access_rank = @access_rank, visibility = @visibility,
    archived_at = CASE WHEN @visibility = 'archived' THEN coalesce(archived_at, @at::timestamptz) ELSE NULL END, updated_at = @at
WHERE id = @id AND workspace_id = @workspace_id;

-- name: EarnTierSetProduct :exec
UPDATE earn_tiers SET stripe_product_live = CASE WHEN @mode = 'live' THEN @product ELSE stripe_product_live END,
    stripe_product_test = CASE WHEN @mode = 'test' THEN @product ELSE stripe_product_test END
WHERE id = @id;

-- name: EarnTierSync :exec
UPDATE earn_tiers SET sync_status = @sync_status, sync_steps = @sync_steps, synced_at = sqlc.narg(synced_at) WHERE id = @id;

-- name: EarnTierDelete :exec
UPDATE earn_tiers SET deleted_at = @at WHERE id = @id AND workspace_id = @workspace_id;

-- name: EarnPrices :many
-- Every price of the workspace's tiers, current first.
SELECT p.* FROM earn_prices p
JOIN earn_tiers t ON t.id = p.tier_id AND t.deleted_at IS NULL
WHERE p.workspace_id = @workspace_id
ORDER BY p.tier_id, (p.status = 'current') DESC, p.period, p.created_at DESC;

-- name: EarnTierPrices :many
SELECT * FROM earn_prices WHERE tier_id = @tier_id ORDER BY (status = 'current') DESC, period, created_at DESC;

-- name: EarnPrice :one
SELECT * FROM earn_prices WHERE id = @id;

-- name: EarnPriceByStripeID :one
-- The price a Stripe price id stands for, in a mode.
SELECT p.* FROM earn_prices p
JOIN earn_tiers t ON t.id = p.tier_id
WHERE p.workspace_id = @workspace_id
  AND ((@mode::text = 'live' AND p.stripe_live_id = @stripe_id) OR (@mode::text = 'test' AND p.stripe_test_id = @stripe_id))
LIMIT 1;

-- name: EarnPriceInsert :exec
INSERT INTO earn_prices (id, workspace_id, tier_id, period, amount_cents, currency, status, stripe_live_id, stripe_test_id, origin, created_at)
VALUES (@id, @workspace_id, @tier_id, @period, @amount_cents, @currency, @status, @stripe_live_id, @stripe_test_id, @origin, @at);

-- name: EarnPriceRetire :exec
UPDATE earn_prices SET status = 'legacy', retired_at = @at WHERE id = @id AND status = 'current';

-- name: EarnPriceSetStripe :exec
UPDATE earn_prices SET stripe_live_id = CASE WHEN @mode = 'live' THEN @stripe_id ELSE stripe_live_id END,
    stripe_test_id = CASE WHEN @mode = 'test' THEN @stripe_id ELSE stripe_test_id END
WHERE id = @id;

-- name: EarnPriceDelete :exec
DELETE FROM earn_prices WHERE id = @id;

-- name: EarnPriceCounts :many
-- Subscriptions per price and period that have access, for tier cards and legacy prices.
SELECT s.price_id, s.period, s.status, count(*)::bigint AS n,
    count(*) FILTER (WHERE s.discount IS NOT NULL)::bigint AS discounted
FROM paid_subscriptions s
WHERE s.workspace_id = @workspace_id AND s.mode = @mode AND s.kind = 'stripe' AND s.ended_at IS NULL AND s.status <> 'incomplete'
GROUP BY s.price_id, s.period, s.status;

-- name: EarnUnmappedPrices :many
SELECT u.*, (SELECT count(*) FROM paid_subscriptions s WHERE s.workspace_id = u.workspace_id AND s.mode = u.mode
    AND s.stripe_price_id = u.stripe_price_id AND s.ended_at IS NULL)::bigint AS subscriptions
FROM earn_unmapped_prices u
WHERE u.workspace_id = @workspace_id AND u.mapped_at IS NULL
ORDER BY u.found_at;

-- name: EarnUnmappedPrice :one
SELECT * FROM earn_unmapped_prices WHERE id = @id AND workspace_id = @workspace_id;

-- name: EarnUnmappedPriceUpsert :one
INSERT INTO earn_unmapped_prices (id, workspace_id, mode, stripe_price_id, period, amount_cents, currency, found_at, found_by)
VALUES (@id, @workspace_id, @mode, @stripe_price_id, @period, @amount_cents, @currency, @found_at, @found_by)
ON CONFLICT (workspace_id, mode, stripe_price_id) DO UPDATE SET period = EXCLUDED.period
RETURNING id, (xmax = 0)::boolean AS inserted;

-- name: EarnUnmappedPriceMapped :exec
UPDATE earn_unmapped_prices SET mapped_at = @at, mapped_price_id = @mapped_price_id, mapped_by = sqlc.narg(mapped_by) WHERE id = @id;

-- name: EarnUnmappedPriceNotified :exec
UPDATE earn_unmapped_prices SET notified_at = @at WHERE id = @id;

-- name: EarnSubscriptionsOnStripePrice :many
SELECT * FROM paid_subscriptions WHERE workspace_id = @workspace_id AND mode = @mode AND stripe_price_id = @stripe_price_id AND ended_at IS NULL
ORDER BY started_at;

-- name: EarnSubscriptionsSetPrice :exec
UPDATE paid_subscriptions SET price_id = @price_id, tier_id = @tier_id, updated_at = @at
WHERE workspace_id = @workspace_id AND mode = @mode AND stripe_price_id = @stripe_price_id;

-- name: EarnMemberEmails :many
SELECT * FROM earn_member_emails WHERE workspace_id = @workspace_id;

-- name: EarnMemberEmailSet :exec
INSERT INTO earn_member_emails (workspace_id, email, template_id) VALUES (@workspace_id, @email, @template_id)
ON CONFLICT (workspace_id, email) DO UPDATE SET template_id = EXCLUDED.template_id;

-- name: EarnPageVisit :exec
INSERT INTO earn_page_visits (workspace_id, day, page, visits) VALUES (@workspace_id, @day, @page, 1)
ON CONFLICT (workspace_id, day, page) DO UPDATE SET visits = earn_page_visits.visits + 1;

-- name: EarnPageVisits :one
SELECT coalesce(sum(visits), 0)::bigint FROM earn_page_visits WHERE workspace_id = @workspace_id AND page = @page AND day >= @since;

-- name: EarnCheckoutCounts :one
-- Live checkouts started and completed since a moment.
SELECT count(*)::bigint AS started, count(*) FILTER (WHERE status = 'completed')::bigint AS completed,
    count(*) FILTER (WHERE status = 'completed' AND trial_days > 0)::bigint AS trials
FROM earn_checkouts WHERE workspace_id = @workspace_id AND mode = 'live' AND created_at >= @since;

-- name: EarnCheckoutInsert :exec
INSERT INTO earn_checkouts (id, workspace_id, mode, tier_id, price_id, period, trial_days, contact_id, email, stripe_session_id, status, attribution, offer, started_by, created_at)
VALUES (@id, @workspace_id, @mode, @tier_id, @price_id, @period, @trial_days, sqlc.narg(contact_id), @email, @stripe_session_id, @status, @attribution, @offer, sqlc.narg(started_by), @created_at);

-- name: EarnCheckoutSetSession :exec
UPDATE earn_checkouts SET stripe_session_id = @stripe_session_id, status = @status WHERE id = @id;

-- name: EarnCheckoutBySession :one
SELECT * FROM earn_checkouts WHERE stripe_session_id = @stripe_session_id;

-- name: EarnCheckout :one
SELECT * FROM earn_checkouts WHERE id = @id;

-- name: EarnCheckoutComplete :exec
UPDATE earn_checkouts SET status = 'completed', completed_at = coalesce(completed_at, @at::timestamptz), subscription_id = coalesce(sqlc.narg(subscription_id), subscription_id),
    contact_id = coalesce(sqlc.narg(contact_id), contact_id), email = CASE WHEN @email::text = '' THEN email ELSE @email END
WHERE id = @id;

-- name: EarnCheckoutCanceled :exec
UPDATE earn_checkouts SET status = 'canceled' WHERE id = @id AND status = 'open';

-- name: EarnListNames :many
-- The names of lists, in the order given.
SELECT l.name FROM unnest(@ids::uuid[]) WITH ORDINALITY AS x(id, n)
JOIN lists l ON l.id = x.id AND l.deleted_at IS NULL
ORDER BY x.n;

-- name: EarnLists :many
-- The workspace's lists, for the tier editor's "Paid members join".
SELECT l.id, l.name,
    (SELECT count(*) FROM list_memberships m WHERE m.list_id = l.id AND m.status = 'subscribed')::bigint AS subscribers
FROM lists l WHERE l.workspace_id = @workspace_id AND l.deleted_at IS NULL ORDER BY l.created_at, l.name;

-- name: EarnCampaignName :one
SELECT name FROM campaigns WHERE id = @id;

-- name: EarnTierSubscriptions :one
-- Whether a tier has ever had paid members (it can't be deleted then).
SELECT count(*)::bigint FROM paid_subscriptions WHERE tier_id = @tier_id;

-- name: EarnSettingsSetUpgradePageOnly :execrows
-- The highlighted tier, saved with the tier editor.
UPDATE earn_settings SET upgrade_page = @upgrade_page WHERE workspace_id = @workspace_id;
