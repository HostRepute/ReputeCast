package approvals

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/readiness"
)

func TestParsePolicyDefaultsAndRoundTrip(t *testing.T) {
	p := ParsePolicy(nil)
	if !p.AudienceMin.Enabled || p.AudienceMin.N != 10000 || !p.WarmingProvider || !p.AuthorLacksSend || p.ReadyWithWarnings || p.ApprovalsRequired != 1 || !p.EmergencyBypass {
		t.Fatalf("default policy: %+v", p)
	}
	if got := strings.Join(p.RuleLabels(), " · "); got != "audience ≥ 10,000 · warming-up provider · author lacks send" {
		t.Fatalf("labels: %s", got)
	}

	// Saving keeps the keys other screens own and folds new providers into
	// the warming rule.
	prev := json.RawMessage(`{"approver_roles":["owner","admin"],"rules":{"new_provider":{"enabled":true,"days":7}},"approvals_required":1}`)
	p.ReadyWithWarnings, p.ApprovalsRequired, p.EmergencyBypass, p.NewProviderDays = true, 2, false, 7
	raw := p.toDB(prev)
	back := ParsePolicy(raw)
	if !back.ReadyWithWarnings || back.ApprovalsRequired != 2 || back.EmergencyBypass || back.NewProviderDays != 7 || !back.WarmingProvider {
		t.Fatalf("round trip: %+v from %s", back, raw)
	}
	if !strings.Contains(string(raw), `"approver_roles":["owner","admin"]`) {
		t.Fatalf("approver roles are kept: %s", raw)
	}

	// Out-of-range values are clamped.
	if q := ParsePolicy(json.RawMessage(`{"approvals_required":5,"rules":{"audience_min":{"enabled":true,"n":0}}}`)); q.ApprovalsRequired != 2 || q.AudienceMin.N != 10000 {
		t.Fatalf("clamped: %+v", q)
	}
}

func TestEvaluateMatchesTheSampleRequest(t *testing.T) {
	now := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	f := RuleFacts{Recipients: 47912, Verdict: readiness.VerdictWarnings, Warnings: 2, AuthorName: "Daniel Park", AuthorRole: "Content Writer", Now: now,
		Shares: []RuleShare{
			{Short: "SES", Recipients: 43121, ConnectedAt: now.AddDate(0, -7, 0)},
			{Short: "Postal", Recipients: 4791, WarmingUp: true, WarmupDay: 9, WarmupDays: 21, ConnectedAt: now.AddDate(0, 0, -8)},
		}}
	rules := ParsePolicy(nil).Evaluate(f)
	want := []string{
		"Audience of 47,912 is at least 10,000",
		"Route includes Postal, which is warming up (day 9 of 21)",
		"Daniel Park doesn't hold campaigns:send · Content Writer",
	}
	if len(rules) != len(want) {
		t.Fatalf("rules: %+v", rules)
	}
	for i, r := range rules {
		if r.Label != want[i] {
			t.Errorf("rule %d: %q, want %q", i, r.Label, want[i])
		}
	}
	if line := rulesLine(rules); line != "rules: audience ≥ 10,000 · warming-up provider · author lacks send" {
		t.Fatalf("inbox line: %s", line)
	}

	// A sender with campaigns:send, a small audience and a healthy route: no rule.
	f.Recipients, f.AuthorCanSend = 2113, true
	f.Shares = f.Shares[:1]
	if rules := ParsePolicy(nil).Evaluate(f); len(rules) != 0 {
		t.Fatalf("nothing matches: %+v", rules)
	}

	// A provider connected yesterday counts as new; the optional rules match
	// when they're on.
	p := ParsePolicy(nil)
	p.ReadyWithWarnings, p.ApprovalLists = true, true
	f.Shares = append(f.Shares, RuleShare{Short: "Postmark", Recipients: 100, ConnectedAt: now.Add(-30 * time.Hour)})
	f.ApprovalLists = []string{"Enterprise customers"}
	rules = p.Evaluate(f)
	labels := []string{}
	for _, r := range rules {
		labels = append(labels, r.Label)
	}
	got := strings.Join(labels, " | ")
	if got != "Route includes Postmark, connected yesterday | Send Readiness is Ready with warnings (2 warnings) | The audience includes Enterprise customers, which requires approval" {
		t.Fatalf("optional rules: %s", got)
	}
}

func TestDiff(t *testing.T) {
	loc := time.UTC
	aud := json.RawMessage(`{"lists":["a","b"]}`)
	sched := json.RawMessage(`{"mode":"at","at":"2026-11-04T14:00:00Z","timezone":"America/New_York"}`)
	snap := Snapshot{Version: 9, ContentHash: "h1", Subject: "November Product Update", Preheader: "Faster exports", Audience: aud, AudienceLabels: []string{"Weekly Digest", "Product Updates"},
		AudienceCount: 47912, RouteKey: "ses:90,postal:10", SplitLabel: "SES 90 / Postal 10", Schedule: sched, Links: []string{}, Sections: []string{}}
	cur := Current{Version: 9, ContentHash: "h1", Subject: snap.Subject, Preheader: snap.Preheader, Audience: json.RawMessage(`{"lists":["b","a"]}`), AudienceCount: 47912,
		RouteKey: snap.RouteKey, SplitLabel: snap.SplitLabel, Schedule: json.RawMessage(`{"mode":"at","at":"2026-11-04T09:00:00-05:00","timezone":"America/New_York"}`)}
	if d := Diff(snap, cur, loc); len(d) != 0 {
		t.Fatalf("the same campaign (lists in another order, the same instant): %+v", d)
	}

	// The audience may move 10% before the approval stops covering it.
	cur.AudienceCount = 52000
	if d := Diff(snap, cur, loc); len(d) != 0 {
		t.Fatalf("+8.5%% is covered: %+v", d)
	}
	cur.AudienceCount = 54620
	d := Diff(snap, cur, loc)
	if len(d) != 1 || d[0].Field != "audience_count" || d[0].Before != "47,912" || d[0].After != "54,620 (+14%)" {
		t.Fatalf("+14%%: %+v", d)
	}
	if r := InvalidationReason(d); r != "audience moved +14% after approval" {
		t.Fatalf("reason: %s", r)
	}

	// Drift is measured against the baseline counted at request time.
	snap.AudienceBaseline = 42368
	cur.AudienceCount = 42368
	if d := Diff(snap, cur, loc); len(d) != 0 {
		t.Fatalf("no drift against the baseline: %+v", d)
	}
	cur.AudienceCount = 37000
	if d := Diff(snap, cur, loc); len(d) != 1 || d[0].After != "42,544 (−13%)" {
		t.Fatalf("drift against the baseline: %+v", d)
	}
	cur.AudienceCount = 42368

	// Subject and schedule changes are listed with before and after.
	cur.Subject = "November Product Update: faster exports"
	cur.Schedule = json.RawMessage(`{"mode":"at","at":"2026-11-05T14:00:00Z","timezone":"America/New_York"}`)
	d = Diff(snap, cur, loc)
	if len(d) != 2 || d[0].Field != "subject" || d[1].Field != "schedule" || d[1].Before != "Nov 4, 09:00" || d[1].After != "Nov 5, 09:00" {
		t.Fatalf("subject and schedule: %+v", d)
	}
	if r := InvalidationReason(d); r != "subject and schedule changed" {
		t.Fatalf("reason: %s", r)
	}

	// Another list and another route.
	cur.Subject, cur.Schedule = snap.Subject, snap.Schedule
	cur.Audience, cur.AudienceNames = json.RawMessage(`{"lists":["a","b","c"]}`), []string{"Weekly Digest", "Product Updates", "Beta Program"}
	cur.RouteKey, cur.SplitLabel = "ses:100", "SES 100"
	d = Diff(snap, cur, loc)
	if len(d) != 2 || d[0].Field != "audience" || d[0].After != "Weekly Digest, Product Updates, Beta Program" || d[1].Field != "route" || d[1].After != "SES 100" {
		t.Fatalf("audience and route: %+v", d)
	}
}

func TestDiffNamesTheChangedLink(t *testing.T) {
	src := `<mjml><mj-body><mj-section><mj-column><mj-button href="https://bit.ly/acme-nov">Read</mj-button><mj-text><a href="https://acme.io/docs">Docs</a></mj-text></mj-column></mj-section></mj-body></mjml>`
	snap := Snapshot{Version: 9, ContentHash: "h1", Subject: "s", Links: Links(src, ""), Sections: Sections(src), Schedule: json.RawMessage(`{"mode":"now"}`)}
	if len(snap.Links) != 2 || snap.Links[0] != "https://bit.ly/acme-nov" {
		t.Fatalf("links: %v", snap.Links)
	}
	next := strings.Replace(src, "https://bit.ly/acme-nov", "https://acme.io/blog/november", 1)
	d := Diff(snap, Current{Version: 10, ContentHash: "h2", Subject: "s", MJML: next, Schedule: json.RawMessage(`{"mode":"now"}`)}, time.UTC)
	if len(d) != 1 || d[0].Field != "link" || d[0].Label != "Link 1" || d[0].After != "https://acme.io/blog/november" {
		t.Fatalf("link: %+v", d)
	}
	if r := InvalidationReason(d); r != "a link changed" {
		t.Fatalf("reason: %s", r)
	}

	// Other content edits name the version.
	d = Diff(snap, Current{Version: 10, ContentHash: "h3", Subject: "s", MJML: src, Schedule: json.RawMessage(`{"mode":"now"}`)}, time.UTC)
	if len(d) != 1 || d[0].Field != "content" || d[0].Before != "v9" || !strings.HasPrefix(d[0].After, "v10") {
		t.Fatalf("content: %+v", d)
	}
}

func TestScheduleText(t *testing.T) {
	ny := json.RawMessage(`{"mode":"at","at":"2026-11-04T14:00:00Z","timezone":"America/New_York"}`)
	if got := ScheduleText(ny, time.UTC); got != "Nov 4, 09:00" {
		t.Fatalf("at: %s", got)
	}
	if got := ScheduleText(json.RawMessage(`{"mode":"now"}`), time.UTC); got != "Send now" {
		t.Fatalf("now: %s", got)
	}
	best := json.RawMessage(`{"mode":"best_time","at":"2026-11-04T14:00:00Z","timezone":"America/New_York"}`)
	if got := ScheduleText(best, time.UTC); got != "Best time within 24 h of Nov 4, 09:00" {
		t.Fatalf("best time: %s", got)
	}
}
