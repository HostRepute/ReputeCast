package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/warmup"
)

// ResumeRequest resumes a mailbox provider or route on hold.
type ResumeRequest struct {
	// Mailbox or ProviderID names the scope (neither = the whole campaign).
	Mailbox    string
	ProviderID *uuid.UUID
	// WithChanges applies the incident's recommended changes: exclude a
	// group, lower the rate, hand the excluded contacts to a flow.
	WithChanges bool
	// Changes override the incident's recommendation.
	Changes *ResumeChanges
	// Limit raises the limit of the metric that tripped, for this send
	// only, in percent (review GS-2).
	Limit *float64
}

// ResumeResult says what resuming changed.
type ResumeResult struct {
	Excluded   int64   `json:"excluded"`
	Rate       float64 `json:"rate,omitempty"`
	Automation string  `json:"automation,omitempty"`
}

// ResumeScope resumes a held mailbox provider or route, or the whole
// campaign, as is or with the recommended changes, optionally raising the
// limit that tripped. The resumed scope's thresholds re-arm from the
// current counts (review GS-2).
func (e *Engine) ResumeScope(ctx context.Context, campaignID, by uuid.UUID, in ResumeRequest) (ResumeResult, error) {
	if in.Mailbox == "" && in.ProviderID == nil {
		return e.resumeCampaign(ctx, campaignID, by, in)
	}
	now := e.now()
	var out ResumeResult
	err := db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
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
		ref, name := e.actorRef(ctx, q, by)
		snd, err := q.SendingGetSendForUpdate(ctx, campaignID)
		if err != nil {
			return err
		}
		var incident *store.CampaignIncident
		open, err := q.SendingOpenIncidents(ctx, campaignID)
		if err != nil {
			return err
		}
		for i := range open {
			in2 := open[i]
			if isNotice(in2) {
				continue
			}
			if (in.Mailbox != "" && in2.Scope == "mailbox_provider" && in2.MailboxProvider == in.Mailbox) ||
				(in.ProviderID != nil && in2.ProviderID != nil && *in2.ProviderID == *in.ProviderID) {
				incident = &open[i]
				break
			}
		}
		resolution := "resumed"
		if in.Mailbox != "" {
			holds := parseHolds(snd.Holds)
			h, ok := holds[in.Mailbox]
			if !ok || h.Resumed {
				return stateErr("state.mailboxNotHeld", i18n.Args{"mailbox": mailboxName(in.Mailbox)})
			}
			ts, err := e.tallies(ctx, campaignID)
			if err != nil {
				return err
			}
			m := ts.mailbox[in.Mailbox]
			if m == nil {
				m = &tally{}
			}
			h.Resumed = true
			h.Rearm = map[string]int64{"complaint": m.complained, "bounce": m.bounced(), "deferral": m.deferred}
			holds[in.Mailbox] = h
			if in.WithChanges || in.Changes != nil {
				ch := in.Changes
				if ch == nil && incident != nil {
					ch = parseDetail(incident.Detail).Changes
				}
				if ch != nil {
					resolution = "resumed_with_changes"
					if err := e.applyChanges(ctx, q, tx, c, snd, in.Mailbox, nil, ch, &out, now); err != nil {
						return err
					}
				}
			}
			b, _ := json.Marshal(holds)
			if err := q.SendingSetHolds(ctx, store.SendingSetHoldsParams{CampaignID: campaignID, Holds: b, At: now}); err != nil {
				return err
			}
			if err := e.rearmScopes(ctx, q, campaignID, resumedScopes{mailboxes: []string{in.Mailbox}}, now); err != nil {
				return err
			}
		} else {
			routes, err := q.SendingRoutes(ctx, campaignID)
			if err != nil {
				return err
			}
			found, tripped := false, false
			for _, r := range routes {
				if r.ProviderID == *in.ProviderID && (r.Status == "held" || r.Status == "paused") {
					// A share today's warm-up cap holds has its own choices
					// (warmup.DecideCapHold, review PR-8).
					if parseRouteDetail(r.StatusDetail).Reason == CapHoldReason {
						p, _ := q.ProvGetAny(ctx, r.ProviderID)
						return stateErr("state.warmupHold", i18n.Args{"provider": state.Short(p.Name, p.Adapter)})
					}
					// "paused": its bounces tripped the threshold; "held": an outage.
					found, tripped = true, r.Status == "paused"
				}
			}
			if !found {
				return stateErr("state.shareNotHeld", nil)
			}
			if in.WithChanges || in.Changes != nil {
				ch := in.Changes
				if ch == nil && incident != nil {
					ch = parseDetail(incident.Detail).Changes
				}
				if ch != nil {
					resolution = "resumed_with_changes"
					if err := e.applyChanges(ctx, q, tx, c, snd, "", in.ProviderID, ch, &out, now); err != nil {
						return err
					}
				}
			}
			if err := q.SendingSetRouteStatus(ctx, store.SendingSetRouteStatusParams{CampaignID: campaignID, ProviderID: *in.ProviderID, Status: "sending", StatusDetail: `{"decision":"resume"}`}); err != nil {
				return err
			}
			e.resetCalls(routeKey{campaignID, *in.ProviderID})
			if tripped {
				if err := e.rearmScopes(ctx, q, campaignID, resumedScopes{routes: []uuid.UUID{*in.ProviderID}}, now); err != nil {
					return err
				}
			}
		}
		if in.Limit != nil {
			if incident == nil || incident.Kind != "threshold" {
				return stateErr("state.nothingTripped", campaignArgs(c.Name, c.Status))
			}
			if _, _, err := changeThresholds(ctx, q, c, limitChange(*incident, *in.Limit), now); err != nil {
				return err
			}
		}
		if incident != nil {
			if err := q.SendingResolveIncident(ctx, store.SendingResolveIncidentParams{ID: incident.ID, At: &now, Resolution: resolution, ResolvedBy: ref}); err != nil {
				return err
			}
		}
		// A decision on one share never undoes a person's pause (review GS-5).
		if err := e.unpause(ctx, q, c, now); err != nil {
			return err
		}
		scope := in.Mailbox
		data := campaignData(c.ID, c.Name, "resumed_by", firstNonEmpty(name, "System"), "scope", "mailbox_provider", "mailbox_provider", in.Mailbox)
		if in.ProviderID != nil {
			scope = in.ProviderID.String()
			data["scope"], data["provider_id"] = "route", scope
			delete(data, "mailbox_provider")
		}
		withChanges := resolution == "resumed_with_changes"
		if withChanges {
			data["with_changes"], data["excluded"] = true, out.Excluded
		}
		if err := emit(ctx, tx, c.WorkspaceID, "campaign.resumed", data); err != nil {
			return err
		}
		if !records(ctx) {
			return nil
		}
		meta := map[string]any{}
		if withChanges {
			meta["summary"] = fmt.Sprintf("%s resumed with changes · %s left out", mailboxName(in.Mailbox), countText(out.Excluded))
			if in.ProviderID != nil {
				meta["summary"] = fmt.Sprintf("Resumed with changes · %s left out", countText(out.Excluded))
			}
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.resumed", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Actor: e.actorFor(ctx, ref, name), After: map[string]any{"scope": scope, "with_changes": withChanges, "excluded": out.Excluded, "rate": out.Rate, "automation": out.Automation},
			Metadata: meta})
	})
	if err == nil {
		e.kick(ctx, campaignID)
	}
	return out, err
}

// applyChanges takes a group out of the rest of the send (or of a mailbox
// provider's or a sending provider's share), lowers the scope's rate and
// hands the excluded contacts to a flow.
func (e *Engine) applyChanges(ctx context.Context, q *store.Queries, tx pgx.Tx, c store.Campaign, snd store.CampaignSend, mailbox string, provider *uuid.UUID, ch *ResumeChanges, out *ResumeResult, now time.Time) error {
	cid := c.ID
	var contacts []uuid.UUID
	if ch.Exclude > 0 {
		why := map[string]string{groupImported: "never-engaged import", groupLapsed: "no open in more than 90 days"}[ch.group()]
		if why == "" {
			why = "never engaged"
		}
		share := "the campaign"
		switch {
		case mailbox != "":
			share = "the " + mailboxName(mailbox) + " share"
		case provider != nil:
			share = "this sending provider's share"
		}
		reason := "Excluded when " + share + " resumed: " + why
		var ids []*uuid.UUID
		var err error
		if provider != nil {
			ids, err = q.SendingExcludeGroupOn(ctx, store.SendingExcludeGroupOnParams{Reason: reason, CampaignID: &cid, ProviderID: provider, Grp: ch.group(), Source: ch.Source})
		} else {
			ids, err = q.SendingExcludeGroup(ctx, store.SendingExcludeGroupParams{Reason: reason, CampaignID: &cid, MailboxProvider: mailbox, Grp: ch.group(), Source: ch.Source})
		}
		if err != nil {
			return err
		}
		out.Excluded = int64(len(ids))
		contacts = derefIDs(ids)
	}
	if ch.Rate > 0 && provider != nil {
		// The route sends slower for the rest of this send.
		g := parseGuard(snd.Guard)
		if g.RouteRates == nil {
			g.RouteRates = map[string]float64{}
		}
		g.RouteRates[provider.String()] = ch.Rate
		if err := saveGuard(ctx, q, cid, g, now); err != nil {
			return err
		}
		out.Rate = ch.Rate
	}
	if ch.Rate > 0 && mailbox != "" {
		th := map[string]float64{}
		_ = json.Unmarshal(snd.Throttles, &th)
		th[mailbox] = ch.Rate
		b, _ := json.Marshal(th)
		if err := q.SendingSetThrottles(ctx, store.SendingSetThrottlesParams{CampaignID: cid, Throttles: b, At: now}); err != nil {
			return err
		}
		out.Rate = ch.Rate
	}
	if ch.AutomationID != "" && len(contacts) > 0 {
		aid, err := uuid.Parse(ch.AutomationID)
		if err == nil {
			var name string
			var trigger json.RawMessage
			if err := tx.QueryRow(ctx, `SELECT name, trigger FROM automations WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL`, aid, c.WorkspaceID).Scan(&name, &trigger); err == nil {
				var t struct {
					Kind   string `json:"kind"`
					Tag    string `json:"tag"`
					ListID string `json:"listId"`
				}
				_ = json.Unmarshal(trigger, &t)
				switch t.Kind {
				case "tag_added":
					if t.Tag != "" {
						if _, err := tx.Exec(ctx, `UPDATE contacts SET tags = array_append(tags, $2), updated_at = now() WHERE id = ANY($1) AND NOT ($2 = ANY(tags))`, contacts, t.Tag); err != nil {
							return err
						}
						out.Automation = name
					}
				case "joined_list":
					if lid, err := uuid.Parse(t.ListID); err == nil {
						if _, err := tx.Exec(ctx, `INSERT INTO list_memberships (list_id, contact_id, status, source, subscribed_at)
							SELECT $1, unnest($2::uuid[]), 'subscribed', 'campaign_resume', now() ON CONFLICT (list_id, contact_id) DO NOTHING`, lid, contacts); err != nil {
							return err
						}
						out.Automation = name
					}
				}
			}
		}
	}
	return nil
}

// CancelShare skips what is left of a mailbox provider or a route; the
// rest of the campaign finishes normally.
func (e *Engine) CancelShare(ctx context.Context, campaignID, by uuid.UUID, mailbox string, provider *uuid.UUID) (int64, error) {
	if mailbox == "" && provider == nil {
		return 0, e.Cancel(ctx, campaignID, by)
	}
	now := e.now()
	var n int64
	err := db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
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
		ref, name := e.actorRef(ctx, q, by)
		who := firstNonEmpty(name, "a member")
		cid := c.ID
		p := store.SendingCancelRemainingParams{CampaignID: &cid, ProviderID: provider}
		scope := ""
		if mailbox != "" {
			p.MailboxProvider = &mailbox
			p.Reason = fmt.Sprintf("The %s share was cancelled by %s", mailboxName(mailbox), who)
			scope = mailbox
		} else {
			p.Reason = "This provider's share was cancelled by " + who
			scope = provider.String()
		}
		if n, err = q.SendingCancelRemaining(ctx, p); err != nil {
			return err
		}
		snd, err := q.SendingGetSendForUpdate(ctx, campaignID)
		if err != nil {
			return err
		}
		if mailbox != "" {
			holds := parseHolds(snd.Holds)
			delete(holds, mailbox)
			b, _ := json.Marshal(holds)
			if err := q.SendingSetHolds(ctx, store.SendingSetHoldsParams{CampaignID: campaignID, Holds: b, At: now}); err != nil {
				return err
			}
		} else {
			d, _ := json.Marshal(map[string]any{"decision": "cancel", "cancelled": n})
			if err := q.SendingSetRouteStatus(ctx, store.SendingSetRouteStatusParams{CampaignID: campaignID, ProviderID: *provider, Status: "cancelled", StatusDetail: string(d)}); err != nil {
				return err
			}
			if err := e.adjustPlanned(ctx, q, campaignID, *provider, -n); err != nil {
				return err
			}
		}
		open, err := q.SendingOpenIncidents(ctx, campaignID)
		if err != nil {
			return err
		}
		for _, in := range open {
			if (mailbox != "" && in.MailboxProvider == mailbox) || (provider != nil && in.ProviderID != nil && *in.ProviderID == *provider) {
				if err := q.SendingResolveIncident(ctx, store.SendingResolveIncidentParams{ID: in.ID, At: &now, Resolution: "cancelled", ResolvedBy: ref}); err != nil {
					return err
				}
			}
		}
		// The rest of the campaign finishes, unless a person paused it.
		if err := e.unpause(ctx, q, c, now); err != nil {
			return err
		}
		if !records(ctx) {
			return nil
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.share_cancelled", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Actor: e.actorFor(ctx, ref, name), After: map[string]any{"scope": scope, "not_sent": n}, Metadata: map[string]any{"summary": p.Reason + " · " + countText(n) + " not sent"}})
	})
	if err == nil {
		e.kick(ctx, campaignID)
	}
	return n, err
}

// MoveShare moves what is left of one provider's share to another (a
// person's decision during an outage, from the live monitor or the
// providers screen; it implements sending.ShareMover). The target is vetted
// as every move is; A/B and canary holdouts stay held on the new provider.
func (e *Engine) MoveShare(ctx context.Context, campaignID, by, from, to uuid.UUID) (int64, error) {
	now := e.now()
	var moved int64
	err := db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
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
		target, err := q.ProvGet(ctx, store.ProvGetParams{WorkspaceID: c.WorkspaceID, ID: to})
		if err != nil {
			return stateErr("state.providerNotConnected", nil)
		}
		if to == from {
			return stateErr("state.providerUnavailable", i18n.Args{"provider": target.Name})
		}
		cc, err := q.SendingCampaignContext(ctx, campaignID)
		if err != nil {
			return err
		}
		left, err := q.SendingRemainingOn(ctx, store.SendingRemainingOnParams{CampaignID: &campaignID, ProviderID: &from})
		if err != nil {
			return err
		}
		vet := e.vetTarget(ctx, q, senderVerification(ctx, q, c.WorkspaceID, cc.SenderID), target, left, state.Location(cc.WorkspaceTimezone))
		if failed, ok := vet.Failed(); ok {
			return vetErr(failed, target.Name)
		}
		fromP, err := q.ProvGetAny(ctx, from)
		if err != nil {
			return ErrNotFound
		}
		ref, name := e.actorRef(ctx, q, by)
		if moved, err = e.moveShare(ctx, q, campaignID, from, to); err != nil {
			return err
		}
		if err := e.shiftWarmup(ctx, q, tx, c.WorkspaceID, fromP, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE campaign_incidents SET resolved_at = $3, resolution = 'moved', resolved_by = $4
			WHERE campaign_id = $1 AND provider_id = $2 AND resolved_at IS NULL`, campaignID, from, now, ref); err != nil {
			return err
		}
		if err := e.unpause(ctx, q, c, now); err != nil {
			return err
		}
		if err := emit(ctx, tx, c.WorkspaceID, "campaign.resumed", campaignData(c.ID, c.Name, "resumed_by", firstNonEmpty(name, "System"), "scope", "route",
			"provider_id", from.String(), "provider", fromP.Name, "moved_to", target.Name, "moved", moved)); err != nil {
			return err
		}
		if !records(ctx) {
			return nil
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.share_moved", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Actor: e.actorFor(ctx, ref, name), After: map[string]any{"from": fromP.Name, "to": target.Name, "moved": moved},
			Metadata: map[string]any{"summary": fmt.Sprintf("%s recipients moved from %s to %s", countText(moved), fromP.Name, target.Name)}})
	})
	if err == nil {
		e.resetCalls(routeKey{campaignID, from})
		e.kick(ctx, campaignID)
	}
	return moved, err
}

// shiftWarmup notes in a warming provider's plan that its share of a send
// moved away during an outage (once per day). The plan moves by counted
// days (PR-8): if too little of today's cap goes out, the daily job repeats
// today's step by itself, so the plan isn't shifted here as well.
func (e *Engine) shiftWarmup(ctx context.Context, q *store.Queries, tx pgx.Tx, ws uuid.UUID, p store.Provider, now time.Time) error {
	plan, err := q.WarmupActiveForProvider(ctx, p.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil || plan.Status == "done" {
		return err
	}
	// The plan's log keeps the entry's messages beside its English, so it
	// reads in each reader's language.
	title, body, texts := warmup.ShiftedEvent(p.Name)
	var shiftedToday bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM warmup_events WHERE plan_id = $1 AND kind = 'settings' AND title = $3 AND occurred_at > $2)`,
		plan.ID, now.Add(-24*time.Hour), title).Scan(&shiftedToday); err != nil || shiftedToday {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO warmup_events (id, plan_id, occurred_at, kind, title, body, texts) VALUES ($1, $2, $3, 'settings', $4, $5, $6)`,
		newID(), plan.ID, now, title, body, texts)
	return err
}

// RetryRoute keeps a held share on its provider and tests it (outages.go):
// one test message every minute, to a seed mailbox when the workspace has
// one; the share resumes by itself after three in a row go through, and a
// person is asked again if it still fails after 15 minutes.
func (e *Engine) RetryRoute(ctx context.Context, campaignID, by, provider uuid.UUID) error {
	now := e.now()
	err := db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
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
		rt, err := routeOf(ctx, q, campaignID, provider)
		if err != nil {
			return err
		}
		if rt == nil || rt.Status != "held" {
			return stateErr("state.shareNotHeld", nil)
		}
		cur := parseRouteDetail(rt.StatusDetail)
		until := now.Add(retryFor)
		seed := e.seedInbox(ctx, c.WorkspaceID) != ""
		d := routeDetail{Decision: "retry", Reason: cur.Reason, Until: &until, NextProbe: &now, Seed: seed}
		if err := q.SendingSetRouteStatus(ctx, store.SendingSetRouteStatusParams{CampaignID: campaignID, ProviderID: provider, Status: "held", StatusDetail: rawRouteDetail(d)}); err != nil {
			return err
		}
		cid := campaignID
		// Messages waiting for a retry time can be the test message now.
		if err := q.SendingRetryNow(ctx, store.SendingRetryNowParams{CampaignID: &cid, ProviderID: &provider}); err != nil {
			return err
		}
		ref, name := e.actorRef(ctx, q, by)
		if _, err := tx.Exec(ctx, `UPDATE campaign_incidents SET resolved_at = $3, resolution = 'retried', resolved_by = $4
			WHERE campaign_id = $1 AND provider_id = $2 AND kind = 'provider_outage' AND resolved_at IS NULL`, campaignID, provider, now, ref); err != nil {
			return err
		}
		// The loop sends the test messages; a person's pause stays.
		if err := e.unpause(ctx, q, c, now); err != nil {
			return err
		}
		if !records(ctx) {
			return nil
		}
		pname, short := "", "the provider"
		if p, err := q.ProvGetAny(ctx, provider); err == nil {
			pname, short = p.Name, state.Short(p.Name, p.Adapter)
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.share_retried", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Actor: e.actorFor(ctx, ref, name), After: map[string]any{"provider": firstNonEmpty(pname, provider.String()), "until": until, "seed_inbox": seed},
			Metadata: map[string]any{"summary": english("audit.shareRetried", i18n.Args{"who": short, "seed": yesNo(seed)})}})
	})
	if err == nil {
		e.resetCalls(routeKey{campaignID, provider})
		e.kick(ctx, campaignID)
	}
	return err
}

// PickWinner ends an A/B test with a person's choice.
func (e *Engine) PickWinner(ctx context.Context, campaignID, by uuid.UUID, variant string) error {
	cc, err := e.Q.SendingCampaignContext(ctx, campaignID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	snd, err := e.Q.SendingGetSend(ctx, campaignID)
	if err != nil {
		return stateErr("state.noSend", campaignArgs(cc.Name, cc.Status))
	}
	ab := parseAB(snd.Ab)
	if ab.Phase != "testing" {
		return stateErr("state.abDecided", campaignArgs(cc.Name, cc.Status))
	}
	known := false
	for _, v := range ab.Variants {
		known = known || v == variant
	}
	if !known {
		args := campaignArgs(cc.Name, cc.Status)
		args["variant"] = variant
		return stateErr("state.noVariant", args)
	}
	ref, name := e.actorRef(ctx, e.Q, by)
	ts, err := e.tallies(ctx, campaignID)
	if err != nil {
		return err
	}
	_, rates := abWinner(ab, ts)
	return e.decideAB(ctx, cc, variant, rates, ref, name, e.now())
}
