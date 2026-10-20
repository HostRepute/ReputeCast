-- +goose Up
-- Revenue and attribution (design expansion phase 6, M27): the MRR ledger
-- behind Earn's numbers, the conversion API's log, and the click token's
-- settings.
--
-- MRR has one definition (ROADMAP default): the monthly value of every
-- Active and Past due subscription, after discounts, before tax, live
-- mode only; a yearly price counts as a twelfth; trials, free periods,
-- complimentary and gift access count nothing. The ledger keeps each
-- subscription's yearly value (twelve times its MRR, exact in minor units)
-- before and after every change, so the start of a period, plus new and
-- expansion, minus contraction and churn, is its end.

-- A subscription's yearly value under the MRR definition.
-- +goose StatementBegin
CREATE FUNCTION earn_arr_cents(kind text, mode text, status text, ended_at timestamptz, period text, amount bigint, discount jsonb)
RETURNS bigint LANGUAGE sql IMMUTABLE AS $$
    SELECT CASE WHEN kind = 'stripe' AND mode = 'live' AND ended_at IS NULL AND status IN ('active','past_due') THEN
        CASE WHEN period = 'year' THEN coalesce((discount->>'amountCents')::bigint, amount)
             ELSE coalesce((discount->>'amountCents')::bigint, amount) * 12 END
    ELSE 0 END
$$;
-- +goose StatementEnd

CREATE TABLE earn_mrr_changes (
    -- Written by the trigger below, so numbered by the database.
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    subscription_id uuid        REFERENCES paid_subscriptions(id) ON DELETE CASCADE,
    contact_id      uuid        REFERENCES contacts(id) ON DELETE CASCADE,
    tier_id         uuid        REFERENCES earn_tiers(id) ON DELETE SET NULL,
    period          text        NOT NULL DEFAULT '',
    at              timestamptz NOT NULL,
    -- new: first payment; reactivation: paid again within 90 days of
    -- ending; expansion: a higher tier or price; contraction: a lower
    -- price, a discount starting, monthly to yearly; churn: access ended.
    kind            text        NOT NULL CHECK (kind IN ('new','reactivation','expansion','contraction','churn')),
    arr_before      bigint      NOT NULL,
    arr_after       bigint      NOT NULL,
    -- event: a change ReputeCast applied; history: read from the provider's
    -- subscription history (before the move); import: a paid member
    -- imported with their subscription.
    source          text        NOT NULL DEFAULT 'event' CHECK (source IN ('event','history','import'))
);
CREATE INDEX earn_mrr_changes_ws_idx ON earn_mrr_changes (workspace_id, at);
CREATE INDEX earn_mrr_changes_contact_idx ON earn_mrr_changes (contact_id, at DESC);
CREATE INDEX earn_mrr_changes_sub_idx ON earn_mrr_changes (subscription_id);

-- Every change to a live subscription's value goes in the ledger, whatever
-- wrote it (webhook events, imports, cancellations, offers). Imported
-- subscriptions count from when they started paying.
-- +goose StatementBegin
CREATE FUNCTION earn_mrr_track() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    v_before bigint := 0;
    v_after  bigint;
    v_at     timestamptz;
    v_kind   text;
    v_source text := 'event';
BEGIN
    IF current_setting('reputecast.mrr_ledger', true) = 'off' THEN
        RETURN NULL;
    END IF;
    v_after := earn_arr_cents(NEW.kind, NEW.mode, NEW.status, NEW.ended_at, NEW.period, NEW.amount_cents, NEW.discount);
    IF TG_OP = 'UPDATE' THEN
        v_before := earn_arr_cents(OLD.kind, OLD.mode, OLD.status, OLD.ended_at, OLD.period, OLD.amount_cents, OLD.discount);
    END IF;
    IF v_before = v_after THEN
        RETURN NULL;
    END IF;
    IF TG_OP = 'INSERT' THEN
        v_at := least(coalesce(NEW.trial_end, NEW.started_at), NEW.created_at);
        IF NEW.import_id IS NOT NULL THEN
            v_source := 'import';
        END IF;
    ELSE
        v_at := coalesce(NEW.updated_at, now());
    END IF;
    IF v_before = 0 THEN
        IF EXISTS (SELECT 1 FROM earn_mrr_changes c WHERE c.workspace_id = NEW.workspace_id AND c.contact_id = NEW.contact_id
                   AND c.kind = 'churn' AND c.at <= v_at AND c.at >= v_at - interval '90 days') THEN
            v_kind := 'reactivation';
        ELSE
            v_kind := 'new';
        END IF;
    ELSIF v_after = 0 THEN
        v_kind := CASE WHEN NEW.ended_at IS NOT NULL THEN 'churn' ELSE 'contraction' END;
    ELSIF v_after > v_before THEN
        v_kind := 'expansion';
    ELSE
        v_kind := 'contraction';
    END IF;
    INSERT INTO earn_mrr_changes (workspace_id, subscription_id, contact_id, tier_id, period, at, kind, arr_before, arr_after, source)
    VALUES (NEW.workspace_id, NEW.id, NEW.contact_id, NEW.tier_id, NEW.period, v_at, v_kind, v_before, v_after, v_source);
    RETURN NULL;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER paid_subscriptions_mrr AFTER INSERT OR UPDATE ON paid_subscriptions
    FOR EACH ROW EXECUTE FUNCTION earn_mrr_track();

-- Subscriptions paying before the ledger existed count from when they
-- started paying.
INSERT INTO earn_mrr_changes (workspace_id, subscription_id, contact_id, tier_id, period, at, kind, arr_before, arr_after, source)
SELECT workspace_id, id, contact_id, tier_id, period, least(coalesce(trial_end, started_at), created_at), 'new', 0,
       earn_arr_cents(kind, mode, status, ended_at, period, amount_cents, discount), CASE WHEN import_id IS NOT NULL THEN 'import' ELSE 'event' END
FROM paid_subscriptions
WHERE earn_arr_cents(kind, mode, status, ended_at, period, amount_cents, discount) > 0;

-- Per workspace: when the subscription history before the move was read
-- from the provider (once per live connection).
CREATE TABLE earn_mrr_history (
    workspace_id  uuid        PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    connection_id uuid,
    built_at      timestamptz NOT NULL,
    subscriptions integer     NOT NULL DEFAULT 0
);

-- The click token: off by default; when on, the click redirect adds
-- rc_click=… to links to these domains only (never to sponsor links), so
-- the conversion API can credit the email that brought the click.
ALTER TABLE earn_settings
    ADD COLUMN click_token_on      boolean NOT NULL DEFAULT false,
    ADD COLUMN click_token_domains text[]  NOT NULL DEFAULT '{}';

-- Human clicks on upgrade links, per campaign (the report's Revenue
-- section). Bot clicks never count.
CREATE TABLE earn_campaign_clicks (
    campaign_id    uuid   PRIMARY KEY REFERENCES campaigns(id) ON DELETE CASCADE,
    workspace_id   uuid   NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    upgrade_clicks bigint NOT NULL DEFAULT 0
);

-- Conversions recorded through the API: a valuable action in the
-- publisher's own product, with a value, credited to the email clicked
-- within the attribution window (by click token, else by the contact's
-- address). Kept here, not in the audit log; the address itself isn't
-- stored, only the contact it matched.
CREATE TABLE conversions (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    api_key_id      uuid        REFERENCES api_keys(id) ON DELETE SET NULL,
    key_name        text        NOT NULL DEFAULT '',
    name            text        NOT NULL,
    -- The value as sent ("29.00") and in minor units of its currency.
    value           text        NOT NULL,
    value_cents     bigint      NOT NULL,
    currency        text        NOT NULL,
    occurred_at     timestamptz NOT NULL,
    received_at     timestamptz NOT NULL,
    contact_id      uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    matched         text        NOT NULL CHECK (matched IN ('click_token','email','none')),
    message_id      uuid,
    campaign_id     uuid        REFERENCES campaigns(id) ON DELETE SET NULL,
    automation_id   uuid        REFERENCES automations(id) ON DELETE SET NULL,
    clicked_at      timestamptz,
    metadata        jsonb       NOT NULL DEFAULT '{}',
    idempotency_key text        NOT NULL DEFAULT '',
    request_hash    text        NOT NULL DEFAULT '',
    -- A test conversion from Earn › Settings › Attribution: shown in the
    -- log, left out of every number.
    test            boolean     NOT NULL DEFAULT false
);
CREATE INDEX conversions_ws_idx ON conversions (workspace_id, received_at DESC);
CREATE INDEX conversions_campaign_idx ON conversions (campaign_id) WHERE campaign_id IS NOT NULL;
CREATE UNIQUE INDEX conversions_idem ON conversions (workspace_id, api_key_id, idempotency_key) WHERE idempotency_key <> '';

-- +goose Down
DROP TABLE conversions;
DROP TABLE earn_campaign_clicks;
ALTER TABLE earn_settings DROP COLUMN click_token_domains, DROP COLUMN click_token_on;
DROP TABLE earn_mrr_history;
DROP TRIGGER paid_subscriptions_mrr ON paid_subscriptions;
DROP FUNCTION earn_mrr_track();
DROP TABLE earn_mrr_changes;
DROP FUNCTION earn_arr_cents(text, text, text, timestamptz, text, bigint, jsonb);
