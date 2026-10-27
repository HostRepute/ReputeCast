package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/sending/rules"
	"github.com/hostrepute/reputecast/internal/store"
)

// Mailbox provider response rules (review GS-12; Traffic rules › Mailbox
// provider responses). A rule counts a mailbox provider's replies that
// contain a text (Yahoo's TSS04, Outlook's S3150) within a window: the
// deferrals and bounces the sending providers report, and the answers an
// SMTP server gives the send loop. At its count it acts on that mailbox
// provider, before any rate threshold: slows it to half rate, pauses it
// (on every route, or on the route that got the replies, with an alert),
// or only alerts, for a while or until someone resumes. A rule acts once
// at a time; once its action ends only new replies count.

// repliesEvery is how often the rules are checked.
const repliesEvery = 10 * time.Second

// rulesEvery is how often the send loop reads the rules again.
const rulesEvery = 30 * time.Second

// replyHit is an answer the send loop got for a message (an SMTP
// server's reply), kept while a rule could count it.
type replyHit struct {
	at      time.Time
	mailbox string
	route   uuid.UUID
	text    string
}

// replyAct is what a rule did in a send (campaign_sends.guard).
type replyAct struct {
	At       time.Time  `json:"at"`
	Until    *time.Time `json:"until,omitempty"`
	Ended    bool       `json:"ended,omitempty"`
	Mailbox  string     `json:"mailbox"`
	Action   string     `json:"action"`
	Route    *uuid.UUID `json:"route,omitempty"`
	Incident string     `json:"incident,omitempty"`
	// Was and To are the mailbox provider's throttle before and after a
	// slowdown (0: none).
	Was float64 `json:"was,omitempty"`
	To  float64 `json:"to,omitempty"`
}

// loadReplyRules reads the workspace's enabled rules, at most every
// rulesEvery.
func (e *Engine) loadReplyRules(ctx context.Context, rs *runState, now time.Time) error {
	if !rs.rulesAt.IsZero() && now.Sub(rs.rulesAt) < rulesEvery && !now.Before(rs.rulesAt) {
		return nil
	}
	all, err := rules.ReplyRules(ctx, e.Q, rs.campaign.WorkspaceID)
	if err != nil {
		return err
	}
	rs.replyRules = rs.replyRules[:0]
	for _, r := range all {
		if r.Enabled {
			rs.replyRules = append(rs.replyRules, r)
		}
	}
	rs.rulesAt = now
	return nil
}

// noteReply keeps a reply the send loop got, when a rule could count it.
func (rs *runState) noteReply(now time.Time, mbp string, route uuid.UUID, text string) {
	for _, r := range rs.replyRules {
		if r.Mailbox == mbp && r.Matches(text) {
			rs.replyHits = append(rs.replyHits, replyHit{now, mbp, route, text})
			if len(rs.replyHits) > 20000 {
				rs.replyHits = rs.replyHits[len(rs.replyHits)-10000:]
			}
			return
		}
	}
}

// replyCount is how many of a mailbox provider's replies matched a rule
// since a time, and the route most of them came through.
func (e *Engine) replyCount(ctx context.Context, rs *runState, r rules.ReplyRule, since time.Time) (int64, *uuid.UUID, error) {
	byRoute := map[uuid.UUID]int64{}
	for _, h := range rs.replyHits {
		if h.mailbox == r.Mailbox && !h.at.Before(since) && r.Matches(h.text) {
			byRoute[h.route]++
		}
	}
	cid := rs.campaign.ID
	rows, err := e.Q.SendingReplyMatches(ctx, store.SendingReplyMatchesParams{CampaignID: &cid, MailboxProvider: r.Mailbox, Since: since, Contains: r.Contains})
	if err != nil {
		return 0, nil, err
	}
	for _, row := range rows {
		if row.ProviderID != nil {
			byRoute[*row.ProviderID] += row.N
		}
	}
	var n, most int64
	var route *uuid.UUID
	for pid, c := range byRoute {
		n += c
		if c > most {
			p := pid
			route, most = &p, c
		}
	}
	return n, route, nil
}

// checkReplies ends the rules' actions whose time is over, then acts on
// the rules whose count is reached. It returns true when the whole send
// stopped.
func (e *Engine) checkReplies(ctx context.Context, rs *runState, now time.Time) error {
	if len(rs.replyRules) == 0 && len(rs.guard.Replies) == 0 {
		return nil
	}
	if !rs.repliedAt.IsZero() && now.Sub(rs.repliedAt) < repliesEvery && !now.Before(rs.repliedAt) {
		return nil
	}
	rs.repliedAt = now
	changed := false
	for _, id := range sortedKeys(rs.guard.Replies) {
		a := rs.guard.Replies[id]
		if a.Ended || a.Until == nil || now.Before(*a.Until) {
			continue
		}
		if err := e.endReply(ctx, rs, id, a, now); err != nil {
			return err
		}
		a.Ended = true
		rs.guard.Replies[id] = a
		changed = true
	}
	counts := map[string]int64{}
	for _, r := range rs.replyRules {
		key := r.ID.String()
		a, acted := rs.guard.Replies[key]
		if acted && !a.Ended {
			continue
		}
		since := now.Add(-r.Window())
		if acted && a.At.After(since) {
			since = a.At
		}
		if acted && a.Until != nil && a.Until.After(since) {
			since = *a.Until
		}
		n, route, err := e.replyCount(ctx, rs, r, since)
		if err != nil {
			return err
		}
		counts[key] = n
		if n < int64(max(r.Count, 1)) {
			continue
		}
		if h, on := rs.holds[r.Mailbox]; on && h.whole() && (r.Action == "pause" || r.Action == "pause_route") {
			continue // already paused
		}
		if err := e.replyAct(ctx, rs, r, n, route, now); err != nil {
			return err
		}
		changed = true
	}
	if !mapsEqual(rs.guard.ReplyCounts, counts) {
		rs.guard.ReplyCounts, changed = counts, true
	}
	if changed {
		return saveGuard(ctx, e.Q, rs.campaign.ID, rs.guard, now)
	}
	return nil
}

func mapsEqual(a, b map[string]int64) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// replyAct does what a rule says.
func (e *Engine) replyAct(ctx context.Context, rs *runState, r rules.ReplyRule, n int64, route *uuid.UUID, now time.Time) error {
	c := rs.campaign
	act := replyAct{At: now, Mailbox: r.Mailbox, Action: r.Action}
	if d := r.Duration(); d > 0 {
		until := now.Add(d)
		act.Until = &until
	}
	var held int64
	for k, cell := range rs.queue {
		if k.mailbox == r.Mailbox && (r.Action != "pause_route" || route == nil || k.provider == *route) {
			held += cell.queued + cell.held
		}
	}
	routeName := ""
	if r.Action == "pause_route" && route != nil {
		act.Route = route
		routeName = rs.routes[*route].shortName()
	}
	words := i18n.Args{"name": quoted(c.Name), "mailbox": r.Mailbox, "contains": r.Contains, "count": n, "windowMin": int64(r.WindowMinutes),
		"action": r.Action, "route": routeName, "timed": yesNo(act.Until != nil), "held": held}
	// The rule's time, as "2 h" or "30 min".
	d := r.Duration()
	words["hours"], words["minutes"], words["unit"] = int64(d/time.Hour), int64(d/time.Minute), "min"
	if d >= time.Hour && d%time.Hour == 0 {
		words["unit"] = "h"
	}
	detail := IncidentDetail{Events: n, Note: "", Action: map[string]string{"slow": "slowed", "alert": "alerted"}[r.Action]}
	detail.addCause(keep("cause.replyLead", words), keep("cause.reply", words))
	if r.Note != "" {
		detail.addRecommendation(keep("rec.replyNote", i18n.Args{"note": r.Note}))
	} else if k := rules.KnownReplyKey(r.Contains); k != "" {
		detail.addRecommendation(keep("reply.means."+k, nil))
	}
	switch r.Action {
	case "slow":
		act.Was = rs.throttles[r.Mailbox]
		act.To = rs.slowRate(r.Mailbox)
		words["slowRate"] = act.To
		th := map[string]float64{}
		for k, x := range rs.throttles {
			th[k] = x
		}
		th[r.Mailbox] = act.To
		b, _ := json.Marshal(th)
		if err := e.Q.SendingSetThrottles(ctx, store.SendingSetThrottlesParams{CampaignID: c.ID, Throttles: b, At: now}); err != nil {
			return err
		}
		rs.throttles = th
		detail.SlowedTo = act.To
		detail.addRecommendation(keep("rec.replySlowed", words))
		detail.setOptions([]option{{key: "acknowledge", recommended: true, title: keep("opt.acknowledgeTitle", nil), body: keep("opt.acknowledgeNotice", i18n.Args{"action": "slowed"}),
			action: keep("opt.acknowledge", nil)}})
	case "alert":
		detail.setOptions([]option{{key: "acknowledge", recommended: true, title: keep("opt.acknowledgeTitle", nil), body: keep("opt.acknowledgeNotice", i18n.Args{"action": "alerted"}),
			action: keep("opt.acknowledge", nil)}})
	default: // pause, pause_route
		sc := i18n.Args{"scope": "mailbox_provider", "mailbox": r.Mailbox, "who": ""}
		opts := []option{{key: "resume", recommended: false, title: keep("opt.resumeTitle", i18n.Args{"recommended": "no"}),
			body: keep("opt.replyResume", words), action: keep("opt.resumeAction", nil)}}
		if act.Until != nil {
			opts = append(opts, option{key: "wait", recommended: true, title: keep("opt.replyWaitTitle", words), body: keep("opt.replyWait", words), action: keep("opt.replyWaitAction", nil)})
		}
		opts = append(opts, option{key: "cancel_share", title: keep("opt.cancelShare", sc), body: keep("opt.cancelShareBody", with(sc, "held", held)), action: keep("opt.cancelShare", sc)})
		if act.Until == nil {
			opts[0].recommended = true
		}
		detail.setOptions(opts)
	}
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		id := newID()
		act.Incident = id.String()
		value := float64(n)
		limit := float64(r.Count)
		if err := q.SendingInsertIncident(ctx, store.SendingInsertIncidentParams{ID: id, CampaignID: c.ID, Kind: "reply_rule", Scope: "mailbox_provider", ProviderID: act.Route,
			MailboxProvider: r.Mailbox, Metric: "reply", Value: &value, Threshold: &limit, Held: map[bool]int64{true: held}[r.Action == "pause" || r.Action == "pause_route"],
			Detail: detail.stored(), TrippedAt: now}); err != nil {
			return err
		}
		if r.Action == "pause" || r.Action == "pause_route" {
			s, err := q.SendingGetSendForUpdate(ctx, c.ID)
			if err != nil {
				return err
			}
			holds := parseHolds(s.Holds)
			holds[r.Mailbox] = hold{Incident: id.String(), Since: now, Until: act.Until, Route: act.Route, Rule: r.ID.String()}
			b, _ := json.Marshal(holds)
			if err := q.SendingSetHolds(ctx, store.SendingSetHoldsParams{CampaignID: c.ID, Holds: b, At: now}); err != nil {
				return err
			}
			rs.holds = holds
		}
		if r.Action != "slow" {
			if err := notify.ToMembersWith(ctx, tx, c.WorkspaceID, authz.CampaignsSend, notify.Notification{
				Kind: notify.KindSendPaused, Severity: map[bool]string{true: "bad", false: "warn"}[r.Action != "alert"],
				TitleText: keep("notify.replyTitle", words), BodyText: keep("notify.replyBody", words),
				Link: liveLink(c.WorkspaceSlug, c.ID), Data: map[string]any{"campaignId": c.ID.String(), "incidentId": id.String(), "rule": r.ID.String()},
			}); err != nil {
				return err
			}
		}
		if err := q.TrafficReplyRuleMatched(ctx, store.TrafficReplyRuleMatchedParams{ID: r.ID, At: &now, CampaignID: &c.ID}); err != nil {
			return err
		}
		if rs.guard.Replies == nil {
			rs.guard.Replies = map[string]replyAct{}
		}
		rs.guard.Replies[r.ID.String()] = act
		if err := saveGuard(ctx, q, c.ID, rs.guard, now); err != nil {
			return err
		}
		summary := fmt.Sprintf("%s replied %q %d times in %d min · %s", mailboxName(r.Mailbox), r.Contains, n, r.WindowMinutes, replyActionText(r, routeName, act.To))
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.reply_rule", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Actor: systemActor, Metadata: map[string]any{"rule": r.ID.String(), "mailbox_provider": r.Mailbox, "contains": r.Contains, "count": n, "action": r.Action,
				"until": act.Until, "summary": summary, "source": "guard", "via": "reply rule"}})
	})
}

// replyActionText says what a rule did, in the audit log's English.
func replyActionText(r rules.ReplyRule, route string, to float64) string {
	who := mailboxName(r.Mailbox)
	out := ""
	switch r.Action {
	case "slow":
		out = fmt.Sprintf("slowed %s to %s", who, fmtPerSecond(to))
	case "pause":
		out = "paused " + who
	case "pause_route":
		out = "paused " + who + " on " + firstNonEmpty(route, "its route")
	default:
		return "alerted"
	}
	if r.DurationMinutes > 0 {
		out += " for " + durationText(r.Duration())
	}
	return out
}

// durationText writes a rule's duration ("30 min", "2 h").
func durationText(d time.Duration) string {
	if d >= time.Hour && d%time.Hour == 0 {
		return fmt.Sprintf("%d h", int(d/time.Hour))
	}
	return fmt.Sprintf("%d min", int(d/time.Minute))
}

// endReply ends a rule's action whose time is over: a slowdown gives the
// mailbox provider its rate back (unless a person changed it since), a
// pause releases it; the incident ends as expired.
func (e *Engine) endReply(ctx context.Context, rs *runState, id string, a replyAct, now time.Time) error {
	c := rs.campaign
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		s, err := q.SendingGetSendForUpdate(ctx, c.ID)
		if err != nil {
			return err
		}
		switch a.Action {
		case "slow":
			th := map[string]float64{}
			_ = json.Unmarshal(s.Throttles, &th)
			if th[a.Mailbox] == a.To {
				if a.Was > 0 {
					th[a.Mailbox] = a.Was
				} else {
					delete(th, a.Mailbox)
				}
				b, _ := json.Marshal(th)
				if err := q.SendingSetThrottles(ctx, store.SendingSetThrottlesParams{CampaignID: c.ID, Throttles: b, At: now}); err != nil {
					return err
				}
				rs.throttles = th
			}
		case "pause", "pause_route":
			holds := parseHolds(s.Holds)
			if h, ok := holds[a.Mailbox]; ok && h.Rule == id && !h.Resumed {
				delete(holds, a.Mailbox)
				b, _ := json.Marshal(holds)
				if err := q.SendingSetHolds(ctx, store.SendingSetHoldsParams{CampaignID: c.ID, Holds: b, At: now}); err != nil {
					return err
				}
				rs.holds = holds
			}
		}
		if a.Incident != "" {
			if iid, err := uuid.Parse(a.Incident); err == nil {
				if err := q.SendingResolveIncident(ctx, store.SendingResolveIncidentParams{ID: iid, At: &now, Resolution: "expired"}); err != nil {
					return err
				}
			}
		}
		if a.Action == "pause" || a.Action == "pause_route" {
			// A send that waited only for this mailbox provider goes on.
			cur, err := q.SendingCampaignForUpdate(ctx, c.ID)
			if err != nil {
				return err
			}
			if err := e.unpause(ctx, q, cur, now); err != nil {
				return err
			}
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.reply_rule_ended", TargetType: "campaign", TargetID: c.ID.String(),
			TargetName: c.Name, Actor: systemActor, Metadata: map[string]any{"rule": id, "mailbox_provider": a.Mailbox, "action": a.Action,
				"summary": fmt.Sprintf("%s: the reply rule's %s ended", mailboxName(a.Mailbox), strings.ReplaceAll(a.Action, "_", " ")), "source": "guard"}})
	})
}

// fmtPerSecond writes a rate as the monitor does ("2 /s", "2.5 /s").
func fmtPerSecond(v float64) string {
	return strings.TrimSuffix(strings.TrimSuffix(fmt.Sprintf("%.1f", v), "0"), ".") + " /s"
}
