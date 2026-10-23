package app

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/growth"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/referrals"
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
// block shows to, and what the Recommendations block shows. The rows' texts
// are Grow's kept messages, which Send Readiness writes in each reader's
// language.
type growReadiness struct{ a *App }

// growRow is a Referral row as Send Readiness keeps it.
func growRow(key string, r referrals.ReadinessRow) readiness.GrowRow {
	row := readiness.GrowRow{Key: key, Status: r.Status, Title: readiness.Kept(r.Title), Detail: readiness.Kept(r.Detail), Label: readiness.Kept(r.Label),
		Short: readiness.Kept(r.Short), Ack: readiness.Kept(r.Ack)}
	for _, l := range r.Links {
		row.Links = append(row.Links, readiness.GrowLink{Label: readiness.Kept(l.Label), Href: l.Href})
	}
	return row
}

// ReadinessRows implements readiness.GrowChecker.
func (g growReadiness) ReadinessRows(ctx context.Context, ws *store.Workspace, campaign uuid.UUID, html string, byList map[uuid.UUID]int64, total int64) []readiness.GrowRow {
	var out []readiness.GrowRow
	if c, err := g.a.referrals().CheckBlock(ctx, ws, html, byList, total); err == nil && c.Present {
		out = append(out, growRow("content.referral_block", c.ReadinessRow(campaign.String(), func(t time.Time) any { return readiness.ArgDayMonthTime(t) })))
	}
	// A referral token typed outside the block prints nothing for readers
	// outside the program.
	if c, err := g.a.referrals().CheckTokens(ctx, ws, html, byList, total); err == nil {
		if r, ok := c.ReadinessRow(campaign.String()); ok {
			out = append(out, growRow("content.referral_tokens", r))
		}
	}
	if c, err := g.a.recommendations().CheckBlock(ctx, ws, html); err == nil && c.Present {
		r := c.ReadinessRow()
		row := readiness.GrowRow{Key: "content.recommendations_block", Status: r.Status, Title: readiness.Kept(r.Title), Label: readiness.Kept(r.Label),
			Short: readiness.Kept(r.Short), Ack: readiness.Kept(r.Ack), Links: []readiness.GrowLink{{Label: readiness.Kept(r.Link), Href: "/grow/recommendations"}}}
		for _, d := range r.Detail {
			row.Detail = append(row.Detail, readiness.Kept(d)...)
		}
		out = append(out, row)
	}
	return out
}
