package engine

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/mailbox"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/sending/rules"
	"github.com/hostrepute/reputecast/internal/store"
)

// LiveWatch is the complaint watch after a send (LiveMonitorStates'
// complete state, review GS-6): until when, whether people are alerted,
// and each mailbox provider that reports complaints against its limit.
type LiveWatch struct {
	Until  time.Time `json:"until"`
	Hours  int       `json:"hours"`
	Active bool      `json:"active"`
	Alert  bool      `json:"alert"`
	// Limit is the complaint limit per mailbox provider.
	Limit float64         `json:"limit"`
	Rows  []LiveThreshold `json:"rows"`
}

// LivePostmaster is Gmail's latest daily spam rate for the sending domain
// (Postmaster Tools): Gmail reports a rate per day, not single
// complaints.
type LivePostmaster struct {
	// SpamRate is in percent.
	SpamRate float64 `json:"spamRate"`
	Day      string  `json:"day" format:"date"`
}

// liveWatch is a send's complaint watch, nil when it has none.
func (e *Engine) liveWatch(ctx context.Context, ws uuid.UUID, snd store.CampaignSend, ts tallies, th Thresholds, now time.Time) *LiveWatch {
	if snd.WatchUntil == nil {
		return nil
	}
	_, w, _ := rules.Load(ctx, e.Q, ws)
	out := &LiveWatch{Until: *snd.WatchUntil, Hours: w.Hours, Active: now.Before(*snd.WatchUntil), Alert: w.Alert, Limit: th.MailboxComplaint, Rows: []LiveThreshold{}}
	for _, k := range routing.MailboxProviders {
		m := ts.mailbox[k]
		if m == nil || m.reported == 0 || !mailbox.ReportsComplaints(k) {
			continue
		}
		row := thresholdRow("mailbox_complaint", m.complained, m.reported, minEvents, minSampleScoped, th.MailboxComplaint)
		row.Mailbox = k
		out.Rows = append(out.Rows, row)
	}
	return out
}

// livePostmaster is Gmail's latest spam rate for the sending domain from
// the last few days, nil without one.
func (e *Engine) livePostmaster(ctx context.Context, ws uuid.UUID, fromEmail string, now time.Time) *LivePostmaster {
	domain := domainOf(fromEmail)
	if domain == "" {
		return nil
	}
	rows, err := e.Q.SendingPostmasterLatest(ctx, store.SendingPostmasterLatestParams{WorkspaceID: ws, Domain: domain, Since: now.AddDate(0, 0, -3)})
	if err != nil || len(rows) == 0 || rows[0].SpamRate == nil {
		return nil
	}
	return &LivePostmaster{SpamRate: math.Round(*rows[0].SpamRate*100*100) / 100, Day: rows[0].Day.Format(time.DateOnly)}
}

// sinceTrip is a tripped scope's rate on what came after the trip (what
// was delivered, or sent, while the scope waited), nil without any.
func sinceTrip(ts tallies, in store.CampaignIncident, d IncidentDetail) *IncidentRate {
	var t tally
	switch in.Scope {
	case "mailbox_provider":
		if m := ts.mailbox[in.MailboxProvider]; m != nil {
			t = *m
		}
	case "route":
		if in.ProviderID != nil {
			if p := ts.provider[*in.ProviderID]; p != nil {
				t = *p
			}
		}
	default:
		t = ts.total
	}
	var events, base int64
	switch in.Metric {
	case "complaint":
		events, base = t.complained-d.Events, t.reported-d.Base
	case "deferral":
		events, base = t.deferred-d.Events, t.sent-d.Base
	default:
		events, base = t.bounced()-d.Events, t.sent-d.Base
	}
	if base <= 0 || events < 0 {
		return nil
	}
	return &IncidentRate{Metric: in.Metric, Events: events, Base: base, Value: round2(rate(events, base))}
}
