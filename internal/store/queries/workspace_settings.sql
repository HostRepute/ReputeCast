-- Workspace settings: general, danger zone and summaries.

-- name: WsSenders :many
SELECT s.id, s.name, s.email, s.verification, d.domain
FROM senders s JOIN sending_domains d ON d.id = s.domain_id
WHERE s.workspace_id = @workspace_id
ORDER BY s.created_at;

-- name: WsProviderNames :many
SELECT id, name, adapter FROM providers WHERE workspace_id = @workspace_id AND disconnected_at IS NULL ORDER BY connected_at;

-- name: WsSenderExists :one
SELECT EXISTS (SELECT 1 FROM senders WHERE id = @id AND workspace_id = @workspace_id);

-- name: WsUpdateGeneral :one
UPDATE workspaces SET name = @name, timezone = @timezone, default_sender_id = @default_sender_id, updated_at = now()
WHERE id = @id AND deleted_at IS NULL
RETURNING id, name, timezone, default_sender_id;

-- name: WsCounts :one
SELECT
  (SELECT count(*) FROM contacts c WHERE c.workspace_id = @workspace_id AND c.erased_at IS NULL)::bigint AS contacts,
  (SELECT count(*) FROM lists l WHERE l.workspace_id = @workspace_id AND l.deleted_at IS NULL)::bigint AS lists,
  (SELECT count(*) FROM segments s WHERE s.workspace_id = @workspace_id)::bigint AS segments,
  (SELECT count(*) FROM campaigns c WHERE c.workspace_id = @workspace_id AND c.deleted_at IS NULL)::bigint AS campaigns,
  (SELECT count(*) FROM automations a WHERE a.workspace_id = @workspace_id)::bigint AS automations,
  (SELECT count(*) FROM templates t WHERE t.workspace_id = @workspace_id AND t.kind = 'template')::bigint AS templates,
  (SELECT count(*) FROM providers p WHERE p.workspace_id = @workspace_id AND p.disconnected_at IS NULL)::bigint AS providers,
  (SELECT count(*) FROM campaigns c WHERE c.workspace_id = @workspace_id AND c.status = 'scheduled' AND c.deleted_at IS NULL)::bigint AS scheduled,
  (SELECT count(*) FROM approval_requests a WHERE a.workspace_id = @workspace_id AND a.status = 'pending')::bigint AS pending_approvals;

-- name: WsMemberCounts :one
SELECT
  count(*) FILTER (WHERE m.status = 'active')::bigint AS active,
  count(*) FILTER (WHERE m.status = 'deactivated')::bigint AS deactivated,
  (SELECT count(*) FROM invitations i WHERE i.workspace_id = @workspace_id AND i.accepted_at IS NULL AND i.revoked_at IS NULL AND i.expires_at > now())::bigint AS invited
FROM workspace_members m WHERE m.workspace_id = @workspace_id;

-- name: WsCancelScheduled :execrows
UPDATE campaigns SET status = 'cancelled', updated_at = now()
WHERE workspace_id = @workspace_id AND status IN ('scheduled','approved') AND deleted_at IS NULL;

-- name: WsSoftDelete :exec
UPDATE workspaces SET deleted_at = now(), slug = slug || '-deleted-' || extract(epoch FROM now())::bigint, updated_at = now()
WHERE id = @id AND deleted_at IS NULL;

-- name: WsPurgeDeleted :many
-- Workspaces past the 7-day restore hold are deleted for good (cascade).
DELETE FROM workspaces WHERE deleted_at IS NOT NULL AND deleted_at < now() - interval '7 days'
RETURNING id, name;

-- name: WsMCPSettings :one
SELECT mcp_settings FROM workspaces WHERE id = @id;

-- name: WsSetMCPSettings :exec
UPDATE workspaces SET mcp_settings = @mcp_settings, updated_at = now() WHERE id = @id;

-- name: WsAuditSettings :one
SELECT audit_retention_days, audit_stream_url, audit_stream_enabled, (audit_stream_secret_enc IS NOT NULL)::boolean AS has_secret,
       audit_stream_status, audit_stream_cursor_at
FROM workspaces WHERE id = @id;
