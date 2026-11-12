-- Discount code pools (internal/coupons, M59 G-22): pools, their codes
-- (sealed, with a keyed hash) and the batches that upload or create them;
-- giving a code to each message that shows one, and matching store orders.

-- ── Pools ───────────────────────────────────────────────────────────────

-- name: CouponPools :many
-- A workspace's pools with their store (one pool by id, or the pools a
-- template names by slug: {{ coupons.<slug>.code }}).
SELECT p.*, coalesce(st.name, '')::text AS store_name, coalesce(st.platform, '')::text AS store_platform,
       coalesce(st.status, '')::text AS store_status, coalesce(st.domain, '')::text AS store_domain,
       coalesce(st.public_domain, '')::text AS store_public_domain, coalesce(st.currency, '')::text AS store_currency,
       coalesce(u.name, '')::text AS created_by_name
FROM coupon_pools p LEFT JOIN stores st ON st.id = p.store_id LEFT JOIN users u ON u.id = p.created_by
WHERE p.workspace_id = @workspace_id
  AND (sqlc.narg(id)::uuid IS NULL OR p.id = sqlc.narg(id)::uuid)
  AND (sqlc.narg(slugs)::text[] IS NULL OR p.slug = ANY(sqlc.narg(slugs)::text[]))
ORDER BY p.status, p.created_at, p.id;

-- name: CouponPoolsActive :many
-- Every active pool with its free codes, for the hourly refill and the
-- low alert.
SELECT p.*, (SELECT count(*) FROM coupon_codes c WHERE c.pool_id = p.id AND c.assigned_at IS NULL AND c.removed_at IS NULL)::bigint AS free,
       coalesce(st.status, '')::text AS store_status, coalesce(st.name, '')::text AS store_name
FROM coupon_pools p LEFT JOIN stores st ON st.id = p.store_id
WHERE p.status = 'active'
ORDER BY p.workspace_id, p.id;

-- name: CouponPoolInsert :one
INSERT INTO coupon_pools (id, workspace_id, name, slug, source, store_id, settings, prefix, one_per_contact, low_at, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @name, @slug, @source, sqlc.narg(store_id), @settings, @prefix, @one_per_contact, @low_at, sqlc.narg(created_by), @now::timestamptz, @now::timestamptz)
RETURNING id;

-- name: CouponPoolUpdate :exec
UPDATE coupon_pools SET name = @name, settings = @settings, prefix = @prefix, one_per_contact = @one_per_contact, low_at = @low_at,
    notified_low_at = CASE WHEN low_at = @low_at THEN notified_low_at END, updated_at = @now::timestamptz
WHERE id = @id AND workspace_id = @workspace_id;

-- name: CouponPoolArchive :exec
UPDATE coupon_pools SET status = 'archived', updated_at = @now::timestamptz WHERE id = @id AND workspace_id = @workspace_id;

-- name: CouponPoolSetDiscount :exec
-- The store's discount the pool's codes belong to, once created.
UPDATE coupon_pools SET external_discount_id = @external_discount_id, updated_at = @now::timestamptz WHERE id = @id;

-- name: CouponPoolMarkLow :one
-- The low alert goes out once; the row says it went.
UPDATE coupon_pools SET notified_low_at = @at::timestamptz WHERE id = @id AND notified_low_at IS NULL RETURNING id;

-- name: CouponPoolRefillChecked :exec
UPDATE coupon_pools SET refill_checked_at = @at::timestamptz WHERE id = @id;

-- name: CouponPoolClearLow :exec
UPDATE coupon_pools SET notified_low_at = NULL WHERE id = @id AND notified_low_at IS NOT NULL;

-- ── Numbers ─────────────────────────────────────────────────────────────

-- name: CouponPoolCounts :many
SELECT c.pool_id,
       count(*) FILTER (WHERE c.removed_at IS NULL)::bigint AS total,
       count(*) FILTER (WHERE c.assigned_at IS NULL AND c.removed_at IS NULL)::bigint AS free,
       count(*) FILTER (WHERE c.assigned_at IS NOT NULL)::bigint AS sent,
       count(*) FILTER (WHERE c.redeemed_at IS NOT NULL)::bigint AS redeemed,
       count(*) FILTER (WHERE c.removed_at IS NOT NULL AND c.assigned_at IS NULL)::bigint AS removed,
       coalesce(extract(epoch FROM min(c.assigned_at)), 0)::bigint AS first_sent_unix
FROM coupon_codes c
WHERE c.pool_id = ANY(@pool_ids::uuid[])
GROUP BY c.pool_id;

-- name: CouponPoolRevenue :many
-- What orders that used the pools' codes brought in: refunds taken off,
-- cancelled and test orders left out.
SELECT c.pool_id, coalesce(sum(o.total_cents - o.refunded_cents), 0)::bigint AS revenue_cents, coalesce(max(o.currency), '')::text AS currency,
       count(DISTINCT o.id)::bigint AS orders
FROM coupon_codes c JOIN store_orders o ON o.id = c.order_id
WHERE c.pool_id = ANY(@pool_ids::uuid[]) AND NOT o.cancelled AND NOT o.test
GROUP BY c.pool_id;

-- name: CouponPoolFree :one
SELECT count(*)::bigint FROM coupon_codes WHERE pool_id = @pool_id AND assigned_at IS NULL AND removed_at IS NULL;

-- name: CouponPoolHasFree :one
SELECT EXISTS (SELECT 1 FROM coupon_codes WHERE pool_id = @pool_id AND assigned_at IS NULL AND removed_at IS NULL)::boolean;

-- name: CouponContactHasCode :one
SELECT EXISTS (SELECT 1 FROM coupon_codes WHERE pool_id = @pool_id AND contact_id = @contact_id)::boolean;

-- name: CouponAssignedSince :one
SELECT count(*)::bigint FROM coupon_codes WHERE pool_id = @pool_id AND assigned_at >= @since::timestamptz;

-- ── Batches ─────────────────────────────────────────────────────────────

-- name: CouponBatchInsert :one
INSERT INTO coupon_batches (id, workspace_id, pool_id, kind, reason, automation_id, requested, created, failed, status, created_by, created_at, finished_at)
VALUES (@id, @workspace_id, @pool_id, @kind, @reason, sqlc.narg(automation_id), @requested, @created, @failed, @status, sqlc.narg(created_by), @now::timestamptz,
        sqlc.narg(finished_at))
RETURNING id;

-- name: CouponBatches :many
-- A pool's batches, newest first, with who asked or the flow refilled for.
SELECT b.*, coalesce(u.name, '')::text AS created_by_name, coalesce(a.name, '')::text AS automation_name
FROM coupon_batches b LEFT JOIN users u ON u.id = b.created_by LEFT JOIN automations a ON a.id = b.automation_id
WHERE b.pool_id = @pool_id
ORDER BY b.created_at DESC, b.id DESC
LIMIT 50;

-- name: CouponPoolHasBatch :one
SELECT EXISTS (SELECT 1 FROM coupon_batches WHERE pool_id = @pool_id AND kind = 'generate')::boolean;

-- name: CouponBatchRetrying :exec
-- The store didn't answer: the batch tries again at until.
UPDATE coupon_batches SET error = @error, waiting_until = @until::timestamptz WHERE id = @id AND status = 'running';

-- name: CouponBatchWaiting :exec
-- The store asked to slow down; the batch goes on at until.
UPDATE coupon_batches SET error = '', waiting_until = @until::timestamptz WHERE id = @id AND status = 'running';

-- name: CouponBatch :one
SELECT * FROM coupon_batches WHERE id = @id;

-- name: CouponBatchRunning :one
SELECT * FROM coupon_batches WHERE pool_id = @pool_id AND kind = 'generate' AND status = 'running';

-- name: CouponBatchLatest :one
SELECT * FROM coupon_batches WHERE pool_id = @pool_id AND kind = 'generate' ORDER BY created_at DESC, id DESC LIMIT 1;

-- name: CouponBatchProgress :exec
-- A step of a batch: the codes confirmed so far and the step under way.
UPDATE coupon_batches SET created = @created, failed = @failed, external_job_id = @external_job_id, pending_sealed = sqlc.narg(pending_sealed),
    waiting_until = NULL, error = ''
WHERE id = @id AND status = 'running';

-- name: CouponBatchFinish :exec
UPDATE coupon_batches SET status = @status, error = @error, created = @created, failed = @failed, external_job_id = '', pending_sealed = NULL,
    waiting_until = NULL, finished_at = @at::timestamptz
WHERE id = @id AND status = 'running';

-- ── Codes ───────────────────────────────────────────────────────────────

-- name: CouponCodesInsert :many
-- Codes a duplicate of one the pool has are left out (ON CONFLICT).
INSERT INTO coupon_codes (id, workspace_id, pool_id, code_sealed, code_hash, batch_id, external_id, created_at)
SELECT x.id, @workspace_id::uuid, @pool_id::uuid, x.code_sealed, x.code_hash, @batch_id::uuid, x.external_id, @now::timestamptz
FROM (SELECT unnest(@ids::uuid[]) AS id, unnest(@codes_sealed::bytea[]) AS code_sealed, unnest(@code_hashes::bytea[]) AS code_hash,
             unnest(@external_ids::text[]) AS external_id) x
ON CONFLICT (pool_id, code_hash) DO NOTHING
RETURNING id;

-- name: CouponCodesRemoveFree :execrows
-- Removes every code nobody got yet; a store pool's removal job then
-- removes them at the store (CouponCodesUnstored).
UPDATE coupon_codes SET removed_at = @now::timestamptz
WHERE pool_id = @pool_id AND assigned_at IS NULL AND removed_at IS NULL;

-- name: CouponCodesUnstored :many
-- Removed codes the store still has.
SELECT id, code_sealed, external_id FROM coupon_codes
WHERE pool_id = @pool_id AND removed_at IS NOT NULL AND external_id <> ''
ORDER BY id
LIMIT @lim;

-- name: CouponCodesStoreRemoved :exec
UPDATE coupon_codes SET external_id = '' WHERE id = ANY(@ids::uuid[]) AND removed_at IS NOT NULL;

-- name: CouponCodesExport :many
SELECT c.code_sealed, c.assigned_at, c.redeemed_at, c.removed_at, coalesce(o.name, '')::text AS order_name
FROM coupon_codes c LEFT JOIN store_orders o ON o.id = c.order_id
WHERE c.pool_id = @pool_id
ORDER BY c.created_at, c.id;

-- name: CouponAllocate :many
-- Gives a page of messages their code from one pool, in one statement: a
-- message keeps the code it was given (a page sent again), a contact
-- keeps their earlier code when the pool gives one per contact, and the
-- others take free codes (skipping those another send is taking), paired
-- by row number. Messages left without a row found no code.
WITH want AS (
    SELECT w.message_id, w.contact_id, w.ord
    FROM (SELECT unnest(@message_ids::uuid[]) AS message_id, unnest(@contact_ids::uuid[]) AS contact_id,
                 generate_subscripts(@message_ids::uuid[], 1) AS ord) w
), held AS (
    SELECT c.message_id, c.code_sealed FROM coupon_codes c
    WHERE c.pool_id = @pool_id AND c.message_id = ANY(@message_ids::uuid[])
), earlier AS (
    SELECT DISTINCT ON (c.contact_id) c.contact_id, c.code_sealed FROM coupon_codes c
    WHERE @one_per_contact::boolean AND c.pool_id = @pool_id AND c.contact_id = ANY(@contact_ids::uuid[])
    ORDER BY c.contact_id, c.assigned_at, c.id
), need AS (
    SELECT w.message_id, w.contact_id, row_number() OVER (ORDER BY w.ord) AS rn FROM want w
    WHERE NOT EXISTS (SELECT 1 FROM held h WHERE h.message_id = w.message_id)
      AND NOT EXISTS (SELECT 1 FROM earlier e WHERE e.contact_id = w.contact_id)
), picked AS (
    SELECT f.id, row_number() OVER (ORDER BY f.id) AS rn
    FROM (SELECT c.id FROM coupon_codes c
          WHERE c.pool_id = @pool_id AND c.assigned_at IS NULL AND c.removed_at IS NULL
          ORDER BY c.id
          LIMIT CASE WHEN @take::boolean THEN (SELECT count(*) FROM need) ELSE 0 END
          FOR UPDATE SKIP LOCKED) f
), taken AS (
    UPDATE coupon_codes c SET assigned_at = @now::timestamptz, contact_id = n.contact_id, message_id = n.message_id
    FROM need n JOIN picked p ON p.rn = n.rn
    WHERE c.id = p.id
    RETURNING c.message_id, c.code_sealed
)
SELECT h.message_id::uuid AS message_id, h.code_sealed FROM held h
UNION ALL
SELECT w.message_id, e.code_sealed FROM want w JOIN earlier e ON e.contact_id = w.contact_id
WHERE NOT EXISTS (SELECT 1 FROM held h WHERE h.message_id = w.message_id)
UNION ALL
SELECT t.message_id::uuid, t.code_sealed FROM taken t;

-- name: CouponCodesOfMessages :many
-- The codes messages were given (view online, link redirects): by
-- message, or the contact's when the pool gives one per contact.
SELECT c.message_id, c.contact_id, c.code_sealed FROM coupon_codes c
WHERE c.pool_id = @pool_id
  AND (c.message_id = ANY(@message_ids::uuid[]) OR (@one_per_contact::boolean AND c.contact_id = ANY(@contact_ids::uuid[])))
ORDER BY c.assigned_at, c.id;

-- name: CouponRedeem :execrows
-- An order used these codes: the first matching code of the workspace for
-- each (one already matched to this order counts once) is redeemed.
UPDATE coupon_codes c SET redeemed_at = @at::timestamptz, order_id = @order_id::uuid
WHERE c.id IN (
    SELECT (SELECT x.id FROM coupon_codes x
            WHERE x.workspace_id = @workspace_id AND x.code_hash = h.hash AND x.redeemed_at IS NULL
            ORDER BY x.assigned_at IS NULL, x.assigned_at, x.id
            LIMIT 1)
    FROM unnest(@hashes::bytea[]) AS h(hash)
    WHERE NOT EXISTS (SELECT 1 FROM coupon_codes y WHERE y.workspace_id = @workspace_id AND y.code_hash = h.hash AND y.order_id = @order_id::uuid)
);

-- ── What uses a pool ────────────────────────────────────────────────────

-- name: CouponLiveAutomationEmails :many
-- The emails live automations send (each email step of their live
-- version), to find the flows a pool keeps sending to.
SELECT a.id, a.name, concat_ws(' ', ev.subject, ev.preheader, ev.html)::text AS content
FROM automations a
JOIN automation_versions v ON v.automation_id = a.id AND v.version = a.live_version
CROSS JOIN LATERAL jsonb_array_elements(v.graph -> 'nodes') AS n (node)
JOIN automation_emails e ON e.workspace_id = a.workspace_id
    AND e.id = CASE WHEN n.node -> 'data' -> 'email' ->> 'emailId' ~ '^[0-9a-fA-F-]{36}$' THEN (n.node -> 'data' -> 'email' ->> 'emailId')::uuid END
JOIN automation_email_versions ev ON ev.email_id = e.id
    AND ev.version = coalesce(nullif(coalesce((n.node -> 'data' -> 'email' ->> 'emailVersion')::int, 0), 0), e.latest_version)
WHERE a.workspace_id = @workspace_id AND a.status = 'live' AND a.deleted_at IS NULL AND n.node ->> 'type' = 'email';

-- name: CouponUsedCampaigns :many
-- The campaigns a pool's page lists: status, audience size and list names.
SELECT c.id, c.name, c.status, c.audience_count,
       coalesce((SELECT string_agg(l.name, ', ' ORDER BY l.name) FROM lists l
                 WHERE l.workspace_id = c.workspace_id AND jsonb_exists(coalesce(c.audience -> 'lists', '[]'::jsonb), l.id::text)), '')::text AS list_names,
       concat_ws(' ', c.subject, c.preheader, c.mjml, c.html)::text AS content
FROM campaigns c
WHERE c.workspace_id = @workspace_id AND c.id = ANY(@ids::uuid[])
ORDER BY c.updated_at DESC, c.id;

-- name: CouponUsedAutomationEmails :many
-- The emails of each automation's draft and live versions whose latest
-- version mentions a pool (narrowed further in Go), with their flow.
SELECT DISTINCT ON (a.name, a.id, e.name, e.id) e.id, e.name, a.id AS automation_id, a.name AS automation_name, a.status AS automation_status,
       concat_ws(' ', ev.subject, ev.preheader, ev.mjml, ev.html)::text AS content
FROM automations a
JOIN automation_versions v ON v.automation_id = a.id AND (v.version = a.draft_version OR v.version = a.live_version)
CROSS JOIN LATERAL jsonb_array_elements(v.graph -> 'nodes') AS n (node)
JOIN automation_emails e ON e.workspace_id = a.workspace_id AND e.deleted_at IS NULL AND e.id::text = n.node -> 'data' -> 'email' ->> 'emailId'
JOIN automation_email_versions ev ON ev.email_id = e.id AND ev.version = e.latest_version
WHERE a.workspace_id = @workspace_id AND a.deleted_at IS NULL AND n.node ->> 'type' = 'email'
  AND strpos(concat_ws(' ', ev.subject, ev.preheader, ev.mjml, ev.html), @needle::text) > 0
ORDER BY a.name, a.id, e.name, e.id;

-- name: CouponHeldRuns :many
-- Runs of these automations whose email step waits for a discount code.
SELECT a.id, a.name, count(*)::bigint AS runs
FROM automation_runs r JOIN automations a ON a.id = r.automation_id
WHERE a.workspace_id = @workspace_id AND a.id = ANY(@ids::uuid[]) AND r.status = 'waiting' AND r.state ->> 'wait' = 'hold'
GROUP BY a.id, a.name
ORDER BY a.name;

-- name: CouponAutomationEmailContent :one
-- An automation email version's subject, preheader and HTML (0: its latest).
SELECT concat_ws(' ', v.subject, v.preheader, v.html)::text AS content
FROM automation_emails e JOIN automation_email_versions v ON v.email_id = e.id
WHERE e.id = @email_id AND e.workspace_id = @workspace_id AND v.version = coalesce(nullif(@version::int, 0), e.latest_version);

-- ── Jobs ────────────────────────────────────────────────────────────────

-- name: CouponPoolByID :one
SELECT * FROM coupon_pools WHERE id = @id;

-- name: CouponBatchesRunning :many
-- The workspace's pools creating codes now.
SELECT * FROM coupon_batches WHERE workspace_id = @workspace_id AND kind = 'generate' AND status = 'running';

-- name: CouponBatchesRunningAll :many
SELECT id FROM coupon_batches WHERE kind = 'generate' AND status = 'running';

-- name: CouponPoolsUnstored :many
-- Pools with removed codes the store still has.
SELECT DISTINCT pool_id FROM coupon_codes WHERE removed_at IS NOT NULL AND external_id <> '';

-- ── Stores ──────────────────────────────────────────────────────────────

-- name: CouponStoreAllowed :one
-- Who first approved a store's discount scopes (the store page's row).
SELECT actor_name, occurred_at FROM audit_events
WHERE target_type = 'store' AND target_id = @store_id::text AND action = 'store.installed' AND after -> 'scopes' ? 'write_discounts'
ORDER BY occurred_at
LIMIT 1;

-- name: CouponStorePools :many
-- The active pools that create their codes in a store.
SELECT name FROM coupon_pools WHERE store_id = @store_id AND status = 'active' ORDER BY name;
