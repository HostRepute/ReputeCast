package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/spf13/cobra"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/blobstore"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

func newStorageCmd(o *rootOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "storage",
		Short: "Move the files kept outside the database to another storage backend",
	}
	var to string
	var dryRun bool
	migrate := &cobra.Command{
		Use:   "migrate --to s3",
		Short: "Copy the files folder into the S3-compatible bucket storage.s3 names",
		Long: "Copies every file of the files folder (files.dir) into the bucket set in storage.s3, under the\n" +
			"same keys, skipping those the bucket holds already at the same size, and reads each copy back\n" +
			"to check its SHA-256. Local files are never deleted. Run it, then set storage.backend to s3 and\n" +
			"restart every process; run it again just before the switch to copy what was uploaded meanwhile.\n" +
			"Safe to run while ReputeCast runs and to run again. See docs/deploy/configuration.md.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if to != config.StorageS3 {
				return usageError{fmt.Errorf("--to must be s3 (the files folder is where files come from), got %q", to)}
			}
			cfg, log, err := o.load(cmd)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			dst, err := migrationBucket(ctx, cfg)
			if err != nil {
				return err
			}
			src := blobstore.Local{Dir: cfg.FilesDir()}
			st, err := migrateFiles(ctx, cmd.OutOrStdout(), cmd.ErrOrStderr(), src, dst, migrationTarget(cfg.Storage.S3), dryRun)
			if err != nil {
				return err
			}
			if !dryRun && st.Copied > 0 {
				auditMigration(ctx, cfg, log, st)
			}
			if st.Failed > 0 {
				return fmt.Errorf("%d files weren't copied (listed above): run the command again", st.Failed)
			}
			return nil
		},
	}
	migrate.Flags().StringVar(&to, "to", "", "the backend to copy to: s3")
	migrate.Flags().BoolVar(&dryRun, "dry-run", false, "count what would be copied, without copying")
	_ = migrate.MarkFlagRequired("to")
	cmd.AddCommand(migrate)
	return cmd
}

// migrationBucket opens the bucket storage.s3 names, refusing settings
// that don't name one, and checks it answers.
func migrationBucket(ctx context.Context, cfg *config.Config) (*blobstore.S3, error) {
	c := *cfg
	c.Storage.Backend = config.StorageS3
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("storage.s3 isn't set up to copy to: set storage.s3.bucket (REPUTECAST_STORAGE__S3__BUCKET), "+
			"and for a service other than AWS storage.s3.endpoint, storage.s3.path_style and the access keys; then run this again.\n%w", err)
	}
	dst, err := blobstore.NewS3(ctx, c.Storage.S3)
	if err != nil {
		return nil, err
	}
	if err := dst.Check(ctx); err != nil {
		return nil, fmt.Errorf("storage.s3: %w", err)
	}
	return dst, nil
}

// migrationTarget names the bucket as the command's output does.
func migrationTarget(c config.S3Config) string {
	where := "the bucket " + c.Bucket
	if p := c.Prefix; p != "" {
		where += " (under " + p + ")"
	}
	return where
}

// migrateFiles copies src into dst (counting only, for a dry run) and
// writes what it did; each file that fails is named on errw.
func migrateFiles(ctx context.Context, out, errw io.Writer, src, dst blobstore.Store, target string, dryRun bool) (blobstore.CopyStats, error) {
	var mu sync.Mutex
	st, err := blobstore.Copy(ctx, dst, src, blobstore.CopyOptions{DryRun: dryRun, Verify: true, Failed: func(key string, err error) {
		mu.Lock()
		defer mu.Unlock()
		_, _ = fmt.Fprintf(errw, "failed: %s: %v\n", key, err)
	}})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return st, errors.New("stopped before every file was copied: run the command again to copy the rest")
		}
		return st, err
	}
	from := ""
	if l, ok := src.(blobstore.Local); ok {
		from = " in " + l.Dir
	}
	_, _ = fmt.Fprintf(out, "%d files, %s%s.\n", st.Objects, humanBytes(st.Bytes), from)
	if dryRun {
		_, _ = fmt.Fprintf(out, "Dry run: %d files (%s) would be copied to %s; %d are there already at the same size; %d couldn't be checked.\n",
			st.Copied, humanBytes(st.CopiedBytes), target, st.Skipped, st.Failed)
		return st, nil
	}
	_, _ = fmt.Fprintf(out, "Copied %d files (%s) to %s and checked each; %d were there already at the same size; %d failed.\n",
		st.Copied, humanBytes(st.CopiedBytes), target, st.Skipped, st.Failed)
	if st.Failed == 0 {
		_, _ = fmt.Fprintln(out, "The local files are still there. Set storage.backend to s3 (REPUTECAST_STORAGE__BACKEND=s3) and restart every process.")
	}
	return st, nil
}

// auditMigration records a copy in the instance audit log; without the
// database it only warns, as the files are copied all the same.
func auditMigration(ctx context.Context, cfg *config.Config, log *slog.Logger, st blobstore.CopyStats) {
	pool, err := db.Open(ctx, cfg.Database.URL, 2)
	if err != nil {
		log.Warn("the copy isn't in the audit log: the database can't be reached", "err", err)
		return
	}
	defer pool.Close()
	if err := audit.New(store.New(pool), log).Log(ctx, audit.Event{Action: "instance.storage_migrated", TargetType: "instance", TargetName: "File storage",
		Metadata: map[string]any{"to": config.StorageS3, "bucket": cfg.Storage.S3.Bucket, "copied": st.Copied, "bytes": st.CopiedBytes, "skipped": st.Skipped, "failed": st.Failed,
			"summary": fmt.Sprintf("Copied %d files from the files folder to the bucket %s", st.Copied, cfg.Storage.S3.Bucket)}}); err != nil {
		log.Warn("the copy isn't in the audit log", "err", err)
	}
}
