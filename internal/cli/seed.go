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
	return cmd
}
