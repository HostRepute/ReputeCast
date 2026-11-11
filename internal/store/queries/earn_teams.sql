-- Teams (M65, G-38): a group checkout's buyer subscription carries a team
-- (earn_groups) whose seats (earn_group_seats) give invited people access
-- through their own paid_subscriptions row (kind and status 'group').

-- name: EarnGroupInsert :one
-- A team, once per buyer subscription.
INSERT INTO earn_groups (id, workspace_id, subscription_id, owner_contact_id, name, seats, created_at, updated_at)
VALUES (@id, @workspace_id, @subscription_id, @owner_contact_id, @name, @seats, @at, @at)
ON CONFLICT (subscription_id) DO UPDATE SET updated_at = earn_groups.updated_at
RETURNING *;

-- name: EarnGroupBySubscription :one
SELECT * FROM earn_groups WHERE subscription_id = @subscription_id;

-- name: EarnGroup :one
SELECT * FROM earn_groups WHERE id = @id AND workspace_id = @workspace_id;

-- name: EarnGroupForUpdate :one
SELECT * FROM earn_groups WHERE id = @id AND workspace_id = @workspace_id FOR UPDATE;

-- name: EarnGroupSetSeats :exec
-- The seats follow the subscription's quantity; the notice of too few seats
-- is sent again only after the team fits again.
UPDATE earn_groups SET seats = @seats, over_notified_at = CASE WHEN @fits::boolean THEN NULL ELSE over_notified_at END, updated_at = @at
WHERE id = @id;

-- name: EarnGroupOverNotified :one
-- Marks that the buyer was told the seats are fewer than the people: once.
UPDATE earn_groups SET over_notified_at = @at::timestamptz WHERE id = @id AND over_notified_at IS NULL RETURNING id;

-- name: EarnGroupSetEnded :exec
UPDATE earn_groups SET ended_at = sqlc.narg(ended_at), updated_at = @at WHERE id = @id AND ended_at IS DISTINCT FROM sqlc.narg(ended_at)::timestamptz;

-- name: EarnGroupPeople :one
-- The people in a team besides the buyer: invited and active seats.
SELECT count(*) FILTER (WHERE status IN ('invited','active'))::bigint AS people,
       count(*) FILTER (WHERE status = 'active')::bigint AS active,
       count(*) FILTER (WHERE status = 'invited')::bigint AS invited
FROM earn_group_seats WHERE group_id = @group_id;

-- name: EarnGroupSeats :many
-- A team's seats with their contact's name: active first, then invited,
-- newest first.
SELECT g.*, coalesce(c.first_name, '')::text AS first_name, coalesce(c.last_name, '')::text AS last_name
FROM earn_group_seats g LEFT JOIN contacts c ON c.id = g.contact_id
WHERE g.group_id = @group_id AND g.status <> 'removed'
ORDER BY (g.status = 'active') DESC, g.invited_at DESC, g.id;

-- name: EarnGroupSeat :one
SELECT * FROM earn_group_seats WHERE id = @id AND group_id = @group_id;

-- name: EarnGroupSeatForUpdate :one
SELECT * FROM earn_group_seats WHERE id = @id AND group_id = @group_id FOR UPDATE;

-- name: EarnGroupSeatByInvite :one
SELECT * FROM earn_group_seats WHERE invite_hash = @invite_hash;

-- name: EarnGroupSeatInsert :exec
INSERT INTO earn_group_seats (id, group_id, workspace_id, email, status, invite_hash, invited_at, invite_sent_at, invite_expires_at)
VALUES (@id, @group_id, @workspace_id, @email, 'invited', @invite_hash, @at, @at, @expires_at);

-- name: EarnGroupSeatReinvite :exec
-- A new invite link: the older one stops working.
UPDATE earn_group_seats SET invite_hash = @invite_hash, invite_sent_at = @at, invite_expires_at = @expires_at
WHERE id = @id AND status = 'invited';

-- name: EarnGroupSeatAccept :exec
UPDATE earn_group_seats SET status = 'active', contact_id = @contact_id, subscription_id = @subscription_id, invite_hash = NULL, accepted_at = @at::timestamptz
WHERE id = @id AND status = 'invited';

-- name: EarnGroupSeatRemove :exec
UPDATE earn_group_seats SET status = 'removed', invite_hash = NULL, removed_at = @at::timestamptz WHERE id = @id AND status <> 'removed';

-- name: EarnGroupSeatEndAccess :one
-- A removed seat's access ends now.
UPDATE paid_subscriptions SET status = 'canceled', ended_at = @at::timestamptz, end_reason = @end_reason, updated_at = @at
WHERE id = @id AND kind = 'group' AND ended_at IS NULL
RETURNING id;

-- name: EarnGroupSyncSeats :many
-- A team's seats follow its buyer: tier, the access rule's pause, the
-- pause and the end (status canceled, reason group_ended). Returns the
-- seats changed, and whether their access ended.
UPDATE paid_subscriptions s SET tier_id = b.tier_id, access_paused = b.access_paused, paused_at = b.paused_at, pause_resumes_at = b.pause_resumes_at,
    status = CASE WHEN b.ended_at IS NOT NULL THEN 'canceled' ELSE s.status END,
    ended_at = b.ended_at,
    end_reason = CASE WHEN b.ended_at IS NOT NULL THEN 'group_ended' ELSE s.end_reason END,
    updated_at = @at
FROM earn_groups g JOIN paid_subscriptions b ON b.id = g.subscription_id
WHERE g.id = @group_id AND s.group_id = g.id AND s.kind = 'group' AND s.ended_at IS NULL
  AND (s.tier_id IS DISTINCT FROM b.tier_id OR s.access_paused <> b.access_paused OR s.paused_at IS DISTINCT FROM b.paused_at
       OR s.pause_resumes_at IS DISTINCT FROM b.pause_resumes_at OR b.ended_at IS NOT NULL)
RETURNING s.id, s.contact_id, (s.ended_at IS NOT NULL)::boolean AS ended;

-- name: EarnGroupsOfOwner :many
-- The teams a contact bought that haven't ended (the reader's account page).
SELECT g.*, coalesce(t.name, '')::text AS tier_name, s.mode
FROM earn_groups g JOIN paid_subscriptions s ON s.id = g.subscription_id LEFT JOIN earn_tiers t ON t.id = s.tier_id
WHERE g.workspace_id = @workspace_id AND g.owner_contact_id = @contact_id AND g.ended_at IS NULL
ORDER BY g.created_at DESC;

-- name: EarnGroupsOfSubscriptions :many
-- The teams of some subscriptions, as buyer or seat (the members list), with
-- the people in each.
SELECT g.id, g.subscription_id, g.name, g.seats, g.ended_at,
       (SELECT count(*) FROM earn_group_seats x WHERE x.group_id = g.id AND x.status IN ('invited','active'))::bigint AS people
FROM earn_groups g
WHERE g.subscription_id = ANY (@ids::uuid[])
   OR g.id IN (SELECT s.group_id FROM paid_subscriptions s WHERE s.id = ANY (@ids::uuid[]) AND s.group_id IS NOT NULL);

-- name: EarnGroupSeatOfSubscription :one
-- The seat a seat holder's subscription belongs to.
SELECT * FROM earn_group_seats WHERE subscription_id = @subscription_id;

-- name: EarnSubscriptionSetQuantity :exec
-- A team's seats changed here; Stripe's event that follows finds them in place.
UPDATE paid_subscriptions SET quantity = @quantity, updated_at = @at WHERE id = @id;
