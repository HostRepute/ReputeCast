package manage

import (
	"context"
	"encoding/json"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/warmup"
)

// SwitchBackMinutes is how long a recovered provider must stay healthy
// before automatic failover switches back to it.
const SwitchBackMinutes = 10

// snapshot gathers everything the overview needs in a few queries.
type snapshot struct {
	ws *store.Workspace
	// lang is how the reader reads (i18n.LangFrom).
	lang      i18n.Lang
	loc       *time.Location
	now       time.Time
	provs     []store.Provider
	byID      map[uuid.UUID]store.Provider
	month     map[uuid.UUID]store.ProvDailyTotalsRow
	week      map[uuid.UUID]store.ProvDailyTotalsRow
	today     map[uuid.UUID]store.ProvDailyTotalsRow
	win3      map[uuid.UUID]store.ProvCallWindowTotalsRow
	winNow    map[uuid.UUID]store.ProvCallWindowTotalsRow
	winToday  map[uuid.UUID]store.ProvCallWindowTotalsRow
	set       routing.Set
	plans     []store.WarmupListPlansRow
	failovers []store.StreamFailover
	sync      map[uuid.UUID]store.ProviderSuppressionSync
	// fromFailover is mail a provider sent for streams failed over to it.
	fromFailover map[uuid.UUID]int64
}

func dayStart(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// dateOf is the calendar date of t in loc as a UTC midnight (Postgres date).
func dateOf(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func (s *Service) load(ctx context.Context, ws *store.Workspace) (*snapshot, error) {
	sn := &snapshot{ws: ws, lang: i18n.LangFrom(ctx), loc: state.Location(ws.Timezone), now: s.now(), byID: map[uuid.UUID]store.Provider{}}
	var err error
	if sn.provs, err = s.Q.ProvListActive(ctx, ws.ID); err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(sn.provs))
	for _, p := range sn.provs {
		sn.byID[p.ID] = p
		ids = append(ids, p.ID)
	}
	today := dateOf(sn.now, sn.loc)
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	daily := func(since time.Time) (map[uuid.UUID]store.ProvDailyTotalsRow, error) {
		rows, err := s.Q.ProvDailyTotals(ctx, store.ProvDailyTotalsParams{ProviderIds: ids, Since: since, Until: today})
		out := map[uuid.UUID]store.ProvDailyTotalsRow{}
		for _, r := range rows {
			out[r.ProviderID] = r
		}
		return out, err
	}
	if sn.month, err = daily(monthStart); err != nil {
		return nil, err
	}
	if sn.week, err = daily(today.AddDate(0, 0, -6)); err != nil {
		return nil, err
	}
	if sn.today, err = daily(today); err != nil {
		return nil, err
	}
	windows := func(since time.Time) (map[uuid.UUID]store.ProvCallWindowTotalsRow, error) {
		rows, err := s.Q.ProvCallWindowTotals(ctx, store.ProvCallWindowTotalsParams{ProviderIds: ids, Since: since})
		out := map[uuid.UUID]store.ProvCallWindowTotalsRow{}
		for _, r := range rows {
			out[r.ProviderID] = r
		}
		return out, err
	}
	if sn.win3, err = windows(sn.now.Add(-3 * time.Minute)); err != nil {
		return nil, err
	}
	// The send rate "now" is the last complete minute: everything since its
	// start minus the minute in progress.
	cur := sn.now.Truncate(time.Minute)
	if sn.winNow, err = windows(cur.Add(-time.Minute)); err != nil {
		return nil, err
	}
	inProgress, err := windows(cur)
	if err != nil {
		return nil, err
	}
	for id, w := range sn.winNow {
		w.Messages -= inProgress[id].Messages
		sn.winNow[id] = w
	}
	if sn.winToday, err = windows(dayStart(sn.now, sn.loc)); err != nil {
		return nil, err
	}
	if sn.set, err = routing.Load(ctx, s.Q, ws.ID); err != nil {
		return nil, err
	}
	if sn.plans, err = s.Q.WarmupListPlans(ctx, ws.ID); err != nil {
		return nil, err
	}
	if sn.failovers, err = s.Q.ProvActiveFailovers(ctx, ws.ID); err != nil {
		return nil, err
	}
	sn.fromFailover = map[uuid.UUID]int64{}
	for _, f := range sn.failovers {
		if f.Mode != "automatic" || f.FallbackProviderID == nil {
			continue
		}
		started := f.StartedAt
		n, err := s.Q.ProvSentThroughSince(ctx, store.ProvSentThroughSinceParams{WorkspaceID: ws.ID, Stream: f.Stream, ProviderID: f.FallbackProviderID, Since: &started})
		if err != nil {
			return nil, err
		}
		sn.fromFailover[*f.FallbackProviderID] += n
	}
	syncRows, err := s.Q.SuppSyncList(ctx, ids)
	if err != nil {
		return nil, err
	}
	sn.sync = map[uuid.UUID]store.ProviderSuppressionSync{}
	for _, r := range syncRows {
		sn.sync[r.ProviderID] = r
	}
	return sn, nil
}

// Overview returns the providers screen.
func (s *Service) Overview(ctx context.Context, ws *store.Workspace) (ProvidersOverview, error) {
	out := ProvidersOverview{Providers: []ProviderCard{}, Routing: []ProviderStreamRoute{}}
	sn, err := s.load(ctx, ws)
	if err != nil {
		return out, err
	}
	out.Summary = s.summary(ctx, sn)
	if len(sn.provs) == 0 {
		return out, nil
	}
	for _, p := range sn.provs {
		out.Providers = append(out.Providers, s.card(sn, p))
	}
	out.Routing = streamRoutes(sn)
	for i, p := range sn.provs {
		h := state.HealthOf(p)
		switch {
		case p.Status == state.Down && h.Down != nil && h.Down.Reason == "auth" && out.ProviderAuthFailure == nil:
			af, err := s.authFailure(ctx, sn, p, out.Providers[i])
			if err != nil {
				return out, err
			}
			out.ProviderAuthFailure = af
		case p.Status == state.Down && out.Outage == nil:
			o, err := s.outage(ctx, sn, p, out.Providers[i])
			if err != nil {
				return out, err
			}
			out.Outage = o
		case h.Webhooks != nil && h.Webhooks.SilentSince != nil && out.Silent == nil:
			sl, err := s.silent(ctx, sn, p)
			if err != nil {
				return out, err
			}
			out.Silent = sl
		}
	}
	return out, nil
}

func (s *Service) summary(ctx context.Context, sn *snapshot) ProvidersSummary {
	today := dateOf(sn.now, sn.loc)
	sum := ProvidersSummary{Connected: len(sn.provs), ByStatus: []ProviderStatusCount{}, SentBy: []ProviderMonthAmount{},
		MonthStart: time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, sn.loc)}
	counts := map[string]int{}
	var oldest *time.Time
	for _, p := range sn.provs {
		counts[p.Status]++
		if p.LastCheckedAt != nil && (oldest == nil || p.LastCheckedAt.Before(*oldest)) {
			t := *p.LastCheckedAt
			oldest = &t
		}
		m := sn.month[p.ID]
		sum.SentThisMonth += m.Sent
		sum.SentBy = append(sum.SentBy, ProviderMonthAmount{ProviderID: p.ID.String(), Short: state.Short(p.Name, p.Adapter), Value: float64(m.Sent)})
		sum.CostToDate += costToDate(p, m)
	}
	sum.CheckedAt = oldest
	slices.SortStableFunc(sum.SentBy, func(a, b ProviderMonthAmount) int { return int(b.Value - a.Value) })
	for _, st := range []string{state.Healthy, state.WarmingUp, state.Degraded, state.Down, state.Sandbox, state.Paused} {
		if counts[st] > 0 {
			sum.ByStatus = append(sum.ByStatus, ProviderStatusCount{Status: st, Count: counts[st]})
		}
	}
	sum.CostToDate = math.Round(sum.CostToDate*100) / 100
	if sum.SentThisMonth > 0 {
		v := math.Round(sum.CostToDate/float64(sum.SentThisMonth)*1000*100) / 100
		sum.CostPer1000 = &v
	}
	sum.Suppressions, _ = s.Q.SuppCount(ctx, sn.ws.ID)
	sum.InSync = true
	for _, p := range sn.provs {
		if !state.Caps(p).SuppressionSync {
			continue
		}
		r, ok := sn.sync[p.ID]
		if !ok || r.Mode == "off" {
			continue
		}
		if r.Status != "ok" || r.Pending > 0 {
			sum.InSync = false
		}
		if r.LastPushAt != nil && (sum.SyncedAt == nil || r.LastPushAt.Before(*sum.SyncedAt)) {
			t := *r.LastPushAt
			sum.SyncedAt = &t
		}
	}
	return sum
}

// costToDate is the month's cost: metered usage, or the plan price.
func costToDate(p store.Provider, m store.ProvDailyTotalsRow) float64 {
	pr := state.PricingOf(p)
	switch pr.Model {
	case "plan":
		return pr.Monthly
	case "self_hosted":
		return float64(m.CostMicros) / 1e6
	}
	return float64(m.CostMicros) / 1e6
}

func (s *Service) card(sn *snapshot, p store.Provider) ProviderCard {
	h := state.HealthOf(p)
	caps := state.Caps(p)
	c := ProviderCard{
		ID: p.ID.String(), Adapter: p.Adapter, Name: p.Name, Short: state.Short(p.Name, p.Adapter), Label: p.Label, Color: state.Color(p.Adapter),
		Status: p.Status, Roles: roles(sn, p.ID), IPMode: p.IpMode, IPs: p.Ips, Account: providers.TextIn(h.Account, h.AccountText, sn.lang), AccountKind: "account",
		Capabilities: providerBadges(sn.lang, p), TakingFailover: []string{}, CheckedAt: p.LastCheckedAt, ConnectedAt: p.ConnectedAt, Credentials: []ProviderCredentialField{},
	}
	if c.IPs == nil {
		c.IPs = []string{}
	}
	if d, ok := providers.Lookup(p.Adapter); ok && (d.Kind == "self_hosted" || d.Kind == "smtp") {
		c.AccountKind = "server"
	}
	if c.Account == "" && h.Version != "" {
		c.Account = h.Version
	}
	c.Webhook = ProviderWebhookFeed{Kind: caps.Webhooks, LastEventAt: p.LastEventAt}
	if h.Webhooks != nil {
		c.Webhook.SilentSince = h.Webhooks.SilentSince
	}
	week := sn.week[p.ID]
	if week.ApiCalls > 0 {
		v := round1(float64(week.ApiErrors) / float64(week.ApiCalls) * 100)
		c.ErrorRate7d = &v
	} else if r, ok := errorRate7dFromHealth(p); ok {
		c.ErrorRate7d = &r
	}
	if w := sn.win3[p.ID]; w.Calls > 0 {
		v := round1(float64(w.Errors) / float64(w.Calls) * 100)
		c.ErrorRate3m = &v
	} else {
		zero := 0.0
		c.ErrorRate3m = &zero
	}
	lim := state.LimitsOf(p)
	stale := h.Webhooks != nil && h.Webhooks.SilentSince != nil
	c.Bounce = ProviderLimitRate{Limit: lim.Bounce.Limit, Review: lim.Bounce.Review, Pause: lim.Bounce.Pause, Label: lim.Bounce.Label, Stale: stale}
	c.Complaint = ProviderLimitRate{Limit: lim.Complaint.Limit, Review: lim.Complaint.Review, Pause: lim.Complaint.Pause, Label: lim.Complaint.Label, Stale: stale}
	if week.Sent > 0 {
		v := round2(float64(week.Bounced) / float64(week.Sent) * 100)
		c.Bounce.Value = &v
	}
	if week.Delivered > 0 {
		v := round2(float64(week.Complained) / float64(week.Delivered) * 100)
		c.Complaint.Value = &v
	}
	// What the provider itself measures wins while it is fresh: its review
	// and pause levels apply to its own numbers.
	if r := state.ReputationOf(h, sn.now); r != nil {
		b, cp := r.Bounce, r.Complaint
		c.Bounce.Value, c.Complaint.Value = &b, &cp
	}
	// Quota and rate from the provider's API, the warm-up plan overriding
	// both while the provider warms up.
	c.Quota = ProviderQuotaUse{Kind: "none"}
	if h.Quota != nil {
		c.Quota = ProviderQuotaUse{Kind: h.Quota.Kind, Max: h.Quota.Max, Used: h.Quota.Used, FromFailover: sn.fromFailover[p.ID]}
	}
	if h.Rate != nil {
		c.Rate = ProviderSendRate{Max: h.Rate.Max, Now: h.Rate.Now, PeakToday: h.Rate.PeakToday, Label: h.Rate.Label}
	}
	if w := sn.winNow[p.ID]; w.Messages > 0 {
		c.Rate.Now = round1(float64(w.Messages) / 60)
	}
	if w := sn.winToday[p.ID]; w.PeakMinute > 0 {
		c.Rate.PeakToday = math.Max(c.Rate.PeakToday, round1(float64(w.PeakMinute)/60))
	}
	if wu := warmupBrief(sn, p); wu != nil {
		c.Warmup = wu
		c.Quota = ProviderQuotaUse{Kind: "warmup_daily", Max: wu.TodayCap, Used: wu.SentToday}
		c.Rate.Max, c.Rate.Label = wu.Rate, "ramp limit"
	}
	if p.Status == state.Down {
		c.Rate.Throttled = true
	}
	pr := state.PricingOf(p)
	m := sn.month[p.ID]
	c.Cost = ProviderCostToDate{Basis: pr.Model, ToDate: round2(costToDate(p, m)), Monthly: pr.Monthly, Source: pr.Source}
	switch pr.Model {
	case "metered":
		if pr.Per1000 > 0 {
			v := pr.Per1000
			c.Cost.Per1000 = &v
		}
	default:
		if m.Sent > 0 && pr.Monthly > 0 {
			v := round2(pr.Monthly / float64(m.Sent) * 1000)
			c.Cost.Per1000 = &v
		}
	}
	if h.Down != nil && p.Status == state.Down {
		c.Down = downOf(sn.lang, p, h)
	}
	for _, f := range sn.failovers {
		if f.FallbackProviderID != nil && *f.FallbackProviderID == p.ID && f.Mode == "automatic" {
			c.TakingFailover = append(c.TakingFailover, f.Stream)
		}
	}
	if d, ok := providers.Lookup(p.Adapter); ok {
		creds, _ := s.Factory.Credentials(p)
		var have map[string]any
		_ = json.Unmarshal(creds, &have)
		for _, f := range d.Fields {
			if f.Secret {
				v, _ := have[f.Key].(string)
				c.Credentials = append(c.Credentials, ProviderCredentialField{Key: f.Key, Label: f.In(sn.lang).Label, Set: v != ""})
			}
		}
	}
	return c
}

// errorRate7dFromHealth reads a seeded or imported 7-day error rate when no
// call has been recorded yet.
func errorRate7dFromHealth(p store.Provider) (float64, bool) {
	var raw struct {
		ErrorRate7d *float64 `json:"errorRate7d"`
	}
	if json.Unmarshal(p.Health, &raw) == nil && raw.ErrorRate7d != nil {
		return *raw.ErrorRate7d, true
	}
	return 0, false
}

func downOf(l i18n.Lang, p store.Provider, h state.Health) *ProviderDownState {
	d := &ProviderDownState{Since: h.Down.Since, Reason: h.Down.Reason, LastSuccessAt: h.LastSuccessAt, LastSuccessOp: h.LastSuccessOp, Errors: []ProviderCallError{}}
	for _, e := range h.RecentErrors {
		d.Errors = append(d.Errors, ProviderCallError{At: e.At, Op: e.Op, Status: e.Status, Code: e.Code, Message: providers.TextIn(e.Message, e.Text, l)})
	}
	if len(d.Errors) == 0 && h.LastError != nil {
		e := h.LastError
		d.Errors = append(d.Errors, ProviderCallError{At: e.At, Op: e.Op, Status: e.Status, Code: e.Code, Message: providers.TextIn(e.Message, e.Text, l)})
	}
	if ps := h.Down.ProviderStatus; ps != nil && ps.Incident {
		d.StatusSummary = providers.TextIn(ps.Summary, ps.SummaryText, l)
	}
	if desc, ok := providers.Lookup(p.Adapter); ok {
		if pg, ok := statusPage(l, desc); ok {
			d.StatusLabel, d.StatusURL = pg[0], pg[1]
		}
		for _, f := range desc.Fields {
			if f.Secret && f.Required {
				// The adapter's English: the banner derives its words
				// ("the token") from it.
				d.CredentialLabel = f.Label
				break
			}
		}
	}
	return d
}

// statusPage is the provider's public status page and its button's label.
func statusPage(l i18n.Lang, d providers.Descriptor) ([2]string, bool) {
	switch d.Adapter {
	case "ses":
		return [2]string{mStatusPage.Text(l, i18n.Args{"provider": "AWS"}), "https://health.aws.amazon.com/health/status"}, true
	case "postmark":
		return [2]string{mStatusPage.Text(l, i18n.Args{"provider": "Postmark"}), "https://status.postmarkapp.com"}, true
	}
	return [2]string{}, false
}

func warmupBrief(sn *snapshot, p store.Provider) *ProviderWarmupBrief {
	for _, w := range sn.plans {
		if w.ProviderID != p.ID || w.Status == "done" {
			continue
		}
		day := min(max(state.WarmupDay(w.StartDate, sn.now, sn.loc), 1), int(w.Days))
		plan := warmup.ParsePlan(w.Schedule, w.Progress)
		cap, rate, share := plan.Cap(day), plan.Rate(day), plan.Share(day)
		return &ProviderWarmupBrief{PlanID: w.ID.String(), Day: day, Days: int(w.Days), Status: w.Status, TodayCap: cap, SentToday: sn.today[p.ID].Sent, Rate: rate, Share: share}
	}
	return nil
}

// roles lists what a provider does in each stream's routing.
func roles(sn *snapshot, id uuid.UUID) []ProviderStreamRole {
	out := []ProviderStreamRole{}
	for _, stream := range routing.Streams {
		def, ok := sn.set.Default(stream)
		if !ok {
			continue
		}
		inSplit := false
		for _, sp := range def.Splits {
			if sp.ProviderID != id || sp.Weight == 0 {
				continue
			}
			inSplit = true
			r := ProviderStreamRole{Stream: string(stream), Role: "primary", Percent: sp.Weight}
			if len(def.Splits) > 1 {
				r.Role = "split"
			}
			for _, f := range sn.failovers {
				if f.Stream == string(stream) && f.ProviderID == id && f.Mode == "held" {
					r.Held = true
				}
			}
			out = append(out, r)
		}
		if !inSplit {
			for _, r := range sn.set.StreamRules(stream) {
				if !r.Conditions.Empty() && len(r.Splits) == 1 && r.Splits[0].ProviderID == id {
					out = append(out, ProviderStreamRole{Stream: string(stream), Role: "condition"})
					break
				}
			}
		}
		if def.Fallback != nil && *def.Fallback == id {
			r := ProviderStreamRole{Stream: string(stream), Role: "fallback"}
			for _, f := range sn.failovers {
				if f.Stream == string(stream) && f.FallbackProviderID != nil && *f.FallbackProviderID == id && f.Mode == "automatic" {
					r.Active = true
				}
			}
			out = append(out, r)
		}
	}
	return out
}

func streamRoutes(sn *snapshot) []ProviderStreamRoute {
	out := []ProviderStreamRoute{}
	for _, stream := range routing.Streams {
		r := ProviderStreamRoute{Stream: string(stream), Splits: []ProviderRouteShare{}, Failover: string(sn.set.Policy(stream))}
		def, ok := sn.set.Default(stream)
		if !ok {
			out = append(out, r)
			continue
		}
		for _, sp := range def.Splits {
			p, ok := sn.byID[sp.ProviderID]
			if !ok || sp.Weight == 0 {
				continue
			}
			r.Splits = append(r.Splits, ProviderRouteShare{ProviderID: p.ID.String(), Short: state.Short(p.Name, p.Adapter), Color: state.Color(p.Adapter), Percent: sp.Weight})
		}
		slices.SortStableFunc(r.Splits, func(a, b ProviderRouteShare) int { return b.Percent - a.Percent })
		if def.Fallback != nil {
			if p, ok := sn.byID[*def.Fallback]; ok {
				r.Fallback = &ProviderRouteShare{ProviderID: p.ID.String(), Short: state.Short(p.Name, p.Adapter), Color: state.Color(p.Adapter)}
			}
		}
		for _, rule := range sn.set.StreamRules(stream) {
			if !rule.Conditions.Empty() {
				r.Conditions++
			}
		}
		out = append(out, r)
	}
	return out
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }

// streamRole is the down provider's role in a stream, or "".
func streamRole(sn *snapshot, stream providers.Stream, id uuid.UUID) string {
	for _, r := range roles(sn, id) {
		if r.Stream == string(stream) {
			return r.Role
		}
	}
	return ""
}

// kindLabel names system mail kinds for sentences ("double opt-in
// confirmations"), in the reader's language; a kind it doesn't know reads as
// its name.
func kindLabel(l i18n.Lang, kind string, n int64) string {
	switch kind {
	case "double_opt_in", "invite", "preference_link", "password_reset", "api", "":
		return mKindLabel.Text(l, i18n.Args{"kind": kind, "count": n})
	}
	return strings.ReplaceAll(kind, "_", " ")
}
