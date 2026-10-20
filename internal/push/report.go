package push

import (
	"bytes"
	"context"
	"encoding/csv"
	"slices"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/store"
)

// The push report (brief §3.8, board PushStates › Report): sent (accepted
// by the push services), shown (reported by the service worker), clicks,
// unsubscribed within 24 hours and expired subscriptions removed, by
// browser and by site.

// PushReportBrowser is one browser's row.
type PushReportBrowser struct {
	Browser  string  `json:"browser" enum:"chrome,chrome_android,edge,firefox,safari,safari_ios,other"`
	Sent     int64   `json:"sent"`
	Shown    int64   `json:"shown"`
	ShownPct float64 `json:"shownPct"`
	Clicks   int64   `json:"clicks"`
}

// PushReportSite is one site's part.
type PushReportSite struct {
	Host string `json:"host"`
	Sent int64  `json:"sent"`
}

// PushReportPoint is the clicks so far, Minutes after the send started.
type PushReportPoint struct {
	Minutes int   `json:"minutes"`
	Clicks  int64 `json:"clicks"`
}

// PushReport is a push campaign's report.
type PushReport struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Status     string     `json:"status" enum:"draft,approval,approved,scheduled,sending,paused,sent,cancelled,failed"`
	SentAt     *time.Time `json:"sentAt,omitempty"`
	SentBy     string     `json:"sentBy"`
	Sites      []string   `json:"sites"`
	Tag        string     `json:"tag"`
	TTLMinutes int        `json:"ttlMinutes"`
	// Planned is everyone the send was for; Skipped those the frequency cap
	// or quiet hours left out, by reason; Held those waiting for their
	// window; Queued those not handed over yet.
	Planned int64            `json:"planned"`
	Skipped map[string]int64 `json:"skipped"`
	Held    int64            `json:"held"`
	Queued  int64            `json:"queued"`
	// Sent were accepted by the push services; Failed weren't, Expired of
	// them because the subscription was gone (404, 410), and removed.
	Sent            int64               `json:"sent"`
	Failed          int64               `json:"failed"`
	Expired         int64               `json:"expired"`
	Shown           int64               `json:"shown"`
	ShownPct        float64             `json:"shownPct"`
	Clicks          int64               `json:"clicks"`
	ClicksPct       float64             `json:"clicksPct"`
	Unsubscribed    int64               `json:"unsubscribed"`
	UnsubscribedPct float64             `json:"unsubscribedPct"`
	ByBrowser       []PushReportBrowser `json:"byBrowser"`
	BySite          []PushReportSite    `json:"bySite"`
	// SitesNotIn are push sites that weren't in this audience.
	SitesNotIn []string          `json:"sitesNotIn"`
	Clicks24   []PushReportPoint `json:"clicks24"`
	CanSend    bool              `json:"canSend"`
}

// browserOrder is the report's order of browsers.
var browserOrder = []string{BrowserChrome, BrowserSafari, BrowserChromeAndroid, BrowserEdge, BrowserFirefox, BrowserSafariIOS, BrowserOther}

// CampaignReport is the report of a push campaign.
func (s *Service) CampaignReport(ctx context.Context, ws *store.Workspace, id uuid.UUID, canSend bool) (PushReport, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return PushReport{}, err
	}
	out := PushReport{ID: c.ID.String(), Name: c.Name, Status: c.Status, SentBy: c.StartedByName, Tag: c.Tag, TTLMinutes: int(c.TtlMinutes), Skipped: map[string]int64{},
		ByBrowser: []PushReportBrowser{}, BySite: []PushReportSite{}, SitesNotIn: []string{}, Clicks24: []PushReportPoint{}, Sites: s.siteLabels(ctx, ws, c.SiteIds),
		CanSend: canSend}
	snd, err := s.Q.PushSend(ctx, id)
	if err == nil {
		out.SentAt = snd.FirstSendAt
		if out.SentAt == nil {
			out.SentAt = &snd.ScheduledAt
		}
	}
	rows, err := s.Pool.Query(ctx, `SELECT p.browser, m.status, m.reason, (m.sent_at IS NOT NULL), (p.shown_at IS NOT NULL), (p.clicked_at IS NOT NULL), count(*)
		FROM channel_messages m JOIN push_messages p ON p.message_id = m.id
		WHERE m.campaign_id = $1 AND m.kind = 'campaign' GROUP BY 1, 2, 3, 4, 5, 6`, id)
	if err != nil {
		return out, err
	}
	byBrowser := map[string]*PushReportBrowser{}
	for rows.Next() {
		var browser, status, reason string
		var sent, shown, clicked bool
		var n int64
		if err := rows.Scan(&browser, &status, &reason, &sent, &shown, &clicked, &n); err != nil {
			rows.Close()
			return out, err
		}
		out.Planned += n
		switch {
		case status == "skipped":
			out.Skipped[reason] += n
			continue
		case status == "queued" && reason == "quiet_hours":
			out.Held += n
		case status == "queued" || status == "sending":
			out.Queued += n
		case status == "failed":
			out.Failed += n
			if reason == "gone" {
				out.Expired += n
			}
		}
		if !sent {
			continue
		}
		b, ok := byBrowser[browser]
		if !ok {
			b = &PushReportBrowser{Browser: browser}
			byBrowser[browser] = b
		}
		b.Sent += n
		out.Sent += n
		if shown {
			b.Shown += n
			out.Shown += n
		}
		if clicked {
			b.Clicks += n
			out.Clicks += n
		}
	}
	rows.Close()
	for _, k := range browserOrder {
		if b, ok := byBrowser[k]; ok {
			b.ShownPct = pct(b.Shown, b.Sent)
			out.ByBrowser = append(out.ByBrowser, *b)
		}
	}
	out.ShownPct, out.ClicksPct = pct(out.Shown, out.Sent), pct(out.Clicks, out.Sent)
	// Unsubscribed within 24 hours of their notification.
	if err := s.Pool.QueryRow(ctx, `SELECT count(DISTINCT s.id) FROM channel_messages m JOIN push_messages p ON p.message_id = m.id
		JOIN push_subscriptions s ON s.id = p.subscription_id
		WHERE m.campaign_id = $1 AND m.kind = 'campaign' AND m.sent_at IS NOT NULL AND s.removed_reason IN ('unsubscribed','turned_off')
		  AND s.removed_at >= m.sent_at AND s.removed_at < m.sent_at + interval '24 hours'`, id).Scan(&out.Unsubscribed); err != nil {
		return out, err
	}
	out.UnsubscribedPct = float64(int64(float64(out.Unsubscribed)*10000/float64(max(out.Sent, 1))+0.5)) / 100
	// By site.
	srows, err := s.Pool.Query(ctx, `SELECT st.host, count(*) FROM channel_messages m JOIN push_messages p ON p.message_id = m.id
		JOIN push_sites st ON st.id = p.site_id
		WHERE m.campaign_id = $1 AND m.kind = 'campaign' AND m.sent_at IS NOT NULL GROUP BY 1 ORDER BY 2 DESC, 1`, id)
	if err != nil {
		return out, err
	}
	for srows.Next() {
		var r PushReportSite
		if srows.Scan(&r.Host, &r.Sent) == nil {
			out.BySite = append(out.BySite, r)
		}
	}
	srows.Close()
	if all, err := s.Q.PushSites(ctx, ws.ID); err == nil {
		for _, st := range all {
			if !slices.Contains(out.Sites, st.Host) {
				out.SitesNotIn = append(out.SitesNotIn, st.Host)
			}
		}
	}
	// Clicks over the first 24 hours, every hour.
	if out.SentAt != nil {
		crows, err := s.Pool.Query(ctx, `SELECT floor(extract(epoch FROM (k.occurred_at - $2::timestamptz)) / 3600)::int AS h, count(DISTINCT k.message_id)
			FROM push_clicks k JOIN channel_messages m ON m.id = k.message_id
			WHERE m.campaign_id = $1 AND m.kind = 'campaign' AND k.occurred_at >= $2 AND k.occurred_at < $2::timestamptz + interval '24 hours'
			  AND NOT EXISTS (SELECT 1 FROM push_clicks k2 WHERE k2.message_id = k.message_id AND k2.occurred_at < k.occurred_at)
			GROUP BY 1 ORDER BY 1`, id, *out.SentAt)
		if err != nil {
			return out, err
		}
		hours := map[int]int64{}
		for crows.Next() {
			var h int
			var n int64
			if crows.Scan(&h, &n) == nil {
				hours[h] = n
			}
		}
		crows.Close()
		var total int64
		out.Clicks24 = append(out.Clicks24, PushReportPoint{Minutes: 0, Clicks: 0})
		last := int(min(24, max(1, time.Since(*out.SentAt).Hours()+1)))
		for h := 0; h < last; h++ {
			total += hours[h]
			out.Clicks24 = append(out.Clicks24, PushReportPoint{Minutes: (h + 1) * 60, Clicks: total})
		}
	}
	return out, nil
}

// ReportCSV exports the report by browser and site (audited).
func (s *Service) ReportCSV(ctx context.Context, ws *store.Workspace, id uuid.UUID) ([]byte, string, error) {
	r, err := s.CampaignReport(ctx, ws, id, false)
	if err != nil {
		return nil, "", err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"group", "name", "sent", "shown", "shown_rate", "clicks"})
	for _, b := range r.ByBrowser {
		_ = w.Write([]string{"browser", b.Browser, strconv.FormatInt(b.Sent, 10), strconv.FormatInt(b.Shown, 10), strconv.FormatFloat(b.ShownPct, 'f', 1, 64),
			strconv.FormatInt(b.Clicks, 10)})
	}
	for _, st := range r.BySite {
		_ = w.Write([]string{"site", st.Host, strconv.FormatInt(st.Sent, 10), "", "", ""})
	}
	_ = w.Write([]string{"total", "", strconv.FormatInt(r.Sent, 10), strconv.FormatInt(r.Shown, 10), strconv.FormatFloat(r.ShownPct, 'f', 1, 64),
		strconv.FormatInt(r.Clicks, 10)})
	w.Flush()
	if err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "report.exported", TargetType: "campaign", TargetID: id.String(), TargetName: r.Name,
		Metadata: map[string]any{"channel": "push", "format": "csv"}}); err != nil {
		return nil, "", err
	}
	name := utmCampaign(r.Name) + "-push-report.csv"
	return buf.Bytes(), name, nil
}

// errNotPush is a campaign of another channel.
var errNotPush = api.NotFound("Push campaign")
