-- +goose Up
-- Collaboration in the composer (review 2026-10, CO-10): comments on blocks
-- with @mentions, review links for people without an account, and who else
-- has the campaign open.

-- Review links: anyone with the link sees one version of one campaign and
-- can comment, without signing in, until it expires (7 days) or is revoked.
-- token_hash finds the link; token_sealed lets members copy it again.
CREATE TABLE campaign_review_links (
    id           uuid        PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    campaign_id  uuid        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    version      integer     NOT NULL,
    token_hash   bytea       NOT NULL,
    token_sealed bytea       NOT NULL,
    created_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL,
    expires_at   timestamptz NOT NULL,
    revoked_at   timestamptz,
    revoked_by   uuid        REFERENCES users(id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX campaign_review_links_token_key ON campaign_review_links (token_hash);
CREATE INDEX campaign_review_links_campaign_idx ON campaign_review_links (campaign_id, created_at DESC);

-- Comments: a thread's first comment has no thread_id; replies point to it.
-- anchor is the rc-comment id of the block a thread is on ('' = the whole
-- email); anchor_label says what it was ("the button “Read the docs”").
-- A reviewer's comment (review link) has a name instead of an author.
CREATE TABLE campaign_comments (
    id             uuid        PRIMARY KEY,
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    campaign_id    uuid        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    thread_id      uuid        REFERENCES campaign_comments(id) ON DELETE CASCADE,
    anchor         text        NOT NULL DEFAULT '' CHECK (length(anchor) <= 40),
    anchor_label   text        NOT NULL DEFAULT '' CHECK (length(anchor_label) <= 200),
    body           text        NOT NULL CHECK (length(body) BETWEEN 1 AND 4000),
    author_id      uuid        REFERENCES users(id) ON DELETE SET NULL,
    reviewer_name  text        NOT NULL DEFAULT '' CHECK (length(reviewer_name) <= 80),
    review_link_id uuid        REFERENCES campaign_review_links(id) ON DELETE SET NULL,
    version        integer     NOT NULL,
    mentions       uuid[]      NOT NULL DEFAULT '{}',
    resolved_at    timestamptz,
    resolved_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL
);
CREATE INDEX campaign_comments_campaign_idx ON campaign_comments (campaign_id, created_at);
CREATE INDEX campaign_comments_link_idx ON campaign_comments (review_link_id, created_at) WHERE review_link_id IS NOT NULL;

-- Who has a campaign open in the composer, refreshed while it stays open.
CREATE TABLE campaign_presence (
    campaign_id uuid        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    editing     boolean     NOT NULL DEFAULT false,
    seen_at     timestamptz NOT NULL,
    PRIMARY KEY (campaign_id, user_id)
);

-- +goose Down
DROP TABLE campaign_presence;
DROP TABLE campaign_comments;
DROP TABLE campaign_review_links;
