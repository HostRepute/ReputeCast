-- +goose Up
-- Offers and gifts (design expansion phase 6, M27). An offer is a discount
-- for readers: a signup offer by code or link, a retention offer shown on
-- the hosted cancel page, or a referral reward of single-use codes issued by
-- Grow. Saving an offer creates its coupon and promotion codes in the
-- payment provider; ReputeCast keeps the offer's rules, its results and what
-- readers decided on the cancel page. A gift is a fixed period of a tier,
-- bought once by one person for another and redeemed without a card.

CREATE TABLE earn_offers (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name            text        NOT NULL,
    kind            text        NOT NULL CHECK (kind IN ('signup','retention','referral')),
    -- A percentage (1–100) or a fixed amount in the workspace currency.
    discount_type   text        NOT NULL CHECK (discount_type IN ('percent','amount')),
    percent_off     integer     CHECK (percent_off BETWEEN 1 AND 100),
    amount_off      bigint      CHECK (amount_off > 0),
    currency        text        NOT NULL DEFAULT '',
    -- once (the first payment: the first year of a yearly price), a number
    -- of months, or forever.
    duration        text        NOT NULL CHECK (duration IN ('once','months','forever')),
    duration_months integer     CHECK (duration_months BETWEEN 1 AND 36),
    -- The tiers and periods it applies to: [{"tier":"…","period":"year"}].
    applies         jsonb       NOT NULL DEFAULT '[]',
    -- How readers get it: a code typed in Checkout, a link to the upgrade
    -- page, or both (signup offers). Codes are stored upper case.
    by_code         boolean     NOT NULL DEFAULT false,
    code            text        NOT NULL DEFAULT '',
    by_link         boolean     NOT NULL DEFAULT false,
    redeem_limit    integer     CHECK (redeem_limit > 0),
    ends_at         timestamptz,
    new_only        boolean     NOT NULL DEFAULT false,
    -- Retention eligibility: paid for at least N months (and, always, no
    -- retention offer in the last 12 months and not past due).
    min_paid_months integer     NOT NULL DEFAULT 0 CHECK (min_paid_months BETWEEN 0 AND 60),
    -- Referral rewards: how long each single-use code is valid.
    code_days       integer     NOT NULL DEFAULT 60 CHECK (code_days BETWEEN 1 AND 365),
    status          text        NOT NULL DEFAULT 'live' CHECK (status IN ('live','ended')),
    -- When and why it ended: its date, its limit, or ended early by a person.
    ended_at        timestamptz,
    ended_reason    text        NOT NULL DEFAULT '' CHECK (ended_reason IN ('','date','limit','early')),
    ended_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    -- The coupon and promotion code per mode, as the provider stores them:
    -- {"live":{"coupon":"FALL26-20","promo":"promo_1Q9bX2rT","coupons":["FALL26-20"],"missingAt":"…"}}.
    -- A changed discount is a new coupon; codes already used keep theirs.
    stripe          jsonb       NOT NULL DEFAULT '{}',
    sync_status     text        NOT NULL DEFAULT 'draft' CHECK (sync_status IN ('draft','synced','failed')),
    sync_error      text        NOT NULL DEFAULT '',
    started_at      timestamptz NOT NULL DEFAULT now(),
    created_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    deleted_at      timestamptz
);
CREATE UNIQUE INDEX earn_offers_code ON earn_offers (workspace_id, upper(code)) WHERE code <> '' AND deleted_at IS NULL;
CREATE INDEX earn_offers_ws_idx ON earn_offers (workspace_id, created_at) WHERE deleted_at IS NULL;

-- A referral reward's single-use codes (Grow issues one when a referrer
-- reaches the milestone). Each is a promotion code of the offer's coupon.
CREATE TABLE earn_offer_codes (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    offer_id     uuid        NOT NULL REFERENCES earn_offers(id) ON DELETE CASCADE,
    code         text        NOT NULL,
    contact_id   uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    issued_to    text        NOT NULL DEFAULT '',
    issued_at    timestamptz NOT NULL,
    expires_at   timestamptz NOT NULL,
    -- The promotion code in the provider, per mode.
    stripe       jsonb       NOT NULL DEFAULT '{}',
    redeemed_at  timestamptz,
    source       text        NOT NULL DEFAULT 'referral'
);
CREATE UNIQUE INDEX earn_offer_codes_code ON earn_offer_codes (workspace_id, upper(code));
CREATE INDEX earn_offer_codes_offer_idx ON earn_offer_codes (offer_id, issued_at DESC);

-- Retention offers accepted on the cancel page (and offers applied by
-- hand): the one-offer-per-12-months rule reads these.
CREATE TABLE earn_offer_uses (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    offer_id        uuid        NOT NULL REFERENCES earn_offers(id) ON DELETE CASCADE,
    contact_id      uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    subscription_id uuid        REFERENCES paid_subscriptions(id) ON DELETE SET NULL,
    mode            text        NOT NULL CHECK (mode IN ('live','test')),
    used_at         timestamptz NOT NULL,
    source          text        NOT NULL CHECK (source IN ('cancel_page','checkout','code'))
);
CREATE INDEX earn_offer_uses_contact_idx ON earn_offer_uses (contact_id, used_at DESC);
CREATE INDEX earn_offer_uses_offer_idx ON earn_offer_uses (offer_id, used_at DESC);

-- Each visit of a paid member's cancel page and what they decided: kept
-- the membership with the offer, went on to cancel, or left. Reasons stay
-- on the paid member's timeline and are summed on the Earn overview.
CREATE TABLE earn_cancel_visits (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    contact_id      uuid        REFERENCES contacts(id) ON DELETE CASCADE,
    subscription_id uuid        REFERENCES paid_subscriptions(id) ON DELETE SET NULL,
    mode            text        NOT NULL DEFAULT 'live' CHECK (mode IN ('live','test')),
    visited_at      timestamptz NOT NULL,
    -- The retention offer shown, if the paid member was eligible.
    offer_id        uuid        REFERENCES earn_offers(id) ON DELETE SET NULL,
    reason          text        NOT NULL DEFAULT '' CHECK (reason IN ('','not_reading','too_expensive','not_for_me','switching','other')),
    decision        text        NOT NULL DEFAULT '' CHECK (decision IN ('','kept','continued')),
    decided_at      timestamptz
);
CREATE INDEX earn_cancel_visits_ws_idx ON earn_cancel_visits (workspace_id, visited_at DESC);
CREATE INDEX earn_cancel_visits_contact_idx ON earn_cancel_visits (contact_id, visited_at DESC);

-- Gift options: a tier for a fixed period at a one-time price, with that
-- price in the provider per mode.
CREATE TABLE earn_gift_options (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    tier_id      uuid        NOT NULL REFERENCES earn_tiers(id) ON DELETE CASCADE,
    months       integer     NOT NULL CHECK (months BETWEEN 1 AND 36),
    amount_cents bigint      NOT NULL CHECK (amount_cents > 0),
    currency     text        NOT NULL,
    position     integer     NOT NULL DEFAULT 0,
    -- {"live":"price_…","test":"price_…"}
    stripe       jsonb       NOT NULL DEFAULT '{}',
    created_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    deleted_at   timestamptz
);
CREATE INDEX earn_gift_options_ws_idx ON earn_gift_options (workspace_id, position) WHERE deleted_at IS NULL;

-- The gift page and gifts' rules, next to the other Earn settings.
ALTER TABLE earn_settings
    ADD COLUMN gift_page_on     boolean     NOT NULL DEFAULT false,
    ADD COLUMN gift_page_at     timestamptz,
    ADD COLUMN gift_page_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    -- How long a gift link works after delivery.
    ADD COLUMN gift_link_months integer     NOT NULL DEFAULT 12 CHECK (gift_link_months IN (3, 6, 12, 24)),
    -- The gift page's copy: {"headline":"…","intro":"…"}.
    ADD COLUMN gift_page        jsonb       NOT NULL DEFAULT '{}';

-- A gift bought on the gift page. The message is kept until the gift ends,
-- plus 30 days. A gift link works once, until it expires; until it's used
-- it can be passed to someone else.
CREATE TABLE earn_gifts (
    id                uuid PRIMARY KEY,
    workspace_id      uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    mode              text        NOT NULL CHECK (mode IN ('live','test')),
    option_id         uuid        REFERENCES earn_gift_options(id) ON DELETE SET NULL,
    tier_id           uuid        REFERENCES earn_tiers(id) ON DELETE SET NULL,
    months            integer     NOT NULL,
    amount_cents      bigint      NOT NULL,
    currency          text        NOT NULL,
    -- checkout: not paid yet; paid: delivery scheduled; delivered; redeemed;
    -- expired (the link ran out); refunded; abandoned (checkout not finished).
    status            text        NOT NULL DEFAULT 'checkout' CHECK (status IN ('checkout','paid','delivered','redeemed','expired','refunded','abandoned')),
    buyer_name        text        NOT NULL DEFAULT '',
    buyer_email       text        NOT NULL,
    -- In person: the gift link goes to the buyer; no recipient address.
    in_person         boolean     NOT NULL DEFAULT false,
    recipient_name    text        NOT NULL DEFAULT '',
    recipient_email   text        NOT NULL DEFAULT '',
    message           text        NOT NULL DEFAULT '' CHECK (char_length(message) <= 300),
    -- When to deliver (null: at once) and the buyer's time zone.
    deliver_at        timestamptz,
    timezone          text        NOT NULL DEFAULT '',
    code              text        NOT NULL,
    stripe_session_id text        NOT NULL DEFAULT '',
    payment_intent    text        NOT NULL DEFAULT '',
    stripe_customer   text        NOT NULL DEFAULT '',
    paid_at           timestamptz,
    delivered_at      timestamptz,
    reminded_at       timestamptz,
    link_expires_at   timestamptz,
    redeemed_at       timestamptz,
    redeemed_email    text        NOT NULL DEFAULT '',
    redeemed_contact  uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    subscription_id   uuid        REFERENCES paid_subscriptions(id) ON DELETE SET NULL,
    ending_sent_at    timestamptz,
    refunded_at       timestamptz,
    message_purged_at timestamptz,
    created_at        timestamptz NOT NULL,
    updated_at        timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX earn_gifts_code ON earn_gifts (code);
CREATE INDEX earn_gifts_ws_idx ON earn_gifts (workspace_id, created_at DESC);
CREATE INDEX earn_gifts_due_idx ON earn_gifts (deliver_at) WHERE status = 'paid';
CREATE UNIQUE INDEX earn_gifts_session ON earn_gifts (stripe_session_id) WHERE stripe_session_id <> '';
CREATE INDEX earn_gifts_intent_idx ON earn_gifts (workspace_id, payment_intent) WHERE payment_intent <> '';

-- A gift's history: bought, delivered, reminded, redeemed, refunded, and
-- the team's changes (the audit log has those too).
CREATE TABLE earn_gift_events (
    id          uuid PRIMARY KEY,
    gift_id     uuid        NOT NULL REFERENCES earn_gifts(id) ON DELETE CASCADE,
    kind        text        NOT NULL,
    occurred_at timestamptz NOT NULL,
    actor_id    uuid        REFERENCES users(id) ON DELETE SET NULL,
    message_id  uuid,
    data        jsonb       NOT NULL DEFAULT '{}'
);
CREATE INDEX earn_gift_events_gift_idx ON earn_gift_events (gift_id, occurred_at DESC);

-- +goose Down
DROP TABLE earn_gift_events;
DROP TABLE earn_gifts;
ALTER TABLE earn_settings DROP COLUMN gift_page, DROP COLUMN gift_link_months, DROP COLUMN gift_page_by, DROP COLUMN gift_page_at, DROP COLUMN gift_page_on;
DROP TABLE earn_gift_options;
DROP TABLE earn_cancel_visits;
DROP TABLE earn_offer_uses;
DROP TABLE earn_offer_codes;
DROP TABLE earn_offers;
