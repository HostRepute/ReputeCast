-- +goose Up
-- Digital products (design expansion phase 7, M29): a file and a price,
-- sold through Stripe Checkout on the publisher's own Stripe account (the
-- payment connection of phase 6). Buyers get a personal download link by
-- email, through the Purchases category's "Product delivery" template on the
-- Transactional route; the link expires after a number of days and of
-- downloads. Refunds and disputes happen in Stripe and arrive here through
-- its webhooks. No card details are ever stored, and the download link
-- itself isn't either: it is signed from its id when it's needed.
--
-- Files live on the instance's data volume (ROADMAP default), not in the
-- database: download_files records where.

CREATE TABLE download_files (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    -- What it belongs to: a product, or a signup form's lead magnet
    -- ("After confirmation: offer a download").
    owner_kind   text        NOT NULL CHECK (owner_kind IN ('product','form')),
    owner_id     uuid        NOT NULL,
    -- The name as uploaded (deliverability-field-guide.pdf).
    name         text        NOT NULL,
    size_bytes   bigint      NOT NULL CHECK (size_bytes >= 0),
    -- What the content is, read from its bytes (pdf, zip, xlsx, epub…), and
    -- the type it's served with.
    kind         text        NOT NULL,
    content_type text        NOT NULL,
    -- A PDF's page count, when it could be read.
    pages        integer,
    sha256       bytea       NOT NULL,
    -- Where the bytes are, under the files directory.
    storage_key  text        NOT NULL,
    position     integer     NOT NULL DEFAULT 0,
    uploaded_by  uuid        REFERENCES users(id) ON DELETE SET NULL,
    uploaded_at  timestamptz NOT NULL DEFAULT now(),
    removed_at   timestamptz
);
CREATE INDEX download_files_owner_idx ON download_files (owner_kind, owner_id) WHERE removed_at IS NULL;

CREATE TABLE products (
    id                   uuid PRIMARY KEY,
    workspace_id         uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    -- The product as edited. Once published, edits are draft changes until
    -- "Publish changes" copies them to live.
    name                 text        NOT NULL,
    slug                 text        NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(slug) <= 80),
    short_desc           text        NOT NULL DEFAULT '' CHECK (length(short_desc) <= 120),
    long_desc            text        NOT NULL DEFAULT '' CHECK (length(long_desc) <= 4000),
    cover_url            text        NOT NULL DEFAULT '',
    cover_alt            text        NOT NULL DEFAULT '',
    -- One-time ($29), or pay what you want (a minimum of at least $1 and a
    -- suggested price). Amounts are in the currency's minor units.
    price_kind           text        NOT NULL DEFAULT 'one_time' CHECK (price_kind IN ('one_time','pwyw')),
    price_cents          bigint      NOT NULL DEFAULT 0 CHECK (price_cents >= 0),
    min_cents            bigint      NOT NULL DEFAULT 0 CHECK (min_cents >= 0),
    suggested_cents      bigint      NOT NULL DEFAULT 0 CHECK (suggested_cents >= 0),
    -- Selling: the sales page (one of the workspace's pages, or the default
    -- product page when null) and "Show in the shop".
    sales_page_id        uuid        REFERENCES pages(id) ON DELETE SET NULL,
    show_in_shop         boolean     NOT NULL DEFAULT true,
    -- The list offer next to the Buy button: unchecked by default, its exact
    -- text kept as consent.
    offer_on             boolean     NOT NULL DEFAULT false,
    offer_list_id        uuid        REFERENCES lists(id) ON DELETE SET NULL,
    offer_text           text        NOT NULL DEFAULT '',
    -- Delivery: the link's validity and downloads, and the template.
    link_days            integer     NOT NULL DEFAULT 7 CHECK (link_days BETWEEN 1 AND 30),
    link_downloads       integer     NOT NULL DEFAULT 5 CHECK (link_downloads BETWEEN 1 AND 20),
    delivery_template_id uuid        REFERENCES transactional_templates(id) ON DELETE SET NULL,
    revoke_on_refund     boolean     NOT NULL DEFAULT true,
    status               text        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','published','archived')),
    -- The published product (what the shop, its pages and checkout use):
    -- a snapshot of the fields above, and the address it answers at.
    live                 jsonb,
    live_slug            text,
    -- The product and its current price in Stripe, per mode:
    -- {"live":{"product":"prod_…","price":"price_…","kind":"one_time","amount":2900,"min":0,"suggested":0,"retired":["price_…"]}}.
    stripe               jsonb       NOT NULL DEFAULT '{}',
    -- The currency it sells in (the Stripe account's, lowercase), fixed once published.
    currency             text        NOT NULL DEFAULT '',
    published_at         timestamptz,
    published_by         uuid        REFERENCES users(id) ON DELETE SET NULL,
    first_published_at   timestamptz,
    archived_at          timestamptz,
    archived_by          uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_by           uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_by           uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at           timestamptz NOT NULL DEFAULT now(),
    deleted_at           timestamptz
);
CREATE UNIQUE INDEX products_slug ON products (workspace_id, slug) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX products_live_slug ON products (workspace_id, live_slug) WHERE deleted_at IS NULL AND live_slug IS NOT NULL;
CREATE INDEX products_list_idx ON products (workspace_id, created_at) WHERE deleted_at IS NULL;

-- Earlier addresses of default product pages: they redirect (301).
CREATE TABLE product_redirects (
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    slug         text        NOT NULL,
    product_id   uuid        NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, slug)
);

-- Order numbers count per workspace (1311).
CREATE TABLE product_order_numbers (
    workspace_id uuid   PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    last         bigint NOT NULL
);

-- A Buy button pressed: the Checkout session it opened, with what travels
-- with it (the list offer's state, the page, the time and the IP address).
CREATE TABLE product_checkouts (
    id             uuid PRIMARY KEY,
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    product_id     uuid        NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    mode           text        NOT NULL CHECK (mode IN ('live','test')),
    connection_id  uuid        REFERENCES payment_connections(id) ON DELETE SET NULL,
    session_id     text        NOT NULL DEFAULT '',
    -- A signed-in member's email, filled in at checkout.
    email          text        NOT NULL DEFAULT '',
    contact_id     uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    -- {"offered":true,"ticked":true,"text":"Also send me the Weekly Digest","listId":"…","page":"news.acme.io/field-guide","ip":"203.0.113.77","at":"…"}
    consent        jsonb       NOT NULL DEFAULT '{}',
    source         text        NOT NULL DEFAULT 'page' CHECK (source IN ('page','shop','product_page')),
    source_page_id uuid        REFERENCES pages(id) ON DELETE SET NULL,
    source_url     text        NOT NULL DEFAULT '',
    status         text        NOT NULL DEFAULT 'open' CHECK (status IN ('open','completed','expired','failed')),
    order_id       uuid,
    created_at     timestamptz NOT NULL DEFAULT now(),
    completed_at   timestamptz
);
CREATE UNIQUE INDEX product_checkouts_session ON product_checkouts (session_id) WHERE session_id <> '';
CREATE INDEX product_checkouts_open_idx ON product_checkouts (created_at) WHERE status = 'open';

CREATE TABLE product_orders (
    id                   uuid PRIMARY KEY,
    workspace_id         uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    number               bigint      NOT NULL,
    product_id           uuid        NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    -- The product's name when it was bought.
    product_name         text        NOT NULL,
    -- Test orders (Stripe in test mode) carry a Test badge and are left out
    -- of every number.
    mode                 text        NOT NULL CHECK (mode IN ('live','test')),
    connection_id        uuid        REFERENCES payment_connections(id) ON DELETE SET NULL,
    checkout_id          uuid        REFERENCES product_checkouts(id) ON DELETE SET NULL,
    -- The buyer. A buyer whose email matches a contact is linked to it;
    -- one who ticked the list offer became a contact on that list; any
    -- other buyer isn't a contact: the order keeps the email for delivery
    -- and support only.
    buyer_email          text        NOT NULL,
    buyer_name           text        NOT NULL DEFAULT '',
    contact_id           uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    buyer                text        NOT NULL DEFAULT 'none' CHECK (buyer IN ('contact','joined','none')),
    amount_cents         bigint      NOT NULL CHECK (amount_cents >= 0),
    tax_cents            bigint      NOT NULL DEFAULT 0,
    currency             text        NOT NULL,
    pay_what_you_want    boolean     NOT NULL DEFAULT false,
    status               text        NOT NULL DEFAULT 'paid' CHECK (status IN ('paid','refunded','partially_refunded','disputed')),
    refunded_cents       bigint      NOT NULL DEFAULT 0,
    refunded_at          timestamptz,
    -- A dispute: Stripe's id, status (open, won, lost), reason and the date
    -- to respond by.
    dispute_id           text        NOT NULL DEFAULT '',
    dispute_status       text        NOT NULL DEFAULT '' CHECK (dispute_status IN ('','open','won','lost')),
    dispute_reason       text        NOT NULL DEFAULT '',
    dispute_fee_cents    bigint      NOT NULL DEFAULT 0,
    disputed_at          timestamptz,
    dispute_due_by       timestamptz,
    -- Access to the download: revoked by a person, or by the system after a
    -- refund or a dispute when the product says so.
    access_revoked_at    timestamptz,
    access_revoked_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    access_revoke_reason text        NOT NULL DEFAULT '' CHECK (access_revoke_reason IN ('','manual','refund','dispute')),
    provider             text        NOT NULL DEFAULT 'stripe',
    session_id           text        NOT NULL DEFAULT '',
    payment_intent       text        NOT NULL DEFAULT '',
    charge_id            text        NOT NULL DEFAULT '',
    -- Where the Buy button was: a page, the shop or the default product page.
    source               text        NOT NULL DEFAULT 'page' CHECK (source IN ('page','shop','product_page')),
    source_page_id       uuid        REFERENCES pages(id) ON DELETE SET NULL,
    source_url           text        NOT NULL DEFAULT '',
    -- The consent given at checkout (as product_checkouts.consent).
    consent              jsonb       NOT NULL DEFAULT '{}',
    -- The email the sale is credited to: the last click within the
    -- attribution window, by the click token or the buyer's email.
    campaign_id          uuid        REFERENCES campaigns(id) ON DELETE SET NULL,
    automation_id        uuid        REFERENCES automations(id) ON DELETE SET NULL,
    attributed_by        text        NOT NULL DEFAULT '' CHECK (attributed_by IN ('','token','email')),
    checkout_started_at  timestamptz,
    paid_at              timestamptz NOT NULL,
    created_at           timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX product_orders_number ON product_orders (workspace_id, number);
CREATE UNIQUE INDEX product_orders_session ON product_orders (session_id) WHERE session_id <> '';
CREATE INDEX product_orders_intent_idx ON product_orders (workspace_id, payment_intent) WHERE payment_intent <> '';
CREATE INDEX product_orders_list_idx ON product_orders (workspace_id, paid_at DESC);
CREATE INDEX product_orders_product_idx ON product_orders (product_id, paid_at DESC);
CREATE INDEX product_orders_contact_idx ON product_orders (contact_id) WHERE contact_id IS NOT NULL;
CREATE INDEX product_orders_email_idx ON product_orders (workspace_id, lower(buyer_email));
CREATE INDEX product_orders_campaign_idx ON product_orders (campaign_id) WHERE campaign_id IS NOT NULL;

-- Download links: the purchase's, and each new one sent by a person or
-- asked for by the buyer (only ever to the order's address). A new link
-- replaces the older ones. The token is signed from the link's id; it's
-- never stored, and neither is the delivery email's download_url.
CREATE TABLE product_order_links (
    id                uuid PRIMARY KEY,
    order_id          uuid        NOT NULL REFERENCES product_orders(id) ON DELETE CASCADE,
    reason            text        NOT NULL CHECK (reason IN ('purchase','person','buyer','account')),
    created_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    expires_at        timestamptz NOT NULL,
    downloads_allowed integer     NOT NULL CHECK (downloads_allowed BETWEEN 1 AND 20),
    downloads_used    integer     NOT NULL DEFAULT 0,
    -- The delivery email in Transactional › Messages, and how sending it went.
    message_id        uuid,
    delivery          text        NOT NULL DEFAULT 'pending' CHECK (delivery IN ('pending','sent','failed')),
    delivery_error    text        NOT NULL DEFAULT '',
    superseded_at     timestamptz
);
CREATE INDEX product_order_links_order_idx ON product_order_links (order_id, created_at DESC);

-- Each download, counted on the server. The IP address is kept only as a
-- hash, enough to say "from 2 IP addresses".
CREATE TABLE product_order_downloads (
    id         uuid PRIMARY KEY,
    order_id   uuid        NOT NULL REFERENCES product_orders(id) ON DELETE CASCADE,
    link_id    uuid        NOT NULL REFERENCES product_order_links(id) ON DELETE CASCADE,
    file_id    uuid        NOT NULL REFERENCES download_files(id) ON DELETE CASCADE,
    ip_hash    bytea,
    at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX product_order_downloads_order_idx ON product_order_downloads (order_id, at);

-- An order's activity: checkout started, paid, delivered, downloaded,
-- refunded, disputed, access revoked or restored, a new link sent.
CREATE TABLE product_order_events (
    id         uuid PRIMARY KEY,
    order_id   uuid        NOT NULL REFERENCES product_orders(id) ON DELETE CASCADE,
    kind       text        NOT NULL,
    at         timestamptz NOT NULL,
    actor_id   uuid        REFERENCES users(id) ON DELETE SET NULL,
    -- {"stripe":"checkout.session.completed","provider":"Postmark","seconds":1.4,"amount":2900,"reason":"fraudulent","respondBy":"…","email":"…"}
    detail     jsonb       NOT NULL DEFAULT '{}'
);
CREATE INDEX product_order_events_order_idx ON product_order_events (order_id, at);

-- Lead magnets: a signup form's download after confirmation. A link per
-- confirmed signup, for the form's number of days.
ALTER TABLE forms
    ADD COLUMN download_file_id uuid REFERENCES download_files(id) ON DELETE SET NULL,
    ADD COLUMN download_days    integer NOT NULL DEFAULT 7 CHECK (download_days BETWEEN 1 AND 30);

CREATE TABLE lead_magnet_links (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    form_id      uuid        NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    file_id      uuid        NOT NULL REFERENCES download_files(id) ON DELETE CASCADE,
    contact_id   uuid        REFERENCES contacts(id) ON DELETE CASCADE,
    created_at   timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL,
    downloads    integer     NOT NULL DEFAULT 0
);
CREATE INDEX lead_magnet_links_form_idx ON lead_magnet_links (form_id, created_at);

-- +goose Down
DROP TABLE lead_magnet_links;
ALTER TABLE forms DROP COLUMN download_days, DROP COLUMN download_file_id;
DROP TABLE product_order_events;
DROP TABLE product_order_downloads;
DROP TABLE product_order_links;
DROP TABLE product_orders;
DROP TABLE product_checkouts;
DROP TABLE product_order_numbers;
DROP TABLE product_redirects;
DROP TABLE products;
DROP TABLE download_files;
