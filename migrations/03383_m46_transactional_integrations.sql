-- +goose Up
-- Transactional, approvals, polls and integrations (review 2026-11 M46).

-- TX-1: a scheduled message can be cancelled before its send time; the
-- log shows it as Cancelled, with who did it.
ALTER TABLE transactional_messages
    ADD COLUMN cancelled_at timestamptz,
    ADD COLUMN cancelled_by text NOT NULL DEFAULT '';

-- TX-4: the per-key send rate, shared by every web node: a token bucket
-- per API key (or person), refilled by elapsed time on each take.
CREATE TABLE transactional_rate_buckets (
    bucket_key  uuid PRIMARY KEY,
    tokens      double precision NOT NULL,
    refilled_at timestamptz      NOT NULL
);

-- IN-1: a rotated secret keeps signing beside the new one until
-- previous_secret_until (24 hours). IN-2: alerted_at is when the workspace
-- was last told the endpoint is failing (once per failing spell).
ALTER TABLE webhooks
    ADD COLUMN previous_secret_enc   bytea,
    ADD COLUMN previous_secret_until timestamptz,
    ADD COLUMN alerted_at            timestamptz;

-- AP-1: when an open request was last checked against its campaign, so
-- the inbox re-checks only the stale ones.
ALTER TABLE approval_requests ADD COLUMN revalidated_at timestamptz;

-- +goose Down
ALTER TABLE approval_requests DROP COLUMN revalidated_at;
ALTER TABLE webhooks DROP COLUMN previous_secret_enc, DROP COLUMN previous_secret_until, DROP COLUMN alerted_at;
DROP TABLE transactional_rate_buckets;
ALTER TABLE transactional_messages DROP COLUMN cancelled_at, DROP COLUMN cancelled_by;
