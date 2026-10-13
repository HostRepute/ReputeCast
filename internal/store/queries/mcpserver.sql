-- Read models behind the MCP server's tools (W03). Writes go through the
-- owning domain's rules; these queries only read or make small, audited changes.

-- name: McpFindCampaign :one
-- A campaign by id or by exact (case-insensitive) name, newest first.
SELECT id, name, status, subject, preheader, audience, audience_count, scheduled_at, started_at, completed_at, version, mjml
FROM campaigns
WHERE workspace_id = @workspace_id AND deleted_at IS NULL
  AND (id::text = @ref::text OR lower(name) = lower(@ref::text))
ORDER BY created_at DESC
LIMIT 1;

-- name: McpCampaignTotals :one
SELECT coalesce(sum(sent), 0)::bigint AS sent, coalesce(sum(delivered), 0)::bigint AS delivered,
       coalesce(sum(deferred), 0)::bigint AS deferred,
       coalesce(sum(hard_bounced + soft_bounced), 0)::bigint AS bounced, coalesce(sum(complained), 0)::bigint AS complained,
       coalesce(sum(opened_human), 0)::bigint AS opened_human, coalesce(sum(opened_machine), 0)::bigint AS opened_machine,
       coalesce(sum(clicked), 0)::bigint AS clicked, coalesce(sum(unsubscribed), 0)::bigint AS unsubscribed
FROM campaign_counters WHERE campaign_id = @campaign_id;

-- name: McpCampaignByProvider :many
SELECT p.name AS provider, coalesce(sum(c.sent), 0)::bigint AS sent, coalesce(sum(c.delivered), 0)::bigint AS delivered,
       coalesce(sum(c.hard_bounced + c.soft_bounced), 0)::bigint AS bounced, coalesce(sum(c.complained), 0)::bigint AS complained
FROM campaign_counters c JOIN providers p ON p.id = c.provider_id
WHERE c.campaign_id = @campaign_id
GROUP BY p.name ORDER BY sent DESC;

-- name: McpCampaignByMailbox :many
SELECT c.mailbox_provider, coalesce(sum(c.sent), 0)::bigint AS sent, coalesce(sum(c.delivered), 0)::bigint AS delivered,
       coalesce(sum(c.opened_human), 0)::bigint AS opened_human, coalesce(sum(c.complained), 0)::bigint AS complained
FROM campaign_counters c
WHERE c.campaign_id = @campaign_id
GROUP BY c.mailbox_provider ORDER BY sent DESC;

-- name: McpLatestReadiness :one
SELECT verdict, checks, estimates, ran_at FROM readiness_runs WHERE campaign_id = @campaign_id ORDER BY ran_at DESC LIMIT 1;

-- name: McpCampaignRoutes :many
SELECT p.name AS provider, r.status, r.status_detail, r.planned
FROM campaign_routes r JOIN providers p ON p.id = r.provider_id
WHERE r.campaign_id = @campaign_id ORDER BY r.planned DESC;

-- name: McpSearchContacts :many
SELECT c.id, c.email, c.first_name, c.last_name, c.status, c.tier, c.created_at, c.last_open_at
FROM contacts c
WHERE c.workspace_id = @workspace_id AND c.erased_at IS NULL
  AND (c.email ILIKE '%' || @q::text || '%' OR (c.first_name || ' ' || c.last_name) ILIKE '%' || @q::text || '%')
  AND (cardinality(@lists::uuid[]) = 0 OR EXISTS (
        SELECT 1 FROM list_memberships lm WHERE lm.contact_id = c.id AND lm.list_id = ANY(@lists::uuid[])))
ORDER BY c.email
LIMIT @lim;

-- name: McpProviders :many
SELECT id, name, adapter, label, status, status_detail, health, ips, last_event_at
FROM providers WHERE workspace_id = @workspace_id AND disconnected_at IS NULL ORDER BY connected_at;

-- name: McpRouting :many
SELECT r.id, r.stream, r.priority, r.name, r.conditions, r.splits, r.fallback_provider_id, r.enabled, r.updated_at
FROM routing_rules r WHERE r.workspace_id = @workspace_id ORDER BY r.stream, r.priority;

-- name: McpAutomations :many
SELECT id, name, status, trigger, updated_at FROM automations
WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY updated_at DESC LIMIT 100;

-- name: McpSendingDomains :many
SELECT domain, status, auth_summary, last_checked_at FROM sending_domains WHERE workspace_id = @workspace_id ORDER BY domain;

-- name: McpBlocklists :many
SELECT asset, source, impact, listed, listed_since FROM blocklist_results
WHERE workspace_id = @workspace_id AND listed ORDER BY impact DESC, asset;

-- name: McpPostmaster :many
SELECT DISTINCT ON (domain) domain, day, spam_rate, domain_reputation, ip_reputation
FROM postmaster_daily WHERE workspace_id = @workspace_id ORDER BY domain, day DESC;

-- name: McpCreateDraft :one
INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, mjml, created_by, updated_by)
VALUES (@id, @workspace_id, @name, 'draft', @subject, @preheader, @mjml, @created_by, @created_by)
RETURNING id, name, status, version, created_at;

-- name: McpUpdateDraft :one
UPDATE campaigns
SET subject = coalesce(sqlc.narg(subject), subject), preheader = coalesce(sqlc.narg(preheader), preheader),
    mjml = coalesce(sqlc.narg(mjml), mjml), version = version + 1, updated_at = now(), updated_by = @updated_by
WHERE id = @id AND workspace_id = @workspace_id AND status IN ('draft','approval','approved') AND deleted_at IS NULL
RETURNING id, name, status, version, subject, preheader;

-- name: McpCreateSegment :one
INSERT INTO segments (id, workspace_id, name, description, mode, definition, created_by)
VALUES (@id, @workspace_id, @name, @description, 'builder', @definition, @created_by)
RETURNING id, name;

-- name: McpAddSuppression :one
INSERT INTO suppressions (id, workspace_id, email, reason, source, detail, created_by)
VALUES (@id, @workspace_id, @email, 'manual', 'mcp', @detail, @created_by)
ON CONFLICT (workspace_id, email) DO NOTHING
RETURNING id;

-- name: McpGetRoutingRule :one
SELECT r.id, r.stream, r.priority, r.name, r.conditions, r.splits, r.fallback_provider_id, r.enabled
FROM routing_rules r WHERE r.workspace_id = @workspace_id AND r.id = @id;

-- name: McpDefaultRoutingRule :one
SELECT r.id, r.stream, r.priority, r.name, r.conditions, r.splits, r.fallback_provider_id, r.enabled
FROM routing_rules r WHERE r.workspace_id = @workspace_id AND r.stream = @stream
ORDER BY (r.conditions = '{}'::jsonb) DESC, r.priority DESC
LIMIT 1;

-- name: McpSetRoutingSplits :exec
UPDATE routing_rules SET splits = @splits, updated_at = now() WHERE id = @id AND workspace_id = @workspace_id;
