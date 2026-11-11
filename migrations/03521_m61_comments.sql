-- +goose Up
-- Comments on posts (M61, G-27): members comment under the posts they can
-- read, the team moderates them in Website › Comments and replies as the
-- publication, and a member hears by email when someone replies to them.

-- Who may comment (off for every site until someone turns it on, signed-in
-- members, paid members only) and which comments wait for review (every
-- comment, a member's first one, none); when the people who moderate were
-- last told that comments are waiting.
ALTER TABLE website_settings
    ADD COLUMN comments text NOT NULL DEFAULT 'off' CHECK (comments IN ('off','members','paid')),
    ADD COLUMN comments_review text NOT NULL DEFAULT 'first' CHECK (comments_review IN ('all','first','none')),
    ADD COLUMN comments_notified_at timestamptz;

-- A post's own switch: "Off for this post".
ALTER TABLE website_posts ADD COLUMN comments_off boolean NOT NULL DEFAULT false;

-- The name a member's comments show (never their address), a moderator's
-- block, and whether replies to their comments are emailed.
ALTER TABLE website_members
    ADD COLUMN display_name text NOT NULL DEFAULT '' CHECK (char_length(display_name) <= 60),
    ADD COLUMN comments_blocked_at timestamptz,
    ADD COLUMN comment_emails boolean NOT NULL DEFAULT true;

-- One level of replies: a reply's parent is a top-level comment. A team
-- reply (team) has no contact and names the member who wrote it (it shows
-- as the publication). A member's comment whose contact is gone shows as
-- removed, so the replies of other members under it stay. held_reason
-- says why a comment waits for review. ip is kept 30 days for the
-- per-network limit (the website's hourly job clears it). A deleted
-- comment's text goes once its Undo has passed; erasing the contact empties
-- it at once. decision_id, undo_until and before_* are the last moderation
-- decision, which Undo takes back for 10 seconds.
CREATE TABLE website_comments (
    id                  uuid        PRIMARY KEY,
    workspace_id        uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    post_id             uuid        NOT NULL REFERENCES website_posts(id) ON DELETE CASCADE,
    parent_id           uuid        REFERENCES website_comments(id) ON DELETE CASCADE,
    contact_id          uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    team                boolean     NOT NULL DEFAULT false,
    user_id             uuid        REFERENCES users(id) ON DELETE SET NULL,
    body                text        NOT NULL,
    status              text        NOT NULL CHECK (status IN ('pending','published','hidden','deleted')),
    held_reason         text        NOT NULL DEFAULT '' CHECK (held_reason IN ('','first_comment','review_all','links','blocked')),
    pinned_at           timestamptz,
    ip                  inet,
    created_at          timestamptz NOT NULL,
    edited_at           timestamptz,
    published_at        timestamptz,
    moderated_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    moderated_at        timestamptz,
    decision_id         uuid,
    undo_until          timestamptz,
    before_status       text        NOT NULL DEFAULT '',
    before_published_at timestamptz,
    CHECK (char_length(body) <= 5000 AND (char_length(body) >= 1 OR status = 'deleted'))
);
CREATE INDEX website_comments_post_idx ON website_comments (post_id, status, created_at);
CREATE INDEX website_comments_ws_idx ON website_comments (workspace_id, status, created_at DESC);
CREATE INDEX website_comments_contact_idx ON website_comments (contact_id);
CREATE INDEX website_comments_parent_idx ON website_comments (parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX website_comments_decision_idx ON website_comments (decision_id) WHERE decision_id IS NOT NULL;
CREATE INDEX website_comments_ip_idx ON website_comments (created_at) WHERE ip IS NOT NULL;

-- Reply emails waiting to go: a member gets at most one an hour. sent_at
-- is when the email that covered the notice went; sent notices are kept a
-- week for the hourly cap.
CREATE TABLE website_comment_notices (
    contact_id uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    comment_id uuid        NOT NULL REFERENCES website_comments(id) ON DELETE CASCADE,
    queued_at  timestamptz NOT NULL,
    sent_at    timestamptz,
    PRIMARY KEY (contact_id, comment_id)
);
CREATE INDEX website_comment_notices_due_idx ON website_comment_notices (queued_at) WHERE sent_at IS NULL;
CREATE INDEX website_comment_notices_comment_idx ON website_comment_notices (comment_id);

-- +goose Down
DROP TABLE website_comment_notices;
DROP TABLE website_comments;
ALTER TABLE website_members DROP COLUMN comment_emails, DROP COLUMN comments_blocked_at, DROP COLUMN display_name;
ALTER TABLE website_posts DROP COLUMN comments_off;
ALTER TABLE website_settings DROP COLUMN comments_notified_at, DROP COLUMN comments_review, DROP COLUMN comments;
