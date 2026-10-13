package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/hostrepute/reputecast/internal/app"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/providers/remote"
	"github.com/hostrepute/reputecast/internal/store"
)

// maxManifest is the largest manifest.json read.
const maxManifest = 256 << 10

// newAdapterCmd manages community adapters from the server: the operator
// runs the adapter process and tells ReputeCast where it listens.
func newAdapterCmd(o *rootOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "adapter",
		Short: "Install community adapters and tell ReputeCast where they run (docs/adapters.md)",
		Long: "Community adapters run in their own process and speak the adapter protocol. Install one from its\n" +
			"manifest, run its process with the signing secret ReputeCast prints, and check it. Loopback and\n" +
			"private addresses need adapters.allow_private_networks (REPUTECAST_ADAPTERS__ALLOW_PRIVATE_NETWORKS=true).",
	}
	run := func(fn func(ctx context.Context, cmd *cobra.Command, s *remote.Service) error) func(*cobra.Command, []string) error {
		return func(cmd *cobra.Command, _ []string) error {
			cfg, log, err := o.load(cmd)
			if err != nil {
				return err
			}
			s, closeFn, err := adapterService(cmd.Context(), cfg, log)
			if err != nil {
				return err
			}
			defer closeFn()
			return fn(cmd.Context(), cmd, s)
		}
	}
	find := func(ctx context.Context, s *remote.Service, id string) (store.RemoteAdapterGetRow, error) {
		return s.RowByKey(ctx, strings.TrimSpace(id))
	}
	var endpoint string
	list := &cobra.Command{
		Use:   "list",
		Short: "List installed community adapters, where they run and their last check",
		Args:  cobra.NoArgs,
		RunE: run(func(ctx context.Context, cmd *cobra.Command, s *remote.Service) error {
			all, err := s.List(ctx)
			if err != nil {
				return err
			}
			if len(all) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No community adapters installed. Install one with `reputecast adapter install <manifest>`.")
				return nil
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 2, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ADAPTER\tVERSION\tADDRESS\tSECRET ISSUED\tLAST CHECK\tCONNECTED")
			for _, a := range all {
				addr := a.EndpointURL
				if addr == "" {
					addr = "not set"
				}
				check := "never"
				if a.Check != nil {
					check = "ok"
					if !a.Check.OK {
						check = "failed"
					}
					check += " · " + a.Check.At.Local().Format("2006-01-02 15:04")
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\n", a.Adapter, a.Version, addr, a.SecretIssuedAt.Local().Format("2006-01-02 15:04"), check, a.Connected)
			}
			return w.Flush()
		}),
	}
	install := &cobra.Command{
		Use:   "install <manifest-url-or-file>",
		Short: "Install a community adapter from its manifest.json, or update it when its id is installed",
		Long: "Reads the manifest from an https:// URL (public addresses only) or a local file. On a first\n" +
			"install ReputeCast prints the adapter's signing secret once: set it on the adapter process as\n" +
			"REPUTECAST_ADAPTER_SECRET.",
		Example: "  reputecast adapter install https://raw.githubusercontent.com/you/example-esp-adapter/main/manifest.json --url http://example-esp:4600\n" +
			"  reputecast adapter install ./manifest.json --url http://127.0.0.1:4600",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, source, err := readManifest(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return run(func(ctx context.Context, cmd *cobra.Command, s *remote.Service) error {
				res, err := s.Install(ctx, raw, source, endpoint, remote.CLIActor)
				if err != nil {
					return err
				}
				out := cmd.OutOrStdout()
				verb := "Installed"
				if res.Updated {
					verb = "Updated"
				}
				fmt.Fprintf(out, "%s %s %s as %s.\n", verb, res.Row.Name, res.Row.Version, res.Row.AdapterKey)
				if res.Secret != "" {
					fmt.Fprintf(out, "\nSigning secret (shown once):\n\n  %s\n\nSet it on the adapter process as REPUTECAST_ADAPTER_SECRET.\n", res.Secret)
				}
				if res.Row.EndpointUrl == "" {
					fmt.Fprintf(out, "Then tell ReputeCast where it runs: reputecast adapter set-url %s <url>\n", strings.TrimPrefix(res.Row.AdapterKey, "community:"))
				} else {
					fmt.Fprintf(out, "Then check it: reputecast adapter check %s\n", strings.TrimPrefix(res.Row.AdapterKey, "community:"))
				}
				return nil
			})(cmd, nil)
		},
	}
	install.Flags().StringVar(&endpoint, "url", "", "where the adapter process listens, like http://example-esp:4600")
	setURL := &cobra.Command{
		Use:     "set-url <id> <url>",
		Short:   "Set where an adapter's process listens",
		Example: "  reputecast adapter set-url example-esp http://example-esp:4600",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(func(ctx context.Context, cmd *cobra.Command, s *remote.Service) error {
				r, err := find(ctx, s, args[0])
				if err != nil {
					return err
				}
				r, err = s.SetEndpoint(ctx, nil, r, args[1], remote.CLIActor)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s now calls %s.\n", r.AdapterKey, r.EndpointUrl)
				return nil
			})(cmd, nil)
		},
	}
	secret := &cobra.Command{
		Use:   "secret <id>",
		Short: "Issue a new signing secret for an adapter and print it once",
		Long: "The previous secret keeps working for 24 hours, so the adapter process can be updated without\n" +
			"downtime (it accepts several comma-separated secrets in REPUTECAST_ADAPTER_SECRET).",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(func(ctx context.Context, cmd *cobra.Command, s *remote.Service) error {
				r, err := find(ctx, s, args[0])
				if err != nil {
					return err
				}
				out, err := s.IssueSecret(ctx, nil, r, remote.CLIActor)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "New signing secret for %s (shown once):\n\n  %s\n\nSet it on the adapter process as REPUTECAST_ADAPTER_SECRET. The previous secret works until %s.\n",
					r.AdapterKey, out.Secret, out.PreviousSecretValidUntil.Local().Format("2006-01-02 15:04"))
				return nil
			})(cmd, nil)
		},
	}
	check := &cobra.Command{
		Use:   "check <id>",
		Short: "Call an adapter's health operation and record the result",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(func(ctx context.Context, cmd *cobra.Command, s *remote.Service) error {
				r, err := find(ctx, s, args[0])
				if err != nil {
					return err
				}
				r, err = s.Check(ctx, nil, r, remote.CLIActor)
				if err != nil {
					return err
				}
				if r.CheckOk == nil || !*r.CheckOk {
					return fmt.Errorf("%s: %s", r.AdapterKey, r.CheckDetail)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", r.AdapterKey, r.CheckDetail)
				return nil
			})(cmd, nil)
		},
	}
	remove := &cobra.Command{
		Use:   "remove <id>",
		Short: "Uninstall an adapter no connected provider uses",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(func(ctx context.Context, cmd *cobra.Command, s *remote.Service) error {
				r, err := find(ctx, s, args[0])
				if err != nil {
					return err
				}
				if err := s.Remove(ctx, r, remote.CLIActor); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Removed %s.\n", r.AdapterKey)
				return nil
			})(cmd, nil)
		},
	}
	cmd.AddCommand(list, install, setURL, secret, check, remove)
	return cmd
}

// adapterService opens what the adapter commands need.
func adapterService(ctx context.Context, cfg *config.Config, log *slog.Logger) (*remote.Service, func(), error) {
	pool, err := db.Open(ctx, cfg.Database.URL, 4)
	if err != nil {
		return nil, nil, err
	}
	box, err := app.SecretBox(cfg, log)
	if err != nil {
		pool.Close()
		return nil, nil, err
	}
	q := store.New(pool)
	allow := func() bool { return cfg.Adapters.AllowPrivateNetworks }
	s := &remote.Service{Pool: pool, Q: q, Audit: audit.New(q, log), Box: box, AllowPrivate: allow,
		Runtime: &remote.Runtime{Pool: pool, Box: box, Log: log, AllowPrivate: allow, PublicURL: func() string { return cfg.HTTP.PublicURL }}}
	return s, pool.Close, nil
}

// readManifest reads a manifest from an https URL on a public address or
// from a file.
func readManifest(ctx context.Context, src string) ([]byte, string, error) {
	if !strings.Contains(src, "://") {
		f, err := os.Open(src) //nolint:gosec // the operator names the file on the command line
		if err != nil {
			return nil, "", err
		}
		defer func() { _ = f.Close() }()
		raw, err := io.ReadAll(io.LimitReader(f, maxManifest+1))
		if err != nil {
			return nil, "", err
		}
		if len(raw) > maxManifest {
			return nil, "", errors.New("the manifest is larger than 256 KiB")
		}
		return raw, "", nil
	}
	u, err := url.Parse(src)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return nil, "", errors.New("give an https:// URL or a file path for the manifest")
	}
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := netguard.Policy{}.Client(15 * time.Second).Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("couldn't download the manifest: %s", netguard.Describe(cctx, err))
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("the manifest URL answered %d (redirects are not followed: give the final URL)", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxManifest+1))
	if err != nil {
		return nil, "", err
	}
	if len(raw) > maxManifest {
		return nil, "", errors.New("the manifest is larger than 256 KiB")
	}
	return raw, u.String(), nil
}
