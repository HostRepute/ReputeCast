-- +goose Up
-- Web push (design expansion phase 8, M32): the workspace's VAPID keys,
-- the sites readers turn notifications on from (the publication website,
-- whose service worker ReputeCast serves, and other sites with the snippet
-- and rc-push-sw.js), browser subscriptions with their consent records,
-- push prompts (a form type), push campaigns, and what each notification
-- did.
--
-- Push builds on X21's channel model: campaigns carry channel 'push', and
-- every notification is a row of the message log (channel_messages, kept 90
-- days, never in the audit log). push_messages adds what only push has: the
-- subscription, its site and browser, and when it was shown and clicked.

-- The key pair that identifies the workspace to the browsers' push
-- services. The private key is sealed (internal/secrets); rotating keeps the
-- old row for the record and ends every subscription made with it.
CREATE TABLE push_keys (
    id               uuid PRIMARY KEY,
    workspace_id     uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    -- The uncompressed P-256 point, base64url: the browsers'
    -- applicationServerKey.
    public_key       text        NOT NULL,
    private_enc      bytea       NOT NULL,
    -- The contact the push services get with every token (the JWT's "sub").
    contact          text        NOT NULL,
    created_by       uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at       timestamptz NOT NULL,
    rotated_at       timestamptz,
    rotated_by       uuid        REFERENCES users(id) ON DELETE SET NULL,
    -- The last time a push service accepted a message signed with it.
    last_accepted_at timestamptz
);
CREATE UNIQUE INDEX push_keys_live_idx ON push_keys (workspace_id) WHERE rotated_at IS NULL;

-- The guards every notification goes through. Quiet hours use the window
-- of the SMS rules (sms_settings: quiet_from, quiet_to), and whether push
-- keeps to it is sms_settings.push_same_window once the workspace has SMS
-- rules; quiet_hours holds the choice until then.
CREATE TABLE push_settings (
    workspace_id  uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    frequency_cap integer     NOT NULL DEFAULT 2 CHECK (frequency_cap BETWEEN 1 AND 24),
    quiet_hours   boolean     NOT NULL DEFAULT true,
    updated_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at    timestamptz NOT NULL
);

-- Sites where readers can turn notifications on. Browsers accept a
-- service worker only from the site's own origin, so each site serves one:
-- ReputeCast serves it on the publication website; another site hosts
-- rc-push-sw.js (or one import line in its own worker), and the check
-- fetches it to see whether it's there, its version and its scope.
CREATE TABLE push_sites (
    id                uuid PRIMARY KEY,
    workspace_id      uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    -- "https://acme.io": subscriptions from any other origin are refused.
    origin            text        NOT NULL,
    host              text        NOT NULL,
    kind              text        NOT NULL CHECK (kind IN ('publication','snippet')),
    -- "Company site"
    label             text        NOT NULL DEFAULT '',
    -- Where the worker answers on a snippet site, and whether the site
    -- hosts our file or imports our worker into its own.
    worker_path       text        NOT NULL DEFAULT '/rc-push-sw.js',
    worker_mode       text        NOT NULL DEFAULT 'file' CHECK (worker_mode IN ('file','import')),
    -- What the last check found.
    check_status      text        NOT NULL DEFAULT 'unchecked'
        CHECK (check_status IN ('unchecked','found','old_version','missing','unreachable','not_a_worker')),
    check_version     integer,
    check_scope       text        NOT NULL DEFAULT '',
    check_http_status integer,
    check_detail      text        NOT NULL DEFAULT '',
    checked_at        timestamptz,
    -- When the check last saw the worker change to missing, so it notifies once.
    missing_since     timestamptz,
    -- The notifications' icon (192 × 192); empty uses the workspace logo.
    icon_media_id     uuid        REFERENCES media_assets(id) ON DELETE SET NULL,
    -- The first time the snippet asked for its prompt ("snippet installed 5 Oct").
    snippet_seen_at   timestamptz,
    -- The demo's sites answer the check from the demo data: they're never fetched.
    demo              boolean     NOT NULL DEFAULT false,
    added_by          uuid        REFERENCES users(id) ON DELETE SET NULL,
    added_at          timestamptz NOT NULL,
    removed_at        timestamptz,
    removed_by        uuid        REFERENCES users(id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX push_sites_origin_idx ON push_sites (workspace_id, origin) WHERE removed_at IS NULL;

-- Browser subscriptions. A subscription is anonymous until ReputeCast
-- knows who it is (a signed-in member, a reader from an email or text link
-- with a token, or the account page): then it's linked to that contact.
-- It keeps its consent record like email consent: the page, the exact
-- prompt text and its version, the IP address and the time. A removed
-- subscription loses its endpoint and keys and keeps the rest, for counts.
CREATE TABLE push_subscriptions (
    id             uuid PRIMARY KEY,
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    site_id        uuid        NOT NULL REFERENCES push_sites(id) ON DELETE CASCADE,
    key_id         uuid        NOT NULL REFERENCES push_keys(id) ON DELETE CASCADE,
    -- The push resource; '' once removed.
    endpoint       text        NOT NULL,
    p256dh         bytea,
    auth           bytea,
    browser        text        NOT NULL CHECK (browser IN ('chrome','chrome_android','edge','firefox','safari','safari_ios','other')),
    platform       text        NOT NULL DEFAULT '',
    -- The browser's time zone, for quiet hours and the frequency cap's day.
    time_zone      text        NOT NULL DEFAULT '',
    contact_id     uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    linked_at      timestamptz,
    -- member: signed in on the website; link: arrived from an email or a
    -- text with a token; account: turned on from the account page.
    linked_by      text        NOT NULL DEFAULT '' CHECK (linked_by IN ('','member','link','account','preferences')),
    -- The consent record.
    prompt_form_id uuid        REFERENCES forms(id) ON DELETE SET NULL,
    prompt_version integer,
    prompt_text    text        NOT NULL DEFAULT '',
    page           text        NOT NULL DEFAULT '',
    ip             inet,
    user_agent     text        NOT NULL DEFAULT '',
    created_at     timestamptz NOT NULL,
    -- The service worker reported in (shown, clicked).
    last_seen_at   timestamptz,
    -- Consecutive refusals (403) from the push service.
    failures       integer     NOT NULL DEFAULT 0,
    removed_at     timestamptz,
    -- gone: the push service answered 404 or 410; unsubscribed: the
    -- browser said so; turned_off: the reader, on the account page or the
    -- preference center; rotated: the keys were rotated; removed: by a
    -- person, from the contact profile.
    removed_reason text        NOT NULL DEFAULT ''
        CHECK (removed_reason IN ('','gone','unsubscribed','turned_off','rotated','site_removed','rejected','removed')),
    demo           boolean     NOT NULL DEFAULT false
);
CREATE UNIQUE INDEX push_subscriptions_endpoint_idx ON push_subscriptions (workspace_id, endpoint) WHERE removed_at IS NULL;
CREATE INDEX push_subscriptions_site_idx ON push_subscriptions (site_id) WHERE removed_at IS NULL;
CREATE INDEX push_subscriptions_contact_idx ON push_subscriptions (contact_id) WHERE removed_at IS NULL AND contact_id IS NOT NULL;
CREATE INDEX push_subscriptions_removed_idx ON push_subscriptions (workspace_id, removed_at) WHERE removed_at IS NOT NULL;

-- A member's own browser, for "Send to my browser".
CREATE TABLE push_test_subscriptions (
    user_id      uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    key_id       uuid        NOT NULL REFERENCES push_keys(id) ON DELETE CASCADE,
    id           uuid        NOT NULL UNIQUE,
    endpoint     text        NOT NULL,
    p256dh       bytea       NOT NULL,
    auth         bytea       NOT NULL,
    browser      text        NOT NULL,
    platform     text        NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL,
    PRIMARY KEY (user_id, workspace_id)
);

-- Push prompts are a form type (Forms & pages › Signup forms › New form ›
-- "Push prompt"): the forms row holds the name, status and who made it;
-- push_prompts holds the prompt.
ALTER TABLE forms DROP CONSTRAINT forms_kind_check;
ALTER TABLE forms ADD CONSTRAINT forms_kind_check CHECK (kind IN ('embedded','popup','hosted','push'));

CREATE TABLE push_prompts (
    form_id      uuid PRIMARY KEY REFERENCES forms(id) ON DELETE CASCADE,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    site_id      uuid        REFERENCES push_sites(id) ON DELETE SET NULL,
    -- The prompt being edited: style, text, when, where, frequency and the
    -- iPhone behaviour (push.PromptConfig).
    draft        jsonb       NOT NULL,
    live_version integer     NOT NULL DEFAULT 0,
    -- The first publication ("Live since 3 Oct").
    live_since   timestamptz,
    updated_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at   timestamptz NOT NULL
);

-- Each published version: consent records point to the version that was
-- live when the reader allowed notifications.
CREATE TABLE push_prompt_versions (
    form_id      uuid        NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    version      integer     NOT NULL,
    config       jsonb       NOT NULL,
    published_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    published_at timestamptz NOT NULL,
    PRIMARY KEY (form_id, version)
);

-- What the prompt did, per day: shown, Allow, the browser's permission
-- granted, Not now, closed, the bell clicked, and on iPhone and iPad the
-- Home Screen steps shown and the site added. Counted, never logged per
-- reader.
CREATE TABLE push_prompt_days (
    form_id      uuid    NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    workspace_id uuid    NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    day          date    NOT NULL,
    shown        integer NOT NULL DEFAULT 0,
    allowed      integer NOT NULL DEFAULT 0,
    granted      integer NOT NULL DEFAULT 0,
    not_now      integer NOT NULL DEFAULT 0,
    closed       integer NOT NULL DEFAULT 0,
    bell_clicked integer NOT NULL DEFAULT 0,
    blocked      integer NOT NULL DEFAULT 0,
    ios_steps    integer NOT NULL DEFAULT 0,
    ios_added    integer NOT NULL DEFAULT 0,
    PRIMARY KEY (form_id, day)
);

-- What a push campaign holds besides the campaign row.
CREATE TABLE push_campaigns (
    campaign_id    uuid PRIMARY KEY REFERENCES campaigns(id) ON DELETE CASCADE,
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title          text        NOT NULL DEFAULT '',
    message        text        NOT NULL DEFAULT '',
    link           text        NOT NULL DEFAULT '',
    image_media_id uuid        REFERENCES media_assets(id) ON DELETE SET NULL,
    -- Up to 2 buttons (Chrome and Edge): [{"label":"Read the post","url":"…"}]
    buttons        jsonb       NOT NULL DEFAULT '[]',
    ttl_minutes    integer     NOT NULL DEFAULT 240 CHECK (ttl_minutes BETWEEN 5 AND 40320),
    tag            text        NOT NULL DEFAULT '',
    replace_tag    boolean     NOT NULL DEFAULT true,
    -- The audience: sites, optionally narrowed to subscriptions linked to
    -- contacts on these lists or in these segments.
    site_ids       uuid[]      NOT NULL DEFAULT '{}',
    lists          uuid[]      NOT NULL DEFAULT '{}',
    segments       uuid[]      NOT NULL DEFAULT '{}',
    schedule_mode  text        NOT NULL DEFAULT 'now' CHECK (schedule_mode IN ('now','at')),
    send_at        timestamptz,
    -- Outside quiet hours: hold until the window opens in the subscriber's
    -- time zone, or skip.
    outside        text        NOT NULL DEFAULT 'hold' CHECK (outside IN ('hold','skip')),
    version        integer     NOT NULL DEFAULT 1
);

-- A push campaign's send.
CREATE TABLE push_sends (
    campaign_id   uuid PRIMARY KEY REFERENCES campaigns(id) ON DELETE CASCADE,
    workspace_id  uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    phase         text        NOT NULL CHECK (phase IN ('scheduled','sending','completed','cancelled')),
    recipients    integer     NOT NULL DEFAULT 0,
    -- Left out before the send: {"frequency_cap":126,"quiet_hours":0}
    left_out      jsonb       NOT NULL DEFAULT '{}',
    started_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    approved_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    scheduled_at  timestamptz NOT NULL,
    first_send_at timestamptz,
    last_send_at  timestamptz,
    completed_at  timestamptz
);

-- What only push has, per notification of the message log.
CREATE TABLE push_messages (
    message_id      uuid PRIMARY KEY REFERENCES channel_messages(id) ON DELETE CASCADE,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    subscription_id uuid,
    site_id         uuid        REFERENCES push_sites(id) ON DELETE SET NULL,
    browser         text        NOT NULL DEFAULT '',
    -- The push service's answer ("201 Created", "410 Gone · Unregistered").
    answer          text        NOT NULL DEFAULT '',
    shown_at        timestamptz,
    clicked_at      timestamptz,
    closed_at       timestamptz
);
CREATE INDEX push_messages_site_idx ON push_messages (site_id);
CREATE INDEX push_messages_subscription_idx ON push_messages (subscription_id);

-- Every notification gets its own click token for the redirect (/t/p/…),
-- so a click is counted even when the service worker can't report it.
CREATE TABLE push_clicks (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    message_id   uuid        NOT NULL REFERENCES channel_messages(id) ON DELETE CASCADE,
    -- 0: the notification itself; 1, 2: its buttons.
    action       integer     NOT NULL DEFAULT 0,
    target       text        NOT NULL,
    occurred_at  timestamptz NOT NULL
);
CREATE INDEX push_clicks_message_idx ON push_clicks (message_id);

-- Push and WhatsApp add their events to the message log: a notification
-- shown or closed, a WhatsApp message read.
ALTER TABLE channel_message_events DROP CONSTRAINT channel_message_events_kind_check;
ALTER TABLE channel_message_events ADD CONSTRAINT channel_message_events_kind_check
    CHECK (kind IN ('queued','sent','delivered','undelivered','failed','clicked','replied','opted_out','opted_in','help','shown','closed','read','gone'));

-- +goose Down
ALTER TABLE channel_message_events DROP CONSTRAINT channel_message_events_kind_check;
DELETE FROM channel_message_events WHERE kind IN ('shown','closed','read','gone');
ALTER TABLE channel_message_events ADD CONSTRAINT channel_message_events_kind_check
    CHECK (kind IN ('queued','sent','delivered','undelivered','failed','clicked','replied','opted_out','opted_in','help'));
DROP TABLE push_clicks;
DROP TABLE push_messages;
DROP TABLE push_sends;
DROP TABLE push_campaigns;
DROP TABLE push_prompt_days;
DROP TABLE push_prompt_versions;
DROP TABLE push_prompts;
DELETE FROM forms WHERE kind = 'push';
ALTER TABLE forms DROP CONSTRAINT forms_kind_check;
ALTER TABLE forms ADD CONSTRAINT forms_kind_check CHECK (kind IN ('embedded','popup','hosted'));
DROP TABLE push_test_subscriptions;
DROP TABLE push_subscriptions;
DROP TABLE push_sites;
DROP TABLE push_settings;
DROP TABLE push_keys;
