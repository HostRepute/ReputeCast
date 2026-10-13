package overview

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/deliverability"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/store"
)

func f(v float64) *float64 { return &v }

// The Main board's sample: clean domain, aligned authentication, rates well
// inside every limit, the Postal IP on UCEPROTECT L3 and its warm-up.
func sample() (deliverability.DeliverabilityOverview, []manage.ProviderCard, map[string]int64) {
	d := deliverability.DeliverabilityOverview{
		Assets: []deliverability.DeliverabilityAssetHealth{
			{Asset: "news.acme.io", Type: "domain", Role: "sending_domain", Blocklists: deliverability.DeliverabilityAssetBlocklists{Via: "hostrepute", Sources: 84, Listings: []string{}},
				Auth: &deliverability.DeliverabilityDomainAuth{SPF: "pass", DKIM: "pass", DMARC: "quarantine"}},
			{Asset: "203.0.113.25", Type: "ip", Role: "dedicated_ip", Provider: "Postal", Blocklists: deliverability.DeliverabilityAssetBlocklists{Via: "hostrepute", Sources: 84, Listed: 1, Listings: []string{"UCEPROTECT Level 3"}}},
		},
		Blocklists: deliverability.DeliverabilityBlocklistStatus{Via: "hostrepute", Sources: 84, Results: []deliverability.DeliverabilityBlocklistSource{
			{Source: "UCEPROTECT Level 3", Impact: "low", Listed: true, ListedAssets: []string{"203.0.113.25"}},
		}},
		DMARC: deliverability.DeliverabilityDMARCSummary{Sources: []deliverability.DeliverabilityDMARCSource{{Label: "Amazon SES", Known: true, SPF: "aligned", DKIM: "aligned"}}},
	}
	cards := []manage.ProviderCard{
		{ID: "ses", Name: "Amazon SES", Short: "SES", Status: "healthy", Bounce: manage.ProviderLimitRate{Value: f(0.6), Limit: 5}, Complaint: manage.ProviderLimitRate{Value: f(0.04), Limit: 0.1},
			Roles: []manage.ProviderStreamRole{{Stream: "broadcast", Role: "split", Percent: 90}}},
		{ID: "postmark", Name: "Postmark", Short: "Postmark", Status: "healthy", Bounce: manage.ProviderLimitRate{Value: f(0.2), Limit: 10}, Complaint: manage.ProviderLimitRate{Value: f(0.01), Limit: 0.1}},
		{ID: "postal", Name: "Postal", Short: "Postal", Status: "warming_up", Bounce: manage.ProviderLimitRate{Value: f(0.9), Limit: 2}, Complaint: manage.ProviderLimitRate{Value: f(0.05), Limit: 0.1},
			Warmup: &manage.ProviderWarmupBrief{Day: 9, Days: 21}, Roles: []manage.ProviderStreamRole{{Stream: "broadcast", Role: "split", Percent: 10}}},
	}
	return d, cards, map[string]int64{"ses": 131604, "postmark": 3412, "postal": 14298}
}

func TestScoreHealthBoard(t *testing.T) {
	d, cards, sent := sample()
	h := scoreHealth(d, cards, sent, time.UTC)
	if h.Score != 92 || h.Label != "Good" || h.Tone != "ok" || h.Domain != "news.acme.io" {
		t.Fatalf("score: %+v", h)
	}
	want := []string{
		"news.acme.io is clean on 84 of 84 blocklists",
		"SPF, DKIM and DMARC (p=quarantine) aligned on every sending provider",
		"Bounces 0.6% and complaints 0.04% across providers, well inside every limit",
		"Postal IP 203.0.113.25 is listed on 1 of 84 sources (UCEPROTECT L3)",
		"Postal warm-up is on day 9 of 21, so 10% of broadcasts ride a young IP",
	}
	if len(h.Reasons) != len(want) {
		t.Fatalf("reasons: %+v", h.Reasons)
	}
	for i, w := range want {
		if h.Reasons[i].Text != w {
			t.Errorf("reason %d: %q, want %q", i, h.Reasons[i].Text, w)
		}
	}
	if a := h.Reasons[3].Action; a == nil || a.Kind != "delisting" || a.Asset != "203.0.113.25" {
		t.Fatalf("delisting action: %+v", a)
	}
}

func TestScoreHealthProblems(t *testing.T) {
	d, cards, sent := sample()
	// A high-impact listing of the domain, DMARC off and a provider down.
	d.Assets[0].Blocklists.Listed, d.Assets[0].Blocklists.Listings, d.Assets[0].Blocklists.HighImpact = 1, []string{"Spamhaus DBL"}, true
	d.Blocklists.Results = append(d.Blocklists.Results, deliverability.DeliverabilityBlocklistSource{Source: "Spamhaus DBL", Impact: "high", Listed: true, ListedAssets: []string{"news.acme.io"}})
	d.Assets[0].Auth.DMARC = "none"
	cards[0].Status = "down"
	h := scoreHealth(d, cards, sent, time.UTC)
	if h.Label != "Poor" || h.Tone != "bad" || h.Score >= fairFrom {
		t.Fatalf("score: %d %s", h.Score, h.Label)
	}
	texts := []string{}
	for _, r := range h.Reasons {
		texts = append(texts, r.Tone+": "+r.Text)
	}
	all := strings.Join(texts, "\n")
	for _, want := range []string{"bad: news.acme.io is listed on 1 of 84 sources (Spamhaus DBL)", "warn: SPF and DKIM are aligned on every sending provider, but DMARC is p=none", "bad: Amazon SES is down"} {
		if !strings.Contains(all, want) {
			t.Errorf("missing %q in\n%s", want, all)
		}
	}
	// Over a provider's own limit.
	_, cards, sent = sample()
	cards[2].Complaint.Value = f(0.12)
	if r, cost := rateReason(cards, sent); r.Tone != "bad" || cost != 25 || r.Text != "Postal complaints are 0.12%, over its 0.1% limit" {
		t.Fatalf("over the limit: %+v", r)
	}
	// Nothing sent.
	if r, _ := rateReason(cards, map[string]int64{}); r.Tone != "ok" || !strings.HasPrefix(r.Text, "Nothing was sent") {
		t.Fatalf("no volume: %+v", r)
	}
}

func TestScoreHealthPending(t *testing.T) {
	h := scoreHealth(deliverability.DeliverabilityOverview{}, nil, nil, time.UTC)
	if h.State != "pending" || h.Score != 0 || len(h.Reasons) != 0 {
		t.Fatalf("pending: %+v", h)
	}
}

func TestProviderNaming(t *testing.T) {
	ses := manage.ProviderCard{Adapter: "ses", Name: "Amazon SES", Label: "us-east-1", IPMode: "shared", Roles: []manage.ProviderStreamRole{
		{Stream: "broadcast", Role: "split", Percent: 90}, {Stream: "automation", Role: "primary", Percent: 100}, {Stream: "system", Role: "fallback"}}}
	postal := manage.ProviderCard{Adapter: "postal", Name: "Postal", Label: "self-hosted · mta.acme.io", IPMode: "dedicated", IPs: []string{"203.0.113.25"},
		Roles: []manage.ProviderStreamRole{{Stream: "broadcast", Role: "split", Percent: 10}}}
	postmark := manage.ProviderCard{Adapter: "postmark", Name: "Postmark", Label: "Transactional stream", Roles: []manage.ProviderStreamRole{{Stream: "system", Role: "primary", Percent: 100}}}
	for _, c := range []struct {
		card         manage.ProviderCard
		name, detail string
	}{
		{ses, "Amazon SES", "us-east-1 · Broadcasts, Automations"},
		{postal, "Postal (self-hosted)", "mta.acme.io · 203.0.113.25 dedicated · 10% of Broadcasts"},
		{postmark, "Postmark", "Transactional stream · System"},
	} {
		if got := displayName(c.card); got != c.name {
			t.Errorf("name %q, want %q", got, c.name)
		}
		if got := detail(c.card); got != c.detail {
			t.Errorf("detail %q, want %q", got, c.detail)
		}
	}
}

func TestRefreshFailure(t *testing.T) {
	ok := time.Date(2026, 10, 12, 9, 12, 0, 0, time.UTC)
	h := map[string]any{
		"lastSuccessAt": ok,
		"lastError":     map[string]any{"at": ok.Add(32 * time.Minute), "op": "GetAccount", "status": 503, "code": "ServiceUnavailable"},
		"recentErrors": []map[string]any{
			{"at": ok.Add(32 * time.Minute), "op": "GetAccount", "status": 503},
			{"at": ok.Add(30 * time.Minute), "op": "GetAccount", "status": 503},
			{"at": ok.Add(29 * time.Minute), "op": "GetAccount", "status": 503},
			{"at": ok.Add(-time.Hour), "op": "GetAccount", "status": 500},
		},
	}
	raw, _ := json.Marshal(h)
	f := refreshFailure(store.Provider{Status: "healthy", Health: raw})
	if f == nil || f.Op != "GetAccount" || f.Status != 503 || len(f.Attempts) != 3 || !f.Attempts[0].Before(f.Attempts[2]) {
		t.Fatalf("refresh failure: %+v", f)
	}
	// Recovered, or down (the providers screen's outage state), isn't stale.
	h["lastSuccessAt"] = ok.Add(40 * time.Minute)
	raw, _ = json.Marshal(h)
	if refreshFailure(store.Provider{Status: "healthy", Health: raw}) != nil {
		t.Fatal("a success after the error clears it")
	}
	if refreshFailure(store.Provider{Status: "down", Health: raw}) != nil {
		t.Fatal("down providers aren't stale")
	}
}

func TestSunsetPolicy(t *testing.T) {
	p := audience.ParsePolicy(json.RawMessage(`{"after_days":180}`))
	if got := policySentence(p); got != "Sunset contacts get one re-engagement email, then stop receiving broadcasts after 30 days." {
		t.Fatal(got)
	}
	for tier, want := range map[string]string{"active": "opened in the last 30 days", "cooling": "last open 31 to 90 days ago", "at_risk": "last open 91 to 180 days ago", "sunset": "no open in 180 days"} {
		if got := describeTier(p, tier); got != want {
			t.Errorf("%s: %q, want %q", tier, got, want)
		}
	}
	// The descriptions follow the thresholds the nightly tier job applies.
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	for days, want := range map[int]string{30: "active", 31: "cooling", 90: "cooling", 91: "at_risk", 180: "at_risk", 181: "sunset"} {
		open := now.AddDate(0, 0, -days)
		if got := audience.TierFor(p, &open, &open, nil, open, now); got != want {
			t.Errorf("last open %d days ago: %s, want %s", days, got, want)
		}
	}
	off := audience.ParsePolicy(json.RawMessage(`{"enabled":false}`))
	if got := policySentence(off); got != "No sunset policy: Sunset contacts keep receiving broadcasts." {
		t.Fatal(got)
	}
	erase := audience.ParsePolicy(json.RawMessage(`{"reengage":false,"stop_after_days":14,"delete_after":true,"delete_after_months":6}`))
	if got := policySentence(erase); got != "Sunset contacts stop receiving broadcasts after 14 days; they're erased after 6 months in Sunset." {
		t.Fatal(got)
	}
}
