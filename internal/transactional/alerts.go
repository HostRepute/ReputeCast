package transactional

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// Delivery alerts: within the last hour, a template's bounce rate (2% by
// default), failed messages (10) or the median delivery time (30 s) passed
// its limit. There is no auto-pause, because password resets must keep
// flowing: members with transactional:view get a notification, in the app
// and by email as their preferences say. Failovers are kept as incidents
// for the overview.

// Alert kinds.
const (
	AlertBounceRate     = "bounce_rate"
	AlertFailed         = "failed"
	AlertMedianDelivery = "median_delivery"
	AlertFailover       = "failover"
)

// AlertMinSample is how many messages an hour needs before its bounce
// rate or median delivery time can raise an alert (one bounce out of three
// sends isn't a trend).
const AlertMinSample = 20

// failoverQuiet is how long without a failover before an incident ends.
const failoverQuiet = 10 * time.Minute

// alertDetail is what an alert keeps.
type alertDetail struct {
	Sent     int64         `json:"sent,omitempty"`
	Bounced  int64         `json:"bounced,omitempty"`
	Hard     int64         `json:"hard,omitempty"`
	Domains  []alertDomain `json:"domains,omitempty"`
	Failed   int64         `json:"failed,omitempty"`
	MedianMs int64         `json:"median_ms,omitempty"`
	// Failover: how many messages went through the fallback, how many
	// were lost, and the fallback's name.
	Messages int64  `json:"messages,omitempty"`
	Lost     int64  `json:"lost,omitempty"`
	Fallback string `json:"fallback,omitempty"`
}

type alertDomain struct {
	Domain string `json:"domain"`
	Count  int64  `json:"count"`
	// TypoOf is the common domain this one looks like a typo of (gmial.com:
	// gmail.com), set when the alert is read.
	TypoOf string `json:"typoOf,omitempty" doc:"The common domain this one looks like a typo of"`
}

// CheckAlerts opens and ends the delivery alerts of every workspace that
// sent transactional mail in the last hour.
func (s *Service) CheckAlerts(ctx context.Context) error {
	now := s.now()
	ids, err := s.Q.TxnAlertWorkspaces(ctx, now.Add(-time.Hour))
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.checkWorkspace(ctx, id, now); err != nil {
			return fmt.Errorf("workspace %s: %w", id, err)
		}
	}
	return nil
}

func (s *Service) checkWorkspace(ctx context.Context, wsID uuid.UUID, now time.Time) error {
	ws, err := s.Q.GetWorkspace(ctx, wsID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	st, err := s.settingsRow(ctx, s.Q, wsID)
	if err != nil {
		return err
	}
	hour := now.Add(-time.Hour)

	// A template's bounce rate.
	tpls, err := s.Q.TxnAlertTemplateHour(ctx, store.TxnAlertTemplateHourParams{WorkspaceID: wsID, Since: hour})
	if err != nil {
		return err
	}
	over := map[uuid.UUID]bool{}
	for _, t := range tpls {
		rate := ratePct(t.Bounced, t.Sent)
		if t.Sent < AlertMinSample || rate <= st.AlertBounceRate {
			continue
		}
		over[t.TemplateID] = true
		doms, err := s.Q.TxnAlertBounceDomains(ctx, store.TxnAlertBounceDomainsParams{WorkspaceID: wsID, TemplateID: &t.TemplateID, Since: hour})
		if err != nil {
			return err
		}
		d := alertDetail{Sent: t.Sent, Bounced: t.Bounced, Hard: t.Hard}
		for _, dm := range doms {
			d.Domains = append(d.Domains, alertDomain{Domain: dm.Domain, Count: dm.N})
		}
		tid := t.TemplateID
		n := notify.Notification{Kind: notify.KindTransactionalAlert, Severity: "bad", TitleText: texts.M("alert.title").Keep(nil),
			BodyText: texts.M("alert.bounceRate").Keep(bounceArgs(t.TemplateName, rate, st.AlertBounceRate, d.Domains)),
			Link:     "/w/" + ws.Slug + "/transactional/messages?" + url.Values{"template": {t.TemplateKey}, "status": {"bounced"}, "range": {"1h"}}.Encode(),
			Data:     map[string]any{"template": t.TemplateKey, "rate": rate, "limit": st.AlertBounceRate}}
		if err := s.raise(ctx, &ws, AlertBounceRate, &tid, nil, rate, st.AlertBounceRate, d, now, &n); err != nil {
			return err
		}
	}
	open, err := s.Q.TxnAlertsOpenFor(ctx, store.TxnAlertsOpenForParams{WorkspaceID: wsID, Kind: AlertBounceRate})
	if err != nil {
		return err
	}
	for _, a := range open {
		if a.TemplateID != nil && !over[*a.TemplateID] {
			if _, err := s.Q.TxnAlertEnd(ctx, store.TxnAlertEndParams{Now: now, WorkspaceID: wsID, Kind: AlertBounceRate, TemplateID: a.TemplateID}); err != nil {
				return err
			}
		}
	}

	// Failed messages and the median delivery time.
	h, err := s.Q.TxnAlertHour(ctx, store.TxnAlertHourParams{WorkspaceID: wsID, Since: hour})
	if err != nil {
		return err
	}
	if h.Failed >= int64(st.AlertFailed) {
		n := notify.Notification{Kind: notify.KindTransactionalAlert, Severity: "bad", TitleText: texts.M("alert.title").Keep(nil),
			BodyText: texts.M("alert.failed").Keep(i18n.Args{"count": h.Failed, "limit": st.AlertFailed}),
			Link:     "/w/" + ws.Slug + "/transactional/messages?status=failed&range=1h", Data: map[string]any{"failed": h.Failed, "limit": st.AlertFailed}}
		if err := s.raise(ctx, &ws, AlertFailed, nil, nil, float64(h.Failed), float64(st.AlertFailed), alertDetail{Failed: h.Failed}, now, &n); err != nil {
			return err
		}
	} else if _, err := s.Q.TxnAlertEnd(ctx, store.TxnAlertEndParams{Now: now, WorkspaceID: wsID, Kind: AlertFailed}); err != nil {
		return err
	}
	limitMs := int64(st.AlertMedianSeconds) * 1000
	if h.Delivered >= AlertMinSample && h.MedianMs > limitMs {
		sec := float64(h.MedianMs) / 1000
		n := notify.Notification{Kind: notify.KindTransactionalAlert, Severity: "warn", TitleText: texts.M("alert.title").Keep(nil),
			BodyText: texts.M("alert.median").Keep(i18n.Args{"seconds": oneDecimalNumber(sec), "limit": st.AlertMedianSeconds}),
			Link:     "/w/" + ws.Slug + "/transactional/messages?range=1h", Data: map[string]any{"median_ms": h.MedianMs, "limit_seconds": st.AlertMedianSeconds}}
		if err := s.raise(ctx, &ws, AlertMedianDelivery, nil, nil, sec, float64(st.AlertMedianSeconds), alertDetail{MedianMs: h.MedianMs}, now, &n); err != nil {
			return err
		}
	} else if _, err := s.Q.TxnAlertEnd(ctx, store.TxnAlertEndParams{Now: now, WorkspaceID: wsID, Kind: AlertMedianDelivery}); err != nil {
		return err
	}

	// Failovers: an incident while messages leave a provider, ended after
	// ten quiet minutes. The provider-down notification already went out.
	fos, err := s.Q.TxnAlertFailovers(ctx, store.TxnAlertFailoversParams{WorkspaceID: wsID, Since: now.Add(-failoverQuiet)})
	if err != nil {
		return err
	}
	if len(fos) == 0 {
		if _, err := s.Q.TxnAlertEnd(ctx, store.TxnAlertEndParams{Now: now, WorkspaceID: wsID, Kind: AlertFailover}); err != nil {
			return err
		}
		return nil
	}
	f := fos[0]
	for _, x := range fos[1:] {
		if x.Messages > f.Messages {
			f = x
		}
	}
	// The incident's totals since it started.
	since := f.FirstAt
	if cur, err := s.Q.TxnAlerts(ctx, store.TxnAlertsParams{WorkspaceID: wsID, Since: now}); err == nil {
		for _, a := range cur {
			if a.Kind == AlertFailover && a.EndedAt == nil && a.StartedAt.Before(since) {
				since = a.StartedAt
			}
		}
	}
	if all, err := s.Q.TxnAlertFailovers(ctx, store.TxnAlertFailoversParams{WorkspaceID: wsID, Since: since}); err == nil {
		for _, x := range all {
			if x.ProviderID == f.ProviderID {
				f = x
			}
		}
	}
	pid := f.ProviderID
	d := alertDetail{Messages: f.Messages, Lost: f.Lost, Fallback: f.FallbackName}
	if f.MedianMs >= 0 {
		d.MedianMs = f.MedianMs
	}
	return s.raise(ctx, &ws, AlertFailover, nil, &pid, float64(f.Messages), 0, d, f.FirstAt, nil)
}

// raise opens an alert (notifying members with transactional:view in the
// same transaction) or refreshes the open one.
func (s *Service) raise(ctx context.Context, ws *store.Workspace, kind string, template, provider *uuid.UUID, value, threshold float64, d alertDetail, at time.Time, n *notify.Notification) error {
	detail := mustJSON(d)
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		_, err := q.TxnAlertInsert(ctx, store.TxnAlertInsertParams{ID: newID(), WorkspaceID: ws.ID, Kind: kind, TemplateID: template, ProviderID: provider,
			Value: value, Threshold: threshold, Detail: detail, StartedAt: at})
		if errors.Is(err, pgx.ErrNoRows) {
			// Already open: keep the worst value and the latest detail.
			return q.TxnAlertRefresh(ctx, store.TxnAlertRefreshParams{Value: value, Detail: detail, WorkspaceID: ws.ID, Kind: kind, TemplateID: template})
		}
		if err != nil || n == nil {
			return err
		}
		n.WorkspaceID = &ws.ID
		return notify.ToMembersWith(ctx, tx, ws.ID, authz.TransactionalView, *n)
	})
}

// oneDecimalNumber is v rounded to one decimal, for a {n, number} argument
// each reader's region writes: 2.4, 41.
func oneDecimalNumber(v float64) float64 { return math.Round(v*10) / 10 }

// bounceArgs are a bounce-rate alert's arguments: the template, its rate
// and limit, and the two domains most bounces went to.
func bounceArgs(template string, rate, limit float64, ds []alertDomain) i18n.Args {
	args := i18n.Args{"template": template, "rate": oneDecimalNumber(rate), "limit": limit, "domains": min(2, len(ds)), "first": "", "second": ""}
	if len(ds) > 0 {
		args["first"] = ds[0].Domain
	}
	if len(ds) > 1 {
		args["second"] = ds[1].Domain
	}
	return args
}

// AlertsArgs checks the delivery alerts.
type AlertsArgs struct{}

// Kind implements river.JobArgs.
func (AlertsArgs) Kind() string { return "transactional.alerts" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (AlertsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Second}}
}

// AlertsWorker runs AlertsArgs.
type AlertsWorker struct {
	river.WorkerDefaults[AlertsArgs]
	S *Service
}

// Timeout bounds one check.
func (w *AlertsWorker) Timeout(*river.Job[AlertsArgs]) time.Duration { return 2 * time.Minute }

// Work implements river.Worker.
func (w *AlertsWorker) Work(ctx context.Context, _ *river.Job[AlertsArgs]) error {
	return w.S.CheckAlerts(ctx)
}

// TransactionalAlert is an open delivery alert or a past incident.
type TransactionalAlert struct {
	ID        string            `json:"id"`
	Kind      string            `json:"kind" enum:"bounce_rate,failed,median_delivery,failover"`
	Template  *TransactionalRef `json:"template,omitempty" doc:"id is the template's ID (password-reset)"`
	Provider  *TransactionalRef `json:"provider,omitempty" doc:"The provider messages failed over from"`
	Value     float64           `json:"value" doc:"bounce rate %, failed messages, median seconds or failed-over messages"`
	Threshold float64           `json:"threshold"`
	StartedAt time.Time         `json:"startedAt"`
	EndedAt   *time.Time        `json:"endedAt,omitempty"`
	SeenAt    *time.Time        `json:"seenAt,omitempty"`
	Sent      int64             `json:"sent,omitempty"`
	Bounced   int64             `json:"bounced,omitempty"`
	Hard      int64             `json:"hard,omitempty"`
	Domains   []alertDomain     `json:"domains,omitempty"`
	Messages  int64             `json:"messages,omitempty"`
	Lost      int64             `json:"lost,omitempty"`
	Fallback  string            `json:"fallback,omitempty"`
	MedianMs  int64             `json:"medianMs,omitempty"`
}

func alertOf(r store.TxnAlertsRow) TransactionalAlert {
	a := TransactionalAlert{ID: r.ID.String(), Kind: r.Kind, Value: r.Value, Threshold: r.Threshold, StartedAt: r.StartedAt, EndedAt: r.EndedAt, SeenAt: r.SeenAt}
	if r.TemplateID != nil {
		a.Template = &TransactionalRef{ID: r.TemplateKey, Name: r.TemplateName}
	}
	if r.ProviderID != nil {
		a.Provider = &TransactionalRef{ID: r.ProviderID.String(), Name: r.ProviderName}
	}
	var d alertDetail
	_ = json.Unmarshal(r.Detail, &d)
	a.Sent, a.Bounced, a.Hard, a.Domains, a.Messages, a.Lost, a.Fallback, a.MedianMs = d.Sent, d.Bounced, d.Hard, d.Domains, d.Messages, d.Lost, d.Fallback, d.MedianMs
	for i := range a.Domains {
		if fix, ok := emailaddr.TypoFix(a.Domains[i].Domain); ok {
			a.Domains[i].TypoOf = fix
		}
	}
	return a
}

// MarkAlertSeen hides an open alert from the overview's banner.
func (s *Service) MarkAlertSeen(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	var user *uuid.UUID
	if p != nil && p.Type == authz.ActorUser {
		u := p.UserID
		user = &u
	}
	n, err := s.Q.TxnAlertSeen(ctx, store.TxnAlertSeenParams{Now: s.now(), UserID: user, ID: id, WorkspaceID: ws.ID})
	if err == nil && n == 0 {
		return problem(404, "not_found", say(ctx, "alert.notFound", nil))
	}
	return err
}
