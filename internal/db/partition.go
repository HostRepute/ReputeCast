package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// delivery_events in monthly partitions (review 2026-11 OP-8; the plan is
// docs/deploy/partitioning.md). The table is partitioned by received_at,
// the time ReputeCast stored the event, so new events always land in the
// current month. The conversion copies nothing: the existing table becomes
// the partition of everything before the first month, and months after it
// get partitions of their own, made ahead of time. A default partition
// catches what falls outside them, so an insert never fails.

// EventPartitionsAhead is how many months past the current one always have
// a partition.
const EventPartitionsAhead = 2

// legacyPartition is the table the conversion turns into the first
// partition.
const legacyPartition = "delivery_events_legacy"

var monthPartition = regexp.MustCompile(`^delivery_events_(\d{4})_(\d{2})$`)

// EventPartition is one monthly partition.
type EventPartition struct {
	Name       string
	From, Till time.Time
}

// EventsPartitioned reports whether delivery_events is partitioned.
func EventsPartitioned(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) (bool, error) {
	var kind string
	err := q.QueryRow(ctx, `SELECT relkind::text FROM pg_class WHERE oid = 'delivery_events'::regclass`).Scan(&kind)
	return kind == "p", err
}

// EventPartitions lists the monthly partitions, oldest first (not the
// legacy and default ones).
func EventPartitions(ctx context.Context, pool *pgxpool.Pool) ([]EventPartition, error) {
	rows, err := pool.Query(ctx, `
		SELECT c.relname FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
		WHERE i.inhparent = 'delivery_events'::regclass ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EventPartition
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		if from, ok := partitionMonth(name); ok {
			out = append(out, EventPartition{Name: name, From: from, Till: from.AddDate(0, 1, 0)})
		}
	}
	return out, rows.Err()
}

// legacyEnd is where the legacy partition ends (zero without one).
func legacyEnd(ctx context.Context, pool *pgxpool.Pool) (time.Time, error) {
	var bound *string
	err := pool.QueryRow(ctx, `
		SELECT pg_get_expr(c.relpartbound, c.oid) FROM pg_class c
		WHERE c.relname = $1 AND c.relispartition`, legacyPartition).Scan(&bound)
	if errors.Is(err, pgx.ErrNoRows) || bound == nil {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	// FOR VALUES FROM (MINVALUE) TO ('2026-12-01 00:00:00+00')
	m := legacyBoundRe.FindStringSubmatch(*bound)
	if m == nil {
		return time.Time{}, fmt.Errorf("unreadable bound of %s: %s", legacyPartition, *bound)
	}
	var end time.Time
	if err := pool.QueryRow(ctx, `SELECT $1::timestamptz`, m[1]).Scan(&end); err != nil {
		return time.Time{}, err
	}
	return end.UTC(), nil
}

var legacyBoundRe = regexp.MustCompile(`TO \('([^']+)'\)`)

func partitionMonth(name string) (time.Time, bool) {
	m := monthPartition.FindStringSubmatch(name)
	if m == nil {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01", m[1]+"-"+m[2])
	return t.UTC(), err == nil
}

func partitionName(month time.Time) string {
	return "delivery_events_" + month.UTC().Format("2006_01")
}

func monthStart(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// EnsureEventPartitions makes the partitions of the current month and the
// EventPartitionsAhead months after it, when delivery_events is
// partitioned. It returns the partitions it made.
func EnsureEventPartitions(ctx context.Context, pool *pgxpool.Pool, now time.Time) ([]string, error) {
	ok, err := EventsPartitioned(ctx, pool)
	if err != nil || !ok {
		return nil, err
	}
	have, err := EventPartitions(ctx, pool)
	if err != nil {
		return nil, err
	}
	// Months before the end of the legacy partition belong to it.
	first, err := legacyEnd(ctx, pool)
	if err != nil {
		return nil, err
	}
	exists := map[string]bool{}
	for _, p := range have {
		exists[p.Name] = true
	}
	var made []string
	for i := 0; i <= EventPartitionsAhead; i++ {
		month := monthStart(now).AddDate(0, i, 0)
		if month.Before(first) || exists[partitionName(month)] {
			continue
		}
		_, err := pool.Exec(ctx, fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s PARTITION OF delivery_events FOR VALUES FROM ('%s') TO ('%s')`,
			partitionName(month), month.Format(time.RFC3339), month.AddDate(0, 1, 0).Format(time.RFC3339)))
		if err != nil {
			// Usually: the default partition already holds rows of that
			// month (the job didn't run for months). They stay there.
			return made, fmt.Errorf("partition for %s: %w", month.Format("2006-01"), err)
		}
		made = append(made, partitionName(month))
	}
	return made, nil
}

// PartitionResult says what the conversion did.
type PartitionResult struct {
	Already    bool
	KeysCopied int64
	FirstMonth time.Time
	Partitions []string
	Took       time.Duration
}

// PartitionDeliveryEvents converts delivery_events to monthly partitions
// without copying its rows. It holds the migration lock, can run while
// ReputeCast runs (the final swap takes the table's lock for a moment),
// and can run again after an interruption.
func PartitionDeliveryEvents(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger, now time.Time) (PartitionResult, error) {
	var res PartitionResult
	start := time.Now()
	err := withMigrateLock(ctx, pool, log, func() error {
		var err error
		res, err = partitionDeliveryEvents(ctx, pool, log, now)
		return err
	})
	res.Took = time.Since(start)
	return res, err
}

func partitionDeliveryEvents(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger, now time.Time) (PartitionResult, error) {
	var res PartitionResult
	done, err := EventsPartitioned(ctx, pool)
	if err != nil {
		return res, err
	}
	if done {
		res.Already = true
		res.Partitions, err = EnsureEventPartitions(ctx, pool, now)
		return res, err
	}

	// The first month: the next one, or later when events are stored with
	// a later time (a clock that was ahead).
	var latest *time.Time
	if err := pool.QueryRow(ctx, `SELECT max(received_at) FROM delivery_events`).Scan(&latest); err != nil {
		return res, err
	}
	first := monthStart(now).AddDate(0, 1, 0)
	if first.Sub(now) < 6*time.Hour {
		first = first.AddDate(0, 1, 0) // too close to the month's end to finish before it
	}
	if latest != nil && !latest.Before(first) {
		first = monthStart(*latest).AddDate(0, 1, 0)
	}
	res.FirstMonth = first

	// 1. The dedupe keys of the last 180 days, a month at a time.
	for from := monthStart(now.AddDate(0, 0, -180)); from.Before(now.Add(time.Hour)); from = from.AddDate(0, 1, 0) {
		tag, err := pool.Exec(ctx, `
			INSERT INTO delivery_event_keys (provider_id, dedupe_key, received_at)
			SELECT provider_id, dedupe_key, max(received_at) FROM delivery_events
			WHERE dedupe_key <> '' AND provider_id IS NOT NULL AND received_at >= $1 AND received_at < $2
			GROUP BY 1, 2
			ON CONFLICT DO NOTHING`, from, from.AddDate(0, 1, 0))
		if err != nil {
			return res, fmt.Errorf("dedupe keys: %w", err)
		}
		res.KeysCopied += tag.RowsAffected()
	}
	log.Info("partitioning delivery_events: dedupe keys copied", "keys", res.KeysCopied)

	// 2. The new primary key, (id, received_at), built without blocking
	// writes. An earlier interrupted build left an invalid index: drop it.
	var valid *bool
	_ = pool.QueryRow(ctx, `SELECT i.indisvalid FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid WHERE c.relname = 'delivery_events_legacy_pkey'`).Scan(&valid)
	if valid != nil && !*valid {
		if _, err := pool.Exec(ctx, `DROP INDEX CONCURRENTLY IF EXISTS delivery_events_legacy_pkey`); err != nil {
			return res, err
		}
	}
	if _, err := pool.Exec(ctx, `CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS delivery_events_legacy_pkey ON delivery_events (id, received_at)`); err != nil {
		return res, fmt.Errorf("primary key: %w", err)
	}
	log.Info("partitioning delivery_events: primary key built")

	// 3. A checked bound, so attaching the table as a partition doesn't scan
	// it under a lock.
	if _, err := pool.Exec(ctx, `ALTER TABLE delivery_events DROP CONSTRAINT IF EXISTS delivery_events_before_partitions`); err != nil {
		return res, err
	}
	if _, err := pool.Exec(ctx, fmt.Sprintf(`ALTER TABLE delivery_events ADD CONSTRAINT delivery_events_before_partitions CHECK (received_at < '%s') NOT VALID`,
		first.Format(time.RFC3339))); err != nil {
		return res, err
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE delivery_events VALIDATE CONSTRAINT delivery_events_before_partitions`); err != nil {
		return res, fmt.Errorf("bound: %w", err)
	}
	log.Info("partitioning delivery_events: bound checked", "first_month", first.Format("2006-01"))

	// 4. The swap, in one short transaction; it waits at most 10 seconds for
	// the table's lock, and is tried a few times.
	var indexes []string
	rows, err := pool.Query(ctx, `
		SELECT pg_get_indexdef(i.indexrelid) FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid
		WHERE i.indrelid = 'delivery_events'::regclass AND NOT i.indisprimary AND NOT i.indisunique
		  AND c.relname <> 'delivery_events_legacy_pkey'`)
	if err != nil {
		return res, err
	}
	for rows.Next() {
		var def string
		if err := rows.Scan(&def); err != nil {
			rows.Close()
			return res, err
		}
		indexes = append(indexes, def)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return res, err
	}
	var swapErr error
	for attempt := 1; attempt <= 5; attempt++ {
		swapErr = Tx(ctx, pool, func(tx pgx.Tx) error { return swapToPartitioned(ctx, tx, first, indexes) })
		if swapErr == nil || !lockTimeout(swapErr) {
			break
		}
		log.Warn("partitioning delivery_events: the table is busy, trying again", "attempt", attempt)
		time.Sleep(time.Duration(attempt) * 2 * time.Second)
	}
	if swapErr != nil {
		return res, fmt.Errorf("swap: %w", swapErr)
	}
	if _, err := pool.Exec(ctx, `ANALYZE delivery_events`); err != nil {
		return res, err
	}
	res.Partitions, err = EnsureEventPartitions(ctx, pool, now)
	log.Info("partitioning delivery_events: done", "first_month", first.Format("2006-01"), "partitions", res.Partitions)
	return res, err
}

// UpgradeSteps names the steps MigrateUp takes after the schema's
// migrations; a test database made before they changed is made again.
const UpgradeSteps = "op11-events-partitioned"

// legacyDedupeIndex is the unpartitioned table's unique index on a
// provider's dedupe key. A per-partition unique index can't hold across
// months, so delivery_event_keys has held the keys since the conversion was
// written (review 2026-11 OP-8); the old index stayed on the legacy
// partition until the upgrade that converts every instance dropped it.
const legacyDedupeIndex = "delivery_events_dedupe_key"

// upgradeEvents is the step of an upgrade that converts delivery_events
// (review 2026-11 OP-11). An instance that ran `reputecast migrate
// partition-events` only gets its months ahead; any other is converted
// here, the same way and as safely (nothing is copied, writes go on, the
// swap waits at most ten seconds for the table). Then the legacy dedupe
// index is dropped without blocking writes. The caller holds the migration
// lock.
func upgradeEvents(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger, now time.Time) error {
	res, err := partitionDeliveryEvents(ctx, pool, log, now)
	if err != nil {
		return fmt.Errorf("partition delivery_events: %w", err)
	}
	if !res.Already {
		log.Info("delivery_events converted to monthly partitions at upgrade", "first_month", res.FirstMonth.Format("2006-01"),
			"dedupe_keys", res.KeysCopied, "partitions", res.Partitions)
	}
	if _, err := pool.Exec(ctx, `DROP INDEX CONCURRENTLY IF EXISTS `+legacyDedupeIndex); err != nil {
		return fmt.Errorf("drop the legacy dedupe index: %w", err)
	}
	return nil
}

func lockTimeout(err error) bool {
	var pg interface{ SQLState() string }
	return errors.As(err, &pg) && pg.SQLState() == "55P03"
}

// swapToPartitioned renames the table to the legacy partition, makes the
// partitioned table in its place with the same columns, constraints,
// foreign keys and indexes, and attaches the old table (its indexes are
// matched, not built again).
func swapToPartitioned(ctx context.Context, tx pgx.Tx, first time.Time, indexes []string) error {
	bound := first.Format(time.RFC3339)
	for _, st := range []string{`SET LOCAL lock_timeout = '10s'`, `LOCK TABLE delivery_events IN ACCESS EXCLUSIVE MODE`} {
		if _, err := tx.Exec(ctx, st); err != nil {
			return err
		}
	}
	// The foreign keys, as the table has them.
	fks, err := tx.Query(ctx, `SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid = 'delivery_events'::regclass AND contype = 'f'`)
	if err != nil {
		return err
	}
	var fkDefs []string
	for fks.Next() {
		var def string
		if err := fks.Scan(&def); err != nil {
			fks.Close()
			return err
		}
		fkDefs = append(fkDefs, def)
	}
	fks.Close()
	if err := fks.Err(); err != nil {
		return err
	}
	stmts := []string{
		`ALTER TABLE delivery_events RENAME TO ` + legacyPartition,
		`ALTER TABLE ` + legacyPartition + ` DROP CONSTRAINT delivery_events_pkey`,
		`ALTER TABLE ` + legacyPartition + ` ADD CONSTRAINT delivery_events_legacy_pkey PRIMARY KEY USING INDEX delivery_events_legacy_pkey`,
		`CREATE TABLE delivery_events (LIKE ` + legacyPartition + ` INCLUDING DEFAULTS INCLUDING CONSTRAINTS INCLUDING STORAGE INCLUDING COMMENTS)
		 PARTITION BY RANGE (received_at)`,
		`ALTER TABLE delivery_events DROP CONSTRAINT delivery_events_before_partitions`,
		`ALTER TABLE delivery_events ADD CONSTRAINT delivery_events_pkey PRIMARY KEY (id, received_at)`,
	}
	for _, s := range stmts {
		if _, err := tx.Exec(ctx, s); err != nil {
			return fmt.Errorf("%s: %w", firstLine(s), err)
		}
	}
	for _, def := range fkDefs {
		if _, err := tx.Exec(ctx, `ALTER TABLE delivery_events ADD `+def); err != nil {
			return fmt.Errorf("foreign key %s: %w", def, err)
		}
	}
	// The old table's indexes, by the same definitions, under their names:
	// the old ones are renamed, and attaching matches them by definition.
	for _, def := range indexes {
		name, ok := indexName(def)
		if !ok {
			return fmt.Errorf("unreadable index definition %q", def)
		}
		if _, err := tx.Exec(ctx, `ALTER INDEX `+name+` RENAME TO `+legacyIndexName(name)); err != nil {
			return err
		}
		onParent := strings.Replace(def, " ON public."+legacyPartition+" ", " ON public.delivery_events ", 1)
		onParent = strings.Replace(onParent, " ON "+legacyPartition+" ", " ON delivery_events ", 1)
		if _, err := tx.Exec(ctx, onParent); err != nil {
			return fmt.Errorf("index %s: %w", name, err)
		}
	}
	if _, err := tx.Exec(ctx, `ALTER TABLE delivery_events ATTACH PARTITION `+legacyPartition+` FOR VALUES FROM (MINVALUE) TO ('`+bound+`')`); err != nil {
		return fmt.Errorf("attach: %w", err)
	}
	if _, err := tx.Exec(ctx, `CREATE TABLE delivery_events_default PARTITION OF delivery_events DEFAULT`); err != nil {
		return fmt.Errorf("default partition: %w", err)
	}
	return nil
}

var indexNameRe = regexp.MustCompile(`^CREATE (?:UNIQUE )?INDEX (\S+) ON `)

func indexName(def string) (string, bool) {
	m := indexNameRe.FindStringSubmatch(def)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// legacyIndexName keeps the name under Postgres's 63 bytes.
func legacyIndexName(name string) string {
	const suffix = "_legacy"
	if len(name)+len(suffix) > 63 {
		name = name[:63-len(suffix)]
	}
	return name + suffix
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
