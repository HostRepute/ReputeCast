-- Paid access for the domains that choose recipients and content (design
-- expansion X16): paid-only campaigns, paid-only blocks, paid tokens and
-- the contacts table. A contact has paid access while a live subscription,
-- complimentary grant or gift hasn't ended and is trialing, active,
-- complimentary or a gift, or past due while Stripe retries (unless the
-- access rule paused it). Test paid members never count.

-- name: PaidAccessRows :many
-- One row per contact with paid access at a tier ranked at least min_rank
-- (1: any paid tier): the highest tier's subscription.
SELECT DISTINCT ON (s.contact_id) s.contact_id, s.status, s.kind, coalesce(t.access_rank, 1)::int AS tier_rank
FROM paid_subscriptions s
LEFT JOIN earn_tiers t ON t.id = s.tier_id
WHERE s.workspace_id = @workspace_id AND s.mode = 'live' AND s.ended_at IS NULL
  AND (s.status IN ('trialing', 'active', 'complimentary', 'gift') OR (s.status = 'past_due' AND NOT s.access_paused))
  AND coalesce(t.access_rank, 1) >= @min_rank::int
ORDER BY s.contact_id, coalesce(t.access_rank, 1) DESC, s.started_at DESC;

-- name: PaidAccessFacts :many
-- Some contacts' paid access, the highest tier first: one row per contact
-- with access, with what personalization and conditions read.
SELECT DISTINCT ON (s.contact_id) s.contact_id, s.status, s.kind, s.period, s.amount_cents, s.currency, s.discount,
       s.started_at, s.trial_end, s.current_period_end, s.cancel_at, s.grant_until, s.tier_id,
       coalesce(t.name, '')::text AS tier_name, coalesce(t.access_rank, 1)::int AS tier_rank,
       coalesce(t.list_ids, '{}')::uuid[] AS tier_lists
FROM paid_subscriptions s
LEFT JOIN earn_tiers t ON t.id = s.tier_id
WHERE s.workspace_id = @workspace_id AND s.contact_id = ANY (@contact_ids::uuid[]) AND s.mode = 'live' AND s.ended_at IS NULL
  AND (s.status IN ('trialing', 'active', 'complimentary', 'gift') OR (s.status = 'past_due' AND NOT s.access_paused))
ORDER BY s.contact_id, coalesce(t.access_rank, 1) DESC, s.started_at DESC;

-- name: PaidAccessTiers :many
-- The workspace's tiers in access order, for the pickers ("Insider and
-- above"), with the names of their lists.
SELECT t.id, t.name, t.access_rank, t.list_ids,
       coalesce((SELECT array_agg(l.name ORDER BY l.created_at) FROM lists l WHERE l.id = ANY (t.list_ids)), '{}')::text[] AS list_names
FROM earn_tiers t
WHERE t.workspace_id = @workspace_id AND t.deleted_at IS NULL AND t.visibility <> 'archived'
ORDER BY t.access_rank, t.created_at;

-- name: PaidAccessTierRank :one
SELECT access_rank FROM earn_tiers WHERE id = @id AND workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: PaidAccessListCounts :one
-- The recipients of lists minus excluded lists and suppressions, as the
-- composer counts them, split by paid access (Who on this list): everyone,
-- those with paid access at a tier ranked at least min_rank, and free
-- subscribers (no paid access at all).
WITH members AS (
    SELECT DISTINCT c.id
    FROM contacts c
    JOIN list_memberships m ON m.contact_id = c.id
    WHERE c.workspace_id = @workspace_id AND c.status = 'active' AND c.erased_at IS NULL
      AND m.status = 'subscribed' AND m.list_id = ANY(ARRAY(SELECT ll.id FROM lists ll WHERE ll.id = ANY(@lists::uuid[]) AND ll.deleted_at IS NULL))
      AND NOT EXISTS (SELECT 1 FROM list_memberships x WHERE x.contact_id = c.id AND x.status = 'subscribed' AND x.list_id = ANY(@exclude_lists::uuid[]))
      AND NOT EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = c.workspace_id AND s.email = c.email)
), access AS (
    SELECT s.contact_id, max(coalesce(t.access_rank, 1))::int AS rank
    FROM paid_subscriptions s
    LEFT JOIN earn_tiers t ON t.id = s.tier_id
    WHERE s.workspace_id = @workspace_id AND s.mode = 'live' AND s.ended_at IS NULL
      AND (s.status IN ('trialing', 'active', 'complimentary', 'gift') OR (s.status = 'past_due' AND NOT s.access_paused))
    GROUP BY s.contact_id
)
SELECT count(*)::bigint AS everyone,
       count(*) FILTER (WHERE a.rank >= @min_rank::int)::bigint AS paid,
       count(*) FILTER (WHERE a.contact_id IS NULL)::bigint AS free
FROM members m
LEFT JOIN access a ON a.contact_id = m.id;

-- name: PaidAccessAudience :many
-- Each contact with paid access, the highest tier's subscription, with what
-- decides whether a campaign to a list reaches them: subscribed to it,
-- unsubscribed from it, suppressed.
SELECT DISTINCT ON (s.contact_id) s.contact_id, s.status, coalesce(t.access_rank, 1)::int AS tier_rank,
       EXISTS (SELECT 1 FROM list_memberships m WHERE m.contact_id = s.contact_id AND m.list_id = @list_id AND m.status = 'subscribed')::boolean AS subscribed,
       EXISTS (SELECT 1 FROM list_memberships m WHERE m.contact_id = s.contact_id AND m.list_id = @list_id AND m.status = 'unsubscribed')::boolean AS unsubscribed,
       EXISTS (SELECT 1 FROM suppressions sp WHERE sp.workspace_id = c.workspace_id AND sp.email = c.email)::boolean AS suppressed,
       (c.status = 'active')::boolean AS active
FROM paid_subscriptions s
JOIN contacts c ON c.id = s.contact_id AND c.erased_at IS NULL
LEFT JOIN earn_tiers t ON t.id = s.tier_id
WHERE s.workspace_id = @workspace_id AND s.mode = 'live' AND s.ended_at IS NULL
  AND (s.status IN ('trialing', 'active', 'complimentary', 'gift') OR (s.status = 'past_due' AND NOT s.access_paused))
ORDER BY s.contact_id, coalesce(t.access_rank, 1) DESC, s.started_at DESC;
