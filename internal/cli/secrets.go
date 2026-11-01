package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/hostrepute/reputecast/internal/app"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/sealed"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

func newSecretsCmd(o *rootOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage the key that encrypts credentials at rest",
	}
	var dryRun bool
	reseal := &cobra.Command{
		Use:   "reseal",
		Short: "Re-encrypt every stored secret with the current key, so previous keys can be dropped",
		Long: "After a key change (secrets.key set to a new key, the old one moved to secrets.previous_keys),\n" +
			"reseal re-encrypts every stored credential, token and secret with the new key. When it reports\n" +
			"nothing left on a previous key, remove that key from secrets.previous_keys after 24 hours\n" +
			"(signup forms hand out tokens sealed with it for that long). Safe to run while ReputeCast runs\n" +
			"and to run again. See docs/deploy/backup-restore.md.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, log, err := o.load(cmd)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			pool, err := db.Open(ctx, cfg.Database.URL, 4)
			if err != nil {
				return err
			}
			defer pool.Close()
			box, err := app.SecretBox(cfg, log)
			if err != nil {
				return err
			}
			res, err := sealed.Reseal(ctx, pool, box, dryRun)
			if err != nil {
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "VALUE\tON A PREVIOUS KEY\tRESEALED\tUNREADABLE")
			var old, done, bad int
			for _, r := range res {
				if r.Checked == 0 && !r.Missing {
					continue
				}
				_, _ = fmt.Fprintf(w, "%s\t%d\t%d\t%d\n", r.Column.Name(), r.Checked, r.Resealed, r.Unreadable)
				old, done, bad = old+r.Checked, done+r.Resealed, bad+r.Unreadable
			}
			_ = w.Flush()
			switch {
			case dryRun:
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Dry run: %d values would be resealed; %d no key opens.\n", old-bad, bad)
			default:
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Resealed %d values with the current key; %d no key opens.\n", done, bad)
				if done > 0 {
					q := store.New(pool)
					_ = audit.New(q, log).Log(ctx, audit.Event{Action: "instance.secrets_resealed", TargetType: "instance", TargetName: "Secret key",
						Metadata: map[string]any{"resealed": done, "unreadable": bad, "summary": fmt.Sprintf("Resealed %d stored secrets with the current key", done)}})
				}
			}
			if bad > 0 {
				return fmt.Errorf("%d values open with no configured key: they were sealed with a key that is gone, or are damaged", bad)
			}
			return nil
		},
	}
	reseal.Flags().BoolVar(&dryRun, "dry-run", false, "count what would be resealed, without writing")
	cmd.AddCommand(reseal)
	cmd.AddCommand(&cobra.Command{
		Use:   "generate-key",
		Short: "Print a new random key for REPUTECAST_SECRETS__KEY",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), secrets.GenerateKey())
			return err
		},
	})
	return cmd
}
