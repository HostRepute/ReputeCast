package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/hostrepute/reputecast/internal/app"
	"github.com/hostrepute/reputecast/internal/providers/simulator"
	"github.com/hostrepute/reputecast/internal/sending/engine"
)

func newDemoCmd(o *rootOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "demo",
		Short: "Development helpers for the Acme demo (REPUTECAST_ENV=development only)",
	}
	var (
		scenario, workspace string
		elapsed, abWindow   time.Duration
		reset               bool
	)
	send := &cobra.Command{
		Use:   `send "<campaign>"`,
		Short: "Send a demo campaign through the simulated providers so the live monitor moves",
		Long: "Starts the campaign's send now as Sofia Rossi, with Marcus Chen's approval recorded, through the demo\n" +
			"transport: nothing leaves the machine; delivery, bounces, complaints, opens and clicks are simulated at the\n" +
			"boards' rates. The running server's worker sends it in real time.\n\n" +
			"Scenarios:\n" +
			"  normal              the LiveMonitor board\n" +
			"  outlook-complaints  lapsed Outlook readers complain until the Outlook share auto-pauses (LiveMonitorPaused)\n" +
			"  postal-outage       Postal refuses connections 14 minutes in; its share is held (LiveMonitorOutage)\n\n" +
			"--elapsed starts the send that long ago and sends that part at once with back-dated times (and the\n" +
			"minute or two that takes), so the monitor opens mid-send: --elapsed 20m shows the boards' 09:21 moment.\n" +
			"--reset clears an earlier send of the campaign, with what it changed outside it (provider statistics,\n" +
			"suppressions from its bounces). Only workspaces whose providers are all demo providers can send.",
		Example: `  reputecast demo send "Weekly Digest #142" --elapsed 20m
  reputecast demo send "Weekly Digest #142" --scenario outlook-complaints --elapsed 20m --reset
  reputecast demo send "Weekly Digest #142" --scenario postal-outage --elapsed 17m --reset`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, log, err := o.load(cmd)
			if err != nil {
				return err
			}
			if !cfg.IsDevelopment() {
				return usageError{fmt.Errorf("refusing to run a demo send outside development; set REPUTECAST_ENV=development")}
			}
			a, err := app.New(cmd.Context(), cfg, log)
			if err != nil {
				return err
			}
			defer a.Close()
			res, err := a.SendingEngine().DemoSend(cmd.Context(), engine.DemoSendOptions{
				Workspace: workspace, Campaign: args[0], Scenario: scenario, Elapsed: elapsed, ABWindow: abWindow, Reset: reset,
			})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "\n  Sending %q to %s recipients through the demo transport (scenario %s).\n", res.Name, count(res.Recipients), scenario)
			if res.Sent > 0 {
				fmt.Fprintf(out, "  Fast-forwarded %s: %s already sent.\n", elapsed, count(res.Sent))
			}
			fmt.Fprintf(out, "  Watch it at %s%s (the server's worker keeps sending).\n\n", a.URLs.Base(), res.Path)
			return nil
		},
	}
	send.Flags().StringVar(&scenario, "scenario", simulator.ScenarioNormal, "normal, outlook-complaints or postal-outage")
	send.Flags().StringVar(&workspace, "workspace", "acme-newsletter", "workspace slug")
	send.Flags().DurationVar(&elapsed, "elapsed", 0, "start the send this long ago and fast-forward to now (e.g. 21m)")
	send.Flags().DurationVar(&abWindow, "ab-window", engine.DemoABWindow, "how long the A/B test runs before the winner goes to the rest")
	send.Flags().BoolVar(&reset, "reset", false, "clear an earlier send of the campaign first")
	cmd.AddCommand(send)
	return cmd
}

func count(n int64) string {
	s := fmt.Sprint(n)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}
