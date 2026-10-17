-- +goose Up
-- Surveys (design expansion, M21): a hosted page with several questions,
-- built in Forms & pages › Forms and linked from an email.

CREATE TABLE surveys (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    -- Signed survey links name the survey by this number (linktoken's Link).
    number       bigint      NOT NULL GENERATED ALWAYS AS IDENTITY UNIQUE,
    slug         text        NOT NULL,
    name         text        NOT NULL,
    description  text        NOT NULL DEFAULT '',
    status       text        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'open', 'closed')),
    -- [{"key": "q1", "type": "single", "text": "…", "required": true, "answers": […], "save": {…}}]
    questions    jsonb       NOT NULL DEFAULT '[]',
    -- One response per contact, people who aren't contacts, thank-you text,
    -- the closing date's time zone.
    settings     jsonb       NOT NULL DEFAULT '{}',
    closes_at    timestamptz,
    published_at timestamptz,
    published_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    closed_at    timestamptz,
    closed_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    deleted_at   timestamptz,
    UNIQUE (workspace_id, slug)
);

-- A response: the answers so far ({"q1": "weekly", "q2": ["flags"]}),
-- started when question 1 came from an email's poll or the page was sent,
-- completed when the person pressed Submit. Erasing the contact unlinks it
-- and clears its text answers.
CREATE TABLE survey_responses (
    id           uuid PRIMARY KEY,
    survey_id    uuid        NOT NULL REFERENCES surveys(id) ON DELETE CASCADE,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    contact_id   uuid        REFERENCES contacts(id) ON DELETE CASCADE,
    message_id   uuid,
    kind         text        NOT NULL DEFAULT 'contact' CHECK (kind IN ('contact', 'non_contact', 'test')),
    test_email   text        NOT NULL DEFAULT '',
    answers      jsonb       NOT NULL DEFAULT '{}',
    -- page: answered on the survey page; poll: question 1 came from a poll.
    source       text        NOT NULL DEFAULT 'page' CHECK (source IN ('page', 'poll')),
    started_at   timestamptz NOT NULL,
    completed_at timestamptz,
    updated_at   timestamptz NOT NULL,
    changes      integer     NOT NULL DEFAULT 0
);
CREATE INDEX survey_responses_survey_idx ON survey_responses (survey_id, completed_at);
CREATE INDEX survey_responses_contact_idx ON survey_responses (survey_id, contact_id) WHERE contact_id IS NOT NULL;

-- +goose Down
DROP TABLE survey_responses;
DROP TABLE surveys;
