-- Routing rules, failover policies and the route tester.

-- name: RoutingListRules :many
SELECT * FROM routing_rules
WHERE workspace_id = @workspace_id
ORDER BY CASE stream WHEN 'broadcast' THEN 0 WHEN 'automation' THEN 1 ELSE 2 END, priority, id;

-- name: RoutingInsertRule :exec
INSERT INTO routing_rules (id, workspace_id, stream, priority, name, conditions, splits, fallback_provider_id, enabled, updated_by, updated_at)
VALUES (@id, @workspace_id, @stream, @priority, @name, @conditions, @splits, @fallback_provider_id, true, @updated_by, @updated_at);

-- name: RoutingListPolicies :many
SELECT * FROM stream_policies WHERE workspace_id = @workspace_id;

-- name: RoutingUpsertPolicy :exec
INSERT INTO stream_policies (workspace_id, stream, failover, bulk_terms_ack_at, bulk_terms_ack_by)
VALUES (@workspace_id, @stream, @failover, @bulk_terms_ack_at, @bulk_terms_ack_by)
ON CONFLICT (workspace_id, stream) DO UPDATE
SET failover = EXCLUDED.failover, bulk_terms_ack_at = EXCLUDED.bulk_terms_ack_at, bulk_terms_ack_by = EXCLUDED.bulk_terms_ack_by;

-- name: RoutingSaved :one
-- When the routing was last saved and by whom.
SELECT r.updated_at, coalesce(u.name, '')::text AS updated_by_name
FROM routing_rules r LEFT JOIN users u ON u.id = r.updated_by
WHERE r.workspace_id = @workspace_id
ORDER BY r.updated_at DESC
LIMIT 1;

-- name: RoutingLists :many
SELECT id, name FROM lists WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY created_at, name;

-- name: RoutingSenderDomains :many
SELECT id, domain, dns_records FROM sending_domains WHERE workspace_id = @workspace_id ORDER BY created_at, domain;

-- name: RoutingIsSuppressed :one
SELECT EXISTS (SELECT 1 FROM suppressions WHERE workspace_id = @workspace_id AND email = @email)::boolean;

-- name: RoutingContact :one
SELECT c.id, c.mailbox_provider, c.mailbox_detail,
       coalesce(array_agg(lm.list_id) FILTER (WHERE lm.status = 'subscribed'), '{}')::uuid[] AS list_ids
FROM contacts c
LEFT JOIN list_memberships lm ON lm.contact_id = c.id
WHERE c.workspace_id = @workspace_id AND c.email = @email AND c.erased_at IS NULL
GROUP BY c.id;

-- name: RoutingBroadcastSizes :one
-- Smallest and largest broadcast audience over a window ("31,988 to 48,213
-- recipients per send").
SELECT coalesce(min(audience_count), 0)::bigint AS smallest, coalesce(max(audience_count), 0)::bigint AS largest
FROM campaigns
WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND origin = '' AND started_at >= @since AND audience_count > 0;

-- name: RoutingLock :exec
-- Serializes routing saves of a workspace (the version check and the write).
SELECT pg_advisory_xact_lock(hashtextextended('routing:' || (@workspace_id::uuid)::text, 0));

-- name: RoutingDeleteStreamRules :exec
-- A save replaces only the streams it changes.
DELETE FROM routing_rules WHERE workspace_id = @workspace_id AND stream = ANY(@streams::text[]);

-- name: RoutingNextBroadcast :one
-- The broadcast the route tester's Campaign tab simulates (PR-14): the next
-- scheduled one, else the draft edited last, with its sender's verification
-- per provider (failover targets are vetted with it, R3.U4c).
SELECT c.id, c.name, c.scheduled_at, c.audience,
       coalesce(split_part(sn.email, '@', 2), '')::text AS sender_domain,
       (sn.id IS NOT NULL)::boolean AS has_sender,
       coalesce(sn.verification, '{}'::jsonb)::jsonb AS sender_verification
FROM campaigns c
LEFT JOIN senders sn ON sn.id = c.sender_id
WHERE c.workspace_id = @workspace_id AND c.deleted_at IS NULL AND c.origin = '' AND c.channel IN ('', 'email')
  AND c.status IN ('scheduled', 'draft')
ORDER BY (c.status = 'scheduled') DESC, c.scheduled_at ASC NULLS LAST, c.updated_at DESC
LIMIT 1;

-- name: RoutingStreamRecipients :many
-- A stream's latest recipients since a time (review R3.U4a): the routing
-- impact preview runs them through the saved and the unsaved routing, with
-- what routing reads at send time (mailbox provider, transactional
-- category). Automations, System and Transactional mail is never a
-- campaign's (messages_stream_recent_idx).
SELECT m.email, m.mailbox_provider, coalesce(t.category_id::text, '')::text AS category
FROM messages m
LEFT JOIN transactional_messages t ON t.message_id = m.id
WHERE m.workspace_id = @workspace_id AND m.stream = @stream AND m.campaign_id IS NULL AND m.queued_at >= @since::timestamptz
ORDER BY m.queued_at DESC
LIMIT @lim;

-- name: RoutingContactAddresses :many
-- Addresses of the workspace's newest contacts, for the impact preview of
-- a stream that mailed nobody lately (R3.U4a).
SELECT c.email, c.mailbox_provider FROM contacts c
WHERE c.workspace_id = @workspace_id AND c.status = 'active' AND c.erased_at IS NULL
ORDER BY c.created_at DESC
LIMIT @lim;

-- name: RoutingCampaignAudience :many
-- A page of a campaign's list audience for the route tester: subscribed,
-- not suppressed, not on an excluded list, with each contact's mailbox
-- provider, tier and lists.
SELECT c.id, c.email, c.mailbox_provider, c.tier,
       coalesce(array_agg(lm.list_id) FILTER (WHERE lm.status = 'subscribed'), '{}')::uuid[] AS list_ids
FROM contacts c
LEFT JOIN list_memberships lm ON lm.contact_id = c.id
WHERE c.workspace_id = @workspace_id AND c.status = 'active' AND c.erased_at IS NULL AND c.id > @after::uuid
  AND EXISTS (SELECT 1 FROM list_memberships m WHERE m.contact_id = c.id AND m.status = 'subscribed'
              AND m.list_id = ANY(ARRAY(SELECT ll.id FROM lists ll WHERE ll.id = ANY(@lists::uuid[]) AND ll.deleted_at IS NULL)))
  AND NOT EXISTS (SELECT 1 FROM list_memberships x WHERE x.contact_id = c.id AND x.status = 'subscribed' AND x.list_id = ANY(@exclude_lists::uuid[]))
  AND NOT EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = c.workspace_id AND s.email = c.email)
GROUP BY c.id
ORDER BY c.id
LIMIT @size;

-- name: RoutingCampaignAudienceSize :one
-- How many recipients the route tester's audience has, the sunset tier
-- apart: a simulation of the first recipients is scaled to it.
SELECT count(*) FILTER (WHERE c.tier <> 'sunset')::bigint AS engaged, count(*) FILTER (WHERE c.tier = 'sunset')::bigint AS sunset
FROM contacts c
WHERE c.workspace_id = @workspace_id AND c.status = 'active' AND c.erased_at IS NULL
  AND EXISTS (SELECT 1 FROM list_memberships m WHERE m.contact_id = c.id AND m.status = 'subscribed'
              AND m.list_id = ANY(ARRAY(SELECT ll.id FROM lists ll WHERE ll.id = ANY(@lists::uuid[]) AND ll.deleted_at IS NULL)))
  AND NOT EXISTS (SELECT 1 FROM list_memberships x WHERE x.contact_id = c.id AND x.status = 'subscribed' AND x.list_id = ANY(@exclude_lists::uuid[]))
  AND NOT EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = c.workspace_id AND s.email = c.email);

-- name: RoutingDefaultSenderVerification :one
-- The default sender's verification per provider: a failover target must
-- be able to sign for it (PR-14 target checks).
SELECT sn.verification FROM senders sn JOIN workspaces w ON w.default_sender_id = sn.id WHERE w.id = @workspace_id;

