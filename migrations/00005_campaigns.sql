-- +goose Up
-- Templates, campaigns, versions, approvals, readiness, messages, normalized
-- delivery events, live counters, incidents (auto-pauses) and report sharing.

CREATE TABLE templates (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    kind         text        NOT NULL DEFAULT 'template' CHECK (kind IN ('template','section')),
    name         text        NOT NULL,
    description  text        NOT NULL DEFAULT '',
    mjml         text        NOT NULL,
    html         text        NOT NULL DEFAULT '',
    created_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX templates_workspace_idx ON templates (workspace_id, kind);

CREATE TABLE campaigns (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name            text        NOT NULL,
    status          text        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','approval','approved','scheduled','sending','paused','sent','cancelled','failed')),
    subject         text        NOT NULL DEFAULT '',
    preheader       text        NOT NULL DEFAULT '',
    sender_id       uuid        REFERENCES senders(id) ON DELETE SET NULL,
    reply_to        text        NOT NULL DEFAULT '',
    -- {"lists":[…],"segments":[…],"exclude_lists":[…],"exclude_segments":[…],"exclude_tiers":["sunset"]}
    audience        jsonb       NOT NULL DEFAULT '{}',
    audience_count  bigint,
    editor_mode     text        NOT NULL DEFAULT 'visual' CHECK (editor_mode IN ('visual','mjml','html')),
    mjml            text        NOT NULL DEFAULT '',
    html            text        NOT NULL DEFAULT '',
    plain_text      text        NOT NULL DEFAULT '',
    version         integer     NOT NULL DEFAULT 1,
    -- A/B test: {"kind":"subject","variants":[{"key":"A","subject":"…"},{"key":"B","subject":"…"}],"test_percent":20,"metric":"clicks","duration_hours":4}
    ab_test         jsonb,
    -- {"mode":"now"|"at"|"best_time","at":"…","timezone":"America/New_York"}
    schedule        jsonb       NOT NULL DEFAULT '{"mode":"now"}',
    -- Auto-pause thresholds: {"bounce":2.0,"complaint":0.1,"scope":"route"}
    thresholds      jsonb       NOT NULL DEFAULT '{}',
    route_override  jsonb,
    tracking        jsonb       NOT NULL DEFAULT '{"opens":true,"clicks":true}',
    created_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    autosaved_at    timestamptz,
    scheduled_at    timestamptz,
    started_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    started_at      timestamptz,
    completed_at    timestamptz,
    deleted_at      timestamptz
);
CREATE INDEX campaigns_workspace_idx ON campaigns (workspace_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX campaigns_status_idx ON campaigns (workspace_id, status) WHERE deleted_at IS NULL;

CREATE TABLE campaign_versions (
    campaign_id uuid        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    version     integer     NOT NULL,
    subject     text        NOT NULL,
    preheader   text        NOT NULL,
    mjml        text        NOT NULL,
    html        text        NOT NULL,
    saved_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    saved_at    timestamptz NOT NULL DEFAULT now(),
    label       text        NOT NULL DEFAULT '',
    PRIMARY KEY (campaign_id, version)
);

-- Send approvals (A5.12). The snapshot is what the approval covers.
CREATE TABLE approval_requests (
    id             uuid PRIMARY KEY,
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    subject_type   text        NOT NULL DEFAULT 'campaign' CHECK (subject_type IN ('campaign','automation')),
    campaign_id    uuid        REFERENCES campaigns(id) ON DELETE CASCADE,
    automation_id  uuid,
    requested_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    requested_at   timestamptz NOT NULL DEFAULT now(),
    note           text        NOT NULL DEFAULT '',
    status         text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','changes_requested','rejected','invalidated','bypassed','sent')),
    matched_rules  text[]      NOT NULL DEFAULT '{}',
    approvals_required integer NOT NULL DEFAULT 1,
    -- {"content_hash":"…","subject":"…","preheader":"…","audience":{…},"audience_count":47912,"route":[…],"schedule":{…},"readiness":{…},"cost":{…}}
    snapshot       jsonb       NOT NULL,
    invalidated_at timestamptz,
    invalidated_reason text    NOT NULL DEFAULT '',
    -- What changed versus the snapshot when invalidated: [{"field":"subject","before":"…","after":"…"}]
    changes        jsonb       NOT NULL DEFAULT '[]',
    bypass_reason  text        NOT NULL DEFAULT '',
    decided_at     timestamptz
);
CREATE INDEX approval_requests_ws_idx ON approval_requests (workspace_id, status, requested_at DESC);
CREATE INDEX approval_requests_campaign_idx ON approval_requests (campaign_id);

CREATE TABLE approval_decisions (
    id          uuid PRIMARY KEY,
    request_id  uuid        NOT NULL REFERENCES approval_requests(id) ON DELETE CASCADE,
    approver_id uuid        REFERENCES users(id) ON DELETE SET NULL,
    decision    text        NOT NULL CHECK (decision IN ('approve','request_changes','reject')),
    comment     text        NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX approval_decisions_request_idx ON approval_decisions (request_id);

CREATE TABLE approval_comments (
    id         uuid PRIMARY KEY,
    request_id uuid        NOT NULL REFERENCES approval_requests(id) ON DELETE CASCADE,
    author_id  uuid        REFERENCES users(id) ON DELETE SET NULL,
    body       text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX approval_comments_request_idx ON approval_comments (request_id, created_at);

-- Send Readiness results. Each run keeps its checks, verdict and estimates.
CREATE TABLE readiness_runs (
    id          uuid PRIMARY KEY,
    campaign_id uuid        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    trigger     text        NOT NULL DEFAULT 'manual' CHECK (trigger IN ('manual','composer','schedule','send_time','approval')),
    verdict     text        NOT NULL CHECK (verdict IN ('ready','warnings','blocked')),
    checks      jsonb       NOT NULL,
    estimates   jsonb       NOT NULL DEFAULT '{}',
    ran_at      timestamptz NOT NULL DEFAULT now(),
    duration_ms integer     NOT NULL DEFAULT 0
);
CREATE INDEX readiness_runs_campaign_idx ON readiness_runs (campaign_id, ran_at DESC);

-- Per-campaign, per-provider share of the send (route breakdown and live bars).
CREATE TABLE campaign_routes (
    campaign_id  uuid        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    provider_id  uuid        NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    planned      bigint      NOT NULL DEFAULT 0,
    status       text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','sending','held','paused','done','cancelled')),
    status_detail text       NOT NULL DEFAULT '',
    cost_micros  bigint      NOT NULL DEFAULT 0,
    PRIMARY KEY (campaign_id, provider_id)
);

-- One row per recipient of a campaign or automation email.
CREATE TABLE messages (
    id                  uuid PRIMARY KEY,
    workspace_id        uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    stream              text        NOT NULL CHECK (stream IN ('broadcast','automation','system')),
    campaign_id         uuid        REFERENCES campaigns(id) ON DELETE CASCADE,
    automation_id       uuid,
    automation_node     text,
    contact_id          uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    email               text        NOT NULL,
    variant             text        NOT NULL DEFAULT '',
    provider_id         uuid        REFERENCES providers(id) ON DELETE SET NULL,
    provider_message_id text,
    mailbox_provider    text        NOT NULL DEFAULT 'other',
    status              text        NOT NULL DEFAULT 'queued' CHECK (status IN ('queued','held','sent','delivered','deferred','bounced','complained','rejected','failed','cancelled')),
    queued_at           timestamptz NOT NULL DEFAULT now(),
    sent_at             timestamptz,
    delivered_at        timestamptz,
    opened_at           timestamptz,
    clicked_at          timestamptz,
    error               text        NOT NULL DEFAULT ''
);
CREATE INDEX messages_campaign_idx ON messages (campaign_id, status);
CREATE INDEX messages_contact_idx ON messages (contact_id, queued_at DESC);
CREATE UNIQUE INDEX messages_provider_msg_key ON messages (provider_id, provider_message_id) WHERE provider_message_id IS NOT NULL;
CREATE INDEX messages_queued_idx ON messages (campaign_id, provider_id) WHERE status IN ('queued','held');

-- Normalized delivery and engagement events. Every provider's event maps to
-- one type and keeps the provider's original code.
CREATE TABLE delivery_events (
    id               uuid PRIMARY KEY,
    workspace_id     uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    message_id       uuid        REFERENCES messages(id) ON DELETE SET NULL,
    campaign_id      uuid,
    contact_id       uuid,
    provider_id      uuid        REFERENCES providers(id) ON DELETE SET NULL,
    raw_event_id     uuid        REFERENCES provider_raw_events(id) ON DELETE SET NULL,
    type             text        NOT NULL CHECK (type IN ('sent','delivered','deferred','hard_bounce','soft_bounce','complaint','rejected','open','click','unsubscribe')),
    occurred_at      timestamptz NOT NULL,
    mailbox_provider text        NOT NULL DEFAULT 'other',
    -- Original provider event name and diagnostic code, e.g. "Bounce/Permanent/General", "550 5.1.1".
    original_type    text        NOT NULL DEFAULT '',
    original_code    text        NOT NULL DEFAULT '',
    diagnostic       text        NOT NULL DEFAULT '',
    -- Bounce taxonomy: mailbox_unknown, mailbox_full, domain_invalid, policy_block, spam_block, reputation, temporary, other
    bounce_class     text        NOT NULL DEFAULT '',
    machine          boolean     NOT NULL DEFAULT false,
    url              text        NOT NULL DEFAULT '',
    user_agent       text        NOT NULL DEFAULT '',
    ip               inet
);
CREATE INDEX delivery_events_campaign_idx ON delivery_events (campaign_id, type, occurred_at);
CREATE INDEX delivery_events_contact_idx ON delivery_events (contact_id, occurred_at DESC);
CREATE INDEX delivery_events_provider_idx ON delivery_events (provider_id, occurred_at DESC);
CREATE INDEX delivery_events_occurred_brin ON delivery_events USING brin (occurred_at);

-- Live and final counters per campaign × sending provider × mailbox provider.
CREATE TABLE campaign_counters (
    campaign_id      uuid    NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    provider_id      uuid    NOT NULL,
    mailbox_provider text    NOT NULL,
    variant          text    NOT NULL DEFAULT '',
    sent             bigint  NOT NULL DEFAULT 0,
    delivered        bigint  NOT NULL DEFAULT 0,
    deferred         bigint  NOT NULL DEFAULT 0,
    hard_bounced     bigint  NOT NULL DEFAULT 0,
    soft_bounced     bigint  NOT NULL DEFAULT 0,
    complained       bigint  NOT NULL DEFAULT 0,
    rejected         bigint  NOT NULL DEFAULT 0,
    opened_human     bigint  NOT NULL DEFAULT 0,
    opened_machine   bigint  NOT NULL DEFAULT 0,
    clicked          bigint  NOT NULL DEFAULT 0,
    unsubscribed     bigint  NOT NULL DEFAULT 0,
    PRIMARY KEY (campaign_id, provider_id, mailbox_provider, variant)
);

-- Hourly engagement for the first 72 hours (report chart).
CREATE TABLE campaign_hourly (
    campaign_id    uuid    NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    hour           integer NOT NULL CHECK (hour BETWEEN 0 AND 720),
    opened_human   bigint  NOT NULL DEFAULT 0,
    opened_machine bigint  NOT NULL DEFAULT 0,
    clicked        bigint  NOT NULL DEFAULT 0,
    PRIMARY KEY (campaign_id, hour)
);

CREATE TABLE campaign_links (
    id           uuid PRIMARY KEY,
    campaign_id  uuid        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    url          text        NOT NULL,
    label        text        NOT NULL DEFAULT '',
    position     integer     NOT NULL DEFAULT 0,
    clicks       bigint      NOT NULL DEFAULT 0,
    unique_clicks bigint     NOT NULL DEFAULT 0
);
CREATE INDEX campaign_links_campaign_idx ON campaign_links (campaign_id, position);

-- Auto-pauses and outages during a send (guarded sending).
CREATE TABLE campaign_incidents (
    id           uuid PRIMARY KEY,
    campaign_id  uuid        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    kind         text        NOT NULL CHECK (kind IN ('threshold','provider_outage','throttled','recheck_blocked')),
    scope        text        NOT NULL DEFAULT 'campaign' CHECK (scope IN ('campaign','route','mailbox_provider')),
    provider_id  uuid        REFERENCES providers(id) ON DELETE SET NULL,
    mailbox_provider text    NOT NULL DEFAULT '',
    metric       text        NOT NULL DEFAULT '',
    value        numeric(8,4),
    threshold    numeric(8,4),
    held         bigint      NOT NULL DEFAULT 0,
    detail       jsonb       NOT NULL DEFAULT '{}',
    tripped_at   timestamptz NOT NULL DEFAULT now(),
    resolved_at  timestamptz,
    resolution   text        NOT NULL DEFAULT '' CHECK (resolution IN ('','resumed','resumed_with_changes','cancelled','moved','retried','auto_recovered')),
    resolved_by  uuid        REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX campaign_incidents_campaign_idx ON campaign_incidents (campaign_id, tripped_at DESC);

CREATE TABLE report_shares (
    id          uuid PRIMARY KEY,
    campaign_id uuid        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    token_hash  bytea       NOT NULL,
    created_by  uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz,
    revoked_at  timestamptz
);
CREATE UNIQUE INDEX report_shares_token_key ON report_shares (token_hash);

CREATE TABLE unsubscribe_feedback (
    id          uuid PRIMARY KEY,
    workspace_id uuid       NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    contact_id  uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    campaign_id uuid        REFERENCES campaigns(id) ON DELETE SET NULL,
    reason      text        NOT NULL CHECK (reason IN ('too_frequent','not_relevant','never_signed_up','spam','other','one_click')),
    comment     text        NOT NULL DEFAULT '',
    chose_fewer boolean     NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX unsubscribe_feedback_campaign_idx ON unsubscribe_feedback (campaign_id);

-- +goose Down
DROP TABLE unsubscribe_feedback;
DROP TABLE report_shares;
DROP TABLE campaign_incidents;
DROP TABLE campaign_links;
DROP TABLE campaign_hourly;
DROP TABLE campaign_counters;
DROP TABLE delivery_events;
DROP TABLE messages;
DROP TABLE campaign_routes;
DROP TABLE readiness_runs;
DROP TABLE approval_comments;
DROP TABLE approval_decisions;
DROP TABLE approval_requests;
DROP TABLE campaign_versions;
DROP TABLE campaigns;
DROP TABLE templates;
