package engine

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/mailbox"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/sending/rules"
	"github.com/hostrepute/reputecast/internal/store"
)

// The thresholds card (LiveMonitor, review GS-6 and GS-14): each rate
// against its limit with the evidence behind it, as the guard judges it
// (a scope a person resumed counts what happened since). A row says
// "collecting evidence" while the rate is at or near its limit but the
// guard can't trip yet: fewer events or less mail than it needs, or not
// enough to be sure the rate is really above the limit.

// guardOf is a send's guard state (none before the send exists).
func guardOf(snd store.CampaignSend, hasSend bool) guardState {
	if !hasSend {
		return guardState{}
	}
	return parseGuard(snd.Guard)
}

// liveThresholds are the thresholds card's rows; api are the API error
// rates of the routes the send still uses.
func liveThresholds(th Thresholds, ts tallies, g guardState, api []LiveThresholdPart, ty typed) []LiveThreshold {
	t := g.judged(rearmCampaign, ts.total)
	out := []LiveThreshold{thresholdRow("bounce", t.bounced(), t.sent, minEvents, minSampleCampaign, th.Bounce)}
	// By bounce type and deferral levels (GS-11): the mailbox provider
	// closest to its limit.
	if row, ok := worstTyped(ts, g, ty, "block_bounce", th); ok {
		out = append(out, row)
	}
	switch {
	case th.DeferralSlowWindow > 0 || th.DeferralWindow > 0:
		if row, ok := worstTyped(ts, g, ty, "deferral", th); ok {
			out = append(out, row)
		}
	default:
		if row, ok := worstMailbox(ts, g, "deferral", th.Deferral); ok {
			out = append(out, row)
		}
	}
	// Complaints count only where mailbox providers report them (F-2).
	complaint := thresholdRow("complaint", t.complained, t.reported, minEvents, minSampleCampaign, th.Complaint)
	if t.reported == 0 && t.delivered > 0 {
		complaint.State, complaint.Reason = "na", mailbox.NotReported
	}
	for _, k := range routing.MailboxProviders {
		if m := ts.mailbox[k]; m != nil && m.reported > 0 {
			complaint.Parts = append(complaint.Parts, LiveThresholdPart{Name: mailboxName(k), Key: k, Value: ptrRate(m.complained, m.reported)})
		}
	}
	out = append(out, complaint)
	if row, ok := worstMailbox(ts, g, "complaint", th.MailboxComplaint); ok {
		out = append(out, row)
	}
	apiRow := LiveThreshold{Key: "api_errors", Limit: th.APIErrors, State: "na", Reason: "no_data", Parts: api}
	for _, p := range api {
		if p.Value != nil && (apiRow.Value == nil || *p.Value > *apiRow.Value) {
			v := *p.Value
			apiRow.Value = &v
		}
	}
	if apiRow.Value != nil {
		apiRow.Reason = ""
		switch v := *apiRow.Value; {
		case v >= th.APIErrors:
			apiRow.State = "collecting"
		case v >= 0.9*th.APIErrors:
			apiRow.State = "near"
		default:
			apiRow.State = "ok"
		}
	}
	return append(out, apiRow)
}

// thresholdRow is a rate's row: its value (null without a base), evidence
// and state.
func thresholdRow(key string, events, base, minEv, minBase int64, limit float64) LiveThreshold {
	r := LiveThreshold{Key: key, Limit: limit, Events: events, Base: base, MinEvents: minEv, MinBase: minBase, AnyVolume: anyVolume(key)}
	if base <= 0 {
		r.State, r.Reason = "na", "no_data"
		return r
	}
	v := round2(rate(events, base))
	r.Value = &v
	r.State = thresholdState(key, events, base, minEv, minBase, limit)
	return r
}

// thresholdState is a rate's state against its limit, as the guard judges
// it (exceeds): collecting evidence while it is at its limit or near it
// but can't trip yet.
func thresholdState(key string, events, base, minEv, minBase int64, limit float64) string {
	v := rate(events, base)
	enough := events >= minEv && base >= minBase
	tripped, _ := exceeds(key, events, base, minEv, minBase, limit)
	switch {
	case tripped:
		return "tripped"
	case v >= limit, v >= 0.9*limit && !enough:
		return "collecting"
	case v >= 0.9*limit:
		return "near"
	}
	return "ok"
}

// worstMailbox is a per-mailbox-provider threshold's row for the mailbox
// provider closest to its limit (complaints: among those that report
// them).
func worstMailbox(ts tallies, g guardState, metric string, limit float64) (LiveThreshold, bool) {
	var best LiveThreshold
	bestRatio := -1.0
	for _, k := range routing.MailboxProviders {
		mt := ts.mailbox[k]
		if mt == nil {
			continue
		}
		m := g.judged(rearmMailbox(k), *mt)
		var row LiveThreshold
		switch metric {
		case "complaint":
			if !mailbox.ReportsComplaints(k) || m.reported == 0 {
				continue
			}
			row = thresholdRow("mailbox_complaint", m.complained, m.reported, minEvents, minSampleScoped, limit)
		default:
			if m.sent == 0 {
				continue
			}
			row = thresholdRow("deferral", m.deferred, m.sent, minDeferrals, minSampleScoped, limit)
		}
		row.Mailbox = k
		ratio := 0.0
		if row.Value != nil && limit > 0 {
			ratio = *row.Value / limit
		}
		if ratio > bestRatio {
			best, bestRatio = row, ratio
		}
	}
	return best, bestRatio >= 0
}

// worstTyped is a typed threshold's row for the mailbox provider closest
// to its limit: block/policy bounces of what was sent, or deferrals
// within the first level's window.
func worstTyped(ts tallies, g guardState, ty typed, metric string, th Thresholds) (LiveThreshold, bool) {
	var best LiveThreshold
	bestRatio := -1.0
	for _, k := range routing.MailboxProviders {
		mt := ts.mailbox[k]
		if mt == nil {
			continue
		}
		key := rearmMailbox(k)
		var row LiveThreshold
		switch metric {
		case "block_bounce":
			m := g.judged(key, *mt)
			if m.sent == 0 || th.BlockBounce <= 0 {
				continue
			}
			row = thresholdRow("block_bounce", max(ty.block[k]-g.Rearm[key].Block, 0), m.sent, minEvents, minSampleScoped, th.BlockBounce)
			row.Action = th.BlockAction
		default:
			limit, window, action := th.DeferralSlow, th.DeferralSlowWindow, th.DeferralSlowAction
			if limit <= 0 || window <= 0 {
				limit, window, action = th.Deferral, th.DeferralWindow, th.DeferralAction
			}
			s := ty.sent[window][k]
			if s == 0 || limit <= 0 {
				continue
			}
			row = thresholdRow("deferral", ty.deferred[window][k], s, minDeferrals, minSampleScoped, limit)
			row.Window, row.Action = window, action
		}
		row.Mailbox = k
		ratio := 0.0
		if row.Value != nil && row.Limit > 0 {
			ratio = *row.Value / row.Limit
		}
		if ratio > bestRatio {
			best, bestRatio = row, ratio
		}
	}
	return best, bestRatio >= 0
}

// replyRow is the thresholds card's row for the reply rules: the rule
// closest to its count in its window (review GS-12), as the send loop
// last counted it.
func (e *Engine) replyRow(ctx context.Context, ws uuid.UUID, g guardState) (LiveThreshold, bool) {
	rs, err := rules.ReplyRules(ctx, e.Q, ws)
	if err != nil {
		return LiveThreshold{}, false
	}
	var on []rules.ReplyRule
	for _, r := range rs {
		if r.Enabled {
			on = append(on, r)
		}
	}
	if len(on) == 0 {
		return LiveThreshold{}, false
	}
	best, bestRatio := on[0], -1.0
	for _, r := range on {
		if ratio := float64(g.ReplyCounts[r.ID.String()]) / float64(max(r.Count, 1)); ratio > bestRatio {
			best, bestRatio = r, ratio
		}
	}
	n := g.ReplyCounts[best.ID.String()]
	row := LiveThreshold{Key: "reply_rules", Limit: float64(best.Count), Events: n, MinEvents: int64(best.Count), MinBase: int64(len(on)), Mailbox: best.Mailbox,
		Window: best.WindowMinutes, Action: best.Action, Duration: best.DurationMinutes, Parts: []LiveThresholdPart{{Name: best.Contains, Key: best.Mailbox}}, State: "ok"}
	v := float64(n)
	row.Value = &v
	switch {
	case n >= int64(best.Count):
		row.State = "tripped"
	case float64(n) >= 0.9*float64(best.Count):
		row.State = "near"
	}
	return row, true
}
