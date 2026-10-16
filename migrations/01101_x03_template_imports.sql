-- +goose Up
-- Template imports (add-on transactional §3.8, §5.9): one row per run of
-- the Import templates dialog. The provider key is sealed and never
-- returned; the provider's template list, the picked templates and the
-- conversion report are kept until the templates arrive as drafts.
CREATE TABLE transactional_template_imports (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    source          text        NOT NULL CHECK (source IN ('postmark','sendgrid','mailgun','notifuse','files')),
    -- The dialog's step: source, connect, pick, review, done (or cancelled).
    status          text        NOT NULL DEFAULT 'connect' CHECK (status IN ('connect','pick','review','done','cancelled')),
    credentials_enc bytea,
    -- {"account":"Acme Cloud production","kind":"server","region":"us","domain":"","baseUrl":"…"}
    connection      jsonb       NOT NULL DEFAULT '{}',
    -- The provider's templates: [{"ref","alias","name","category","suggested"}]
    available       jsonb       NOT NULL DEFAULT '[]',
    picked          text[]      NOT NULL DEFAULT '{}',
    -- Uploaded MJML and HTML files: [{"name","content"}]
    files           jsonb       NOT NULL DEFAULT '[]',
    -- The converted templates with their notes, in the order picked.
    report          jsonb       NOT NULL DEFAULT '[]',
    -- The last failure talking to the provider: {"code","message","status"}
    error           jsonb,
    -- The drafts made: [{"id","name","status"}]
    result          jsonb       NOT NULL DEFAULT '[]',
    created_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    finished_at     timestamptz
);
CREATE INDEX transactional_template_imports_ws_idx ON transactional_template_imports (workspace_id, created_at DESC);

-- +goose Down
DROP TABLE transactional_template_imports;
