-- +goose Up
-- Payments (design expansion phase 6, M26): a workspace's payment provider
-- connections and the payments event log. Stripe connects with a restricted
-- key per mode (live and test), sealed and write-only, and its webhook is
-- registered automatically; every event is verified, stored raw, answered
-- at once and applied by a River job. Raw events are kept 90 days, like
-- provider events. Payment events never go to the audit log (ROADMAP's
-- default): this log, the paid members' timelines and Stripe's own logs are
-- their record.

CREATE TABLE payment_connections (
    id                    uuid PRIMARY KEY,
    workspace_id          uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    provider              text        NOT NULL DEFAULT 'stripe' CHECK (provider IN ('stripe')),
    mode                  text        NOT NULL CHECK (mode IN ('live','test')),
    -- setup: the connect wizard is under way; connected; disconnected (kept
    -- so the event log and paid members still name the account).
    status                text        NOT NULL DEFAULT 'setup' CHECK (status IN ('setup','connected','disconnected')),
    -- The wizard's step, so a reload resumes it: check, webhook, settings, done.
    step                  text        NOT NULL DEFAULT 'check' CHECK (step IN ('check','webhook','settings','done')),
    key_sealed            bytea,
    -- rk_live_…9fQ2: the prefix and the last four characters, for the masked field.
    key_hint              text        NOT NULL DEFAULT '',
    account_id            text        NOT NULL DEFAULT '',
    account_name          text        NOT NULL DEFAULT '',
    account_country       text        NOT NULL DEFAULT '',
    -- The account's default currency, lowercase ISO 4217 ("usd").
    currency              text        NOT NULL DEFAULT '',
    -- Stripe Tax is on in the account.
    tax_enabled           boolean     NOT NULL DEFAULT false,
    -- The 13 permission checks: [{"resource":"customers","access":"write","ok":true}].
    permissions           jsonb       NOT NULL DEFAULT '[]',
    checked_at            timestamptz,
    -- Stripe answered 401 for the key (rolled or deleted).
    key_rejected_at       timestamptz,
    -- The webhook: /hooks/stripe/{token}, its id in Stripe and its signing
    -- secret (sealed). manual: the secret was pasted by a person because
    -- the public URL can't be reached from the internet.
    webhook_token         text        UNIQUE,
    webhook_id            text        NOT NULL DEFAULT '',
    webhook_secret_sealed bytea,
    webhook_manual        boolean     NOT NULL DEFAULT false,
    webhook_events        text[]      NOT NULL DEFAULT '{}',
    webhook_registered_at timestamptz,
    -- The endpoint is gone in Stripe (seen by the hourly check).
    webhook_missing_at    timestamptz,
    -- The last reachability check of the public URL: {"ok":false,"code":"ECONNREFUSED","reason":"…","at":"…"}.
    webhook_reach         jsonb       NOT NULL DEFAULT '{}',
    first_event_at        timestamptz,
    first_event_type      text        NOT NULL DEFAULT '',
    last_event_at         timestamptz,
    last_event_type       text        NOT NULL DEFAULT '',
    -- The hourly gap check: when it last ran, what it applied, and the
    -- newest event it has seen; history keeps the last runs for the card.
    gap_checked_at        timestamptz,
    gap_history           jsonb       NOT NULL DEFAULT '[]',
    -- The customer portal ReputeCast configured in Stripe.
    portal_configuration  text        NOT NULL DEFAULT '',
    -- The demo's sample accounts answer from an in-process Stripe; nothing
    -- leaves the machine.
    demo                  boolean     NOT NULL DEFAULT false,
    connected_by          uuid        REFERENCES users(id) ON DELETE SET NULL,
    connected_at          timestamptz,
    disconnected_at       timestamptz,
    disconnected_by       uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_by            uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now()
);
-- One live and one test connection at a time.
CREATE UNIQUE INDEX payment_connections_mode ON payment_connections (workspace_id, provider, mode) WHERE status <> 'disconnected';

-- Every event a payment provider sent (or the gap check read), next to
-- what ReputeCast did with it. A replayed or duplicate event is stored
-- once (the provider's event id); an event whose signature failed is kept
-- for diagnosis only, cut short, and never applied.
CREATE TABLE payment_events (
    id                uuid PRIMARY KEY,
    workspace_id      uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    connection_id     uuid        REFERENCES payment_connections(id) ON DELETE SET NULL,
    mode              text        NOT NULL CHECK (mode IN ('live','test')),
    -- evt_…; empty for a payload whose signature failed.
    provider_event_id text        NOT NULL DEFAULT '',
    type              text        NOT NULL DEFAULT '',
    source            text        NOT NULL CHECK (source IN ('webhook','gap_check','history','manual')),
    -- When the provider created the event, and when it reached ReputeCast.
    occurred_at       timestamptz NOT NULL,
    received_at       timestamptz NOT NULL,
    payload           jsonb,
    signature_ok      boolean     NOT NULL DEFAULT true,
    status            text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','applied','ignored','failed','rejected')),
    -- What ReputeCast did: a key the event log words ("renewed",
    -- "sent_trial_ending") with its facts.
    outcome           text        NOT NULL DEFAULT '',
    outcome_detail    jsonb       NOT NULL DEFAULT '{}',
    contact_id        uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    subscription_id   uuid,
    amount_cents      bigint,
    currency          text        NOT NULL DEFAULT '',
    attempts          integer     NOT NULL DEFAULT 0,
    error             text        NOT NULL DEFAULT '',
    applied_at        timestamptz
);
CREATE UNIQUE INDEX payment_events_once ON payment_events (workspace_id, mode, provider_event_id) WHERE provider_event_id <> '';
CREATE INDEX payment_events_log_idx ON payment_events (workspace_id, received_at DESC);
CREATE INDEX payment_events_pending_idx ON payment_events (received_at) WHERE status = 'pending';
CREATE INDEX payment_events_contact_idx ON payment_events (contact_id, occurred_at DESC) WHERE contact_id IS NOT NULL;

-- +goose Down
DROP TABLE payment_events;
DROP TABLE payment_connections;
