package readiness

import (
	"math"
	"net/url"
	"strconv"
	"strings"
)

// What a fix changes (SR-13): a short note under its button ("+3 min ·
// +$0.31", "Changes 2 links · preview first", "No change to this send")
// and the effects the fix review lists before it is applied.

// fixEffects fills in the notes and effects of the fixes Assemble offers.
func (b *builder) fixEffects() {
	for i := range b.checks {
		c := &b.checks[i]
		for j := range c.Fixes {
			f := &c.Fixes[j]
			switch {
			case f.Kind == "apply" && f.Action == FixRouteOnly:
				b.routeEffects(f)
			case f.Kind == "apply" && f.Action == FixRuleAddParam:
				b.paramEffects(f)
			case f.Kind == "apply" && f.Action == FixExcludeSunset:
				if n, err := strconv.ParseInt(f.Params["count"], 10, 64); err == nil {
					f.Note = tx("fixnote.audience", A{"count": b.f.Audience.Final - n})
				}
			case f.Kind == "apply" && f.Action == FixIncludeUnengaged:
				f.Note = tx("fixnote.audience", A{"count": b.f.Audience.Final + b.f.Audience.SunsetExcluded})
			case f.Kind == "apply" && f.Action == FixUseSubject:
				f.Note = tx("fixnote.subject", nil)
			case f.Kind == "apply" && f.Action == FixInsertFooter:
				f.Note = tx("fixnote.footer", nil)
			case f.Kind == "apply" && f.Action == FixReplaceLink:
				f.Note = tx("fixnote.link", nil)
			case f.Kind == "link" && strings.HasSuffix(f.Href, "/deliverability") && c.Group == GroupReputation:
				f.Note = tx("fixnote.noChange", nil)
			}
		}
	}
}

// routeMinutes is how long the routes send as they are, in parallel: the
// slowest route by the engine's estimate (waits such as the A/B test's left
// out, as they are the same either way), else by its rate.
func (b *builder) routeMinutes() int {
	longest := 0.0
	for _, s := range b.f.Route.Shares {
		longest = math.Max(longest, b.shareSeconds(s, s.Recipients))
	}
	return int(math.Ceil(longest / 60))
}

// shareSeconds is how long a route takes to send n messages: the engine's
// estimate for the route scaled to n, else n at the route's rate.
func (b *builder) shareSeconds(s ShareFacts, n int64) float64 {
	if est := b.f.Route.Estimate; est != nil && s.Recipients > 0 {
		for _, p := range est.Providers {
			if p.ProviderID.String() == s.ProviderID && p.Seconds > 0 {
				return p.Seconds * float64(n) / float64(s.Recipients)
			}
		}
	}
	if s.Rate > 0 && n > 0 {
		return float64(n) / s.Rate
	}
	return 0
}

func (b *builder) routeCost() float64 {
	total := 0.0
	for _, s := range b.f.Route.Shares {
		total += float64(s.Recipients) * s.Pricing.Per1000 / 1000
	}
	return math.Round(total*100) / 100
}

// routeEffects: the whole send through one provider, at its rate and price.
func (b *builder) routeEffects(f *fix) {
	var to *ShareFacts
	for i := range b.f.Route.Shares {
		if b.f.Route.Shares[i].ProviderID == f.Params["providerId"] {
			to = &b.f.Route.Shares[i]
		}
	}
	total := b.f.Audience.Final
	if to == nil || total <= 0 {
		return
	}
	oldCost, newCost := b.routeCost(), math.Round(float64(total)*to.Pricing.Per1000/1000*100)/100
	oldMin := b.routeMinutes()
	newMin := oldMin
	if secs := b.shareSeconds(*to, total); secs > 0 {
		newMin = int(math.Ceil(secs / 60))
	}
	dMin, dCost := newMin-oldMin, math.Round((newCost-oldCost)*100)/100
	f.Note = tx("fixnote.route", A{"minutes": abs(dMin), "time": change(float64(dMin)), "cost": ArgMoney(math.Abs(dCost)), "costChange": change(dCost)})
	f.Effects = []techRow{
		{Label: tx("effect.route", nil), Value: tx("effect.routeValue", A{"provider": to.Short})},
		{Label: tx("effect.duration", nil), Value: tx("effect.durationValue", A{"from": oldMin, "to": newMin})},
		{Label: tx("effect.cost", nil), Value: tx("effect.costValue", A{"to": ArgMoney(newCost), "from": ArgMoney(oldCost)})},
	}
	for _, s := range b.f.Route.Shares {
		if s.ProviderID != to.ProviderID && s.WarmupCap > 0 {
			f.Effects = append(f.Effects, techRow{Label: tx("effect.warmup", nil), Value: tx("effect.warmupValue", A{"provider": s.Short, "cap": s.WarmupCap})})
		}
	}
	f.Effects = append(f.Effects, techRow{Label: tx("effect.scope", nil), Value: tx("effect.scopeValue", A{"stream": streamText(b.f.Route.Stream)})})
}

// paramEffects: the links the parameter is added to, each before and after.
func (b *builder) paramEffects(f *fix) {
	param, value := f.Params["param"], f.Params["value"]
	n := 0
	for _, l := range b.f.Content.Report.LinkList {
		u, err := url.Parse(l.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Query().Has(param) {
			continue
		}
		sep := "?"
		if strings.Contains(l.URL, "?") {
			sep = "&"
		}
		label := tx("effect.link", A{"text": l.Text, "kind": l.Kind})
		f.Effects = append(f.Effects, techRow{Label: label, Value: RawText(l.URL + sep + url.QueryEscape(param) + "=" + url.QueryEscape(value)), Before: l.URL})
		n++
	}
	f.Note = tx("fixnote.links", A{"count": n})
	f.Effects = append(f.Effects, techRow{Label: tx("effect.unchanged", nil), Value: tx("effect.unchangedValue", nil)})
}

// change says whether a fix makes something more, less or the same.
func change(d float64) string {
	switch {
	case d > 0:
		return "more"
	case d < 0:
		return "less"
	}
	return "same"
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
