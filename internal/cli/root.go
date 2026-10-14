// Package cli implements the reputecast command line with cobra.
package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/logging"
	"github.com/hostrepute/reputecast/internal/version"
)

type rootOpts struct {
	configFile string
}

// Execute runs the CLI and returns the process exit code.
func Execute() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := newRoot().ExecuteContext(ctx); err != nil {
		var ue usageError
		if errors.As(err, &ue) {
			fmt.Fprintln(os.Stderr, "Error:", err)
			return 2
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 1
	}
	return 0
}

type usageError struct{ error }

func newRoot() *cobra.Command {
	o := &rootOpts{}
	root := &cobra.Command{
		Use:           "reputecast",
		Short:         "ReputeCast: the self-hosted newsletter platform that protects your inbox placement",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.Get().Version,
	}
	root.PersistentFlags().StringVar(&o.configFile, "config", "", "path to a YAML config file (or REPUTECAST_CONFIG)")
	root.PersistentFlags().String("database-url", "", "Postgres URL (or REPUTECAST_DATABASE__URL / DATABASE_URL)")
	root.PersistentFlags().String("log-level", "", "debug, info, warn or error")
	root.AddCommand(
		newServeCmd(o),
		newWorkerCmd(o),
		newMigrateCmd(o),
		newInstallCmd(o),
		newUserCmd(o),
		newSeedCmd(o),
		newDemoCmd(o),
		newOpenAPICmd(),
		newSecretsCmd(o),
		newAdapterCmd(o),
		newVersionCmd(),
	)
	return root
}

// load reads configuration with the command's flags as the top layer.
func (o *rootOpts) load(cmd *cobra.Command) (*config.Config, *slog.Logger, error) {
	cfg, err := config.Load(o.configFile, cmd.Flags())
	if err != nil {
		return nil, nil, err
	}
	log := logging.New(os.Stderr, cfg.Log.Level, cfg.Log.Format)
	slog.SetDefault(log)
	return cfg, log, nil
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			v := version.Get()
			fmt.Fprintf(cmd.OutOrStdout(), "reputecast %s", v.Version)
			if v.Commit != "" {
				fmt.Fprintf(cmd.OutOrStdout(), " (%s)", shortCommit(v.Commit))
			}
			fmt.Fprintf(cmd.OutOrStdout(), " %s\n", v.GoVersion)
			return nil
		},
	}
}

func shortCommit(c string) string {
	if len(c) > 12 {
		return c[:12]
	}
	return c
}
