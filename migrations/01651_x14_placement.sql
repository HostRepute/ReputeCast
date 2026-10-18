-- +goose Up
-- Placement tests (design expansion, M24): seed mailboxes the workspace
-- owns, tests started by a person (one copy of an email to each seed through
-- a route or providers), and where each copy landed.

-- A seed mailbox. Its password or OAuth tokens are sealed (write-only).
CREATE TABLE placement_seeds (
    id               uuid PRIMARY KEY,
    workspace_id     uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email            text        NOT NULL CHECK (email = lower(email)),
    provider         text        NOT NULL CHECK (provider IN ('gmail', 'google_workspace', 'outlook', 'microsoft_365', 'yahoo', 'aol', 'icloud', 'gmx', 'webde')),
    auth             text        NOT NULL CHECK (auth IN ('password', 'oauth')),
    imap_host        text        NOT NULL DEFAULT '',
    imap_port        integer     NOT NULL DEFAULT 993,
    username         text        NOT NULL DEFAULT '',
    credentials_enc  bytea       NOT NULL,
    -- connected, expired (an OAuth grant that ended), failed (a password
    -- that was rejected), unchecked; with since when and what happened.
    status           text        NOT NULL DEFAULT 'unchecked' CHECK (status IN ('connected', 'expired', 'failed', 'unchecked')),
    status_at        timestamptz,
    status_detail    text        NOT NULL DEFAULT '',
    -- The folders found when it was checked: [{"role","name","label"}].
    folders          jsonb       NOT NULL DEFAULT '[]',
    checked_at       timestamptz,
    last_used_at     timestamptz,
    last_test_id     uuid,
    last_test_label  text        NOT NULL DEFAULT '',
    added_by         uuid        REFERENCES users(id) ON DELETE SET NULL,
    added_at         timestamptz NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, email)
);

-- When copies are deleted from the seeds: 24 hours after the test
-- (default), when it finishes, or never.
CREATE TABLE placement_settings (
    workspace_id  uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    delete_after  text        NOT NULL DEFAULT '24h' CHECK (delete_after IN ('24h', 'finished', 'never')),
    updated_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at    timestamptz NOT NULL DEFAULT now()
);

-- OAuth consent in progress: the state names the person, the workspace and
-- the seed being added or reconnected. Single use, ten minutes.
CREATE TABLE placement_oauth_states (
    state_hash    bytea PRIMARY KEY,
    workspace_id  uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id       uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind          text        NOT NULL CHECK (kind IN ('google', 'microsoft')),
    provider      text        NOT NULL,
    seed_id       uuid,
    verifier_enc  bytea       NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    expires_at    timestamptz NOT NULL
);

-- A test. Only people start tests; there are no schedules.
CREATE TABLE placement_tests (
    id                 uuid PRIMARY KEY,
    workspace_id       uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name               text        NOT NULL,
    content_kind       text        NOT NULL CHECK (content_kind IN ('campaign', 'template', 'digest', 'transactional')),
    content_id         uuid,
    content_name       text        NOT NULL,
    -- How the content stood: draft, scheduled, sent, template or a
    -- transactional template's key and published version; content_at is
    -- when the campaign is scheduled or was sent.
    content_note       text        NOT NULL DEFAULT '',
    content_key        text        NOT NULL DEFAULT '',
    content_version    integer     NOT NULL DEFAULT 0,
    content_at         timestamptz,
    variant            text        NOT NULL DEFAULT '',
    language           text        NOT NULL DEFAULT '',
    personalization    text        NOT NULL DEFAULT '',
    through            text        NOT NULL CHECK (through IN ('route', 'provider', 'compare')),
    stream             text        NOT NULL,
    provider_ids       uuid[]      NOT NULL DEFAULT '{}',
    sender_id          uuid,
    from_name          text        NOT NULL DEFAULT '',
    from_email         text        NOT NULL,
    origin             text        NOT NULL DEFAULT 'placement' CHECK (origin IN ('placement', 'review', 'composer', 'warmup', 'template')),
    status             text        NOT NULL DEFAULT 'sending' CHECK (status IN ('sending', 'waiting', 'finished', 'stopped')),
    -- What was sent: the rendered subject, bodies and the tracked links
    -- (clicks on a copy redirect to these; they are never counted).
    subject            text        NOT NULL DEFAULT '',
    html               text        NOT NULL DEFAULT '',
    text_body          text        NOT NULL DEFAULT '',
    links              jsonb       NOT NULL DEFAULT '[]',
    track_opens        boolean     NOT NULL DEFAULT true,
    track_clicks       boolean     NOT NULL DEFAULT true,
    -- The providers as they were: {"<provider id>": {"name","short","adapter",
    -- "detail","ip","status","warmDay","warmDays","cost"}}.
    providers          jsonb       NOT NULL DEFAULT '{}',
    started_by         uuid        REFERENCES users(id) ON DELETE SET NULL,
    started_at         timestamptz NOT NULL DEFAULT now(),
    last_sent_at       timestamptz,
    -- A copy missing 15 minutes after the last hand-off is Not arrived; one
    -- that turns up within 60 minutes updates the result, marked late.
    wait_until         timestamptz,
    late_until         timestamptz,
    finished_at        timestamptz,
    stopped_by         uuid        REFERENCES users(id) ON DELETE SET NULL,
    stopped_at         timestamptz,
    delete_after       text        NOT NULL DEFAULT '24h' CHECK (delete_after IN ('24h', 'finished', 'never')),
    delete_at          timestamptz,
    copies_deleted_at  timestamptz,
    copies_deleted_by  uuid        REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX placement_tests_ws_idx ON placement_tests (workspace_id, started_at DESC);
CREATE INDEX placement_tests_open_idx ON placement_tests (status) WHERE status IN ('sending', 'waiting');

-- One copy to one seed through one provider. Its id is the message id of
-- the send (messages.kind = 'placement').
CREATE TABLE placement_copies (
    id                   uuid PRIMARY KEY,
    test_id              uuid        NOT NULL REFERENCES placement_tests(id) ON DELETE CASCADE,
    workspace_id         uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    seed_id              uuid        REFERENCES placement_seeds(id) ON DELETE SET NULL,
    seed_email           text        NOT NULL,
    seed_provider        text        NOT NULL,
    position             integer     NOT NULL,
    provider_id          uuid        REFERENCES providers(id) ON DELETE SET NULL,
    provider_name        text        NOT NULL DEFAULT '',
    token                text        NOT NULL UNIQUE,
    round                integer     NOT NULL DEFAULT 1,
    send_at              timestamptz NOT NULL,
    status               text        NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'sent', 'failed', 'found', 'not_arrived', 'not_checked')),
    folder               text        NOT NULL DEFAULT '' CHECK (folder IN ('', 'inbox', 'tab', 'spam')),
    folder_name          text        NOT NULL DEFAULT '',
    tab                  text        NOT NULL DEFAULT '',
    late                 boolean     NOT NULL DEFAULT false,
    handed_at            timestamptz,
    received_at          timestamptz,
    found_at             timestamptz,
    provider_message_id  text        NOT NULL DEFAULT '',
    error                text        NOT NULL DEFAULT '',
    -- SPF, DKIM and DMARC from Authentication-Results; the hand-over hop;
    -- the headers as the mailbox reported them; where to delete it from.
    auth                 jsonb       NOT NULL DEFAULT '{}',
    hop                  jsonb       NOT NULL DEFAULT '{}',
    headers              text        NOT NULL DEFAULT '',
    ref                  jsonb       NOT NULL DEFAULT '{}',
    -- Why a copy wasn't looked for, or stopped being looked for:
    -- signin_failed, signin_expired, seed_removed, stopped, unsent.
    not_checked          text        NOT NULL DEFAULT '',
    deleted_at           timestamptz
);
CREATE INDEX placement_copies_test_idx ON placement_copies (test_id, position, round);
CREATE INDEX placement_copies_seed_idx ON placement_copies (seed_id) WHERE seed_id IS NOT NULL;

-- +goose Down
DROP TABLE placement_copies;
DROP TABLE placement_tests;
DROP TABLE placement_oauth_states;
DROP TABLE placement_settings;
DROP TABLE placement_seeds;
