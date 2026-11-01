package sponsors

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/store"
)

// SponsorsFile is a CSV download.
type SponsorsFile struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	CacheControl       string `header:"Cache-Control"`
	Body               []byte
}

func csvDownload(name string, body []byte) *SponsorsFile {
	return &SponsorsFile{ContentType: "text/csv; charset=utf-8", ContentDisposition: `attachment; filename="` + name + `"`, CacheControl: "no-store", Body: body}
}

// csvMoney is minor units as "1250.00".
func csvMoney(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

// day is a date as 2006-01-02, or "" when unset.
func day(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.DateOnly)
}

// ExportBookings writes every booking of the workspace, newest issue first,
// as CSV (review 2026-11 EA-2b): its slot, issue, sponsor, price, payment
// and what it reached. Sponsors' contact addresses aren't in it. The export
// is audited, like Earn's other exports.
func (s *Service) ExportBookings(ctx context.Context, ws *store.Workspace) ([]byte, int, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT sl.name, b.issue_number, b.issue_on, sp.name, b.status, b.price_cents, b.currency, b.payment_status, b.paid_on, coalesce(b.paid_cents, 0)::bigint,
		       b.sent_at, coalesce(c.name, ''), coalesce((SELECT sum(cc.sent) FROM campaign_counters cc WHERE cc.campaign_id = b.campaign_id), 0)::bigint
		FROM sponsor_bookings b
		JOIN sponsor_slots sl ON sl.id = b.slot_id
		JOIN sponsors sp ON sp.id = b.sponsor_id
		LEFT JOIN campaigns c ON c.id = b.campaign_id
		WHERE b.workspace_id = $1
		ORDER BY b.issue_on DESC, sl.name, b.id`, ws.ID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"slot", "issue", "issue_on", "sponsor", "status", "price", "currency", "payment", "paid_on", "paid", "sent_at", "campaign", "sent_to"})
	n := 0
	for rows.Next() {
		var slot, sponsor, status, currency, payment, campaign string
		var issue int32
		var issueOn time.Time
		var price, paid, sentTo int64
		var paidOn, sentAt *time.Time
		if err := rows.Scan(&slot, &issue, &issueOn, &sponsor, &status, &price, &currency, &payment, &paidOn, &paid, &sentAt, &campaign, &sentTo); err != nil {
			return nil, 0, err
		}
		sent := ""
		if sentAt != nil {
			sent = sentAt.UTC().Format(time.RFC3339)
		}
		_ = w.Write([]string{csvSafe(slot), strconv.Itoa(int(issue)), issueOn.Format(time.DateOnly), csvSafe(sponsor), status, csvMoney(price), currency, payment,
			day(paidOn), csvMoney(paid), sent, csvSafe(campaign), strconv.FormatInt(sentTo, 10)})
		n++
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	w.Flush()
	err = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "sponsor.bookings_exported", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
		Metadata: map[string]any{"rows": n, "summary": fmt.Sprintf("Exported %d sponsor bookings as CSV", n)}})
	return buf.Bytes(), n, err
}

// ExportReport writes a sent booking's report as CSV (review 2026-11
// EA-2b): the numbers, then each link's clicks. Only counts, no people.
func (s *Service) ExportReport(ctx context.Context, ws *store.Workspace, id uuid.UUID) ([]byte, SponsorReport, error) {
	r, err := s.Report(ctx, ws, id)
	if err != nil {
		return nil, r, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	sent := ""
	if r.SentAt != nil {
		sent = r.SentAt.UTC().Format(time.RFC3339)
	}
	pct := func(f float64) string { return strconv.FormatFloat(f*100, 'f', 1, 64) }
	for _, row := range [][]string{
		{"metric", "value"},
		{"sponsor", csvSafe(r.Sponsor)},
		{"slot", csvSafe(r.SlotName)},
		{"issue", strconv.Itoa(r.Issue)},
		{"issue_on", r.IssueOn.Format(time.DateOnly)},
		{"list", csvSafe(r.ListName)},
		{"sent_at", sent},
		{"final", strconv.FormatBool(r.Final)},
		{"delivered", strconv.FormatInt(r.Delivered, 10)},
		{"human_opens", strconv.FormatInt(r.HumanOpens, 10)},
		{"human_open_rate_percent", pct(r.HumanOpenRate)},
		{"unique_clickers", strconv.FormatInt(r.UniqueClickers, 10)},
		{"clicks", strconv.FormatInt(r.Clicks, 10)},
		{"click_rate_percent", pct(r.ClickRate)},
		{"click_to_open_percent", pct(r.ClickToOpen)},
	} {
		_ = w.Write(row)
	}
	_ = w.Write(nil)
	_ = w.Write([]string{"link", "kind", "clicks", "share_percent"})
	for _, l := range r.Links {
		_ = w.Write([]string{csvSafe(l.Label), l.Kind, strconv.FormatInt(l.Clicks, 10), pct(l.Share)})
	}
	w.Flush()
	err = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "sponsor.report_exported", TargetType: "sponsor_booking", TargetID: id.String(),
		TargetName: fmt.Sprintf("%s · #%d", r.Sponsor, r.Issue), Metadata: map[string]any{"summary": "Exported a sponsor report as CSV"}})
	return buf.Bytes(), r, err
}

// csvSafe keeps a spreadsheet from reading a cell as a formula.
func csvSafe(v string) string {
	if v != "" && (v[0] == '=' || v[0] == '+' || v[0] == '-' || v[0] == '@' || v[0] == '\t' || v[0] == '\r') {
		return "'" + v
	}
	return v
}
