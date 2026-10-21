package texting

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// The report's Export CSV and Duplicate (SmsReport), and what MCP agents
// may read and draft.

// SmsCSV is a CSV download.
type SmsCSV struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	CacheControl       string `header:"Cache-Control"`
	Body               []byte
}

// fileSlug turns a campaign's name into a file name: "summit-8-days-to-go".
func fileSlug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	out := strings.TrimSuffix(b.String(), "-")
	if out == "" {
		return "sms-campaign"
	}
	if len(out) > 60 {
		out = strings.TrimSuffix(out[:60], "-")
	}
	return out
}

// ReportCSV is the report by country as CSV: each country's sender, texts,
// delivery, failures, clicks, opt-outs, skipped texts and cost. Counts
// only, no numbers or names, so it needs what the report needs; the
// download is logged like other exports.
func (s *Service) ReportCSV(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (*SmsCSV, error) {
	r, err := s.Report(ctx, ws, p, id)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"country", "sender", "sent", "delivered", "delivered_pct", "failed", "clicks", "click_pct", "opt_outs", "skipped", "skip_reason", "cost_usd"})
	f := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
	n := func(v int64) string { return strconv.FormatInt(v, 10) }
	for _, c := range r.Countries {
		_ = w.Write([]string{c.Country, c.Sender, n(c.Sent), n(c.Delivered), f(c.DeliveredPct), n(c.Failed), n(c.Clicks), f(c.ClickPct), n(c.OptOuts),
			n(c.Skipped), c.SkipReason, strconv.FormatFloat(c.Cost, 'f', 2, 64)})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	name := fileSlug(r.Name) + "-sms-report.csv"
	_ = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.report_exported", TargetType: "campaign", TargetID: r.CampaignID, TargetName: r.Name,
		Metadata: map[string]any{"channel": "sms", "format": "csv", "rows": len(r.Countries), "file": name}})
	return &SmsCSV{ContentType: "text/csv; charset=utf-8", ContentDisposition: fmt.Sprintf(`attachment; filename="%s"`, name), CacheControl: "no-store", Body: b.Bytes()}, nil
}

// Duplicate copies an SMS campaign's text and audience into a new draft
// ("Copy of …"), whatever its status, audited like any new campaign.
func (s *Service) Duplicate(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (SmsCampaign, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return SmsCampaign{}, err
	}
	name := "Copy of " + c.Name
	if len([]rune(name)) > 200 {
		name = string([]rune(name)[:200])
	}
	return s.create(ctx, ws, p, name, parseAudience(c.Audience), c.Body, &c.ID)
}
