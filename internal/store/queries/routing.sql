-- Routing rules, failover policies and the route tester.

-- name: RoutingListRules :many
SELECT * FROM routing_rules
WHERE workspace_id = @workspace_id
ORDER BY CASE stream WHEN 'broadcast' THEN 0 WHEN 'automation' THEN 1 ELSE 2 END, priority, id;

-- name: RoutingDeleteRules :exec
DELETE FROM routing_rules WHERE workspace_id = @workspace_id;

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
