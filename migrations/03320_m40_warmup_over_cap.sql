-- +goose Up
-- Transactional mail never waits for a warm-up (review R3.L6b): when a
-- warming provider is at today's cap, or its plan is paused, and no warm
-- provider of the message's route can take it, a password reset still goes
-- out through the warming provider. It counts toward the day's cap like any
-- other message; over_cap is how many went out over it on over_cap_on (the
-- workspace's day), shown on the warm-up page and the provider's card.
-- Adding columns with a constant default doesn't rewrite the table.
ALTER TABLE warmup_plans
    ADD COLUMN over_cap bigint NOT NULL DEFAULT 0,
    ADD COLUMN over_cap_on date;

-- +goose Down
ALTER TABLE warmup_plans
    DROP COLUMN over_cap_on,
    DROP COLUMN over_cap;
