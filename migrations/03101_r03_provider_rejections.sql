-- +goose Up
-- PR-7: messages a provider refused (a permanent per-message answer such
-- as SES's MessageRejected for an unverified sender), per provider and
-- minute. Every send path records them through manage.RecordSend, so the
-- health job watches the share refused instead of mail being dropped
-- unnoticed.
ALTER TABLE provider_call_windows ADD COLUMN rejected integer NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE provider_call_windows DROP COLUMN rejected;
