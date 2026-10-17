-- Growth sources (audience-insights brief §3.4).

-- name: GrowthSaveTouch :exec
-- Puts a signup's UTM parameters and pages on the consent records it wrote.
UPDATE consent_events
SET utm_source = @utm_source, utm_medium = @utm_medium, utm_campaign = @utm_campaign, utm_content = @utm_content,
    utm_term = @utm_term, page = @page, referrer = @referrer
WHERE contact_id = @contact_id AND occurred_at = @at
  AND action IN ('subscribed', 'resubscribed', 'confirmed');

-- name: GrowthStateGet :one
SELECT * FROM growth_state WHERE workspace_id = @workspace_id;

-- name: GrowthStateEnsure :one
-- The first run of a workspace starts UTM capture and attributes the
-- contacts it already has.
INSERT INTO growth_state (workspace_id, capture_started_at, updated_at)
VALUES (@workspace_id, @at::timestamptz, @at::timestamptz)
ON CONFLICT (workspace_id) DO UPDATE SET updated_at = growth_state.updated_at
RETURNING *;

-- name: GrowthStateSetAttributed :exec
UPDATE growth_state SET attributed_through = @attributed_through, updated_at = now() WHERE workspace_id = @workspace_id;

-- name: GrowthStateSetRolledUp :exec
UPDATE growth_state SET rolled_up_through = @rolled_up_through, updated_at = now() WHERE workspace_id = @workspace_id;

-- name: GrowthWorkspaces :many
SELECT w.id, w.slug, w.timezone, s.capture_started_at, s.attributed_through, s.rolled_up_through
FROM workspaces w
LEFT JOIN growth_state s ON s.workspace_id = w.id
WHERE w.deleted_at IS NULL
ORDER BY w.id;

-- name: GrowthWorkspaceProviders :many
SELECT id FROM providers WHERE workspace_id = @workspace_id;

-- name: GrowthFormsOf :many
SELECT id, name, kind, slug, public_id, list_choice, list_ids FROM forms WHERE workspace_id = @workspace_id ORDER BY created_at;

-- name: GrowthImportsOf :many
SELECT id, source, created_at, finished_at, coalesce((stats ->> 'imported')::bigint, 0)::bigint AS imported
FROM imports WHERE workspace_id = @workspace_id AND status = 'done'
ORDER BY created_at;

-- name: GrowthKeysOf :many
SELECT id, name, prefix, integration_id FROM api_keys WHERE workspace_id = @workspace_id;

-- name: GrowthIntegrationsOf :many
SELECT i.id, i.kind, i.label FROM integrations i WHERE i.workspace_id = @workspace_id;

-- name: GrowthWorkflowsOf :many
SELECT w.integration_id, w.key, w.name
FROM integration_workflows w
JOIN integrations i ON i.id = w.integration_id
WHERE i.workspace_id = @workspace_id;

-- name: GrowthSourceCounts :many
-- Contacts per source, for the contacts filter and segment conditions.
SELECT g.source_kind, g.source_id, count(*)::bigint AS contacts, min(g.joined_at)::timestamptz AS first_join
FROM contact_growth g
JOIN contacts c ON c.id = g.contact_id AND c.erased_at IS NULL
WHERE g.workspace_id = @workspace_id
GROUP BY g.source_kind, g.source_id
ORDER BY 3 DESC;

-- name: GrowthContactTouch :one
-- A contact's attribution, for the profile's join line.
SELECT * FROM contact_growth WHERE contact_id = @contact_id;

-- name: McpListByName :one
-- A list by its name, for MCP tools that take a list name.
SELECT id FROM lists WHERE workspace_id = @workspace_id AND lower(name) = lower(@name::text) AND deleted_at IS NULL
ORDER BY created_at LIMIT 1;
