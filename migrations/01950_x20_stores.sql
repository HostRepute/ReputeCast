-- +goose Up
-- Shopify and WooCommerce (design expansion phase 7, M30): a workspace's
-- connected stores, what they sync (products, customers who accepted
-- marketing, orders and abandoned checkouts), their webhooks and the raw
-- events received, the store's own activity log, and what each abandoned
-- checkout did in the automations that watch it.
--
-- Never stored: addresses, phone numbers, payment details and order notes.
-- Customers who didn't accept marketing are kept only as a link to an
-- existing contact (no email, no name); anyone else counts in the store's
-- totals and attribution without their email. Orders and store webhooks
-- aren't in the audit log (ROADMAP default): the orders, this activity log
-- and the raw events (30 days) are their record.

CREATE TABLE stores (
    id                       uuid PRIMARY KEY,
    workspace_id             uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    platform                 text        NOT NULL CHECK (platform IN ('shopify', 'woocommerce')),
    -- connecting: the connect dialog is under way (step says where);
    -- importing: the first import runs; connected; disconnected (by a
    -- person, or because the store revoked access: revoked_at).
    status                   text        NOT NULL DEFAULT 'connecting' CHECK (status IN ('connecting', 'importing', 'connected', 'disconnected')),
    -- Shopify: address, app, keys, install, sync. WooCommerce: address,
    -- key, webhooks, plugin. done once connected.
    step                     text        NOT NULL DEFAULT 'address' CHECK (step IN ('address', 'app', 'keys', 'install', 'sync', 'key', 'webhooks', 'plugin', 'done')),
    name                     text        NOT NULL DEFAULT '',
    -- What was typed ("shop.acme.io"), the canonical host
    -- ("acme-merch.myshopify.com") and the storefront's host for links.
    address                  text        NOT NULL,
    domain                   text        NOT NULL DEFAULT '',
    public_domain            text        NOT NULL DEFAULT '',
    currency                 text        NOT NULL DEFAULT '',
    api_version              text        NOT NULL DEFAULT '',
    app_name                 text        NOT NULL DEFAULT '',
    -- Shopify's app client ID (not a secret; shown as 4f1c…a9e2).
    client_id                text        NOT NULL DEFAULT '',
    -- Sealed JSON: the client secret, the access token (and its refresh
    -- token), or WooCommerce's REST API key. Write-only.
    credentials_sealed       bytea,
    -- The key's hint, for "ck_…3f2a"; never the key.
    key_hint                 text        NOT NULL DEFAULT '',
    scopes                   text[]      NOT NULL DEFAULT '{}',
    -- The approval's state while it's under way (OAuth).
    oauth_state              text        UNIQUE,
    oauth_state_expires_at   timestamptz,
    -- The last refused client secret or key, for the dialog.
    connect_error            text        NOT NULL DEFAULT '',
    installed_at             timestamptz,
    -- Webhooks post to /hooks/{platform}/{webhook_token}; WooCommerce's are
    -- signed with webhook_secret (sealed), Shopify's with the client secret.
    webhook_token            text        UNIQUE,
    webhook_secret_sealed    bytea,
    -- What syncs.
    sync_products            boolean     NOT NULL DEFAULT true,
    sync_customers           boolean     NOT NULL DEFAULT true,
    sync_orders              boolean     NOT NULL DEFAULT true,
    sync_checkouts           boolean     NOT NULL DEFAULT true,
    import_months            integer     NOT NULL DEFAULT 12 CHECK (import_months IN (3, 6, 12)),
    -- The list customers who accepted marketing join.
    list_id                  uuid        REFERENCES lists(id) ON DELETE SET NULL,
    write_back               boolean     NOT NULL DEFAULT true,
    tag_links                boolean     NOT NULL DEFAULT true,
    attribution_days         integer     NOT NULL DEFAULT 5 CHECK (attribution_days BETWEEN 1 AND 30),
    attribution_set_by       uuid        REFERENCES users(id) ON DELETE SET NULL,
    attribution_set_at       timestamptz,
    -- WooCommerce: the ReputeCast plugin on the store's site.
    plugin_found             boolean     NOT NULL DEFAULT false,
    plugin_checked_at        timestamptz,
    plugin_skipped           boolean     NOT NULL DEFAULT false,
    -- Health.
    last_event_at            timestamptz,
    health_checked_at        timestamptz,
    -- The first import: {"products":{"done":38,"total":38},"customers":{…,"accepted":2318,"notAdded":3922,"known":1947},
    -- "orders":{…},"matching":{…},"cursor":"…","phase":"orders","by":"…"}.
    import_progress          jsonb       NOT NULL DEFAULT '{}',
    import_started_at        timestamptz,
    import_started_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    import_finished_at       timestamptz,
    -- The nightly reconciliation (03:00 in the workspace's timezone).
    reconciled_at            timestamptz,
    reconcile_missed         integer     NOT NULL DEFAULT 0,
    reconcile_products       integer     NOT NULL DEFAULT 0,
    -- The store throttled ReputeCast: the sync slows down until then.
    rate_limited_until       timestamptz,
    -- The store refused the credentials (uninstalled: Shopify's
    -- app/uninstalled; revoked: a 401).
    revoked_at               timestamptz,
    revoked_reason           text        NOT NULL DEFAULT '' CHECK (revoked_reason IN ('', 'uninstalled', 'revoked')),
    disconnected_at          timestamptz,
    disconnected_by          uuid        REFERENCES users(id) ON DELETE SET NULL,
    notified_at              timestamptz,
    -- The demo's sample store answers from an in-process fake and is
    -- never polled.
    demo                     boolean     NOT NULL DEFAULT false,
    connected_by             uuid        REFERENCES users(id) ON DELETE SET NULL,
    connected_at             timestamptz,
    created_by               uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at               timestamptz NOT NULL DEFAULT now(),
    updated_at               timestamptz NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, platform, address)
);
CREATE INDEX stores_workspace_idx ON stores (workspace_id);

-- Collections (Shopify) and product categories (WooCommerce).
CREATE TABLE store_collections (
    store_id    uuid    NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    external_id text    NOT NULL,
    title       text    NOT NULL,
    handle      text    NOT NULL DEFAULT '',
    products    integer NOT NULL DEFAULT 0,
    position    integer NOT NULL DEFAULT 0,
    PRIMARY KEY (store_id, external_id)
);

CREATE TABLE store_products (
    id               uuid PRIMARY KEY,
    store_id         uuid        NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    workspace_id     uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    external_id      text        NOT NULL,
    title            text        NOT NULL,
    handle           text        NOT NULL DEFAULT '',
    url              text        NOT NULL DEFAULT '',
    image_url        text        NOT NULL DEFAULT '',
    image_alt        text        NOT NULL DEFAULT '',
    -- Minor units of currency; compare_at is 0 when not on sale.
    price_cents      bigint      NOT NULL DEFAULT 0,
    compare_at_cents bigint      NOT NULL DEFAULT 0,
    currency         text        NOT NULL DEFAULT '',
    available        boolean     NOT NULL DEFAULT true,
    -- Stock when the store tracks it.
    inventory        integer,
    status           text        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'draft', 'archived')),
    collections      text[]      NOT NULL DEFAULT '{}',
    created_at_store timestamptz,
    updated_at_store timestamptz,
    -- Deleted in the store (products/delete, or gone at the reconciliation).
    removed_at       timestamptz,
    synced_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (store_id, external_id)
);
CREATE INDEX store_products_store_idx ON store_products (store_id, created_at_store DESC);

-- Customers who accepted marketing (email, name, country, the consent),
-- and customers who are contacts (only the link: the contact has the rest).
CREATE TABLE store_customers (
    id               uuid PRIMARY KEY,
    store_id         uuid        NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    workspace_id     uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    external_id      text        NOT NULL,
    contact_id       uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    email            text        NOT NULL DEFAULT '',
    first_name       text        NOT NULL DEFAULT '',
    last_name        text        NOT NULL DEFAULT '',
    country          text        NOT NULL DEFAULT '',
    marketing        text        NOT NULL DEFAULT 'not_subscribed' CHECK (marketing IN ('subscribed', 'not_subscribed', 'unsubscribed', 'pending')),
    opt_in_level     text        NOT NULL DEFAULT '',
    consent_at       timestamptz,
    -- When the sync added them to the store's list, and when ReputeCast
    -- last told the store about an unsubscribe.
    joined_list_at   timestamptz,
    told_store_at    timestamptz,
    updated_at_store timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (store_id, external_id)
);
CREATE INDEX store_customers_contact_idx ON store_customers (contact_id) WHERE contact_id IS NOT NULL;

CREATE TABLE store_orders (
    id                   uuid PRIMARY KEY,
    store_id             uuid        NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    workspace_id         uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    external_id          text        NOT NULL,
    name                 text        NOT NULL,
    -- The buyer, when they are a contact; never their email otherwise.
    contact_id           uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    customer_external_id text        NOT NULL DEFAULT '',
    accepted_marketing   boolean     NOT NULL DEFAULT false,
    placed_at            timestamptz NOT NULL,
    currency             text        NOT NULL,
    subtotal_cents       bigint      NOT NULL DEFAULT 0,
    total_cents          bigint      NOT NULL DEFAULT 0,
    refunded_cents       bigint      NOT NULL DEFAULT 0,
    discount_codes       text[]      NOT NULL DEFAULT '{}',
    -- [{"productId","title","variant","quantity","price"}]
    items                jsonb       NOT NULL DEFAULT '[]',
    -- The landing page without the click token, and its UTM parameters.
    landing_page         text        NOT NULL DEFAULT '',
    utm_source           text        NOT NULL DEFAULT '',
    utm_medium           text        NOT NULL DEFAULT '',
    utm_campaign         text        NOT NULL DEFAULT '',
    utm_content          text        NOT NULL DEFAULT '',
    utm_term             text        NOT NULL DEFAULT '',
    checkout_external_id text        NOT NULL DEFAULT '',
    cancelled            boolean     NOT NULL DEFAULT false,
    test                 boolean     NOT NULL DEFAULT false,
    -- The contact's first order in this store.
    first_order          boolean     NOT NULL DEFAULT false,
    -- Attribution: one order, one email (a campaign or an automation step),
    -- the last one whose link the buyer clicked within the window.
    matched              text        NOT NULL DEFAULT 'none' CHECK (matched IN ('click_token', 'email', 'none')),
    message_id           uuid,
    campaign_id          uuid        REFERENCES campaigns(id) ON DELETE SET NULL,
    automation_id        uuid        REFERENCES automations(id) ON DELETE SET NULL,
    automation_node      text        NOT NULL DEFAULT '',
    clicked_at           timestamptz,
    -- import: the first import; webhook; poll (a missing webhook); reconcile.
    source               text        NOT NULL DEFAULT 'webhook' CHECK (source IN ('import', 'webhook', 'poll', 'reconcile')),
    updated_at_store     timestamptz,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    UNIQUE (store_id, external_id)
);
CREATE INDEX store_orders_placed_idx ON store_orders (store_id, placed_at DESC);
CREATE INDEX store_orders_ws_placed_idx ON store_orders (workspace_id, placed_at DESC);
CREATE INDEX store_orders_contact_idx ON store_orders (contact_id, placed_at DESC) WHERE contact_id IS NOT NULL;
CREATE INDEX store_orders_campaign_idx ON store_orders (campaign_id) WHERE campaign_id IS NOT NULL;
CREATE INDEX store_orders_automation_idx ON store_orders (automation_id, placed_at) WHERE automation_id IS NOT NULL;
CREATE INDEX store_orders_items_idx ON store_orders USING gin (items jsonb_path_ops);

-- Checkouts with an email that may become abandoned checkouts.
CREATE TABLE store_checkouts (
    id                   uuid PRIMARY KEY,
    store_id             uuid        NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    workspace_id         uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    -- The checkout's token.
    external_id          text        NOT NULL,
    contact_id           uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    customer_external_id text        NOT NULL DEFAULT '',
    has_email            boolean     NOT NULL DEFAULT false,
    accepted_marketing   boolean     NOT NULL DEFAULT false,
    items                jsonb       NOT NULL DEFAULT '[]',
    subtotal_cents       bigint      NOT NULL DEFAULT 0,
    total_cents          bigint      NOT NULL DEFAULT 0,
    currency             text        NOT NULL DEFAULT '',
    recovery_url         text        NOT NULL DEFAULT '',
    created_at_store     timestamptz NOT NULL,
    updated_at_store     timestamptz NOT NULL,
    -- Became an order.
    completed_at         timestamptz,
    order_id             uuid        REFERENCES store_orders(id) ON DELETE SET NULL,
    source               text        NOT NULL DEFAULT 'webhook' CHECK (source IN ('import', 'webhook', 'poll', 'reconcile', 'plugin')),
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    UNIQUE (store_id, external_id)
);
CREATE INDEX store_checkouts_open_idx ON store_checkouts (store_id, updated_at_store) WHERE completed_at IS NULL;
CREATE INDEX store_checkouts_contact_idx ON store_checkouts (contact_id) WHERE contact_id IS NOT NULL;

-- What an abandoned checkout did in each automation that watches its
-- store: entered, or why not (the trigger's numbers: "44 entered · 34 not
-- emailed: didn't accept marketing").
CREATE TABLE store_checkout_entries (
    automation_id uuid        NOT NULL REFERENCES automations(id) ON DELETE CASCADE,
    checkout_id   uuid        NOT NULL REFERENCES store_checkouts(id) ON DELETE CASCADE,
    outcome       text        NOT NULL CHECK (outcome IN ('entered', 'no_email', 'no_consent', 'skip_rule', 'not_eligible')),
    at            timestamptz NOT NULL,
    PRIMARY KEY (automation_id, checkout_id)
);

-- The store's webhooks, one row per topic.
CREATE TABLE store_webhooks (
    store_id       uuid        NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    topic          text        NOT NULL,
    external_id    text        NOT NULL DEFAULT '',
    status         text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'registered', 'missing', 'failed')),
    registered_at  timestamptz,
    registered_by  uuid        REFERENCES users(id) ON DELETE SET NULL,
    -- Gone from the store (seen by the 15-minute check): ReputeCast polls
    -- its data every 15 minutes until it's registered again.
    missing_since  timestamptz,
    last_event_at  timestamptz,
    last_polled_at timestamptz,
    error          text        NOT NULL DEFAULT '',
    position       integer     NOT NULL DEFAULT 0,
    PRIMARY KEY (store_id, topic)
);

-- Every delivery received, verified or not, next to what ReputeCast did
-- with it. A retried delivery (the same delivery id) is stored once.
CREATE TABLE store_events (
    id           uuid PRIMARY KEY,
    store_id     uuid        NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    topic        text        NOT NULL DEFAULT '',
    delivery_id  text        NOT NULL DEFAULT '',
    triggered_at timestamptz,
    received_at  timestamptz NOT NULL,
    -- The headers parsing needs (topic, ids, signature), and the raw body.
    headers      jsonb       NOT NULL DEFAULT '{}',
    body         bytea       NOT NULL,
    signature_ok boolean     NOT NULL,
    status       text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'applied', 'ignored', 'stale', 'failed', 'rejected')),
    outcome      text        NOT NULL DEFAULT '',
    applied_at   timestamptz
);
CREATE UNIQUE INDEX store_events_once ON store_events (store_id, delivery_id) WHERE delivery_id <> '';
CREATE INDEX store_events_pending_idx ON store_events (received_at) WHERE status = 'pending';
CREATE INDEX store_events_store_idx ON store_events (store_id, received_at DESC);

-- The store's activity log (30 days): orders, abandoned checkouts, the
-- reconciliation, webhooks missing and registered again, rate limits,
-- errors. Structured facts; the app words them.
CREATE TABLE store_activity (
    id           uuid PRIMARY KEY,
    store_id     uuid        NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    at           timestamptz NOT NULL,
    kind         text        NOT NULL,
    tone         text        NOT NULL DEFAULT 'neutral' CHECK (tone IN ('neutral', 'ok', 'info', 'warn', 'bad')),
    data         jsonb       NOT NULL DEFAULT '{}',
    order_id     uuid        REFERENCES store_orders(id) ON DELETE SET NULL,
    checkout_id  uuid        REFERENCES store_checkouts(id) ON DELETE SET NULL,
    contact_id   uuid        REFERENCES contacts(id) ON DELETE SET NULL
);
CREATE INDEX store_activity_store_idx ON store_activity (store_id, at DESC);

-- A list a store connection created ("Acme Merch"): "All subscribers"
-- keeps meaning every newsletter list.
ALTER TABLE lists ADD COLUMN origin text NOT NULL DEFAULT '' CHECK (origin IN ('', 'store'));

-- WordPress sites running WooCommerce, as the ReputeCast plugin reports
-- them (1.4.0): WooCommerce's version, and whether the plugin's checkout
-- consent and abandoned-checkout reporting are on.
ALTER TABLE wordpress_sites
    ADD COLUMN woocommerce_version   text    NOT NULL DEFAULT '',
    ADD COLUMN woocommerce_consent   boolean NOT NULL DEFAULT false,
    ADD COLUMN woocommerce_checkouts boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE wordpress_sites DROP COLUMN woocommerce_checkouts, DROP COLUMN woocommerce_consent, DROP COLUMN woocommerce_version;
ALTER TABLE lists DROP COLUMN origin;
DROP TABLE store_activity;
DROP TABLE store_events;
DROP TABLE store_webhooks;
DROP TABLE store_checkout_entries;
DROP TABLE store_checkouts;
DROP TABLE store_orders;
DROP TABLE store_customers;
DROP TABLE store_products;
DROP TABLE store_collections;
DROP TABLE stores;
