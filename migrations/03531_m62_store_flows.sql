-- +goose Up
-- M62 G-29–G-31: store flows from the site tracker.

-- When a contact last entered a browse, cart, back-in-stock or price-drop
-- flow, so the same flow doesn't take them again within 7 days.
CREATE TABLE tracker_trigger_entries (
    automation_id uuid        NOT NULL REFERENCES automations(id) ON DELETE CASCADE,
    contact_id    uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    kind          text        NOT NULL CHECK (kind IN ('viewed_product', 'added_to_cart', 'back_in_stock', 'price_drop')),
    entered_at    timestamptz NOT NULL,
    PRIMARY KEY (automation_id, contact_id, kind)
);
CREATE INDEX tracker_trigger_entries_contact_idx ON tracker_trigger_entries (contact_id);

-- "Email me when it's back": a contact waiting for a sold-out product.
-- notified_at is set once the back-in-stock flow has taken them; asking
-- again clears it.
CREATE TABLE store_stock_requests (
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    contact_id   uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    product_id   uuid        NOT NULL REFERENCES store_products(id) ON DELETE CASCADE,
    created_at   timestamptz NOT NULL,
    notified_at  timestamptz,
    PRIMARY KEY (contact_id, product_id)
);
CREATE INDEX store_stock_requests_waiting_idx ON store_stock_requests (product_id) WHERE notified_at IS NULL;

-- Products bought together, rebuilt nightly from the last 180 days of
-- orders: how often other_id is in an order with product_id, divided by
-- product_id's orders; the top 20 per product.
CREATE TABLE store_product_pairs (
    store_id   uuid NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    product_id uuid NOT NULL REFERENCES store_products(id) ON DELETE CASCADE,
    other_id   uuid NOT NULL REFERENCES store_products(id) ON DELETE CASCADE,
    score      real NOT NULL,
    PRIMARY KEY (store_id, product_id, other_id)
);

-- +goose Down
DROP TABLE store_product_pairs;
DROP TABLE store_stock_requests;
DROP TABLE tracker_trigger_entries;
