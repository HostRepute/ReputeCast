-- +goose Up
-- Demo data: the outgoing webhook endpoints and the audit stream the demo
-- seeds point at fictional addresses on real domains. They are never called,
-- until someone points them at a URL of their own.
ALTER TABLE webhooks ADD COLUMN demo boolean NOT NULL DEFAULT false;
ALTER TABLE workspaces ADD COLUMN audit_stream_demo boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE workspaces DROP COLUMN audit_stream_demo;
ALTER TABLE webhooks DROP COLUMN demo;
