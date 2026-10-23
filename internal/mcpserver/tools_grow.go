package mcpserver

import (
	"context"
	"fmt"
	"net/netip"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/recommendations"
	"github.com/hostrepute/reputecast/internal/referrals"
	"github.com/hostrepute/reputecast/internal/store"
)

// Grow (design expansion phase 5, brief §3.11): agents read referral
// results, referrers' counts, the review queue with addresses masked, and
// the recommendations. They can propose a review decision for a person;
// they never decide, change rewards or touch partner links.

// GrowReader reads Grow › Referrals (internal/referrals).
type GrowReader interface {
	Programs(ctx context.Context, ws *store.Workspace) (referrals.ReferralProgramList, error)
	Results(ctx context.Context, ws *store.Workspace, programID uuid.UUID, period, grouping string) (referrals.ReferralResults, error)
	Referrers(ctx context.Context, ws *store.Workspace, programID uuid.UUID, p referrals.ReferrerParams) (referrals.ReferralReferrerList, error)
	Review(ctx context.Context, ws *store.Workspace, programID uuid.UUID, signal string, link *uuid.UUID) (referrals.ReferralReviewQueue, error)
}

// RecommendationsReader reads Grow › Recommendations.
type RecommendationsReader interface {
	View(ctx context.Context, ws *store.Workspace) (recommendations.RecommendationsView, error)
}

type growProgramIn struct {
	List   string `json:"list,omitempty" jsonschema:"the program's list name (default: the first program)"`
	Period string `json:"period,omitempty" jsonschema:"7d, 30d or all (default all)"`
}

type growReferrersIn struct {
	List  string `json:"list,omitempty" jsonschema:"the program's list name (default: the first program)"`
	Query string `json:"query,omitempty" jsonschema:"a name to look for"`
	Held  bool   `json:"held,omitempty" jsonschema:"only referrers with referrals held for review"`
}

type growDecisionIn struct {
	List     string `json:"list,omitempty" jsonschema:"the program's list name (default: the first program)"`
	Referrer string `json:"referrer" jsonschema:"the referrer's name, as list_held_referrals shows it"`
	Decision string `json:"decision" jsonschema:"count, dont or dont_pause"`
	Reason   string `json:"reason,omitempty" jsonschema:"why, for the person who decides"`
}

// growProgram picks a program by its list's name, or the first; ok is
// false when there is none.
func (s *Service) growProgram(ctx context.Context, ws *store.Workspace, list string) (id uuid.UUID, name string, ok bool, err error) {
	progs, err := s.Grow.Programs(ctx, ws)
	if err != nil {
		return uuid.Nil, "", false, err
	}
	for _, p := range progs.Items {
		if list == "" || strings.EqualFold(p.ListName, list) {
			id, err := uuid.Parse(p.ID)
			return id, p.ListName, err == nil, err
		}
	}
	return uuid.Nil, "", false, nil
}

// noProgram is the answer when a list has no referral program.
func noProgram(list string) map[string]any {
	if list == "" {
		return map[string]any{"message": "This workspace has no referral program."}
	}
	return map[string]any{"message": fmt.Sprintf("There's no referral program on %q.", list)}
}

// maskIP keeps an address's network: "203.0.113.x".
func maskIP(raw string) string {
	a, err := netip.ParseAddr(raw)
	if err != nil {
		return ""
	}
	if a.Is4() {
		b := a.As4()
		return fmt.Sprintf("%d.%d.%d.x", b[0], b[1], b[2])
	}
	p, _ := a.Prefix(48)
	return p.String()
}

// addGrowTools registers Grow's tools.
func (s *Service) addGrowTools(srv *mcp.Server) {
	tool(s, srv, "get_referral_results", authz.GrowView, func(ctx context.Context, c *agentCall, in growProgramIn) (any, entry, error) {
		e := entry{action: "referral.results_read", targetType: "referral_program", targetText: keep("target.referralResults", nil)}
		if s.Grow == nil {
			return map[string]any{"message": "Grow isn't available on this server."}, e, nil
		}
		ws := c.Workspace
		id, list, ok, err := s.growProgram(ctx, &ws, in.List)
		if err != nil {
			return nil, e, err
		}
		if !ok {
			return noProgram(in.List), e, nil
		}
		e.targetID, e.targetName = id.String(), list
		period := in.Period
		if period != "7d" && period != "30d" {
			period = "all"
		}
		r, err := s.Grow.Results(ctx, &ws, id, period, "week")
		if err != nil {
			return nil, e, err
		}
		top := []map[string]any{}
		for _, t := range r.Top {
			top = append(top, map[string]any{"referrer": t.Name, "counted": t.Counted, "in_hold": t.InHold, "held": t.Held})
		}
		channels := []map[string]any{}
		for _, ch := range r.Channels {
			channels = append(channels, map[string]any{"channel": ch.Channel, "visits": ch.Visits, "signups": ch.Signups})
		}
		return map[string]any{"list": list, "period": period, "funnel": r.Funnel, "channels": channels, "top_referrers": top, "rewards": r.Rewards}, e, nil
	}, nil)

	tool(s, srv, "list_referrers", authz.GrowView, func(ctx context.Context, c *agentCall, in growReferrersIn) (any, entry, error) {
		e := entry{action: "referral.referrers_read", targetType: "referral_program", targetText: keep("target.referrers", nil)}
		if s.Grow == nil {
			return map[string]any{"message": "Grow isn't available on this server."}, e, nil
		}
		ws := c.Workspace
		id, list, ok, err := s.growProgram(ctx, &ws, in.List)
		if err != nil {
			return nil, e, err
		}
		if !ok {
			return noProgram(in.List), e, nil
		}
		e.targetID, e.targetName = id.String(), list
		l, err := s.Grow.Referrers(ctx, &ws, id, referrals.ReferrerParams{Query: in.Query, Held: in.Held, Page: 1, PageSize: 50})
		if err != nil {
			return nil, e, err
		}
		items := []map[string]any{}
		for _, r := range l.Items {
			// Counts and milestones only: addresses stay in ReputeCast.
			items = append(items, map[string]any{"referrer": r.Name, "counted": r.Counted, "in_hold": r.InHold, "held": r.Held, "milestones": r.Milestones,
				"last_referral": r.LastReferral, "link_status": r.LinkStatus})
		}
		return map[string]any{"list": list, "total": l.Total, "referrers": items}, e, nil
	}, nil)

	tool(s, srv, "list_held_referrals", authz.GrowView, func(ctx context.Context, c *agentCall, in growProgramIn) (any, entry, error) {
		e := entry{action: "referral.review_read", targetType: "referral_program", targetText: keep("target.reviewQueue", nil)}
		if s.Grow == nil {
			return map[string]any{"message": "Grow isn't available on this server."}, e, nil
		}
		ws := c.Workspace
		id, list, ok, err := s.growProgram(ctx, &ws, in.List)
		if err != nil {
			return nil, e, err
		}
		if !ok {
			return noProgram(in.List), e, nil
		}
		e.targetID, e.targetName = id.String(), list
		q, err := s.Grow.Review(ctx, &ws, id, "", nil)
		if err != nil {
			return nil, e, err
		}
		groups := []map[string]any{}
		for _, g := range q.Groups {
			held := []map[string]any{}
			for i, it := range g.Items {
				held = append(held, map[string]any{"number": i + 1, "signed_up_at": it.SignedUpAt, "confirmed_at": it.ConfirmedAt,
					"network": maskIP(it.IP), "network_name": it.Network.Name, "signals": it.Signals})
			}
			kinds := []string{}
			for _, sg := range g.Signals {
				kinds = append(kinds, sg.Kind)
			}
			groups = append(groups, map[string]any{"referrer": g.Referrer.Name, "counted": g.Referrer.Counted, "held": len(g.Items),
				"suggestion": g.Suggestion, "signals": kinds, "referrals": held})
		}
		return map[string]any{"list": list, "held": q.Held, "groups": groups,
			"note": "Addresses are masked to their network. Use propose_referral_decision to ask a person to decide."}, e, nil
	}, nil)

	tool(s, srv, "propose_referral_decision", authz.GrowView, func(ctx context.Context, c *agentCall, in growDecisionIn) (any, entry, error) {
		e := entry{action: "referral.decision_proposed", targetType: "referral_program", targetName: in.Referrer}
		verb := map[string]string{"count": "count", "dont": "not count", "dont_pause": "not count, and pause the link of"}[in.Decision]
		if verb == "" {
			return map[string]any{"message": "decision is count, dont or dont_pause"}, e, nil
		}
		reason := strings.TrimSpace(in.Reason)
		e.detailText = keep("detail.referralDecision", i18n.Args{"decision": in.Decision, "referrer": in.Referrer, "hasReason": yesNo(reason != ""), "reason": reason})
		s.propose(ctx, c, "decide_held_referrals", authz.GrowReview, entry{targetType: "referral_program", targetName: in.Referrer})
		return map[string]any{"proposed": true, "referrer": in.Referrer, "decision": in.Decision,
			"message": fmt.Sprintf("Deciding needs a person with grow:review: the decision to %s %s's held referrals was proposed in ReputeCast.", verb, in.Referrer)}, e, nil
	}, nil)

	tool(s, srv, "list_recommendations", authz.GrowView, func(ctx context.Context, c *agentCall, _ struct{}) (any, entry, error) {
		e := entry{action: "recommendation.list_read", targetType: "recommendations", targetText: keep("target.recommendations", nil)}
		if s.Recommendations == nil {
			return map[string]any{"message": "Recommendations aren't available on this server."}, e, nil
		}
		ws := c.Workspace
		v, err := s.Recommendations.View(ctx, &ws)
		if err != nil {
			return nil, e, err
		}
		items := []map[string]any{}
		for _, it := range v.Items {
			items = append(items, map[string]any{"name": it.Name, "kind": it.Kind, "state": it.State, "hidden": it.Hidden, "shows_on_confirmation_page": it.ShowConfirm,
				"shows_in_emails": it.ShowBlock, "shown_30d": it.Shown, "clicks_30d": it.Clicks, "requests_30d": it.Requests, "confirmed_30d": it.Confirmed})
		}
		partners := []map[string]any{}
		for _, p := range v.RecommendingYou {
			partners = append(partners, map[string]any{"partner": p.Name, "since": p.Since, "requests_30d": p.Requests, "confirmed_30d": p.Confirmed,
				"daily_limit": p.DailyLimit, "state": p.State})
		}
		return map[string]any{"you_recommend": items, "recommending_you": partners}, e, nil
	}, nil)
}
