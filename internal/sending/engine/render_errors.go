package engine

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// A recipient the template fails for (review CO-4): Send Readiness blocks
// syntax errors, but a recipient's own data can still break a render (a
// filter on a value of the wrong type, output over the limit). That
// message fails with the error and the others keep going out: one
// recipient's data says nothing about the rest. The send gets one notice
// (an incident of kind render_error) that counts them, with a few masked
// samples, for the people who can send; it tells them once, and they
// acknowledge it on the live monitor.

// renderSamples is how many failed recipients a notice keeps as samples.
const renderSamples = 3

// renderFailure is one recipient the template failed for.
type renderFailure struct {
	email string
	err   string
}

// noteRenderErrors adds a batch's failed recipients to the send's notice,
// raising it on the first.
func (e *Engine) noteRenderErrors(ctx context.Context, rs *runState, fails []renderFailure, now time.Time) error {
	if len(fails) == 0 {
		return nil
	}
	c := rs.campaign
	return db.Tx(context.WithoutCancel(ctx), e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		// One notice per send: later failures add to it (even once it was
		// acknowledged) without telling anyone again.
		known, err := q.SendingIncidentsOfKind(ctx, store.SendingIncidentsOfKindParams{CampaignID: c.ID, Kind: "render_error"})
		if err != nil {
			return err
		}
		if len(known) > 0 {
			open := known[len(known)-1]
			d := parseDetail(open.Detail)
			d.Events += int64(len(fails))
			for _, f := range fails {
				if len(d.Log) >= renderSamples {
					break
				}
				d.Log = append(d.Log, IncidentLogLine{At: now, Text: MaskAddress(f.email) + ": " + f.err, Bad: true})
			}
			words := renderWords(c.Name, d.Events)
			d.Causes, d.Recommendations, d.texts = nil, nil, nil
			d.addCause(keep("cause.renderLead", words), keep("cause.render", words))
			d.addRecommendation(keep("rec.render", words))
			return q.SendingUpdateIncidentDetail(ctx, store.SendingUpdateIncidentDetailParams{ID: open.ID, Detail: d.stored(), Held: 0})
		}
		d := IncidentDetail{Events: int64(len(fails)), Action: "alerted"}
		for _, f := range fails {
			if len(d.Log) >= renderSamples {
				break
			}
			d.Log = append(d.Log, IncidentLogLine{At: now, Text: MaskAddress(f.email) + ": " + f.err, Bad: true})
		}
		words := renderWords(c.Name, d.Events)
		d.addCause(keep("cause.renderLead", words), keep("cause.render", words))
		d.addRecommendation(keep("rec.render", words))
		id := newID()
		count := float64(len(fails))
		if err := q.SendingInsertIncident(ctx, store.SendingInsertIncidentParams{ID: id, CampaignID: c.ID, Kind: "render_error", Scope: "campaign", Metric: "render",
			Value: &count, Detail: d.stored(), TrippedAt: now}); err != nil {
			return err
		}
		if err := notify.ToMembersWith(ctx, tx, c.WorkspaceID, authz.CampaignsSend, notify.Notification{
			Kind: notify.KindRenderFailed, Severity: "warn", TitleText: keep("notify.renderTitle", words), BodyText: keep("notify.renderBody", words),
			Link: liveLink(rs.campaign.WorkspaceSlug, c.ID), Data: map[string]any{"campaignId": c.ID.String(), "incidentId": id.String()},
		}); err != nil {
			return err
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.render_failed", TargetType: "campaign", TargetID: c.ID.String(),
			TargetName: c.Name, Actor: e.systemFor(ctx, c.ID), Metadata: map[string]any{"recipients": len(fails),
				"summary": "The template failed for " + countText(int64(len(fails))) + " recipients · the others keep getting it"}})
	})
}

func renderWords(name string, n int64) i18n.Args {
	return i18n.Args{"name": quoted(name), "failed": n}
}
