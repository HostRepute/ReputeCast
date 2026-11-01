package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/hostrepute/reputecast/internal/app"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/loadtest"
)

func newSeedCmd(o *rootOpts) *cobra.Command {
	var reset bool
	var sample string
	var contacts int
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Load the Acme demo data (development and screenshots only)",
		Long: "Loads the sample instance from the design: Acme Newsletter with 48,213 subscribers,\n" +
			"Amazon SES, Postmark and a warming Postal server, campaigns, automations, team and audit history.\n" +
			"Every demo account uses the password printed at the end.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, log, err := o.load(cmd)
			if err != nil {
				return err
			}
			if !cfg.IsDevelopment() && !reset {
				return usageError{fmt.Errorf("refusing to seed demo data outside development; set REPUTECAST_ENV=development")}
			}
			pool, err := db.Open(cmd.Context(), cfg.Database.URL, 8)
			if err != nil {
				return err
			}
			defer pool.Close()
			if err := db.MigrateUp(cmd.Context(), pool, log); err != nil {
				return err
			}
			box, err := app.SecretBox(cfg, log)
			if err != nil {
				return err
			}
			if sample == "northwind" {
				// The agency sample (agency accounts): its own database.
				res, err := demo.SeedNorthwind(cmd.Context(), pool, cfg, box, log)
				if err != nil {
					return err
				}
				out := cmd.OutOrStdout()
				fmt.Fprintf(out, "\n  Sample loaded (Northwind Studio, an agency with seven clients).\n\n")
				fmt.Fprintf(out, "  Agency manager  %s   password %q   authenticator secret %s\n", res.OwnerEmail, res.Password, res.TOTPSecret)
				fmt.Fprintf(out, "  Instance admin  %s   (same password and secret)\n", demo.NorthwindOwner)
				fmt.Fprintf(out, "  Client approver jane.morales@contoso.example   password %q\n\n", res.Password)
				return nil
			}
			if sample == "load" {
				// The load test data set (review 2026-11 OP-9), on top of an
				// instance that has a user.
				res, err := loadtest.Seed(cmd.Context(), pool, contacts, log)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "\n  Load test data loaded in %s: workspace %q, %d contacts, %d messages, %d delivery events.\n\n",
					res.Took.Round(time.Second), loadtest.Slug, res.Contacts, res.Messages, res.Events)
				return nil
			}
			if sample != "" && sample != "acme" {
				return usageError{fmt.Errorf("unknown sample %q: use acme, northwind or load", sample)}
			}
			res, err := demo.Seed(cmd.Context(), pool, cfg, box, log)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "\n  Demo data loaded (Acme ReputeCast).\n\n")
			fmt.Fprintf(out, "  Owner      %s   password %q   authenticator secret %s\n", res.OwnerEmail, res.Password, res.TOTPSecret)
			fmt.Fprintf(out, "  Viewer     aisha@acme.io   password %q (no two-factor yet)\n", res.Password)
			fmt.Fprintf(out, "  Recovery   linh's codes: %s …\n\n", demo.DemoRecoveryCodes[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&reset, "force", false, "seed even when REPUTECAST_ENV is not development")
	cmd.Flags().StringVar(&sample, "sample", "acme", "which sample: acme; northwind (an agency with seven clients, into an empty database); or load (a large workspace for load tests, after acme)")
	cmd.Flags().IntVar(&contacts, "contacts", 1_000_000, "with --sample load: how many contacts")
	return cmd
}
