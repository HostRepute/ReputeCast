package growth

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/store"
)

// Export writes the Growth tables as CSV: totals only, never an address.
// Each export is in the audit log, as report exports are.
func (s *Service) Export(ctx context.Context, ws *store.Workspace, in GrowthQuery) ([]byte, string, error) {
	p, err := s.Page(ctx, ws, in)
	if err != nil {
		return nil, "", err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	pct := func(v *float64) string {
		if v == nil {
			return ""
		}
		return strconv.FormatFloat(*v*100, 'f', 2, 64)
	}
	num := func(v *int64) string {
		if v == nil {
			return ""
		}
		return strconv.FormatInt(*v, 10)
	}
	scope := "contacts"
	if p.List != nil {
		scope = "members of " + p.List.Name
	}
	_ = w.Write([]string{"section", "period", p.Period.From, p.Period.To, scope})
	_ = w.Write([]string{"source", "kind", "joined", "left", "net", "contacts", "sent", "delivered", "bounce_rate_pct", "complaint_rate_pct", "human_open_rate_pct", "too_few"})
	for _, r := range p.Sources {
		_ = w.Write([]string{EnglishLabel(r.Source), r.Source.Kind, num(r.Joined), strconv.FormatInt(r.Left, 10), num(r.Net), strconv.FormatInt(r.Contacts, 10),
			strconv.FormatInt(r.Sent, 10), strconv.FormatInt(r.Delivered, 10), pct(r.BounceRate), pct(r.ComplaintRate), pct(r.HumanOpenRate), r.TooFew})
	}
	_ = w.Write([]string{"Total", "", strconv.FormatInt(p.Total.Joined, 10), strconv.FormatInt(p.Total.Left, 10), strconv.FormatInt(p.Total.Net, 10)})
	_ = w.Write(nil)
	_ = w.Write([]string{"departure", "count"})
	for _, d := range [][2]any{{"unsubscribed_from_every_list", p.Departures.Unsubscribed}, {"hard_bounced", p.Departures.HardBounced},
		{"complained", p.Departures.Complained}, {"erased", p.Departures.Erased}} {
		_ = w.Write([]string{d[0].(string), fmt.Sprint(d[1])})
	}
	_ = w.Write(nil)
	_ = w.Write([]string{"utm_source", "utm_medium", "utm_campaign", "joins", "human_open_rate_pct"})
	if p.UTM.NotCaptured > 0 {
		_ = w.Write([]string{"(not captured)", "", "", strconv.FormatInt(p.UTM.NotCaptured, 10), pct(p.UTM.NotCapRate)})
	}
	_ = w.Write([]string{"(none)", "", "", strconv.FormatInt(p.UTM.None.Joins, 10), pct(p.UTM.None.HumanOpenRate)})
	for _, r := range p.UTM.Rows {
		_ = w.Write([]string{r.Source, r.Medium, r.Campaign, strconv.FormatInt(r.Joins, 10), pct(r.HumanOpenRate)})
	}
	if p.UTM.Other.Count > 0 {
		_ = w.Write([]string{fmt.Sprintf("(other, %d combinations)", p.UTM.Other.Count), "", "", strconv.FormatInt(p.UTM.Other.Joins, 10), pct(p.UTM.Other.HumanOpenRate)})
	}
	_ = w.Write(nil)
	_ = w.Write([]string{"referring_page", "joins"})
	for _, r := range p.Referrers.Rows {
		_ = w.Write([]string{r.Page, strconv.FormatInt(r.Joins, 10)})
	}
	_ = w.Write([]string{"(none)", strconv.FormatInt(p.Referrers.None, 10)})
	if p.Referrers.Other.Count > 0 {
		_ = w.Write([]string{fmt.Sprintf("(other, %d pages)", p.Referrers.Other.Count), strconv.FormatInt(p.Referrers.Other.Joins, 10)})
	}
	_ = w.Write(nil)
	_ = w.Write([]string{"cohort", "contacts", "still_subscribed_30d_pct", "still_subscribed_90d_pct"})
	ret := func(r GrowthRetention) string {
		if r.State == "done" {
			return pct(r.Rate)
		}
		return "in progress"
	}
	for _, c := range p.Cohorts {
		_ = w.Write([]string{c.Month, strconv.FormatInt(c.Contacts, 10), ret(c.After30), ret(c.After90)})
	}
	w.Flush()
	name := fmt.Sprintf("growth-%s-%s.csv", p.Period.From, p.Period.To)
	err = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "growth.exported", TargetType: "growth", TargetID: ws.ID.String(), TargetName: "Audience · Growth",
		Metadata: map[string]any{"summary": fmt.Sprintf("%s – %s · %s · %d sources", p.Period.From, p.Period.To, scope, len(p.Sources)),
			"period": p.Period, "list": p.List, "include_imports": p.IncludeImports, "rows": len(p.Sources)}})
	return buf.Bytes(), name, err
}
