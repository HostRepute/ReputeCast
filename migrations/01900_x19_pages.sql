-- +goose Up
-- Landing and sales pages (design expansion phase 7, M29): pages built with
-- blocks on the publisher's theme, in the composer's page mode. A page is a
-- GrapesJS project plus the HTML it renders (pages aren't MJML, ROADMAP
-- default): the editor's draft is kept here, and each published version is
-- a snapshot in page_versions that the public site serves. Blocks that show
-- live data (a product and its Buy button, a signup form, a countdown, the
-- event details, recent posts, the paid tiers) are custom elements
-- (<rc-product …>) the site draws on every request.
--
-- Pages work with or without the website. Members and Paid visibility need
-- sign-in, so they need the website.

CREATE TABLE pages (
    id                 uuid PRIMARY KEY,
    workspace_id       uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    -- The draft: the page as edited. Its settings are part of it, so the
    -- publish dialog lists their changes too.
    name               text        NOT NULL,
    slug               text        NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(slug) <= 80),
    template           text        NOT NULL CHECK (template IN ('about','lead_magnet','launch','webinar','sales','blank')),
    seo_title          text        NOT NULL DEFAULT '',
    seo_description    text        NOT NULL DEFAULT '',
    social_image       text        NOT NULL DEFAULT '',
    visibility         text        NOT NULL DEFAULT 'public' CHECK (visibility IN ('public','members','paid')),
    -- Paid pages: the tiers that may read it (empty: every paid tier).
    tiers              uuid[]      NOT NULL DEFAULT '{}',
    -- "Hide the site header and navigation" (focused landing pages: the logo only).
    hide_header        boolean     NOT NULL DEFAULT false,
    -- "Show in navigation" and the header link's label ("Field Guide").
    show_in_nav        boolean     NOT NULL DEFAULT false,
    nav_label          text        NOT NULL DEFAULT '',
    -- The GrapesJS project (layer names and the like) and the page's HTML,
    -- checked against the page blocks' allow-list when saved.
    project            jsonb       NOT NULL DEFAULT '{}',
    html               text        NOT NULL DEFAULT '',
    -- The draft's version, for autosave and conflicts; saved_* is the last save.
    version            integer     NOT NULL DEFAULT 1,
    saved_at           timestamptz NOT NULL DEFAULT now(),
    saved_by           uuid        REFERENCES users(id) ON DELETE SET NULL,
    autosaves          integer     NOT NULL DEFAULT 0,
    status             text        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','scheduled','published','unpublished')),
    -- Scheduled pages go live at publish_at (written in publish_tz).
    publish_at         timestamptz,
    publish_tz         text        NOT NULL DEFAULT '',
    -- What readers see: the published version and the address it answers at.
    live_version       integer,
    live_slug          text,
    published_at       timestamptz,
    published_by       uuid        REFERENCES users(id) ON DELETE SET NULL,
    first_published_at timestamptz,
    unpublished_at     timestamptz,
    unpublished_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    unpublish_reason   text        NOT NULL DEFAULT '',
    -- An unpublished page answers 410, or redirects here (a path on the site).
    redirect_to        text        NOT NULL DEFAULT '',
    created_by         uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_by         uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at         timestamptz NOT NULL DEFAULT now(),
    deleted_at         timestamptz
);
CREATE UNIQUE INDEX pages_slug ON pages (workspace_id, slug) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX pages_live_slug ON pages (workspace_id, live_slug) WHERE deleted_at IS NULL AND live_slug IS NOT NULL;
CREATE INDEX pages_list_idx ON pages (workspace_id, updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX pages_due_idx ON pages (publish_at) WHERE status = 'scheduled' AND deleted_at IS NULL;

-- Saved versions (Version history) and published snapshots. A published
-- version keeps the settings it went live with.
CREATE TABLE page_versions (
    page_id      uuid        NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    version      integer     NOT NULL,
    name         text        NOT NULL,
    slug         text        NOT NULL,
    -- {"seoTitle":"…","seoDescription":"…","socialImage":"…","visibility":"public","tiers":[],"hideHeader":false,"showInNav":true,"navLabel":"Field Guide"}
    settings     jsonb       NOT NULL DEFAULT '{}',
    project      jsonb       NOT NULL DEFAULT '{}',
    html         text        NOT NULL DEFAULT '',
    label        text        NOT NULL DEFAULT '',
    saved_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    saved_at     timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz,
    published_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    PRIMARY KEY (page_id, version)
);

-- Earlier addresses of pages: they redirect (301) to the page.
CREATE TABLE page_redirects (
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    slug         text        NOT NULL,
    page_id      uuid        NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, slug)
);
CREATE INDEX page_redirects_page_idx ON page_redirects (page_id);

-- What a page brought in: confirmed signups from its forms, registrations
-- from a webinar page's form and sales from its Buy buttons. The shop's
-- sales have no page (page_id is null and source is 'shop').
CREATE TABLE page_conversions (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    page_id      uuid        REFERENCES pages(id) ON DELETE CASCADE,
    source       text        NOT NULL DEFAULT 'page' CHECK (source IN ('page','shop','product_page')),
    kind         text        NOT NULL CHECK (kind IN ('signup','registration','sale')),
    contact_id   uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    -- A sale's order and amount (the order's currency's minor units).
    order_id     uuid,
    amount_cents bigint      NOT NULL DEFAULT 0,
    occurred_at  timestamptz NOT NULL
);
CREATE INDEX page_conversions_idx ON page_conversions (workspace_id, page_id, kind, occurred_at);
CREATE UNIQUE INDEX page_conversions_order ON page_conversions (order_id) WHERE order_id IS NOT NULL;

-- The shop (/products): a built-in page, on by default.
CREATE TABLE shop_settings (
    workspace_id uuid        PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    shop_on      boolean     NOT NULL DEFAULT true,
    title        text        NOT NULL DEFAULT '',
    intro        text        NOT NULL DEFAULT '',
    updated_at   timestamptz NOT NULL DEFAULT now(),
    updated_by   uuid        REFERENCES users(id) ON DELETE SET NULL
);

-- Settings › Approvals' "Publishing a post or page needs approval" covers
-- pages: approval requests for them.
ALTER TABLE approval_requests DROP CONSTRAINT approval_requests_subject_type_check;
ALTER TABLE approval_requests ADD CONSTRAINT approval_requests_subject_type_check
    CHECK (subject_type IN ('campaign','automation','transactional_template','website_post','website_page'));
ALTER TABLE approval_requests
    ADD COLUMN website_page_id uuid REFERENCES pages(id) ON DELETE CASCADE;
CREATE UNIQUE INDEX approval_requests_open_page ON approval_requests (website_page_id)
    WHERE status = 'pending' AND website_page_id IS NOT NULL;
CREATE INDEX approval_requests_page_idx ON approval_requests (website_page_id) WHERE website_page_id IS NOT NULL;

-- +goose Down
DROP INDEX approval_requests_page_idx;
DROP INDEX approval_requests_open_page;
DELETE FROM approval_requests WHERE subject_type = 'website_page';
ALTER TABLE approval_requests DROP COLUMN website_page_id;
ALTER TABLE approval_requests DROP CONSTRAINT approval_requests_subject_type_check;
ALTER TABLE approval_requests ADD CONSTRAINT approval_requests_subject_type_check
    CHECK (subject_type IN ('campaign','automation','transactional_template','website_post'));
DROP TABLE shop_settings;
DROP TABLE page_conversions;
DROP TABLE page_redirects;
DROP TABLE page_versions;
DROP TABLE pages;
