-- +goose Up
-- M59 G-22: unique discount codes per recipient. A pool holds codes that
-- were uploaded or that ReputeCast created in a connected store; each
-- campaign message or automation email that shows the pool's code gets
-- one when it is sent ({{ coupons.<slug>.code }}).

CREATE TABLE coupon_pools (
    id                   uuid PRIMARY KEY,
    workspace_id         uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name                 text        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 80),
    -- The name templates use: {{ coupons.welcome15.code }}. Never changes.
    slug                 text        NOT NULL CHECK (slug ~ '^[a-z0-9_]{1,32}$'),
    source               text        NOT NULL CHECK (source IN ('upload', 'shopify', 'woocommerce')),
    store_id             uuid        REFERENCES stores(id) ON DELETE SET NULL,
    -- Store pools: {"kind": "percentage|fixed_amount|free_shipping", "value",
    -- "minimumCents", "collectionIds", "endsDays"}; upload pools:
    -- {"applyUrl": "https://shop.example.com/discount/{code}"}.
    settings             jsonb       NOT NULL DEFAULT '{}',
    -- Created codes start with it (WELCOME-7KQ4M2XA).
    prefix               text        NOT NULL DEFAULT '' CHECK (prefix ~ '^[A-Z0-9-]{0,12}$'),
    one_per_contact      boolean     NOT NULL DEFAULT true,
    -- Members with coupons:manage are told once free codes fall under it.
    low_at               integer     NOT NULL DEFAULT 200 CHECK (low_at BETWEEN 0 AND 1000000),
    status               text        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    -- The store's discount the codes belong to (Shopify), once created.
    external_discount_id text        NOT NULL DEFAULT '',
    -- When the low alert went out; cleared once the pool is above low_at.
    notified_low_at      timestamptz,
    -- When the hourly refill last looked at the pool (store pools that
    -- live automations use).
    refill_checked_at    timestamptz,
    created_by           uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, slug)
);
CREATE INDEX coupon_pools_store_idx ON coupon_pools (store_id) WHERE store_id IS NOT NULL;

-- An upload or a request to create codes in a store. A store creates codes
-- in steps (Shopify adds them in the background): the codes of the step
-- under way are kept sealed in pending_sealed with the store's job id
-- until it confirms them; only confirmed codes become coupon_codes.
CREATE TABLE coupon_batches (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    pool_id         uuid        NOT NULL REFERENCES coupon_pools(id) ON DELETE CASCADE,
    kind            text        NOT NULL CHECK (kind IN ('upload', 'generate')),
    -- Why codes were created: the pool's first codes, a member asking for
    -- more, or the hourly refill for the live automation that uses it.
    reason          text        NOT NULL DEFAULT 'request' CHECK (reason IN ('pool', 'request', 'refill')),
    automation_id   uuid        REFERENCES automations(id) ON DELETE SET NULL,
    requested       integer     NOT NULL DEFAULT 0,
    created         integer     NOT NULL DEFAULT 0,
    failed          integer     NOT NULL DEFAULT 0,
    status          text        NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'done', 'failed')),
    -- Why a batch failed or waits to try again (a code the pool page writes
    -- in the reader's language: unreachable, not_granted, read_only, …).
    error           text        NOT NULL DEFAULT '',
    external_job_id text        NOT NULL DEFAULT '',
    -- The store asked ReputeCast to slow down: the batch goes on then.
    waiting_until   timestamptz,
    pending_sealed  bytea,
    created_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    finished_at     timestamptz
);
CREATE INDEX coupon_batches_pool_idx ON coupon_batches (pool_id, created_at DESC);
CREATE UNIQUE INDEX coupon_batches_running_idx ON coupon_batches (pool_id) WHERE status = 'running' AND kind = 'generate';

-- One code. Sealed (secrets box, "coupon_code:<pool>"), with a keyed hash
-- of the upper-cased code for duplicates and for matching store orders.
CREATE TABLE coupon_codes (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    pool_id      uuid        NOT NULL REFERENCES coupon_pools(id) ON DELETE CASCADE,
    code_sealed  bytea       NOT NULL,
    code_hash    bytea       NOT NULL,
    batch_id     uuid        REFERENCES coupon_batches(id) ON DELETE SET NULL,
    -- The store's id of the code (a Shopify redeem code, a WooCommerce
    -- coupon); cleared once a removed code was removed at the store too.
    external_id  text        NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    -- Given out: to whom and in which message (one code per message).
    assigned_at  timestamptz,
    contact_id   uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    message_id   uuid,
    -- Used in a store order.
    redeemed_at  timestamptz,
    order_id     uuid        REFERENCES store_orders(id) ON DELETE SET NULL,
    removed_at   timestamptz
);
CREATE UNIQUE INDEX coupon_codes_key ON coupon_codes (pool_id, code_hash);
CREATE INDEX coupon_codes_free_idx ON coupon_codes (pool_id, id) WHERE assigned_at IS NULL AND removed_at IS NULL;
CREATE UNIQUE INDEX coupon_codes_message_idx ON coupon_codes (pool_id, message_id) WHERE message_id IS NOT NULL;
CREATE INDEX coupon_codes_pool_contact_idx ON coupon_codes (pool_id, contact_id) WHERE contact_id IS NOT NULL;
CREATE INDEX coupon_codes_hash_idx ON coupon_codes (workspace_id, code_hash);
-- Erasure and contact deletion find a contact's codes; store deletion its
-- orders' codes; the store removal job the removed codes still at the store.
CREATE INDEX coupon_codes_contact_idx ON coupon_codes (contact_id) WHERE contact_id IS NOT NULL;
CREATE INDEX coupon_codes_order_idx ON coupon_codes (order_id) WHERE order_id IS NOT NULL;
CREATE INDEX coupon_codes_unstored_idx ON coupon_codes (pool_id) WHERE removed_at IS NOT NULL AND external_id <> '';

-- +goose Down
DROP TABLE coupon_codes;
DROP TABLE coupon_batches;
DROP TABLE coupon_pools;
