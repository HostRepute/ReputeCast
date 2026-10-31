-- +goose Up
-- A/B tests end to end (review AN-1). The recipients who wait for the
-- winner get the winning variant's content (variant names it, as for the
-- sample), and ab_rollout marks them: their results are counted apart from
-- the test sample's (campaign_counters' variant '' is the rollout), so the
-- report compares the samples only and says what the rest got.
ALTER TABLE messages ADD COLUMN ab_rollout boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE messages DROP COLUMN ab_rollout;
