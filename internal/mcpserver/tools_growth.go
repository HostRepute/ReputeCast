package mcpserver

import (
	"context"
	"math"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/growth"
	"github.com/hostrepute/reputecast/internal/scoring"
	"github.com/hostrepute/reputecast/internal/store"
)

// Contact scoring and growth sources (audience-insights brief §6): agents
// read scores and bands with subscribers:view, like contacts, and Growth's
// totals. They never change scoring rules through these tools.

// GrowthReader returns Audience › Growth (internal/growth).
type GrowthReader interface {
	Page(ctx context.Context, ws *store.Workspace, in growth.GrowthQuery) (growth.GrowthPage, error)
}

type growthIn struct {
	Period string `json:"period,omitempty" jsonschema:"30d (default), 90d or 12m"`
	List   string `json:"list,omitempty" jsonschema:"a list name or id, to count its members instead of contacts"`
}

func round1(v float64) float64 { return math.Round(v*1000) / 10 }

// addGrowthTools registers get_growth_sources.
func (s *Service) addGrowthTools(srv *mcp.Server) {
	tool(s, srv, "get_growth_sources", authz.SubscribersView, func(ctx context.Context, c *agentCall, in growthIn) (any, entry, error) {
		e := entry{action: "growth.read", targetType: "growth", targetText: keep("target.growth", nil)}
		if s.Growth == nil {
			return map[string]any{"message": "Growth isn't available on this server."}, e, nil
		}
		q := growth.GrowthQuery{Period: in.Period}
		if in.Period != "" && in.Period != "30d" && in.Period != "90d" && in.Period != "12m" {
			q.Period = "30d"
		}
		if in.List != "" {
			if id, err := uuid.Parse(in.List); err == nil {
				q.List = id.String()
			} else if l, err := s.Q.McpListByName(ctx, store.McpListByNameParams{WorkspaceID: c.Workspace.ID, Name: in.List}); err == nil {
				q.List = l.String()
			}
		}
		ws := c.Workspace
		p, err := s.Growth.Page(ctx, &ws, q)
		if err != nil {
			return nil, e, err
		}
		rate := func(v *float64) any {
			if v == nil {
				return nil
			}
			return round1(*v)
		}
		sources := []map[string]any{}
		for _, r := range p.Sources {
			m := map[string]any{"source": growth.EnglishLabel(r.Source), "kind": r.Source.Kind, "left": r.Left, "contacts": r.Contacts}
			if r.Joined != nil {
				m["joined"], m["net"] = *r.Joined, *r.Net
			} else {
				m["joined"] = "counted apart (import)"
			}
			if r.TooFew != "" {
				m["rates"] = "too few to compare"
			} else {
				m["bounce_rate_pct"], m["complaint_rate_pct"], m["human_open_rate_pct"] = rate(r.BounceRate), rate(r.ComplaintRate), rate(r.HumanOpenRate)
				if r.BounceFlag != nil || r.ComplaintFlag != nil {
					m["flagged"] = true
				}
			}
			sources = append(sources, m)
		}
		cohorts := []map[string]any{}
		for _, ch := range p.Cohorts {
			m := map[string]any{"joined_in": ch.Month, "contacts": ch.Contacts}
			for _, x := range []struct {
				key string
				r   growth.GrowthRetention
			}{{"after_30_days_pct", ch.After30}, {"after_90_days_pct", ch.After90}} {
				if x.r.State == "done" {
					m[x.key] = rate(x.r.Rate)
				} else {
					m[x.key] = "in progress, complete " + x.r.CompleteOn
				}
			}
			cohorts = append(cohorts, m)
		}
		counts := "contacts"
		if p.List != nil {
			counts = "members of " + p.List.Name
			e.detail = p.List.Name
		}
		return map[string]any{
			"period": map[string]any{"from": p.Period.From, "to": p.Period.To}, "counts": counts,
			"joined": p.Tiles.Joined, "left": p.Tiles.Left, "net": p.Tiles.Net, "imports_counted_apart": p.Tiles.Imports,
			"workspace_rates": map[string]any{"bounce_pct": round1(p.Workspace.Bounce), "complaint_pct": round1(p.Workspace.Complaint)},
			"sources":         sources,
			"departures": map[string]any{"unsubscribed_from_every_list": p.Departures.Unsubscribed, "hard_bounced": p.Departures.HardBounced,
				"complained": p.Departures.Complained, "erased": p.Departures.Erased},
			"cohorts": cohorts,
		}, e, nil
	}, nil)
}

// scoresFor adds each contact's score and band to search results, while
// the workspace has scoring.
func (s *Service) scoresFor(ctx context.Context, wsID uuid.UUID, out []map[string]any, ids []uuid.UUID) {
	bands, ok, err := scoring.Bands(ctx, s.Q, wsID, ids)
	for i, id := range ids {
		if err != nil || !ok {
			out[i]["score"], out[i]["score_band"] = nil, nil
			continue
		}
		b := bands[id]
		out[i]["score"], out[i]["score_band"] = b.Score, b.Band
	}
}
