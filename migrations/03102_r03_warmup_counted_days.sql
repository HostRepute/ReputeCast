-- +goose Up
-- PR-8: a warm-up plan moves by counted days, not by the calendar. A day
-- counts once at least 80% of its cap went out and the guards (which judge
-- a day only after 500 messages) held; a day that doesn't count repeats,
-- a pause freezes the plan, and after a guard trips the cap holds until two
-- healthy days in a row.
--
-- start_date now moves forward by every day that didn't count (so the
-- plan's day today is still today minus start_date plus one everywhere it
-- is read); started_on keeps the day the plan began.
ALTER TABLE warmup_plans ADD COLUMN started_on date;
UPDATE warmup_plans SET started_on = start_date;
ALTER TABLE warmup_plans ALTER COLUMN started_on SET NOT NULL;
-- The last day (in the workspace's time zone) the daily job judged.
ALTER TABLE warmup_plans ADD COLUMN evaluated_through date;
-- While a slowdown holds: the cap it holds and the healthy days so far.
ALTER TABLE warmup_plans ADD COLUMN held_cap bigint;
ALTER TABLE warmup_plans ADD COLUMN healthy_streak integer NOT NULL DEFAULT 0;
-- What's over today's cap goes through a warm provider instead of waiting.
ALTER TABLE warmup_plans ADD COLUMN overflow boolean NOT NULL DEFAULT true;
-- Guards judge a day only after this many messages.
ALTER TABLE warmup_plans ADD COLUMN min_sample integer NOT NULL DEFAULT 500 CHECK (min_sample BETWEEN 1 AND 100000);
-- Messages over the cap that went through a warm provider on overflowed_on
-- (the workspace's day): "Over the cap so far today: 0".
ALTER TABLE warmup_plans ADD COLUMN overflowed bigint NOT NULL DEFAULT 0;
ALTER TABLE warmup_plans ADD COLUMN overflowed_on date;

-- +goose Down
ALTER TABLE warmup_plans DROP COLUMN overflowed_on;
ALTER TABLE warmup_plans DROP COLUMN overflowed;
ALTER TABLE warmup_plans DROP COLUMN min_sample;
ALTER TABLE warmup_plans DROP COLUMN overflow;
ALTER TABLE warmup_plans DROP COLUMN healthy_streak;
ALTER TABLE warmup_plans DROP COLUMN held_cap;
ALTER TABLE warmup_plans DROP COLUMN evaluated_through;
ALTER TABLE warmup_plans DROP COLUMN started_on;
