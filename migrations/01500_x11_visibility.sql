-- +goose Up
-- Conditional content (design expansion, M21): Visibility rules on blocks.
-- The rules live in the email's MJML (rc-visible on the block), so they are
-- versioned with the content; the tables here hold what the send decides
-- and what the composer counted.

-- The conditional blocks a recipient doesn't get: the ids of the blocks
-- whose rule leaves them out, comma-separated, decided once when the send's
-- audience is built so a contact gets one consistent version. Empty for
-- every email without conditional blocks.
ALTER TABLE messages ADD COLUMN hidden_blocks text NOT NULL DEFAULT '';

-- The versions a campaign's audience gets, counted with the live audience
-- (internal/visibility). Kept until the rules or the audience change, so the
-- composer, Send Readiness and placement tests read the same count.
CREATE TABLE visibility_counts (
    campaign_id   uuid PRIMARY KEY REFERENCES campaigns(id) ON DELETE CASCADE,
    workspace_id  uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    -- What was counted: the rules (with what they refer to) and the audience.
    rules_hash    text        NOT NULL,
    audience_hash text        NOT NULL,
    total         bigint      NOT NULL DEFAULT 0,
    -- [{"hidden": "a1b2c3,d4e5f6", "recipients": 9812}], largest first.
    versions      jsonb       NOT NULL DEFAULT '[]',
    -- Recipients who see each conditional block: {"a1b2c3": 1402}.
    shown         jsonb       NOT NULL DEFAULT '{}',
    -- Recipients per condition group, for the footer line ("At risk 3,450").
    factors       jsonb       NOT NULL DEFAULT '{}',
    counted_at    timestamptz NOT NULL DEFAULT now(),
    duration_ms   integer     NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE visibility_counts;
ALTER TABLE messages DROP COLUMN hidden_blocks;
