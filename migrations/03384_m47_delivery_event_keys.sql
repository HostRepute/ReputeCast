-- +goose Up
-- Monthly partitions of delivery_events (review 2026-11 OP-8, plan in
-- docs/deploy/partitioning.md). A unique index on a partitioned table must
-- include the partition key, so the provider's dedupe key (one event per
-- provider and key, across months) moves to a table of its own, written in
-- the same statement as the event. Keys older than 180 days are purged: no
-- provider replays events that old.
CREATE TABLE delivery_event_keys (
    provider_id uuid        NOT NULL,
    dedupe_key  text        NOT NULL,
    received_at timestamptz NOT NULL,
    PRIMARY KEY (provider_id, dedupe_key)
);
CREATE INDEX delivery_event_keys_received_idx ON delivery_event_keys (received_at);

-- Which monthly partitions have had their open and click detail cleared
-- for the workspaces keeping it a given number of days (DP-6 by
-- partition): a partition past the cutoff is cleared in one pass, once.
CREATE TABLE delivery_event_partitions (
    partition       text        NOT NULL,
    detail_days     integer     NOT NULL,
    cleared_at      timestamptz NOT NULL,
    PRIMARY KEY (partition, detail_days)
);

-- +goose Down
DROP TABLE delivery_event_partitions;
DROP TABLE delivery_event_keys;
