-- +goose Up
-- Real renders from the workspace's own rendering service (review 2026-10,
-- CO-13): Litmus or Mailgun Inspect. The composer's own previews are drawn
-- in the browser and say so; this is the optional real thing.

-- One connection per workspace. The API key is sealed (secrets.Box) and
-- never returned; key_hint is its last four characters. clients is the
-- service's list of email clients as it last answered.
CREATE TABLE rendering_services (
    workspace_id uuid        PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    service      text        NOT NULL CHECK (service IN ('litmus','mailgun_inspect')),
    region       text        NOT NULL DEFAULT '' CHECK (region IN ('','us','eu')),
    credentials  bytea       NOT NULL,
    key_hint     text        NOT NULL DEFAULT '',
    status       text        NOT NULL DEFAULT 'connected' CHECK (status IN ('connected','failing')),
    last_error   text        NOT NULL DEFAULT '',
    clients      jsonb       NOT NULL DEFAULT '[]',
    connected_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    connected_at timestamptz NOT NULL DEFAULT now(),
    checked_at   timestamptz
);

-- Renders requested from it: a saved version of a campaign (one A/B
-- variant), the clients asked for, and each client's screenshot as the
-- service reports it. external_id is the service's own id for the test.
CREATE TABLE campaign_renders (
    id           uuid        PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    campaign_id  uuid        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    version      integer     NOT NULL,
    variant      text        NOT NULL DEFAULT 'A' CHECK (variant IN ('A','B','C')),
    service      text        NOT NULL CHECK (service IN ('litmus','mailgun_inspect')),
    external_id  text        NOT NULL,
    clients      text[]      NOT NULL,
    status       text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','done','failed')),
    results      jsonb       NOT NULL DEFAULT '[]',
    error        text        NOT NULL DEFAULT '',
    requested_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    requested_at timestamptz NOT NULL,
    polled_at    timestamptz,
    completed_at timestamptz
);
CREATE INDEX campaign_renders_campaign_idx ON campaign_renders (campaign_id, requested_at DESC);

-- +goose Down
DROP TABLE campaign_renders;
DROP TABLE rendering_services;
