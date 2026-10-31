package readiness_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/readiness/readinesstest"
)

// SR-11 (C2): a real run leaves the unengaged out by default and offers to
// include them; including them (audited) puts them back in the count and
// the estimates, and excluding them again takes them out.
func TestUnengagedLeftOutUntilIncluded(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	id := campaignID(t, c, "Weekly Digest #142")
	fixed := func(kind string) int {
		t.Helper()
		var n int
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'campaign.update' AND target_id = $1 AND metadata->>'fix' = $2`, id, kind).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	recipients := func(run readiness.ReadinessRun) int64 {
		var n int64
		for _, s := range run.Estimates.Route.Shares {
			n += s.Recipients
		}
		return n
	}

	run := runChecks(t, env, c, id)
	a := run.Estimates.Audience
	last := a.Rows[len(a.Rows)-1]
	if !a.UnengagedExcluded || a.ExcludedUnengaged == 0 || last.Action != readiness.FixIncludeUnengaged || last.Count != a.ExcludedUnengaged || last.Kind != "minus" {
		t.Fatalf("left out by default: %+v", a)
	}
	if ch, _ := checkOf(run, "audience.unengaged"); ch.Status != readiness.Pass || !strings.HasSuffix(ch.Title, "unengaged excluded by your sunset policy") {
		t.Fatalf("the check: %+v", ch)
	}
	if recipients(run) != a.Final {
		t.Fatalf("the route plans %d of %d", recipients(run), a.Final)
	}

	// Include them.
	fix := func(action string) {
		t.Helper()
		readinesstest.Fix(t, env, c, ws+"/campaigns/"+id+"/readiness/fixes", map[string]any{"fixes": []map[string]any{{"action": action}}}).MustStatus(t, 200)
	}
	fix(readiness.FixIncludeUnengaged)
	inc := readinesstest.Latest(t, c, "acme-newsletter", id)
	ia := inc.Estimates.Audience
	if ia.UnengagedExcluded || ia.ExcludedUnengaged != 0 || ia.Final != a.Final+a.ExcludedUnengaged || recipients(inc) != ia.Final {
		t.Fatalf("included: %+v (before %+v)", ia, a)
	}
	for _, r := range ia.Rows {
		if r.Action != "" {
			t.Errorf("a line still offers %s", r.Action)
		}
	}
	ch, _ := checkOf(inc, "audience.unengaged")
	if ch.Status != readiness.Warn || len(ch.Fixes) == 0 || ch.Fixes[0].Action != readiness.FixExcludeSunset {
		t.Fatalf("included, the check warns: %+v", ch)
	}
	if fixed(readiness.FixIncludeUnengaged) != 1 {
		t.Error("including them isn't audited")
	}

	// Exclude them again.
	fix(readiness.FixExcludeSunset)
	back := readinesstest.Latest(t, c, "acme-newsletter", id)
	if !back.Estimates.Audience.UnengagedExcluded || back.Estimates.Audience.Final != a.Final {
		t.Fatalf("excluded again: %+v", back.Estimates.Audience)
	}
	if fixed(readiness.FixExcludeSunset) != 1 {
		t.Error("excluding them isn't audited")
	}
}
