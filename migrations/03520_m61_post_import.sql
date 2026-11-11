-- +goose Up
-- Importing posts from Substack, Ghost and WordPress (M61, G-26): an
-- export is uploaded, read in the background into staged posts, reviewed
-- (choices, tier and author mapping, a decision per post) and imported.
-- The old site's addresses lead to the imported posts.

CREATE TABLE website_imports (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    source       text        NOT NULL CHECK (source IN ('substack','ghost','wordpress')),
    status       text        NOT NULL DEFAULT 'uploaded'
                             CHECK (status IN ('uploaded','analysing','ready','importing','done','failed','canceled')),
    -- The choices: {"publish":"as_published","images":"copy","tiers":{"Paid subscribers":"<tier id>|any"},"author":{"Jordan Ellis":"<user id>|keep"}}.
    options      jsonb       NOT NULL DEFAULT '{}',
    -- What the analysis found and the run did, and where the run is:
    -- {"found":15,"drafts":3,…,"imported":12,"cursor":"<staged post id>"}.
    stats        jsonb       NOT NULL DEFAULT '{}',
    -- Why it failed, in English (the system's own words).
    error        text        NOT NULL DEFAULT '',
    -- The report (title, source URL, outcome, problems), kept 30 days.
    report_csv   bytea,
    file_name    text        NOT NULL DEFAULT '',
    file_size    bigint      NOT NULL DEFAULT 0,
    created_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    finished_at  timestamptz
);
CREATE INDEX website_imports_workspace_idx ON website_imports (workspace_id, created_at DESC);

-- The uploaded file, dropped once the import is done, failed or canceled.
CREATE TABLE website_import_files (
    import_id uuid  PRIMARY KEY REFERENCES website_imports(id) ON DELETE CASCADE,
    data      bytea NOT NULL
);

-- The posts the export holds, as read, with the decision for each.
CREATE TABLE website_import_posts (
    id           uuid PRIMARY KEY,
    import_id    uuid        NOT NULL REFERENCES website_imports(id) ON DELETE CASCADE,
    -- The platform's id of the post (Substack's post_id, Ghost's id, WordPress's wp:post_id).
    source_id    text        NOT NULL,
    title        text        NOT NULL DEFAULT '',
    -- The address it gets: the old one, or "<slug>-2" when another post holds it.
    slug         text        NOT NULL DEFAULT '',
    published_at timestamptz,
    -- As it was on the old site.
    status       text        NOT NULL CHECK (status IN ('published','draft')),
    visibility   text        NOT NULL CHECK (visibility IN ('public','members','paid')),
    -- The old site's tiers that could read it ("Paid subscribers"), mapped by the options.
    tier_refs    text[]      NOT NULL DEFAULT '{}',
    tags         text[]      NOT NULL DEFAULT '{}',
    author       text        NOT NULL DEFAULT '',
    excerpt      text        NOT NULL DEFAULT '',
    cover_url    text        NOT NULL DEFAULT '',
    -- {"title":"…","description":"…","canonicalUrl":"…","coverAlt":"…","sourceUrl":"…"}
    seo          jsonb       NOT NULL DEFAULT '{}',
    -- The old addresses that lead to the post ("p/my-post", "my-old-slug", "?p=123").
    legacy_paths text[]      NOT NULL DEFAULT '{}',
    html         text        NOT NULL DEFAULT '',
    images       integer     NOT NULL DEFAULT 0,
    -- [{"code":"embed_link"},{"code":"slug_taken","slug":"…"}]
    problems     jsonb       NOT NULL DEFAULT '[]',
    decision     text        NOT NULL DEFAULT 'import' CHECK (decision IN ('import','skip','update')),
    -- The post it became (or, already imported, the post it was).
    post_id      uuid        REFERENCES website_posts(id) ON DELETE SET NULL,
    UNIQUE (import_id, source_id)
);

-- An imported post remembers where it came from: importing it again finds
-- it instead of making a second one.
ALTER TABLE website_posts ADD COLUMN import_source text, ADD COLUMN import_source_id text;
CREATE UNIQUE INDEX website_posts_import_source ON website_posts (workspace_id, import_source, import_source_id)
    WHERE import_source IS NOT NULL AND deleted_at IS NULL;

-- The old site's addresses (lower case, no trailing slash, "?p=<id>" with
-- its query): on the site's own host they answer 301 to the post.
CREATE TABLE website_legacy_paths (
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    path         text        NOT NULL,
    post_id      uuid        NOT NULL REFERENCES website_posts(id) ON DELETE CASCADE,
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, path)
);
CREATE INDEX website_legacy_paths_post_idx ON website_legacy_paths (post_id);

-- +goose Down
DROP TABLE website_legacy_paths;
DROP INDEX website_posts_import_source;
ALTER TABLE website_posts DROP COLUMN import_source_id, DROP COLUMN import_source;
DROP TABLE website_import_posts;
DROP TABLE website_import_files;
DROP TABLE website_imports;
