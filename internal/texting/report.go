package texting

import (
	"context"
	"encoding/json"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/channels"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// The SMS report (brief §3.7, §5.5): sent, delivered, failed by reason with
// the provider's own code, clicks by people apart from link previews,
// opt-outs, help requests, other replies and cost; by country and by
// sender; who was held or skipped, and why. While the send runs, it shows
// the progress by time zone with Pause and Cancel: SMS campaigns don't open
// the live monitor.

// SmsReportTiles are the report's tiles.
type SmsReportTiles struct {
	Sent         int64              `json:"sent"`
	ByProvider   []SmsProviderCount `json:"byProvider"`
	Skipped      int64              `json:"skipped"`
	Delivered    int64              `json:"delivered"`
	DeliveredPct float64            `json:"deliveredPct"`
	// Waiting are texts sent with no receipt yet.
	Waiting   int64            `json:"waiting"`
	Failed    int64            `json:"failed"`
	FailedBy  map[string]int64 `json:"failedBy"`
	Clicks    int64            `json:"clicks"`
	ClickPct  float64          `json:"clickPct"`
	Machine   int64            `json:"machine"`
	OptOuts   int64            `json:"optOuts"`
	OptOutPct float64          `json:"optOutPct"`
	// OptOutsByKeyword is true when every opt-out was a reply (STOP).
	OptOutsByKeyword bool    `json:"optOutsByKeyword"`
	Cost             float64 `json:"cost"`
	PerText          float64 `json:"perText"`
	// Planned is the whole send's cost (while it runs).
	Planned      float64 `json:"planned"`
	WithinBudget bool    `json:"withinBudget"`
	// Queued are texts waiting for their time; Held ones held (pause,
	// outage, quiet hours).
	Queued int64 `json:"queued"`
	Held   int64 `json:"held"`
}

// SmsProviderCount is a provider's share of the texts.
type SmsProviderCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// SmsTimePoint is the cumulative delivered count at a moment.
type SmsTimePoint struct {
	At        time.Time `json:"at"`
	Delivered int64     `json:"delivered"`
}

// SmsZoneStep is a time zone's texts going out: "08:00 UTC · Germany,
// Netherlands · +76".
type SmsZoneStep struct {
	At        time.Time `json:"at"`
	Countries []string  `json:"countries"`
	// Region names a part of a country with several zones: atlantic, east,
	// central, mountain, pacific.
	Region string `json:"region,omitempty"`
	Zone   string `json:"zone"`
	// Local and Abbr: the local time and the zone's abbreviation ("10:00",
	// "CEST").
	Local     string `json:"local"`
	Abbr      string `json:"abbr"`
	Total     int64  `json:"total"`
	Sent      int64  `json:"sent"`
	Delivered int64  `json:"delivered"`
	// Status is sent, sending, waiting, paused or held.
	Status string `json:"status" enum:"sent,sending,waiting,paused,held"`
}

// SmsSenderReport is a sender's part of the send.
type SmsSenderReport struct {
	Provider     string   `json:"provider"`
	Adapter      string   `json:"adapter"`
	Sender       string   `json:"sender"`
	SenderKind   string   `json:"senderKind"`
	Countries    []string `json:"countries"`
	Sent         int64    `json:"sent"`
	Delivered    int64    `json:"delivered"`
	DeliveredPct float64  `json:"deliveredPct"`
	Failed       int64    `json:"failed"`
	Cost         float64  `json:"cost"`
	Throughput   float64  `json:"throughput"`
	// ZoneSeconds is the longest any time zone took to go out.
	ZoneSeconds float64 `json:"zoneSeconds"`
}

// SmsCountryReport is a country's line.
type SmsCountryReport struct {
	Country      string  `json:"country"`
	Sender       string  `json:"sender"`
	Sent         int64   `json:"sent"`
	Delivered    int64   `json:"delivered"`
	DeliveredPct float64 `json:"deliveredPct"`
	Failed       int64   `json:"failed"`
	Clicks       int64   `json:"clicks"`
	ClickPct     float64 `json:"clickPct"`
	OptOuts      int64   `json:"optOuts"`
	Cost         float64 `json:"cost"`
	Skipped      int64   `json:"skipped"`
	SkipReason   string  `json:"skipReason,omitempty"`
}

// SmsFailureReason is a failure reason with the providers' own codes.
type SmsFailureReason struct {
	Reason string           `json:"reason"`
	Count  int64            `json:"count"`
	Codes  []SmsFailureCode `json:"codes"`
	// Rate and Limit: the share of texts with a receipt, against the
	// auto-pause threshold it counts toward.
	Rate  float64 `json:"rate"`
	Limit float64 `json:"limit"`
}

// SmsFailureCode is a provider's code and how often it came.
type SmsFailureCode struct {
	Provider string `json:"provider"`
	Code     string `json:"code"`
	Count    int64  `json:"count"`
}

// SmsClickLink is a link with its clicks.
type SmsClickLink struct {
	URL     string `json:"url"`
	Domain  string `json:"domain"`
	People  int64  `json:"people"`
	Machine int64  `json:"machine"`
}

// SmsReply is a reply to the campaign.
type SmsReply struct {
	ID            string    `json:"id"`
	At            time.Time `json:"at"`
	Contact       string    `json:"contact"`
	ContactID     string    `json:"contactId,omitempty"`
	Number        string    `json:"number"`
	Body          string    `json:"body"`
	Kind          string    `json:"kind" enum:"opt_out,opt_in,help,yes,reply"`
	Keyword       string    `json:"keyword,omitempty"`
	Lists         []string  `json:"lists"`
	Replied       bool      `json:"replied"`
	ReplyChars    int       `json:"replyChars"`
	ReplySegments int       `json:"replySegments"`
	// Zone is the person's time zone, for "Chicago local".
	Zone string `json:"zone"`
	// AnyCase is true when the keyword matched in another case ("Stop").
	AnyCase bool `json:"anyCase"`
}

// SmsHeld is who got nothing and why.
type SmsHeld struct {
	Skipped          int64    `json:"skipped"`
	SkippedCountries []string `json:"skippedCountries"`
	SkipReason       string   `json:"skipReason,omitempty"`
	SkipSender       string   `json:"skipSender,omitempty"`
	QuietHours       int64    `json:"quietHours"`
	NoConsent        int64    `json:"noConsent"`
	Suppressed       int64    `json:"suppressed"`
	Lists            []string `json:"lists"`
	// NotSent counts texts cancelled while held (a provider down past the
	// window, a cancelled send), by reason.
	NotSent map[string]int64 `json:"notSent"`
}

// SmsPauseView is why a send is paused.
type SmsPauseView struct {
	Kind      string     `json:"kind" enum:"auto,person,outage,budget"`
	At        *time.Time `json:"at,omitempty"`
	By        string     `json:"by,omitempty"`
	Metric    string     `json:"metric,omitempty" enum:"failed,filtered,opt_outs,"`
	Value     float64    `json:"value"`
	Limit     float64    `json:"limit"`
	Events    int64      `json:"events"`
	Base      int64      `json:"base"`
	Code      string     `json:"code,omitempty"`
	Provider  string     `json:"provider,omitempty"`
	CodeCount int64      `json:"codeCount"`
	Held      int64      `json:"held"`
}

// SmsOutageView is a provider outage holding the send's texts.
type SmsOutageView struct {
	Provider     string     `json:"provider"`
	ProviderID   string     `json:"providerId"`
	Status       string     `json:"status"`
	Since        *time.Time `json:"since,omitempty"`
	Answer       string     `json:"answer,omitempty"`
	Failures     int        `json:"failures"`
	Held         int64      `json:"held"`
	Countries    []string   `json:"countries"`
	HasFallback  bool       `json:"hasFallback"`
	LastAccepted *time.Time `json:"lastAccepted,omitempty"`
	NextTry      *time.Time `json:"nextTry,omitempty"`
	Told         []string   `json:"told"`
}

// SmsTimelineSample is one recipient's events, as their timeline shows them.
type SmsTimelineSample struct {
	ContactID string                `json:"contactId"`
	Name      string                `json:"name"`
	Events    []SmsTimelineSampleEv `json:"events"`
}

// SmsTimelineSampleEv is one event of the sample.
type SmsTimelineSampleEv struct {
	Kind     string    `json:"kind"`
	At       time.Time `json:"at"`
	Provider string    `json:"provider,omitempty"`
	Sender   string    `json:"sender,omitempty"`
	Seconds  float64   `json:"seconds,omitempty"`
	Link     string    `json:"link,omitempty"`
	Target   string    `json:"target,omitempty"`
	Body     string    `json:"body,omitempty"`
}

// SmsReport is an SMS campaign's report.
type SmsReport struct {
	CampaignID   string            `json:"campaignId"`
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	Phase        string            `json:"phase" enum:"scheduled,sending,paused,completed,cancelled,"`
	ScheduleMode string            `json:"scheduleMode" enum:"now,at,local"`
	LocalTime    string            `json:"localTime,omitempty"`
	ScheduledAt  *time.Time        `json:"scheduledAt,omitempty"`
	StartedAt    *time.Time        `json:"startedAt,omitempty"`
	CompletedAt  *time.Time        `json:"completedAt,omitempty"`
	Window       string            `json:"window"`
	Recipients   int64             `json:"recipients"`
	Tiles        SmsReportTiles    `json:"tiles"`
	Points       []SmsTimePoint    `json:"points"`
	Steps        []SmsZoneStep     `json:"steps"`
	Senders      []SmsSenderReport `json:"senders"`
	// ReceiptsWithin is the slowest receipt, in seconds.
	ReceiptsWithin float64            `json:"receiptsWithin"`
	Countries      []SmsCountryReport `json:"countries"`
	Failures       []SmsFailureReason `json:"failures"`
	Links          []SmsClickLink     `json:"links"`
	Replies        []SmsReply         `json:"replies"`
	ReplyCounts    map[string]int64   `json:"replyCounts"`
	Held           SmsHeld            `json:"held"`
	Pause          *SmsPauseView      `json:"pause,omitempty"`
	Outage         *SmsOutageView     `json:"outage,omitempty"`
	Sample         *SmsTimelineSample `json:"sample,omitempty"`
	ShortDomain    string             `json:"shortDomain"`
	CanPause       bool               `json:"canPause"`
	CanResume      bool               `json:"canResume"`
	Helpers        []string           `json:"helpers"`
	Timezone       string             `json:"timezone"`
}

// usRegion names the part of the US and Canada a zone covers.
func usRegion(zone string) string {
	switch zone {
	case "America/Halifax", "America/Moncton", "America/Glace_Bay", "America/Goose_Bay", "America/St_Johns":
		return "atlantic"
	case "America/New_York", "America/Toronto", "America/Detroit", "America/Montreal", "America/Indiana/Indianapolis", "America/Kentucky/Louisville", "America/Nassau":
		return "east"
	case "America/Chicago", "America/Winnipeg", "America/Regina", "America/Menominee", "America/Indiana/Knox":
		return "central"
	case "America/Denver", "America/Edmonton", "America/Phoenix", "America/Boise":
		return "mountain"
	case "America/Los_Angeles", "America/Vancouver":
		return "pacific"
	case "America/Anchorage", "America/Juneau":
		return "alaska"
	case "Pacific/Honolulu":
		return "hawaii"
	}
	return ""
}

// Report builds an SMS campaign's report.
func (s *Service) Report(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (SmsReport, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return SmsReport{}, err
	}
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return SmsReport{}, err
	}
	out := SmsReport{CampaignID: c.ID.String(), Name: c.Name, Status: c.Status, ScheduleMode: c.ScheduleMode, ScheduledAt: c.ScheduledAt, StartedAt: c.StartedAt,
		CompletedAt: c.CompletedAt, Window: g.rules.Window.String(), Points: []SmsTimePoint{}, Steps: []SmsZoneStep{}, Senders: []SmsSenderReport{},
		Countries: []SmsCountryReport{}, Failures: []SmsFailureReason{}, Links: []SmsClickLink{}, Replies: []SmsReply{}, ReplyCounts: map[string]int64{},
		ShortDomain: g.rules.ShortDomain, CanPause: p.Can(authz.CampaignsPause), CanResume: canSend(p), Helpers: []string{}, Timezone: ws.Timezone}
	if c.LocalMinute != nil {
		out.LocalTime = channels.Clock(int(*c.LocalMinute))
	}
	if !out.CanResume {
		out.Helpers = access.Helpers(ctx, s.Q, ws.ID, authz.SMSSend)
	}
	snd, sendErr := s.Q.SmsSend(ctx, c.ID)
	if sendErr == nil {
		out.Phase = snd.Phase
		out.Recipients = int64(snd.Recipients)
	}
	t := SmsReportTiles{ByProvider: []SmsProviderCount{}, FailedBy: map[string]int64{}}
	// Totals by provider.
	rows, err := s.Pool.Query(ctx, `SELECT coalesce(p.name, ''), count(*) FILTER (WHERE m.sent_at IS NOT NULL), count(*) FILTER (WHERE m.status = 'delivered'),
			count(*) FILTER (WHERE m.status IN ('undelivered','failed')), count(*) FILTER (WHERE m.status = 'sent'),
			count(*) FILTER (WHERE m.status = 'skipped'), count(*) FILTER (WHERE m.status = 'queued'), count(*) FILTER (WHERE m.status = 'held'),
			coalesce(sum(m.cost) FILTER (WHERE m.status IN ('sending','sent','delivered','undelivered','failed')), 0)::float8,
			coalesce(sum(m.cost) FILTER (WHERE m.status NOT IN ('skipped','cancelled')), 0)::float8
		FROM channel_messages m LEFT JOIN sms_providers p ON p.id = m.provider_id
		WHERE m.campaign_id = $1 AND m.kind = 'campaign' GROUP BY 1 ORDER BY min(p.connected_at)`, c.ID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var name string
		var sent, delivered, failed, waiting, skipped, queued, held int64
		var cost, planned float64
		if err := rows.Scan(&name, &sent, &delivered, &failed, &waiting, &skipped, &queued, &held, &cost, &planned); err != nil {
			rows.Close()
			return out, err
		}
		if name != "" && sent+queued+held > 0 {
			t.ByProvider = append(t.ByProvider, SmsProviderCount{Name: name, Count: sent})
		}
		t.Sent += sent
		t.Delivered += delivered
		t.Failed += failed
		t.Waiting += waiting
		t.Skipped += skipped
		t.Queued += queued
		t.Held += held
		t.Cost += cost
		t.Planned += planned
	}
	rows.Close()
	t.Cost, t.Planned = round2(t.Cost), round2(t.Planned)
	t.DeliveredPct = pct(t.Delivered, t.Sent)
	if t.Sent > 0 {
		t.PerText = math.Round(t.Cost/float64(t.Sent)*1000) / 1000
	}
	t.WithinBudget = g.rules.Budget <= 0
	if !t.WithinBudget {
		spent, _ := s.spentThisMonth(ctx, ws)
		t.WithinBudget = spent <= g.rules.Budget+1e-9
	}
	rows, err = s.Pool.Query(ctx, `SELECT reason, count(*) FROM channel_messages WHERE campaign_id = $1 AND kind = 'campaign' AND status IN ('undelivered','failed')
		GROUP BY reason`, c.ID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var reason string
		var n int64
		if rows.Scan(&reason, &n) == nil {
			t.FailedBy[reason] = n
		}
	}
	rows.Close()
	_ = s.Pool.QueryRow(ctx, `SELECT count(DISTINCT contact_id) FILTER (WHERE NOT machine), count(*) FILTER (WHERE machine) FROM sms_link_clicks WHERE campaign_id = $1`,
		c.ID).Scan(&t.Clicks, &t.Machine)
	t.ClickPct = pct(t.Clicks, t.Delivered)
	var byProvider int64
	rows, err = s.Pool.Query(ctx, `SELECT kind, count(*) FROM sms_inbound WHERE campaign_id = $1 GROUP BY kind`, c.ID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var kind string
		var n int64
		if rows.Scan(&kind, &n) == nil {
			out.ReplyCounts[kind] = n
		}
	}
	rows.Close()
	t.OptOuts = out.ReplyCounts["opt_out"]
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM sms_inbound WHERE campaign_id = $1 AND kind = 'opt_out' AND (effect->>'providerAnswered')::boolean`, c.ID).Scan(&byProvider)
	t.OptOutsByKeyword = byProvider == 0
	t.OptOutPct = pct(t.OptOuts, t.Sent)
	out.Tiles = t
	// Delivery over time and the zone steps.
	if err := s.reportTimeline(ctx, &out, c.ID); err != nil {
		return out, err
	}
	// By sender.
	rows, err = s.Pool.Query(ctx, `SELECT m.sender_id, coalesce(p.name, ''), coalesce(p.adapter, ''), m.sender, array_agg(DISTINCT m.country),
			count(*) FILTER (WHERE m.sent_at IS NOT NULL), count(*) FILTER (WHERE m.status = 'delivered'), count(*) FILTER (WHERE m.status IN ('undelivered','failed')),
			coalesce(sum(m.cost) FILTER (WHERE m.status IN ('sending','sent','delivered','undelivered','failed')), 0)::float8,
			coalesce(max(extract(epoch FROM (m.delivered_at - m.sent_at))), 0)::float8
		FROM channel_messages m LEFT JOIN sms_providers p ON p.id = m.provider_id
		WHERE m.campaign_id = $1 AND m.kind = 'campaign' AND m.sender_id IS NOT NULL AND m.status NOT IN ('skipped','cancelled')
		GROUP BY 1, 2, 3, 4 ORDER BY min(p.connected_at)`, c.ID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var sid uuid.UUID
		var r SmsSenderReport
		var slowest float64
		if err := rows.Scan(&sid, &r.Provider, &r.Adapter, &r.Sender, &r.Countries, &r.Sent, &r.Delivered, &r.Failed, &r.Cost, &slowest); err != nil {
			rows.Close()
			return out, err
		}
		if sn, ok := g.senders[sid]; ok {
			r.SenderKind, r.Throughput = sn.Kind, sn.Throughput
			r.Countries = orderCountries(r.Countries, sn.Countries)
		}
		r.Sender = senderShown(r.Sender)
		r.Cost = round2(r.Cost)
		r.DeliveredPct = pct(r.Delivered, r.Sent)
		if slowest > out.ReceiptsWithin {
			out.ReceiptsWithin = slowest
		}
		out.Senders = append(out.Senders, r)
	}
	rows.Close()
	for i := range out.Senders {
		var secs float64
		_ = s.Pool.QueryRow(ctx, `SELECT coalesce(max(d), 0)::float8 FROM (SELECT extract(epoch FROM (max(sent_at) - min(sent_at))) AS d FROM channel_messages
			WHERE campaign_id = $1 AND kind = 'campaign' AND sender = $2 AND sent_at IS NOT NULL GROUP BY time_zone) z`, c.ID, rawSender(out.Senders[i].Sender)).Scan(&secs)
		out.Senders[i].ZoneSeconds = math.Ceil(secs)
	}
	// By country.
	rows, err = s.Pool.Query(ctx, `SELECT m.country, m.sender, count(*) FILTER (WHERE m.sent_at IS NOT NULL), count(*) FILTER (WHERE m.status = 'delivered'),
			count(*) FILTER (WHERE m.status IN ('undelivered','failed')), count(*) FILTER (WHERE m.status = 'skipped'), max(m.reason) FILTER (WHERE m.status = 'skipped'),
			coalesce(sum(m.cost) FILTER (WHERE m.status IN ('sending','sent','delivered','undelivered','failed')), 0)::float8,
			(SELECT count(DISTINCT k.contact_id) FROM sms_link_clicks k JOIN channel_messages x ON x.id = k.message_id
			 WHERE k.campaign_id = $1 AND NOT k.machine AND x.country = m.country),
			(SELECT count(*) FROM sms_inbound i JOIN channel_messages x ON x.id = i.message_id WHERE i.campaign_id = $1 AND i.kind = 'opt_out' AND x.country = m.country)
		FROM channel_messages m WHERE m.campaign_id = $1 AND m.kind = 'campaign' GROUP BY m.country, m.sender`, c.ID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var r SmsCountryReport
		var reason *string
		if err := rows.Scan(&r.Country, &r.Sender, &r.Sent, &r.Delivered, &r.Failed, &r.Skipped, &reason, &r.Cost, &r.Clicks, &r.OptOuts); err != nil {
			rows.Close()
			return out, err
		}
		if reason != nil {
			r.SkipReason = *reason
		}
		r.Sender = senderShown(r.Sender)
		r.Cost = round2(r.Cost)
		r.DeliveredPct, r.ClickPct = pct(r.Delivered, r.Sent), pct(r.Clicks, r.Delivered)
		out.Countries = append(out.Countries, r)
	}
	rows.Close()
	merged := map[string]*SmsCountryReport{}
	order := []string{}
	for i := range out.Countries {
		r := out.Countries[i]
		if m, ok := merged[r.Country]; ok {
			m.Sent, m.Delivered, m.Failed, m.Skipped, m.Cost = m.Sent+r.Sent, m.Delivered+r.Delivered, m.Failed+r.Failed, m.Skipped+r.Skipped, round2(m.Cost+r.Cost)
			if m.Sender == "" {
				m.Sender = r.Sender
			}
			if r.SkipReason != "" {
				m.SkipReason = r.SkipReason
			}
			m.DeliveredPct, m.ClickPct = pct(m.Delivered, m.Sent), pct(m.Clicks, m.Delivered)
			continue
		}
		x := r
		merged[r.Country] = &x
		order = append(order, r.Country)
	}
	countries := []SmsCountryReport{}
	for _, cc := range order {
		countries = append(countries, *merged[cc])
	}
	sort.SliceStable(countries, func(i, j int) bool {
		a, b := countries[i], countries[j]
		if (a.Sent == 0) != (b.Sent == 0) {
			return a.Sent > 0
		}
		ia, ib := senderRank(out.Senders, a.Sender), senderRank(out.Senders, b.Sender)
		if ia != ib {
			return ia < ib
		}
		if a.Sent != b.Sent {
			return a.Sent > b.Sent
		}
		return a.Country < b.Country
	})
	for i := range countries {
		if countries[i].Sent == 0 && countries[i].Skipped > 0 {
			// A skipped country's sender is the one its route names.
			if d := g.decide(countries[i].Country); d.Route != nil {
				if sn, ok := g.senders[d.Route.SenderID]; ok {
					countries[i].Sender = senderLabel(sn)
				}
			}
		}
	}
	out.Countries = countries
	// Failures by reason, with codes.
	th := thresholdsOf(snd.Thresholds, g.rules)
	receipts := out.Tiles.Delivered + out.Tiles.Failed
	rows, err = s.Pool.Query(ctx, `SELECT m.reason, coalesce(p.name, ''), m.provider_code, count(*) FROM channel_messages m LEFT JOIN sms_providers p ON p.id = m.provider_id
		WHERE m.campaign_id = $1 AND m.kind = 'campaign' AND m.status IN ('undelivered','failed') GROUP BY 1, 2, 3 ORDER BY 4 DESC`, c.ID)
	if err != nil {
		return out, err
	}
	byReason := map[string]*SmsFailureReason{}
	reasons := []string{}
	for rows.Next() {
		var reason, prov, code string
		var n int64
		if rows.Scan(&reason, &prov, &code, &n) != nil {
			continue
		}
		f, ok := byReason[reason]
		if !ok {
			f = &SmsFailureReason{Reason: reason, Codes: []SmsFailureCode{}}
			byReason[reason] = f
			reasons = append(reasons, reason)
		}
		f.Count += n
		f.Codes = append(f.Codes, SmsFailureCode{Provider: prov, Code: code, Count: n})
	}
	rows.Close()
	// Each reason shows the rate of the threshold it counts toward, as the
	// dispatcher computes it: filtered texts have their own; every other
	// failure counts toward the failed threshold together.
	var failed int64
	for _, r := range reasons {
		if r != "filtered" {
			failed += byReason[r].Count
		}
	}
	for _, r := range reasons {
		f := byReason[r]
		switch r {
		case "filtered":
			f.Limit = th.Filtered
			f.Rate = pct(f.Count, receipts)
		default:
			f.Limit = th.Failed
			f.Rate = pct(failed, receipts)
		}
		out.Failures = append(out.Failures, *f)
	}
	sort.SliceStable(out.Failures, func(i, j int) bool { return out.Failures[i].Count > out.Failures[j].Count })
	// Links.
	rows, err = s.Pool.Query(ctx, `SELECT l.original, count(DISTINCT k.contact_id) FILTER (WHERE NOT k.machine), count(k.id) FILTER (WHERE k.machine)
		FROM sms_links l LEFT JOIN sms_link_clicks k ON k.code = l.code WHERE l.campaign_id = $1 GROUP BY l.original ORDER BY 2 DESC`, c.ID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var l SmsClickLink
		if rows.Scan(&l.URL, &l.People, &l.Machine) == nil {
			l.Domain = g.rules.ShortDomain
			out.Links = append(out.Links, l)
		}
	}
	rows.Close()
	// Replies: the first six.
	if out.Replies, err = s.replies(ctx, ws, c.ID, "", 6, 0); err != nil {
		return out, err
	}
	// Held and skipped.
	out.Held = SmsHeld{SkippedCountries: []string{}, Lists: []string{}, NotSent: map[string]int64{}}
	for _, r := range out.Countries {
		if r.Skipped > 0 {
			out.Held.Skipped += r.Skipped
			out.Held.SkippedCountries = append(out.Held.SkippedCountries, r.Country)
			out.Held.SkipReason, out.Held.SkipSender = r.SkipReason, r.Sender
		}
	}
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM channel_messages m WHERE m.campaign_id = $1 AND m.kind = 'campaign' AND m.sent_at IS NOT NULL
		AND m.sent_at > coalesce((SELECT scheduled_at FROM sms_sends WHERE campaign_id = $1), m.created_at) + interval '5 minutes'
		AND (SELECT schedule_mode FROM sms_campaigns WHERE campaign_id = $1) = 'now'`, c.ID).Scan(&out.Held.QuietHours)
	if sendErr == nil {
		var left struct {
			NoConsent  int64 `json:"noConsent"`
			Suppressed int64 `json:"suppressed"`
		}
		_ = json.Unmarshal(snd.LeftOut, &left)
		out.Held.NoConsent, out.Held.Suppressed = left.NoConsent, left.Suppressed
	}
	out.Held.Lists, _ = s.audienceNames(ctx, ws, parseAudience(c.Audience))
	rows, err = s.Pool.Query(ctx, `SELECT reason, count(*) FROM channel_messages WHERE campaign_id = $1 AND kind = 'campaign' AND status = 'cancelled' GROUP BY reason`, c.ID)
	if err == nil {
		for rows.Next() {
			var r string
			var n int64
			if rows.Scan(&r, &n) == nil {
				out.Held.NotSent[r] = n
			}
		}
		rows.Close()
	}
	// Pause and outage.
	if sendErr == nil && snd.Phase == "paused" {
		pv := &SmsPauseView{Kind: snd.PausedKind, At: snd.PausedAt}
		var pause map[string]any
		_ = json.Unmarshal(snd.Pause, &pause)
		pv.Metric, _ = pause["metric"].(string)
		pv.Value, _ = pause["value"].(float64)
		pv.Limit, _ = pause["limit"].(float64)
		if v, ok := pause["events"].(float64); ok {
			pv.Events = int64(v)
		}
		if v, ok := pause["base"].(float64); ok {
			pv.Base = int64(v)
		}
		pv.Code, _ = pause["code"].(string)
		pv.Provider, _ = pause["provider"].(string)
		if v, ok := pause["codeCount"].(float64); ok {
			pv.CodeCount = int64(v)
		}
		pv.By, _ = pause["by"].(string)
		_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM channel_messages WHERE campaign_id = $1 AND status = 'held'`, c.ID).Scan(&pv.Held)
		out.Pause = pv
	}
	var downProv *uuid.UUID
	_ = s.Pool.QueryRow(ctx, `SELECT provider_id FROM channel_messages WHERE campaign_id = $1 AND status = 'held' AND reason = 'provider_down' LIMIT 1`, c.ID).Scan(&downProv)
	if downProv != nil {
		if pr, err := s.Q.SmsProviderAny(ctx, *downProv); err == nil {
			h := healthOf(pr)
			o := &SmsOutageView{Provider: pr.Name, ProviderID: pr.ID.String(), Status: pr.Status, Since: pr.StatusSince, Answer: answerIn(i18n.LangFrom(ctx), h.Answer),
				Failures:  h.ConsecutiveFailures,
				Countries: []string{}, Told: access.Helpers(ctx, s.Q, ws.ID, authz.ProvidersManage)}
			_ = s.Pool.QueryRow(ctx, `SELECT count(*), coalesce(array_agg(DISTINCT country), '{}') FROM channel_messages WHERE campaign_id = $1 AND status = 'held'
				AND reason = 'provider_down'`, c.ID).Scan(&o.Held, &o.Countries)
			_ = s.Pool.QueryRow(ctx, `SELECT max(sent_at) FROM channel_messages WHERE campaign_id = $1 AND provider_id = $2`, c.ID, pr.ID).Scan(&o.LastAccepted)
			for _, r := range g.routes {
				if sn, ok := g.senders[r.SenderID]; ok && sn.ProviderID == pr.ID && r.FallbackSenderID != nil {
					o.HasFallback = true
				}
			}
			if pr.LastCheckedAt != nil {
				next := pr.LastCheckedAt.Add(2 * time.Minute)
				o.NextTry = &next
			}
			out.Outage = o
		}
	}
	// A recipient's timeline: the latest one who replied, else who clicked.
	out.Sample = s.timelineSample(ctx, ws, c.ID)
	return out, nil
}

// rawSender turns a shown sender back into the stored value.
func rawSender(shown string) string {
	if strings.HasPrefix(shown, "+") {
		return "+" + strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, shown)
	}
	return shown
}

func senderRank(senders []SmsSenderReport, sender string) int {
	for i, s := range senders {
		if s.Sender == sender {
			return i
		}
	}
	return len(senders)
}

// orderCountries keeps the sender's own order of its countries.
func orderCountries(got, order []string) []string {
	out := []string{}
	for _, c := range order {
		if slices.Contains(got, c) {
			out = append(out, c)
		}
	}
	for _, c := range got {
		if !slices.Contains(out, c) {
			out = append(out, c)
		}
	}
	return out
}

// reportTimeline is the cumulative delivered count over time and each time
// zone's step.
func (s *Service) reportTimeline(ctx context.Context, out *SmsReport, id uuid.UUID) error {
	rows, err := s.Pool.Query(ctx, `SELECT date_trunc('minute', delivered_at) AS t, count(*) FROM channel_messages
		WHERE campaign_id = $1 AND kind = 'campaign' AND delivered_at IS NOT NULL GROUP BY 1 ORDER BY 1`, id)
	if err != nil {
		return err
	}
	var total int64
	for rows.Next() {
		var at time.Time
		var n int64
		if rows.Scan(&at, &n) == nil {
			total += n
			out.Points = append(out.Points, SmsTimePoint{At: at, Delivered: total})
		}
	}
	rows.Close()
	rows, err = s.Pool.Query(ctx, `SELECT date_trunc('minute', coalesce(send_after, created_at)) AS t, time_zone, country, count(*),
			count(*) FILTER (WHERE sent_at IS NOT NULL), count(*) FILTER (WHERE status = 'delivered'), count(*) FILTER (WHERE status = 'held'),
			count(*) FILTER (WHERE status IN ('queued','sending'))
		FROM channel_messages WHERE campaign_id = $1 AND kind = 'campaign' AND status NOT IN ('skipped')
		GROUP BY 1, 2, 3 ORDER BY 1, 3`, id)
	if err != nil {
		return err
	}
	type part struct {
		at                                    time.Time
		zone, country                         string
		total, sent, delivered, held, waiting int64
	}
	parts := []part{}
	for rows.Next() {
		var p part
		if rows.Scan(&p.at, &p.zone, &p.country, &p.total, &p.sent, &p.delivered, &p.held, &p.waiting) == nil {
			parts = append(parts, p)
		}
	}
	rows.Close()
	now := s.now()
	type key struct {
		at     time.Time
		region string
	}
	steps := map[key]*SmsZoneStep{}
	keys := []key{}
	for _, p := range parts {
		region := ""
		if p.country == "US" || p.country == "CA" {
			region = usRegion(p.zone)
		}
		k := key{p.at, region}
		st, ok := steps[k]
		if !ok {
			loc := location(p.zone)
			lt := p.at.In(loc)
			st = &SmsZoneStep{At: p.at, Countries: []string{}, Region: region, Zone: p.zone, Local: lt.Format("15:04"), Abbr: lt.Format("MST")}
			steps[k] = st
			keys = append(keys, k)
		}
		if !slices.Contains(st.Countries, p.country) {
			st.Countries = append(st.Countries, p.country)
		}
		st.Total += p.total
		st.Sent += p.sent
		st.Delivered += p.delivered
		switch {
		case p.held > 0 && out.Phase == "paused":
			st.Status = "paused"
		case p.held > 0 && st.Status != "paused":
			st.Status = "held"
		case p.waiting > 0 && !p.at.After(now) && st.Status == "":
			st.Status = "sending"
		case p.waiting > 0 && st.Status == "":
			st.Status = "waiting"
		}
	}
	for _, k := range keys {
		st := steps[k]
		if st.Status == "" {
			st.Status = "sent"
		}
		if out.Phase == "paused" && st.Status == "waiting" {
			st.Status = "held"
		}
		sort.Strings(st.Countries)
		out.Steps = append(out.Steps, *st)
	}
	return nil
}

// replies lists a campaign's replies, newest last as they came.
func (s *Service) replies(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, kind string, limit, offset int) ([]SmsReply, error) {
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT i.id, i.received_at, i.phone, i.body, i.kind, i.keyword, i.effect, i.reply_message_id,
			coalesce((SELECT trim(c.first_name || ' ' || c.last_name) FROM contacts c WHERE c.id = i.contact_ids[1]), ''),
			i.contact_ids[1], coalesce((SELECT m.time_zone FROM channel_messages m WHERE m.id = i.message_id), ''),
			coalesce((SELECT m.body FROM channel_messages m WHERE m.id = i.reply_message_id), '')
		FROM sms_inbound i WHERE i.campaign_id = $1 AND ($2 = '' OR i.kind = $2 OR ($2 = 'other' AND i.kind NOT IN ('opt_out','help')))
		ORDER BY i.received_at LIMIT $3 OFFSET $4`, campaignID, kind, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SmsReply{}
	for rows.Next() {
		var r SmsReply
		var id uuid.UUID
		var ph string
		var effect []byte
		var replyID, contactID *uuid.UUID
		var replyBody string
		if err := rows.Scan(&id, &r.At, &ph, &r.Body, &r.Kind, &r.Keyword, &effect, &replyID, &r.Contact, &contactID, &r.Zone, &replyBody); err != nil {
			return nil, err
		}
		r.ID, r.Number = id.String(), maskPhone(ph)
		if contactID != nil {
			r.ContactID = contactID.String()
		}
		var eff inboundEffect
		_ = json.Unmarshal(effect, &eff)
		r.Lists, r.Replied = eff.Lists, eff.Replied
		if r.Lists == nil {
			r.Lists = []string{}
		}
		if replyBody != "" {
			cnt := countOf(replyBody)
			r.ReplyChars, r.ReplySegments = cnt.Characters, cnt.Segments
		}
		r.AnyCase = r.Keyword != "" && strings.TrimSpace(r.Body) != r.Keyword
		_ = g
		out = append(out, r)
	}
	return out, rows.Err()
}

// SmsReplyList is a page of a campaign's replies.
type SmsReplyList struct {
	Items  []SmsReply       `json:"items"`
	Counts map[string]int64 `json:"counts"`
}

// Replies lists a campaign's replies by kind: opt_out, help, other or all.
func (s *Service) Replies(ctx context.Context, ws *store.Workspace, id uuid.UUID, kind string, limit, offset int) (SmsReplyList, error) {
	if _, err := s.campaign(ctx, ws, id); err != nil {
		return SmsReplyList{}, err
	}
	items, err := s.replies(ctx, ws, id, kind, limit, offset)
	if err != nil {
		return SmsReplyList{}, err
	}
	out := SmsReplyList{Items: items, Counts: map[string]int64{}}
	rows, err := s.Pool.Query(ctx, `SELECT kind, count(*) FROM sms_inbound WHERE campaign_id = $1 GROUP BY kind`, id)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		var n int64
		if rows.Scan(&k, &n) == nil {
			out.Counts[k] = n
		}
	}
	return out, rows.Err()
}

// timelineSample is a recipient's events for the report's timeline box.
func (s *Service) timelineSample(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) *SmsTimelineSample {
	var contactID uuid.UUID
	var msgID uuid.UUID
	err := s.Pool.QueryRow(ctx, `SELECT i.contact_ids[1], i.message_id FROM sms_inbound i WHERE i.campaign_id = $1 AND i.kind = 'reply'
		AND cardinality(i.contact_ids) > 0 AND i.message_id IS NOT NULL
		AND EXISTS (SELECT 1 FROM sms_link_clicks k WHERE k.message_id = i.message_id AND NOT k.machine)
		ORDER BY i.received_at LIMIT 1`, campaignID).Scan(&contactID, &msgID)
	if err != nil {
		return nil
	}
	out := &SmsTimelineSample{ContactID: contactID.String(), Events: []SmsTimelineSampleEv{}}
	_ = s.Pool.QueryRow(ctx, `SELECT trim(first_name || ' ' || last_name) FROM contacts WHERE id = $1`, contactID).Scan(&out.Name)
	m, err := s.Q.ChanMessage(ctx, msgID)
	if err != nil {
		return nil
	}
	prov := ""
	if m.ProviderID != nil {
		if p, err := s.Q.SmsProviderAny(ctx, *m.ProviderID); err == nil {
			prov = p.Name
		}
	}
	if m.DeliveredAt != nil {
		ev := SmsTimelineSampleEv{Kind: "delivered", At: *m.DeliveredAt, Provider: prov, Sender: senderShown(m.Sender)}
		if m.SentAt != nil {
			ev.Seconds = math.Round(m.DeliveredAt.Sub(*m.SentAt).Seconds())
		}
		out.Events = append(out.Events, ev)
	}
	var code, target string
	var at time.Time
	if err := s.Pool.QueryRow(ctx, `SELECT k.code, l.original, k.occurred_at FROM sms_link_clicks k JOIN sms_links l ON l.code = k.code WHERE k.message_id = $1 AND NOT k.machine
		ORDER BY k.occurred_at LIMIT 1`, msgID).Scan(&code, &target, &at); err == nil {
		r, _ := s.rules(ctx, s.Q, ws)
		out.Events = append(out.Events, SmsTimelineSampleEv{Kind: "clicked", At: at, Link: r.ShortDomain + "/" + code, Target: target})
	}
	var body string
	if err := s.Pool.QueryRow(ctx, `SELECT body, received_at FROM sms_inbound WHERE message_id = $1 AND kind = 'reply' ORDER BY received_at LIMIT 1`, msgID).Scan(&body, &at); err == nil {
		out.Events = append(out.Events, SmsTimelineSampleEv{Kind: "replied", At: at, Body: body})
	}
	return out
}
