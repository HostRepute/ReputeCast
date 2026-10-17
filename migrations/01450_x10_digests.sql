-- +goose Up
-- Recurring and RSS digests (design expansion, phase 3, M20).
--
-- A digest reads a feed (RSS 2.0, Atom 1.0, JSON Feed 1.1 or a connected
-- WordPress site) and builds an issue from its new items on a schedule.
-- Every issue is a campaign: it goes through Send Readiness, the approval
-- rules and the sending engine like a campaign written by hand. The digest's
-- template is a campaign too (digest_role 'template'), so the composer edits
-- it with its versions, autosave and checks; the next issue, once someone
-- checks or tests it before its preparation, is a hidden campaign ('next')
-- that becomes the issue ('issue') when it is prepared.

-- Campaigns of a digest: its template, its next issue and its issues. The
-- template and the next issue stay out of the campaigns list.
ALTER TABLE campaigns
    ADD COLUMN digest_id   uuid,
    ADD COLUMN digest_role text NOT NULL DEFAULT '' CHECK (digest_role IN ('', 'template', 'next', 'issue'));

CREATE TABLE digests (
    id             uuid PRIMARY KEY,
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name           text        NOT NULL,
    -- The campaign the composer edits: content, subject and preheader (with
    -- feed tokens), audience, sender, route override and auto-pause
    -- thresholds. Each issue starts as a copy of it.
    template_id    uuid        REFERENCES campaigns(id) ON DELETE SET NULL,
    -- off: never turned on; on; paused (by a person, because the owner lost
    -- campaigns:send, or after three feed failures in a row).
    state          text        NOT NULL DEFAULT 'off' CHECK (state IN ('off', 'on', 'paused')),
    -- auto: each issue goes out at its time once its checks and approvals
    -- pass; draft: each issue becomes a draft someone sends.
    mode           text        NOT NULL DEFAULT 'auto' CHECK (mode IN ('auto', 'draft')),
    -- The feed: an address, or a WordPress site of the integrations hub
    -- (whose reported feed address is read).
    source         text        NOT NULL DEFAULT 'url' CHECK (source IN ('url', 'wordpress')),
    feed_url       text        NOT NULL DEFAULT '',
    site_id        uuid        REFERENCES integrations(id) ON DELETE SET NULL,
    -- Filters: items in any of these categories (empty: every category),
    -- with any of these tags (empty: any tag), not older than max_age_days.
    categories     text[]      NOT NULL DEFAULT '{}',
    tags           text[]      NOT NULL DEFAULT '{}',
    max_age_days   integer     CHECK (max_age_days BETWEEN 1 AND 365),
    -- The first issue: only items published after the digest was first
    -- turned on, or the latest N items.
    start_mode     text        NOT NULL DEFAULT 'after_on' CHECK (start_mode IN ('after_on', 'latest')),
    start_latest   integer     NOT NULL DEFAULT 3 CHECK (start_latest BETWEEN 1 AND 20),
    -- {"frequency":"weekly","weekday":5,"time":"08:00","timezone":"America/New_York"};
    -- daily ("weekdaysOnly"), monthly ("monthDay", or "monthRule":"first"
    -- with "weekday"), or items ("count", "windowFrom", "windowUntil").
    schedule       jsonb       NOT NULL DEFAULT '{}',
    prepare_hours  integer     NOT NULL DEFAULT 3 CHECK (prepare_hours BETWEEN 1 AND 24),
    late_hours     integer     NOT NULL DEFAULT 24 CHECK (late_hours IN (6, 24, 48)),
    archive        boolean     NOT NULL DEFAULT true,
    -- Issues send on the owner's authority: the person who turned it on.
    owner_id       uuid        REFERENCES users(id) ON DELETE SET NULL,
    owner_since    timestamptz,
    -- When it was first turned on (the default starting point).
    first_on_at    timestamptz,
    paused_at      timestamptz,
    paused_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    pause_reason   text        NOT NULL DEFAULT '' CHECK (pause_reason IN ('', 'person', 'owner', 'feed')),
    pause_note     text        NOT NULL DEFAULT '',
    -- What paused it automatically: {"from":"Editor","to":"Content Writer","notified":["…"]}.
    pause_detail   jsonb       NOT NULL DEFAULT '{}',
    -- Issues skipped in a row because the feed failed (three pause it).
    feed_skips     integer     NOT NULL DEFAULT 0,
    -- The feed as last read.
    feed_status    text        NOT NULL DEFAULT 'unread' CHECK (feed_status IN ('unread', 'ok', 'unreachable', 'invalid')),
    feed_kind      text        NOT NULL DEFAULT '' CHECK (feed_kind IN ('', 'rss', 'atom', 'json')),
    feed_title     text        NOT NULL DEFAULT '',
    feed_items     integer     NOT NULL DEFAULT 0,
    feed_bytes     integer     NOT NULL DEFAULT 0,
    feed_etag      text        NOT NULL DEFAULT '',
    feed_modified  text        NOT NULL DEFAULT '',
    feed_error     text        NOT NULL DEFAULT '',
    -- The first lines of what answered instead of a feed (an HTML page).
    feed_snippet   text        NOT NULL DEFAULT '',
    -- A permanent redirect seen on the last read: offered as "Use the new address".
    feed_moved_to  text        NOT NULL DEFAULT '',
    last_read_at   timestamptz,
    last_read_via  text        NOT NULL DEFAULT '',
    last_ok_at     timestamptz,
    created_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at     timestamptz NOT NULL DEFAULT now(),
    deleted_at     timestamptz
);
CREATE INDEX digests_workspace_idx ON digests (workspace_id, created_at) WHERE deleted_at IS NULL;
CREATE INDEX digests_on_idx ON digests (state) WHERE deleted_at IS NULL AND state = 'on';
CREATE INDEX digests_site_idx ON digests (site_id) WHERE deleted_at IS NULL;

ALTER TABLE campaigns ADD CONSTRAINT campaigns_digest_fk FOREIGN KEY (digest_id) REFERENCES digests(id) ON DELETE SET NULL;
CREATE INDEX campaigns_digest_idx ON campaigns (digest_id) WHERE digest_id IS NOT NULL;

-- The feed's items, matched by their id (guid, id), else their link, so an
-- edited post isn't sent twice. Excerpts are plain text; the image is
-- copied into the media library at the block's width.
CREATE TABLE digest_items (
    id             uuid PRIMARY KEY,
    digest_id      uuid        NOT NULL REFERENCES digests(id) ON DELETE CASCADE,
    key            text        NOT NULL,
    url            text        NOT NULL DEFAULT '',
    title          text        NOT NULL DEFAULT '',
    author         text        NOT NULL DEFAULT '',
    published_at   timestamptz,
    categories     text[]      NOT NULL DEFAULT '{}',
    tags           text[]      NOT NULL DEFAULT '{}',
    excerpt        text        NOT NULL DEFAULT '',
    -- The image as the feed gives it, its alt text (empty: the title is
    -- used, marked "from the title"), and the library copy.
    image_url      text        NOT NULL DEFAULT '',
    image_alt      text        NOT NULL DEFAULT '',
    media_id       uuid        REFERENCES media_assets(id) ON DELETE SET NULL,
    media_url      text        NOT NULL DEFAULT '',
    media_width    integer     NOT NULL DEFAULT 0,
    media_height   integer     NOT NULL DEFAULT 0,
    image_error    text        NOT NULL DEFAULT '',
    first_seen_at  timestamptz NOT NULL,
    -- How it was found: plugin (the WordPress plugin reported the post),
    -- schedule (the 30-minute read), preparation, person (Read now), setup.
    first_seen_via text        NOT NULL DEFAULT '',
    last_seen_at   timestamptz NOT NULL,
    -- No longer in the feed (it keeps its place in the issues it went in).
    gone_at        timestamptz,
    UNIQUE (digest_id, key)
);
CREATE INDEX digest_items_recent_idx ON digest_items (digest_id, published_at DESC NULLS LAST, first_seen_at DESC);

-- Issues: the one being collected or prepared, and the history (sent,
-- skipped with the reason, not sent). Numbers count sent issues only.
CREATE TABLE digest_issues (
    id            uuid PRIMARY KEY,
    digest_id     uuid        NOT NULL REFERENCES digests(id) ON DELETE CASCADE,
    workspace_id  uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    number        integer,
    -- collecting: before preparation; preparing: the feed failed at
    -- preparation and is retried; prepared: built, checked and scheduled
    -- (waiting for approval when a request is open); held: Send Readiness
    -- blocked it, or an approver asked for changes; draft: draft mode,
    -- waiting for someone to send it; sending; sent; skipped; not_sent.
    status        text        NOT NULL DEFAULT 'collecting'
                  CHECK (status IN ('collecting', 'preparing', 'prepared', 'held', 'draft', 'sending', 'sent', 'skipped', 'not_sent')),
    mode          text        NOT NULL DEFAULT 'auto' CHECK (mode IN ('auto', 'draft')),
    -- An item-count issue has no send time until it is prepared.
    send_at       timestamptz,
    prepare_at    timestamptz,
    -- send_at plus the late limit: not out by then, it is skipped.
    deadline      timestamptz,
    campaign_id   uuid        REFERENCES campaigns(id) ON DELETE SET NULL,
    prepared_at   timestamptz,
    held_at       timestamptz,
    hold_reason   text        NOT NULL DEFAULT '' CHECK (hold_reason IN ('', 'readiness', 'changes', 'rejected', 'no_approvers')),
    started_at    timestamptz,
    sent_at       timestamptz,
    skipped_at    timestamptz,
    skip_reason   text        NOT NULL DEFAULT '' CHECK (skip_reason IN ('', 'no_items', 'feed', 'not_approved', 'held', 'person')),
    skip_note     text        NOT NULL DEFAULT '',
    skipped_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    -- Reads that failed at preparation: [{"at":"…","error":"timed out"}].
    feed_tries    jsonb       NOT NULL DEFAULT '[]',
    item_count    integer     NOT NULL DEFAULT 0,
    -- What the issue was built on, for "since the last approved issue":
    -- {"template_version":7,"audience":{…},"route":{…},"schedule":"…","sender":"…"}.
    basis         jsonb       NOT NULL DEFAULT '{}',
    -- Notifications already sent for this issue (held, feed, draft).
    notified      text[]      NOT NULL DEFAULT '{}',
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX digest_issues_number_key ON digest_issues (digest_id, number) WHERE number IS NOT NULL;
-- One issue in flight per digest (the next one starts collecting once it
-- starts sending).
CREATE UNIQUE INDEX digest_issues_open_key ON digest_issues (digest_id) WHERE status IN ('collecting', 'preparing', 'prepared', 'held', 'draft');
CREATE INDEX digest_issues_digest_idx ON digest_issues (digest_id, coalesce(send_at, created_at) DESC);
CREATE INDEX digest_issues_campaign_idx ON digest_issues (campaign_id) WHERE campaign_id IS NOT NULL;
CREATE INDEX digest_issues_due_idx ON digest_issues (status, prepare_at, send_at) WHERE status IN ('collecting', 'preparing', 'prepared', 'held', 'draft', 'sending');

-- The items an issue went out with (or was built with), in order. Items
-- of a sent issue are used up; a skipped issue's stay new.
CREATE TABLE digest_issue_items (
    issue_id uuid    NOT NULL REFERENCES digest_issues(id) ON DELETE CASCADE,
    item_id  uuid    NOT NULL REFERENCES digest_items(id) ON DELETE CASCADE,
    position integer NOT NULL,
    PRIMARY KEY (issue_id, item_id)
);
CREATE INDEX digest_issue_items_item_idx ON digest_issue_items (item_id);

-- Every read of a digest's feed (its own log, like the transactional
-- message log: reads are automatic and frequent, so they aren't in the
-- audit log). Kept 90 days.
CREATE TABLE digest_feed_reads (
    id           bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    digest_id    uuid        NOT NULL REFERENCES digests(id) ON DELETE CASCADE,
    at           timestamptz NOT NULL,
    via          text        NOT NULL DEFAULT '',
    ok           boolean     NOT NULL,
    http_status  integer     NOT NULL DEFAULT 0,
    not_modified boolean     NOT NULL DEFAULT false,
    kind         text        NOT NULL DEFAULT '',
    items        integer     NOT NULL DEFAULT 0,
    new_items    integer     NOT NULL DEFAULT 0,
    bytes        integer     NOT NULL DEFAULT 0,
    ms           integer     NOT NULL DEFAULT 0,
    error        text        NOT NULL DEFAULT '',
    -- The redirects followed: [{"url":"…","status":301,"note":"…"}].
    hops         jsonb       NOT NULL DEFAULT '[]'
);
CREATE INDEX digest_feed_reads_digest_idx ON digest_feed_reads (digest_id, at DESC);

-- +goose Down
DROP TABLE digest_feed_reads;
DROP TABLE digest_issue_items;
DROP TABLE digest_issues;
DROP TABLE digest_items;
DROP INDEX campaigns_digest_idx;
ALTER TABLE campaigns DROP CONSTRAINT campaigns_digest_fk;
DROP TABLE digests;
ALTER TABLE campaigns DROP COLUMN digest_role, DROP COLUMN digest_id;
