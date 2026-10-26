package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Resuming after an auto-pause (review GS-2). Every scope a person resumes
// (the whole campaign, a mailbox provider, a sending provider) is judged
// from then on by what happened since, so the events that tripped it never
// pause it again; resuming with changes works at every scope; and the
// send's thresholds can be changed while it sends or waits, audited.

// resumedScopes are the tripped scopes a resume releases.
type resumedScopes struct {
	campaign  bool
	mailboxes []string
	routes    []uuid.UUID
}

func (s resumedScopes) empty() bool {
	return !s.campaign && len(s.mailboxes) == 0 && len(s.routes) == 0
}

// trippedScopes are what a threshold holds in a campaign: the campaign
// itself, mailbox providers on hold and routes paused for their bounces
// (a route held for an outage isn't a threshold's).
func trippedScopes(ctx context.Context, q *store.Queries, campaignID uuid.UUID) (resumedScopes, error) {
	var out resumedScopes
	snd, err := q.SendingGetSend(ctx, campaignID)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	for _, k := range sortedKeys(parseHolds(snd.Holds)) {
		if h := parseHolds(snd.Holds)[k]; !h.Resumed {
			out.mailboxes = append(out.mailboxes, k)
		}
	}
	routes, err := q.SendingRoutes(ctx, campaignID)
	if err != nil {
		return out, err
	}
	for _, r := range routes {
		if r.Status == "paused" {
			out.routes = append(out.routes, r.ProviderID)
		}
	}
	open, err := q.SendingOpenIncidents(ctx, campaignID)
	if err != nil {
		return out, err
	}
	for _, in := range open {
		out.campaign = out.campaign || (in.Kind == "threshold" && in.Scope == "campaign")
	}
	return out, nil
}

// rearmScopes re-arms the thresholds of the scopes a person resumed from
// the current counts. A mailbox provider or route that joins the
// campaign-wide check again brings its events so far out of it too.
func (e *Engine) rearmScopes(ctx context.Context, q *store.Queries, campaignID uuid.UUID, s resumedScopes, now time.Time) error {
	if s.empty() {
		return nil
	}
	snd, err := q.SendingGetSend(ctx, campaignID)
	if err != nil {
		return err
	}
	rows, err := q.SendingCounters(ctx, campaignID)
	if err != nil {
		return err
	}
	ts := tallyRows(rows)
	g := parseGuard(snd.Guard)
	for _, m := range s.mailboxes {
		t := tally{}
		if x := ts.mailbox[m]; x != nil {
			t = *x
		}
		g.rearm(rearmMailbox(m), countsOf(t))
		if !s.campaign {
			g.rejoin(t)
		}
	}
	for _, p := range s.routes {
		t := tally{}
		if x := ts.provider[p]; x != nil {
			t = *x
		}
		g.rearm(rearmRoute(p), countsOf(t))
		if !s.campaign {
			g.rejoin(t)
		}
	}
	if s.campaign {
		// With the campaign scope any trip pauses everything, the mailbox
		// provider's or route's that tripped included: all of them re-arm.
		g.rearm(rearmCampaign, countsOf(ts.total))
		for k, t := range ts.mailbox {
			g.rearm(rearmMailbox(k), countsOf(*t))
		}
		for p, t := range ts.provider {
			g.rearm(rearmRoute(p), countsOf(*t))
		}
	}
	return saveGuard(ctx, q, campaignID, g, now)
}

// ThresholdChange is a person's change of a running or waiting send's
// thresholds, in percent; nil leaves one as it is.
type ThresholdChange struct {
	Bounce           *float64
	Complaint        *float64
	MailboxComplaint *float64
	Deferral         *float64
	APIErrors        *float64
	// Scope is "route" or "campaign" ("" leaves it).
	Scope string
}

// thresholdRange is what a person may set a threshold to, in percent (the
// ranges Review & send allows).
var thresholdRange = map[string][2]float64{
	"bounce": {0.1, 20}, "complaint": {0.01, 5}, "mailbox_complaint": {0.01, 5}, "deferral": {1, 100}, "api_errors": {1, 100},
}

// apply writes the change into thresholds; it returns the key of a value
// out of its range.
func (c ThresholdChange) apply(t *Thresholds) (string, bool) {
	for _, f := range []struct {
		key string
		v   *float64
		dst *float64
	}{{"bounce", c.Bounce, &t.Bounce}, {"complaint", c.Complaint, &t.Complaint}, {"mailbox_complaint", c.MailboxComplaint, &t.MailboxComplaint},
		{"deferral", c.Deferral, &t.Deferral}, {"api_errors", c.APIErrors, &t.APIErrors}} {
		if f.v == nil {
			continue
		}
		r := thresholdRange[f.key]
		if *f.v < r[0] || *f.v > r[1] || math.IsNaN(*f.v) {
			return f.key, false
		}
		*f.dst = math.Round(*f.v*100) / 100
	}
	if c.Scope == "route" || c.Scope == "campaign" {
		t.Scope = c.Scope
	}
	return "", true
}

// sendThresholds are a send's thresholds as the guard reads them.
func sendThresholds(snd store.CampaignSend) Thresholds {
	t := ParseThresholds(snd.Thresholds, nil)
	if len(snd.Thresholds) > 0 {
		_ = json.Unmarshal(snd.Thresholds, &t)
	}
	return t
}

// changeThresholds applies a change to a send's thresholds in a
// transaction and returns them before and after.
func changeThresholds(ctx context.Context, q *store.Queries, c store.Campaign, ch ThresholdChange, now time.Time) (Thresholds, Thresholds, error) {
	snd, err := q.SendingGetSendForUpdate(ctx, c.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Thresholds{}, Thresholds{}, stateErr("state.noSend", campaignArgs(c.Name, c.Status))
	}
	if err != nil {
		return Thresholds{}, Thresholds{}, err
	}
	before := sendThresholds(snd)
	after := before
	if key, ok := ch.apply(&after); !ok {
		r := thresholdRange[key]
		return before, after, stateErr("state.thresholdRange", i18n.Args{"metric": key, "minPct": r[0], "maxPct": r[1]})
	}
	b, _ := json.Marshal(after)
	return before, after, q.SendingSetSendThresholds(ctx, store.SendingSetSendThresholdsParams{CampaignID: c.ID, Thresholds: b, At: now})
}

// SetThresholds changes a running or waiting send's thresholds, audited
// with what they were (review GS-2). The guard reads them again within
// seconds.
func (e *Engine) SetThresholds(ctx context.Context, campaignID, by uuid.UUID, ch ThresholdChange) error {
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
		if c.Status != "sending" && c.Status != "paused" {
			return stateErr("state.notSending", campaignArgs(c.Name, c.Status))
		}
		before, after, err := changeThresholds(ctx, q, c, ch, now)
		if err != nil {
			return err
		}
		if !records(ctx) {
			return nil
		}
		ref, name := e.actorRef(ctx, q, by)
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.thresholds_changed", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Actor: e.actorFor(ctx, ref, name), Before: before, After: after, Metadata: map[string]any{"summary": thresholdSummary(before, after), "status": c.Status}})
	})
}

// thresholdSummary says what changed ("Complaints per mailbox provider
// 0.10% → 0.20%").
func thresholdSummary(before, after Thresholds) string {
	out := ""
	add := func(label string, a, b float64) {
		if a == b {
			return
		}
		if out != "" {
			out += " · "
		}
		out += fmt.Sprintf("%s %s → %s", label, fmtPct(a), fmtPct(b))
	}
	add("Bounces", before.Bounce, after.Bounce)
	add("Complaints", before.Complaint, after.Complaint)
	add("Complaints per mailbox provider", before.MailboxComplaint, after.MailboxComplaint)
	add("Deferrals", before.Deferral, after.Deferral)
	add("API errors", before.APIErrors, after.APIErrors)
	if before.Scope != after.Scope {
		if out != "" {
			out += " · "
		}
		out += "Scope " + before.Scope + " → " + after.Scope
	}
	if out == "" {
		out = "No change"
	}
	return out
}

// limitChange is the change of the tripped metric's limit a person chose
// when resuming ("Resume with the threshold raised to 0.20% for this
// campaign only").
func limitChange(in store.CampaignIncident, limit float64) ThresholdChange {
	v := limit
	var ch ThresholdChange
	switch {
	case in.Metric == "complaint" && in.Scope == "mailbox_provider":
		ch.MailboxComplaint = &v
	case in.Metric == "complaint":
		ch.Complaint = &v
	case in.Metric == "deferral":
		ch.Deferral = &v
	default:
		ch.Bounce = &v
	}
	return ch
}
