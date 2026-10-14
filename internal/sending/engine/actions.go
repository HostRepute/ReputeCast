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
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
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
}

// ResumeResult says what resuming changed.
type ResumeResult struct {
	Excluded   int64   `json:"excluded"`
	Rate       float64 `json:"rate,omitempty"`
	Automation string  `json:"automation,omitempty"`
}

// ResumeScope resumes a held mailbox provider or route, as is or with the
// recommended changes.
func (e *Engine) ResumeScope(ctx context.Context, campaignID, by uuid.UUID, in ResumeRequest) (ResumeResult, error) {
	if in.Mailbox == "" && in.ProviderID == nil {
		return ResumeResult{}, e.Resume(ctx, campaignID, by)
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
			return stateErr("%q isn't sending (it is %s).", c.Name, c.Status)
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
				return stateErr("%s isn't on hold.", mailboxName(in.Mailbox))
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
			if in.WithChanges {
				ch := in.Changes
				if ch == nil && incident != nil {
					var d IncidentDetail
					_ = json.Unmarshal(incident.Detail, &d)
					ch = d.Changes
				}
				if ch != nil {
					resolution = "resumed_with_changes"
					if err := e.applyChanges(ctx, q, tx, c, snd, in.Mailbox, ch, &out, now); err != nil {
						return err
					}
				}
			}
			b, _ := json.Marshal(holds)
			if err := q.SendingSetHolds(ctx, store.SendingSetHoldsParams{CampaignID: campaignID, Holds: b, At: now}); err != nil {
				return err
			}
		} else {
			routes, err := q.SendingRoutes(ctx, campaignID)
			if err != nil {
				return err
			}
			found := false
			for _, r := range routes {
				if r.ProviderID == *in.ProviderID && (r.Status == "held" || r.Status == "paused") {
					found = true
				}
			}
			if !found {
				return stateErr("That provider's share isn't on hold.")
			}
			if err := q.SendingSetRouteStatus(ctx, store.SendingSetRouteStatusParams{CampaignID: campaignID, ProviderID: *in.ProviderID, Status: "sending", StatusDetail: `{"decision":"resume"}`}); err != nil {
				return err
			}
			e.resetCalls(routeKey{campaignID, *in.ProviderID})
		}
		if c.Status == "paused" {
			if err := q.SendingSetCampaignStatus(ctx, store.SendingSetCampaignStatusParams{ID: c.ID, Status: "sending", At: now}); err != nil {
				return err
			}
		}
		if incident != nil {
			if err := q.SendingResolveIncident(ctx, store.SendingResolveIncidentParams{ID: incident.ID, At: &now, Resolution: resolution, ResolvedBy: ref}); err != nil {
				return err
			}
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

// applyChanges takes a group out of the rest of the send, lowers a mailbox
// provider's rate and hands the excluded contacts to a flow.
func (e *Engine) applyChanges(ctx context.Context, q *store.Queries, tx pgx.Tx, c store.Campaign, snd store.CampaignSend, mailbox string, ch *ResumeChanges, out *ResumeResult, now time.Time) error {
	cid := c.ID
	var contacts []uuid.UUID
	if ch.Exclude > 0 {
		why := map[string]string{groupImported: "never-engaged import", groupLapsed: "no open in more than 90 days"}[ch.group()]
		if why == "" {
			why = "never engaged"
		}
		share := "the campaign"
		if mailbox != "" {
			share = "the " + mailboxName(mailbox) + " share"
		}
		reason := "Excluded when " + share + " resumed: " + why
		ids, err := q.SendingExcludeGroup(ctx, store.SendingExcludeGroupParams{Reason: reason, CampaignID: &cid, MailboxProvider: mailbox, Grp: ch.group(), Source: ch.Source})
		if err != nil {
			return err
		}
		out.Excluded = int64(len(ids))
		contacts = derefIDs(ids)
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
			return stateErr("%q isn't sending (it is %s).", c.Name, c.Status)
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
		if c.Status == "paused" {
			if err := q.SendingSetCampaignStatus(ctx, store.SendingSetCampaignStatusParams{ID: c.ID, Status: "sending", At: now}); err != nil {
				return err
			}
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
// person's decision during an outage).
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
			return stateErr("%q isn't sending (it is %s).", c.Name, c.Status)
		}
		target, err := q.ProvGet(ctx, store.ProvGetParams{WorkspaceID: c.WorkspaceID, ID: to})
		if err != nil {
			return stateErr("That provider isn't connected to this workspace.")
		}
		if to == from || !state.Available(target) {
			return stateErr("%s can't take mail right now.", target.Name)
		}
		if !state.Caps(target).BulkAllowed() {
			return stateErr("%s's terms don't allow bulk marketing.", target.Name)
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
		if c.Status == "paused" {
			if err := q.SendingSetCampaignStatus(ctx, store.SendingSetCampaignStatusParams{ID: c.ID, Status: "sending", At: now}); err != nil {
				return err
			}
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

// shiftWarmup moves a warming provider's plan by a day when its share of
// a send was moved away during an outage, so the IP doesn't miss its ramp
// (once per day).
func (e *Engine) shiftWarmup(ctx context.Context, q *store.Queries, tx pgx.Tx, ws uuid.UUID, p store.Provider, now time.Time) error {
	plan, err := q.WarmupActiveForProvider(ctx, p.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil || plan.Status == "done" {
		return err
	}
	var shiftedToday bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM warmup_events WHERE plan_id = $1 AND kind = 'settings' AND title = 'Schedule shifted by a day.' AND occurred_at > $2)`,
		plan.ID, now.Add(-24*time.Hour)).Scan(&shiftedToday); err != nil || shiftedToday {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE warmup_plans SET start_date = start_date + 1, updated_at = $2 WHERE id = $1`, plan.ID, now); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO warmup_events (id, plan_id, occurred_at, kind, title, body) VALUES ($1, $2, $3, 'settings', 'Schedule shifted by a day.', $4)`,
		newID(), plan.ID, now, fmt.Sprintf("%s was down during a send and its share moved to another provider, so the warm-up repeats today's step tomorrow.", p.Name))
	return err
}

// RetryRoute keeps a held share on its provider and retries every minute;
// if it is still down after 15 minutes, a person is asked again.
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
			return stateErr("%q isn't sending (it is %s).", c.Name, c.Status)
		}
		routes, err := q.SendingRoutes(ctx, campaignID)
		if err != nil {
			return err
		}
		held := false
		for _, r := range routes {
			held = held || (r.ProviderID == provider && r.Status == "held")
		}
		if !held {
			return stateErr("That provider's share isn't on hold.")
		}
		until := now.Add(retryFor)
		d, _ := json.Marshal(routeDetail{Decision: "retry", Until: &until})
		if err := q.SendingSetRouteStatus(ctx, store.SendingSetRouteStatusParams{CampaignID: campaignID, ProviderID: provider, Status: "sending", StatusDetail: string(d)}); err != nil {
			return err
		}
		cid := campaignID
		if err := q.SendingRetryNow(ctx, store.SendingRetryNowParams{CampaignID: &cid, ProviderID: &provider}); err != nil {
			return err
		}
		ref, name := e.actorRef(ctx, q, by)
		if _, err := tx.Exec(ctx, `UPDATE campaign_incidents SET resolved_at = $3, resolution = 'retried', resolved_by = $4
			WHERE campaign_id = $1 AND provider_id = $2 AND resolved_at IS NULL`, campaignID, provider, now, ref); err != nil {
			return err
		}
		if c.Status == "paused" {
			if err := q.SendingSetCampaignStatus(ctx, store.SendingSetCampaignStatusParams{ID: c.ID, Status: "sending", At: now}); err != nil {
				return err
			}
		}
		pname := ""
		if p, err := q.ProvGetAny(ctx, provider); err == nil {
			pname = p.Name
		}
		if err := emit(ctx, tx, c.WorkspaceID, "campaign.resumed", campaignData(c.ID, c.Name, "resumed_by", firstNonEmpty(name, "System"), "scope", "route",
			"provider_id", provider.String(), "provider", pname, "decision", "retry")); err != nil {
			return err
		}
		if !records(ctx) {
			return nil
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.share_retried", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Actor: e.actorFor(ctx, ref, name), After: map[string]any{"provider": provider.String(), "until": until},
			Metadata: map[string]any{"summary": "Retrying " + firstNonEmpty(pname, "the provider") + " every minute for 15 minutes"}})
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
		return stateErr("%q has no send running.", cc.Name)
	}
	ab := parseAB(snd.Ab)
	if ab.Phase != "testing" {
		return stateErr("The A/B test of %q is already decided.", cc.Name)
	}
	known := false
	for _, v := range ab.Variants {
		known = known || v == variant
	}
	if !known {
		return stateErr("%q has no variant %s.", cc.Name, variant)
	}
	ref, name := e.actorRef(ctx, e.Q, by)
	ts, err := e.tallies(ctx, campaignID)
	if err != nil {
		return err
	}
	_, rates := abWinner(ab, ts)
	return e.decideAB(ctx, cc, variant, rates, ref, name, e.now())
}
