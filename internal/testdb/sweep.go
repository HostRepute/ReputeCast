package testdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Clones and demo templates carry their owner in their name: a tag for the
// host, the PID of the test binary (of the `go test` run, for a Shared
// template) and when it made them. A later run drops what a finished owner
// left behind on this host, and anything of another host past cloneMaxAge. Migration templates of other migration sets go once no
// run has used them for templateMaxAge.
const (
	cloneMaxAge    = 2 * time.Hour
	templateMaxAge = 24 * time.Hour
)

var hostTag = func() string {
	h, _ := os.Hostname()
	sum := sha256.Sum256([]byte(h))
	return hex.EncodeToString(sum[:3])
}()

// owned names a database this binary makes: prefix_host_pid_unix_rand.
func owned(prefix string) string {
	return prefix + "_" + hostTag + "_" + strconv.Itoa(os.Getpid()) + "_" + strconv.FormatInt(time.Now().Unix(), 10) + "_" + randHex(3)
}

// verdict is what the sweep does with a database.
type verdict int

const (
	keep     verdict = iota
	drop             // drop it, even while connections are open
	dropIdle         // drop it only if nobody is connected
)

// judgeClone decides about an rc_test_ or rc_fill_ database. Names without
// an owner come from older code: a timestamp, when present, still dates
// them; without one they go only when idle.
func judgeClone(name string, now time.Time, alive func(int) bool) verdict {
	parts := strings.Split(name, "_")
	n := len(parts)
	if n >= 6 && len(parts[n-4]) == 6 {
		host, pidStr, tsStr := parts[n-4], parts[n-3], parts[n-2]
		pid, perr := strconv.Atoi(pidStr)
		ts, terr := strconv.ParseInt(tsStr, 10, 64)
		if perr == nil && terr == nil {
			if host == hostTag {
				if !alive(pid) {
					return drop
				}
				return keep
			}
			if now.Sub(time.Unix(ts, 0)) > cloneMaxAge {
				return drop
			}
			return keep
		}
	}
	if n >= 2 {
		if ts, err := strconv.ParseInt(parts[n-2], 10, 64); err == nil && ts > 1_000_000_000 {
			if now.Sub(time.Unix(ts, 0)) > cloneMaxAge {
				return drop
			}
			return keep
		}
	}
	return dropIdle
}

// judgeTemplate decides about a migration template of another migration
// set from its comment ("<fingerprint> used:<unix>").
func judgeTemplate(comment string, now time.Time) verdict {
	if used, ok := usedAt(comment); ok && now.Sub(used) < templateMaxAge {
		return keep
	}
	return dropIdle
}

func usedAt(comment string) (time.Time, bool) {
	for _, f := range strings.Fields(comment) {
		if v, ok := strings.CutPrefix(f, "used:"); ok {
			if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
				return time.Unix(ts, 0), true
			}
		}
	}
	return time.Time{}, false
}

// templateComment marks the template complete for fingerprint and records
// when a run last used it.
func templateComment(fingerprint string, now time.Time) string {
	return fingerprint + " used:" + strconv.FormatInt(now.Unix(), 10)
}

// sweep drops what earlier runs left behind. It is best effort: a database
// another sweep dropped first, or one still in use, is skipped.
func sweep(ctx context.Context, admin *pgx.Conn, now time.Time) {
	rows, err := admin.Query(ctx, `SELECT datname, coalesce(shobj_description(oid, 'pg_database'), '') FROM pg_database
		WHERE (datname LIKE 'rc\_test\_%' OR datname LIKE 'rc\_fill\_%' OR datname LIKE 'rc\_tpl\_%') AND NOT datistemplate`)
	if err != nil {
		return
	}
	type db struct{ name, comment string }
	dbs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (db, error) {
		var d db
		return d, r.Scan(&d.name, &d.comment)
	})
	if err != nil {
		return
	}
	for _, d := range dbs {
		var v verdict
		switch {
		case d.name == templateName:
			continue
		case strings.HasPrefix(d.name, "rc_tpl_"):
			v = judgeTemplate(d.comment, now)
		default:
			v = judgeClone(d.name, now, processAlive)
		}
		var err error
		switch v {
		case keep:
			continue
		case drop:
			_, err = admin.Exec(ctx, `DROP DATABASE IF EXISTS `+pgx.Identifier{d.name}.Sanitize()+` WITH (FORCE)`)
		case dropIdle:
			_, err = admin.Exec(ctx, `DROP DATABASE IF EXISTS `+pgx.Identifier{d.name}.Sanitize())
		}
		// A shared template's files (Shared) go with it.
		if err == nil && strings.HasPrefix(d.name, "rc_fill_") {
			_ = os.RemoveAll(sharedDir(d.name))
		}
	}
}
