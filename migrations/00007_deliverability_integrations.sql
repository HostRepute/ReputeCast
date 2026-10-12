-- +goose Up
-- Deliverability data (blocklists, Postmaster, SNDS, DMARC, incidents),
-- the HostRepute connection and outgoing webhooks.

CREATE TABLE blocklist_results (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    asset_type   text        NOT NULL CHECK (asset_type IN ('domain','ip')),
    asset        text        NOT NULL,
    source       text        NOT NULL,
    impact       text        NOT NULL DEFAULT 'low' CHECK (impact IN ('low','medium','high')),
    listed       boolean     NOT NULL,
    listed_since timestamptz,
    delist_url   text        NOT NULL DEFAULT '',
    checked_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX blocklist_results_key ON blocklist_results (workspace_id, asset, source);

CREATE TABLE postmaster_daily (
    workspace_id      uuid    NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    domain            text    NOT NULL,
    day               date    NOT NULL,
    spam_rate         numeric(6,4),
    domain_reputation text    NOT NULL DEFAULT '',
    ip_reputation     text    NOT NULL DEFAULT '',
    spf_rate          numeric(6,4),
    dkim_rate         numeric(6,4),
    dmarc_rate        numeric(6,4),
    tls_rate          numeric(6,4),
    delivery_errors   numeric(6,4),
    PRIMARY KEY (workspace_id, domain, day)
);

CREATE TABLE snds_daily (
    workspace_id   uuid    NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    ip             text    NOT NULL,
    day            date    NOT NULL,
    filter_result  text    NOT NULL DEFAULT '' CHECK (filter_result IN ('','green','yellow','red')),
    complaint_rate numeric(6,4),
    trap_hits      integer NOT NULL DEFAULT 0,
    messages       bigint  NOT NULL DEFAULT 0,
    PRIMARY KEY (workspace_id, ip, day)
);

CREATE TABLE dmarc_reports (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    domain       text        NOT NULL,
    org_name     text        NOT NULL,
    report_id    text        NOT NULL,
    date_begin   timestamptz NOT NULL,
    date_end     timestamptz NOT NULL,
    policy       jsonb       NOT NULL DEFAULT '{}',
    -- [{"source_ip":"…","count":120,"disposition":"none","dkim":"pass","spf":"pass","header_from":"…"}]
    records      jsonb       NOT NULL DEFAULT '[]',
    received_at  timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX dmarc_reports_key ON dmarc_reports (workspace_id, org_name, report_id);
CREATE INDEX dmarc_reports_domain_idx ON dmarc_reports (workspace_id, domain, date_end DESC);

-- Incident timeline linked to campaigns and providers.
CREATE TABLE incidents (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    occurred_at  timestamptz NOT NULL DEFAULT now(),
    kind         text        NOT NULL,
    severity     text        NOT NULL DEFAULT 'info' CHECK (severity IN ('info','ok','warn','bad')),
    title        text        NOT NULL,
    detail       text        NOT NULL DEFAULT '',
    campaign_id  uuid        REFERENCES campaigns(id) ON DELETE SET NULL,
    provider_id  uuid        REFERENCES providers(id) ON DELETE SET NULL,
    resolved_at  timestamptz
);
CREATE INDEX incidents_workspace_idx ON incidents (workspace_id, occurred_at DESC);

CREATE TABLE hostrepute_connections (
    workspace_id   uuid        PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    method         text        NOT NULL CHECK (method IN ('oauth','api_key')),
    account_email  text        NOT NULL DEFAULT '',
    team_name      text        NOT NULL DEFAULT '',
    scopes         text[]      NOT NULL DEFAULT '{}',
    token_enc      bytea       NOT NULL,
    refresh_enc    bytea,
    token_expires_at timestamptz,
    status         text        NOT NULL DEFAULT 'connected' CHECK (status IN ('connected','revoked','expired','limit_reached','error')),
    status_detail  text        NOT NULL DEFAULT '',
    -- [{"asset":"news.acme.io","type":"domain","status":"synced","last_sync_at":"…"}]
    monitored      jsonb       NOT NULL DEFAULT '[]',
    monitor_limit  integer,
    alerts_to_notifications boolean NOT NULL DEFAULT true,
    connected_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    connected_at   timestamptz NOT NULL DEFAULT now(),
    last_sync_at   timestamptz
);

CREATE TABLE webhooks (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name         text        NOT NULL DEFAULT '',
    url          text        NOT NULL,
    events       text[]      NOT NULL,
    secret_enc   bytea       NOT NULL,
    enabled      boolean     NOT NULL DEFAULT true,
    created_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_status  integer,
    last_delivery_at timestamptz,
    failing_since timestamptz
);
CREATE INDEX webhooks_workspace_idx ON webhooks (workspace_id);

CREATE TABLE webhook_deliveries (
    id            uuid PRIMARY KEY,
    webhook_id    uuid        NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event         text        NOT NULL,
    payload       jsonb       NOT NULL,
    attempt       integer     NOT NULL DEFAULT 1,
    status_code   integer,
    duration_ms   integer,
    response      text        NOT NULL DEFAULT '',
    error         text        NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX webhook_deliveries_webhook_idx ON webhook_deliveries (webhook_id, created_at DESC);

-- +goose Down
DROP TABLE webhook_deliveries;
DROP TABLE webhooks;
DROP TABLE hostrepute_connections;
DROP TABLE incidents;
DROP TABLE dmarc_reports;
DROP TABLE snds_daily;
DROP TABLE postmaster_daily;
DROP TABLE blocklist_results;
