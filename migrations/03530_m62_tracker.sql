-- +goose Up
-- M62 G-28: the site tracker, for the publisher's own sites and stores
-- (never ReputeCast's hosted pages, which stay cookie-less).

-- A site the tracker runs on: its origin (scheme, host and port), the
-- store whose product ids its events name, whether it waits for the
-- visitor's consent before storing anything, and the secret its server
-- signs identify calls with (sealed, write-only).
CREATE TABLE tracker_sites (
    id                     uuid        PRIMARY KEY,
    workspace_id           uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name                   text        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 80),
    origin                 text        NOT NULL CHECK (char_length(origin) BETWEEN 8 AND 255),
    store_id               uuid        REFERENCES stores(id) ON DELETE SET NULL,
    wait_consent           boolean     NOT NULL DEFAULT true,
    identify_secret_sealed bytea,
    created_by             uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at             timestamptz NOT NULL,
    updated_at             timestamptz NOT NULL,
    last_event_at          timestamptz,
    deleted_at             timestamptz
);
CREATE UNIQUE INDEX tracker_sites_origin ON tracker_sites (workspace_id, origin) WHERE deleted_at IS NULL;

-- Events refused because they came from another origin than the site's
-- (Settings › Integrations › Website tracking says where from), counted
-- in memory and flushed with the event batches; kept 7 days.
CREATE TABLE tracker_site_refusals (
    site_id  uuid        NOT NULL REFERENCES tracker_sites(id) ON DELETE CASCADE,
    origin   text        NOT NULL CHECK (char_length(origin) <= 255),
    count    bigint      NOT NULL DEFAULT 0,
    first_at timestamptz NOT NULL,
    last_at  timestamptz NOT NULL,
    PRIMARY KEY (site_id, origin)
);

-- A browser the tracker has seen, by the hash of its random visitor id
-- (the id itself is never stored), and the contact it was linked to by a
-- signed click token, a form signup or a signed identify call.
CREATE TABLE tracker_visitors (
    id            uuid        PRIMARY KEY,
    workspace_id  uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    vid_hash      bytea       NOT NULL,
    contact_id    uuid        REFERENCES contacts(id) ON DELETE CASCADE,
    identified_by text        NOT NULL DEFAULT '' CHECK (identified_by IN ('', 'click', 'form', 'identify')),
    first_seen    timestamptz NOT NULL,
    last_seen     timestamptz NOT NULL,
    identified_at timestamptz,
    UNIQUE (workspace_id, vid_hash)
);
CREATE INDEX tracker_visitors_contact_idx ON tracker_visitors (contact_id) WHERE contact_id IS NOT NULL;
CREATE INDEX tracker_visitors_anonymous_idx ON tracker_visitors (last_seen) WHERE contact_id IS NULL;

-- What visitors did, in monthly partitions by occurred_at (the retention
-- job drops whole months and makes the coming ones). kind: 1 page view,
-- 2 product view, 3 added to cart, 4 a custom event (name, props). path
-- keeps no query but its utm_* parameters; product_id is the linked
-- store's product, NULL when the store doesn't know the id.
CREATE TABLE tracker_events (
    id              uuid        NOT NULL,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    site_id         uuid        NOT NULL,
    visitor_id      uuid        NOT NULL,
    contact_id      uuid,
    kind            smallint    NOT NULL CHECK (kind BETWEEN 1 AND 4),
    product_id      uuid,
    quantity        integer     NOT NULL DEFAULT 0,
    path            text        NOT NULL DEFAULT '' CHECK (char_length(path) <= 512),
    referrer_domain text        NOT NULL DEFAULT '' CHECK (char_length(referrer_domain) <= 255),
    name            text        NOT NULL DEFAULT '' CHECK (char_length(name) <= 40),
    props           jsonb,
    occurred_at     timestamptz NOT NULL
) PARTITION BY RANGE (occurred_at);
CREATE INDEX tracker_events_contact_idx ON tracker_events (contact_id, occurred_at) WHERE contact_id IS NOT NULL;
CREATE INDEX tracker_events_visitor_idx ON tracker_events (visitor_id, occurred_at);
CREATE INDEX tracker_events_site_idx ON tracker_events (site_id, occurred_at);

-- The default partition catches what falls outside the monthly ones, so
-- an insert never fails; the months are made ahead of time (the retention
-- job keeps two ahead). The three months before this one exist too, so
-- events from within the 90 days kept never land in the default.
CREATE TABLE tracker_events_default PARTITION OF tracker_events DEFAULT;
-- +goose StatementBegin
DO $$
DECLARE
    m date := date_trunc('month', now() AT TIME ZONE 'UTC')::date;
BEGIN
    FOR i IN -3..2 LOOP
        EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF tracker_events FOR VALUES FROM (%L) TO (%L)',
            'tracker_events_' || to_char(m + make_interval(months => i), 'YYYY_MM'),
            (m + make_interval(months => i))::timestamp AT TIME ZONE 'UTC',
            (m + make_interval(months => i + 1))::timestamp AT TIME ZONE 'UTC');
    END LOOP;
END
$$;
-- +goose StatementEnd

-- A known contact's product views and add-to-carts, per product (browse
-- and cart triggers, back in stock and price drop, recently viewed).
CREATE TABLE tracker_product_views (
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    contact_id     uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    product_id     uuid        NOT NULL REFERENCES store_products(id) ON DELETE CASCADE,
    views          integer     NOT NULL DEFAULT 0,
    carts          integer     NOT NULL DEFAULT 0,
    last_viewed_at timestamptz,
    last_carted_at timestamptz,
    PRIMARY KEY (contact_id, product_id)
);
CREATE INDEX tracker_product_views_viewed_idx ON tracker_product_views (workspace_id, last_viewed_at);
CREATE INDEX tracker_product_views_carted_idx ON tracker_product_views (workspace_id, last_carted_at) WHERE last_carted_at IS NOT NULL;
CREATE INDEX tracker_product_views_product_idx ON tracker_product_views (product_id);

-- The same per visitor, until the visitor is linked to a contact (then
-- merged into tracker_product_views).
CREATE TABLE tracker_visitor_products (
    visitor_id     uuid        NOT NULL REFERENCES tracker_visitors(id) ON DELETE CASCADE,
    product_id     uuid        NOT NULL REFERENCES store_products(id) ON DELETE CASCADE,
    views          integer     NOT NULL DEFAULT 0,
    carts          integer     NOT NULL DEFAULT 0,
    last_viewed_at timestamptz,
    last_carted_at timestamptz,
    PRIMARY KEY (visitor_id, product_id)
);

-- +goose Down
DROP TABLE tracker_visitor_products;
DROP TABLE tracker_product_views;
DROP TABLE tracker_events;
DROP TABLE tracker_visitors;
DROP TABLE tracker_site_refusals;
DROP TABLE tracker_sites;
