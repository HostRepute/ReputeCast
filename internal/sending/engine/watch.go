package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/mailbox"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/sending/rules"
	"github.com/hostrepute/reputecast/internal/store"
)

// The complaint watch after a send (review GS-6). Complaints keep arriving
// for days after delivery, so when a send ends its complaints are watched
// for 72 hours (Traffic rules › After the send): a mailbox provider that
// passes its complaint limit with enough evidence, or the send as a whole,
// raises an alert for the people who can resume sends. Nothing is paused:
// the send is over; the alert says what to look at before the next one.

// startWatch starts a finished send's complaint watch.
func startWatch(ctx context.Context, q *store.Queries, c store.SendingCampaignContextRow, now time.Time) error {
	_, w, err := rules.Load(ctx, q, c.WorkspaceID)
	if err != nil || !w.Enabled {
		return err
	}
	until := now.Add(time.Duration(w.Hours) * time.Hour)
	return q.SendingSetWatch(ctx, store.SendingSetWatchParams{CampaignID: c.ID, WatchUntil: &until})
}

// ComplaintWatchArgs runs the complaint watch of every send that has one.
type ComplaintWatchArgs struct{}

// Kind implements river.JobArgs.
func (ComplaintWatchArgs) Kind() string { return "sending.complaint_watch" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (ComplaintWatchArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 9 * time.Minute}}
}

// ComplaintWatchWorker runs WatchComplaints.
type ComplaintWatchWorker struct {
	river.WorkerDefaults[ComplaintWatchArgs]
	E *Engine
}

// Work implements river.Worker.
func (w *ComplaintWatchWorker) Work(ctx context.Context, _ *river.Job[ComplaintWatchArgs]) error {
	return w.E.WatchComplaints(ctx)
}

// WatchComplaints checks the complaints of every watched send and ends
// the watches whose time is over.
func (e *Engine) WatchComplaints(ctx context.Context) error {
	now := e.now()
	ids, err := e.Q.SendingWatchDue(ctx, now)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := e.watchCampaign(ctx, id, now); err != nil {
			e.log().WarnContext(ctx, "complaint watch", "campaign", id, "err", err)
		}
	}
	return e.Q.SendingEndWatches(ctx, now)
}

// watchTrip is a scope over its complaint limit during the watch.
type watchTrip struct {
	mailbox      string // "" for the whole send
	events, base int64
	value, limit float64
}

// watchTrips are the scopes over their complaint limit with enough
// evidence: each mailbox provider that reports complaints, then the whole
// send where they're reported.
func watchTrips(th Thresholds, ts tallies) []watchTrip {
	var out []watchTrip
	for _, k := range routing.MailboxProviders {
		m := ts.mailbox[k]
		if m == nil || !mailbox.ReportsComplaints(k) {
			continue
		}
		if m.reported >= minSampleScoped && over(m.complained, m.reported, th.MailboxComplaint, minEvents) {
			out = append(out, watchTrip{mailbox: k, events: m.complained, base: m.reported, value: m.complaintRate(), limit: th.MailboxComplaint})
		}
	}
	t := ts.total
	if t.reported >= minSampleCampaign && over(t.complained, t.reported, th.Complaint, minEvents) {
		out = append(out, watchTrip{events: t.complained, base: t.reported, value: t.complaintRate(), limit: th.Complaint})
	}
	return out
}

// watchCampaign raises the alerts a watched send has earned; each scope
// alerts once.
func (e *Engine) watchCampaign(ctx context.Context, id uuid.UUID, now time.Time) error {
	cc, err := e.Q.SendingCampaignContext(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	_, w, err := rules.Load(ctx, e.Q, cc.WorkspaceID)
	if err != nil || !w.Alert {
		return err
	}
	snd, err := e.Q.SendingGetSend(ctx, id)
	if err != nil {
		return err
	}
	ts, err := e.tallies(ctx, id)
	if err != nil {
		return err
	}
	trips := watchTrips(sendThresholds(snd), ts)
	if len(trips) == 0 {
		return nil
	}
	alerted, err := e.Q.SendingIncidentsOfKind(ctx, store.SendingIncidentsOfKindParams{CampaignID: id, Kind: "complaint_watch"})
	if err != nil {
		return err
	}
	for _, tr := range trips {
		done := false
		for _, in := range alerted {
			done = done || in.MailboxProvider == tr.mailbox
		}
		if done {
			continue
		}
		if err := e.watchAlert(ctx, cc, snd, tr, w.Hours, now); err != nil {
			return err
		}
	}
	return nil
}

// watchAlert records a complaint alert after the send and tells the people
// who can resume sends.
func (e *Engine) watchAlert(ctx context.Context, c store.SendingCampaignContextRow, snd store.CampaignSend, tr watchTrip, hours int, now time.Time) error {
	scope, who := "campaign", ""
	if tr.mailbox != "" {
		scope, who = "mailbox_provider", mailboxName(tr.mailbox)
	}
	words := i18n.Args{"name": quoted(c.Name), "mailbox": tr.mailbox, "who": who, "scope": scope, "valuePct": round2(tr.value), "limitPct": tr.limit,
		"events": tr.events, "base": tr.base, "watchHours": int64(hours)}
	d := IncidentDetail{Events: tr.events, Base: tr.base, Delivered: tr.base}
	d.setNote(keep("note.watch", words))
	d.addCause(keep("cause.watchLead", words), keep("cause.watch", words))
	d.addRecommendation(keep("rec.watchSuppressed", words))
	d.addRecommendation(keep("rec.watchNext", words))
	d.setOptions([]option{{key: "acknowledge", recommended: true, title: keep("opt.acknowledgeTitle", nil), body: keep("opt.acknowledgeBody", words), action: keep("opt.acknowledge", nil)}})
	value, limit := round4(tr.value), round4(tr.limit)
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		id := newID()
		if err := q.SendingInsertIncident(ctx, store.SendingInsertIncidentParams{ID: id, CampaignID: c.ID, Kind: "complaint_watch", Scope: scope,
			MailboxProvider: tr.mailbox, Metric: "complaint", Value: &value, Threshold: &limit, Held: 0, Detail: d.stored(), TrippedAt: now}); err != nil {
			return err
		}
		if err := notify.ToMembersWith(ctx, tx, c.WorkspaceID, authz.CampaignsSend, notify.Notification{
			Kind: notify.KindComplaintWatch, Severity: "warn", TitleText: keep("notify.watchTitle", words), BodyText: keep("notify.watchBody", words),
			Link: liveLink(c.WorkspaceSlug, c.ID), Data: map[string]any{"campaignId": c.ID.String(), "incidentId": id.String(), "mailbox": tr.mailbox},
		}); err != nil {
			return err
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.complaint_alert", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Actor: systemActor, Metadata: map[string]any{"scope": scope, "mailbox_provider": tr.mailbox, "value": value, "limit": limit, "complaints": tr.events, "delivered": tr.base,
				"summary": fmt.Sprintf("%s complaints reached %s after the send (limit %s)", firstNonEmpty(who, "The send's"), fmtRate(tr.value), fmtRate(tr.limit)), "source": "complaint watch"}})
	})
}

// Acknowledge closes an alert a person has read: a complaint alert after
// the send.
func (e *Engine) Acknowledge(ctx context.Context, campaignID, by, incidentID uuid.UUID) error {
	now := e.now()
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := q.SendingCampaignForUpdate(ctx, campaignID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		open, err := q.SendingOpenIncidents(ctx, campaignID)
		if err != nil {
			return err
		}
		var in *store.CampaignIncident
		for i := range open {
			if open[i].ID == incidentID && open[i].Kind == "complaint_watch" {
				in = &open[i]
			}
		}
		if in == nil {
			return stateErr("state.nothingToAcknowledge", campaignArgs(c.Name, c.Status))
		}
		ref, name := e.actorRef(ctx, q, by)
		if err := q.SendingResolveIncident(ctx, store.SendingResolveIncidentParams{ID: in.ID, At: &now, Resolution: "acknowledged", ResolvedBy: ref}); err != nil {
			return err
		}
		if !records(ctx) {
			return nil
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.alert_acknowledged", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Actor: e.actorFor(ctx, ref, name), Metadata: map[string]any{"incident": in.ID.String(), "kind": in.Kind, "mailbox_provider": in.MailboxProvider}})
	})
}
