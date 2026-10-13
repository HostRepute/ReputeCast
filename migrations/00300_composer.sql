-- +goose Up
-- Composer (7.4): version metadata, the content fingerprint approvals compare,
-- templates and saved sections, the image library and the cache of URI
-- blocklist lookups used by inline content checks.

ALTER TABLE campaigns
    -- sha256 of subject, preheader, MJML, HTML and A/B variants. Approvals
    -- snapshot it; any change after approval invalidates the approval.
    ADD COLUMN content_hash     text  NOT NULL DEFAULT '',
    -- Composer options applied to the stored HTML: {"minify":true}
    ADD COLUMN content_settings jsonb NOT NULL DEFAULT '{}';

ALTER TABLE campaign_versions
    ADD COLUMN plain_text text    NOT NULL DEFAULT '',
    ADD COLUMN ab_test    jsonb,
    -- How the version was made: the first draft, an autosave, ⌘S, a restore,
    -- a template applied, an MJML/HTML import or a duplicate.
    ADD COLUMN source     text    NOT NULL DEFAULT 'autosave'
        CHECK (source IN ('create','autosave','manual','restore','template','import','duplicate','api')),
    ADD COLUMN size_bytes integer NOT NULL DEFAULT 0;

ALTER TABLE templates
    -- Wireframe of the layout for the gallery: [{"h":10,"w":40},{"row":[{"h":28},{"h":28}]}…]
    ADD COLUMN thumb              jsonb       NOT NULL DEFAULT '[]',
    ADD COLUMN size_bytes         integer     NOT NULL DEFAULT 0,
    ADD COLUMN source_campaign_id uuid        REFERENCES campaigns(id) ON DELETE SET NULL,
    ADD COLUMN updated_by         uuid        REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN deleted_at         timestamptz;
CREATE INDEX templates_live_idx ON templates (workspace_id, kind, updated_at DESC) WHERE deleted_at IS NULL;

-- Images uploaded to the library. Bytes live in Postgres so a single binary
-- and one database are all an install needs; they are served publicly at
-- /media/{id}/{file} (email clients fetch them) with immutable caching.
CREATE TABLE media_assets (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    file_name    text        NOT NULL,
    content_type text        NOT NULL CHECK (content_type IN ('image/png','image/jpeg','image/gif')),
    size_bytes   integer     NOT NULL CHECK (size_bytes > 0),
    width        integer     NOT NULL CHECK (width > 0),
    height       integer     NOT NULL CHECK (height > 0),
    has_alpha    boolean     NOT NULL DEFAULT false,
    sha256       bytea       NOT NULL,
    data         bytea       NOT NULL,
    created_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    -- Removed from the library. Still served, so emails already sent keep their images.
    hidden_at    timestamptz
);
CREATE UNIQUE INDEX media_assets_sha_key ON media_assets (workspace_id, sha256);
CREATE INDEX media_assets_ws_idx ON media_assets (workspace_id, created_at DESC) WHERE hidden_at IS NULL;

-- URI blocklist results per linked domain (SURBL, Spamhaus DBL, URIBL…),
-- shared by every workspace. Filled by lookups (deliverability) and read by
-- the composer's inline checks and Send Readiness. Empty lists = clean.
CREATE TABLE link_domain_listings (
    domain     text        PRIMARY KEY CHECK (domain = lower(domain)),
    lists      text[]      NOT NULL DEFAULT '{}',
    source     text        NOT NULL DEFAULT '',
    checked_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE link_domain_listings;
DROP TABLE media_assets;
DROP INDEX templates_live_idx;
ALTER TABLE templates
    DROP COLUMN deleted_at,
    DROP COLUMN updated_by,
    DROP COLUMN source_campaign_id,
    DROP COLUMN size_bytes,
    DROP COLUMN thumb;
ALTER TABLE campaign_versions
    DROP COLUMN size_bytes,
    DROP COLUMN source,
    DROP COLUMN ab_test,
    DROP COLUMN plain_text;
ALTER TABLE campaigns DROP COLUMN content_settings, DROP COLUMN content_hash;
