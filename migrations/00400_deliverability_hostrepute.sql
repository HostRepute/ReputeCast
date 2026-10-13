-- +goose Up
-- W07 · Deliverability and Connect HostRepute (spec 7.8, add-on 5.15).
-- Extends the tables of 00007 with where each piece of data comes from
-- (HostRepute, the built-in checks, a direct connection or an upload), adds
-- the OAuth handshake state, check runs and the local DNS check results.

-- Blocklist results come from HostRepute (80+ sources, hourly) or from the
-- built-in check of 8 public DNSBLs (daily). Both can exist at once: the
-- built-in check runs as a fallback while HostRepute is unreachable.
ALTER TABLE blocklist_results
    ADD COLUMN via text NOT NULL DEFAULT 'builtin' CHECK (via IN ('builtin','hostrepute')),
    -- Display order of the source within its impact (lower = more important).
    ADD COLUMN rank integer NOT NULL DEFAULT 100,
    -- Extra context for a listing: "Network-range listing (/16)".
    ADD COLUMN detail text NOT NULL DEFAULT '',
    -- Delisting workflow (HostRepute): requested → pending → delisted | denied.
    ADD COLUMN delisting text NOT NULL DEFAULT '' CHECK (delisting IN ('','requested','pending','delisted','denied')),
    ADD COLUMN delisting_requested_at timestamptz;
DROP INDEX blocklist_results_key;
CREATE UNIQUE INDEX blocklist_results_key ON blocklist_results (workspace_id, via, asset, source);

-- Google Postmaster Tools and Microsoft SNDS arrive through HostRepute or a
-- direct connection to Google / Microsoft.
ALTER TABLE postmaster_daily
    ADD COLUMN via text NOT NULL DEFAULT 'hostrepute' CHECK (via IN ('hostrepute','direct')),
    -- Feedback-loop identifiers Gmail flagged that day (NULL = no FBL data).
    ADD COLUMN fbl_flagged integer;
ALTER TABLE snds_daily
    ADD COLUMN via text NOT NULL DEFAULT 'hostrepute' CHECK (via IN ('hostrepute','direct')),
    ADD COLUMN jmrp_reports integer NOT NULL DEFAULT 0;

-- DMARC aggregate reports parsed by HostRepute (rua) or uploaded as XML.
ALTER TABLE dmarc_reports
    ADD COLUMN via text NOT NULL DEFAULT 'hostrepute' CHECK (via IN ('hostrepute','upload'));

-- Incidents: who detected them, the asset involved, and HostRepute's alert id
-- so an alert is stored (and notified) once.
ALTER TABLE incidents
    ADD COLUMN source text NOT NULL DEFAULT 'reputecast' CHECK (source IN ('reputecast','hostrepute','builtin')),
    ADD COLUMN asset text NOT NULL DEFAULT '',
    ADD COLUMN external_id text;
CREATE UNIQUE INDEX incidents_external_key ON incidents (workspace_id, external_id) WHERE external_id IS NOT NULL;

-- The HostRepute connection (one per workspace). Tokens stay encrypted in
-- token_enc / refresh_enc and are never returned by the API.
ALTER TABLE hostrepute_connections
    -- API base used for this connection ("demo:" = the built-in demo HostRepute).
    ADD COLUMN api_url text NOT NULL DEFAULT '',
    ADD COLUMN account_id text NOT NULL DEFAULT '',
    ADD COLUMN team_id text NOT NULL DEFAULT '',
    ADD COLUMN plan text NOT NULL DEFAULT '',
    -- Blocklist sources HostRepute checks and how often (84, every 60 min).
    ADD COLUMN sources integer NOT NULL DEFAULT 0,
    ADD COLUMN check_interval_minutes integer NOT NULL DEFAULT 60,
    -- HostRepute's last full check of the monitored assets.
    ADD COLUMN last_check_at timestamptz,
    -- Address HostRepute parses DMARC aggregate reports from (rua=).
    ADD COLUMN dmarc_rua text NOT NULL DEFAULT '',
    -- Sync health: the last attempt and why it failed (HostRepute unreachable).
    ADD COLUMN last_sync_attempt_at timestamptz,
    ADD COLUMN last_sync_error text NOT NULL DEFAULT '',
    -- When HostRepute rejected the token (revoked or expired): data is stale since.
    ADD COLUMN token_rejected_at timestamptz,
    -- Alerts are fetched incrementally from here.
    ADD COLUMN alerts_since timestamptz,
    ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();

-- OAuth 2 authorization-code flow with PKCE: one row per pending handshake.
-- Only the SHA-256 of the state is stored; the verifier is encrypted.
CREATE TABLE hostrepute_oauth_states (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id      uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    state_hash   bytea       NOT NULL,
    verifier_enc bytea       NOT NULL,
    redirect_uri text        NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL
);
CREATE UNIQUE INDEX hostrepute_oauth_states_key ON hostrepute_oauth_states (state_hash);
CREATE INDEX hostrepute_oauth_states_expiry_idx ON hostrepute_oauth_states (expires_at);

-- "Run all checks now" and the scheduled checks, with live progress
-- ("61 of 84 sources answered").
CREATE TABLE deliverability_check_runs (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    via          text        NOT NULL CHECK (via IN ('hostrepute','builtin')),
    trigger      text        NOT NULL DEFAULT 'schedule' CHECK (trigger IN ('schedule','manual','connect','fallback')),
    status       text        NOT NULL DEFAULT 'running' CHECK (status IN ('running','done','failed')),
    assets       text[]      NOT NULL DEFAULT '{}',
    total        integer     NOT NULL DEFAULT 0,
    answered     integer     NOT NULL DEFAULT 0,
    listed       integer     NOT NULL DEFAULT 0,
    error        text        NOT NULL DEFAULT '',
    requested_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    started_at   timestamptz NOT NULL DEFAULT now(),
    finished_at  timestamptz
);
CREATE INDEX deliverability_check_runs_ws_idx ON deliverability_check_runs (workspace_id, started_at DESC);

-- Local DNS checks per asset (internal/dnscheck): SPF, DKIM, DMARC, BIMI and
-- MAIL FROM alignment for domains; reverse DNS for IPs.
CREATE TABLE deliverability_asset_checks (
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    asset        text        NOT NULL,
    asset_type   text        NOT NULL CHECK (asset_type IN ('domain','ip')),
    result       jsonb       NOT NULL DEFAULT '{}',
    checked_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, asset)
);

-- +goose Down
DROP TABLE deliverability_asset_checks;
DROP TABLE deliverability_check_runs;
DROP TABLE hostrepute_oauth_states;
ALTER TABLE hostrepute_connections
    DROP COLUMN api_url, DROP COLUMN account_id, DROP COLUMN team_id, DROP COLUMN plan,
    DROP COLUMN sources, DROP COLUMN check_interval_minutes, DROP COLUMN last_check_at, DROP COLUMN dmarc_rua,
    DROP COLUMN last_sync_attempt_at, DROP COLUMN last_sync_error, DROP COLUMN token_rejected_at,
    DROP COLUMN alerts_since, DROP COLUMN updated_at;
DROP INDEX incidents_external_key;
ALTER TABLE incidents DROP COLUMN source, DROP COLUMN asset, DROP COLUMN external_id;
ALTER TABLE dmarc_reports DROP COLUMN via;
ALTER TABLE snds_daily DROP COLUMN via, DROP COLUMN jmrp_reports;
ALTER TABLE postmaster_daily DROP COLUMN via, DROP COLUMN fbl_flagged;
DROP INDEX blocklist_results_key;
DELETE FROM blocklist_results WHERE via = 'hostrepute';
ALTER TABLE blocklist_results DROP COLUMN via, DROP COLUMN rank, DROP COLUMN detail, DROP COLUMN delisting, DROP COLUMN delisting_requested_at;
CREATE UNIQUE INDEX blocklist_results_key ON blocklist_results (workspace_id, asset, source);
