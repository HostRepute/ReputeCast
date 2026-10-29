package readiness_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/readiness"
)

// Send Readiness reads a provider's sending IPs as the IP check keeps them
// (review GS-10): a dedicated IP Postal began sending from since
// Deliverability's last blocklist run gets its own row, unknown until the
// checks reach it, while the IP they cover keeps its verdict.
func TestReviewReadsRefreshedIPs(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	if _, err := env.App.Pool.Exec(ctx, `UPDATE providers SET ips = '{198.51.100.40,203.0.113.25}' WHERE adapter = 'postal'
		AND workspace_id = (SELECT id FROM workspaces WHERE slug = 'acme-newsletter')`); err != nil {
		t.Fatal(err)
	}
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	run := runChecks(t, env, c, campaignID(t, c, "Weekly Digest #142"))
	ch, ok := checkOf(run, "reputation.ip_blocklists.198.51.100.40")
	if !ok || ch.Status != readiness.Unknown || !strings.Contains(ch.Title, "isn't checked on IP blocklists yet") {
		t.Fatalf("the new IP: %+v (found %v)", ch, ok)
	}
	if ch, ok := checkOf(run, "reputation.ip_blocklists.203.0.113.25"); !ok || ch.Status == readiness.Unknown {
		t.Fatalf("the checked IP: %+v (found %v)", ch, ok)
	}
}
