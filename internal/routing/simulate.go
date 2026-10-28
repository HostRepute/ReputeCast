package routing

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// The route tester's Campaign tab (review PR-14): the next broadcast's
// audience through the saved routing, and through the unsaved draft beside
// it, so a change shows who moves before it is saved ("1,560 recipients
// move; everyone else keeps their provider"), with what the send would do
// on its day: a warming provider's tiers, cap per mailbox provider,
// overflow or hold, and where a provider that can't take mail now would
// send its share (review R3.U4c).

// RouteCampaignInput asks for the simulation; Draft is the unsaved routing.
type RouteCampaignInput struct {
	Draft *RoutingSaveInput `json:"draft,omitempty"`
}

// RouteCampaignShare is how many recipients a provider gets.
type RouteCampaignShare struct {
	ProviderID string `json:"providerId"`
	Provider   string `json:"provider"`
	Short      string `json:"short"`
	Color      string `json:"color" enum:"ses,postal,postmark,other"`
	Count      int64  `json:"count"`
	// Policy is what happens if this provider fails during the send, and
	// Fallback the provider automatic failover would use.
	Policy   string `json:"policy" enum:"automatic,pause_and_ask,warm_only"`
	Fallback string `json:"fallback,omitempty"`
	// Down is what the send would do with this share now, while the
	// provider can't take mail (review R3.U4c).
	Down *RouteCampaignDown `json:"down,omitempty"`
}

// RouteCampaignDown is where a share goes while its provider can't take
// mail (down, paused, or its webhooks silent), as the audience build
// decides it: through the stream's fallback when the policy moves mail by
// itself and the fallback passes its checks, else nowhere, waiting.
type RouteCampaignDown struct {
	// Status is the provider's status (down, paused, degraded).
	Status string `json:"status"`
	// Moved go through To.
	Moved int64  `json:"moved"`
	To    string `json:"to,omitempty"`
	// Held wait for someone to decide; Reason says why they don't move.
	Held   int64  `json:"held"`
	Reason string `json:"reason,omitempty" enum:"pause_and_ask,no_fallback,warming,checks"`
	// Checks are the fallback's failed checks, in the reader's language.
	Checks []string `json:"checks"`
}

// RouteCampaignCondition counts the recipients a condition matched.
type RouteCampaignCondition struct {
	Field  string   `json:"field"`
	Values []string `json:"values"`
	Count  int64    `json:"count"`
}

// RouteCampaignWarmup is what a warming provider's plan does to its share
// on the send day (review R3.U4c), as the send would: recipients outside the
// day's engagement tiers go through a warm provider of their rule; the
// others go out up to the day's cap (per mailbox provider when the plan
// caps them separately); the rest goes through a warm provider when the
// plan allows overflow, or waits for the next day's cap (or the plan's
// resume) as a hold.
type RouteCampaignWarmup struct {
	ProviderID string    `json:"providerId"`
	Provider   string    `json:"provider"`
	Day        int       `json:"day"`
	Date       time.Time `json:"date"`
	Cap        int64     `json:"cap"`
	// Left is what the cap leaves on the send day, less what went out
	// already when that day is today.
	Left int64 `json:"left"`
	// Spread: the plan caps each mailbox provider separately; MailboxCaps
	// are the parts, with the recipients each part gets.
	Spread      bool                      `json:"spread"`
	MailboxCaps []RouteCampaignMailboxCap `json:"mailboxCaps"`
	Paused      bool                      `json:"paused"`
	// Tier recipients are outside the day's engagement tiers and go
	// through TierTo instead.
	Tier   int64  `json:"tier"`
	TierTo string `json:"tierTo,omitempty"`
	// Over recipients are over the cap, or every one of them while the
	// plan is paused. To takes them when the plan allows overflow; with
	// none they wait until Resumes (the next day's cap; nil while paused,
	// until someone resumes the plan).
	Over    int64      `json:"over"`
	To      string     `json:"to,omitempty"`
	Resumes *time.Time `json:"resumes,omitempty"`
}

// RouteCampaignMailboxCap is one mailbox provider's part of a warming
// provider's cap on the send day.
type RouteCampaignMailboxCap struct {
	Mailbox    string `json:"mailbox"`
	Cap        int64  `json:"cap"`
	Left       int64  `json:"left"`
	Recipients int64  `json:"recipients"`
	Over       int64  `json:"over"`
}

// RouteCampaignDraft is the unsaved routing applied to the same audience.
type RouteCampaignDraft struct {
	Shares []RouteCampaignShare `json:"shares"`
	// Moved counts recipients the draft sends to another provider.
	Moved int64 `json:"moved"`
}

// RouteCampaignResult is the Campaign tab.
type RouteCampaignResult struct {
	// None is true when there's no broadcast to simulate yet.
	None        bool                     `json:"none"`
	CampaignID  string                   `json:"campaignId"`
	Name        string                   `json:"name"`
	ScheduledAt *time.Time               `json:"scheduledAt,omitempty"`
	Timezone    string                   `json:"timezone"`
	Lists       []string                 `json:"lists"`
	Domain      string                   `json:"domain"`
	Total       int64                    `json:"total"`
	Sunset      int64                    `json:"sunset"`
	Shares      []RouteCampaignShare     `json:"shares"`
	Conditions  []RouteCampaignCondition `json:"conditions"`
	Warmup      []RouteCampaignWarmup    `json:"warmup"`
	Draft       *RouteCampaignDraft      `json:"draft,omitempty"`
	// Impact is who the unsaved routing moves in the other streams (empty
	// without a draft, and for streams it doesn't change).
	Impact []RouteStreamImpact `json:"impact"`
	// Estimated is true when the audience was larger than one simulation
	// reads: Simulated recipients went through the routing and every count
	// is scaled to the whole audience (Total and Sunset are exact).
	Estimated bool  `json:"estimated"`
	Simulated int64 `json:"simulated"`
}

// RouteStreamImpact is who an unsaved routing change moves in a stream
// other than Broadcasts (review R3.U4a): of the addresses the stream mailed
// in the last 30 days (or, when it mailed none, the workspace's contacts),
// how many the draft sends through another provider, and where most of
// them go from and to.
type RouteStreamImpact struct {
	Stream    string `json:"stream" enum:"automation,system,transactional"`
	Basis     string `json:"basis" enum:"recent,contacts"`
	Addresses int64  `json:"addresses"`
	Moved     int64  `json:"moved"`
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
}

// impactLimit bounds the addresses an impact preview reads per stream.
const impactLimit = 10_000

// impactPeriod is how far back an impact preview reads a stream's mail.
const impactPeriod = 30 * 24 * time.Hour

// simulateLimit bounds the recipients one simulation reads, so the tester
// answers within a request on any audience size.
const simulateLimit = 200_000

// simulatePage is how many recipients one query reads.
const simulatePage = 2000

// WarmCap is a warming provider's warm-up on a day (DI from warmup, which
// imports routing): its cap and what is left of it, the cap's parts per
// mailbox provider when the plan caps them separately (gmail, outlook,
// yahoo, icloud, other), the engagement tiers the day sends to, whether the
// plan sends what's over the cap elsewhere or is paused, and when the next
// day's cap starts.
type WarmCap struct {
	Cap         int64
	Left        int64
	Day         int
	Days        int
	Overflow    bool
	Paused      bool
	Tiers       []string
	MailboxCaps map[string]int64
	MailboxLeft map[string]int64
	NextDay     time.Time
}

// bucket is a mailbox provider's part of the cap: its own, or "other".
func (w WarmCap) bucket(mbp string) string {
	if _, ok := w.MailboxCaps[mbp]; ok {
		return mbp
	}
	return "other"
}

// SimulateCampaign runs the next broadcast's audience through the routing,
// with the warm-up and failover diversions the send would make (R3.U4c).
func (s *Service) SimulateCampaign(ctx context.Context, ws *store.Workspace, in RouteCampaignInput) (RouteCampaignResult, error) {
	out := RouteCampaignResult{Timezone: ws.Timezone, Lists: []string{}, Shares: []RouteCampaignShare{}, Conditions: []RouteCampaignCondition{}, Warmup: []RouteCampaignWarmup{},
		Impact: []RouteStreamImpact{}}
	saved, err := Load(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	var draft *Set
	if in.Draft != nil {
		d, err := toSet(ctx, *in.Draft, saved)
		if err != nil {
			return out, err
		}
		draft = &d
	}
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	byID := map[uuid.UUID]store.Provider{}
	for _, p := range provs {
		byID[p.ID] = p
	}
	if draft != nil {
		if out.Impact, err = s.streamImpact(ctx, ws, saved, *draft, byID); err != nil {
			return out, err
		}
	}
	c, err := s.Q.RoutingNextBroadcast(ctx, ws.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		out.None = true
		return out, nil
	}
	if err != nil {
		return out, err
	}
	out.CampaignID, out.Name, out.ScheduledAt = c.ID.String(), c.Name, c.ScheduledAt
	def := parseCampaignAudience(c.Audience)
	out.Lists = def.lists
	out.Domain = c.SenderDomain
	var sender map[string]string
	if c.HasSender {
		sender = state.SenderVerification(c.SenderVerification)
	}
	// The send day: the campaign's schedule, or now.
	at := s.now()
	if c.ScheduledAt != nil && c.ScheduledAt.After(at) {
		at = *c.ScheduledAt
	}
	warm := map[uuid.UUID]*WarmCap{}
	warmOf := func(id uuid.UUID) *WarmCap {
		if w, ok := warm[id]; ok {
			return w
		}
		var w *WarmCap
		if s.WarmCap != nil {
			if wc, ok := s.WarmCap(ctx, id, at); ok {
				w = &wc
			}
		}
		warm[id] = w
		return w
	}
	counts := map[uuid.UUID]int64{}
	draftCounts := map[uuid.UUID]int64{}
	matched := map[string]*RouteCampaignCondition{}
	order := []string{}
	// What a warming provider's plan does: recipients outside the day's
	// tiers (and where they go), and the others per mailbox part.
	tierMoved := map[uuid.UUID]int64{}
	tierTo := map[uuid.UUID]uuid.UUID{}
	capped := map[uuid.UUID]map[string]int64{}
	// Recipients of a provider that can't take mail now, by the fallback
	// their rule names (uuid.Nil for none).
	down := map[uuid.UUID]map[uuid.UUID]int64{}
	alts := map[string]uuid.UUID{}
	var moved, seen int64
	var after uuid.UUID
	limit := s.SimulateLimit
	if limit <= 0 {
		limit = simulateLimit
	}
	full := true
	for full && seen < limit {
		size := min(int64(simulatePage), limit-seen)
		rows, err := s.Q.RoutingCampaignAudience(ctx, store.RoutingCampaignAudienceParams{WorkspaceID: ws.ID, Lists: def.listIDs(), ExcludeLists: def.excludeIDs(),
			After: after, Size: int32(size)})
		if err != nil {
			return out, err
		}
		full = int64(len(rows)) == size
		for _, r := range rows {
			after = r.ID
			seen++
			if r.Tier == "sunset" && !def.includeSunset {
				out.Sunset++
				continue
			}
			out.Total++
			lists := make([]string, 0, len(r.ListIds))
			for _, l := range r.ListIds {
				lists = append(lists, l.String())
			}
			mbp := r.MailboxProvider
			rc := Recipient{Stream: providers.StreamBroadcast, Email: r.Email, Lists: lists, SenderDomain: c.SenderDomain, MailboxProvider: mbp}
			d, ok := saved.Resolve(rc)
			if !ok {
				continue
			}
			pid := d.ProviderID
			counts[pid]++
			if d.Matched != "" {
				key := d.Matched + ":" + strings.Join(conditionValues(d.Rule, d.Matched), ",")
				if matched[key] == nil {
					matched[key] = &RouteCampaignCondition{Field: d.Matched, Values: conditionValues(d.Rule, d.Matched)}
					order = append(order, key)
				}
				matched[key].Count++
			}
			if w := warmOf(pid); w != nil {
				// A warming provider sends only to its day's tiers; the
				// others go through a warm provider of their rule.
				alt, hasAlt := uuid.Nil, false
				if len(w.Tiers) > 0 && !slices.Contains(w.Tiers, r.Tier) {
					k := d.Rule.ID.String() + ":" + pid.String()
					if a, ok := alts[k]; ok {
						alt, hasAlt = a, a != uuid.Nil
					} else {
						alt, hasAlt = warmAlternative(d.Rule, pid, byID, warmOf)
						alts[k] = alt
					}
				}
				if hasAlt {
					tierMoved[pid]++
					tierTo[pid] = alt
				} else {
					if capped[pid] == nil {
						capped[pid] = map[string]int64{}
					}
					capped[pid][w.bucket(mbp)]++
				}
			}
			if p, ok := byID[pid]; !ok || !state.Available(p) {
				fb := uuid.Nil
				if d.Fallback != nil {
					fb = *d.Fallback
				}
				if down[pid] == nil {
					down[pid] = map[uuid.UUID]int64{}
				}
				down[pid][fb]++
			}
			if draft != nil {
				dd, ok := draft.Resolve(rc)
				if ok {
					draftCounts[dd.ProviderID]++
					if dd.ProviderID != pid {
						moved++
					}
				}
			}
		}
	}
	if full {
		// The audience goes on past the limit: scale what was read to it.
		size, err := s.Q.RoutingCampaignAudienceSize(ctx, store.RoutingCampaignAudienceSizeParams{WorkspaceID: ws.ID, Lists: def.listIDs(),
			ExcludeLists: def.excludeIDs()})
		if err != nil {
			return out, err
		}
		total, sunset := size.Engaged, size.Sunset
		if def.includeSunset {
			total, sunset = size.Engaged+size.Sunset, 0
		}
		if out.Total > 0 && total > out.Total {
			out.Estimated, out.Simulated = true, out.Total
			scale := func(n int64) int64 { return int64(math.Round(float64(n) * float64(total) / float64(out.Simulated))) }
			for id := range counts {
				counts[id] = scale(counts[id])
			}
			for id := range draftCounts {
				draftCounts[id] = scale(draftCounts[id])
			}
			for _, m := range matched {
				m.Count = scale(m.Count)
			}
			for id := range tierMoved {
				tierMoved[id] = scale(tierMoved[id])
			}
			for _, parts := range capped {
				for b := range parts {
					parts[b] = scale(parts[b])
				}
			}
			for _, fbs := range down {
				for fb := range fbs {
					fbs[fb] = scale(fbs[fb])
				}
			}
			moved = scale(moved)
			out.Total, out.Sunset = total, sunset
		}
	}
	policy := saved.Policy(providers.StreamBroadcast)
	l := i18n.LangFrom(ctx)
	share := func(id uuid.UUID, n int64, set Set) RouteCampaignShare {
		p := byID[id]
		sh := RouteCampaignShare{ProviderID: id.String(), Provider: p.Name, Short: state.Short(p.Name, p.Adapter), Color: state.Color(p.Adapter), Count: n,
			Policy: string(set.Policy(providers.StreamBroadcast))}
		if def, ok := set.Default(providers.StreamBroadcast); ok && def.Fallback != nil && *def.Fallback != id {
			if fp, ok := byID[*def.Fallback]; ok {
				sh.Fallback = fp.Name
			}
		}
		return sh
	}
	for _, id := range orderedProviders(saved, counts) {
		sh := share(id, counts[id], saved)
		if fbs := down[id]; len(fbs) > 0 {
			sh.Down = s.downShare(ctx, l, byID[id], fbs, policy, sender, byID)
		}
		out.Shares = append(out.Shares, sh)
	}
	for _, k := range order {
		out.Conditions = append(out.Conditions, *matched[k])
	}
	// Warm-up: what each warming provider's plan does to its share.
	for _, id := range orderedProviders(saved, counts) {
		w := warmOf(id)
		if w == nil {
			continue
		}
		wu := RouteCampaignWarmup{ProviderID: id.String(), Provider: byID[id].Name, Day: w.Day, Date: at, Cap: w.Cap, Left: w.Left, Spread: len(w.MailboxCaps) > 0,
			MailboxCaps: []RouteCampaignMailboxCap{}, Paused: w.Paused, Tier: tierMoved[id]}
		if wu.Tier > 0 {
			wu.TierTo = byID[tierTo[id]].Name
		}
		var stay, perPart int64
		for _, n := range capped[id] {
			stay += n
		}
		if wu.Spread {
			for _, b := range mailboxParts {
				mc := RouteCampaignMailboxCap{Mailbox: b, Cap: w.MailboxCaps[b], Left: w.MailboxLeft[b], Recipients: capped[id][b]}
				mc.Over = max(mc.Recipients-mc.Left, 0)
				perPart += mc.Over
				wu.MailboxCaps = append(wu.MailboxCaps, mc)
			}
		}
		wu.Over = max(stay-w.Left, perPart, 0)
		if w.Paused {
			wu.Over = stay
		}
		if w.Overflow {
			if t := overflowTarget(saved, id, max(wu.Over, 1), sender, byID, warmOf); t != nil {
				wu.To = t.Name
			}
		}
		if wu.Over > 0 && wu.To == "" && !w.Paused {
			next := w.NextDay
			wu.Resumes = &next
		}
		out.Warmup = append(out.Warmup, wu)
	}
	if draft != nil {
		dr := &RouteCampaignDraft{Shares: []RouteCampaignShare{}, Moved: moved}
		for _, id := range orderedProviders(*draft, draftCounts) {
			dr.Shares = append(dr.Shares, share(id, draftCounts[id], *draft))
		}
		out.Draft = dr
	}
	return out, nil
}

// streamImpact runs the latest recipients of each stream the draft changes
// (Broadcasts aside: the next campaign shows its own) through the saved and
// the unsaved routing, with what routing reads at send time.
func (s *Service) streamImpact(ctx context.Context, ws *store.Workspace, saved, draft Set, byID map[uuid.UUID]store.Provider) ([]RouteStreamImpact, error) {
	out := []RouteStreamImpact{}
	since := s.now().Add(-impactPeriod)
	for _, st := range []providers.Stream{providers.StreamAutomation, providers.StreamSystem, StreamTransactional} {
		if sameRouting(saved, draft, st) {
			continue
		}
		type address struct{ email, mailbox, category string }
		addrs := []address{}
		seen := map[string]bool{}
		rows, err := s.Q.RoutingStreamRecipients(ctx, store.RoutingStreamRecipientsParams{WorkspaceID: ws.ID, Stream: string(st), Since: since, Lim: impactLimit})
		if err != nil {
			return out, err
		}
		for _, r := range rows {
			k := strings.ToLower(r.Email) + "\x00" + r.Category
			if !seen[k] {
				seen[k] = true
				addrs = append(addrs, address{r.Email, r.MailboxProvider, r.Category})
			}
		}
		imp := RouteStreamImpact{Stream: string(st), Basis: "recent"}
		if len(addrs) == 0 {
			imp.Basis = "contacts"
			contacts, err := s.Q.RoutingContactAddresses(ctx, store.RoutingContactAddressesParams{WorkspaceID: ws.ID, Lim: impactLimit})
			if err != nil {
				return out, err
			}
			for _, c := range contacts {
				addrs = append(addrs, address{c.Email, c.MailboxProvider, ""})
			}
		}
		gained, lost := map[uuid.UUID]int64{}, map[uuid.UUID]int64{}
		for _, a := range addrs {
			rc := Recipient{Stream: st, Email: a.email, MailboxProvider: a.mailbox, Category: a.category}
			imp.Addresses++
			after, ok := draft.Resolve(rc)
			if !ok {
				continue
			}
			before, had := saved.Resolve(rc)
			if had && before.ProviderID == after.ProviderID {
				continue
			}
			imp.Moved++
			gained[after.ProviderID]++
			if had {
				lost[before.ProviderID]++
			}
		}
		imp.To, imp.From = byID[largest(gained)].Name, byID[largest(lost)].Name
		out = append(out, imp)
	}
	return out, nil
}

// largest is the provider with the most addresses (uuid.Nil for none).
func largest(n map[uuid.UUID]int64) uuid.UUID {
	best, most := uuid.Nil, int64(0)
	for id, c := range n {
		if c > most || c == most && most > 0 && id.String() < best.String() {
			best, most = id, c
		}
	}
	return best
}

// sameRouting reports whether two routings send a stream's mail the same
// way: the same rules (conditions, weights, fallbacks, in priority order;
// a split's order never matters) and the same failover policy.
func sameRouting(a, b Set, st providers.Stream) bool {
	key := func(s Set) string {
		type rule struct {
			C Conditions
			S []Split
			F *uuid.UUID
		}
		rules := []rule{}
		for _, r := range s.StreamRules(st) {
			sp := slices.Clone(r.Splits)
			slices.SortFunc(sp, func(x, y Split) int { return strings.Compare(x.ProviderID.String(), y.ProviderID.String()) })
			rules = append(rules, rule{r.Conditions, sp, r.Fallback})
		}
		raw, _ := json.Marshal(struct {
			R []rule
			P Policy
		}{rules, s.Policy(st)})
		return string(raw)
	}
	return key(a) == key(b)
}

// mailboxParts are the mailbox providers a warming provider's cap is split
// between, in the order the warm-up page lists them.
var mailboxParts = []string{"gmail", "outlook", "yahoo", "icloud", "other"}

// warmAlternative is where a warming provider's recipient outside its
// day's tiers goes, as the audience build sends it: the largest share of
// the same rule that isn't warming up and may take bulk mail, else the
// rule's fallback.
func warmAlternative(rule Rule, not uuid.UUID, byID map[uuid.UUID]store.Provider, warmOf func(uuid.UUID) *WarmCap) (uuid.UUID, bool) {
	best, bestW := uuid.Nil, -1
	for _, sp := range rule.Splits {
		p, ok := byID[sp.ProviderID]
		if sp.ProviderID == not || !ok || !state.Caps(p).BulkAllowed() || warmOf(sp.ProviderID) != nil {
			continue
		}
		if sp.Weight > bestW {
			best, bestW = sp.ProviderID, sp.Weight
		}
	}
	if bestW >= 0 {
		return best, true
	}
	if rule.Fallback != nil && *rule.Fallback != not {
		if _, ok := byID[*rule.Fallback]; ok {
			return *rule.Fallback, true
		}
	}
	return uuid.Nil, false
}

// overflowTarget is the warm provider over-cap recipients go through, as
// the send loop picks it: the largest share of the Broadcasts split that
// isn't warming up and can take them (state.Vet with the campaign's
// sender).
func overflowTarget(set Set, not uuid.UUID, need int64, sender map[string]string, byID map[uuid.UUID]store.Provider, warmOf func(uuid.UUID) *WarmCap) *store.Provider {
	def, ok := set.Default(providers.StreamBroadcast)
	if !ok {
		return nil
	}
	var best *store.Provider
	bestW := -1
	for _, sp := range def.Splits {
		p, ok := byID[sp.ProviderID]
		if sp.ProviderID == not || !ok || sp.Weight <= bestW || warmOf(sp.ProviderID) != nil {
			continue
		}
		if !state.Vet(p, state.Load{Broadcast: true, Messages: need, Sender: sender}).Automatic(false) {
			continue
		}
		best, bestW = &p, sp.Weight
	}
	return best
}

// downShare is what the audience build does with a share whose provider
// can't take mail now: with "pause and ask" it waits for someone to
// decide; otherwise, per fallback its rules name, the failover policy and
// the fallback's checks decide whether the recipients move there or wait.
func (s *Service) downShare(ctx context.Context, l i18n.Lang, p store.Provider, byFallback map[uuid.UUID]int64, policy Policy, sender map[string]string,
	byID map[uuid.UUID]store.Provider) *RouteCampaignDown {
	out := &RouteCampaignDown{Status: p.Status, Checks: []string{}}
	if p.DisconnectedAt != nil {
		out.Status = "disconnected"
	}
	if policy == PauseAndAsk {
		for _, n := range byFallback {
			out.Held += n
		}
		out.Reason = "pause_and_ask"
		return out
	}
	fbs := make([]uuid.UUID, 0, len(byFallback))
	for fb := range byFallback {
		fbs = append(fbs, fb)
	}
	slices.SortFunc(fbs, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	for _, fb := range fbs {
		n := byFallback[fb]
		fp, ok := byID[fb]
		if fb == uuid.Nil || !ok || fb == p.ID {
			out.Held += n
			out.Reason = cmp.Or(out.Reason, "no_fallback")
			continue
		}
		load := state.Load{Broadcast: true, Messages: n, Sender: sender}
		if s.WarmupToday != nil {
			load.Warmup = s.WarmupToday(ctx, fp.ID)
		}
		v := state.Vet(fp, load)
		id := fp.ID
		target, hold := Failover(policy, &id, func(uuid.UUID) ProviderState {
			return ProviderState{Available: v.Automatic(true), WarmingUp: v.Warming()}
		})
		if !hold && target != nil {
			out.Moved += n
			out.To = cmp.Or(out.To, fp.Name)
			continue
		}
		out.Held += n
		switch {
		case !v.Automatic(true):
			out.Reason = cmp.Or(out.Reason, "checks")
			for _, c := range v.In(l, fp.Name).Checks {
				if c.Result != state.VetPass && c.Text != "" && !slices.Contains(out.Checks, c.Text) {
					out.Checks = append(out.Checks, c.Text)
				}
			}
		default:
			out.Reason = cmp.Or(out.Reason, "warming")
		}
	}
	return out
}

// orderedProviders lists the providers with recipients, in the order the
// stream's split names them, then the others.
func orderedProviders(set Set, counts map[uuid.UUID]int64) []uuid.UUID {
	out := []uuid.UUID{}
	if def, ok := set.Default(providers.StreamBroadcast); ok {
		for _, sp := range def.Splits {
			if counts[sp.ProviderID] > 0 && !slices.Contains(out, sp.ProviderID) {
				out = append(out, sp.ProviderID)
			}
		}
	}
	rest := []uuid.UUID{}
	for id, n := range counts {
		if n > 0 && !slices.Contains(out, id) {
			rest = append(rest, id)
		}
	}
	slices.SortFunc(rest, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	return append(out, rest...)
}

func conditionValues(r Rule, field string) []string {
	switch field {
	case "list":
		return r.Conditions.Lists
	case "sender_domain":
		return r.Conditions.SenderDomains
	case "category":
		return r.Conditions.Categories
	}
	return r.Conditions.MailboxProviders
}

// campaignAudience is the part of a campaign's audience the simulation
// reads: lists, excluded lists and the sunset override. Segments are left
// out: their members are counted when the send is prepared.
type campaignAudience struct {
	lists, exclude []string
	includeSunset  bool
}

func parseCampaignAudience(raw json.RawMessage) campaignAudience {
	var in struct {
		Lists          []string `json:"lists"`
		ExcludeLists   []string `json:"excludeLists"`
		ExcludeLists2  []string `json:"exclude_lists"`
		IncludeSunset  bool     `json:"include_sunset"`
		IncludeSunset2 bool     `json:"includeSunset"`
	}
	_ = json.Unmarshal(raw, &in)
	return campaignAudience{lists: in.Lists, exclude: append(in.ExcludeLists, in.ExcludeLists2...), includeSunset: in.IncludeSunset || in.IncludeSunset2}
}

func uuids(in []string) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(in))
	for _, s := range in {
		if u, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
			out = append(out, u)
		}
	}
	return out
}

func (a campaignAudience) listIDs() []uuid.UUID    { return uuids(a.lists) }
func (a campaignAudience) excludeIDs() []uuid.UUID { return uuids(a.exclude) }
