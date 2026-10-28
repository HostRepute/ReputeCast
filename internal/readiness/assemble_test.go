package readiness_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/readiness"
)

func f64(v float64) *float64 { return &v }

var now = time.Date(2026, 10, 13, 13, 40, 0, 0, time.UTC)

func ses(n int64, pct float64) readiness.ShareFacts {
	return readiness.ShareFacts{ProviderID: "p-ses", Name: "Amazon SES", Short: "SES", Adapter: "ses", Label: "us-east-1", Recipients: n, Percent: pct,
		Status: state.Healthy, Available: true, Rate: 14, Pricing: state.Pricing{Model: "metered", Per1000: 0.10}, Bounce: f64(0.6), Complaint: f64(0.04),
		Limits:  state.Limits{Bounce: state.Limit{Limit: 5, Review: 5, Pause: 10, Label: "SES"}, Complaint: state.Limit{Limit: 0.1, Review: 0.1, Pause: 0.5, Label: "SES"}},
		Account: "Production · no enforcement", Quota: &state.Quota{Kind: "rolling_24h", Max: 50000, Used: 2140}, BulkAllowed: true, Terms: "yes, with production access",
		Tracking: state.Tracking{Scope: "Configuration set rc-broadcasts", Mode: "reputecast"}, ConnectedAt: now.AddDate(0, -7, 0)}
}

func postal(n int64, pct float64) readiness.ShareFacts {
	return readiness.ShareFacts{ProviderID: "p-postal", Name: "Postal", Short: "Postal", Adapter: "postal", Label: "self-hosted · mta.acme.io",
		Recipients: n, Percent: pct, Status: state.WarmingUp, Available: true, WarmupDay: 9, WarmupDays: 21, WarmupCap: 6000, WarmupToday: true, Rate: 2, RateRamp: true,
		Pricing: state.Pricing{Model: "self_hosted", Monthly: 24}, Bounce: f64(0.9), Complaint: f64(0.05),
		Limits: state.Limits{Bounce: state.Limit{Limit: 2, Label: "warm-up guard"}, Complaint: state.Limit{Limit: 0.1}}, Account: "Postal 3.3.4 · reachable",
		BulkAllowed: true, Terms: "self-hosted, your own policy", Tracking: state.Tracking{Scope: "mail server mta.acme.io", Mode: "reputecast"},
		IPs: []string{"203.0.113.25"}, ConnectedAt: now.AddDate(0, 0, -8)}
}

func links(urls ...string) []contentcheck.ContentCheckLink {
	out := []contentcheck.ContentCheckLink{}
	for _, u := range urls {
		host := strings.SplitN(strings.TrimPrefix(u, "https://"), "/", 2)[0]
		out = append(out, contentcheck.ContentCheckLink{URL: u, Host: host, Text: host, Kind: "link", Path: []int{}, Probed: true, Status: 200, Millis: 140})
	}
	return out
}

// digest is the Weekly Digest #142 of the SendReadiness board: Ready with
// two warnings (a low-impact IP listing and 412 unengaged contacts).
func digest() readiness.Facts {
	checked := now.Add(-time.Minute)
	return readiness.Facts{
		Now: now, Location: state.Location("America/New_York"), Workspace: "acme-newsletter", MailboxRates: readiness.DefaultMailboxRates,
		Campaign: readiness.CampaignFacts{ID: "c1", Name: "Weekly Digest #142", Subject: "Weekly Digest #142: feature flags are GA", Version: 14, CompiledAt: &checked,
			SenderName: "Acme Newsletter", SenderEmail: "digest@news.acme.io", SenderDomain: "news.acme.io", StartAt: now.Add(4 * time.Hour), ScheduleMode: "at"},
		Audience: readiness.AudienceFacts{BaseLabel: "Weekly Digest list", GroupLabel: "Weekly Digest", Lists: []string{"Weekly Digest"}, Base: 31988, Suppressed: 331,
			Role: 27, Duplicates: 26, Final: 31604, Unengaged: 412, UnengagedDays: 180,
			Mailboxes: map[string]int64{"gmail": 14538, "outlook": 6637, "yahoo": 2844, "icloud": 2528, "other": 5057}},
		Route: readiness.RouteFacts{Stream: "broadcast", Policy: "pause_and_ask", Shares: []readiness.ShareFacts{ses(28444, 90), postal(3160, 10)}},
		Reputation: readiness.ReputationFacts{Status: readiness.Warn, Via: "hostrepute", Sources: 84, CheckedAt: &checked,
			DomainSources: []string{"Spamhaus DBL", "SURBL", "URIBL"}, IPSources: []string{"Spamhaus ZEN", "Barracuda", "UCEPROTECT Level 3"},
			Listings: []readiness.ListingFacts{{Asset: "203.0.113.25", Type: "ip", Source: "UCEPROTECT Level 3", Impact: "low", Detail: "Network-range listing (/16), low impact",
				DelistURL: "https://www.uceprotect.net/en/rblcheck.php"}}},
		Auth: readiness.AuthFacts{Domain: "news.acme.io", Checked: true, CheckedAt: &checked, SPFRecord: "v=spf1 include:amazonses.com include:spf.mta.acme.io -all",
			SPF: []readiness.ProviderAuthFacts{{Short: "SES", OK: true}, {Short: "Postal", OK: true}},
			DKIM: []readiness.ProviderAuthFacts{{Short: "SES", OK: true, Detail: "3 Easy DKIM CNAMEs verified", KeyBits: 2048},
				{Short: "Postal", OK: true, Detail: "selector postal-2026 verified", KeyBits: 2048}},
			MailFrom:    []readiness.ProviderAuthFacts{{Short: "SES", OK: true, Domain: "bounce.news.acme.io"}, {Short: "Postal", OK: true, Domain: "rp.news.acme.io"}},
			DMARCStatus: "pass", DMARCPolicy: "quarantine", DMARCRecord: "v=DMARC1; p=quarantine; adkim=s", DMARCAligned: f64(100), BIMIStatus: "missing"},
		Content: readiness.ContentFacts{Checked: true, Address: "Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105", InlineCSSBytes: 11 * 1024,
			URIBlocklists: readiness.URIBlocklists, OwnDomain: "acme.io",
			Report: contentcheck.ContentCheckReport{Verdict: readiness.VerdictReady, SizeBytes: 64 * 1024, ClipBytes: contentcheck.ClipBytes, Images: 7, Links: 3,
				SubjectHeuristics: []string{}, Issues: []contentcheck.ContentCheckIssue{}, Sections: []contentcheck.ContentCheckSection{}, PlainText: strings.Repeat("x", 9*1024),
				CheckedAt: checked, ImageShare: 0.38, Words: 1140, LinkList: links("https://acme.io/blog", "https://docs.acme.io/flags", "https://github.com/acme/flags")}},
		Compliance: readiness.ComplianceFacts{UnsubscribeURL: "https://news.acme.io/u/{token}", HTTPS: true, ComplaintRate: f64(0.04)},
	}
}

func byKey(res readiness.ReadinessResult) map[string]readiness.ReadinessCheck {
	out := map[string]readiness.ReadinessCheck{}
	for _, c := range res.Checks {
		out[c.Key] = c
	}
	return out
}

func statuses(res readiness.ReadinessResult, status string) []string {
	out := []string{}
	for _, c := range res.Checks {
		if c.Status == status {
			out = append(out, c.Key)
		}
	}
	return out
}

func hasFix(c readiness.ReadinessCheck, action string) bool {
	for _, f := range c.Fixes {
		if f.Action == action {
			return true
		}
	}
	return false
}

func TestAssembleReadyWithWarnings(t *testing.T) {
	res := readiness.Assemble(digest())
	if res.Verdict != readiness.VerdictWarnings || res.Counts.Warnings != 2 || res.Counts.Blocks != 0 || res.Incomplete {
		t.Fatalf("verdict %s %+v, warnings %v, blocks %v", res.Verdict, res.Counts, statuses(res, readiness.Warn), statuses(res, readiness.Block))
	}
	if res.Counts.Passed+res.Counts.Warnings != res.Counts.Total || len(res.Groups) != 7 {
		t.Fatalf("counts %+v, %d groups", res.Counts, len(res.Groups))
	}
	checks := byKey(res)
	if c := checks["audience.unengaged"]; c.Status != readiness.Warn || !hasFix(c, "exclude_sunset") {
		t.Fatalf("unengaged: %+v", c)
	}
	if c := checks["reputation.ip_blocklists.203.0.113.25"]; c.Status != readiness.Warn || !strings.Contains(c.Detail+c.Title, "UCEPROTECT") {
		t.Fatalf("low-impact IP listing: %+v (warnings %v)", c, statuses(res, readiness.Warn))
	}
	for _, c := range res.Checks {
		if c.Title == "" || c.FlatTitle == "" || c.Evidence == "" || c.Fixes == nil {
			t.Errorf("every check explains itself: %+v", c)
		}
	}

	// The estimates of the board: 31,604 recipients, SES $2.84 in 34 minutes,
	// Postal at its ramp, every mailbox provider throttled.
	e := res.Estimates
	if e.Audience.Final != 31604 || len(e.Audience.Rows) != 4 || e.Audience.Rows[0].Count != 31988 {
		t.Fatalf("audience: %+v", e.Audience)
	}
	if e.Route.SplitLabel != "Broadcasts 90 / 10" || e.Route.TotalCost != 2.84 || len(e.Route.Shares) != 2 {
		t.Fatalf("route: %+v", e.Route)
	}
	if s := e.Route.Shares[0]; s.Cost != 2.84 || s.Minutes != 34 || s.PriceLabel != "$0.10 per 1,000" {
		t.Fatalf("SES: %+v", s)
	}
	if s := e.Route.Shares[1]; s.Cost != 0 || s.Minutes != 26 || s.PriceLabel != "self-hosted" || s.RateLabel != "2 /s ramp" {
		t.Fatalf("Postal: %+v", s)
	}
	if len(e.Throttling) != 5 || e.Throttling[0].Name != "Gmail" || e.Throttling[0].Recipients != 14538 {
		t.Fatalf("throttling: %+v", e.Throttling)
	}
	if e.Route.TotalMinutes < 34 {
		t.Fatalf("the send takes as long as its slowest lane: %d", e.Route.TotalMinutes)
	}

	// Results survive storage.
	checksJSON, estJSON, err := readiness.EncodeResult(res)
	if err != nil || !json.Valid(checksJSON) || !json.Valid(estJSON) {
		t.Fatalf("encode: %v", err)
	}
}

func TestAssembleBlocks(t *testing.T) {
	f := digest()
	// The footer was removed, Postal is down and the domain is on Spamhaus DBL.
	f.Content.Report.Issues = append(f.Content.Report.Issues, contentcheck.ContentCheckIssue{ID: "compliance.footer", Severity: "block", Group: "Compliance", Title: "No unsubscribe link in the footer", Path: []int{}, Section: -1})
	down := now.Add(-48 * time.Minute)
	f.Route.Shares[1].Status, f.Route.Shares[1].Available, f.Route.Shares[1].DownSince = state.Down, false, &down
	f.Route.Shares[1].DownDetail = "mta.acme.io has refused connections since 08:52 (ECONNREFUSED on port 25 and 443)"
	f.Reputation.Status = readiness.Block
	f.Reputation.Listings = append(f.Reputation.Listings, readiness.ListingFacts{Asset: "news.acme.io", Type: "domain", Source: "Spamhaus DBL", Impact: "high",
		Detail: "Listed as a spam source", DelistURL: "https://check.spamhaus.org/"})
	res := readiness.Assemble(f)
	if res.Verdict != readiness.VerdictBlocked {
		t.Fatalf("verdict: %s %+v", res.Verdict, res.Counts)
	}
	checks := byKey(res)
	blocks := statuses(res, readiness.Block)
	if c := checks["compliance.footer"]; c.Status != readiness.Block || !hasFix(c, "insert_footer") {
		t.Fatalf("footer: %+v (blocks %v)", c, blocks)
	}
	if c := checks["routing.health"]; c.Status != readiness.Block || !hasFix(c, "route_only") {
		t.Fatalf("provider down: %+v (blocks %v)", c, blocks)
	}
	if c := checks["reputation.domain_blocklists"]; c.Status != readiness.Block {
		t.Fatalf("domain listing: %+v (blocks %v)", c, blocks)
	}
	// The held share can move to SES: the estimate says what it adds.
	if m := res.Estimates.Route.Move; m == nil || m.From != "Postal" || m.To != "SES" || m.AddCost != 0.32 || res.Estimates.Route.Down != 1 {
		t.Fatalf("move: %+v", res.Estimates.Route)
	}
	// Groups list blocks first in their status.
	for _, g := range res.Groups {
		if g.Key == "compliance" && g.Status != readiness.Block {
			t.Fatalf("compliance group: %+v", g)
		}
	}
}

func TestAssembleIncompleteAndEmpty(t *testing.T) {
	f := digest()
	f.Reputation = readiness.ReputationFacts{Status: readiness.Unknown, Via: "hostrepute", Reason: "HostRepute didn't answer",
		Outage: "Couldn't reach api.hostrepute.com (timeout after 30 s, 3 attempts)"}
	res := readiness.Assemble(f)
	if !res.Incomplete || res.Counts.Unknown == 0 || res.Problem != f.Reputation.Outage || res.Verdict == readiness.VerdictBlocked {
		t.Fatalf("incomplete: %+v %s %s", res.Counts, res.Problem, res.Verdict)
	}
	// R1.U8: only sender reputation data is missing, so the send goes on
	// the providers' own data ("Continue with provider data only")…
	if !readiness.ProviderDataOnly(res.Checks) {
		t.Fatal("HostRepute's outage leaves the providers' data")
	}
	// …but content that couldn't be checked is skipped, not provider data
	// ("Continue without the skipped checks").
	f.Content.Checked = false
	if res := readiness.Assemble(f); readiness.ProviderDataOnly(res.Checks) || !res.Incomplete {
		t.Fatalf("content unchecked: %+v", res.Counts)
	}
	if readiness.ProviderDataOnly(readiness.Assemble(digest()).Checks) {
		t.Fatal("a run where every check ran skipped nothing")
	}

	f = digest()
	f.Audience = readiness.AudienceFacts{Empty: true, Mailboxes: map[string]int64{}}
	f.Route.Shares = []readiness.ShareFacts{ses(0, 90), postal(0, 10)}
	res = readiness.Assemble(f)
	if c := byKey(res)["audience.empty"]; c.Status != readiness.Block || res.Verdict != readiness.VerdictBlocked {
		t.Fatalf("no audience: %+v %s", c, res.Verdict)
	}
	if res.Estimates.Audience.Final != 0 || len(res.Estimates.Audience.Rows) != 0 || res.Estimates.Route.TotalCost != 0 {
		t.Fatalf("empty estimates: %+v", res.Estimates)
	}
}

func TestFixes(t *testing.T) {
	src := `<mjml><mj-body><mj-section><mj-column><mj-text>Hi</mj-text><mj-button href="https://bit.ly/acme-nov">Read</mj-button></mj-column></mj-section></mj-body></mjml>`
	out, ok := readiness.InsertFooter(src, "Acme Inc · 500 Market St")
	if !ok || !strings.Contains(out, "Acme Inc · 500 Market St") || !strings.Contains(out, "unsubscribe") || !strings.HasSuffix(strings.TrimSpace(out), "</mjml>") {
		t.Fatalf("footer: %v %s", ok, out)
	}
	// A footer that lost its address is put back, not added twice.
	edited := strings.Replace(out, "Acme Inc · 500 Market St", "", 1)
	if again, ok := readiness.InsertFooter(edited, "Acme Inc · 500 Market St"); !ok || again != out || strings.Count(again, "rc:footer") != 1 {
		t.Fatalf("footer replaced: %s", again)
	}
	next, n := readiness.ReplaceLink(src, "https://bit.ly/acme-nov", "https://acme.io/blog/november")
	if n != 1 || strings.Contains(next, "bit.ly") {
		t.Fatalf("replace: %d %s", n, next)
	}
	if _, n := readiness.ReplaceLink(src, "https://nowhere.example/x", "https://acme.io"); n != 0 {
		t.Fatal("a link that isn't there isn't replaced")
	}
}

func TestSettings(t *testing.T) {
	ws := json.RawMessage(`{"bounce":2,"complaint":0.1,"mailbox_complaint":0.3,"provider_errors":5,"scope":"route"}`)
	if th := readiness.ParseThresholds(nil, ws); th.Bounce != 2 || th.Scope != "route" {
		t.Fatalf("workspace defaults: %+v", th)
	}
	if th := readiness.ParseThresholds(json.RawMessage(`{"bounce":1.5,"complaint":0.08,"mailbox_complaint":0.1,"provider_errors":5,"scope":"campaign"}`), ws); th.Bounce != 1.5 || th.Scope != "campaign" {
		t.Fatalf("campaign thresholds: %+v", th)
	}
	sv := readiness.ScheduleView(json.RawMessage(`{"mode":"at","at":"2026-11-04T14:00:00Z","timezone":"America/New_York"}`), "UTC")
	if sv.Mode != "at" || sv.Date != "2026-11-04" || sv.Time != "09:00" || sv.Timezone != "America/New_York" {
		t.Fatalf("schedule view: %+v", sv)
	}
	if sv := readiness.ScheduleView(nil, "Europe/Berlin"); sv.Mode != "now" || sv.Timezone != "Europe/Berlin" {
		t.Fatalf("default schedule: %+v", sv)
	}
}

func TestText(t *testing.T) {
	cases := map[string]string{
		readiness.Count(31604):                     "31,604",
		readiness.Count(-1485):                     "−1,485",
		readiness.JoinAnd([]string{"a"}):           "a",
		readiness.JoinAnd([]string{"a", "b"}):      "a and b",
		readiness.JoinAnd([]string{"a", "b", "c"}): "a, b and c",
		readiness.Plural(1, "block", "blocks"):     "block",
		readiness.Plural(3, "block", "blocks"):     "blocks",
		readiness.Money(4.31):                      "$4.31",
		readiness.Per1000(0.10):                    "$0.10 per 1,000",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
	if m := readiness.Minutes(28444.0 / 14); m != 34 {
		t.Fatalf("minutes: %d", m)
	}
}
