-- +goose Up
-- Paid subscriptions (design expansion phase 6, M26): Earn's settings,
-- tiers and their prices, every contact's paid status with its timeline,
-- checkout sessions, the upgrade page's visits and the eight member emails.
-- Stripe holds the money; these tables mirror it. No card details, bank
-- details or billing addresses are ever stored: only Stripe's ids, the
-- tier, price, status, dates, discounts, the amounts paid and refunded
-- (lifetime value) and the billing email.

CREATE TABLE earn_settings (
    workspace_id        uuid        PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    -- The workspace currency: the account's default, fixed once a tier has prices.
    currency            text        NOT NULL DEFAULT '',
    tax_at_checkout     boolean     NOT NULL DEFAULT false,
    prices_include_tax  boolean     NOT NULL DEFAULT false,
    -- What the customer portal lets readers do:
    -- {"update_card":true,"invoices":true,"billing_email":true,"switch_period":true,"cancel":true}.
    portal              jsonb       NOT NULL DEFAULT '{}',
    -- When a payment fails: keep access while Stripe retries, or end it at once.
    on_payment_failure  text        NOT NULL DEFAULT 'keep' CHECK (on_payment_failure IN ('keep','end')),
    -- Shown in the footer of the upgrade, cancel and gift pages.
    terms_url           text        NOT NULL DEFAULT '',
    support_email       text        NOT NULL DEFAULT '',
    -- The upgrade page as readers see it (headline, intro, tiers and their
    -- order, default period, highlighted tier, free card, FAQ, welcome text).
    upgrade_page        jsonb       NOT NULL DEFAULT '{}',
    upgrade_published_at timestamptz,
    upgrade_published_by uuid       REFERENCES users(id) ON DELETE SET NULL,
    -- How long after an email click an upgrade is credited to it (X17).
    attribution_days    integer     NOT NULL DEFAULT 7 CHECK (attribution_days BETWEEN 1 AND 30),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    updated_by          uuid        REFERENCES users(id) ON DELETE SET NULL
);

CREATE TABLE earn_tiers (
    id                  uuid PRIMARY KEY,
    workspace_id        uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name                text        NOT NULL,
    -- The hidden tier's own link: /upgrade/{slug}.
    slug                text        NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    description         text        NOT NULL DEFAULT '',
    benefits            text[]      NOT NULL DEFAULT '{}',
    trial_days          integer     NOT NULL DEFAULT 0 CHECK (trial_days IN (0, 7, 14, 30)),
    -- The lists its paid members join.
    list_ids            uuid[]      NOT NULL DEFAULT '{}',
    -- A higher tier includes a lower tier's paid content.
    access_rank         integer     NOT NULL DEFAULT 1 CHECK (access_rank BETWEEN 1 AND 20),
    visibility          text        NOT NULL DEFAULT 'public' CHECK (visibility IN ('public','hidden','archived')),
    -- The product in Stripe, per mode, and how saving it there went:
    -- [{"step":"product","status":"done","at":"…","id":"prod_…"},{"step":"year","status":"failed","error":"503 …"}].
    stripe_product_live text        NOT NULL DEFAULT '',
    stripe_product_test text        NOT NULL DEFAULT '',
    sync_status         text        NOT NULL DEFAULT 'draft' CHECK (sync_status IN ('draft','synced','failed')),
    sync_steps          jsonb       NOT NULL DEFAULT '[]',
    synced_at           timestamptz,
    -- "By Substack in Acme Media · mapped at the import on 18 Sep".
    origin              text        NOT NULL DEFAULT '',
    created_by          uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    archived_at         timestamptz,
    deleted_at          timestamptz
);
CREATE UNIQUE INDEX earn_tiers_slug ON earn_tiers (workspace_id, slug) WHERE deleted_at IS NULL;

-- A tier's prices. Stripe prices can't be edited: a new amount is a new
-- price, and the old one becomes a legacy price that current paid members
-- keep paying while new ones get the current price.
CREATE TABLE earn_prices (
    id             uuid PRIMARY KEY,
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    tier_id        uuid        NOT NULL REFERENCES earn_tiers(id) ON DELETE CASCADE,
    period         text        NOT NULL CHECK (period IN ('month','year')),
    amount_cents   bigint      NOT NULL CHECK (amount_cents >= 0),
    currency       text        NOT NULL,
    status         text        NOT NULL DEFAULT 'current' CHECK (status IN ('current','legacy')),
    stripe_live_id text        NOT NULL DEFAULT '',
    stripe_test_id text        NOT NULL DEFAULT '',
    -- tier: made by saving the tier; mapped: an existing Stripe price
    -- (from an import or the sync) mapped to the tier.
    origin         text        NOT NULL DEFAULT 'tier' CHECK (origin IN ('tier','mapped')),
    created_at     timestamptz NOT NULL DEFAULT now(),
    retired_at     timestamptz
);
CREATE UNIQUE INDEX earn_prices_current ON earn_prices (tier_id, period) WHERE status = 'current';
CREATE INDEX earn_prices_live_idx ON earn_prices (workspace_id, stripe_live_id) WHERE stripe_live_id <> '';
CREATE INDEX earn_prices_test_idx ON earn_prices (workspace_id, stripe_test_id) WHERE stripe_test_id <> '';

-- Prices found in Stripe that no tier knows (the sync error): their
-- subscriptions are left out of MRR and show no tier until mapped.
CREATE TABLE earn_unmapped_prices (
    id               uuid PRIMARY KEY,
    workspace_id     uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    mode             text        NOT NULL CHECK (mode IN ('live','test')),
    stripe_price_id  text        NOT NULL,
    period           text        NOT NULL DEFAULT '' CHECK (period IN ('','month','year')),
    amount_cents     bigint      NOT NULL DEFAULT 0,
    currency         text        NOT NULL DEFAULT '',
    found_at         timestamptz NOT NULL,
    found_by         text        NOT NULL DEFAULT 'sync' CHECK (found_by IN ('sync','event','import')),
    notified_at      timestamptz,
    mapped_at        timestamptz,
    mapped_price_id  uuid        REFERENCES earn_prices(id) ON DELETE SET NULL,
    mapped_by        uuid        REFERENCES users(id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX earn_unmapped_prices_once ON earn_unmapped_prices (workspace_id, mode, stripe_price_id);

-- A contact's paid access: one row per Stripe subscription, complimentary
-- grant or gift. A contact has at most one with access shown at a time
-- (the highest tier); a second active Stripe subscription is a sync warning.
CREATE TABLE paid_subscriptions (
    id                     uuid PRIMARY KEY,
    workspace_id           uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    contact_id             uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    mode                   text        NOT NULL CHECK (mode IN ('live','test')),
    kind                   text        NOT NULL CHECK (kind IN ('stripe','complimentary','gift')),
    tier_id                uuid        REFERENCES earn_tiers(id) ON DELETE SET NULL,
    price_id               uuid        REFERENCES earn_prices(id) ON DELETE SET NULL,
    stripe_price_id        text        NOT NULL DEFAULT '',
    -- incomplete: a checkout that didn't finish; it is never shown.
    status                 text        NOT NULL CHECK (status IN ('incomplete','trialing','active','past_due','canceled','complimentary','gift')),
    period                 text        NOT NULL DEFAULT '' CHECK (period IN ('','month','year')),
    -- The recurring amount before any discount.
    amount_cents           bigint      NOT NULL DEFAULT 0,
    currency               text        NOT NULL DEFAULT '',
    -- {"label":"FALL26","percent":20,"amount_cents":6400,"duration":"once","ends_at":"…","offer":"…"}
    discount               jsonb,
    started_at             timestamptz NOT NULL,
    trial_end              timestamptz,
    current_period_end     timestamptz,
    -- A scheduled end ("Cancels 2 Nov") and when it was asked for.
    cancel_at              timestamptz,
    canceled_at            timestamptz,
    canceled_by            uuid        REFERENCES users(id) ON DELETE SET NULL,
    -- When access ended, and why: canceled, trial_ended, payment_failed,
    -- grant_removed, grant_ended, disconnected.
    ended_at               timestamptz,
    end_reason             text        NOT NULL DEFAULT '',
    -- The reader's reason or the team's note.
    cancel_reason          text        NOT NULL DEFAULT '',
    -- Lifetime value: paid minus refunded, before tax.
    paid_cents             bigint      NOT NULL DEFAULT 0,
    refunded_cents         bigint      NOT NULL DEFAULT 0,
    -- A failed payment: Stripe's reason and the next try. access_paused is
    -- set when the access rule ended access at once (it comes back when a
    -- retry succeeds); with the default rule past due keeps access.
    failed_at              timestamptz,
    access_paused          boolean     NOT NULL DEFAULT false,
    failure_reason         text        NOT NULL DEFAULT '',
    next_retry_at          timestamptz,
    stripe_customer_id     text        NOT NULL DEFAULT '',
    stripe_subscription_id text        NOT NULL DEFAULT '',
    billing_email          text        NOT NULL DEFAULT '',
    -- How the reader got here: campaign, automation, page, offer, website,
    -- import, gift, grant or direct, with what the timeline names.
    source_kind            text        NOT NULL DEFAULT '' CHECK (source_kind IN ('','campaign','automation','page','offer','website','import','gift','grant','direct')),
    source_campaign_id     uuid        REFERENCES campaigns(id) ON DELETE SET NULL,
    source_label           text        NOT NULL DEFAULT '',
    source_detail          jsonb       NOT NULL DEFAULT '{}',
    -- Complimentary access: why, an optional note, who gave it and until when (null: no end).
    grant_reason           text        NOT NULL DEFAULT '' CHECK (grant_reason IN ('','team','contributor','press','partner','giveaway','other')),
    grant_note             text        NOT NULL DEFAULT '',
    granted_by             uuid        REFERENCES users(id) ON DELETE SET NULL,
    grant_until            timestamptz,
    import_id              uuid,
    -- The newest Stripe event whose subscription state was applied, so an
    -- older event that arrives later never undoes a newer state.
    stripe_synced_at       timestamptz,
    -- Emails already sent for this subscription's milestones (welcome,
    -- canceled), so a replayed event never sends them twice.
    emailed                text[]      NOT NULL DEFAULT '{}',
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX paid_subscriptions_contact_idx ON paid_subscriptions (contact_id);
CREATE INDEX paid_subscriptions_ws_idx ON paid_subscriptions (workspace_id, mode, status);
CREATE INDEX paid_subscriptions_customer_idx ON paid_subscriptions (workspace_id, stripe_customer_id) WHERE stripe_customer_id <> '';
CREATE UNIQUE INDEX paid_subscriptions_stripe ON paid_subscriptions (workspace_id, mode, stripe_subscription_id) WHERE stripe_subscription_id <> '';
CREATE UNIQUE INDEX paid_subscriptions_one_grant ON paid_subscriptions (contact_id, mode) WHERE kind = 'complimentary' AND ended_at IS NULL;

-- A paid member's timeline: checkout, payments and failures, renewals,
-- plan changes, offers, member emails, cancellations, refunds, imports and
-- grants. The payments event log keeps the raw events; this keeps what
-- they meant for the person.
CREATE TABLE paid_events (
    id               uuid PRIMARY KEY,
    workspace_id     uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    contact_id       uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    subscription_id  uuid        REFERENCES paid_subscriptions(id) ON DELETE CASCADE,
    mode             text        NOT NULL CHECK (mode IN ('live','test')),
    kind             text        NOT NULL,
    occurred_at      timestamptz NOT NULL,
    amount_cents     bigint,
    currency         text        NOT NULL DEFAULT '',
    payment_event_id uuid        REFERENCES payment_events(id) ON DELETE SET NULL,
    stripe_event     text        NOT NULL DEFAULT '',
    message_id       uuid,
    actor_id         uuid        REFERENCES users(id) ON DELETE SET NULL,
    data             jsonb       NOT NULL DEFAULT '{}'
);
CREATE INDEX paid_events_contact_idx ON paid_events (contact_id, occurred_at DESC);
CREATE INDEX paid_events_subscription_idx ON paid_events (subscription_id, occurred_at DESC);

-- Checkout sessions opened from the upgrade page (and test checkouts from
-- its editor), with what they're credited to.
CREATE TABLE earn_checkouts (
    id                uuid PRIMARY KEY,
    workspace_id      uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    mode              text        NOT NULL CHECK (mode IN ('live','test')),
    tier_id           uuid        REFERENCES earn_tiers(id) ON DELETE SET NULL,
    price_id          uuid        REFERENCES earn_prices(id) ON DELETE SET NULL,
    period            text        NOT NULL CHECK (period IN ('month','year')),
    trial_days        integer     NOT NULL DEFAULT 0,
    contact_id        uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    email             text        NOT NULL DEFAULT '',
    stripe_session_id text        NOT NULL DEFAULT '',
    status            text        NOT NULL DEFAULT 'open' CHECK (status IN ('open','completed','canceled','expired','failed')),
    -- {"kind":"campaign","campaign_id":"…","message_id":"…","clicked_at":"…"} or {"kind":"page"|"offer"|"direct"}
    attribution       jsonb       NOT NULL DEFAULT '{}',
    offer             text        NOT NULL DEFAULT '',
    started_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at        timestamptz NOT NULL,
    completed_at      timestamptz,
    subscription_id   uuid        REFERENCES paid_subscriptions(id) ON DELETE SET NULL
);
CREATE INDEX earn_checkouts_ws_idx ON earn_checkouts (workspace_id, created_at DESC);
CREATE UNIQUE INDEX earn_checkouts_session ON earn_checkouts (stripe_session_id) WHERE stripe_session_id <> '';

-- Visits of the hosted Earn pages per day.
CREATE TABLE earn_page_visits (
    workspace_id uuid   NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    day          date   NOT NULL,
    page         text   NOT NULL CHECK (page IN ('upgrade','cancel','gift')),
    visits       bigint NOT NULL DEFAULT 0,
    PRIMARY KEY (workspace_id, day, page)
);

-- The eight member emails: transactional templates in the Essential
-- "Memberships" category, created as drafts when Stripe is first connected.
CREATE TABLE earn_member_emails (
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email        text NOT NULL CHECK (email IN ('welcome','trial_ending','payment_failed','renewal_reminder','canceled','gift_purchased','gift_received','gift_ending')),
    template_id  uuid REFERENCES transactional_templates(id) ON DELETE SET NULL,
    PRIMARY KEY (workspace_id, email)
);

-- +goose Down
DROP TABLE earn_member_emails;
DROP TABLE earn_page_visits;
DROP TABLE earn_checkouts;
DROP TABLE paid_events;
DROP TABLE paid_subscriptions;
DROP TABLE earn_unmapped_prices;
DROP TABLE earn_prices;
DROP TABLE earn_tiers;
DROP TABLE earn_settings;
