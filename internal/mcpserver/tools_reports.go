package mcpserver

import (
	"context"
	"math"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/reports"
	"github.com/hostrepute/reputecast/internal/store"
)

// Reports for agents (review AN-14): the same report the API and the app
// show, behind the same permissions, with the same honest numbers (human,
// privacy-protected and scanner opens apart, complaints only where they
// are reported, the A/B decision with its confidence).

// ReportReader reads campaign reports and makes read-only links to them
// (internal/reports).
type ReportReader interface {
	Report(ctx context.Context, ws *store.Workspace, id uuid.UUID) (reports.CampaignReport, error)
	CreateShare(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, in reports.ShareInput) (reports.ReportShareLink, error)
}

type shareReportIn struct {
	Campaign      string `json:"campaign" jsonschema:"the campaign's name (exact) or id"`
	ExpiresInDays int    `json:"expires_in_days,omitempty" jsonschema:"how long the link works: 1 to 90 days (30 by default)"`
	IncludeCost   bool   `json:"include_cost,omitempty" jsonschema:"also show the cost per sending provider"`
	IncludeWarmup bool   `json:"include_warmup,omitempty" jsonschema:"also show each sending provider's warm-up day"`
	IncludeEmail  bool   `json:"include_email,omitempty" jsonschema:"also show the email as sent, in the click map"`
}

// r1 and r2 round a rate as the report shows it: one decimal, or two for
// complaint, unsubscribe and overall bounce rates. A rate that can't be
// worked out stays empty.
func r1(p *float64) *float64 { return roundTo(p, 10) }
func r2(p *float64) *float64 { return roundTo(p, 100) }

func roundTo(p *float64, f float64) *float64 {
	if p == nil {
		return nil
	}
	v := math.Round(*p*f) / f
	return &v
}

// reportOut is a report as an agent reads it.
func reportOut(r reports.CampaignReport) map[string]any {
	h := r.Headline
	out := map[string]any{
		"campaign": r.Campaign.Name, "status": r.Campaign.Status, "final": r.Campaign.Final,
		"sent": h.Sent, "delivered": h.Delivered, "delivered_pct": r1(h.DeliveredRate),
		"human_opens": h.HumanOpens, "human_opens_pct_of_delivered": r1(h.HumanOpenRate),
		"measurable": h.Measurable, "human_opens_pct_of_measurable": r1(h.HumanOpenRateMeasurable),
		"privacy_protected_opens": h.PrivacyOpens, "privacy_protected_pct": r1(h.PrivacyOpenRate), "scanner_opens_left_out": h.MachineOpens,
		"human_clickers": h.Clicks, "click_pct": r1(h.ClickRate), "click_to_open_pct": r1(h.ClickToOpen), "bot_clicks_left_out": h.BotClicks,
		"unsubscribes": h.Unsubscribes, "unsubscribe_pct": r2(h.UnsubscribeRate),
		"complaints": h.Complaints, "complaint_pct_where_reported": r2(h.ComplaintRateReported), "complaints_not_reported_by": h.NotReportedBy,
		"bounces": h.Bounces, "bounce_pct": r2(h.BounceRate), "undelivered": h.Undelivered,
		"note": "Human opens leave out privacy-protected opens (Apple Mail opens every message on delivery, so whether anyone read it can't be measured) and scanner opens; the measurable rate is human opens of the recipients whose opens can be measured. Gmail and iCloud don't report complaints.",
	}
	if len(r.Comparisons.Tiles) > 0 {
		cmp := []map[string]any{}
		for _, t := range r.Comparisons.Tiles {
			cmp = append(cmp, map[string]any{"metric": t.Metric, "change": t.Change, "unit": t.Unit, "median": t.Median, "reads_as": t.Tone})
		}
		out["vs_last_issues"] = map[string]any{"issues": r.Comparisons.Issues, "tiles": cmp}
	}
	boxes := []map[string]any{}
	for _, m := range r.Mailboxes {
		b := map[string]any{"mailbox_provider": m.Name, "sent": m.Sent, "delivered_pct": r1(m.DeliveredRate), "human_opens_pct_of_measurable": r1(m.HumanOpenRateMeasurable),
			"privacy_protected_pct": r1(m.PrivacyOpenRate), "click_pct": r1(m.ClickRate), "bounce_pct": r1(m.BounceRate), "unsubscribe_pct": r2(m.UnsubscribeRate)}
		if m.ComplaintRateNA != "" {
			b["complaint_pct"] = "n/a: doesn't report complaints"
		} else {
			b["complaint_pct"] = r2(m.ComplaintRate)
		}
		boxes = append(boxes, b)
	}
	out["by_mailbox_provider"] = boxes
	provs := []map[string]any{}
	for _, p := range r.Providers.Columns {
		provs = append(provs, map[string]any{"sending_provider": p.Name, "sent": p.Sent, "delivered_pct": r1(p.DeliveredRate),
			"human_opens_pct_of_measurable": r1(p.HumanOpenRateMeasurable), "bounce_pct": r1(p.BounceRate), "complaint_pct": r2(p.ComplaintRate)})
	}
	out["by_sending_provider"] = provs
	links := []map[string]any{}
	add := func(variant string, ls []reports.ReportLink) {
		for _, l := range ls {
			if l.UniqueClicks > 0 || l.BotClicks > 0 {
				x := map[string]any{"link": l.Label, "url": l.URL, "people_who_clicked": l.UniqueClicks, "share_pct": l.Share, "bot_clicks": l.BotClicks}
				if variant != "" {
					// A test on content: the share is of the variant's own email's clicks.
					x["variant"] = variant
				}
				links = append(links, x)
			}
		}
	}
	if r.ClickMap.PerVariant {
		for _, v := range r.ClickMap.Variants {
			add(v.Key, v.Links)
		}
	} else {
		add("", r.ClickMap.Links)
	}
	out["links"] = links
	if t := r.ABTest; t != nil {
		ab := map[string]any{"kind": t.Kind, "metric": t.Metric, "state": t.State, "outcome": t.Outcome, "winner": t.Winner, "default": t.Default,
			"confidence_pct": t.Confidence, "lift_points": t.Lift, "min_sample": t.MinSample, "threshold_pct": t.Threshold, "remaining": t.Remaining}
		vs := []map[string]any{}
		for _, v := range t.Variants {
			vs = append(vs, map[string]any{"variant": v.Key, "label": v.Label, "recipients": v.Recipients, "human_opens_pct_of_measurable": r1(v.HumanOpenRate),
				"click_pct": r1(v.ClickRate), "complaint_pct": r2(v.ComplaintRate)})
		}
		ab["variants"] = vs
		out["ab_test"] = ab
	}
	if g := r.Growth; g.Since != nil {
		out["growth"] = map[string]any{"new_subscribers": g.NewSubscribers, "from_forwards": g.Forwards, "from_referrals": g.Referrals, "from_the_web_post": g.WebPost,
			"web_views": g.WebViews}
	}
	return out
}

func (s *Service) addReportTools(srv *mcp.Server) {
	if s.Reports == nil {
		return
	}
	tool(s, srv, "get_campaign_report", authz.ReportsView, func(ctx context.Context, c *agentCall, in CampaignRef) (any, entry, error) {
		e := entry{action: "report.read", targetType: "campaign", targetName: in.Campaign}
		cp, err := s.findCampaign(ctx, c.Workspace.ID, in.Campaign)
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName = cp.ID.String(), cp.Name
		r, err := s.Reports.Report(ctx, &c.Workspace, cp.ID)
		if err != nil {
			return nil, e, err
		}
		return reportOut(r), e, nil
	}, s.campaignTarget("report.read"))
	tool(s, srv, "share_campaign_report", authz.ReportsShare, func(ctx context.Context, c *agentCall, in shareReportIn) (any, entry, error) {
		e := entry{action: "report_share.create", targetType: "campaign", targetName: in.Campaign}
		cp, err := s.findCampaign(ctx, c.Workspace.ID, in.Campaign)
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName = cp.ID.String(), cp.Name
		link, err := s.Reports.CreateShare(ctx, &c.Workspace, cp.ID, reports.ShareInput{ExpiresInDays: in.ExpiresInDays, IncludeCost: in.IncludeCost,
			IncludeWarmup: in.IncludeWarmup, IncludeEmail: in.IncludeEmail})
		if err != nil {
			return nil, e, err
		}
		e.detailText = keep("detail.reportShared", i18n.Args{"campaign": cp.Name})
		return map[string]any{"url": link.URL, "expires_at": link.ExpiresAt, "includes": link.Includes,
			"note": "Anyone with the link sees the report's numbers without signing in, until it expires or is revoked; never the AI summary."}, e, nil
	}, func(ctx context.Context, c *agentCall, in shareReportIn) entry {
		return s.campaignTarget("report_share.create")(ctx, c, CampaignRef{Campaign: in.Campaign})
	})
}
