package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"

	"github.com/hostrepute/reputecast/internal/app"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/geoip"
	"github.com/hostrepute/reputecast/internal/sealed"
	"github.com/hostrepute/reputecast/internal/secrets"
)

// doctor checks what a restore or a move to another host can get wrong
// (review 2026-11 OP-4): the database and its schema, the secret key
// against the values it sealed, the files folder and the role SQL
// segments run as, which pg_dump doesn't carry.

type checkLevel int

const (
	checkOK checkLevel = iota
	checkWarn
	checkFail
)

type check struct {
	name  string
	level checkLevel
	text  string
}

func newDoctorCmd(o *rootOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check the database, schema, secret key, files folder and roles (after a restore or a move)",
		Long: "Check what a restore or a move to another host can get wrong: the database and its schema,\n" +
			"whether the secret key opens the values it sealed, the files folder, the GeoIP database, and the role\n" +
			"SQL segments run as (pg_dump doesn't carry roles). Exits 1 when a check fails. See docs/deploy/backup-restore.md.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, log, err := o.load(cmd)
			if err != nil {
				return err
			}
			checks := runDoctor(cmd.Context(), cfg, func(cfg *config.Config) (*secrets.Box, error) { return app.SecretBox(cfg, log) })
			return printChecks(cmd.OutOrStdout(), checks)
		},
	}
}

func printChecks(w io.Writer, checks []check) error {
	failed := 0
	for _, c := range checks {
		mark := "ok  "
		switch c.level {
		case checkWarn:
			mark = "warn"
		case checkFail:
			mark = "FAIL"
			failed++
		}
		_, _ = fmt.Fprintf(w, "[%s] %-18s %s\n", mark, c.name, c.text)
	}
	if failed > 0 {
		return fmt.Errorf("%d checks failed", failed)
	}
	return nil
}

// runDoctor runs every check; a check that can't run fails, and the
// others still run.
func runDoctor(ctx context.Context, cfg *config.Config, openBox func(*config.Config) (*secrets.Box, error)) []check {
	var out []check
	add := func(name string, level checkLevel, format string, args ...any) {
		out = append(out, check{name: name, level: level, text: fmt.Sprintf(format, args...)})
	}
	pool, err := db.Open(ctx, cfg.Database.URL, 2)
	if err != nil {
		add("Database", checkFail, "can't connect: %v", err)
		return append(out, filesCheck(cfg), geoipCheck(cfg))
	}
	defer pool.Close()
	var num int
	var who string
	if err := pool.QueryRow(ctx, `SELECT current_setting('server_version_num')::int, current_user`).Scan(&num, &who); err != nil {
		add("Database", checkFail, "can't query: %v", err)
	} else if major := num / 10000; major < 16 || major > 18 {
		add("Database", checkWarn, "PostgreSQL %d as %s: ReputeCast supports 16 to 18", major, who)
	} else {
		add("Database", checkOK, "PostgreSQL %d as %s", major, who)
	}

	st, err := db.Schema(ctx, pool)
	switch {
	case err != nil:
		add("Schema", checkFail, "can't read it: %v", err)
	case len(st.Pending) > 0 || !st.RiverCurrent:
		add("Schema", checkFail, "%d migrations pending (newest applied %d, this release needs %d): run `reputecast migrate up`", len(st.Pending), st.Applied, st.Expected)
	case st.Newer:
		add("Schema", checkWarn, "migrated by a newer release (%d, this one knows %d): downgrades are supported one minor release back", st.Applied, st.Expected)
	default:
		add("Schema", checkOK, "current (%d)", st.Applied)
	}

	out = append(out, keyChecks(ctx, cfg, pool, openBox)...)
	out = append(out, filesCheck(cfg), geoipCheck(cfg))
	out = append(out, segmentsRoleCheck(ctx, pool))
	if u, err := url.Parse(cfg.HTTP.PublicURL); err != nil || u.Host == "" {
		add("Public URL", checkFail, "http.public_url isn't a URL: %q", cfg.HTTP.PublicURL)
	} else if local := u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"; local && cfg.HTTP.PublicURLLocked {
		add("Public URL", checkWarn, "%s is locked (http.public_url_locked) but only this machine reaches it: set REPUTECAST_PUBLIC_URL to the address people open", cfg.HTTP.PublicURL)
	} else if u.Scheme != "https" && !local {
		add("Public URL", checkWarn, "%s isn't https: links in emails and sign-in cookies need it", cfg.HTTP.PublicURL)
	} else {
		add("Public URL", checkOK, "%s", cfg.HTTP.PublicURL)
	}
	return out
}

func keyChecks(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool, openBox func(*config.Config) (*secrets.Box, error)) []check {
	var out []check
	if cfg.Secrets.Key == "" && cfg.Secrets.KeyFile != "" {
		if fi, err := os.Stat(cfg.Secrets.KeyFile); err != nil {
			out = append(out, check{"Secret key file", checkFail, fmt.Sprintf("%s: %v", cfg.Secrets.KeyFile, err)})
			return out
		} else if fi.Mode().Perm()&0o077 != 0 {
			out = append(out, check{"Secret key file", checkWarn, fmt.Sprintf("%s is readable by others (%s): chmod 600 it", cfg.Secrets.KeyFile, fi.Mode().Perm())})
		}
	}
	box, err := openBox(cfg)
	if err != nil {
		return append(out, check{"Secret key", checkFail, err.Error()})
	}
	res, err := sealed.Check(ctx, pool, box, 3)
	if err != nil {
		return append(out, check{"Secret key", checkFail, fmt.Sprintf("can't read sealed values: %v", err)})
	}
	var checked, bad, old int
	var badNames []string
	for _, r := range res {
		checked += r.Checked
		bad += r.Unreadable
		old += r.Old
		if r.Unreadable > 0 {
			badNames = append(badNames, r.Column.Name())
		}
	}
	switch {
	case bad > 0:
		out = append(out, check{"Secret key", checkFail, fmt.Sprintf("%d of %d sampled values don't open (%s): is this the key backed up with the database?",
			bad, checked, strings.Join(badNames, ", "))})
	case checked == 0:
		out = append(out, check{"Secret key", checkOK, "nothing sealed yet"})
	default:
		out = append(out, check{"Secret key", checkOK, fmt.Sprintf("opens the %d values sampled", checked)})
	}
	if old > 0 {
		out = append(out, check{"Previous keys", checkWarn, fmt.Sprintf("%d values still use a previous key: run `reputecast secrets reseal`", old)})
	}
	return out
}

// filesCheck checks the files folder exists, can be written and how much
// it holds.
func filesCheck(cfg *config.Config) check {
	dir := cfg.FilesDir()
	fi, err := os.Stat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return check{"Files folder", checkWarn, dir + " doesn't exist yet: it is created with the first upload. Restore it with the database when you have one."}
	}
	if err != nil || !fi.IsDir() {
		return check{"Files folder", checkFail, fmt.Sprintf("%s isn't a folder: %v", dir, err)}
	}
	f, err := os.CreateTemp(dir, ".doctor-*")
	if err != nil {
		return check{"Files folder", checkFail, fmt.Sprintf("%s can't be written: %v", dir, err)}
	}
	_ = f.Close()
	_ = os.Remove(f.Name())
	var size int64
	var files int
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if info, err := d.Info(); err == nil {
				size += info.Size()
				files++
			}
		}
		return nil
	})
	return check{"Files folder", checkOK, fmt.Sprintf("%s: %d files, %s", dir, files, humanBytes(size))}
}

// geoipCheck opens the GeoIP database the way serve does, but at once.
func geoipCheck(cfg *config.Config) check {
	if cfg.GeoIP.Database == "" {
		return check{"GeoIP", checkOK, "off (geoip.database isn't set): reports show no countries"}
	}
	d, err := geoip.Open(cfg.GeoIP.Database)
	if err != nil {
		return check{"GeoIP", checkWarn, fmt.Sprintf("%v: reports show no countries until it is fixed", err)}
	}
	return check{"GeoIP", checkOK, d.Describe()}
}

// segmentsRoleCheck checks the role SQL segments run as.
func segmentsRoleCheck(ctx context.Context, pool *pgxpool.Pool) check {
	var role, member, fn bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'reputecast_segments'),
		       EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'reputecast_segments') AND pg_has_role(current_user, 'reputecast_segments', 'MEMBER'),
		       EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid = p.pronamespace JOIN pg_roles r ON r.oid = p.proowner
		               WHERE n.nspname = 'reputecast_sql' AND p.proname = 'segment_ids' AND r.rolname = 'reputecast_segments')`).Scan(&role, &member, &fn)
	switch {
	case err != nil:
		return check{"SQL segments role", checkFail, err.Error()}
	case !role:
		return check{"SQL segments role", checkWarn, "reputecast_segments doesn't exist (pg_dump doesn't carry roles), so SQL segments are off: " +
			"CREATE ROLE reputecast_segments NOLOGIN NOINHERIT; GRANT reputecast_segments TO <app user>; then `reputecast migrate segments-role`"}
	case !member:
		return check{"SQL segments role", checkWarn, "the app's user isn't a member of reputecast_segments: GRANT reputecast_segments TO <app user>"}
	case !fn:
		return check{"SQL segments role", checkWarn, "reputecast_sql.segment_ids is missing or not owned by reputecast_segments: run `reputecast migrate segments-role`"}
	}
	return check{"SQL segments role", checkOK, "reputecast_segments runs SQL segments"}
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return strconv.FormatInt(n, 10) + " B"
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
