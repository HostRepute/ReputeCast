-- +goose Up
-- Workspace sending rules (SR-10): checks a workspace adds to Send
-- Readiness on every send in their scope, with Warn or Block, and the
-- audience defaults of Settings › Sending rules.
CREATE TABLE sending_rules (
    id                 uuid        PRIMARY KEY,
    workspace_id       uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name               text        NOT NULL CHECK (length(name) BETWEEN 1 AND 120),
    kind               text        NOT NULL CHECK (kind IN ('link_param', 'link_domain', 'subject_length', 'contains_text')),
    operator           text        NOT NULL CHECK (operator IN ('on_every', 'on_none', 'not_one_of', 'one_of', 'at_most', 'at_least',
                                                                'footer_contains', 'footer_lacks', 'body_contains', 'subject_contains')),
    value              text        NOT NULL DEFAULT '',
    severity           text        NOT NULL CHECK (severity IN ('warn', 'block')),
    applies_to         text        NOT NULL CHECK (applies_to IN ('broadcast', 'automation', 'both')),
    enabled            boolean     NOT NULL DEFAULT true,
    message            text        NOT NULL DEFAULT '',
    created_by         uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at         timestamptz NOT NULL,
    updated_at         timestamptz NOT NULL,
    -- The last send a run found breaking the rule: when, which campaign,
    -- how many findings (links), and whether a later run found it fixed.
    triggered_at       timestamptz,
    triggered_campaign uuid        REFERENCES campaigns(id) ON DELETE SET NULL,
    triggered_name     text        NOT NULL DEFAULT '',
    triggered_count    integer     NOT NULL DEFAULT 0,
    triggered_fixed    boolean     NOT NULL DEFAULT false
);
CREATE INDEX sending_rules_workspace ON sending_rules (workspace_id, created_at);

-- How a workspace's sends count and confirm their audience: the
-- unengaged (the sunset tier) are left out of every send unless the
-- workspace turns that off (SR-11); a sender types the recipient count
-- when the audience is more than typed_multiple times the usual one (SR-13).
CREATE TABLE workspace_send_defaults (
    workspace_id      uuid         PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    exclude_unengaged boolean      NOT NULL DEFAULT true,
    typed_multiple    numeric(4,1) NOT NULL DEFAULT 1.5 CHECK (typed_multiple >= 1.1 AND typed_multiple <= 20),
    updated_at        timestamptz  NOT NULL,
    updated_by        uuid         REFERENCES users(id) ON DELETE SET NULL
);

-- +goose Down
DROP TABLE workspace_send_defaults;
DROP TABLE sending_rules;
