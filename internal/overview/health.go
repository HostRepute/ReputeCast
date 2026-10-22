package overview

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/deliverability"
	"github.com/hostrepute/reputecast/internal/i18n"
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
	return scoreHealth(i18n.LangFrom(ctx), d, po.Providers, weights, state.Location(ws.Timezone)), nil
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

// action is the link at the end of a reason, in the reader's language.
func action(l i18n.Lang, m i18n.Msg, kind, asset string) *OverviewHealthAction {
	return &OverviewHealthAction{Label: m.Text(l, nil), Kind: kind, Asset: asset}
}

// scoreHealth turns the checks into a score and reasons, in the board's
// order: domain blocklists, authentication, rates, IP listings, warm-up,
// then provider problems. The reasons are in the reader's language.
func scoreHealth(l i18n.Lang, d deliverability.DeliverabilityOverview, cards []manage.ProviderCard, sent map[string]int64, loc *time.Location) OverviewSenderHealth {
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
		add(OverviewHealthReason{Key: "checks", Tone: "warn", Text: mNotChecked.Text(l, i18n.Args{"domain": out.Domain}),
			Action: action(l, mActOpenDeliv, "deliverability", "")}, 0)
	case len(clean) > 0:
		add(OverviewHealthReason{Key: "blocklist_domain", Tone: "ok", Text: mClean.Text(l, i18n.Args{"assets": i18n.AndList(l.Locale, clean), "count": len(clean), "sources": sources})}, 0)
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
		for _, n := range a.Blocklists.Listings {
			names = append(names, shortSource(n))
		}
		add(OverviewHealthReason{Key: "blocklist_domain", Tone: tone,
			Text:   mDomainListed.Text(l, i18n.Args{"asset": a.Asset, "listed": a.Blocklists.Listed, "sources": a.Blocklists.Sources, "names": strings.Join(names, ", ")}),
			Action: action(l, mActDelist, "delisting", a.Asset)}, min(45, impactCost[imp][0]*a.Blocklists.Listed))
	}
	for _, a := range domains {
		if a.Stale != nil {
			at := a.Stale.In(loc)
			add(OverviewHealthReason{Key: "checks", Tone: "warn", Text: mStale.Text(l, i18n.Args{"asset": a.Asset, "day": l.Region.ShortDate(at), "time": clockIn(l, at)}),
				Action: action(l, mActConnection, "integrations", "")}, 5)
			break
		}
	}

	// Authentication of the main sending domain, aligned per provider.
	if auth := domains[0].Auth; auth == nil {
		add(OverviewHealthReason{Key: "auth", Tone: "warn", Text: mAuthNotChecked.Text(l, i18n.Args{"domain": out.Domain}),
			Action: action(l, mActOpenDeliv, "deliverability", "")}, 0)
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
			add(OverviewHealthReason{Key: "auth", Tone: "bad", Text: mAuthFails.Text(l, i18n.Args{"failing": i18n.AndList(l.Locale, failing), "domain": out.Domain}),
				Action: action(l, mActFixRecords, "deliverability", "")}, 15*len(failing))
		case policy == "missing" || policy == "invalid" || policy == "":
			add(OverviewHealthReason{Key: "auth", Tone: "warn", Text: mNoDMARC.Text(l, i18n.Args{"domain": out.Domain}),
				Action: action(l, mActAddDMARC, "deliverability", "")}, 10)
		case len(unaligned) > 0:
			add(OverviewHealthReason{Key: "auth", Tone: "warn", Text: mUnaligned.Text(l, i18n.Args{"providers": i18n.AndList(l.Locale, unaligned), "domain": out.Domain}),
				Action: action(l, mActSeeDMARC, "deliverability", "")}, 6)
		case policy == "none":
			add(OverviewHealthReason{Key: "auth", Tone: "warn", Text: mPolicyNone.Text(l, nil)}, 4)
		default:
			add(OverviewHealthReason{Key: "auth", Tone: "ok", Text: mAligned.Text(l, i18n.Args{"policy": policy})}, 0)
		}
	}

	// Bounce and complaint rates.
	if r, cost := rateReason(l, cards, sent); r != nil {
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
		for _, n := range a.Blocklists.Listings {
			names = append(names, shortSource(n))
		}
		args := i18n.Args{"asset": a.Asset, "listed": a.Blocklists.Listed, "sources": a.Blocklists.Sources, "names": strings.Join(names, ", ")}
		text := mIPListed.Text(l, args)
		if a.Provider != "" {
			args["provider"] = state.Short(a.Provider, "")
			text = mProviderIP.Text(l, args)
		}
		add(OverviewHealthReason{Key: "blocklist_ip", Tone: tone, Text: text,
			Action: action(l, mActDelist, "delisting", a.Asset)}, min(30, impactCost[imp][1]*a.Blocklists.Listed))
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
		args := i18n.Args{"provider": c.Short, "day": c.Warmup.Day, "days": c.Warmup.Days}
		text := mWarmup.Text(l, args)
		cost := 2
		if share > 0 {
			args["share"] = l.Region.Percent(float64(share), 0)
			text = mWarmupShare.Text(l, args)
			cost = 4
		}
		add(OverviewHealthReason{Key: "warmup", Tone: "warn", Text: text, Action: nil}, cost)
	}
	for _, c := range cards {
		switch c.Status {
		case state.Down:
			text := mDown.Text(l, i18n.Args{"provider": c.Name})
			if c.Down != nil {
				text = mDownSince.Text(l, i18n.Args{"provider": c.Name, "time": clockIn(l, c.Down.Since.In(loc))})
				if c.Down.Reason == "auth" {
					text = mDownAuth.Text(l, i18n.Args{"provider": c.Name})
				}
			}
			add(OverviewHealthReason{Key: "provider_status", Tone: "bad", Text: text, Action: action(l, mActStreams, "providers", "")}, 20)
		case state.Degraded:
			text := mDegraded.Text(l, i18n.Args{"provider": c.Name})
			if c.Webhook.SilentSince != nil {
				text = mSilent.Text(l, i18n.Args{"provider": c.Name, "time": clockIn(l, c.Webhook.SilentSince.In(loc))})
			}
			add(OverviewHealthReason{Key: "provider_status", Tone: "warn", Text: text, Action: action(l, mActDiagnose, "providers", "")}, 8)
		case state.Sandbox:
			add(OverviewHealthReason{Key: "provider_status", Tone: "warn", Text: mSandbox.Text(l, i18n.Args{"provider": c.Name}),
				Action: action(l, mActProduction, "providers", "")}, 5)
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
func rateReason(l i18n.Lang, cards []manage.ProviderCard, sent map[string]int64) (*OverviewHealthReason, int) {
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
		return &OverviewHealthReason{Key: "rates", Tone: "ok", Text: mNothingSent.Text(l, nil)}, 0
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
	args := i18n.Args{"provider": worst, "kind": worstKind, "value": rateIn(l, worstValue, digits), "limit": pctIn(l, worstLimit),
		"bounce": rateIn(l, round(bounce, 1), 1), "complaint": rateIn(l, round(complaint, 2), 2)}
	switch {
	case worstUsage >= 100:
		return &OverviewHealthReason{Key: "rates", Tone: "bad", Text: mRateOver.Text(l, args), Action: action(l, mActSeeProvider, "providers", "")}, 25
	case worstUsage > 75:
		return &OverviewHealthReason{Key: "rates", Tone: "warn", Text: mRateClose.Text(l, args), Action: action(l, mActSeeProvider, "providers", "")}, 10
	case worstUsage > 50:
		return &OverviewHealthReason{Key: "rates", Tone: "warn", Text: mRateHalf.Text(l, args)}, 4
	}
	return &OverviewHealthReason{Key: "rates", Tone: "ok", Text: mRatesOK.Text(l, args)}, 0
}

func round(v float64, digits int) float64 {
	p := math.Pow(10, float64(digits))
	return math.Round(v*p) / p
}
