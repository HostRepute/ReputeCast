# Partitioning delivery_events

`delivery_events` holds every send, delivery, bounce, open and click. It is
the largest table on most instances and grows without end. Splitting it by
month keeps each month's indexes small, lets the retention jobs work a month
at a time, and leaves room to archive or drop old months later (review
2026-11 OP-8).

## The plan

| Choice | Why |
|---|---|
| Partition by `received_at`, monthly | It is when ReputeCast stored the event: always about now, so new events land in the current month, never in an old one. `occurred_at` can be months back (a provider's backfill). |
| Convert without copying | The table as it is becomes `delivery_events_legacy`, the partition of everything before the first month. Only months from then on get partitions of their own. A million-contact instance converts in seconds. |
| Months made ahead | The hourly maintenance job keeps the current month and the next two. A `delivery_events_default` partition catches anything outside them, so an insert never fails. |
| Primary key `(id, received_at)` | A unique index on a partitioned table must contain the partition key. Lookups by `id` alone still use it. |
| Dedupe keys in their own table | One event per provider and dedupe key must hold across months, which a per-partition unique index can't. `delivery_event_keys` takes the key in the same statement as the event; keys are kept 180 days (no provider replays older events). |
| Detail retention by partition | DP-6 clears opens' and clicks' IP addresses and user agents after a workspace's setting (12 months by default). Once a month's partition is past a setting's cutoff (plus a day), it is cleared in one pass and recorded in `delivery_event_partitions`; the per-workspace batches handle the legacy partition and anything left. |

The conversion is opt-in for this release: `reputecast migrate
partition-events`. The code works the same on either shape, so instances
convert when it suits them. A later release will convert the rest at
upgrade.

## Converting

Back up first ([backup and restore](backup-restore.md)), then:

```bash
reputecast migrate partition-events
```

It holds the migration lock, so two runs never overlap, and it can run
while ReputeCast runs:

1. Copies the dedupe keys of the last 180 days into `delivery_event_keys`.
2. Builds the new primary key's index with `CREATE INDEX CONCURRENTLY`
   (writes go on).
3. Adds `received_at < <first month>` as a constraint and validates it
   without blocking writes, so attaching the table later doesn't scan it
   under a lock.
4. In one transaction that waits at most 10 seconds for the table's lock
   (tried 5 times): renames the table to `delivery_events_legacy`, creates
   the partitioned `delivery_events` with the same columns, checks, foreign
   keys and indexes, attaches the old table as its first partition (its
   indexes are matched, not rebuilt) and adds the default partition.
5. Makes the monthly partitions and runs `ANALYZE`.

An interrupted run can run again: each step checks what is done. On a
converted database it only makes missing months.

On the one-million-contact load test data (3.2 million events, 1.4 GB) it
took 5 seconds, and every count and report read the same before and after
([performance](performance.md)).

## Rolling back

Converting back isn't automated. To undo it, restore the backup taken
before, or, while only the legacy partition holds rows, run in a
transaction: `ALTER TABLE delivery_events DETACH PARTITION
delivery_events_legacy`, drop the partitioned table and its monthly
partitions (after moving their rows into the legacy table), rename
`delivery_events_legacy` back to `delivery_events` and recreate its
`delivery_events_pkey (id)` primary key.

## Later

- Archive or drop months past a retention period by detaching their
  partition, once there is such a setting.
- Partition `messages` the same way, on `queued_at`.
