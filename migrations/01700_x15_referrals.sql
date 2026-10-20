-- +goose Up
-- Referral programs (design expansion, phase 5, M25; brief
-- reputecast-design-addon-grow.md §3.1–3.7).
--
-- One program per list. Every confirmed subscriber of the list gets a
-- personal link; a signup through a link is a referral, which counts once
-- its subscriber confirmed and stayed through the hold period. Milestones
-- give rewards (a download, a code from an uploaded list, something sent by
-- hand, a Stripe coupon once payments exist, or a webhook call), once per
-- milestone per referrer, in order.
--
-- Automatic events (each signup, confirmation and automatic count) are kept
-- in referral_events, the program's referral log, not in the audit log
-- (ROADMAP › Design expansion, "Logs and the audit log"). Everything a
-- person changes is audited.

CREATE TABLE referral_programs (
    id                  uuid PRIMARY KEY,
    workspace_id        uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    -- The list readers join through a link. Fixed once the program has
    -- been on (first_on_at is set).
    list_id             uuid        NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    status              text        NOT NULL DEFAULT 'off' CHECK (status IN ('off', 'on')),
    -- The set-up guide's step (1 to 5) until the program is first turned
    -- on, so a reload resumes the same step; 0 afterwards.
    setup_step          integer     NOT NULL DEFAULT 1 CHECK (setup_step BETWEEN 0 AND 5),
    setup_saved_at      timestamptz,
    setup_saved_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    first_on_at         timestamptz,
    first_on_by         uuid        REFERENCES users(id) ON DELETE SET NULL,
    -- The last status change ("off since 15 Oct, 09:12 (Sofia Rossi)").
    status_at           timestamptz,
    status_by           uuid        REFERENCES users(id) ON DELETE SET NULL,
    -- Counting rules (§3.2). Self-referral, "already on the list" and
    -- "never confirmed" can't be turned off, so they have no column.
    hold_days           integer     NOT NULL DEFAULT 7 CHECK (hold_days IN (0, 3, 7, 14, 30)),
    rule_same_ip        boolean     NOT NULL DEFAULT true,
    rule_disposable     boolean     NOT NULL DEFAULT true,
    rule_left           boolean     NOT NULL DEFAULT true,
    -- Suspicion rules: a match holds the referral for review.
    burst_on            boolean     NOT NULL DEFAULT true,
    burst_threshold     integer     NOT NULL DEFAULT 5 CHECK (burst_threshold BETWEEN 1 AND 1000),
    shared_ip_on        boolean     NOT NULL DEFAULT true,
    shared_ip_threshold integer     NOT NULL DEFAULT 3 CHECK (shared_ip_threshold BETWEEN 2 AND 1000),
    numbered_on         boolean     NOT NULL DEFAULT true,
    -- Links and pages. The invite headline holds {first_name}, replaced by
    -- the referrer's first name when they allow it, otherwise by the
    -- page's "A reader".
    invite_headline     text        NOT NULL DEFAULT '',
    invite_text         text        NOT NULL DEFAULT '',
    hub_intro           text        NOT NULL DEFAULT '',
    -- {link} becomes the reader's own link.
    share_message       text        NOT NULL DEFAULT '',
    share_buttons       text[]      NOT NULL DEFAULT '{copy,email,x,linkedin,whatsapp}',
    -- The public leaderboard (off by default, names opt-in).
    leaderboard_on      boolean     NOT NULL DEFAULT false,
    leaderboard_at      timestamptz,
    leaderboard_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    leaderboard_periods text[]      NOT NULL DEFAULT '{month,all}',
    leaderboard_rows    integer     NOT NULL DEFAULT 10 CHECK (leaderboard_rows IN (5, 10, 25)),
    -- Milestone emails: Automations stream, from the program's sender (the
    -- Automations default sender unless changed) with an optional reply-to.
    sender_id           uuid        REFERENCES senders(id) ON DELETE SET NULL,
    reply_to            text        NOT NULL DEFAULT '',
    -- "Referrals held for review" goes out when the queue goes from empty
    -- to waiting, then at most once a day while it isn't empty.
    review_notified_at  timestamptz,
    created_by          uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    -- Deleted while off: links stop for good; counted referrals and sent
    -- rewards stay in contacts' timelines.
    deleted_at          timestamptz,
    deleted_by          uuid        REFERENCES users(id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX referral_programs_list_key ON referral_programs (list_id) WHERE deleted_at IS NULL;
CREATE INDEX referral_programs_ws_idx ON referral_programs (workspace_id) WHERE deleted_at IS NULL;

-- Files for download rewards (PDF, ZIP or images up to 100 MB). Bytes live
-- in Postgres like the media library's, in 1 MiB chunks so a download
-- streams instead of loading the whole file.
CREATE TABLE referral_files (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    program_id   uuid        NOT NULL REFERENCES referral_programs(id) ON DELETE CASCADE,
    file_name    text        NOT NULL,
    content_type text        NOT NULL,
    size_bytes   bigint      NOT NULL CHECK (size_bytes > 0 AND size_bytes <= 104857600),
    sha256       bytea       NOT NULL,
    downloads    bigint      NOT NULL DEFAULT 0,
    uploaded_by  uuid        REFERENCES users(id) ON DELETE SET NULL,
    uploaded_at  timestamptz NOT NULL DEFAULT now(),
    removed_at   timestamptz
);
CREATE INDEX referral_files_program_idx ON referral_files (program_id) WHERE removed_at IS NULL;

CREATE TABLE referral_file_chunks (
    file_id uuid    NOT NULL REFERENCES referral_files(id) ON DELETE CASCADE,
    seq     integer NOT NULL,
    data    bytea   NOT NULL,
    PRIMARY KEY (file_id, seq)
);

CREATE TABLE referral_milestones (
    id                 uuid PRIMARY KEY,
    workspace_id       uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    program_id         uuid        NOT NULL REFERENCES referral_programs(id) ON DELETE CASCADE,
    -- Counted referrals that reach the milestone.
    count              integer     NOT NULL CHECK (count BETWEEN 1 AND 100000),
    reward_name        text        NOT NULL CHECK (char_length(reward_name) BETWEEN 1 AND 120),
    reward_description text        NOT NULL DEFAULT '' CHECK (char_length(reward_description) <= 120),
    image_id           uuid        REFERENCES media_assets(id) ON DELETE SET NULL,
    reward_type        text        NOT NULL CHECK (reward_type IN ('download', 'codes', 'manual', 'coupon', 'webhook')),
    -- download: an uploaded file, or a URL.
    file_id            uuid        REFERENCES referral_files(id) ON DELETE SET NULL,
    download_url       text        NOT NULL DEFAULT '',
    -- codes: "Low on codes" below this many codes left (or under two
    -- weeks at the current pace).
    codes_alert_below  integer     NOT NULL DEFAULT 20 CHECK (codes_alert_below BETWEEN 0 AND 100000),
    codes_alerted      text        NOT NULL DEFAULT '' CHECK (codes_alerted IN ('', 'low', 'out')),
    -- manual: ask the referrer for a shipping address on the hub.
    ask_address        boolean     NOT NULL DEFAULT false,
    -- coupon (phase 6): an offer of the kind "Referral reward" in the
    -- payment provider, and how long its single-use codes stay valid.
    coupon_offer_id    text        NOT NULL DEFAULT '',
    coupon_offer_name  text        NOT NULL DEFAULT '',
    coupon_valid_days  integer     NOT NULL DEFAULT 60 CHECK (coupon_valid_days IN (30, 60, 90)),
    -- webhook: referral.milestone_reached is posted here, signed with the
    -- secret (sealed, write-only) like outgoing webhooks.
    webhook_url        text        NOT NULL DEFAULT '',
    webhook_secret     bytea,
    -- The last call's answer, for the editor ("answered 502 to the last 8
    -- calls"); the answer's code or URL lives on the reward, sealed.
    webhook_status     integer,
    webhook_called_at  timestamptz,
    webhook_failures   integer     NOT NULL DEFAULT 0,
    webhook_alerted_at timestamptz,
    -- The milestone email, or none because an automation on "Reached a
    -- milestone" sends one. It is an automation email outside any flow
    -- (automation_emails.automation_id is NULL), so it goes out through the
    -- Automations stream's outbox; Save in the composer's milestone mode
    -- adds a version to it. The draft below autosaves.
    email_on           boolean     NOT NULL DEFAULT true,
    email_id           uuid        REFERENCES automation_emails(id) ON DELETE SET NULL,
    email_subject      text        NOT NULL DEFAULT '',
    email_preheader    text        NOT NULL DEFAULT '',
    email_mjml         text        NOT NULL DEFAULT '',
    email_html         text        NOT NULL DEFAULT '',
    email_revision     integer     NOT NULL DEFAULT 0,
    email_saved_at     timestamptz,
    email_saved_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_by         uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    -- Deleted milestones keep their rewards with the referrers who got them.
    deleted_at         timestamptz
);
CREATE UNIQUE INDEX referral_milestones_count_key ON referral_milestones (program_id, count) WHERE deleted_at IS NULL;

-- A referrer's personal link: 6 characters without look-alikes (no 0, O,
-- 1, I or L). Erasing the referrer removes it.
CREATE TABLE referral_links (
    id               uuid PRIMARY KEY,
    workspace_id     uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    program_id       uuid        NOT NULL REFERENCES referral_programs(id) ON DELETE CASCADE,
    contact_id       uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    code             text        NOT NULL CHECK (code ~ '^[2-9A-HJKMNP-Z]{6}$'),
    status           text        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused')),
    paused_at        timestamptz,
    paused_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    pause_note       text        NOT NULL DEFAULT '',
    -- The reader's own choices on the hub, both off by default.
    show_name        boolean     NOT NULL DEFAULT false,
    show_leaderboard boolean     NOT NULL DEFAULT false,
    choices_at       timestamptz,
    visits           bigint      NOT NULL DEFAULT 0,
    signups          bigint      NOT NULL DEFAULT 0,
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX referral_links_code_key ON referral_links (workspace_id, code);
CREATE UNIQUE INDEX referral_links_contact_key ON referral_links (program_id, contact_id);

-- Addresses a referrer used in the last 30 days to open their hub or a
-- page of their link (their own signups are read from consent_events),
-- for the "same IP address as the referrer" rule. Older rows are pruned.
CREATE TABLE referral_referrer_ips (
    link_id uuid        NOT NULL REFERENCES referral_links(id) ON DELETE CASCADE,
    ip      inet        NOT NULL,
    seen_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (link_id, ip)
);

-- A signup through a referral link. Referrer and referred contact may be
-- erased later: the row stays, so counts and earned rewards never change.
CREATE TABLE referrals (
    id                uuid PRIMARY KEY,
    workspace_id      uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    program_id        uuid        NOT NULL REFERENCES referral_programs(id) ON DELETE CASCADE,
    link_id           uuid        REFERENCES referral_links(id) ON DELETE SET NULL,
    referrer_id       uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    contact_id        uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    -- "Referral 3" on the referrer's hub: the order of signups on the link.
    seq               integer     NOT NULL,
    status            text        NOT NULL CHECK (status IN ('awaiting', 'hold', 'review', 'counted', 'not_counted')),
    reason            text        NOT NULL DEFAULT ''
        CHECK (reason IN ('', 'self', 'same_ip', 'disposable', 'left', 'already', 'never_confirmed', 'reviewer')),
    -- For reason 'left': unsubscribed, hard_bounce or complaint.
    left_kind         text        NOT NULL DEFAULT '',
    -- The suspicion rules that held it: burst, shared_ip, numbered.
    signals           text[]      NOT NULL DEFAULT '{}',
    -- Where the link was shared (?via=): copy, email, x, linkedin, whatsapp, other.
    via               text        NOT NULL DEFAULT 'copy',
    -- The signup's address, kept 90 days for review, then reduced to its
    -- network (ip_network stays).
    ip                inet,
    ip_network        text        NOT NULL DEFAULT '',
    -- Optional details about the network (a lookup the instance may add;
    -- empty by default): its name, autonomous system and kind (hosting,
    -- office, residential, mobile).
    network_name      text        NOT NULL DEFAULT '',
    network_asn       text        NOT NULL DEFAULT '',
    network_kind      text        NOT NULL DEFAULT '',
    signed_up_at      timestamptz NOT NULL,
    reminded_at       timestamptz,
    confirmed_at      timestamptz,
    hold_until        timestamptz,
    counted_at        timestamptz,
    not_counted_at    timestamptz,
    held_at           timestamptz,
    decided_at        timestamptz,
    decided_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    decision          text        NOT NULL DEFAULT '' CHECK (decision IN ('', 'count', 'dont', 'dont_pause')),
    decision_note     text        NOT NULL DEFAULT '',
    -- The decision's group (several referrals decided together) and how
    -- long Undo stays possible (10 seconds, before any reward goes out).
    decision_id       uuid,
    undo_until        timestamptz,
    -- What the referral was before the decision, for Undo.
    before_status     text        NOT NULL DEFAULT '',
    ip_reduced_at     timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);
-- One referrer per signup: a contact is referred to a program at most once.
CREATE UNIQUE INDEX referrals_contact_key ON referrals (program_id, contact_id) WHERE contact_id IS NOT NULL;
CREATE INDEX referrals_link_idx ON referrals (link_id, seq);
CREATE INDEX referrals_program_status_idx ON referrals (program_id, status, signed_up_at);
CREATE INDEX referrals_referrer_idx ON referrals (referrer_id) WHERE referrer_id IS NOT NULL;
CREATE INDEX referrals_hold_idx ON referrals (hold_until) WHERE status = 'hold';
CREATE INDEX referrals_awaiting_idx ON referrals (signed_up_at) WHERE status = 'awaiting';
CREATE INDEX referrals_confirmed_idx ON referrals (program_id, confirmed_at) WHERE confirmed_at IS NOT NULL;
CREATE INDEX referrals_ip_idx ON referrals (program_id, ip) WHERE ip IS NOT NULL;

-- The referral log: each signup, reminder, confirmation, hold, count and
-- exclusion, review decision, milestone and reward, with what it was about.
CREATE TABLE referral_events (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    program_id   uuid        NOT NULL REFERENCES referral_programs(id) ON DELETE CASCADE,
    referral_id  uuid        REFERENCES referrals(id) ON DELETE CASCADE,
    link_id      uuid        REFERENCES referral_links(id) ON DELETE SET NULL,
    -- The contact the event belongs to on their timeline (the referrer for
    -- counts, milestones and rewards; the referred contact for joining).
    contact_id   uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    kind         text        NOT NULL,
    detail       jsonb       NOT NULL DEFAULT '{}',
    occurred_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX referral_events_program_idx ON referral_events (program_id, occurred_at DESC);
CREATE INDEX referral_events_contact_idx ON referral_events (contact_id, occurred_at DESC) WHERE contact_id IS NOT NULL;

-- Visits and signups per link, day and channel. A visit counts once per
-- network and browser a day, without cookies; bots are left out.
CREATE TABLE referral_visits_daily (
    link_id    uuid    NOT NULL REFERENCES referral_links(id) ON DELETE CASCADE,
    program_id uuid    NOT NULL REFERENCES referral_programs(id) ON DELETE CASCADE,
    day        date    NOT NULL,
    via        text    NOT NULL,
    visits     integer NOT NULL DEFAULT 0,
    signups    integer NOT NULL DEFAULT 0,
    PRIMARY KEY (link_id, day, via)
);
CREATE INDEX referral_visits_program_idx ON referral_visits_daily (program_id, day);

-- The day's visit keys (a keyed hash of the link, the network and the
-- browser), discarded after 24 hours.
CREATE TABLE referral_visit_keys (
    key     bytea       PRIMARY KEY,
    seen_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX referral_visit_keys_seen_idx ON referral_visit_keys (seen_at);

-- Rewards: one per milestone per referrer. Name and type are kept as
-- given, so a deleted or changed milestone leaves earned rewards alone.
-- Erasing the referrer removes their rewards.
CREATE TABLE referral_rewards (
    id               uuid PRIMARY KEY,
    workspace_id     uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    program_id       uuid        NOT NULL REFERENCES referral_programs(id) ON DELETE CASCADE,
    milestone_id     uuid        REFERENCES referral_milestones(id) ON DELETE SET NULL,
    contact_id       uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    link_id          uuid        REFERENCES referral_links(id) ON DELETE SET NULL,
    milestone_count  integer     NOT NULL,
    reward_name      text        NOT NULL,
    reward_type      text        NOT NULL CHECK (reward_type IN ('download', 'codes', 'manual', 'coupon', 'webhook')),
    -- pending: being delivered (a webhook call or a coupon in flight);
    -- sent: available to the referrer; waiting_codes: the list ran out;
    -- to_fulfill: sent by hand, not yet; fulfilled; failed (webhook gave
    -- up after 6 hours, Retry); expired (a coupon past its date).
    status           text        NOT NULL CHECK (status IN ('pending', 'sent', 'waiting_codes', 'to_fulfill', 'fulfilled', 'failed', 'expired')),
    reached_at       timestamptz NOT NULL,
    sent_at          timestamptz,
    -- Used: downloaded, claimed or fulfilled.
    used_at          timestamptz,
    downloads        integer     NOT NULL DEFAULT 0,
    -- codes: the code it took.
    code_id          uuid,
    -- coupon: the claim link to the upgrade page and when its code expires.
    claim_url        text        NOT NULL DEFAULT '',
    expires_at       timestamptz,
    -- webhook: the answer's code or URL (sealed), the attempts.
    answer           bytea,
    attempts         integer     NOT NULL DEFAULT 0,
    next_attempt_at  timestamptz,
    last_error       text        NOT NULL DEFAULT '',
    failed_at        timestamptz,
    -- manual: the referrer's shipping address (sealed), deleted 30 days
    -- after the reward is marked fulfilled; the note the referrer sees.
    address          bytea,
    address_at       timestamptz,
    fulfilled_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    fulfill_note     text        NOT NULL DEFAULT '',
    address_purge_at timestamptz,
    -- The milestone email, once sent (the message is in messages).
    email_sent_at    timestamptz,
    email_message_id uuid,
    resent_at        timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX referral_rewards_key ON referral_rewards (milestone_id, contact_id) WHERE milestone_id IS NOT NULL;
CREATE INDEX referral_rewards_program_idx ON referral_rewards (program_id, status);
CREATE INDEX referral_rewards_contact_idx ON referral_rewards (contact_id);
CREATE INDEX referral_rewards_retry_idx ON referral_rewards (next_attempt_at) WHERE status = 'pending';

-- Codes uploaded for a code reward, sealed; shown only to members with
-- grow:review and never written to the audit log (only their count).
CREATE TABLE referral_reward_codes (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    milestone_id uuid        NOT NULL REFERENCES referral_milestones(id) ON DELETE CASCADE,
    code         bytea       NOT NULL,
    -- A keyed hash for duplicates within the milestone.
    code_hash    bytea       NOT NULL,
    batch_id     uuid        NOT NULL,
    uploaded_by  uuid        REFERENCES users(id) ON DELETE SET NULL,
    uploaded_at  timestamptz NOT NULL DEFAULT now(),
    reward_id    uuid        REFERENCES referral_rewards(id) ON DELETE SET NULL,
    assigned_at  timestamptz,
    removed_at   timestamptz
);
CREATE UNIQUE INDEX referral_reward_codes_key ON referral_reward_codes (milestone_id, code_hash);
CREATE INDEX referral_reward_codes_free_idx ON referral_reward_codes (milestone_id, uploaded_at) WHERE reward_id IS NULL AND removed_at IS NULL;

ALTER TABLE referral_rewards ADD CONSTRAINT referral_rewards_code_fk
    FOREIGN KEY (code_id) REFERENCES referral_reward_codes(id) ON DELETE SET NULL;

-- The leaderboard, refreshed hourly: the top referrers this month and all
-- time by counted referrals (ties share a rank).
CREATE TABLE referral_leaderboard (
    program_id  uuid        NOT NULL REFERENCES referral_programs(id) ON DELETE CASCADE,
    period      text        NOT NULL CHECK (period IN ('month', 'all')),
    link_id     uuid        NOT NULL REFERENCES referral_links(id) ON DELETE CASCADE,
    rank        integer     NOT NULL,
    counted     integer     NOT NULL,
    computed_at timestamptz NOT NULL,
    PRIMARY KEY (program_id, period, link_id)
);
CREATE INDEX referral_leaderboard_rank_idx ON referral_leaderboard (program_id, period, rank);

-- +goose Down
DROP TABLE referral_leaderboard;
ALTER TABLE referral_rewards DROP CONSTRAINT referral_rewards_code_fk;
DROP TABLE referral_reward_codes;
DROP TABLE referral_rewards;
DROP TABLE referral_visit_keys;
DROP TABLE referral_visits_daily;
DROP TABLE referral_events;
DROP TABLE referrals;
DROP TABLE referral_referrer_ips;
DROP TABLE referral_links;
DROP TABLE referral_milestones;
DROP TABLE referral_file_chunks;
DROP TABLE referral_files;
DROP TABLE referral_programs;
