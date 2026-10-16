-- +goose Up
-- Accessibility checks (design expansion, M19).

-- The workspace's policy (Settings › General › Accessibility): whether
-- serious issues only warn or block sends and publishing, and the smallest
-- text size. A workspace without a row warns, at 12 px.
CREATE TABLE accessibility_policies (
    workspace_id uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    on_serious   text        NOT NULL DEFAULT 'warn' CHECK (on_serious IN ('warn', 'block')),
    min_text_px  integer     NOT NULL DEFAULT 12 CHECK (min_text_px BETWEEN 12 AND 16),
    updated_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- Titles of linked pages, read for the link-text suggestions ("Use
-- 'Scheduled rollouts · Acme Docs'"). Public pages only, shared by
-- workspaces like the URI blocklist results.
CREATE TABLE accessibility_link_titles (
    url        text PRIMARY KEY,
    title      text        NOT NULL DEFAULT '',
    lang       text        NOT NULL DEFAULT '',
    status     integer     NOT NULL DEFAULT 0,
    fetched_at timestamptz NOT NULL DEFAULT now()
);

-- One-click fixes applied from Review & send. Send Readiness shows the
-- latest as a passing row while the content is still what the fix saved.
CREATE TABLE accessibility_fixes (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    campaign_id  uuid        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    content_hash text        NOT NULL,
    kinds        text[]      NOT NULL DEFAULT '{}',
    summary      text        NOT NULL DEFAULT '',
    languages    text[]      NOT NULL DEFAULT '{}',
    fixed_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    fixed_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX accessibility_fixes_campaign_idx ON accessibility_fixes (campaign_id, fixed_at DESC);

-- +goose Down
DROP TABLE accessibility_fixes;
DROP TABLE accessibility_link_titles;
DROP TABLE accessibility_policies;
