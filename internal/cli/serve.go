package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/hostrepute/reputecast/internal/app"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/db"
)

func newServeCmd(o *rootOpts) *cobra.Command {
	var autoMigrate, autoInstall bool
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the web app, API, webhook ingest and workers",
		Long: "Run ReputeCast. By default one process runs every role; use --role=web,worker,ingest\n" +
			"to split them across processes (for example several web and ingest nodes and one worker).",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, log, err := o.load(cmd)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			a, err := app.New(ctx, cfg, log)
			if err != nil {
				return err
			}
			defer a.Close()
			if autoMigrate {
				if err := db.MigrateUp(ctx, a.Pool, log); err != nil {
					return err
				}
			} else if pending, err := db.HasPending(ctx, a.Pool, log); err == nil && pending {
				log.Warn("database has pending migrations; run `reputecast migrate up` or pass --auto-migrate")
			}
			if autoInstall {
				if res, err := installIfEmpty(ctx, a.Q, cfg, installInput{}); err != nil {
					return err
				} else if res != nil {
					printInstall(cmd.ErrOrStderr(), cfg, res)
				}
			}
			return runServe(ctx, a)
		},
	}
	cmd.Flags().String("role", "", "comma-separated roles: web, worker, ingest (default all)")
	cmd.Flags().String("addr", "", "listen address (default :8080)")
	cmd.Flags().BoolVar(&autoMigrate, "auto-migrate", envBool("REPUTECAST_AUTO_MIGRATE", false), "apply pending migrations at start (REPUTECAST_AUTO_MIGRATE)")
	cmd.Flags().BoolVar(&autoInstall, "auto-install", envBool("REPUTECAST_AUTO_INSTALL", false), "create the first Owner when the database has no users (REPUTECAST_AUTO_INSTALL; uses REPUTECAST_ADMIN_EMAIL)")
	return cmd
}

// runServe runs the app and, in web processes, the SMTP relay (off until an
// instance admin turns it on); both stop gracefully when ctx ends.
func runServe(ctx context.Context, a *app.App) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	relay := make(chan error, 1)
	go func() { relay <- a.RunSMTPRelay(ctx) }()
	err := a.Run(ctx)
	cancel()
	if rerr := <-relay; rerr != nil && err == nil {
		err = rerr
	}
	return err
}

func newWorkerCmd(o *rootOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "worker",
		Short: "Run only background workers (same as serve --role=worker)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, log, err := o.load(cmd)
			if err != nil {
				return err
			}
			cfg.Roles = []string{config.RoleWorker}
			a, err := app.New(cmd.Context(), cfg, log)
			if err != nil {
				return err
			}
			defer a.Close()
			return a.Run(cmd.Context())
		},
	}
}
