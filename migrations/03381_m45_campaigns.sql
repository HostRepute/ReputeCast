-- +goose Up
-- Campaigns and everyday sending (review 2026-11 M45).

-- CA-1: sent campaigns are archived, never deleted (D4): their reports,
-- web posts, retries and approvals keep pointing at them. Drafts and
-- campaigns that sent nothing are deleted, restorable for 30 days.
-- CA-2: tags. (CA-6's cadence is part of the audience, so approvals cover it.)
ALTER TABLE campaigns
    ADD COLUMN archived_at timestamptz,
    ADD COLUMN archived_by uuid REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN deleted_by  uuid REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN tags        text[] NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE campaigns DROP COLUMN archived_at, DROP COLUMN archived_by, DROP COLUMN deleted_by, DROP COLUMN tags;
