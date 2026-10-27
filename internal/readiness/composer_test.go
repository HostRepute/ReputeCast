package readiness_test

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/readiness"
)

// The composer's checks show Send Readiness's workspace rules and sender
// reputation from the latest run (Composer board): each group's warnings
// and blocks, and when the blocklists were checked.
func TestComposerFindings(t *testing.T) {
	f := digest()
	f.Content.Report.LinkList = links("https://acme.io/roadmap?utm_source=newsletter&utm_medium=email")
	f.Rules = []readiness.RuleFacts{utmRule()}
	run := readiness.ReadinessRun{ID: "run-1", RanAt: now, ReadinessResult: stored(t, readiness.Assemble(f))}
	got := readiness.ComposerFindings(&run)
	if got.RanAt == nil || got.CheckedAt == nil || got.Rules == nil || got.Rules.Warnings != 1 || len(got.Rules.Checks) != 1 || got.Rules.Checks[0].Key != "rules.r-utm" {
		t.Fatalf("rules: %+v", got.Rules)
	}
	if got.Reputation == nil || got.Reputation.Warnings != 1 || got.Reputation.Checks[0].Key != "reputation.ip_blocklists.203.0.113.25" {
		t.Fatalf("reputation: %+v", got.Reputation)
	}
	// Without rules there is no rules row; never ran, nothing.
	f.Rules = nil
	run.ReadinessResult = readiness.Assemble(f)
	if got := readiness.ComposerFindings(&run); got.Rules != nil {
		t.Errorf("no rules: %+v", got.Rules)
	}
	if got := readiness.ComposerFindings(nil); got.RanAt != nil || got.Reputation != nil {
		t.Errorf("never ran: %+v", got)
	}
}

// The endpoint reads the latest run for anyone who can see the campaign,
// in their language.
func TestComposerReadinessAPI(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	aisha := env.Client()
	aisha.SignInDemo("aisha@acme.io")
	digest := campaignID(t, aisha, "Weekly Digest #142")
	var got readiness.ComposerReadiness
	aisha.Do("GET", ws+"/campaigns/"+digest+"/readiness/composer", nil).MustStatus(t, 200).JSON(t, &got)
	if got.RanAt == nil || got.Rules == nil || got.Rules.Warnings == 0 || got.Reputation == nil || got.Reputation.Warnings == 0 {
		t.Fatalf("Weekly Digest #142: %+v", got)
	}
	for _, c := range append(got.Rules.Checks, got.Reputation.Checks...) {
		if c.Title == "" || (c.Status != readiness.Warn && c.Status != readiness.Block) {
			t.Errorf("check %+v", c)
		}
	}
	aisha.Do("GET", ws+"/campaigns/00000000-0000-7000-8000-000000000000/readiness/composer", nil).MustStatus(t, 404)
}
