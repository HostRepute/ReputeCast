-- +goose Up
-- Recommendations and partner links (design expansion, phase 5, M25; brief
-- reputecast-design-addon-grow.md §3.8–3.9).
--
-- A recommendation is a newsletter shown on the opt-in confirmation page
-- and in the Recommendations block: a link to any subscribe page, or a
-- partner on another ReputeCast instance, connected with a partner link.
-- Partner links go one direction at a time ("Acme recommends Ops Weekly"
-- is one, "Ops Weekly recommends Acme" another). Each is set up with a
-- single-use partner code and signs every message with a shared secret
-- agreed over HTTPS, sealed on both sides. The protocol is documented in
-- docs/grow/partners.md.
--
-- Subscribe requests are automatic and high volume: they're kept in the
-- partner request logs (90 days), not the audit log. The recommending side
-- keeps only counts, never which of its readers confirmed.

-- The workspace's partner profile (what partners see and show) and where
-- recommendations show.
CREATE TABLE recommendation_settings (
    workspace_id    uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    -- Up to 3 recommendations under "While you're here" on the
    -- confirmation page; on by default once a recommendation exists.
    confirm_page    boolean     NOT NULL DEFAULT true,
    profile_name        text    NOT NULL DEFAULT '',
    profile_description text    NOT NULL DEFAULT '' CHECK (char_length(profile_description) <= 120),
    profile_logo_id     uuid    REFERENCES media_assets(id) ON DELETE SET NULL,
    -- The list partners' readers join (each partner code can offer another).
    profile_list_id     uuid    REFERENCES lists(id) ON DELETE SET NULL,
    updated_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at      timestamptz NOT NULL DEFAULT now()
);

-- Partner codes this workspace created so another instance can recommend
-- it: single use, valid 7 days, shown once (only a hash is kept).
CREATE TABLE partner_codes (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    code_hash    bytea       NOT NULL,
    -- A label for whom it's for ("The Deliverability Desk").
    label        text        NOT NULL,
    list_id      uuid        NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    daily_limit  integer     NOT NULL DEFAULT 50 CHECK (daily_limit BETWEEN 1 AND 100000),
    created_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL,
    used_at      timestamptz,
    used_link_id uuid,
    revoked_at   timestamptz,
    revoked_by   uuid        REFERENCES users(id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX partner_codes_hash_key ON partner_codes (code_hash);
CREATE INDEX partner_codes_ws_idx ON partner_codes (workspace_id, created_at DESC);

-- A partner link. direction 'out': this workspace recommends the remote
-- newsletter (it sends subscribe requests); 'in': the remote newsletter
-- recommends this workspace (it receives them, with a daily limit).
CREATE TABLE partner_links (
    id                 uuid PRIMARY KEY,
    workspace_id       uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    direction          text        NOT NULL CHECK (direction IN ('out', 'in')),
    -- The other instance's public base URL and its id for this link.
    remote_url         text        NOT NULL,
    remote_link_id     text        NOT NULL,
    protocol_version   integer     NOT NULL DEFAULT 1,
    -- The other side's newsletter: its profile for 'out' (name, logo,
    -- description, list), its newsletter name for 'in'.
    remote_name        text        NOT NULL DEFAULT '',
    remote_description text        NOT NULL DEFAULT '',
    remote_logo_url    text        NOT NULL DEFAULT '',
    remote_list_name   text        NOT NULL DEFAULT '',
    -- 'out': the partner's daily limit for our readers; 'in': the limit we
    -- gave them.
    daily_limit        integer     NOT NULL DEFAULT 50,
    -- 'in': the list partner readers join, the code's label, the code.
    list_id            uuid        REFERENCES lists(id) ON DELETE SET NULL,
    label              text        NOT NULL DEFAULT '',
    code_id            uuid        REFERENCES partner_codes(id) ON DELETE SET NULL,
    -- 'out': the hash of the partner code pasted to connect ("Marcus Chen
    -- used this code on 22 Sep").
    code_hash          bytea,
    -- The last rotation applied (a repeated rotate message changes nothing).
    last_rotation      text        NOT NULL DEFAULT '',
    -- The shared signing secret (sealed), and the previous one for 24
    -- hours after a rotation.
    secret             bytea       NOT NULL,
    prev_secret        bytea,
    prev_secret_until  timestamptz,
    secret_rotated_at  timestamptz,
    status             text        NOT NULL DEFAULT 'connected' CHECK (status IN ('connected', 'paused', 'invalid', 'ended')),
    -- Who paused: this side ('local') or the partner ('remote').
    paused_side        text        NOT NULL DEFAULT '' CHECK (paused_side IN ('', 'local', 'remote')),
    paused_at          timestamptz,
    paused_by          uuid        REFERENCES users(id) ON DELETE SET NULL,
    -- 'out': the partner rejected our signature or ended the link.
    invalid_since      timestamptz,
    invalid_reason     text        NOT NULL DEFAULT '',
    -- 'out': the partner's daily limit was reached; hidden until then
    -- (its next 00:00 UTC).
    limit_until        timestamptz,
    -- 'out': requests have failed since then (unreachable), until one goes
    -- through.
    failing_since      timestamptz,
    ended_at           timestamptz,
    ended_by           uuid        REFERENCES users(id) ON DELETE SET NULL,
    ended_side         text        NOT NULL DEFAULT '' CHECK (ended_side IN ('', 'local', 'remote')),
    connected_at       timestamptz NOT NULL DEFAULT now(),
    connected_by       uuid        REFERENCES users(id) ON DELETE SET NULL,
    notified_at        timestamptz,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX partner_links_ws_idx ON partner_links (workspace_id, direction);

ALTER TABLE partner_codes ADD CONSTRAINT partner_codes_link_fk
    FOREIGN KEY (used_link_id) REFERENCES partner_links(id) ON DELETE SET NULL;

-- The newsletters this workspace recommends, in the order readers see them.
CREATE TABLE recommendations (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    kind            text        NOT NULL CHECK (kind IN ('link', 'partner')),
    name            text        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 120),
    description     text        NOT NULL DEFAULT '' CHECK (char_length(description) <= 120),
    logo_id         uuid        REFERENCES media_assets(id) ON DELETE SET NULL,
    -- link: the newsletter's subscribe page (https).
    url             text        NOT NULL DEFAULT '',
    -- partner: the partner link it sends subscribe requests through.
    partner_link_id uuid        REFERENCES partner_links(id) ON DELETE SET NULL,
    show_confirm    boolean     NOT NULL DEFAULT true,
    show_block      boolean     NOT NULL DEFAULT true,
    status          text        NOT NULL DEFAULT 'showing' CHECK (status IN ('showing', 'paused')),
    paused_at       timestamptz,
    paused_by       uuid        REFERENCES users(id) ON DELETE SET NULL,
    position        integer     NOT NULL DEFAULT 0,
    created_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    removed_at      timestamptz
);
CREATE INDEX recommendations_ws_idx ON recommendations (workspace_id, position) WHERE removed_at IS NULL;

-- Numbers per recommendation and day: shown (confirmation pages and
-- delivered emails that showed it; opens aren't used), clicks for links,
-- subscribe requests and confirmations for partners.
CREATE TABLE recommendation_daily (
    recommendation_id uuid    NOT NULL REFERENCES recommendations(id) ON DELETE CASCADE,
    day               date    NOT NULL,
    shown_confirm     integer NOT NULL DEFAULT 0,
    shown_email       integer NOT NULL DEFAULT 0,
    clicks            integer NOT NULL DEFAULT 0,
    requests          integer NOT NULL DEFAULT 0,
    confirmed         integer NOT NULL DEFAULT 0,
    PRIMARY KEY (recommendation_id, day)
);

-- Subscribe requests this workspace sends to partners (direction 'out').
-- The reader's address is sealed and cleared once the partner accepted or
-- the request failed; only the id stays, for the partner's confirmation.
CREATE TABLE partner_requests_out (
    id                uuid PRIMARY KEY,
    workspace_id      uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    link_id           uuid        NOT NULL REFERENCES partner_links(id) ON DELETE CASCADE,
    recommendation_id uuid        REFERENCES recommendations(id) ON DELETE SET NULL,
    email             bytea,
    -- Where the reader asked: the confirmation page or the recommendations page.
    origin            text        NOT NULL DEFAULT 'confirm' CHECK (origin IN ('confirm', 'page')),
    status            text        NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'accepted', 'limit', 'failed', 'rejected')),
    attempts          integer     NOT NULL DEFAULT 0,
    next_attempt_at   timestamptz,
    last_error        text        NOT NULL DEFAULT '',
    created_at        timestamptz NOT NULL DEFAULT now(),
    sent_at           timestamptz,
    confirmed_at      timestamptz
);
CREATE INDEX partner_requests_out_link_idx ON partner_requests_out (link_id, created_at DESC);
CREATE INDEX partner_requests_out_queue_idx ON partner_requests_out (next_attempt_at) WHERE status = 'queued';

-- Subscribe requests partners sent this workspace (direction 'in'): the
-- request id once per link (replay protection), the outcome, and the
-- signed confirmation sent back when the reader confirms.
CREATE TABLE partner_requests_in (
    id                 uuid PRIMARY KEY,
    workspace_id       uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    link_id            uuid        NOT NULL REFERENCES partner_links(id) ON DELETE CASCADE,
    request_id         text        NOT NULL,
    status             text        NOT NULL CHECK (status IN ('accepted', 'limit', 'paused')),
    contact_id         uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    -- Whether a confirmation email goes out (not for an existing
    -- subscriber), and its sending: queued, sent, or failed after 3 days
    -- without a System provider.
    asked              boolean     NOT NULL DEFAULT false,
    optin_status       text        NOT NULL DEFAULT '' CHECK (optin_status IN ('', 'queued', 'sent', 'failed')),
    optin_attempts     integer     NOT NULL DEFAULT 0,
    optin_next_at      timestamptz,
    optin_sent_at      timestamptz,
    -- The recommending newsletter's name, for the email's "You asked for
    -- this on …'s page."
    source_name        text        NOT NULL DEFAULT '',
    received_at        timestamptz NOT NULL DEFAULT now(),
    confirmed_at       timestamptz,
    notice_status      text        NOT NULL DEFAULT '' CHECK (notice_status IN ('', 'queued', 'sent', 'failed')),
    notice_attempts    integer     NOT NULL DEFAULT 0,
    notice_next_at     timestamptz,
    notice_sent_at     timestamptz
);
CREATE UNIQUE INDEX partner_requests_in_key ON partner_requests_in (link_id, request_id);
CREATE INDEX partner_requests_in_day_idx ON partner_requests_in (link_id, received_at);
CREATE INDEX partner_requests_in_contact_idx ON partner_requests_in (contact_id) WHERE contact_id IS NOT NULL;
CREATE INDEX partner_requests_in_notice_idx ON partner_requests_in (notice_next_at) WHERE notice_status = 'queued';
CREATE INDEX partner_requests_in_optin_idx ON partner_requests_in (optin_next_at) WHERE optin_status = 'queued';

-- Recommendations-page visits are personal (a signed link): no table; the
-- page answers from the signed token.

-- +goose Down
DROP TABLE partner_requests_in;
DROP TABLE partner_requests_out;
DROP TABLE recommendation_daily;
DROP TABLE recommendations;
ALTER TABLE partner_codes DROP CONSTRAINT partner_codes_link_fk;
DROP TABLE partner_links;
DROP TABLE partner_codes;
DROP TABLE recommendation_settings;
