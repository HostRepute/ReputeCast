package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/growth"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/recommendations"
	"github.com/hostrepute/reputecast/internal/store"
)

// Grow's growth sources (brief §3.10): readers who joined through a
// referral link are "Referral", partner signups "Partner · Ops Weekly", in
// Audience › Growth like any other source.
func init() {
	growth.RegisterKind(growth.Kind{Key: "referral", Order: 12})
	growth.RegisterKind(growth.Kind{Key: "partner", Order: 14})
	growth.RegisterAttributor(func(contactSource, consentSource string) (string, string, bool) {
		if id, ok := strings.CutPrefix(consentSource, "partner:"); ok {
			return "partner", id, true
		}
		if contactSource == "referral" || strings.HasPrefix(contactSource, "referral:") {
			return "referral", "", true
		}
		return "", "", false
	})
	growth.RegisterNamer("referral", func(context.Context, *store.Queries, *store.Workspace) (map[string]string, error) {
		return map[string]string{"": "Referral"}, nil
	})
	growth.RegisterNamer("partner", func(ctx context.Context, q *store.Queries, ws *store.Workspace) (map[string]string, error) {
		rows, err := q.PartnerLinkNames(ctx, ws.ID)
		if err != nil {
			return nil, err
		}
		out := map[string]string{}
		for _, r := range rows {
			out[r.ID.String()] = "Partner · " + r.RemoteName
		}
		return out, nil
	})
}

// growReadiness is Send Readiness's view of Grow's blocks: who the Referral
// block shows to, and what the Recommendations block shows.
type growReadiness struct{ a *App }

// ReadinessRows implements readiness.GrowChecker.
func (g growReadiness) ReadinessRows(ctx context.Context, ws *store.Workspace, campaign uuid.UUID, html string, byList map[uuid.UUID]int64, total int64) []readiness.GrowRow {
	var out []readiness.GrowRow
	if c, err := g.a.referrals().CheckBlock(ctx, ws, html, byList, total); err == nil && c.Present {
		status, title, detail, ack := c.ReadinessText()
		row := readiness.GrowRow{Key: "content.referral_block", Status: status, Title: title, Detail: detail, Label: "Referral block", Ack: ack,
			Short: "referral block"}
		switch {
		case c.Off:
			row.Links = []readiness.GrowLink{{Label: "Remove the block", Href: "/campaigns/" + campaign.String() + "/edit"},
				{Label: "Referral settings", Href: "/grow/referrals/settings?program=" + c.ProgramID}}
		case status != "pass":
			row.Links = []readiness.GrowLink{{Label: "Set up referrals", Href: "/grow/referrals"}}
		}
		out = append(out, row)
	}
	// A referral token typed outside the block prints nothing for readers
	// outside the program.
	if c, err := g.a.referrals().CheckTokens(ctx, ws, html, byList, total); err == nil {
		if title, detail, ack, ok := c.ReadinessText(); ok {
			out = append(out, readiness.GrowRow{Key: "content.referral_tokens", Status: "warn", Title: title, Detail: detail, Label: "Referral tokens", Ack: ack,
				Short: "empty referral tokens", Links: []readiness.GrowLink{{Label: "Open in the composer", Href: "/campaigns/" + campaign.String() + "/edit"}}})
		}
	}
	if c, err := g.a.recommendations().CheckBlock(ctx, ws, html); err == nil && c.Present {
		row := readiness.GrowRow{Key: "content.recommendations_block", Status: "pass", Label: "Recommendations block", Short: "recommendations block",
			Links: []readiness.GrowLink{{Label: "Open recommendations", Href: "/grow/recommendations"}}}
		if c.Shows == 0 {
			row.Status = "warn"
			row.Title = "No recommendation can show right now. The Recommendations block will be hidden."
			row.Ack = "I understand the Recommendations block won't show."
		} else {
			unit := "recommendations show"
			if c.Shows == 1 {
				unit = "recommendation shows"
			}
			row.Title = fmt.Sprintf("Recommendations block: %d %s", c.Shows, unit)
		}
		var notes []string
		for _, h := range c.Hidden {
			if h.State == "not_in_emails" {
				continue
			}
			notes = append(notes, fmt.Sprintf("%s is hidden (%s).", h.Name, recommendations.HiddenReason(h.State)))
		}
		row.Detail = strings.Join(notes, " ")
		out = append(out, row)
	}
	return out
}
