-- +goose Up
-- The publication website (design expansion phase 7, M28): the publisher's
-- site at their custom domain, grown out of the public archive. Posts come
-- from campaigns ("Publish to web") or are written for the web only; tags,
-- the home page and the navigation shape the site; members sign in with a
-- link and a 6-digit code; visibility, the free preview and the meter gate
-- posts on the server.
--
-- Nothing here identifies a visitor: views are counted from hashes with a
-- salt that changes daily, and the hashes are deleted with their day.

-- One row per workspace, created the first time the Website tab is read.
CREATE TABLE website_settings (
    workspace_id        uuid        PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    -- off: the archive; turning_on: the job is creating posts and
    -- redirects; on: the website.
    status              text        NOT NULL DEFAULT 'off' CHECK (status IN ('off','turning_on','on')),
    turned_on_at        timestamptz,
    turned_on_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    turned_off_at       timestamptz,
    turned_off_by       uuid        REFERENCES users(id) ON DELETE SET NULL,
    -- Turning on, step by step: {"started_at":"…","tags":{"done":2,"total":2,"names":["Weekly Digest"]},
    -- "posts":{"done":140,"total":182},"redirects":{"done":96,"total":182},"pages":"waiting|done","finished_at":"…"}.
    progress            jsonb       NOT NULL DEFAULT '{}',
    -- Site identity and search.
    title               text        NOT NULL DEFAULT '',
    tagline             text        NOT NULL DEFAULT '',
    description         text        NOT NULL DEFAULT '',
    social_image        text        NOT NULL DEFAULT '',
    index_site          boolean     NOT NULL DEFAULT true,
    -- Home: featured_and_latest, latest_list or grid; the featured posts in
    -- their order (up to 3); the sections in their order:
    -- [{"kind":"subscribe","on":true},{"kind":"featured","on":true},{"kind":"latest","on":true,"count":6},
    --  {"kind":"tag","on":true,"tag":"…","count":3},{"kind":"about","on":true},{"kind":"tiers","on":true}].
    home_layout         text        NOT NULL DEFAULT 'featured' CHECK (home_layout IN ('featured','list','grid')),
    featured            uuid[]      NOT NULL DEFAULT '{}',
    sections            jsonb       NOT NULL DEFAULT '[]',
    about_title         text        NOT NULL DEFAULT '',
    about_text          text        NOT NULL DEFAULT '',
    about_link          text        NOT NULL DEFAULT '',
    -- Navigation: header links (up to 6) [{"kind":"tag|page|home|archive|url","ref":"…","label":"…","url":"…"}]
    -- and the footer's switches {"archive":true,"rss":true,"preferences":true}.
    header_links        jsonb       NOT NULL DEFAULT '[]',
    footer              jsonb       NOT NULL DEFAULT '{}',
    -- Members and sign-in.
    signin_on           boolean     NOT NULL DEFAULT true,
    link_minutes        integer     NOT NULL DEFAULT 30 CHECK (link_minutes IN (10, 15, 30, 60)),
    session_days        integer     NOT NULL DEFAULT 90 CHECK (session_days BETWEEN 7 AND 365),
    signup_on           boolean     NOT NULL DEFAULT true,
    -- The lists offered on sign-up and the ones checked: [{"list":"…","checked":true}].
    signup_lists        jsonb       NOT NULL DEFAULT '[]',
    -- Default visibility: of web-only posts and pages, and of posts from
    -- campaigns ("audience" follows the campaign: paid-only becomes Paid).
    default_visibility  text        NOT NULL DEFAULT 'public' CHECK (default_visibility IN ('public','members','paid')),
    campaign_visibility text        NOT NULL DEFAULT 'audience' CHECK (campaign_visibility IN ('audience','public','members','paid')),
    -- The meter: both rules off by default; calendar months.
    meter_visitors_on   boolean     NOT NULL DEFAULT false,
    meter_visitors      integer     NOT NULL DEFAULT 2 CHECK (meter_visitors BETWEEN 1 AND 20),
    meter_members_on    boolean     NOT NULL DEFAULT false,
    meter_members       integer     NOT NULL DEFAULT 1 CHECK (meter_members BETWEEN 1 AND 20),
    -- The paywall card: {"heading":"…","text":"…","tiers":["…"],"upgrade":"Upgrade","signup":"Sign up free","free":"…"}.
    paywall             jsonb       NOT NULL DEFAULT '{}',
    -- When the domain or certificate problem was last notified (once per problem).
    domain_notified     text        NOT NULL DEFAULT '',
    updated_at          timestamptz NOT NULL DEFAULT now(),
    updated_by          uuid        REFERENCES users(id) ON DELETE SET NULL
);

-- Tags: each list with archived campaigns becomes one when the website is
-- turned on; more can be made by hand. A post has up to 5.
CREATE TABLE website_tags (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name         text        NOT NULL,
    slug         text        NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    description  text        NOT NULL DEFAULT '',
    -- The list it was made from, when turning the website on.
    list_id      uuid        REFERENCES lists(id) ON DELETE SET NULL,
    created_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX website_tags_slug ON website_tags (workspace_id, slug);

-- Posts: from a campaign (kind campaign, the content is the campaign's) or
-- written for the web (kind web, the content is the post's own MJML).
CREATE TABLE website_posts (
    id               uuid PRIMARY KEY,
    workspace_id     uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    kind             text        NOT NULL CHECK (kind IN ('campaign','web')),
    campaign_id      uuid        REFERENCES campaigns(id) ON DELETE SET NULL,
    -- A campaign's "Publish to <domain>" switch: off keeps the settings but
    -- the campaign isn't published.
    web_on           boolean     NOT NULL DEFAULT true,
    status           text        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','scheduled','published','unpublished')),
    -- When it goes live: with the campaign's send, at a time, or now.
    publish_mode     text        NOT NULL DEFAULT 'now' CHECK (publish_mode IN ('with_send','at','now')),
    publish_at       timestamptz,
    publish_tz       text        NOT NULL DEFAULT '',
    title            text        NOT NULL DEFAULT '',
    -- The title and excerpt default to the campaign's subject and preheader
    -- until someone edits them.
    title_edited     boolean     NOT NULL DEFAULT false,
    slug             text        NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(slug) <= 120),
    excerpt          text        NOT NULL DEFAULT '' CHECK (length(excerpt) <= 200),
    excerpt_edited   boolean     NOT NULL DEFAULT false,
    cover_url        text        NOT NULL DEFAULT '',
    cover_alt        text        NOT NULL DEFAULT '',
    -- The byline: a member of the workspace, or a written name.
    byline_user_id   uuid        REFERENCES users(id) ON DELETE SET NULL,
    byline_name      text        NOT NULL DEFAULT '',
    visibility       text        NOT NULL DEFAULT 'public' CHECK (visibility IN ('public','members','paid')),
    -- Paid posts: the tiers that may read it (empty: every paid tier).
    tiers            uuid[]      NOT NULL DEFAULT '{}',
    -- Left out of the meter ("Never free").
    never_free       boolean     NOT NULL DEFAULT false,
    seo_title        text        NOT NULL DEFAULT '',
    seo_description  text        NOT NULL DEFAULT '',
    social_image     text        NOT NULL DEFAULT '',
    canonical_url    text        NOT NULL DEFAULT '',
    hide_from_search boolean     NOT NULL DEFAULT false,
    -- Web-only posts: the content as MJML (the composer's post mode), its
    -- version for autosave and conflicts.
    mjml             text        NOT NULL DEFAULT '',
    version          integer     NOT NULL DEFAULT 1,
    autosaved_at     timestamptz,
    -- The web version, drawn by internal/website/render when the post or
    -- its campaign changes: the whole body, the part before the free
    -- preview marker, their plain text (search reads the public part only),
    -- the words and the blocks left out.
    body_html        text        NOT NULL DEFAULT '',
    preview_html     text        NOT NULL DEFAULT '',
    has_marker       boolean     NOT NULL DEFAULT false,
    marker_after     integer     NOT NULL DEFAULT 0,
    body_text        text        NOT NULL DEFAULT '',
    preview_text     text        NOT NULL DEFAULT '',
    words            integer     NOT NULL DEFAULT 0,
    -- {"hidden":[{"name":"Footer","reason":"email_only"}],"fallbacks":[{"token":"first_name","fallback":"there","sample":"Hi there"}],"paid":["…"]}
    render_notes     jsonb       NOT NULL DEFAULT '{}',
    rendered_hash    text        NOT NULL DEFAULT '',
    published_at     timestamptz,
    published_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    first_published_at timestamptz,
    unpublished_at   timestamptz,
    unpublished_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    unpublish_reason text        NOT NULL DEFAULT '',
    -- An unpublished post's address answers 410, or redirects here (a path on the site).
    redirect_to      text        NOT NULL DEFAULT '',
    -- A web-only post sent as a campaign: the campaign draft it became.
    sent_as_campaign uuid        REFERENCES campaigns(id) ON DELETE SET NULL,
    created_by       uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_by       uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at       timestamptz NOT NULL DEFAULT now(),
    deleted_at       timestamptz
);
CREATE UNIQUE INDEX website_posts_slug ON website_posts (workspace_id, slug) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX website_posts_campaign ON website_posts (campaign_id) WHERE campaign_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX website_posts_list_idx ON website_posts (workspace_id, status, published_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX website_posts_due_idx ON website_posts (publish_at) WHERE status IN ('draft','scheduled') AND deleted_at IS NULL;

CREATE TABLE website_post_tags (
    post_id  uuid     NOT NULL REFERENCES website_posts(id) ON DELETE CASCADE,
    tag_id   uuid     NOT NULL REFERENCES website_tags(id) ON DELETE CASCADE,
    position smallint NOT NULL DEFAULT 0,
    PRIMARY KEY (post_id, tag_id)
);
CREATE INDEX website_post_tags_tag_idx ON website_post_tags (tag_id);

-- Earlier addresses of posts (a changed slug, an archive issue's address):
-- they redirect (301) to the post.
CREATE TABLE website_redirects (
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    -- "posts/feature-flags" or "archive/the-case-for-boring-infrastructure".
    path         text        NOT NULL,
    post_id      uuid        NOT NULL REFERENCES website_posts(id) ON DELETE CASCADE,
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, path)
);
CREATE INDEX website_redirects_post_idx ON website_redirects (post_id);

-- Saved versions of web-only posts (Version history in post mode).
CREATE TABLE website_post_versions (
    post_id  uuid        NOT NULL REFERENCES website_posts(id) ON DELETE CASCADE,
    version  integer     NOT NULL,
    title    text        NOT NULL,
    mjml     text        NOT NULL,
    saved_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    saved_at timestamptz NOT NULL DEFAULT now(),
    label    text        NOT NULL DEFAULT '',
    PRIMARY KEY (post_id, version)
);

-- Views, counted on the server without cookies: one view per visitor,
-- page and day. Pages are posts, the home page, the archive and tag pages.
CREATE TABLE website_view_days (
    workspace_id uuid   NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    day          date   NOT NULL,
    page_kind    text   NOT NULL CHECK (page_kind IN ('post','home','archive','tag','page')),
    -- The post, tag or page (the nil UUID for the home page and the archive).
    page_id      uuid   NOT NULL,
    views        bigint NOT NULL DEFAULT 0,
    PRIMARY KEY (workspace_id, day, page_kind, page_id)
);
-- Visitors per day on the whole site (the same visitor on two days counts twice).
CREATE TABLE website_visitor_days (
    workspace_id uuid   NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    day          date   NOT NULL,
    visitors     bigint NOT NULL DEFAULT 0,
    PRIMARY KEY (workspace_id, day)
);
-- Where a post's views came from: the referring domain ("" for email
-- clients and direct) and the UTM source.
CREATE TABLE website_view_sources (
    workspace_id uuid   NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    post_id      uuid   NOT NULL REFERENCES website_posts(id) ON DELETE CASCADE,
    day          date   NOT NULL,
    dim          text   NOT NULL CHECK (dim IN ('referrer','utm')),
    key          text   NOT NULL,
    views        bigint NOT NULL DEFAULT 0,
    PRIMARY KEY (post_id, day, dim, key)
);
-- The day's salt and the hashes already counted that day. Both are
-- deleted once the day is over: nothing links a visitor across days.
CREATE TABLE website_view_salts (
    day  date  PRIMARY KEY,
    salt bytea NOT NULL
);
CREATE TABLE website_view_seen (
    day  date  NOT NULL,
    hash bytea NOT NULL,
    PRIMARY KEY (day, hash)
);

-- Members who joined from a post (signups) and readers who upgraded to a
-- paid tier from one: the post's numbers.
CREATE TABLE website_post_conversions (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    post_id      uuid        NOT NULL REFERENCES website_posts(id) ON DELETE CASCADE,
    kind         text        NOT NULL CHECK (kind IN ('signup','upgrade')),
    contact_id   uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    occurred_at  timestamptz NOT NULL
);
CREATE INDEX website_post_conversions_idx ON website_post_conversions (workspace_id, post_id, kind, occurred_at);

-- Members: contacts who signed in to the website. Being a member doesn't
-- mean receiving newsletters.
CREATE TABLE website_members (
    contact_id        uuid        PRIMARY KEY REFERENCES contacts(id) ON DELETE CASCADE,
    workspace_id      uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    first_signed_in   timestamptz NOT NULL,
    last_signed_in    timestamptz NOT NULL,
    last_method       text        NOT NULL DEFAULT 'link' CHECK (last_method IN ('link','code','signup')),
    sign_ins          integer     NOT NULL DEFAULT 1
);
CREATE INDEX website_members_ws_idx ON website_members (workspace_id, last_signed_in DESC);

-- A member's website events, shown on the contact's profile: sign-ins,
-- refused requests (a suppressed address), sign-outs everywhere.
CREATE TABLE website_member_events (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    contact_id   uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    kind         text        NOT NULL CHECK (kind IN ('signed_in','link_refused','signed_out_everywhere','joined','email_changed')),
    -- link, code or signup; the person who signed the member out.
    method       text        NOT NULL DEFAULT '',
    same_browser boolean,
    actor_id     uuid        REFERENCES users(id) ON DELETE SET NULL,
    detail       text        NOT NULL DEFAULT '',
    occurred_at  timestamptz NOT NULL
);
CREATE INDEX website_member_events_contact_idx ON website_member_events (contact_id, occurred_at DESC);

-- Sign-in links: a link and a 6-digit code, both for one sign-in within
-- link_minutes. A newer link replaces older ones. Only hashes are kept.
CREATE TABLE website_signin_links (
    id            uuid PRIMARY KEY,
    workspace_id  uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    contact_id    uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    -- signin; signup confirms a sign-up from the site; email confirms a new address.
    purpose       text        NOT NULL DEFAULT 'signin' CHECK (purpose IN ('signin','signup','email')),
    token_hash    bytea       NOT NULL,
    code_hash     bytea       NOT NULL,
    -- The browser that asked (a hash of its sign-in cookie), so the code
    -- is typed where the link was asked for.
    browser_hash  bytea       NOT NULL,
    -- Where the reader goes after signing in (a path on the site).
    return_to     text        NOT NULL DEFAULT '',
    -- A new address waiting for its confirmation (purpose email).
    new_email     text        NOT NULL DEFAULT '',
    -- Lists picked on sign-up (purpose signup).
    lists         uuid[]      NOT NULL DEFAULT '{}',
    created_at    timestamptz NOT NULL,
    expires_at    timestamptz NOT NULL,
    used_at       timestamptz,
    used_same_browser boolean,
    replaced_at   timestamptz,
    code_tries    integer     NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX website_signin_links_token ON website_signin_links (token_hash);
CREATE INDEX website_signin_links_contact_idx ON website_signin_links (contact_id, created_at DESC);
CREATE INDEX website_signin_links_browser_idx ON website_signin_links (browser_hash, created_at DESC);

-- Members' sessions: a first-party cookie on the site's domain. Only the
-- token's hash is kept.
CREATE TABLE website_sessions (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    contact_id   uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    token_hash   bytea       NOT NULL,
    method       text        NOT NULL DEFAULT 'link' CHECK (method IN ('link','code','signup')),
    created_at   timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL,
    expires_at   timestamptz NOT NULL,
    ended_at     timestamptz,
    end_reason   text        NOT NULL DEFAULT '' CHECK (end_reason IN ('','signed_out','everywhere','email_changed','expired'))
);
CREATE UNIQUE INDEX website_sessions_token ON website_sessions (token_hash);
CREATE INDEX website_sessions_contact_idx ON website_sessions (contact_id) WHERE ended_at IS NULL;

-- The meter's reads: one row per reader, calendar month and gated post.
-- Readers are members (by contact) or visitors (by the hash of their
-- browser's meter cookie).
CREATE TABLE website_meter_reads (
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    month        date        NOT NULL,
    reader       text        NOT NULL,
    post_id      uuid        NOT NULL REFERENCES website_posts(id) ON DELETE CASCADE,
    -- visitor (a members-only post read without signing in) or member (a paid post read for free).
    rule         text        NOT NULL CHECK (rule IN ('visitor','member')),
    read_at      timestamptz NOT NULL,
    PRIMARY KEY (workspace_id, month, reader, post_id)
);

-- Settings › Approvals: "Publishing a post or page needs approval" (off by
-- default), and approval requests for posts.
ALTER TABLE approval_requests DROP CONSTRAINT approval_requests_subject_type_check;
ALTER TABLE approval_requests ADD CONSTRAINT approval_requests_subject_type_check
    CHECK (subject_type IN ('campaign','automation','transactional_template','website_post'));
ALTER TABLE approval_requests
    ADD COLUMN website_post_id uuid REFERENCES website_posts(id) ON DELETE CASCADE;
CREATE UNIQUE INDEX approval_requests_open_post ON approval_requests (website_post_id)
    WHERE status = 'pending' AND website_post_id IS NOT NULL;
CREATE INDEX approval_requests_post_idx ON approval_requests (website_post_id) WHERE website_post_id IS NOT NULL;

-- +goose Down
DROP INDEX approval_requests_post_idx;
DROP INDEX approval_requests_open_post;
DELETE FROM approval_requests WHERE subject_type = 'website_post';
ALTER TABLE approval_requests DROP COLUMN website_post_id;
ALTER TABLE approval_requests DROP CONSTRAINT approval_requests_subject_type_check;
ALTER TABLE approval_requests ADD CONSTRAINT approval_requests_subject_type_check
    CHECK (subject_type IN ('campaign','automation','transactional_template'));
DROP TABLE website_meter_reads;
DROP TABLE website_sessions;
DROP TABLE website_signin_links;
DROP TABLE website_member_events;
DROP TABLE website_members;
DROP TABLE website_post_conversions;
DROP TABLE website_view_seen;
DROP TABLE website_view_salts;
DROP TABLE website_view_sources;
DROP TABLE website_visitor_days;
DROP TABLE website_view_days;
DROP TABLE website_post_versions;
DROP TABLE website_redirects;
DROP TABLE website_post_tags;
DROP TABLE website_posts;
DROP TABLE website_tags;
DROP TABLE website_settings;
