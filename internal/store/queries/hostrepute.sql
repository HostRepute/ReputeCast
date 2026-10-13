-- name: GetHostReputeConnection :one
SELECT * FROM hostrepute_connections WHERE workspace_id = @workspace_id;

-- name: GetHostReputeConnectionForUpdate :one
SELECT * FROM hostrepute_connections WHERE workspace_id = @workspace_id FOR UPDATE;

-- name: UpsertHostReputeConnection :one
-- Connect or reconnect: new tokens and account details, status back to
-- connected. Monitored assets and alert history survive a reconnect.
INSERT INTO hostrepute_connections (workspace_id, method, account_email, team_name, scopes, token_enc, refresh_enc, token_expires_at,
    status, status_detail, api_url, account_id, team_id, plan, monitor_limit, sources, check_interval_minutes, dmarc_rua,
    connected_by, connected_at, token_rejected_at, last_sync_error, updated_at)
VALUES (@workspace_id, @method, @account_email, @team_name, @scopes, @token_enc, @refresh_enc, @token_expires_at,
    'connected', '', @api_url, @account_id, @team_id, @plan, @monitor_limit, @sources, @check_interval_minutes, @dmarc_rua,
    @connected_by, @connected_at, NULL, '', @connected_at)
ON CONFLICT (workspace_id) DO UPDATE SET
    method = EXCLUDED.method, account_email = EXCLUDED.account_email, team_name = EXCLUDED.team_name, scopes = EXCLUDED.scopes,
    token_enc = EXCLUDED.token_enc, refresh_enc = EXCLUDED.refresh_enc, token_expires_at = EXCLUDED.token_expires_at,
    status = 'connected', status_detail = '', api_url = EXCLUDED.api_url, account_id = EXCLUDED.account_id, team_id = EXCLUDED.team_id,
    plan = EXCLUDED.plan, monitor_limit = EXCLUDED.monitor_limit, sources = EXCLUDED.sources,
    check_interval_minutes = EXCLUDED.check_interval_minutes, dmarc_rua = EXCLUDED.dmarc_rua,
    connected_by = EXCLUDED.connected_by, connected_at = EXCLUDED.connected_at, token_rejected_at = NULL, last_sync_error = '',
    updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: UpdateHostReputeTokens :exec
UPDATE hostrepute_connections SET token_enc = @token_enc, refresh_enc = @refresh_enc, token_expires_at = @token_expires_at, updated_at = now()
WHERE workspace_id = @workspace_id;

-- name: MarkHostReputeSynced :exec
-- A sync finished: account, team, scopes and monitored assets as HostRepute
-- reports them now.
UPDATE hostrepute_connections SET
    account_email = @account_email, team_name = @team_name, team_id = @team_id, plan = @plan, scopes = @scopes,
    monitored = @monitored, monitor_limit = @monitor_limit, sources = @sources, check_interval_minutes = @check_interval_minutes,
    dmarc_rua = @dmarc_rua, last_check_at = @last_check_at, alerts_since = @alerts_since,
    last_sync_at = @synced_at, last_sync_attempt_at = @synced_at, last_sync_error = '',
    status = 'connected', status_detail = '', token_rejected_at = NULL, updated_at = @synced_at
WHERE workspace_id = @workspace_id;

-- name: RecordHostReputeSyncError :exec
UPDATE hostrepute_connections SET last_sync_attempt_at = @attempted_at, last_sync_error = @error, updated_at = @attempted_at
WHERE workspace_id = @workspace_id;

-- name: MarkHostReputeTokenRejected :execrows
-- HostRepute rejected the token (revoked on its side, or expired and not
-- refreshable). Only the first rejection is recorded.
UPDATE hostrepute_connections SET status = @status, status_detail = @status_detail, token_rejected_at = @rejected_at,
    last_sync_attempt_at = @rejected_at, updated_at = @rejected_at
WHERE workspace_id = @workspace_id AND status = 'connected';

-- name: SetHostReputeMonitored :exec
UPDATE hostrepute_connections SET monitored = @monitored, monitor_limit = @monitor_limit, updated_at = now()
WHERE workspace_id = @workspace_id;

-- name: DeleteHostReputeConnection :execrows
DELETE FROM hostrepute_connections WHERE workspace_id = @workspace_id;

-- name: ListConnectedHostRepute :many
-- Workspaces whose HostRepute connection the hourly sync refreshes.
SELECT c.workspace_id FROM hostrepute_connections c
JOIN workspaces w ON w.id = c.workspace_id AND w.deleted_at IS NULL
WHERE c.status = 'connected'
ORDER BY c.last_sync_at NULLS FIRST;

-- name: CreateHostReputeOAuthState :exec
INSERT INTO hostrepute_oauth_states (id, workspace_id, user_id, state_hash, verifier_enc, redirect_uri, expires_at)
VALUES (@id, @workspace_id, @user_id, @state_hash, @verifier_enc, @redirect_uri, @expires_at);

-- name: ConsumeHostReputeOAuthState :one
-- A state is single use: reading it deletes it.
DELETE FROM hostrepute_oauth_states WHERE state_hash = @state_hash RETURNING *;

-- name: DeleteExpiredHostReputeOAuthStates :execrows
DELETE FROM hostrepute_oauth_states WHERE expires_at < now();

-- name: DeleteDeliverabilityBlocklistResults :exec
DELETE FROM blocklist_results WHERE workspace_id = @workspace_id AND via = @via;

-- name: InsertDeliverabilityBlocklistResult :exec
INSERT INTO blocklist_results (id, workspace_id, via, asset_type, asset, source, impact, rank, listed, listed_since, delist_url, detail,
    delisting, delisting_requested_at, checked_at)
VALUES (@id, @workspace_id, @via, @asset_type, @asset, @source, @impact, @rank, @listed, @listed_since, @delist_url, @detail,
    @delisting, @delisting_requested_at, @checked_at)
ON CONFLICT (workspace_id, via, asset, source) DO UPDATE SET
    asset_type = EXCLUDED.asset_type, impact = EXCLUDED.impact, rank = EXCLUDED.rank, listed = EXCLUDED.listed,
    listed_since = CASE WHEN EXCLUDED.listed THEN coalesce(blocklist_results.listed_since, EXCLUDED.listed_since) END,
    delist_url = EXCLUDED.delist_url, detail = EXCLUDED.detail, delisting = EXCLUDED.delisting,
    delisting_requested_at = EXCLUDED.delisting_requested_at, checked_at = EXCLUDED.checked_at;

-- name: UpsertDeliverabilityPostmasterDay :exec
INSERT INTO postmaster_daily (workspace_id, domain, day, via, spam_rate, domain_reputation, ip_reputation, spf_rate, dkim_rate, dmarc_rate,
    tls_rate, delivery_errors, fbl_flagged)
VALUES (@workspace_id, @domain, @day, @via, @spam_rate, @domain_reputation, @ip_reputation, @spf_rate, @dkim_rate, @dmarc_rate,
    @tls_rate, @delivery_errors, @fbl_flagged)
ON CONFLICT (workspace_id, domain, day) DO UPDATE SET
    via = EXCLUDED.via, spam_rate = EXCLUDED.spam_rate, domain_reputation = EXCLUDED.domain_reputation,
    ip_reputation = EXCLUDED.ip_reputation, spf_rate = EXCLUDED.spf_rate, dkim_rate = EXCLUDED.dkim_rate,
    dmarc_rate = EXCLUDED.dmarc_rate, tls_rate = EXCLUDED.tls_rate, delivery_errors = EXCLUDED.delivery_errors,
    fbl_flagged = EXCLUDED.fbl_flagged;

-- name: UpsertDeliverabilitySNDSDay :exec
INSERT INTO snds_daily (workspace_id, ip, day, via, filter_result, complaint_rate, trap_hits, messages, jmrp_reports)
VALUES (@workspace_id, @ip, @day, @via, @filter_result, @complaint_rate, @trap_hits, @messages, @jmrp_reports)
ON CONFLICT (workspace_id, ip, day) DO UPDATE SET
    via = EXCLUDED.via, filter_result = EXCLUDED.filter_result, complaint_rate = EXCLUDED.complaint_rate,
    trap_hits = EXCLUDED.trap_hits, messages = EXCLUDED.messages, jmrp_reports = EXCLUDED.jmrp_reports;

-- name: UpsertDeliverabilityDMARCReport :exec
INSERT INTO dmarc_reports (id, workspace_id, domain, org_name, report_id, date_begin, date_end, policy, records, via, received_at)
VALUES (@id, @workspace_id, @domain, @org_name, @report_id, @date_begin, @date_end, @policy, @records, @via, @received_at)
ON CONFLICT (workspace_id, org_name, report_id) DO UPDATE SET
    domain = EXCLUDED.domain, date_begin = EXCLUDED.date_begin, date_end = EXCLUDED.date_end, policy = EXCLUDED.policy,
    records = EXCLUDED.records, via = EXCLUDED.via;

-- name: DeleteHostReputeData :exec
-- Disconnect forgets what was synced from HostRepute (HostRepute keeps it).
WITH b AS (DELETE FROM blocklist_results WHERE blocklist_results.workspace_id = @workspace_id AND blocklist_results.via = 'hostrepute'),
     p AS (DELETE FROM postmaster_daily WHERE postmaster_daily.workspace_id = @workspace_id AND postmaster_daily.via = 'hostrepute'),
     s AS (DELETE FROM snds_daily WHERE snds_daily.workspace_id = @workspace_id AND snds_daily.via = 'hostrepute')
DELETE FROM dmarc_reports WHERE dmarc_reports.workspace_id = @workspace_id AND dmarc_reports.via = 'hostrepute';

-- name: InsertDeliverabilityIncident :one
-- Incidents with an external id (HostRepute alerts) are stored once; the
-- returned row count says whether this one is new.
INSERT INTO incidents (id, workspace_id, occurred_at, kind, severity, title, detail, campaign_id, provider_id, resolved_at, source, asset, external_id)
VALUES (@id, @workspace_id, @occurred_at, @kind, @severity, @title, @detail, @campaign_id, @provider_id, @resolved_at, @source, @asset, @external_id)
ON CONFLICT (workspace_id, external_id) WHERE external_id IS NOT NULL DO NOTHING
RETURNING id;

-- name: HostReputeProviderForIP :one
-- The connected provider that sends from a dedicated IP.
SELECT id, name FROM providers
WHERE workspace_id = @workspace_id AND disconnected_at IS NULL AND @ip::text = ANY(ips)
ORDER BY connected_at LIMIT 1;

-- name: HostReputeAssetSources :many
-- Everything an asset list is built from: sending domains (with the records
-- their providers require) …
SELECT id, domain, dns_records, created_at FROM sending_domains
WHERE workspace_id = @workspace_id ORDER BY created_at, domain;

-- name: HostReputeActiveProviders :many
-- … and the connected sending providers (dedicated IPs, shared pools, server hostnames).
SELECT id, adapter, name, label, config, status, status_detail, ip_mode, ips, capabilities, limits, health, pricing, connected_at
FROM providers
WHERE workspace_id = @workspace_id AND disconnected_at IS NULL
ORDER BY connected_at, name;

-- name: HostReputeWorkspaceSlug :one
SELECT slug FROM workspaces WHERE id = @id;

-- name: HostReputeCountWebhooks :one
SELECT count(*)::bigint AS endpoints,
       count(*) FILTER (WHERE failing_since IS NOT NULL OR (last_status IS NOT NULL AND (last_status < 200 OR last_status >= 300)))::bigint AS failing
FROM webhooks WHERE workspace_id = @workspace_id AND enabled;

-- name: HostReputeCountAgents :one
SELECT count(*)::bigint FROM api_keys
WHERE workspace_id = @workspace_id AND kind = 'mcp_agent' AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > now());

-- name: HostReputeDataSourceVia :one
-- Where Postmaster and SNDS data come from now (hostrepute, direct or none).
SELECT coalesce((SELECT via FROM postmaster_daily WHERE postmaster_daily.workspace_id = @workspace_id ORDER BY day DESC LIMIT 1), '')::text AS postmaster,
       coalesce((SELECT via FROM snds_daily WHERE snds_daily.workspace_id = @workspace_id ORDER BY day DESC LIMIT 1), '')::text AS snds;

-- name: HostReputeUserName :one
SELECT name, email FROM users WHERE id = @id;
