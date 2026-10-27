-- +goose Up
-- The brand kit (review 2026-10, CO-11): what new emails, templates and
-- blocks start from. One per workspace; style holds the colours, fonts and
-- button style (brandkit.Style). Emails already written keep their own
-- styles; only synced blocks change in them.
CREATE TABLE brand_kits (
    workspace_id       uuid        PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    logo_light_id      uuid        REFERENCES media_assets(id) ON DELETE SET NULL,
    logo_dark_id       uuid        REFERENCES media_assets(id) ON DELETE SET NULL,
    logo_alt           text        NOT NULL DEFAULT '' CHECK (length(logo_alt) <= 120),
    style              jsonb       NOT NULL DEFAULT '{}',
    website            text        NOT NULL DEFAULT '' CHECK (length(website) <= 200),
    updated_by         uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at         timestamptz NOT NULL
);

-- Synced blocks: edit once, and the drafts and scheduled emails that hold
-- the block (an element with rc-synced="<kind>") get the change. Sent
-- emails keep the version they went out with.
CREATE TABLE synced_blocks (
    id           uuid        PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    kind         text        NOT NULL CHECK (kind IN ('header','footer')),
    mjml         text        NOT NULL CHECK (length(mjml) <= 200000),
    version      integer     NOT NULL DEFAULT 1,
    updated_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at   timestamptz NOT NULL,
    created_at   timestamptz NOT NULL,
    UNIQUE (workspace_id, kind)
);

-- +goose Down
DROP TABLE synced_blocks;
DROP TABLE brand_kits;
