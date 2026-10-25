package manage

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// outage describes how each stream handles a provider that is down.
func (s *Service) outage(ctx context.Context, sn *snapshot, p store.Provider, card ProviderCard) (*ProviderOutage, error) {
	h := state.HealthOf(p)
	since := sn.now
	if h.Down != nil {
		since = h.Down.Since
	}
	o := &ProviderOutage{ProviderID: p.ID.String(), Provider: p.Name, Short: state.Short(p.Name, p.Adapter), Since: since,
		Codes: []string{}, Streams: []ProviderOutageStream{}, Forecasts: []ProviderQuotaForecast{}, WhyNot: []ProviderWhyNotMove{}, Incidents: []ProviderIncidentEntry{}}
	if card.Down != nil {
		o.Down = *card.Down
	} else {
		o.Down = ProviderDownState{Since: since, Reason: "outage", Errors: []ProviderCallError{}}
	}
	o.ErrorRate = card.ErrorRate3m
	seen := map[string]bool{}
	for _, e := range o.Down.Errors {
		code := strings.TrimSpace(e.Code)
		if e.Status >= 500 && code != "" && !strings.HasPrefix(code, "5") {
			code = itoa(e.Status) + " " + code
		}
		if code == "" {
			code = e.Message
		}
		if code != "" && !seen[code] {
			seen[code] = true
			o.Codes = append(o.Codes, code)
		}
		if o.Operation == "" {
			o.Operation = e.Op
		}
	}
	campaignIDs := []uuid.UUID{}
	for _, stream := range routing.Streams {
		role := streamRole(sn, stream, p.ID)
		if role == "" {
			continue
		}
		st, err := s.outageStream(ctx, sn, p, stream, role, since)
		if err != nil {
			return nil, err
		}
		if st.Campaign != nil {
			campaignIDs = append(campaignIDs, uuid.MustParse(st.Campaign.ID))
		}
		o.Streams = append(o.Streams, st)
	}
	for _, st := range o.Streams {
		if st.State != "failover_active" || st.Role == "fallback" || st.Fallback == "" {
			continue
		}
		for _, other := range sn.provs {
			if other.Name != st.Fallback {
				continue
			}
			if f := s.forecast(ctx, sn, other, st.Stream); f != nil {
				o.Forecasts = append(o.Forecasts, *f)
			}
		}
	}
	for _, st := range o.Streams {
		if st.Campaign == nil {
			continue
		}
		for _, alt := range st.Alternatives {
			if !alt.WarmingUp {
				continue
			}
			other := sn.byID[uuid.MustParse(alt.ProviderID)]
			wu := warmupBrief(sn, other)
			if wu == nil || wu.TodayCap == 0 {
				continue
			}
			rate := 0.0
			if hh := state.HealthOf(p); hh.Rate != nil {
				rate = hh.Rate.Max
			}
			o.WhyNot = append(o.WhyNot, ProviderWhyNotMove{ProviderID: alt.ProviderID, Held: st.Campaign.Held, TodayCap: wu.TodayCap,
				Multiple: math.Round(float64(st.Campaign.Held)/float64(wu.TodayCap)*10) / 10, AgeDays: wu.Day, Rate: rate})
		}
	}
	rows, err := s.Q.ProvIncidentsSince(ctx, store.ProvIncidentsSinceParams{WorkspaceID: sn.ws.ID, Since: since.Add(-15 * time.Minute), ProviderIds: []uuid.UUID{p.ID}, CampaignIds: campaignIDs})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		// The titles the health job kept as messages read in the reader's
		// language; incidents from before, and other areas', in English.
		var texts map[string]i18n.Text
		_ = json.Unmarshal(r.Texts, &texts)
		inc := ProviderIncidentEntry{ID: r.ID.String(), At: r.OccurredAt, Kind: r.Kind, Severity: r.Severity,
			Title: providers.TextIn(r.Title, texts["title"], sn.lang), Detail: providers.TextIn(r.Detail, texts["detail"], sn.lang)}
		if r.CampaignID != nil {
			inc.CampaignID = r.CampaignID.String()
		}
		o.Incidents = append(o.Incidents, inc)
	}
	return o, nil
}

// outageStream is one stream's column of the outage banner.
func (s *Service) outageStream(ctx context.Context, sn *snapshot, p store.Provider, stream providers.Stream, role string, since time.Time) (ProviderOutageStream, error) {
	def, _ := sn.set.Default(stream)
	policy := sn.set.Policy(stream)
	st := ProviderOutageStream{Stream: string(stream), Role: role, Policy: string(policy), Since: since, ReroutedBy: []ProviderNamedCount{}, Alternatives: []ProviderFailoverAlternative{}, SwitchBackMinutes: SwitchBackMinutes}
	var fo *store.StreamFailover
	for i, f := range sn.failovers {
		if f.Stream == string(stream) && f.ProviderID == p.ID {
			fo = &sn.failovers[i]
		}
	}
	if fo != nil {
		st.Since = fo.StartedAt
	}
	switch role {
	case "fallback":
		// The down provider was this stream's safety net; the primary carries on.
		st.State = "failover_active"
		if len(def.Splits) > 0 {
			if prim, ok := sn.byID[def.Splits[0].ProviderID]; ok {
				st.Primary = prim.Name
				if !state.Available(prim) {
					st.State = "fallback_unavailable"
				}
				counts, err := s.Q.ProvSentByKind(ctx, store.ProvSentByKindParams{WorkspaceID: sn.ws.ID, Stream: string(stream), ProviderID: &prim.ID, Since: &st.Since})
				if err != nil {
					return st, err
				}
				for _, c := range counts {
					st.Rerouted += c.N
					st.ReroutedBy = append(st.ReroutedBy, ProviderNamedCount{Name: kindLabel(sn.lang, c.Kind, c.N), Count: c.N})
				}
			}
		}
		for _, other := range sn.provs {
			if other.ID == p.ID || !state.Available(other) || inRoute(def, other.ID) {
				continue
			}
			st.Suggestion = state.Short(other.Name, other.Adapter)
			break
		}
		return st, nil
	}
	var fallback *store.Provider
	if fo != nil && fo.FallbackProviderID != nil {
		if fp, ok := sn.byID[*fo.FallbackProviderID]; ok {
			fallback = &fp
		}
	} else if def.Fallback != nil {
		if fp, ok := sn.byID[*def.Fallback]; ok {
			fallback = &fp
		}
	}
	automatic := fo != nil && fo.Mode == "automatic" && fallback != nil
	if fo == nil && fallback != nil {
		_, hold := routing.Failover(policy, &fallback.ID, func(id uuid.UUID) routing.ProviderState {
			fp := sn.byID[id]
			return routing.ProviderState{Available: state.Available(fp), WarmingUp: warmupBrief(sn, fp) != nil}
		})
		automatic = !hold
	}
	if automatic {
		st.State = "failover_active"
		st.Fallback, st.FallbackShort = fallback.Name, state.Short(fallback.Name, fallback.Adapter)
		if stream == providers.StreamAutomation {
			rows, err := s.Q.ProvReroutedByAutomation(ctx, store.ProvReroutedByAutomationParams{WorkspaceID: sn.ws.ID, ProviderID: &fallback.ID, Since: &st.Since})
			if err != nil {
				return st, err
			}
			for _, r := range rows {
				st.Rerouted += r.N
				st.ReroutedBy = append(st.ReroutedBy, ProviderNamedCount{Name: strings.ToLower(r.Name), Count: r.N})
			}
		} else {
			rows, err := s.Q.ProvSentByKind(ctx, store.ProvSentByKindParams{WorkspaceID: sn.ws.ID, Stream: string(stream), ProviderID: &fallback.ID, Since: &st.Since})
			if err != nil {
				return st, err
			}
			for _, r := range rows {
				st.Rerouted += r.N
				st.ReroutedBy = append(st.ReroutedBy, ProviderNamedCount{Name: kindLabel(sn.lang, r.Kind, r.N), Count: r.N})
			}
		}
		return st, nil
	}
	st.State = "held"
	if fallback != nil {
		st.Fallback, st.FallbackShort = fallback.Name, state.Short(fallback.Name, fallback.Adapter)
	}
	held, err := s.Q.ProvHeldRoutes(ctx, store.ProvHeldRoutesParams{WorkspaceID: sn.ws.ID, ProviderID: p.ID})
	if err != nil {
		return st, err
	}
	if len(held) > 0 {
		h := held[0]
		st.State = "waiting"
		hc := &ProviderHeldCampaign{ID: h.CampaignID.String(), Name: h.CampaignName, Planned: h.Planned, Sent: h.Sent, Held: max(h.Planned-h.Sent, 0), OtherShares: []ProviderRouteShare{}}
		var detail struct {
			Decision string `json:"decision"`
		}
		_ = json.Unmarshal([]byte(h.StatusDetail), &detail)
		hc.Decision = detail.Decision
		routes, err := s.Q.ProvCampaignRoutes(ctx, h.CampaignID)
		if err != nil {
			return st, err
		}
		var total int64
		for _, r := range routes {
			total += r.Planned
		}
		for _, r := range routes {
			if r.ProviderID == p.ID || total == 0 {
				continue
			}
			hc.OtherShares = append(hc.OtherShares, ProviderRouteShare{ProviderID: r.ProviderID.String(), Short: state.Short(r.ProviderName, r.Adapter), Color: state.Color(r.Adapter),
				Percent: int(math.Round(float64(r.Planned) / float64(total) * 100))})
		}
		st.Campaign = hc
		sender := campaignSender(ctx, s.Q, sn.ws.ID, h.CampaignID)
		for _, other := range sn.provs {
			if other.ID == p.ID {
				continue
			}
			load := state.Load{Broadcast: true, Messages: hc.Held, Sender: sender}
			wu := warmupBrief(sn, other)
			if wu != nil {
				load.Warmup = &state.WarmupToday{Cap: wu.TodayCap, Sent: wu.SentToday, Paused: wu.Status == "paused"}
			}
			vet := state.Vet(other, load)
			// Providers that can't take mail at all aren't alternatives.
			if failed, ok := vet.Failed(); ok && (failed.Key == state.VetAvailable || failed.Key == state.VetBulkTerms) {
				continue
			}
			short := state.Short(other.Name, other.Adapter)
			alt := ProviderFailoverAlternative{ProviderID: other.ID.String(), Name: other.Name, Short: short,
				Allowed: vet.Allowed(), Recommended: vet.Automatic(false), Checks: vet.In(sn.lang, short).Checks}
			if wu != nil {
				alt.WarmingUp = true
				alt.ExceedsCap = hc.Held > wu.TodayCap-wu.SentToday
			}
			st.Alternatives = append(st.Alternatives, alt)
		}
	}
	return st, nil
}

func inRoute(def routing.Rule, id uuid.UUID) bool {
	if def.Fallback != nil && *def.Fallback == id {
		return true
	}
	for _, sp := range def.Splits {
		if sp.ProviderID == id {
			return true
		}
	}
	return false
}

// forecast says when a provider on a monthly plan runs out while it carries
// a stream for a provider that is down.
func (s *Service) forecast(ctx context.Context, sn *snapshot, p store.Provider, stream string) *ProviderQuotaForecast {
	h := state.HealthOf(p)
	if h.Quota == nil || h.Quota.Kind != "monthly_plan" || h.Quota.Max <= 0 {
		return nil
	}
	// The last seven whole days, as the routing screen counts them.
	today := dayStart(sn.now, sn.loc)
	week := today.AddDate(0, 0, -7)
	rows, err := s.Q.ProvStreamVolume(ctx, store.ProvStreamVolumeParams{WorkspaceID: sn.ws.ID, Since: &week, Until: &today})
	if err != nil {
		return nil
	}
	var perDay int64
	for _, r := range rows {
		if r.Stream == stream {
			perDay = r.N / 7
		}
	}
	if perDay <= 0 {
		return nil
	}
	f := &ProviderQuotaForecast{ProviderID: p.ID.String(), PerDay: roundCount(perDay), Stream: stream}
	remaining := h.Quota.Max - h.Quota.Used
	if remaining > 0 {
		days := int(math.Ceil(float64(remaining) / float64(perDay)))
		t := dayStart(sn.now, sn.loc).AddDate(0, 0, days)
		f.RunsOutOn = &t
	}
	if pr := state.PricingOf(p); pr.NextTier != nil {
		f.NextTierAdds, f.NextTierPrice = pr.NextTier.Adds, pr.NextTier.Price
	}
	return f
}

func roundCount(n int64) int64 {
	switch {
	case n >= 1000:
		return int64(math.Round(float64(n)/100)) * 100
	case n >= 100:
		return int64(math.Round(float64(n)/10)) * 10
	}
	return n
}

// silent describes a provider whose webhooks went quiet.
func (s *Service) silent(ctx context.Context, sn *snapshot, p store.Provider) (*ProviderWebhooksSilent, error) {
	h := state.HealthOf(p)
	since := *h.Webhooks.SilentSince
	out := &ProviderWebhooksSilent{ProviderID: p.ID.String(), Provider: p.Name, Short: state.Short(p.Name, p.Adapter), Since: since,
		Campaigns: []ProviderNamedCount{}, Checks: []ProviderWebhookCheck{}, Causes: []string{}, Awaiting: []ProviderAwaitingMessage{}, Hourly: make([]int64, 24),
		WebhookURL: WebhookURL(s.URLs, p), BroadcastPolicy: string(sn.set.Policy(providers.StreamBroadcast))}
	cnt, err := s.Q.ProvAwaitingCount(ctx, store.ProvAwaitingCountParams{ProviderID: &p.ID, Since: &since})
	if err != nil {
		return nil, err
	}
	out.Accepted = cnt.Total
	camps, err := s.Q.ProvAwaitingCampaigns(ctx, store.ProvAwaitingCampaignsParams{ProviderID: &p.ID, Since: &since})
	if err != nil {
		return nil, err
	}
	for _, c := range camps {
		out.Campaigns = append(out.Campaigns, ProviderNamedCount{Name: c.Name})
	}
	if def, ok := sn.set.Default(providers.StreamBroadcast); ok {
		for _, sp := range def.Splits {
			if sp.ProviderID == p.ID && sp.Weight > 0 {
				out.BroadcastShareHeld = true
			}
		}
	}
	for _, c := range h.Webhooks.Checks {
		out.Checks = append(out.Checks, ProviderWebhookCheck{Key: c.Key, OK: c.OK, At: c.At, Detail: providers.TextIn(c.Detail, c.DetailText, sn.lang), ID: c.ID, URL: c.URL,
			Count: c.Count, Queued: c.Queued})
	}
	out.Causes = append(out.Causes, h.Webhooks.Causes...)
	from := sn.now.Truncate(time.Hour).Add(-23 * time.Hour)
	out.HourlyFrom = from
	hours, err := s.Q.ProvRawEventsByHour(ctx, store.ProvRawEventsByHourParams{ProviderID: p.ID, Since: from})
	if err != nil {
		return nil, err
	}
	for _, hr := range hours {
		i := int(hr.Hour.Sub(from).Hours())
		if i >= 0 && i < 24 {
			out.Hourly[i] = hr.N
		}
	}
	// The newest messages without an event, then the last one the provider
	// reported on before it went quiet.
	rows, err := s.Q.ProvAwaitingStatus(ctx, store.ProvAwaitingStatusParams{ProviderID: &p.ID, Since: ptrTime(since.Add(time.Nanosecond)), Lim: AwaitingRows})
	if err != nil {
		return nil, err
	}
	last, err := s.Q.ProvLastConfirmed(ctx, store.ProvLastConfirmedParams{ProviderID: &p.ID, Before: &since})
	if err != nil {
		return nil, err
	}
	for _, r := range last {
		rows = append(rows, store.ProvAwaitingStatusRow(r))
	}
	for _, r := range rows {
		m := ProviderAwaitingMessage{Recipient: MaskEmail(r.Email), Source: sourceLabel(sn.lang, r.CampaignName, r.AutomationName, r.AutomationNode, r.Kind), Status: "accepted",
			DeliveredAt: r.DeliveredAt}
		if r.SentAt != nil {
			m.At = *r.SentAt
		}
		if r.ProviderMessageID != nil {
			m.ProviderMessageID = *r.ProviderMessageID
		}
		switch r.Status {
		case "delivered":
			m.Status = "delivered"
		case "bounced", "deferred", "complained", "rejected":
			m.Status = r.Status
		}
		out.Awaiting = append(out.Awaiting, m)
	}
	if adapter, err := s.Factory.Open(ctx, p); err == nil {
		_, out.CanBackfill = adapter.(providers.EventBackfiller)
		_, out.CanRegister = adapter.(providers.WebhookRegistrar)
	}
	out.CanSendTest = true
	return out, nil
}

// authFailure describes a provider that rejects its credentials.
func (s *Service) authFailure(ctx context.Context, sn *snapshot, p store.Provider, card ProviderCard) (*ProviderAuthFailure, error) {
	af := &ProviderAuthFailure{ProviderID: p.ID.String(), Provider: p.Name, Short: state.Short(p.Name, p.Adapter), Streams: []ProviderOutageStream{}}
	if card.Down != nil {
		af.Down = *card.Down
	}
	for _, stream := range routing.Streams {
		role := streamRole(sn, stream, p.ID)
		if role == "" {
			continue
		}
		st, err := s.outageStream(ctx, sn, p, stream, role, af.Down.Since)
		if err != nil {
			return nil, err
		}
		if role == "fallback" {
			st.State = "fallback_unavailable"
		} else {
			af.Rerouted += st.Rerouted
		}
		af.Streams = append(af.Streams, st)
	}
	return af, nil
}

// sourceLabel names what sent a message: the campaign, the automation step
// or the kind of system mail (in the reader's language).
func sourceLabel(l i18n.Lang, campaign, automation string, node *string, kind string) string {
	switch {
	case campaign != "":
		return campaign
	case automation != "":
		if node != nil && *node != "" {
			return automation + " · " + *node
		}
		return automation
	}
	return mSource.Text(l, i18n.Args{"kind": kind})
}

// MaskEmail hides most of an address's local part: "r.w···@outlook.com".
func MaskEmail(email string) string {
	at := strings.LastIndexByte(email, '@')
	if at <= 0 {
		return email
	}
	local := []rune(email[:at])
	keep := min(3, len(local))
	if len(local) <= 3 {
		keep = 1
	}
	return string(local[:keep]) + "···" + email[at:]
}

func ptrTime(t time.Time) *time.Time { return &t }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}
