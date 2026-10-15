-- Integrations hub: connections, their keys, the 7-day call log, the
-- workflows seen, managed webhooks and WordPress sites (internal/integrations).

-- name: IntegrationsActive :many
SELECT i.*, coalesce(u.name, '')::text AS connected_by_name
FROM integrations i
LEFT JOIN users u ON u.id = i.connected_by
WHERE i.workspace_id = @workspace_id AND i.disconnected_at IS NULL
ORDER BY i.connected_at, i.id;

-- name: IntegrationsAllActive :many
-- Every connection of the instance, for the health job.
SELECT i.* FROM integrations i
JOIN workspaces w ON w.id = i.workspace_id AND w.deleted_at IS NULL
WHERE i.disconnected_at IS NULL;

-- name: IntegrationGet :one
SELECT i.*, coalesce(u.name, '')::text AS connected_by_name
FROM integrations i
LEFT JOIN users u ON u.id = i.connected_by
WHERE i.id = @id AND i.workspace_id = @workspace_id;

-- name: IntegrationActiveByKind :one
SELECT i.*, coalesce(u.name, '')::text AS connected_by_name
FROM integrations i
LEFT JOIN users u ON u.id = i.connected_by
WHERE i.workspace_id = @workspace_id AND i.kind = @kind AND i.disconnected_at IS NULL
ORDER BY i.connected_at DESC
LIMIT 1;

-- name: IntegrationLastLabel :one
-- The label the first automation platform was connected with ("Acme ops"),
-- to suggest the next key's name.
SELECT label FROM integrations
WHERE workspace_id = @workspace_id AND kind <> 'wordpress' AND label <> ''
ORDER BY connected_at
LIMIT 1;

-- name: IntegrationInsert :exec
INSERT INTO integrations (id, workspace_id, kind, label, items, off_items, settings, connected_by, connected_at)
VALUES (@id, @workspace_id, @kind, @label, @items, @off_items, @settings, @connected_by, @connected_at);

-- name: IntegrationSetItems :exec
UPDATE integrations SET items = @items, off_items = @off_items WHERE id = @id;

-- name: IntegrationSetLabel :exec
UPDATE integrations SET label = @label WHERE id = @id;

-- name: IntegrationSetSettings :exec
UPDATE integrations SET settings = @settings WHERE id = @id;

-- name: IntegrationReconnected :exec
-- A new key after a revoked or expired one: waiting for its first call again.
UPDATE integrations SET first_call_at = NULL, attention = '[]', notified = '{}' WHERE id = @id;

-- name: IntegrationDisconnect :execrows
UPDATE integrations SET disconnected_at = @at::timestamptz, disconnected_by = @by
WHERE id = @id AND workspace_id = @workspace_id AND disconnected_at IS NULL;

-- name: IntegrationTouchCall :exec
UPDATE integrations SET last_call_at = greatest(coalesce(last_call_at, @at::timestamptz), @at::timestamptz),
    first_call_at = coalesce(first_call_at, @at::timestamptz)
WHERE id = @id;

-- name: IntegrationSetAttention :exec
UPDATE integrations SET attention = @attention, notified = @notified WHERE id = @id;

-- name: IntegrationKeyLimit :one
-- What the authenticator needs about an integration key's connection.
SELECT i.id, i.workspace_id, i.kind, i.label, i.items, i.disconnected_at
FROM integrations i WHERE i.id = @id;

-- name: IntegrationKeys :many
-- The keys of a connection, newest first: the current key, then rotated ones.
SELECT k.id, k.workspace_id, k.name, k.prefix, k.permissions, k.scope, k.created_by, k.created_at, k.expires_at,
       k.last_used_at, k.last_used_ip, k.last_used_client, k.revoked_at, k.grace_until, k.rotated_from,
       coalesce(u.name, '')::text AS owner_name, coalesce(u.avatar_color, '')::text AS owner_color,
       coalesce(m.status, '')::text AS owner_status, m.deactivated_at AS owner_left_at
FROM api_keys k
LEFT JOIN users u ON u.id = k.created_by
LEFT JOIN workspace_members m ON m.workspace_id = k.workspace_id AND m.user_id = k.created_by
WHERE k.integration_id = @integration_id
ORDER BY k.created_at DESC, k.id DESC;

-- name: IntegrationKeysInWorkspace :many
-- Every key of the workspace's active connections (Keys & MCP's second table).
SELECT k.id, k.integration_id, k.name, k.prefix, k.permissions, k.scope, k.created_by, k.created_at, k.expires_at,
       k.last_used_at, k.last_used_ip, k.last_used_client, k.revoked_at, k.grace_until,
       i.kind, i.label, coalesce(u.name, '')::text AS owner_name, coalesce(u.avatar_color, '')::text AS owner_color
FROM api_keys k
JOIN integrations i ON i.id = k.integration_id AND i.disconnected_at IS NULL
LEFT JOIN users u ON u.id = k.created_by
WHERE k.workspace_id = @workspace_id
ORDER BY i.connected_at, k.created_at DESC;

-- name: IntegrationKeyInsert :exec
INSERT INTO api_keys (id, workspace_id, kind, name, description, prefix, token_hash, permissions, scope, created_by, created_at, expires_at, rotated_from, integration_id)
VALUES (@id, @workspace_id, 'api_key', @name, '', @prefix, @token_hash, @permissions, @scope, @created_by, @created_at, sqlc.narg(expires_at), sqlc.narg(rotated_from), @integration_id);

-- name: IntegrationKeyRevoke :execrows
-- Revokes a key; with grace_until it keeps working until then (rotation).
UPDATE api_keys SET revoked_at = @at::timestamptz, grace_until = sqlc.narg(grace_until)
WHERE id = @id AND integration_id = @integration_id AND revoked_at IS NULL;

-- name: IntegrationKeyStopGrace :execrows
UPDATE api_keys SET grace_until = NULL
WHERE integration_id = @integration_id AND revoked_at IS NOT NULL AND grace_until > now();

-- name: IntegrationKeysRevokeAll :many
UPDATE api_keys SET revoked_at = coalesce(revoked_at, @at::timestamptz), grace_until = NULL
WHERE integration_id = @integration_id AND (revoked_at IS NULL OR grace_until IS NOT NULL)
RETURNING id, name, prefix;

-- name: IntegrationKeySetPermissions :exec
UPDATE api_keys SET permissions = @permissions WHERE integration_id = @integration_id AND (revoked_at IS NULL OR grace_until > now());

-- name: IntegrationKeySetScope :exec
UPDATE api_keys SET scope = @scope WHERE integration_id = @integration_id AND revoked_at IS NULL;

-- name: IntegrationKeySetName :exec
UPDATE api_keys SET name = @name WHERE integration_id = @integration_id AND revoked_at IS NULL;

-- name: IntegrationKeyTakeOver :exec
-- Gives the connection's live keys (and a rotated one still in its grace
-- period) a new owner; the secret stays the same.
UPDATE api_keys SET created_by = @owner
WHERE integration_id = @integration_id AND (revoked_at IS NULL OR grace_until > now());

-- name: IntegrationsOwnedBy :many
-- Connections whose live key a member owns (deactivate or remove dialog).
SELECT DISTINCT i.id, i.kind, i.label, i.connected_at
FROM integrations i
JOIN api_keys k ON k.integration_id = i.id AND k.created_by = @user_id AND (k.revoked_at IS NULL OR k.grace_until > now())
WHERE i.workspace_id = @workspace_id AND i.disconnected_at IS NULL
ORDER BY i.connected_at;

-- name: IntegrationCallInsert :exec
INSERT INTO integration_calls (id, integration_id, workspace_id, api_key_id, occurred_at, workflow_key, workflow_name, operation, item,
    method, path, status, code, detail, duration_ms, target, contact_id, note, request, response, ip, client)
VALUES (@id, @integration_id, @workspace_id, sqlc.narg(api_key_id), @occurred_at, @workflow_key, @workflow_name, @operation, @item,
    @method, @path, @status, @code, @detail, @duration_ms, @target, sqlc.narg(contact_id), @note, sqlc.narg(request), sqlc.narg(response), sqlc.narg(ip), @client);

-- name: IntegrationCalls :many
-- The call log, newest first. Errors are answers of 400 and above.
SELECT c.id, c.occurred_at, c.workflow_key, c.workflow_name, c.operation, c.item, c.method, c.path, c.status, c.code, c.detail,
       c.duration_ms, c.target, c.contact_id, c.note, c.ip, c.client,
       coalesce(ct.email, '')::text AS contact_email, (ct.erased_at IS NOT NULL)::boolean AS contact_erased
FROM integration_calls c
LEFT JOIN contacts ct ON ct.id = c.contact_id
WHERE c.integration_id = @integration_id AND c.occurred_at >= @since
  AND (NOT @errors_only::boolean OR c.status >= 400)
ORDER BY c.occurred_at DESC, c.id DESC
LIMIT @lim OFFSET @off;

-- name: IntegrationCallCounts :one
SELECT count(*)::bigint AS calls, count(*) FILTER (WHERE status >= 400)::bigint AS errors,
       count(*) FILTER (WHERE status = 409)::bigint AS consent_kept
FROM integration_calls WHERE integration_id = @integration_id AND occurred_at >= @since;

-- name: IntegrationCallsLater :many
-- Successful calls after an error from the same workflow and action: the
-- error is resolved.
SELECT occurred_at FROM integration_calls
WHERE integration_id = @integration_id AND workflow_key = @workflow_key AND item = @item
  AND occurred_at > @after AND status < 400
ORDER BY occurred_at
LIMIT 3;

-- name: IntegrationCallGet :one
SELECT c.*, coalesce(ct.email, '')::text AS contact_email, (ct.erased_at IS NOT NULL)::boolean AS contact_erased
FROM integration_calls c
LEFT JOIN contacts ct ON ct.id = c.contact_id
WHERE c.id = @id AND c.integration_id = @integration_id;

-- name: IntegrationCallsByWorkflow :many
-- Calls and errors per workflow in the window.
SELECT workflow_key, count(*)::bigint AS calls, count(*) FILTER (WHERE status >= 400)::bigint AS errors,
       count(*) FILTER (WHERE status >= 400 AND status <> 409)::bigint AS failures,
       count(*) FILTER (WHERE status = 409)::bigint AS consent_kept
FROM integration_calls WHERE integration_id = @integration_id AND occurred_at >= @since
GROUP BY workflow_key;

-- name: IntegrationCallsRejected :many
-- Calls refused since a key stopped working (revoked, expired, owner left).
SELECT occurred_at, workflow_key, workflow_name, status, code
FROM integration_calls
WHERE integration_id = @integration_id AND occurred_at >= @since AND code = ANY (@codes::text[])
ORDER BY occurred_at DESC
LIMIT 200;

-- name: IntegrationCallsHealth :one
-- The last hour, for "more than half of the calls failing".
SELECT count(*)::bigint AS calls,
       count(*) FILTER (WHERE status >= 500 OR (status >= 400 AND status NOT IN (404, 409, 422)))::bigint AS failed,
       min(occurred_at) FILTER (WHERE status >= 500 OR (status >= 400 AND status NOT IN (404, 409, 422)))::timestamptz AS first_failed
FROM integration_calls WHERE integration_id = @integration_id AND occurred_at >= @since;

-- name: IntegrationCallsPurge :execrows
DELETE FROM integration_calls WHERE occurred_at < @before;

-- name: IntegrationCallsForgetErased :execrows
-- Bodies of calls about a contact who has since been erased.
UPDATE integration_calls c SET request = NULL, response = NULL, target = '{}'
FROM contacts ct
WHERE ct.id = c.contact_id AND ct.erased_at IS NOT NULL AND (c.request IS NOT NULL OR c.response IS NOT NULL);

-- name: IntegrationWorkflowSeen :exec
INSERT INTO integration_workflows (integration_id, key, external_id, name, uses, first_seen_at, last_seen_at)
VALUES (@integration_id, @key, @external_id, @name, @uses, @at, @at)
ON CONFLICT (integration_id, key) DO UPDATE SET
    name = CASE WHEN excluded.name <> '' THEN excluded.name ELSE integration_workflows.name END,
    external_id = CASE WHEN excluded.external_id <> '' THEN excluded.external_id ELSE integration_workflows.external_id END,
    uses = excluded.uses,
    last_seen_at = greatest(integration_workflows.last_seen_at, excluded.last_seen_at);

-- name: IntegrationWorkflow :one
SELECT * FROM integration_workflows WHERE integration_id = @integration_id AND key = @key;

-- name: IntegrationWorkflows :many
SELECT * FROM integration_workflows
WHERE integration_id = @integration_id AND last_seen_at >= @since
ORDER BY first_seen_at, key;

-- name: IntegrationWorkflowsPurge :execrows
DELETE FROM integration_workflows WHERE last_seen_at < @before;

-- name: IntegrationWebhookInsert :exec
INSERT INTO integration_webhooks (webhook_id, integration_id, api_key_id, event, workflow_key, workflow_name, created_at)
VALUES (@webhook_id, @integration_id, sqlc.narg(api_key_id), @event, @workflow_key, @workflow_name, @created_at);

-- name: IntegrationWebhooks :many
-- The webhooks a connection manages, with 24-hour delivery counts.
SELECT h.id, h.url, h.events, h.enabled, h.created_at, h.last_status, h.last_delivery_at, h.failing_since, h.demo,
       iw.event, iw.workflow_key, iw.workflow_name, iw.api_key_id,
       coalesce(s.delivered, 0)::bigint AS delivered_24h, coalesce(s.total, 0)::bigint AS total_24h,
       coalesce(q.retrying, 0)::bigint AS retrying, coalesce(q.queued, 0)::bigint AS queued,
       coalesce(q.dead, 0)::bigint AS dead, (SELECT min(d2.next_attempt_at) FROM webhook_deliveries d2 WHERE d2.webhook_id = h.id AND d2.state IN ('pending','retrying')) AS next_attempt_at,
       coalesce(w7.deliveries, 0)::bigint AS deliveries_7d
FROM integration_webhooks iw
JOIN webhooks h ON h.id = iw.webhook_id
LEFT JOIN LATERAL (
    SELECT count(*) FILTER (WHERE d.state = 'delivered') AS delivered, count(*) AS total
    FROM webhook_deliveries d WHERE d.webhook_id = h.id AND d.created_at > now() - interval '24 hours'
) s ON true
LEFT JOIN LATERAL (
    SELECT count(*) FILTER (WHERE d.state = 'retrying') AS retrying,
           count(*) FILTER (WHERE d.state IN ('pending','retrying','held','delivering')) AS queued,
           count(*) FILTER (WHERE d.state = 'dead') AS dead,
           min(d.next_attempt_at) FILTER (WHERE d.state IN ('pending','retrying')) AS next_attempt_at
    FROM webhook_deliveries d WHERE d.webhook_id = h.id
) q ON true
LEFT JOIN LATERAL (
    SELECT count(*) AS deliveries FROM webhook_deliveries d
    WHERE d.webhook_id = h.id AND d.state = 'delivered' AND d.created_at > now() - interval '7 days'
) w7 ON true
WHERE iw.integration_id = @integration_id
ORDER BY h.created_at;

-- name: IntegrationWebhookOf :one
SELECT iw.webhook_id, iw.event, h.url FROM integration_webhooks iw
JOIN webhooks h ON h.id = iw.webhook_id
WHERE iw.webhook_id = @webhook_id AND iw.integration_id = @integration_id;

-- name: IntegrationManagedHooks :many
-- Endpoints of the workspace an integration manages (the Webhooks page).
SELECT iw.webhook_id, i.id AS integration_id, i.kind, i.label, iw.workflow_name, h.url
FROM integration_webhooks iw
JOIN integrations i ON i.id = iw.integration_id
JOIN webhooks h ON h.id = iw.webhook_id
WHERE i.workspace_id = @workspace_id;

-- name: IntegrationManagedHook :one
SELECT iw.webhook_id, i.id AS integration_id, i.workspace_id, i.kind, i.label, iw.event, iw.workflow_name, h.url
FROM integration_webhooks iw
JOIN integrations i ON i.id = iw.integration_id
JOIN webhooks h ON h.id = iw.webhook_id
WHERE iw.webhook_id = @webhook_id;

-- name: IntegrationWebhookDelete :execrows
-- Deleting the endpoint deletes its queued and dead-letter deliveries too.
DELETE FROM webhooks WHERE id = @id;

-- name: IntegrationWebhooksDeleteAll :many
DELETE FROM webhooks h USING integration_webhooks iw
WHERE iw.webhook_id = h.id AND iw.integration_id = @integration_id
RETURNING h.id, h.url, h.events;

-- name: IntegrationWebhookQueued :one
-- Deliveries a disconnect removes: queued, retrying and dead-lettered.
SELECT count(*) FILTER (WHERE d.state IN ('pending','retrying','held','delivering'))::bigint AS queued,
       count(*) FILTER (WHERE d.state = 'dead')::bigint AS dead
FROM webhook_deliveries d JOIN integration_webhooks iw ON iw.webhook_id = d.webhook_id
WHERE iw.integration_id = @integration_id;

-- name: IntegrationDeadLetters :many
SELECT d.id, d.event, d.payload, d.created_at, d.last_attempt_at, d.status_code, d.error
FROM webhook_deliveries d
WHERE d.webhook_id = @webhook_id AND d.state = 'dead'
ORDER BY d.created_at
LIMIT 50;

-- name: IntegrationLastAttempt :one
-- The newest attempt of a failing managed webhook ("Attempt 4 of 8 · 502").
SELECT d.id, d.event, d.attempt, d.max_attempts, d.status_code, d.error, d.last_attempt_at, d.next_attempt_at, d.created_at
FROM webhook_deliveries d
WHERE d.webhook_id = @webhook_id AND d.state IN ('retrying','pending','delivering') AND d.attempt > 0
ORDER BY d.created_at
LIMIT 1;

-- name: WordPressSiteInsert :exec
INSERT INTO wordpress_sites (integration_id, workspace_id, address) VALUES (@integration_id, @workspace_id, @address);

-- name: WordPressSite :one
SELECT * FROM wordpress_sites WHERE integration_id = @integration_id;

-- name: WordPressSites :many
SELECT s.*, i.label, i.connected_at, i.connected_by, i.first_call_at, i.last_call_at, coalesce(u.name, '')::text AS connected_by_name
FROM wordpress_sites s
JOIN integrations i ON i.id = s.integration_id AND i.disconnected_at IS NULL
LEFT JOIN users u ON u.id = i.connected_by
WHERE s.workspace_id = @workspace_id
ORDER BY i.connected_at, i.id;

-- name: WordPressSiteReport :exec
UPDATE wordpress_sites SET site_url = @site_url, name = @name, plugin_version = @plugin_version, wp_version = @wp_version,
    php_version = @php_version, feed_url = @feed_url, language_plugin = @language_plugin, categories = @categories, forms = @forms,
    first_report_at = coalesce(first_report_at, @at::timestamptz), last_report_at = @at::timestamptz
WHERE integration_id = @integration_id;

-- name: WordPressSiteChecked :exec
UPDATE wordpress_sites SET last_check_at = @at::timestamptz, last_check = @last_check
WHERE integration_id = @integration_id;

-- name: WordPressSiteSetRule :exec
UPDATE wordpress_sites SET rule_enabled = @enabled, rule = @rule, rule_updated_by = @by, rule_updated_at = @at::timestamptz
WHERE integration_id = @integration_id;

-- name: WordPressSiteElsewhere :one
-- A site connects to one workspace: another active connection of the same address.
SELECT w.name AS workspace_name, w.slug AS workspace_slug
FROM wordpress_sites s
JOIN integrations i ON i.id = s.integration_id AND i.disconnected_at IS NULL
JOIN workspaces w ON w.id = s.workspace_id AND w.deleted_at IS NULL
WHERE lower(s.site_url) = lower(@site_url) AND s.integration_id <> @integration_id AND s.site_url <> ''
LIMIT 1;

-- name: WordPressDraftByPost :one
SELECT * FROM wordpress_drafts WHERE integration_id = @integration_id AND post_id = @post_id;

-- name: WordPressDraftInsert :exec
INSERT INTO wordpress_drafts (id, integration_id, workspace_id, campaign_id, post_id, post_title, post_url, published_at, notified_user, created_at)
VALUES (@id, @integration_id, @workspace_id, @campaign_id, @post_id, @post_title, @post_url, sqlc.narg(published_at), sqlc.narg(notified_user), @created_at);

-- name: WordPressDrafts :many
SELECT d.*, coalesce(c.name, '')::text AS campaign_name, coalesce(c.status, '')::text AS campaign_status,
       coalesce(u.name, '')::text AS notified_name, coalesce(l.name, '')::text AS audience_name
FROM wordpress_drafts d
LEFT JOIN campaigns c ON c.id = d.campaign_id AND c.deleted_at IS NULL
LEFT JOIN users u ON u.id = d.notified_user
LEFT JOIN LATERAL (
    SELECT l2.name FROM lists l2 WHERE l2.id = ((c.audience -> 'lists') ->> 0)::uuid
) l ON true
WHERE d.integration_id = @integration_id
ORDER BY d.created_at DESC
LIMIT @lim;

-- name: WordPressDraftsWeek :one
SELECT count(*)::bigint FROM wordpress_drafts WHERE integration_id = @integration_id AND created_at >= @since;

-- name: WordPressForms :many
-- The workspace's signup forms, for the block and the site page.
SELECT f.id, f.public_id, f.name, f.kind, f.status, f.allowed_origins, f.list_ids, f.double_opt_in
FROM forms f WHERE f.workspace_id = @workspace_id
ORDER BY f.created_at;

-- name: IntegrationReleaseGet :one
SELECT * FROM integration_releases WHERE kind = @kind;

-- name: IntegrationReleaseSet :exec
INSERT INTO integration_releases (kind, version, source, checked_at) VALUES (@kind, @version, @source, now())
ON CONFLICT (kind) DO UPDATE SET version = excluded.version, source = excluded.source, checked_at = excluded.checked_at;

-- name: WebhookContactEventsTake :many
-- The next contact events to deliver; deleted as they are taken.
DELETE FROM webhook_contact_events
WHERE id IN (SELECT e.id FROM webhook_contact_events e ORDER BY e.id LIMIT @lim FOR UPDATE SKIP LOCKED)
RETURNING id, workspace_id, event, contact_id, list_id, data, created_at;

-- name: IntegrationEventContacts :many
-- Contacts for event payloads, with their lists.
SELECT c.id, c.workspace_id, c.email, c.first_name, c.last_name, c.company, c.country, c.timezone, c.language, c.tags, c.attributes,
       c.status, c.source, c.created_at, c.erased_at,
       coalesce((SELECT jsonb_agg(jsonb_build_object('id', l.id, 'name', l.name, 'status', m.status) ORDER BY l.name)
                 FROM list_memberships m JOIN lists l ON l.id = m.list_id AND l.deleted_at IS NULL
                 WHERE m.contact_id = c.id), '[]')::jsonb AS lists
FROM contacts c WHERE c.id = ANY (@ids::uuid[]);

-- name: IntegrationListNames :many
SELECT id, name, double_opt_in FROM lists WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY name;

-- name: IntegrationMembership :one
SELECT m.status, m.subscribed_at, m.unsubscribed_at, m.created_at FROM list_memberships m
WHERE m.list_id = @list_id AND m.contact_id = @contact_id;

-- name: IntegrationLastLeft :one
-- When and how a contact left a list ("left Beta Program on 2 Oct").
SELECT e.occurred_at, e.source FROM consent_events e
WHERE e.contact_id = @contact_id AND e.list_id = @list_id AND e.action = 'unsubscribed'
ORDER BY e.occurred_at DESC
LIMIT 1;

-- name: IntegrationContactInsert :one
INSERT INTO contacts (id, workspace_id, email, first_name, last_name, company, language, mailbox_provider, consent, source, tags, attributes,
                      role_address, last_activity_at)
VALUES (@id, @workspace_id, @email, @first_name, @last_name, @company, @language, @mailbox_provider, @consent, @source, @tags, @attributes,
        @role_address, @now::timestamptz)
RETURNING id, email;

-- name: IntegrationContactUpdate :exec
UPDATE contacts SET first_name = @first_name, last_name = @last_name, company = @company, language = @language,
    tags = @tags, attributes = @attributes, updated_at = @now
WHERE id = @id AND workspace_id = @workspace_id;

-- name: IntegrationJoinList :exec
-- Adds a contact to a list, or confirms a pending one. Never touches a
-- membership the contact left: the caller refuses those first.
INSERT INTO list_memberships (list_id, contact_id, status, source, subscribed_at)
VALUES (@list_id, @contact_id, @status, @source, CASE WHEN @status::text = 'subscribed' THEN @now::timestamptz END)
ON CONFLICT (list_id, contact_id) DO UPDATE SET
    status = CASE WHEN list_memberships.status = 'subscribed' THEN 'subscribed' ELSE excluded.status END,
    subscribed_at = coalesce(list_memberships.subscribed_at, excluded.subscribed_at)
WHERE list_memberships.status <> 'unsubscribed';

-- name: IntegrationConsentEvent :exec
INSERT INTO consent_events (id, contact_id, list_id, action, source, detail, note, occurred_at)
VALUES (@id, @contact_id, sqlc.narg(list_id), @action, @source, @detail, @note, @occurred_at);

-- name: IntegrationCustomFields :many
SELECT key, label, type, options FROM custom_fields WHERE workspace_id = @workspace_id ORDER BY position, key;

-- name: IntegrationEventFlows :many
-- Live automations with their trigger and graph, to name the flows an API
-- event starts or ends ("which starts the Onboarding automation").
SELECT a.name, a.trigger, v.graph
FROM automations a
JOIN automation_versions v ON v.automation_id = a.id AND v.version = a.live_version
WHERE a.workspace_id = @workspace_id AND a.status = 'live' AND a.deleted_at IS NULL
ORDER BY a.name;

-- name: IntegrationTransactionalTemplates :many
-- Published transactional templates with their live version's variables,
-- for "Send a transactional email" in the apps.
SELECT t.key, t.name, t.live_version, v.variables
FROM transactional_templates t
JOIN transactional_template_versions v ON v.template_id = t.id AND v.version = t.live_version
WHERE t.workspace_id = @workspace_id AND t.deleted_at IS NULL AND t.archived_at IS NULL AND t.live_version IS NOT NULL
ORDER BY t.name;

-- name: IntegrationTemplateVariables :one
-- A transactional template's live variables, to mask its sensitive ones in
-- the call log.
SELECT v.variables
FROM transactional_templates t
JOIN transactional_template_versions v ON v.template_id = t.id AND v.version = t.live_version
WHERE t.workspace_id = @workspace_id AND t.key = @key AND t.deleted_at IS NULL;

-- name: IntegrationTemplates :many
-- Saved templates for the post-draft rule (kind template, not deleted).
SELECT id, name, mjml FROM templates WHERE workspace_id = @workspace_id AND kind = 'template' AND deleted_at IS NULL
ORDER BY name;

-- name: IntegrationTemplate :one
SELECT id, name, mjml, html FROM templates WHERE id = @id AND workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: IntegrationSenders :many
SELECT s.id, s.name, s.email FROM senders s WHERE s.workspace_id = @workspace_id ORDER BY s.email;

-- name: IntegrationDraftInsert :exec
-- A campaign draft created from a WordPress post.
INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, sender_id, audience, audience_count, editor_mode,
                       mjml, html, plain_text, version, content_hash, origin, note, created_by, updated_by, created_at, updated_at, autosaved_at)
VALUES (@id, @workspace_id, @name, 'draft', @subject, @preheader, sqlc.narg(sender_id), @audience, sqlc.narg(audience_count), 'visual',
        @mjml, @html, @plain_text, 1, @content_hash, 'wordpress', @note, sqlc.narg(created_by), sqlc.narg(created_by), @now, @now, @now);

-- name: IntegrationDraftVersion :exec
INSERT INTO campaign_versions (campaign_id, version, subject, preheader, mjml, html, plain_text, saved_by, saved_at, label, source, size_bytes)
VALUES (@campaign_id, 1, @subject, @preheader, @mjml, @html, @plain_text, sqlc.narg(saved_by), @saved_at, @label, 'api', @size_bytes);

-- name: IntegrationInstanceAdmins :many
SELECT name, email FROM users WHERE is_instance_admin AND status = 'active' ORDER BY created_at LIMIT 3;

-- name: IntegrationKeyRevokedBy :one
-- Who revoked a key in Keys & MCP, and when.
SELECT e.actor_name, e.occurred_at FROM audit_events e
WHERE e.workspace_id = @workspace_id AND e.action = 'apikey.revoked' AND e.target_id = @key_id::text
ORDER BY e.occurred_at DESC
LIMIT 1;

-- name: IntegrationKeyTouchClient :exec
UPDATE api_keys SET last_used_client = @client WHERE id = @id AND @client::text <> '';

-- name: IntegrationListSizes :many
SELECT l.id, l.name, l.double_opt_in, coalesce(n.subscribed, 0)::bigint AS contacts
FROM lists l
LEFT JOIN LATERAL (
    SELECT count(*) AS subscribed FROM list_memberships m WHERE m.list_id = l.id AND m.status = 'subscribed'
) n ON true
WHERE l.workspace_id = @workspace_id AND l.deleted_at IS NULL
ORDER BY l.name;

-- name: IntegrationMembers :many
-- Active members, Admins first then Owners, with their role (helpers and pickers).
SELECT u.id, u.name, u.email, u.avatar_color, r.key AS role_key, r.name AS role_name, r.builtin, r.permissions
FROM workspace_members m
JOIN roles r ON r.id = m.role_id
JOIN users u ON u.id = m.user_id AND u.status = 'active'
WHERE m.workspace_id = @workspace_id AND m.status = 'active'
ORDER BY CASE r.key WHEN 'admin' THEN 0 WHEN 'owner' THEN 1 ELSE 2 END, u.name;

-- name: IntegrationFirstCall :one
-- The first call made with a key (the "waiting for a first call" timeline).
SELECT c.id, c.occurred_at, c.operation, c.status, c.ip, c.client, c.workflow_name
FROM integration_calls c
WHERE c.integration_id = @integration_id AND c.api_key_id = @api_key_id
ORDER BY c.occurred_at
LIMIT 1;

-- name: IntegrationWebhookCounts :one
SELECT count(*)::bigint AS endpoints,
       count(*) FILTER (WHERE failing_since IS NOT NULL)::bigint AS failing
FROM webhooks WHERE workspace_id = @workspace_id;
