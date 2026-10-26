package routing

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// The route tester's Campaign tab (review PR-14): the next broadcast's
// audience through the saved routing, and through the unsaved draft beside
// it, so a change shows who moves before it is saved ("1,560 recipients
// move; everyone else keeps their provider").

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
}

// RouteCampaignCondition counts the recipients a condition matched.
type RouteCampaignCondition struct {
	Field  string   `json:"field"`
	Values []string `json:"values"`
	Count  int64    `json:"count"`
}

// RouteCampaignOverflow is what a warming provider's cap on the send day
// sends to another provider.
type RouteCampaignOverflow struct {
	ProviderID string    `json:"providerId"`
	Provider   string    `json:"provider"`
	To         string    `json:"to"`
	Count      int64     `json:"count"`
	Cap        int64     `json:"cap"`
	Day        int       `json:"day"`
	Date       time.Time `json:"date"`
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
	Overflow    []RouteCampaignOverflow  `json:"overflow"`
	Draft       *RouteCampaignDraft      `json:"draft,omitempty"`
	// Estimated is true when the audience was larger than one simulation
	// reads: Simulated recipients went through the routing and every count
	// is scaled to the whole audience (Total and Sunset are exact).
	Estimated bool  `json:"estimated"`
	Simulated int64 `json:"simulated"`
}

// simulateLimit bounds the recipients one simulation reads, so the tester
// answers within a request on any audience size.
const simulateLimit = 200_000

// simulatePage is how many recipients one query reads.
const simulatePage = 2000

// WarmCap is a warming provider's cap on a day (DI from warmup, which
// imports routing).
type WarmCap struct {
	Cap      int64
	Day      int
	Days     int
	Overflow bool
}

// SimulateCampaign runs the next broadcast's audience through the routing.
func (s *Service) SimulateCampaign(ctx context.Context, ws *store.Workspace, in RouteCampaignInput) (RouteCampaignResult, error) {
	out := RouteCampaignResult{Timezone: ws.Timezone, Lists: []string{}, Shares: []RouteCampaignShare{}, Conditions: []RouteCampaignCondition{}, Overflow: []RouteCampaignOverflow{}}
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
	counts := map[uuid.UUID]int64{}
	draftCounts := map[uuid.UUID]int64{}
	matched := map[string]*RouteCampaignCondition{}
	order := []string{}
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
			counts[d.ProviderID]++
			if d.Matched != "" {
				key := d.Matched + ":" + strings.Join(conditionValues(d.Rule, d.Matched), ",")
				if matched[key] == nil {
					matched[key] = &RouteCampaignCondition{Field: d.Matched, Values: conditionValues(d.Rule, d.Matched)}
					order = append(order, key)
				}
				matched[key].Count++
			}
			if draft != nil {
				dd, ok := draft.Resolve(rc)
				if ok {
					draftCounts[dd.ProviderID]++
					if dd.ProviderID != d.ProviderID {
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
			moved = scale(moved)
			out.Total, out.Sunset = total, sunset
		}
	}
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
		out.Shares = append(out.Shares, share(id, counts[id], saved))
	}
	for _, k := range order {
		out.Conditions = append(out.Conditions, *matched[k])
	}
	// Warm-up: a warming provider takes up to its cap on the send day; the
	// rest goes to a warm provider when overflow is on.
	at := s.now()
	if c.ScheduledAt != nil && c.ScheduledAt.After(at) {
		at = *c.ScheduledAt
	}
	if s.WarmCap != nil {
		for _, id := range orderedProviders(saved, counts) {
			wc, ok := s.WarmCap(ctx, id, at)
			if !ok || counts[id] <= wc.Cap {
				continue
			}
			to := ""
			for _, other := range orderedProviders(saved, counts) {
				if _, warming := s.WarmCap(ctx, other, at); other != id && !warming {
					to = byID[other].Name
					break
				}
			}
			if !wc.Overflow || to == "" {
				continue
			}
			out.Overflow = append(out.Overflow, RouteCampaignOverflow{ProviderID: id.String(), Provider: byID[id].Name, To: to, Count: counts[id] - wc.Cap, Cap: wc.Cap, Day: wc.Day,
				Date: at})
		}
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
