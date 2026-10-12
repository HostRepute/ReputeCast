-- +goose Up
-- Sending domains and senders, sending providers, routing, warm-up, raw
-- provider events and the shared suppression list.

CREATE TABLE sending_domains (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    domain          text        NOT NULL CHECK (domain = lower(domain)),
    -- Required DNS records per provider with their live status:
    -- [{"provider_id":"…","type":"CNAME","name":"…","value":"…","purpose":"dkim","status":"verified","checked_at":"…"}]
    dns_records     jsonb       NOT NULL DEFAULT '[]',
    -- Authentication summary: {"spf":"pass","dkim":"pass","dmarc":"quarantine","bimi":"missing","return_path":"aligned"}
    auth_summary    jsonb       NOT NULL DEFAULT '{}',
    status          text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','verified','partial','failed')),
    last_checked_at timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX sending_domains_key ON sending_domains (workspace_id, domain);

CREATE TABLE senders (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    domain_id    uuid        NOT NULL REFERENCES sending_domains(id) ON DELETE CASCADE,
    name         text        NOT NULL,
    email        text        NOT NULL CHECK (email = lower(email)),
    reply_to     text        NOT NULL DEFAULT '',
    -- Verification per sending provider: {"<provider_id>":"verified"}
    verification jsonb       NOT NULL DEFAULT '{}',
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX senders_key ON senders (workspace_id, email);

ALTER TABLE workspaces ADD CONSTRAINT workspaces_default_sender_fk
    FOREIGN KEY (default_sender_id) REFERENCES senders(id) ON DELETE SET NULL;

-- One row per connected sending provider. The adapter is the code that talks
-- to it (internal/providers/<adapter>); credentials are write-only and
-- encrypted with the instance secret key.
CREATE TABLE providers (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    adapter         text        NOT NULL,
    name            text        NOT NULL,
    label           text        NOT NULL DEFAULT '',
    -- Non-secret connection settings (region, server URL, message stream, host, port…).
    config          jsonb       NOT NULL DEFAULT '{}',
    credentials_enc bytea,
    -- Token embedded in the webhook URL (/hooks/<adapter>/<token>) plus an
    -- optional signing secret, both generated at connect time.
    webhook_token   text        NOT NULL,
    webhook_secret_enc bytea,
    status          text        NOT NULL DEFAULT 'healthy' CHECK (status IN ('healthy','degraded','down','sandbox','warming_up','paused')),
    status_detail   text        NOT NULL DEFAULT '',
    ip_mode         text        NOT NULL DEFAULT 'shared' CHECK (ip_mode IN ('shared','dedicated')),
    ips             text[]      NOT NULL DEFAULT '{}',
    -- Capability snapshot from the adapter plus connection specifics.
    capabilities    jsonb       NOT NULL DEFAULT '{}',
    -- Limits used by health and auto-pause: {"bounce":{"limit":5,"review":5,"pause":10},"complaint":{…}}
    limits          jsonb       NOT NULL DEFAULT '{}',
    -- Latest health snapshot written by the health job: quota, rate, error rate, account status…
    health          jsonb       NOT NULL DEFAULT '{}',
    -- Pricing: {"model":"metered","per_1000":0.10} or {"model":"plan","monthly":15} or {"model":"self_hosted","monthly":24}
    pricing         jsonb       NOT NULL DEFAULT '{}',
    -- Provider-side tracking found by the tracking check: {"open_pixel":false,"link_rewrite":true}
    tracking        jsonb       NOT NULL DEFAULT '{}',
    last_event_at   timestamptz,
    last_checked_at timestamptz,
    connected_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    connected_at    timestamptz NOT NULL DEFAULT now(),
    disconnected_at timestamptz
);
CREATE INDEX providers_workspace_idx ON providers (workspace_id) WHERE disconnected_at IS NULL;
CREATE UNIQUE INDEX providers_webhook_token_key ON providers (webhook_token);

-- Daily health samples per provider for trends and the 7-day error rate.
CREATE TABLE provider_daily_stats (
    provider_id  uuid    NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    day          date    NOT NULL,
    sent         bigint  NOT NULL DEFAULT 0,
    delivered    bigint  NOT NULL DEFAULT 0,
    bounced      bigint  NOT NULL DEFAULT 0,
    complained   bigint  NOT NULL DEFAULT 0,
    api_calls    bigint  NOT NULL DEFAULT 0,
    api_errors   bigint  NOT NULL DEFAULT 0,
    cost_micros  bigint  NOT NULL DEFAULT 0,
    PRIMARY KEY (provider_id, day)
);

-- Routing: one rule set per stream. Rules are evaluated by priority; the
-- first whose conditions match decides the weighted split.
CREATE TABLE routing_rules (
    id                   uuid PRIMARY KEY,
    workspace_id         uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    stream               text        NOT NULL CHECK (stream IN ('broadcast','automation','system')),
    priority             integer     NOT NULL DEFAULT 100,
    name                 text        NOT NULL DEFAULT '',
    -- {"lists":[…],"sender_domains":[…],"mailbox_providers":["outlook"]}; empty = default rule.
    conditions           jsonb       NOT NULL DEFAULT '{}',
    -- [{"provider_id":"…","weight":90},{"provider_id":"…","weight":10}]
    splits               jsonb       NOT NULL,
    fallback_provider_id uuid        REFERENCES providers(id) ON DELETE SET NULL,
    enabled              boolean     NOT NULL DEFAULT true,
    updated_by           uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at           timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX routing_rules_ws_idx ON routing_rules (workspace_id, stream, priority);

-- Failover policy per stream.
CREATE TABLE stream_policies (
    workspace_id    uuid    NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    stream          text    NOT NULL CHECK (stream IN ('broadcast','automation','system')),
    failover        text    NOT NULL CHECK (failover IN ('automatic','pause_and_ask','warm_only')),
    PRIMARY KEY (workspace_id, stream)
);

CREATE TABLE warmup_plans (
    id                 uuid PRIMARY KEY,
    provider_id        uuid        NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    ip                 text        NOT NULL DEFAULT '',
    start_date         date        NOT NULL,
    days               integer     NOT NULL CHECK (days BETWEEN 1 AND 120),
    -- [{"day":1,"volume":200},…]
    schedule           jsonb       NOT NULL,
    engaged_first      boolean     NOT NULL DEFAULT true,
    slowdown_bounce    numeric(5,2) NOT NULL DEFAULT 2.0,
    slowdown_complaint numeric(5,3) NOT NULL DEFAULT 0.1,
    status             text        NOT NULL DEFAULT 'active' CHECK (status IN ('active','slowed','paused','done')),
    status_detail      text        NOT NULL DEFAULT '',
    -- Actual results per day: [{"day":1,"sent":200,"bounce":0.5,"complaint":0.0,"slowed":false}]
    progress           jsonb       NOT NULL DEFAULT '[]',
    created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX warmup_plans_provider_key ON warmup_plans (provider_id) WHERE status <> 'done';

-- Webhook ingest stores the raw payload first and answers 200; a River job
-- normalizes it into delivery_events later.
CREATE TABLE provider_raw_events (
    id                uuid PRIMARY KEY,
    workspace_id      uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    provider_id       uuid        NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    received_at       timestamptz NOT NULL DEFAULT now(),
    content_type      text        NOT NULL DEFAULT '',
    body              bytea       NOT NULL,
    signature_ok      boolean     NOT NULL,
    normalized_at     timestamptz,
    normalize_error   text
);
CREATE INDEX provider_raw_events_pending_idx ON provider_raw_events (received_at) WHERE normalized_at IS NULL;
CREATE INDEX provider_raw_events_provider_idx ON provider_raw_events (provider_id, received_at DESC);

-- One suppression list per workspace, shared by every provider.
CREATE TABLE suppressions (
    id                 uuid PRIMARY KEY,
    workspace_id       uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email              text        NOT NULL CHECK (email = lower(email)),
    reason             text        NOT NULL CHECK (reason IN ('hard_bounce','complaint','unsubscribe','manual','imported','role','spam_trap','invalid')),
    source             text        NOT NULL DEFAULT '',
    source_provider_id uuid        REFERENCES providers(id) ON DELETE SET NULL,
    original_code      text        NOT NULL DEFAULT '',
    detail             text        NOT NULL DEFAULT '',
    created_by         uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX suppressions_key ON suppressions (workspace_id, email);
CREATE INDEX suppressions_created_idx ON suppressions (workspace_id, created_at DESC);

-- Suppression sync state per provider (push our list, pull theirs).
CREATE TABLE provider_suppression_sync (
    provider_id    uuid        PRIMARY KEY REFERENCES providers(id) ON DELETE CASCADE,
    mode           text        NOT NULL DEFAULT 'two_way' CHECK (mode IN ('two_way','push','pull','off')),
    last_push_at   timestamptz,
    last_pull_at   timestamptz,
    pushed         bigint      NOT NULL DEFAULT 0,
    pulled         bigint      NOT NULL DEFAULT 0,
    pending        bigint      NOT NULL DEFAULT 0,
    status         text        NOT NULL DEFAULT 'ok',
    status_detail  text        NOT NULL DEFAULT ''
);

-- Community adapters installed on this instance (built-ins come from code).
CREATE TABLE installed_adapters (
    id            uuid PRIMARY KEY,
    name          text        NOT NULL,
    kind          text        NOT NULL CHECK (kind IN ('http','wasm')),
    version       text        NOT NULL,
    maintainer    text        NOT NULL DEFAULT '',
    source_url    text        NOT NULL DEFAULT '',
    manifest      jsonb       NOT NULL DEFAULT '{}',
    latest_version text,
    installed_at  timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX installed_adapters_name_key ON installed_adapters (name);

-- +goose Down
DROP TABLE installed_adapters;
DROP TABLE provider_suppression_sync;
DROP TABLE suppressions;
DROP TABLE provider_raw_events;
DROP TABLE warmup_plans;
DROP TABLE stream_policies;
DROP TABLE routing_rules;
DROP TABLE provider_daily_stats;
DROP TABLE providers;
ALTER TABLE workspaces DROP CONSTRAINT workspaces_default_sender_fk;
DROP TABLE senders;
DROP TABLE sending_domains;
