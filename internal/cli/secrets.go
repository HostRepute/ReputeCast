package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hostrepute/reputecast/internal/secrets"
)

func newSecretsCmd(_ *rootOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage the key that encrypts credentials at rest",
	}
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
