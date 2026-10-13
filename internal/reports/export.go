package reports

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audit"
	provstate "github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// ExportInput asks for an export of the report.
type ExportInput struct {
	// Format: csv returns the file; pdf records the export of the printable
	// report, which the browser saves as a PDF.
	Format string `json:"format" enum:"csv,pdf"`
}

// ReportExport is a file to download.
type ReportExport struct {
	Filename string
	Body     []byte
}

// Export writes the report as CSV (one section per table) or records a PDF
// export. Every export is in the audit log.
func (s *Service) Export(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, format string) (ReportExport, error) {
	rep, err := s.Report(ctx, ws, campaignID)
	if err != nil {
		return ReportExport{}, err
	}
	name := fileStem(rep.Campaign.Name)
	out := ReportExport{Filename: name + "-report." + format}
	if format == "csv" {
		out.Body = reportCSV(rep, provstate.Location(ws.Timezone))
	}
	err = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "report.export", TargetType: "campaign", TargetID: campaignID.String(),
		TargetName: rep.Campaign.Name, Metadata: map[string]any{"format": format}})
	return out, err
}

var nonFile = regexp.MustCompile(`[^a-z0-9]+`)

// fileStem is "october-product-update".
func fileStem(name string) string {
	s := strings.Trim(nonFile.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if s == "" {
		return "campaign"
	}
	if len(s) > 60 {
		s = strings.TrimRight(s[:60], "-")
	}
	return s
}

// csvSafe neutralizes values a spreadsheet would run as a formula.
func csvSafe(vals ...string) []string {
	out := make([]string, len(vals))
	for i, v := range vals {
		if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
			v = "'" + v
		}
		out[i] = v
	}
	return out
}

func pctText(v *float64, digits int) string {
	if v == nil {
		return ""
	}
	return strconv.FormatFloat(*v, 'f', digits, 64) + "%"
}

func num(n int64) string { return strconv.FormatInt(n, 10) }

// reportCSV writes every table of the report, one section after another.
func reportCSV(r CampaignReport, loc *time.Location) []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	row := func(vals ...string) { _ = w.Write(csvSafe(vals...)) }
	c := r.Campaign
	row("Campaign report", c.Name)
	if c.Audience != "" {
		row("Audience", c.Audience)
	}
	if c.StartedAt != nil {
		sent := c.StartedAt.In(loc).Format("Mon 2 Jan 2006 15:04")
		if c.CompletedAt != nil {
			sent += " to " + c.CompletedAt.In(loc).Format("15:04")
		}
		row("Sent", sent+" ("+loc.String()+")")
	}
	status := "results still coming in"
	if c.Final {
		status = "final"
	} else if c.FinalAt != nil {
		status += ", final " + c.FinalAt.In(loc).Format("Mon 2 Jan 15:04")
	}
	row("Results", status)
	row()

	h := r.Headline
	row("Metric", "Count", "Rate", "Rate of")
	row("Sent", num(h.Sent), "", "")
	row("Delivered", num(h.Delivered), pctText(h.DeliveredRate, 2), "sent")
	row("Human opens", num(h.HumanOpens), pctText(h.HumanOpenRate, 2), "delivered")
	row("Machine opens", num(h.MachineOpens), pctText(h.MachineOpenRate, 2), "delivered")
	row("Clicks", num(h.Clicks), pctText(h.ClickRate, 2), "delivered")
	row("Click-to-open", "", pctText(h.ClickToOpen, 2), "human opens")
	row("Unsubscribes", num(h.Unsubscribes), pctText(h.UnsubscribeRate, 2), "delivered")
	row("Complaints", num(h.Complaints), pctText(h.ComplaintRate, 2), "delivered")
	row("Bounces", num(h.Bounces), pctText(h.BounceRate, 2), "sent")
	row("Hard bounces", num(h.HardBounces), "", "")
	row("Soft bounces", num(h.SoftBounces), "", "")
	row("Not delivered or bounced", num(h.Undelivered), "", "")
	for _, p := range h.SentBy {
		row("Sent by "+p.Short, num(p.Count), "", "")
	}
	row()

	if len(r.Mailboxes) > 0 {
		row("Mailbox provider", "Sent", "Delivered", "Human opens", "Machine opens", "Clicks", "Bounces", "Complaints", "Unsubscribes")
		for _, m := range r.Mailboxes {
			row(m.Name, num(m.Sent), pctText(m.DeliveredRate, 2), pctText(m.HumanOpenRate, 2), pctText(m.MachineOpenRate, 2), pctText(m.ClickRate, 2),
				pctText(m.BounceRate, 2), pctText(m.ComplaintRate, 2), pctText(m.UnsubscribeRate, 2))
		}
		row()
	}
	if len(r.Providers.Columns) > 0 {
		row("Sending provider", "Sent", "Delivered", "Human opens", "Clicks", "Bounces", "Complaints", "Median time to delivery (s)", "Cost (USD)")
		for _, p := range r.Providers.Columns {
			median := ""
			if p.MedianDeliveryMs != nil {
				median = strconv.FormatFloat(float64(*p.MedianDeliveryMs)/1000, 'f', 1, 64)
			}
			row(p.Name, num(p.Sent), pctText(p.DeliveredRate, 2), pctText(p.HumanOpenRate, 2), pctText(p.ClickRate, 2), pctText(p.BounceRate, 2),
				pctText(p.ComplaintRate, 2), median, strconv.FormatFloat(p.Cost, 'f', 2, 64))
		}
		row()
	}
	if r.Engagement.Available {
		row("Hours after send", "Human opens (cumulative)", "Machine opens (cumulative)", "Clicks (cumulative)")
		for _, p := range r.Engagement.Points {
			row(strconv.Itoa(p.Hour), num(p.HumanOpens), num(p.MachineOpens), num(p.Clicks))
		}
		row()
	}
	if len(r.ClickMap.Links) > 0 {
		row("Link", "URL", "Clicks", "Unique clicks", "Share of clicks")
		for _, l := range r.ClickMap.Links {
			share := strconv.FormatFloat(l.Share, 'f', 1, 64) + "%"
			row(l.Label, l.URL, num(l.Clicks), num(l.UniqueClicks), share)
		}
		row()
	}
	if len(r.Bounces.Rows) > 0 {
		row("Bounce type", "Count", "Original codes")
		for _, b := range r.Bounces.Rows {
			codes := make([]string, 0, len(b.Codes))
			for _, c := range b.Codes {
				codes = append(codes, fmt.Sprintf("%s (%d)", c.CodeText(), c.Count))
			}
			row(b.Label, num(b.Count), strings.Join(codes, " · "))
		}
		row()
	}
	if len(r.Unsubscribes.Reasons) > 0 {
		row("Unsubscribe reason", "Count", "Share")
		for _, u := range r.Unsubscribes.Reasons {
			row(u.Label, num(u.Count), strconv.FormatFloat(u.Share, 'f', 1, 64)+"%")
		}
		row("Chose fewer emails instead", num(r.Unsubscribes.ChoseFewer), "")
		row()
	}
	if t := r.ABTest; t != nil {
		row("A/B variant", "Label", "Recipients", "Human opens", "Clicks", "Complaints", "Winner", "Got the winner after the test")
		for _, v := range t.Variants {
			winner := ""
			if v.Winner {
				winner = "yes"
			}
			row(v.Key, v.Label, num(v.Recipients), pctText(v.HumanOpenRate, 2), pctText(v.ClickRate, 2), pctText(v.ComplaintRate, 2), winner, num(v.Remaining))
		}
		if t.PValue != nil {
			row("Two-proportion z-test", "metric "+strings.ReplaceAll(t.Metric, "_", " "), "p = "+strconv.FormatFloat(*t.PValue, 'f', 4, 64),
				"lift "+strconv.FormatFloat(*t.Lift, 'f', 1, 64)+" points")
		}
	}
	w.Flush()
	return buf.Bytes()
}
