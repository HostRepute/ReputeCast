package db

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Tables in monthly partitions: delivery_events (OP-8) and the site
// tracker's tracker_events (M62). A month's partition is named
// <table>_YYYY_MM and holds [the 1st of the month, the 1st of the next) in
// UTC; <table>_default catches what falls outside them, so an insert never
// fails.

// MonthlyTable is a table partitioned by month on Key.
type MonthlyTable struct {
	Name string
	// Key is the partition key's column.
	Key string
	// Ahead is how many months past the current one always have a
	// partition.
	Ahead int
	// Adopt moves the rows the default partition holds for a month into
	// the month's partition when it is made (else making it fails, and the
	// rows stay where they are).
	Adopt bool
}

var identRe = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

func (t MonthlyTable) check() {
	if !identRe.MatchString(t.Name) || !identRe.MatchString(t.Key) {
		panic("db: bad monthly table " + t.Name + "(" + t.Key + ")")
	}
}

func (t MonthlyTable) partition(month time.Time) string {
	return t.Name + "_" + month.UTC().Format("2006_01")
}

func (t MonthlyTable) month(name string) (time.Time, bool) {
	rest, ok := strings.CutPrefix(name, t.Name+"_")
	if !ok || len(rest) != 7 || rest[4] != '_' {
		return time.Time{}, false
	}
	m, err := time.Parse("2006_01", rest)
	return m.UTC(), err == nil
}

func monthStart(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// MonthlyPartitions lists t's monthly partitions, oldest first (not its
// default partition nor any other).
func MonthlyPartitions(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, t MonthlyTable) ([]EventPartition, error) {
	t.check()
	rows, err := q.Query(ctx, `
		SELECT c.relname FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
		WHERE i.inhparent = $1::text::regclass ORDER BY 1`, t.Name)
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
		if from, ok := t.month(name); ok {
			out = append(out, EventPartition{Name: name, From: from, Till: from.AddDate(0, 1, 0)})
		}
	}
	return out, rows.Err()
}

// EnsureMonthlyPartitions makes t's partitions of the month of now and the
// t.Ahead months after it, but none of a month before notBefore. It returns
// the partitions it made.
func EnsureMonthlyPartitions(ctx context.Context, pool *pgxpool.Pool, t MonthlyTable, now, notBefore time.Time) ([]string, error) {
	have, err := MonthlyPartitions(ctx, pool, t)
	if err != nil {
		return nil, err
	}
	exists := map[string]bool{}
	for _, p := range have {
		exists[p.Name] = true
	}
	var made []string
	for i := 0; i <= t.Ahead; i++ {
		month := monthStart(now).AddDate(0, i, 0)
		name := t.partition(month)
		if month.Before(notBefore) || exists[name] {
			continue
		}
		if err := makeMonth(ctx, pool, t, month); err != nil {
			return made, fmt.Errorf("partition for %s: %w", month.Format("2006-01"), err)
		}
		made = append(made, name)
	}
	return made, nil
}

// makeMonth makes one month's partition. When the default partition
// already holds rows of that month (the job didn't run for months), they
// stay there unless t adopts them: then the partition is made beside the
// table, the rows move into it and it is attached, in one transaction.
func makeMonth(ctx context.Context, pool *pgxpool.Pool, t MonthlyTable, month time.Time) error {
	name := pgx.Identifier{t.partition(month)}.Sanitize()
	table := pgx.Identifier{t.Name}.Sanitize()
	from, till := month.Format(time.RFC3339), month.AddDate(0, 1, 0).Format(time.RFC3339)
	_, err := pool.Exec(ctx, fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s PARTITION OF %s FOR VALUES FROM ('%s') TO ('%s')`, name, table, from, till))
	if err == nil || !t.Adopt || !checkViolation(err) {
		return err
	}
	def := pgx.Identifier{t.Name + "_default"}.Sanitize()
	key := pgx.Identifier{t.Key}.Sanitize()
	return Tx(ctx, pool, func(tx pgx.Tx) error {
		for _, st := range []string{
			`SET LOCAL lock_timeout = '10s'`,
			fmt.Sprintf(`CREATE TABLE %s (LIKE %s INCLUDING DEFAULTS INCLUDING CONSTRAINTS INCLUDING STORAGE)`, name, table),
			fmt.Sprintf(`WITH moved AS (DELETE FROM %s WHERE %s >= '%s' AND %s < '%s' RETURNING *) INSERT INTO %s SELECT * FROM moved`,
				def, key, from, key, till, name),
			fmt.Sprintf(`ALTER TABLE %s ATTACH PARTITION %s FOR VALUES FROM ('%s') TO ('%s')`, table, name, from, till),
		} {
			if _, err := tx.Exec(ctx, st); err != nil {
				return fmt.Errorf("%s: %w", firstLine(st), err)
			}
		}
		return nil
	})
}

// checkViolation reports a check violation: making a partition whose rows
// the default partition holds.
func checkViolation(err error) bool {
	var pg interface{ SQLState() string }
	return errors.As(err, &pg) && pg.SQLState() == "23514"
}

// DropMonthlyPartitions drops t's partitions that end at or before before,
// a whole month at a time (each waits at most five seconds for the table's
// lock; one that can't get it is dropped next time). It returns the
// partitions it dropped.
func DropMonthlyPartitions(ctx context.Context, pool *pgxpool.Pool, t MonthlyTable, before time.Time) ([]string, error) {
	have, err := MonthlyPartitions(ctx, pool, t)
	if err != nil {
		return nil, err
	}
	var dropped []string
	for _, p := range have {
		if p.Till.After(before) {
			continue
		}
		err := Tx(ctx, pool, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `SET LOCAL lock_timeout = '5s'`); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, `DROP TABLE IF EXISTS `+pgx.Identifier{p.Name}.Sanitize())
			return err
		})
		if lockTimeout(err) {
			continue
		}
		if err != nil {
			return dropped, fmt.Errorf("drop %s: %w", p.Name, err)
		}
		dropped = append(dropped, p.Name)
	}
	return dropped, nil
}
