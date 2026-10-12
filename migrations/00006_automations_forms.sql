-- +goose Up
-- Automations (React Flow graphs, versions, runs, per-step stats) and signup forms.

CREATE TABLE automations (
    id             uuid PRIMARY KEY,
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name           text        NOT NULL,
    description    text        NOT NULL DEFAULT '',
    status         text        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','live','paused')),
    template_key   text        NOT NULL DEFAULT '',
    live_version   integer,
    draft_version  integer     NOT NULL DEFAULT 1,
    -- Trigger summary for list views: {"kind":"joined_list","list_id":"…"}
    trigger        jsonb       NOT NULL DEFAULT '{}',
    created_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    deleted_at     timestamptz
);
CREATE INDEX automations_workspace_idx ON automations (workspace_id) WHERE deleted_at IS NULL;

ALTER TABLE approval_requests ADD CONSTRAINT approval_requests_automation_fk
    FOREIGN KEY (automation_id) REFERENCES automations(id) ON DELETE CASCADE;
ALTER TABLE messages ADD CONSTRAINT messages_automation_fk
    FOREIGN KEY (automation_id) REFERENCES automations(id) ON DELETE CASCADE;

CREATE TABLE automation_versions (
    automation_id uuid        NOT NULL REFERENCES automations(id) ON DELETE CASCADE,
    version       integer     NOT NULL,
    -- React Flow graph: {"nodes":[{"id":"n1","type":"trigger","position":{…},"data":{…}}],"edges":[…]}
    graph         jsonb       NOT NULL,
    created_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    published_by  uuid        REFERENCES users(id) ON DELETE SET NULL,
    published_at  timestamptz,
    retired_at    timestamptz,
    PRIMARY KEY (automation_id, version)
);

-- A contact's journey through one version of a flow.
CREATE TABLE automation_runs (
    id            uuid PRIMARY KEY,
    automation_id uuid        NOT NULL REFERENCES automations(id) ON DELETE CASCADE,
    version       integer     NOT NULL,
    contact_id    uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    node_id       text        NOT NULL,
    status        text        NOT NULL DEFAULT 'active' CHECK (status IN ('active','waiting','completed','exited','goal','failed')),
    state         jsonb       NOT NULL DEFAULT '{}',
    entered_at    timestamptz NOT NULL DEFAULT now(),
    next_at       timestamptz,
    finished_at   timestamptz
);
CREATE INDEX automation_runs_due_idx ON automation_runs (next_at) WHERE status IN ('active','waiting');
CREATE INDEX automation_runs_automation_idx ON automation_runs (automation_id, version, status);
CREATE UNIQUE INDEX automation_runs_active_key ON automation_runs (automation_id, contact_id) WHERE status IN ('active','waiting');

CREATE TABLE automation_step_stats (
    automation_id uuid    NOT NULL REFERENCES automations(id) ON DELETE CASCADE,
    version       integer NOT NULL,
    node_id       text    NOT NULL,
    entered       bigint  NOT NULL DEFAULT 0,
    waiting       bigint  NOT NULL DEFAULT 0,
    completed     bigint  NOT NULL DEFAULT 0,
    sent          bigint  NOT NULL DEFAULT 0,
    delivered     bigint  NOT NULL DEFAULT 0,
    opened        bigint  NOT NULL DEFAULT 0,
    clicked       bigint  NOT NULL DEFAULT 0,
    branch_yes    bigint  NOT NULL DEFAULT 0,
    branch_no     bigint  NOT NULL DEFAULT 0,
    goal_reached  bigint  NOT NULL DEFAULT 0,
    PRIMARY KEY (automation_id, version, node_id)
);

CREATE TABLE forms (
    id             uuid PRIMARY KEY,
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name           text        NOT NULL,
    kind           text        NOT NULL DEFAULT 'embedded' CHECK (kind IN ('embedded','popup','hosted')),
    slug           text        NOT NULL CHECK (slug ~ '^[a-z0-9][a-z0-9-]{0,62}$'),
    list_ids       uuid[]      NOT NULL DEFAULT '{}',
    double_opt_in  boolean     NOT NULL DEFAULT true,
    -- [{"key":"email","label":"Email","required":true},{"key":"first_name","label":"First name"}]
    fields         jsonb       NOT NULL DEFAULT '[]',
    -- Copy and look: heading, description, button label, success message, popup trigger…
    content        jsonb       NOT NULL DEFAULT '{}',
    -- {"honeypot":true,"min_seconds":3,"rate_limit_per_ip":5,"block_disposable":true}
    spam_protection jsonb      NOT NULL DEFAULT '{}',
    status         text        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','live','paused')),
    views          bigint      NOT NULL DEFAULT 0,
    submissions    bigint      NOT NULL DEFAULT 0,
    confirmed      bigint      NOT NULL DEFAULT 0,
    blocked        bigint      NOT NULL DEFAULT 0,
    created_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX forms_slug_key ON forms (workspace_id, slug);

-- Double opt-in and preference-center tokens (hashed).
CREATE TABLE subscriber_tokens (
    id          uuid PRIMARY KEY,
    workspace_id uuid       NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    contact_id  uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    kind        text        NOT NULL CHECK (kind IN ('confirm','preferences')),
    token_hash  bytea       NOT NULL,
    form_id     uuid        REFERENCES forms(id) ON DELETE SET NULL,
    list_ids    uuid[]      NOT NULL DEFAULT '{}',
    created_at  timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL,
    used_at     timestamptz
);
CREATE UNIQUE INDEX subscriber_tokens_key ON subscriber_tokens (token_hash);

-- +goose Down
DROP TABLE subscriber_tokens;
DROP TABLE forms;
DROP TABLE automation_step_stats;
DROP TABLE automation_runs;
DROP TABLE automation_versions;
ALTER TABLE messages DROP CONSTRAINT messages_automation_fk;
ALTER TABLE approval_requests DROP CONSTRAINT approval_requests_automation_fk;
DROP TABLE automations;
