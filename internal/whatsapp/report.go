package whatsapp

import (
	"bytes"
	"context"
	"encoding/csv"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/store"
)

// The WhatsApp report (brief §3.9): sent, delivered, read (only from people
// who share read receipts), failed with Meta's codes, button clicks
// (quick replies), replies, opt-outs and the cost Meta charges.

// WhatsAppFailure is a reason messages failed for.
type WhatsAppFailure struct {
	Reason string `json:"reason"`
	Code   string `json:"code"`
	Count  int64  `json:"count"`
}

// WhatsAppReportCountry is a country's part.
type WhatsAppReportCountry struct {
	Country   string  `json:"country"`
	Sent      int64   `json:"sent"`
	Delivered int64   `json:"delivered"`
	Read      int64   `json:"read"`
	Failed    int64   `json:"failed"`
	Cost      float64 `json:"cost"`
}

// WhatsAppReport is a WhatsApp campaign's report.
type WhatsAppReport struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Status   string     `json:"status" enum:"draft,approval,approved,scheduled,sending,paused,sent,cancelled,failed"`
	SentAt   *time.Time `json:"sentAt,omitempty"`
	SentBy   string     `json:"sentBy"`
	Template string     `json:"template"`
	Category string     `json:"category"`
	// Planned is everyone the send was for; Skipped those left out, by
	// reason; Queued those not sent yet (the next day's share of the
	// limit among them).
	Planned      int64                   `json:"planned"`
	Skipped      map[string]int64        `json:"skipped"`
	Queued       int64                   `json:"queued"`
	Held         int64                   `json:"held"`
	Sent         int64                   `json:"sent"`
	Delivered    int64                   `json:"delivered"`
	DeliveredPct float64                 `json:"deliveredPct"`
	Read         int64                   `json:"read"`
	ReadPct      float64                 `json:"readPct"`
	Failed       int64                   `json:"failed"`
	Failures     []WhatsAppFailure       `json:"failures"`
	Buttons      int64                   `json:"buttons"`
	Replies      int64                   `json:"replies"`
	OptOuts      int64                   `json:"optOuts"`
	Cost         float64                 `json:"cost"`
	Estimate     float64                 `json:"estimate"`
	ByCountry    []WhatsAppReportCountry `json:"byCountry"`
	RatesAsOf    time.Time               `json:"ratesAsOf"`
}

func pct(n, d int64) float64 {
	if d <= 0 {
		return 0
	}
	return float64(int64(float64(n)*1000/float64(d)+0.5)) / 10
}

// CampaignReport is the report of a WhatsApp campaign.
func (s *Service) CampaignReport(ctx context.Context, ws *store.Workspace, id uuid.UUID) (WhatsAppReport, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return WhatsAppReport{}, err
	}
	out := WhatsAppReport{ID: c.ID.String(), Name: c.Name, Status: c.Status, SentBy: c.StartedByName, Template: c.TemplateName, Skipped: map[string]int64{},
		Failures: []WhatsAppFailure{}, ByCountry: []WhatsAppReportCountry{}, RatesAsOf: s.rateCard(ctx, s.Q, ws).asOf}
	if snd, err := s.Q.WhatsAppSend(ctx, id); err == nil {
		out.SentAt = snd.FirstSendAt
		if out.SentAt == nil {
			out.SentAt = &snd.ScheduledAt
		}
	}
	rows, err := s.Pool.Query(ctx, `SELECT m.country, m.status, m.reason, m.provider_code, (m.sent_at IS NOT NULL), (w.read_at IS NOT NULL), w.category,
			count(*), coalesce(sum(m.cost), 0), coalesce(sum(m.price), 0)
		FROM channel_messages m JOIN whatsapp_messages w ON w.message_id = m.id
		WHERE m.campaign_id = $1 AND m.kind = 'campaign' GROUP BY 1, 2, 3, 4, 5, 6, 7`, id)
	if err != nil {
		return out, err
	}
	countries := map[string]*WhatsAppReportCountry{}
	failures := map[string]*WhatsAppFailure{}
	for rows.Next() {
		var country, status, reason, code, category string
		var sent, read bool
		var n int64
		var cost, price float64
		if err := rows.Scan(&country, &status, &reason, &code, &sent, &read, &category, &n, &cost, &price); err != nil {
			rows.Close()
			return out, err
		}
		out.Planned += n
		if category != "" {
			out.Category = category
		}
		switch status {
		case "skipped", "cancelled":
			out.Skipped[reason] += n
			continue
		case "queued", "sending":
			out.Queued += n
		case "held":
			out.Held += n
		case "failed", "undelivered":
			out.Failed += n
			k := reason + "/" + code
			if f, ok := failures[k]; ok {
				f.Count += n
			} else {
				failures[k] = &WhatsAppFailure{Reason: reason, Code: code, Count: n}
			}
		}
		cty, ok := countries[country]
		if !ok {
			cty = &WhatsAppReportCountry{Country: country}
			countries[country] = cty
		}
		if status != "queued" && status != "held" && status != "sending" {
			out.Estimate += price
		}
		if status == "failed" || status == "undelivered" {
			cty.Failed += n
		}
		if !sent {
			continue
		}
		out.Sent += n
		cty.Sent += n
		if status == "delivered" {
			out.Delivered += n
			cty.Delivered += n
		}
		if read {
			out.Read += n
			cty.Read += n
		}
		out.Cost += cost
		cty.Cost += cost
	}
	rows.Close()
	out.DeliveredPct, out.ReadPct = pct(out.Delivered, out.Sent), pct(out.Read, out.Delivered)
	out.Cost = float64(int64(out.Cost*100+0.5)) / 100
	out.Estimate = float64(int64(out.Estimate*100+0.5)) / 100
	for _, f := range failures {
		out.Failures = append(out.Failures, *f)
	}
	sort.Slice(out.Failures, func(i, j int) bool { return out.Failures[i].Count > out.Failures[j].Count })
	for _, cty := range countries {
		cty.Cost = float64(int64(cty.Cost*100+0.5)) / 100
		out.ByCountry = append(out.ByCountry, *cty)
	}
	sort.Slice(out.ByCountry, func(i, j int) bool {
		if out.ByCountry[i].Sent != out.ByCountry[j].Sent {
			return out.ByCountry[i].Sent > out.ByCountry[j].Sent
		}
		return out.ByCountry[i].Country < out.ByCountry[j].Country
	})
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE kind = 'button'), count(*) FILTER (WHERE kind = 'reply'), count(*) FILTER (WHERE kind = 'opt_out')
		FROM whatsapp_inbound WHERE campaign_id = $1`, id).Scan(&out.Buttons, &out.Replies, &out.OptOuts); err != nil {
		return out, err
	}
	return out, nil
}

// ReportCSV exports the report by country (logged).
func (s *Service) ReportCSV(ctx context.Context, ws *store.Workspace, id uuid.UUID) ([]byte, string, error) {
	r, err := s.CampaignReport(ctx, ws, id)
	if err != nil {
		return nil, "", err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"country", "sent", "delivered", "read", "failed", "cost_usd"})
	for _, c := range r.ByCountry {
		_ = w.Write([]string{c.Country, strconv.FormatInt(c.Sent, 10), strconv.FormatInt(c.Delivered, 10), strconv.FormatInt(c.Read, 10),
			strconv.FormatInt(c.Failed, 10), strconv.FormatFloat(c.Cost, 'f', 2, 64)})
	}
	_ = w.Write([]string{"total", strconv.FormatInt(r.Sent, 10), strconv.FormatInt(r.Delivered, 10), strconv.FormatInt(r.Read, 10),
		strconv.FormatInt(r.Failed, 10), strconv.FormatFloat(r.Cost, 'f', 2, 64)})
	w.Flush()
	if err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "report.exported", TargetType: "campaign", TargetID: id.String(), TargetName: r.Name,
		Metadata: map[string]any{"channel": "whatsapp", "format": "csv"}}); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), slug(r.Name) + "-whatsapp-report.csv", nil
}

// slug is a file name's part: "summit-see-you-tomorrow".
func slug(name string) string {
	out := make([]rune, 0, len(name))
	dash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
			dash = false
		case r >= 'A' && r <= 'Z':
			out = append(out, r+('a'-'A'))
			dash = false
		default:
			if !dash && len(out) > 0 {
				out = append(out, '-')
				dash = true
			}
		}
	}
	s := string(out)
	for len(s) > 0 && s[len(s)-1] == '-' {
		s = s[:len(s)-1]
	}
	if s == "" {
		return "campaign"
	}
	return clip(s, 60)
}
