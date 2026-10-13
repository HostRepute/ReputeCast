-- +goose Up
-- W10 · Connect a sending provider, senders and domains, system mail.

-- Sending domains keep the DNS records every sending provider needs
-- (dns_records, see internal/domains for the shape) with their live status.
-- These columns add what Settings › Senders shows next to them: a note
-- ("primary sending domain"), the custom link-tracking domain, where the zone
-- is hosted (from its NS records) and the detail of the last authentication
-- check (the SPF "all" mechanism, the DMARC policy, MAIL FROM hosts, BIMI).
ALTER TABLE sending_domains
    ADD COLUMN note            text        NOT NULL DEFAULT '',
    ADD COLUMN tracking_domain text        NOT NULL DEFAULT '',
    ADD COLUMN tracking_status text        NOT NULL DEFAULT ''
        CHECK (tracking_status IN ('', 'pending', 'verified', 'wrong_value', 'missing', 'error')),
    ADD COLUMN dns_host        text        NOT NULL DEFAULT '',
    ADD COLUMN nameservers     text[]      NOT NULL DEFAULT '{}',
    ADD COLUMN auth_detail     jsonb       NOT NULL DEFAULT '{}',
    ADD COLUMN created_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN updated_at      timestamptz NOT NULL DEFAULT now();
CREATE INDEX sending_domains_check_idx ON sending_domains (last_checked_at NULLS FIRST);

-- Sender identities: a note shown when the sender is no stream's default
-- ("personal sends"). senders.verification keeps one word per sending
-- provider id: verified, pending, failed, not_set_up, or not_used when a
-- person chose that this provider never sends as this address.
ALTER TABLE senders
    ADD COLUMN note       text        NOT NULL DEFAULT '',
    ADD COLUMN created_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();
CREATE INDEX senders_domain_idx ON senders (domain_id);

-- The default sender of each stream: default_sender_id (00002) is the
-- Broadcasts default ("Default From for new campaigns"); Automations and
-- System get theirs here. sending_defaults holds the workspace-wide sending
-- settings of Settings › Senders and Settings › Providers that are not
-- auto-pause thresholds (those stay in send_thresholds): reply handling, the
-- link tracking domain, throttles per mailbox provider, the suppression
-- sync interval and when the webhook signing secrets were last rotated.
ALTER TABLE workspaces
    ADD COLUMN default_automation_sender_id uuid  REFERENCES senders(id) ON DELETE SET NULL,
    ADD COLUMN default_system_sender_id     uuid  REFERENCES senders(id) ON DELETE SET NULL,
    ADD COLUMN sending_defaults             jsonb NOT NULL DEFAULT '{}';

-- Connect a sending provider: the setup wizard's progress, so a reload (or
-- a teammate) resumes at the same step. state holds what each step found
-- (credential check, provisioning progress, the domain, the test message,
-- streams, contacts). Secrets never go in state: they are sealed on the
-- provider row as soon as the provider accepted them.
CREATE TABLE provider_setups (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    adapter      text        NOT NULL,
    provider_id  uuid        REFERENCES providers(id) ON DELETE SET NULL,
    step         text        NOT NULL DEFAULT 'connection'
        CHECK (step IN ('connection', 'domain', 'test', 'streams', 'contacts', 'done')),
    status       text        NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'finished', 'abandoned')),
    state        jsonb       NOT NULL DEFAULT '{}',
    created_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    finished_at  timestamptz
);
CREATE INDEX provider_setups_open_idx ON provider_setups (workspace_id, adapter, updated_at DESC) WHERE status = 'open';
CREATE INDEX provider_setups_provider_idx ON provider_setups (provider_id) WHERE provider_id IS NOT NULL;

-- +goose Down
DROP TABLE provider_setups;
ALTER TABLE workspaces
    DROP COLUMN sending_defaults,
    DROP COLUMN default_system_sender_id,
    DROP COLUMN default_automation_sender_id;
DROP INDEX senders_domain_idx;
ALTER TABLE senders
    DROP COLUMN updated_at,
    DROP COLUMN created_by,
    DROP COLUMN note;
DROP INDEX sending_domains_check_idx;
ALTER TABLE sending_domains
    DROP COLUMN updated_at,
    DROP COLUMN created_by,
    DROP COLUMN auth_detail,
    DROP COLUMN nameservers,
    DROP COLUMN dns_host,
    DROP COLUMN tracking_status,
    DROP COLUMN tracking_domain,
    DROP COLUMN note;
