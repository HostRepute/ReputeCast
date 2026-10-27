-- Real renders from the workspace's own rendering service (CO-13).

-- name: RenderingServiceGet :one
SELECT r.workspace_id, r.service, r.region, r.credentials, r.key_hint, r.status, r.last_error, r.clients,
       r.connected_by, r.connected_at, r.checked_at, coalesce(u.name, '')::text AS connected_by_name
FROM rendering_services r
LEFT JOIN users u ON u.id = r.connected_by
WHERE r.workspace_id = @workspace_id;

-- name: RenderingServiceUpsert :exec
INSERT INTO rendering_services (workspace_id, service, region, credentials, key_hint, status, last_error, clients, connected_by, connected_at, checked_at)
VALUES (@workspace_id, @service, @region, @credentials, @key_hint, 'connected', '', @clients, sqlc.narg(connected_by), @now, @now)
ON CONFLICT (workspace_id) DO UPDATE SET service = EXCLUDED.service, region = EXCLUDED.region, credentials = EXCLUDED.credentials,
    key_hint = EXCLUDED.key_hint, status = 'connected', last_error = '', clients = EXCLUDED.clients,
    connected_by = EXCLUDED.connected_by, connected_at = EXCLUDED.connected_at, checked_at = EXCLUDED.checked_at;

-- name: RenderingServiceChecked :exec
-- A test of the stored key: its outcome, and the clients when it worked.
UPDATE rendering_services
SET status = @status, last_error = @last_error, checked_at = @now,
    clients = CASE WHEN @status::text = 'connected' THEN @clients::jsonb ELSE clients END
WHERE workspace_id = @workspace_id;

-- name: RenderingServiceDelete :execrows
DELETE FROM rendering_services WHERE workspace_id = @workspace_id;

-- name: CampaignRenderInsert :exec
INSERT INTO campaign_renders (id, workspace_id, campaign_id, version, variant, service, external_id, clients, status, results, requested_by, requested_at)
VALUES (@id, @workspace_id, @campaign_id, @version, @variant, @service, @external_id, @clients, 'pending', @results, sqlc.narg(requested_by), @requested_at);

-- name: CampaignRenderPending :one
-- A render of this campaign still running, if any.
SELECT id FROM campaign_renders
WHERE workspace_id = @workspace_id AND campaign_id = @campaign_id AND status = 'pending'
ORDER BY requested_at DESC LIMIT 1;

-- name: CampaignRendersList :many
SELECT r.id, r.campaign_id, r.version, r.variant, r.service, r.external_id, r.clients, r.status, r.results, r.error,
       r.requested_at, r.polled_at, r.completed_at, coalesce(u.name, '')::text AS requested_by_name
FROM campaign_renders r
LEFT JOIN users u ON u.id = r.requested_by
WHERE r.workspace_id = @workspace_id AND r.campaign_id = @campaign_id
ORDER BY r.requested_at DESC
LIMIT @lim;

-- name: CampaignRenderUpdate :exec
-- What the service answered when it was asked again.
UPDATE campaign_renders
SET status = @status, results = @results, error = @error, polled_at = @now,
    completed_at = CASE WHEN @status::text = 'pending' THEN NULL ELSE coalesce(completed_at, @now) END
WHERE workspace_id = @workspace_id AND id = @id;
