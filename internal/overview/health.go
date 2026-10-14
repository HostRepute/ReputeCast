package overview

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/deliverability"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// Score thresholds: Good from 90, Fair from 75, Poor below.
const (
	goodFrom = 90
	fairFrom = 75
)

// SenderHealth scores the workspace's sending from the Deliverability checks
// (blocklists, authentication) and the providers' own numbers (bounce and
// complaint rates against each provider's limits, warm-up, status).
func (s *Service) SenderHealth(ctx context.Context, ws *store.Workspace) (OverviewSenderHealth, error) {
	d, err := s.Deliverability.Overview(ctx, ws)
	if err != nil {
		return OverviewSenderHealth{}, err
	}
	po, err := s.Providers.Overview(ctx, ws)
	if err != nil {
		return OverviewSenderHealth{}, err
	}
	weights, err := s.sentLastWeek(ctx, ws, po.Providers)
	if err != nil {
		return OverviewSenderHealth{}, err
	}
	return scoreHealth(d, po.Providers, weights, state.Location(ws.Timezone)), nil
}

// sentLastWeek is each provider's volume over the last 7 days, to weigh
// their rates into one figure.
func (s *Service) sentLastWeek(ctx context.Context, ws *store.Workspace, cards []manage.ProviderCard) (map[string]int64, error) {
	ids := make([]uuid.UUID, 0, len(cards))
	for _, c := range cards {
		if id, err := uuid.Parse(c.ID); err == nil {
			ids = append(ids, id)
		}
	}
	out := map[string]int64{}
	if len(ids) == 0 {
		return out, nil
	}
	loc := state.Location(ws.Timezone)
	n := s.now().In(loc)
	today := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
	rows, err := s.Q.ProvDailyTotals(ctx, store.ProvDailyTotalsParams{ProviderIds: ids, Since: today.AddDate(0, 0, -6), Until: today})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ProviderID.String()] = r.Sent
	}
	return out, nil
}

// shortSource is the short name of a blocklist: "UCEPROTECT Level 3" →
// "UCEPROTECT L3".
func shortSource(s string) string {
	return strings.Replace(s, "Level ", "L", 1)
}

func joinAnd(xs []string) string {
	switch len(xs) {
	case 0:
		return ""
	case 1:
		return xs[0]
	}
	return strings.Join(xs[:len(xs)-1], ", ") + " and " + xs[len(xs)-1]
}

func pctText(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64) + "%"
}

// rateText is "0.6%" for bounces (one decimal) and "0.04%" for complaints.
func rateText(v float64, digits int) string {
	return strconv.FormatFloat(v, 'f', digits, 64) + "%"
}

var impactCost = map[string][2]int{"high": {30, 20}, "medium": {12, 8}, "low": {6, 4}}

func impactOf(sources []deliverability.DeliverabilityBlocklistSource, asset string) string {
	worst := ""
	rank := map[string]int{"high": 3, "medium": 2, "low": 1}
	for _, r := range sources {
		if !r.Listed {
			continue
		}
		for _, a := range r.ListedAssets {
			if strings.EqualFold(a, asset) && rank[r.Impact] > rank[worst] {
				worst = r.Impact
			}
		}
	}
	return worst
}

// scoreHealth turns the checks into a score and reasons, in the board's
// order: domain blocklists, authentication, rates, IP listings, warm-up,
// then provider problems.
func scoreHealth(d deliverability.DeliverabilityOverview, cards []manage.ProviderCard, sent map[string]int64, loc *time.Location) OverviewSenderHealth {
	out := OverviewSenderHealth{State: "ready", Score: 100, Reasons: []OverviewHealthReason{}, Via: d.Blocklists.Via, Sources: d.Blocklists.Sources, CheckedAt: d.Blocklists.CheckedAt}
	var domains, ips []deliverability.DeliverabilityAssetHealth
	for _, a := range d.Assets {
		switch a.Role {
		case "sending_domain":
			domains = append(domains, a)
		case "dedicated_ip":
			ips = append(ips, a)
		}
	}
	if len(domains) == 0 {
		out.State, out.Score, out.Tone, out.Label = "pending", 0, "neutral", ""
		return out
	}
	out.Domain = domains[0].Asset
	add := func(r OverviewHealthReason, cost int) {
		out.Reasons = append(out.Reasons, r)
		out.Score -= cost
	}

	// Blocklists: clean domains and IPs on one line, each listing on its own.
	var clean []string
	sources := 0
	for _, a := range append(append([]deliverability.DeliverabilityAssetHealth{}, domains...), ips...) {
		if a.Blocklists.Via == "none" || a.Blocklists.Sources == 0 {
			continue
		}
		sources = max(sources, a.Blocklists.Sources)
		if a.Blocklists.Listed == 0 {
			clean = append(clean, a.Asset)
		}
	}
	switch {
	case d.Blocklists.Via == "none":
		add(OverviewHealthReason{Key: "checks", Tone: "warn", Text: out.Domain + " hasn't been checked against blocklists yet; the first check runs within the hour",
			Action: &OverviewHealthAction{Label: "open Deliverability", Kind: "deliverability"}}, 0)
	case len(clean) > 0:
		verb := "is"
		if len(clean) > 1 {
			verb = "are"
		}
		add(OverviewHealthReason{Key: "blocklist_domain", Tone: "ok", Text: fmt.Sprintf("%s %s clean on %d of %d blocklists", joinAnd(clean), verb, sources, sources)}, 0)
	}
	for _, a := range domains {
		if a.Blocklists.Listed == 0 {
			continue
		}
		imp := impactOf(d.Blocklists.Results, a.Asset)
		tone := "warn"
		if a.Blocklists.HighImpact || imp == "high" {
			tone = "bad"
		}
		names := make([]string, 0, len(a.Blocklists.Listings))
		for _, l := range a.Blocklists.Listings {
			names = append(names, shortSource(l))
		}
		add(OverviewHealthReason{Key: "blocklist_domain", Tone: tone, Text: fmt.Sprintf("%s is listed on %d of %d sources (%s)", a.Asset, a.Blocklists.Listed, a.Blocklists.Sources, strings.Join(names, ", ")),
			Action: &OverviewHealthAction{Label: "request delisting", Kind: "delisting", Asset: a.Asset}}, min(45, impactCost[imp][0]*a.Blocklists.Listed))
	}
	for _, a := range domains {
		if a.Stale != nil {
			add(OverviewHealthReason{Key: "checks", Tone: "warn", Text: fmt.Sprintf("Blocklist results for %s are from %s: HostRepute can't be refreshed", a.Asset, a.Stale.In(loc).Format("Mon 2 Jan 15:04")),
				Action: &OverviewHealthAction{Label: "check the connection", Kind: "integrations"}}, 5)
			break
		}
	}

	// Authentication of the main sending domain, aligned per provider.
	if auth := domains[0].Auth; auth == nil {
		add(OverviewHealthReason{Key: "auth", Tone: "warn", Text: "SPF, DKIM and DMARC for " + out.Domain + " haven't been checked yet",
			Action: &OverviewHealthAction{Label: "open Deliverability", Kind: "deliverability"}}, 0)
	} else {
		var unaligned []string
		for _, src := range d.DMARC.Sources {
			if src.Known && (src.SPF != "aligned" || src.DKIM != "aligned") {
				unaligned = append(unaligned, strings.SplitN(src.Label, " · ", 2)[0])
			}
		}
		policy := auth.DMARC
		switch {
		case auth.SPF == "fail" || auth.DKIM == "fail":
			failing := []string{}
			if auth.SPF == "fail" {
				failing = append(failing, "SPF")
			}
			if auth.DKIM == "fail" {
				failing = append(failing, "DKIM")
			}
			add(OverviewHealthReason{Key: "auth", Tone: "bad", Text: joinAnd(failing) + " fails on " + out.Domain + "; receivers may reject or spam-folder the mail",
				Action: &OverviewHealthAction{Label: "fix the records", Kind: "deliverability"}}, 15*len(failing))
		case policy == "missing" || policy == "invalid" || policy == "":
			add(OverviewHealthReason{Key: "auth", Tone: "warn", Text: out.Domain + " has no valid DMARC record, so receivers can't verify the domain",
				Action: &OverviewHealthAction{Label: "add DMARC", Kind: "deliverability"}}, 10)
		case len(unaligned) > 0:
			add(OverviewHealthReason{Key: "auth", Tone: "warn", Text: fmt.Sprintf("%s mail isn't aligned with %s in DMARC reports", joinAnd(unaligned), out.Domain),
				Action: &OverviewHealthAction{Label: "see DMARC", Kind: "deliverability"}}, 6)
		case policy == "none":
			add(OverviewHealthReason{Key: "auth", Tone: "warn", Text: "SPF and DKIM are aligned on every sending provider, but DMARC is p=none, so receivers don't act on failures"}, 4)
		default:
			add(OverviewHealthReason{Key: "auth", Tone: "ok", Text: fmt.Sprintf("SPF, DKIM and DMARC (p=%s) aligned on every sending provider", policy)}, 0)
		}
	}

	// Bounce and complaint rates.
	if r, cost := rateReason(cards, sent); r != nil {
		add(*r, cost)
	}

	// Dedicated IPs on blocklists.
	for _, a := range ips {
		if a.Blocklists.Listed == 0 {
			continue
		}
		imp := impactOf(d.Blocklists.Results, a.Asset)
		tone := "warn"
		if a.Blocklists.HighImpact || imp == "high" {
			tone = "bad"
		}
		names := make([]string, 0, len(a.Blocklists.Listings))
		for _, l := range a.Blocklists.Listings {
			names = append(names, shortSource(l))
		}
		who := "Dedicated IP"
		if a.Provider != "" {
			who = state.Short(a.Provider, "") + " IP"
		}
		add(OverviewHealthReason{Key: "blocklist_ip", Tone: tone, Text: fmt.Sprintf("%s %s is listed on %d of %d sources (%s)", who, a.Asset, a.Blocklists.Listed, a.Blocklists.Sources, strings.Join(names, ", ")),
			Action: &OverviewHealthAction{Label: "request delisting", Kind: "delisting", Asset: a.Asset}}, min(30, impactCost[imp][1]*a.Blocklists.Listed))
	}

	// Warm-up and provider problems.
	for _, c := range cards {
		if c.Warmup == nil {
			continue
		}
		share := 0
		for _, r := range c.Roles {
			if r.Stream == "broadcast" && (r.Role == "split" || r.Role == "primary") {
				share = r.Percent
			}
		}
		text := fmt.Sprintf("%s warm-up is on day %d of %d", c.Short, c.Warmup.Day, c.Warmup.Days)
		cost := 2
		if share > 0 {
			text += fmt.Sprintf(", so %d%% of broadcasts ride a young IP", share)
			cost = 4
		}
		add(OverviewHealthReason{Key: "warmup", Tone: "warn", Text: text, Action: nil}, cost)
	}
	for _, c := range cards {
		switch c.Status {
		case state.Down:
			text := c.Name + " is down"
			if c.Down != nil {
				text = fmt.Sprintf("%s has been down since %s", c.Name, c.Down.Since.In(loc).Format("15:04"))
				if c.Down.Reason == "auth" {
					text = c.Name + " rejects ReputeCast's credentials, so nothing goes through it"
				}
			}
			add(OverviewHealthReason{Key: "provider_status", Tone: "bad", Text: text, Action: &OverviewHealthAction{Label: "see what each stream is doing", Kind: "providers"}}, 20)
		case state.Degraded:
			text := c.Name + " is degraded"
			if c.Webhook.SilentSince != nil {
				text = fmt.Sprintf("%s webhooks have been silent since %s, so its bounces and complaints aren't counted", c.Name, c.Webhook.SilentSince.In(loc).Format("15:04"))
			}
			add(OverviewHealthReason{Key: "provider_status", Tone: "warn", Text: text, Action: &OverviewHealthAction{Label: "diagnose", Kind: "providers"}}, 8)
		case state.Sandbox:
			add(OverviewHealthReason{Key: "provider_status", Tone: "warn", Text: c.Name + " is in sandbox mode and only sends to verified addresses",
				Action: &OverviewHealthAction{Label: "request production access", Kind: "providers"}}, 5)
		}
	}

	out.Score = max(0, min(100, out.Score))
	switch {
	case out.Score >= goodFrom:
		out.Label, out.Tone = "Good", "ok"
	case out.Score >= fairFrom:
		out.Label, out.Tone = "Fair", "warn"
	default:
		out.Label, out.Tone = "Poor", "bad"
	}
	return out
}

// rateReason is the bounce and complaint line: the volume-weighted rates and
// the provider closest to its own limit.
func rateReason(cards []manage.ProviderCard, sent map[string]int64) (*OverviewHealthReason, int) {
	var bw, cw, bsum, csum float64
	worst, worstUsage, worstKind := "", 0.0, ""
	var worstValue, worstLimit float64
	for _, c := range cards {
		if c.Status == state.Paused {
			continue
		}
		w := float64(sent[c.ID])
		for _, r := range []struct {
			kind string
			l    manage.ProviderLimitRate
		}{{"bounces", c.Bounce}, {"complaints", c.Complaint}} {
			if r.l.Value == nil || r.l.Limit <= 0 {
				continue
			}
			if r.kind == "bounces" {
				bsum += *r.l.Value * w
				bw += w
			} else {
				csum += *r.l.Value * w
				cw += w
			}
			if u := round(*r.l.Value/r.l.Limit*100, 1); u > worstUsage {
				worst, worstUsage, worstKind, worstValue, worstLimit = c.Name, u, r.kind, *r.l.Value, r.l.Limit
			}
		}
	}
	if bw == 0 && cw == 0 {
		return &OverviewHealthReason{Key: "rates", Tone: "ok", Text: "Nothing was sent in the last 7 days, so there are no bounce or complaint rates to check"}, 0
	}
	bounce, complaint := 0.0, 0.0
	if bw > 0 {
		bounce = bsum / bw
	}
	if cw > 0 {
		complaint = csum / cw
	}
	digits := 2
	if worstKind == "bounces" {
		digits = 1
	}
	switch {
	case worstUsage >= 100:
		return &OverviewHealthReason{Key: "rates", Tone: "bad", Text: fmt.Sprintf("%s %s are %s, over its %s limit", worst, worstKind, rateText(worstValue, digits), pctText(worstLimit)),
			Action: &OverviewHealthAction{Label: "see the provider", Kind: "providers"}}, 25
	case worstUsage > 75:
		return &OverviewHealthReason{Key: "rates", Tone: "warn", Text: fmt.Sprintf("%s %s are %s, close to its %s limit", worst, worstKind, rateText(worstValue, digits), pctText(worstLimit)),
			Action: &OverviewHealthAction{Label: "see the provider", Kind: "providers"}}, 10
	case worstUsage > 50:
		return &OverviewHealthReason{Key: "rates", Tone: "warn", Text: fmt.Sprintf("Bounces %s and complaints %s across providers; %s %s are past half of its %s limit",
			rateText(round(bounce, 1), 1), rateText(round(complaint, 2), 2), worst, worstKind, pctText(worstLimit))}, 4
	}
	return &OverviewHealthReason{Key: "rates", Tone: "ok", Text: fmt.Sprintf("Bounces %s and complaints %s across providers, well inside every limit", rateText(round(bounce, 1), 1), rateText(round(complaint, 2), 2))}, 0
}

func round(v float64, digits int) float64 {
	p := math.Pow(10, float64(digits))
	return math.Round(v*p) / p
}
