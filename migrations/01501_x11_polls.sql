-- +goose Up
-- Polls (design expansion, M21). A poll is a block of an email: its
-- question, answers and settings are in the email's MJML (rc-poll on the
-- section) and copied here when a test or a send needs its links. Answers
-- are recorded by the answer page, never by the click on the link.

CREATE TABLE polls (
    id                  uuid PRIMARY KEY,
    workspace_id        uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    -- Signed answer links name the poll by this number (linktoken's Link).
    number              bigint      NOT NULL GENERATED ALWAYS AS IDENTITY UNIQUE,
    campaign_id         uuid        REFERENCES campaigns(id) ON DELETE CASCADE,
    automation_email_id uuid        REFERENCES automation_emails(id) ON DELETE CASCADE,
    -- The block's id in the email (rc-poll's "id").
    block_id            text        NOT NULL,
    kind                text        NOT NULL CHECK (kind IN ('choice', 'scale', 'emoji')),
    question            text        NOT NULL DEFAULT '',
    -- [{"key": 1, "label": "Postgres internals", "tag": "wants:postgres"}]
    answers             jsonb       NOT NULL DEFAULT '[]',
    -- Save to, close, after answering, non-contacts, results, wording.
    settings            jsonb       NOT NULL DEFAULT '{}',
    -- When the poll stops recording: a date, N days after sending (set when
    -- the send starts), or null (closed by hand).
    closes_at           timestamptz,
    closed_at           timestamptz,
    closed_by           uuid        REFERENCES users(id) ON DELETE SET NULL,
    reopened_at         timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CHECK (num_nonnulls(campaign_id, automation_email_id) = 1)
);
CREATE UNIQUE INDEX polls_campaign_block_idx ON polls (campaign_id, block_id) WHERE campaign_id IS NOT NULL;
CREATE UNIQUE INDEX polls_email_block_idx ON polls (automation_email_id, block_id) WHERE automation_email_id IS NOT NULL;
CREATE INDEX polls_workspace_idx ON polls (workspace_id);

-- One answer per contact and poll (the latest counts; the history is in
-- poll_answer_events), answers from people who aren't contacts one row
-- each, and test answers per test address. Answers and comments are
-- contact data: erasing the contact unlinks them and clears the comment.
CREATE TABLE poll_answers (
    id           uuid PRIMARY KEY,
    poll_id      uuid        NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    contact_id   uuid        REFERENCES contacts(id) ON DELETE CASCADE,
    message_id   uuid,
    kind         text        NOT NULL DEFAULT 'contact' CHECK (kind IN ('contact', 'non_contact', 'test')),
    test_email   text        NOT NULL DEFAULT '',
    answer       integer     NOT NULL,
    first_answer integer     NOT NULL,
    changes      integer     NOT NULL DEFAULT 0,
    comment      text        NOT NULL DEFAULT '',
    commented_at timestamptz,
    client       text        NOT NULL DEFAULT '',
    answered_at  timestamptz NOT NULL,
    updated_at   timestamptz NOT NULL
);
CREATE UNIQUE INDEX poll_answers_contact_idx ON poll_answers (poll_id, contact_id) WHERE kind = 'contact';
CREATE UNIQUE INDEX poll_answers_test_idx ON poll_answers (poll_id, test_email) WHERE kind = 'test';
CREATE INDEX poll_answers_poll_idx ON poll_answers (poll_id, answered_at);
CREATE INDEX poll_answers_contact_lookup_idx ON poll_answers (contact_id) WHERE contact_id IS NOT NULL;

-- Every answer, change and comment as it happened.
CREATE TABLE poll_answer_events (
    id        bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    answer_id uuid        NOT NULL REFERENCES poll_answers(id) ON DELETE CASCADE,
    poll_id   uuid        NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    kind      text        NOT NULL CHECK (kind IN ('answered', 'changed', 'commented')),
    answer    integer,
    previous  integer,
    at        timestamptz NOT NULL
);
CREATE INDEX poll_answer_events_answer_idx ON poll_answer_events (answer_id, at);

-- Visits to an answer page that the tracking classifier attributed to a
-- machine (security gateways, prefetching proxies): counted, never answers.
CREATE TABLE poll_automated_visits (
    poll_id uuid        NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    reason  text        NOT NULL CHECK (reason IN ('gateway', 'prefetch')),
    n       bigint      NOT NULL DEFAULT 0,
    last_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (poll_id, reason)
);

-- The addresses a test email with polls went to: a test answer link names
-- its row, so the answer is recorded as that address's test answer.
CREATE TABLE poll_test_recipients (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    campaign_id  uuid        REFERENCES campaigns(id) ON DELETE CASCADE,
    email        text        NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);

-- "Answered a poll or survey" starts automations: its events join the
-- trigger events. The kinds are checked by the engine, which knows the
-- triggers of every phase; the table only keeps the names well formed, so
-- later phases add triggers without redefining this list.
ALTER TABLE automation_events DROP CONSTRAINT IF EXISTS automation_events_kind_check;
ALTER TABLE automation_events ADD CONSTRAINT automation_events_kind_check CHECK (kind ~ '^[a-z][a-z_]{1,39}$');

-- +goose Down
ALTER TABLE automation_events DROP CONSTRAINT IF EXISTS automation_events_kind_check;
DELETE FROM automation_events WHERE kind NOT IN ('joined_list', 'form_submitted', 'tag_added', 'api_event', 'date_field');
ALTER TABLE automation_events ADD CONSTRAINT automation_events_kind_check CHECK (kind IN ('joined_list', 'form_submitted', 'tag_added', 'api_event', 'date_field'));
DROP TABLE poll_test_recipients;
DROP TABLE poll_automated_visits;
DROP TABLE poll_answer_events;
DROP TABLE poll_answers;
DROP TABLE polls;
