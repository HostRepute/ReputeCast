package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hostrepute/reputecast/internal/app"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/demo"
)

func newSeedCmd(o *rootOpts) *cobra.Command {
	var reset bool
	var sample string
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
			if sample != "" && sample != "acme" {
				return usageError{fmt.Errorf("unknown sample %q: use acme or northwind", sample)}
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
	cmd.Flags().StringVar(&sample, "sample", "acme", "which sample: acme, or northwind (an agency with seven clients, into an empty database)")
	return cmd
}
