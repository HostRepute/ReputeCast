-- +goose Up
-- Address verification services (design expansion, M24): the workspace's
-- own ZeroBounce, NeverBounce, Kickbox, Bouncer or Emailable account, its
-- uses (imports, signup forms, lists, segments, tiers, selections, one
-- contact) and the latest result on each contact.

-- A connected service. The API key is sealed (write-only); the price per
-- check is the workspace's plan price, in millionths of a dollar.
CREATE TABLE verification_services (
    id                uuid PRIMARY KEY,
    workspace_id      uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    service           text        NOT NULL,
    region            text        NOT NULL DEFAULT '',
    credentials_enc   bytea       NOT NULL,
    price_micros      bigint      NOT NULL DEFAULT 0 CHECK (price_micros >= 0),
    default_bulk      boolean     NOT NULL DEFAULT false,
    default_forms     boolean     NOT NULL DEFAULT false,
    -- Balance, read at each use and every hour.
    balance           bigint,
    balance_at        timestamptz,
    last_used_at      timestamptz,
    last_used_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    last_used_what    text        NOT NULL DEFAULT '',
    -- The last error and when the streak began ("503 from 15:05 to 15:10").
    last_error        text        NOT NULL DEFAULT '',
    last_error_at     timestamptz,
    last_error_since  timestamptz,
    last_success_at   timestamptz,
    last_success_what text        NOT NULL DEFAULT '',
    tested_at         timestamptz,
    test_ok           boolean,
    -- When members were told the credits run low (once until it recovers).
    low_notified_at   timestamptz,
    connected_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    connected_at      timestamptz NOT NULL DEFAULT now(),
    updated_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, service)
);
-- One default for bulk checks and imports, one for signup forms.
CREATE UNIQUE INDEX verification_services_bulk_idx ON verification_services (workspace_id) WHERE default_bulk;
CREATE UNIQUE INDEX verification_services_forms_idx ON verification_services (workspace_id) WHERE default_forms;

-- One bulk use: a list, a segment, a tier, a selection, an import's new
-- addresses or one contact. Counts fill in as the job goes, so a stopped or
-- cancelled job keeps its results so far.
CREATE TABLE verification_jobs (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    service_id      uuid        REFERENCES verification_services(id) ON DELETE SET NULL,
    service         text        NOT NULL,
    kind            text        NOT NULL CHECK (kind IN ('import', 'list', 'segment', 'tier', 'selection', 'contact')),
    target          jsonb       NOT NULL DEFAULT '{}',
    label           text        NOT NULL,
    origin          text        NOT NULL DEFAULT '',
    campaign_id     uuid        REFERENCES campaigns(id) ON DELETE SET NULL,
    import_id       uuid        REFERENCES imports(id) ON DELETE SET NULL,
    status          text        NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'paused', 'stopped', 'cancelled', 'finished')),
    stop_reason     text        NOT NULL DEFAULT '' CHECK (stop_reason IN ('', 'no_credits', 'service_down', 'key_rejected', 'cancelled')),
    skip_recent     boolean     NOT NULL DEFAULT true,
    total           integer     NOT NULL DEFAULT 0,
    skipped         integer     NOT NULL DEFAULT 0,
    checked         integer     NOT NULL DEFAULT 0,
    valid           integer     NOT NULL DEFAULT 0,
    invalid         integer     NOT NULL DEFAULT 0,
    risky           integer     NOT NULL DEFAULT 0,
    unknown         integer     NOT NULL DEFAULT 0,
    suppressed      integer     NOT NULL DEFAULT 0,
    -- Counts per reason: invalid_address, spam_trap, catch_all, role…
    reasons         jsonb       NOT NULL DEFAULT '{}',
    price_micros    bigint      NOT NULL DEFAULT 0,
    balance_before  bigint,
    balance_after   bigint,
    started_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    started_at      timestamptz,
    finished_at     timestamptz,
    stopped_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    -- A service that stopped answering: when it began, the next try, and
    -- the hour after which the job stops with what it has.
    error_since     timestamptz,
    next_try_at     timestamptz,
    retry_until     timestamptz,
    last_error      text        NOT NULL DEFAULT '',
    -- A worker on the job holds it until then, so two never check the same addresses.
    claimed_until   timestamptz
);
CREATE INDEX verification_jobs_ws_idx ON verification_jobs (workspace_id, created_at DESC);

-- The addresses of a job, so it can resume and keep partial results. An
-- import's new addresses have no contact yet.
CREATE TABLE verification_job_items (
    job_id      uuid        NOT NULL REFERENCES verification_jobs(id) ON DELETE CASCADE,
    email       text        NOT NULL,
    contact_id  uuid,
    position    integer     NOT NULL,
    done        boolean     NOT NULL DEFAULT false,
    result      text        NOT NULL DEFAULT '',
    reason      text        NOT NULL DEFAULT '',
    code        text        NOT NULL DEFAULT '',
    checked_at  timestamptz,
    PRIMARY KEY (job_id, email)
);
CREATE INDEX verification_job_items_pending_idx ON verification_job_items (job_id, position) WHERE NOT done;

-- The latest result of each contact: "Valid · ZeroBounce · 14 Oct".
CREATE TABLE contact_address_checks (
    contact_id    uuid PRIMARY KEY REFERENCES contacts(id) ON DELETE CASCADE,
    workspace_id  uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    result        text        NOT NULL CHECK (result IN ('valid', 'invalid', 'risky', 'unknown')),
    reason        text        NOT NULL DEFAULT '',
    code          text        NOT NULL DEFAULT '',
    service       text        NOT NULL,
    job_id        uuid        REFERENCES verification_jobs(id) ON DELETE SET NULL,
    checked_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX contact_address_checks_ws_idx ON contact_address_checks (workspace_id, result, checked_at);

-- Real-time checks per signup form ("Check each address with Kickbox as
-- people sign up"). form_key is the form's id or 'subscribe_page'.
CREATE TABLE verification_form_checks (
    workspace_id  uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    form_key      text        NOT NULL,
    enabled       boolean     NOT NULL DEFAULT false,
    updated_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, form_key)
);

-- Real-time checks are counted by day and form, not audited one by one
-- (the brief's §8 conflict 1, the roadmap's default).
CREATE TABLE verification_realtime_daily (
    workspace_id  uuid    NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    service       text    NOT NULL,
    form_key      text    NOT NULL,
    day           date    NOT NULL,
    checks        integer NOT NULL DEFAULT 0,
    valid         integer NOT NULL DEFAULT 0,
    invalid       integer NOT NULL DEFAULT 0,
    risky         integer NOT NULL DEFAULT 0,
    unknown       integer NOT NULL DEFAULT 0,
    no_answer     integer NOT NULL DEFAULT 0,
    PRIMARY KEY (workspace_id, service, form_key, day)
);

-- Signups the service didn't answer for within 3 seconds: they went
-- through and are checked once the service answers again.
CREATE TABLE verification_realtime_pending (
    id            uuid PRIMARY KEY,
    workspace_id  uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email         text        NOT NULL,
    form_key      text        NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    tries         integer     NOT NULL DEFAULT 0,
    UNIQUE (workspace_id, email)
);

-- +goose Down
DROP TABLE verification_realtime_pending;
DROP TABLE verification_realtime_daily;
DROP TABLE verification_form_checks;
DROP TABLE contact_address_checks;
DROP TABLE verification_job_items;
DROP TABLE verification_jobs;
DROP TABLE verification_services;
