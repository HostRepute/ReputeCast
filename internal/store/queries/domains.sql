-- W10 · Sending domains and sender identities (Settings › Senders, the
-- domain step of Connect a sending provider, system mail). Query names start
-- with Dom or Sender so they never collide with other domains.

-- name: DomList :many
SELECT * FROM sending_domains
WHERE workspace_id = @workspace_id
ORDER BY created_at, domain;

-- name: DomGet :one
SELECT * FROM sending_domains
WHERE workspace_id = @workspace_id AND id = @id;

-- name: DomGetAny :one
SELECT * FROM sending_domains WHERE id = @id;

-- name: DomGetByName :one
SELECT * FROM sending_domains
WHERE workspace_id = @workspace_id AND domain = @domain;

-- name: DomCreate :one
INSERT INTO sending_domains (id, workspace_id, domain, note, dns_records, status, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @domain, @note, @dns_records, 'pending', @created_by, @created_at, @created_at)
RETURNING *;

-- name: DomUpdateNote :exec
UPDATE sending_domains SET note = @note, updated_at = @updated_at WHERE id = @id;

-- name: DomUpdateTracking :exec
UPDATE sending_domains SET tracking_domain = @tracking_domain, tracking_status = @tracking_status, updated_at = @updated_at
WHERE id = @id;

-- name: DomUpdateRecords :exec
UPDATE sending_domains SET dns_records = @dns_records, updated_at = @updated_at WHERE id = @id;

-- name: DomSaveCheck :exec
-- The result of a live DNS check: every record's status, the authentication
-- summary and detail, the tracking domain and where the zone is hosted.
UPDATE sending_domains
SET dns_records = @dns_records, auth_summary = @auth_summary, auth_detail = @auth_detail, status = @status,
    tracking_status = @tracking_status, dns_host = @dns_host, nameservers = @nameservers::text[], last_checked_at = @checked_at
WHERE id = @id;

-- name: DomDelete :exec
DELETE FROM sending_domains WHERE id = @id;

-- name: DomDue :many
-- Sending domains whose DNS was last checked before a time (the periodic
-- re-verification job), oldest first.
SELECT d.id FROM sending_domains d
JOIN workspaces w ON w.id = d.workspace_id AND w.deleted_at IS NULL
WHERE d.last_checked_at IS NULL OR d.last_checked_at < @before
ORDER BY d.last_checked_at NULLS FIRST, d.id
LIMIT @lim;

-- name: DomDMARCReports :one
-- Where HostRepute collects DMARC aggregate reports for this workspace (rua=).
SELECT coalesce((SELECT dmarc_rua FROM hostrepute_connections WHERE workspace_id = @workspace_id AND status = 'connected'), '')::text AS rua;

-- name: SenderList :many
SELECT s.*, d.domain AS domain_name
FROM senders s
JOIN sending_domains d ON d.id = s.domain_id
WHERE s.workspace_id = @workspace_id
ORDER BY s.created_at, s.email;

-- name: SenderGet :one
SELECT s.*, d.domain AS domain_name
FROM senders s
JOIN sending_domains d ON d.id = s.domain_id
WHERE s.workspace_id = @workspace_id AND s.id = @id;

-- name: SenderGetByEmail :one
SELECT s.*, d.domain AS domain_name
FROM senders s
JOIN sending_domains d ON d.id = s.domain_id
WHERE s.workspace_id = @workspace_id AND s.email = @email;

-- name: SenderListByDomain :many
SELECT * FROM senders WHERE domain_id = @domain_id ORDER BY created_at, email;

-- name: SenderCreate :one
INSERT INTO senders (id, workspace_id, domain_id, name, email, reply_to, note, verification, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @domain_id, @name, @email, @reply_to, @note, @verification, @created_by, @created_at, @created_at)
RETURNING *;

-- name: SenderUpdate :exec
UPDATE senders
SET name = @name, reply_to = @reply_to, note = @note, verification = @verification, updated_at = @updated_at
WHERE id = @id;

-- name: SenderSetVerification :exec
UPDATE senders SET verification = @verification WHERE id = @id;

-- name: SenderDelete :exec
DELETE FROM senders WHERE id = @id;

-- name: SenderCampaignCount :one
-- Campaigns that are not sent yet and use the sender (deleting it clears them).
SELECT count(*)::bigint AS n FROM campaigns
WHERE sender_id = @sender_id AND deleted_at IS NULL AND status IN ('draft', 'approval', 'approved', 'scheduled', 'sending', 'paused');

-- name: SenderWorkspaceDefaults :one
SELECT default_sender_id, default_automation_sender_id, default_system_sender_id, sending_defaults, send_thresholds, branding
FROM workspaces WHERE id = @id;

-- name: SenderSetStreamDefaults :exec
UPDATE workspaces
SET default_sender_id = @default_sender_id, default_automation_sender_id = @default_automation_sender_id,
    default_system_sender_id = @default_system_sender_id, updated_at = @updated_at
WHERE id = @id;

-- name: SenderSetSendingDefaults :exec
UPDATE workspaces SET sending_defaults = @sending_defaults, updated_at = @updated_at WHERE id = @id;

-- name: SenderSetFooterAddress :exec
-- The publisher's postal address (branding.footer_address), shown in email
-- footers and on public pages.
UPDATE workspaces
SET branding = jsonb_set(branding, '{footer_address}', to_jsonb(@footer_address::text)), updated_at = @updated_at
WHERE id = @id;
