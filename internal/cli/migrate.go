package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/hostrepute/reputecast/internal/db"
)

func newMigrateCmd(o *rootOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Manage the database schema",
	}
	run := func(fn func(cmd *cobra.Command) error) func(*cobra.Command, []string) error {
		return func(cmd *cobra.Command, _ []string) error { return fn(cmd) }
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "Apply all pending migrations",
		RunE: run(func(cmd *cobra.Command) error {
			cfg, log, err := o.load(cmd)
			if err != nil {
				return err
			}
			pool, err := db.Open(cmd.Context(), cfg.Database.URL, 4)
			if err != nil {
				return err
			}
			defer pool.Close()
			if err := db.MigrateUp(cmd.Context(), pool, log); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Database is up to date.")
			return nil
		}),
	}, &cobra.Command{
		Use:   "partition-events",
		Short: "Split delivery_events into monthly partitions, without copying it (docs/deploy/partitioning.md)",
		Long: "Turns delivery_events into a table partitioned by month. Nothing is copied: the current table\n" +
			"becomes the partition of everything before next month, and later months get their own. It\n" +
			"builds an index without blocking writes, checks a bound, then swaps tables in a moment's lock.\n" +
			"Safe while ReputeCast runs and safe to run again. Back up first.",
		RunE: run(func(cmd *cobra.Command) error {
			cfg, log, err := o.load(cmd)
			if err != nil {
				return err
			}
			pool, err := db.Open(cmd.Context(), cfg.Database.URL, 4)
			if err != nil {
				return err
			}
			defer pool.Close()
			res, err := db.PartitionDeliveryEvents(cmd.Context(), pool, log, time.Now())
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if res.Already {
				_, err = fmt.Fprintf(out, "delivery_events is already partitioned; made %d partitions ahead.\n", len(res.Partitions))
				return err
			}
			_, err = fmt.Fprintf(out, "delivery_events is partitioned by month from %s (%s): %d dedupe keys kept, partitions %s.\n",
				res.FirstMonth.Format("2006-01"), res.Took.Round(time.Second), res.KeysCopied, strings.Join(res.Partitions, ", "))
			return err
		}),
	}, &cobra.Command{
		Use:   "segments-role",
		Short: "Set up the role SQL segments run as again (after restoring on a new server)",
		Long: "pg_dump doesn't carry roles. After restoring on a new server, create the role (as a superuser,\n" +
			"unless the app's user has CREATEROLE):\n\n" +
			"  CREATE ROLE reputecast_segments NOLOGIN NOINHERIT;\n  GRANT reputecast_segments TO <app user>;\n\n" +
			"then run this, which hands the segment function back to the role. Safe to run again.",
		RunE: run(func(cmd *cobra.Command) error {
			cfg, _, err := o.load(cmd)
			if err != nil {
				return err
			}
			pool, err := db.Open(cmd.Context(), cfg.Database.URL, 2)
			if err != nil {
				return err
			}
			defer pool.Close()
			if err := db.SetUpSegmentsRole(cmd.Context(), pool); err != nil {
				return err
			}
			c := segmentsRoleCheck(cmd.Context(), pool)
			_, err = fmt.Fprintln(cmd.OutOrStdout(), c.text)
			return err
		}),
	}, &cobra.Command{
		Use:   "down",
		Short: "Roll back the latest migration",
		RunE: run(func(cmd *cobra.Command) error {
			cfg, log, err := o.load(cmd)
			if err != nil {
				return err
			}
			pool, err := db.Open(cmd.Context(), cfg.Database.URL, 4)
			if err != nil {
				return err
			}
			defer pool.Close()
			return db.MigrateDown(cmd.Context(), pool, log)
		}),
	}, &cobra.Command{
		Use:   "status",
		Short: "List migrations and whether they are applied",
		RunE: run(func(cmd *cobra.Command) error {
			cfg, log, err := o.load(cmd)
			if err != nil {
				return err
			}
			pool, err := db.Open(cmd.Context(), cfg.Database.URL, 4)
			if err != nil {
				return err
			}
			defer pool.Close()
			st, err := db.Status(cmd.Context(), pool, log)
			if err != nil {
				return err
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "VERSION\tAPPLIED\tFILE")
			for _, s := range st {
				applied := "pending"
				if s.Applied {
					applied = s.AppliedAt.Format("2006-01-02 15:04")
				}
				fmt.Fprintf(tw, "%05d\t%s\t%s\n", s.Version, applied, s.Name)
			}
			return tw.Flush()
		}),
	})
	return cmd
}
