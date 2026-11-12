-- +goose Up
-- M63 G-33: the holdout group. While a period is open, a random share of
-- the workspace's contacts gets no campaigns and/or automation emails, so
-- the nightly measurement can compare them with everyone else.
--
-- A contact is held out while a period is open and
-- rc_holdout_bucket(contact, seed) < basis_points (500 = 5%). Buckets are
-- 0–9999 from the contact's id and the period's seed, so raising the share
-- keeps everyone already held out (nested buckets); starting over picks a
-- new seed. The function is the one definition of "held out": the sending
-- engine, Send Readiness, automations, channel campaigns, the segment field
-- and the measurement all call it.
-- +goose StatementBegin
CREATE FUNCTION rc_holdout_bucket(contact uuid, seed bigint) RETURNS int
    LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
    AS $$ SELECT ((hashtextextended($1::text, $2) & 2147483647) % 10000)::int $$;
-- +goose StatementEnd

-- Each change of the settings ends the open period and opens the next one
-- (with the same seed; "Start over" draws a new one), so a period's
-- results always describe one share and one scope.
CREATE TABLE holdout_periods (
    id                   uuid PRIMARY KEY,
    workspace_id         uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    seed                 bigint      NOT NULL,
    basis_points         int         NOT NULL CHECK (basis_points BETWEEN 10 AND 2000),
    campaigns            boolean     NOT NULL DEFAULT true,
    automations          boolean     NOT NULL DEFAULT true,
    started_at           timestamptz NOT NULL,
    ended_at             timestamptz,
    started_by           uuid        REFERENCES users(id) ON DELETE SET NULL,
    -- Why the period started over ("Start over"), as the person wrote it.
    reason               text        NOT NULL DEFAULT '' CHECK (length(reason) <= 500),
    -- When someone last asked to recompute the results (at most once
    -- every 10 minutes); the results are "computing" until computed_at
    -- passes it.
    measure_requested_at timestamptz,
    CHECK (campaigns OR automations),
    CHECK (ended_at IS NULL OR ended_at >= started_at)
);
CREATE UNIQUE INDEX holdout_periods_open ON holdout_periods (workspace_id) WHERE ended_at IS NULL;
CREATE INDEX holdout_periods_ws ON holdout_periods (workspace_id, started_at DESC);

-- The measurement of a period over a span: the whole period, or its last
-- 30 or 90 days ("window" is a reserved word).
CREATE TABLE holdout_results (
    period_id   uuid        NOT NULL REFERENCES holdout_periods(id) ON DELETE CASCADE,
    span        text        NOT NULL CHECK (span IN ('period', '30d', '90d')),
    computed_at timestamptz NOT NULL,
    data        jsonb       NOT NULL,
    PRIMARY KEY (period_id, span)
);

-- Automation steps a held-out contact passed without a message.
ALTER TABLE automation_step_stats ADD COLUMN held_out bigint NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE automation_step_stats DROP COLUMN held_out;
DROP TABLE holdout_results;
DROP TABLE holdout_periods;
DROP FUNCTION rc_holdout_bucket(uuid, bigint);
