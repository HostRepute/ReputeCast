package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/hostrepute/reputecast/internal/config"
)

// newHealthcheckCmd checks the process running next to it, for container
// health checks (review 2026-11 OP-2). It asks the HTTP listener's
// /readyz, or the metrics listener's when the process has no HTTP listener
// (a worker), and exits 1 when the answer isn't 200.
func newHealthcheckCmd(_ *rootOpts) *cobra.Command {
	var live bool
	var url string
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "healthcheck",
		Short: "Check that the ReputeCast process on this host is ready (for container health checks)",
		Long: "Ask the local process whether it is ready: its database answers, the schema has every\n" +
			"migration and, in a worker, the workers run. Exits 0 when ready, 1 otherwise.\n" +
			"It reads REPUTECAST_HTTP__ADDR and REPUTECAST_METRICS__ADDR like serve does.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := "/readyz"
			if live {
				path = "/healthz"
			}
			urls := []string{url}
			if url == "" {
				urls = localURLs(envOr("REPUTECAST_HTTP__ADDR", ":8080"), envOr("REPUTECAST_METRICS__ADDR", config.MetricsDefaultAddr), path)
			}
			client := &http.Client{Timeout: timeout}
			var last error
			for _, u := range urls {
				status, body, err := probe(cmd.Context(), client, u)
				if err != nil {
					last = err
					continue // nothing listens there: try the next listener
				}
				if status != http.StatusOK {
					return fmt.Errorf("not ready: %s answered %d %s", u, status, strings.TrimSpace(body))
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", u, strings.TrimSpace(body))
				return err
			}
			return fmt.Errorf("no ReputeCast process answered: %w", last)
		},
	}
	cmd.Flags().BoolVar(&live, "live", false, "check liveness (/healthz) instead of readiness")
	cmd.Flags().StringVar(&url, "url", "", "the URL to check (default: the local HTTP listener, then the metrics listener)")
	cmd.Flags().DurationVar(&timeout, "timeout", 3*time.Second, "how long to wait for an answer")
	return cmd
}

// localURLs turns listen addresses (":8080", "0.0.0.0:8080",
// "[::]:9464") into loopback URLs for path.
func localURLs(httpAddr, metricsAddr, path string) []string {
	var out []string
	for _, a := range []string{httpAddr, metricsAddr} {
		if a == "" {
			continue
		}
		host, port, err := net.SplitHostPort(a)
		if err != nil {
			continue
		}
		switch host {
		case "", "0.0.0.0", "::", "[::]":
			host = "127.0.0.1"
		}
		out = append(out, "http://"+net.JoinHostPort(host, port)+path)
	}
	return out
}

func probe(ctx context.Context, c *http.Client, url string) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, "", err
	}
	resp, err := c.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 512))
	if err != nil && !errors.Is(err, io.EOF) {
		return 0, "", err
	}
	return resp.StatusCode, string(b), nil
}
