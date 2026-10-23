package texting

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/applang"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/channels"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// The dispatcher sends queued texts when their time comes, sender by
// sender at each sender's throughput, inside each recipient's quiet-hours
// window. A provider that's down holds its texts, or fails them over to
// the route's fallback; held texts stay valid until the window closes in
// each recipient's time zone, then they're cancelled as "not sent: provider
// down". Every text is claimed before it's sent, so none goes twice.

// systemActor is ReputeCast acting on its own.
var systemActor = &audit.Actor{Type: authz.ActorSystem, Name: "System"}

// maxAttempts is how often a temporary failure is retried.
const maxAttempts = 3

// kick asks the dispatcher to run now.
func (s *Service) kick(ctx context.Context) {
	if c := s.jobs(); c != nil {
		if _, err := c.Insert(ctx, DispatchArgs{}, nil); err != nil {
			s.log().WarnContext(ctx, "sms dispatch kick", "err", err)
		}
	}
}

// Dispatch sends due texts for up to budget. again is when to look next;
// done is true when nothing is queued at all.
func (s *Service) Dispatch(ctx context.Context, budget time.Duration) (again time.Duration, done bool, err error) {
	start := time.Now()
	for time.Since(start) < budget {
		n, err := s.dispatchRound(ctx)
		if err != nil {
			return 0, false, err
		}
		if n > 0 {
			continue
		}
		next, err := s.nextDue(ctx)
		if err != nil {
			return 0, false, err
		}
		if next == nil {
			return 0, true, nil
		}
		wait := time.Until(*next)
		if s.Now != nil {
			wait = next.Sub(s.now())
		}
		if wait > 5*time.Second {
			return min(wait, 30*time.Second), false, nil
		}
		select {
		case <-ctx.Done():
			return 0, false, ctx.Err()
		case <-time.After(max(wait, 200*time.Millisecond)):
		}
	}
	return 10 * time.Millisecond, false, nil
}

// nextDue is when the next queued text is due; nil when none is queued.
func (s *Service) nextDue(ctx context.Context) (*time.Time, error) {
	var next *time.Time
	var n int64
	err := s.Pool.QueryRow(ctx, `SELECT min(coalesce(send_after, created_at)), count(*) FROM channel_messages WHERE status = 'queued' AND channel = 'sms'`).Scan(&next, &n)
	if err != nil || n == 0 {
		return nil, err
	}
	return next, nil
}

// dispatchRound does one pass: scheduled sends start, expired texts are
// rescheduled or cancelled, and each sender sends what's due for about two
// seconds. It returns how many texts it handed to providers.
func (s *Service) dispatchRound(ctx context.Context) (int, error) {
	now := s.now()
	if err := s.startScheduled(ctx, now); err != nil {
		return 0, err
	}
	if err := s.expire(ctx, now); err != nil {
		return 0, err
	}
	senders, err := s.Q.ChanSendersDue(ctx, &now)
	if err != nil {
		return 0, err
	}
	if len(senders) == 0 {
		return 0, s.finishSends(ctx, now)
	}
	type job struct {
		ws     *store.Workspace
		g      *graph
		sender store.SmsSendersRow
		spent  *budgetState
	}
	graphs := map[uuid.UUID]*graph{}
	budgets := map[uuid.UUID]*budgetState{}
	wss := map[uuid.UUID]*store.Workspace{}
	jobs := []job{}
	for _, sid := range senders {
		if sid == nil {
			continue
		}
		var wsID uuid.UUID
		if err := s.Pool.QueryRow(ctx, `SELECT workspace_id FROM sms_senders WHERE id = $1`, *sid).Scan(&wsID); err != nil {
			continue
		}
		ws, ok := wss[wsID]
		if !ok {
			w, err := s.Q.GetWorkspace(ctx, wsID)
			if err != nil {
				continue
			}
			ws = &w
			wss[wsID] = ws
		}
		g, ok := graphs[wsID]
		if !ok {
			if g, err = s.graph(ctx, s.Q, ws); err != nil {
				return 0, err
			}
			graphs[wsID] = g
			spent, err := s.spentThisMonth(ctx, ws)
			if err != nil {
				return 0, err
			}
			budgets[wsID] = &budgetState{budget: g.rules.Budget, spent: spent}
		}
		sn, ok := g.senders[*sid]
		if !ok {
			// The sender was removed: its texts can't go.
			if _, err := s.Pool.Exec(ctx, `UPDATE channel_messages SET status = 'cancelled', reason = 'sender_removed', updated_at = $2
				WHERE sender_id = $1 AND status = 'queued'`, *sid, now); err != nil {
				return 0, err
			}
			continue
		}
		jobs = append(jobs, job{ws: ws, g: g, sender: sn, spent: budgets[wsID]})
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	sent := 0
	sem := make(chan struct{}, 8)
	for _, j := range jobs {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			n := s.sendDue(ctx, j.ws, j.g, j.sender, j.spent)
			mu.Lock()
			sent += n
			mu.Unlock()
		}()
	}
	wg.Wait()
	for wsID, b := range budgets {
		if b.budget > 0 {
			s.budgetAlerts(ctx, wss[wsID], graphs[wsID].rules, b)
		}
	}
	return sent, s.finishSends(ctx, now)
}

// budgetState is a workspace's month spend during a round.
type budgetState struct {
	mu     sync.Mutex
	budget float64
	spent  float64
}

// take reserves a text's cost; false when the budget can't cover it.
func (b *budgetState) take(cost float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.budget > 0 && b.spent+cost > b.budget+1e-9 {
		return false
	}
	b.spent += cost
	return true
}

// sendDue sends a sender's due texts for about two seconds, paced at its
// throughput.
func (s *Service) sendDue(ctx context.Context, ws *store.Workspace, g *graph, sn store.SmsSendersRow, b *budgetState) int {
	rate := sn.Throughput
	if rate <= 0 {
		rate = 1
	}
	lim := int32(math.Max(1, math.Ceil(rate*2)))
	now := s.now()
	due, err := s.Q.ChanMessagesDue(ctx, store.ChanMessagesDueParams{SenderID: &sn.ID, Now: &now, Lim: lim})
	if err != nil {
		s.log().WarnContext(ctx, "sms due texts", "sender", sn.ID, "err", err)
		return 0
	}
	gap := time.Duration(float64(time.Second) / rate)
	sent := 0
	for i, m := range due {
		if ctx.Err() != nil {
			break
		}
		if i > 0 {
			time.Sleep(gap)
		}
		if s.sendQueued(ctx, ws, g, m, b) {
			sent++
		}
	}
	return sent
}

// sendQueued sends one queued text: quiet hours, the budget and the
// provider's health are checked first.
func (s *Service) sendQueued(ctx context.Context, ws *store.Workspace, g *graph, m store.ChannelMessage, b *budgetState) bool {
	now := s.now()
	w := g.rules.Window
	loc := location(m.TimeZone)
	if m.Kind != "test" && m.Kind != "reply" && !w.Inside(now, loc) {
		next := w.NextOpen(now, loc)
		closes := w.Closes(next, loc)
		_, _ = s.Pool.Exec(ctx, `UPDATE channel_messages SET send_after = $2, expires_at = $3, updated_at = $4 WHERE id = $1 AND status = 'queued'`, m.ID, next, closes, now)
		return false
	}
	if (m.Kind == "campaign" || m.Kind == "automation") && !b.take(m.Cost) {
		s.budgetReached(ctx, ws, m)
		return false
	}
	if m.ProviderID == nil {
		return false
	}
	prov, ok := g.providers[*m.ProviderID]
	if !ok {
		_, _ = s.Pool.Exec(ctx, `UPDATE channel_messages SET status = 'cancelled', reason = 'provider_removed', updated_at = $2 WHERE id = $1`, m.ID, now)
		return false
	}
	if g.down(prov.ID) {
		d := g.decide(m.Country)
		if d.Fallback != nil && !g.down(d.Fallback.ProviderID) {
			fb := d.Fallback
			price := 0.0
			if p, ok := g.prices[fb.ProviderID][m.Country]; ok {
				price = p.Price
			}
			if _, err := s.Pool.Exec(ctx, `UPDATE channel_messages SET provider_id = $2, sender_id = $3, sender = $4, price = $5, cost = $5 * units, updated_at = $6
				WHERE id = $1 AND status = 'queued'`, m.ID, fb.ProviderID, fb.ID, fb.Value, price, now); err != nil {
				return false
			}
			m.ProviderID, m.SenderID, m.Sender = &fb.ProviderID, &fb.ID, fb.Value
			prov = g.providers[fb.ProviderID]
		} else {
			_, _ = s.Pool.Exec(ctx, `UPDATE channel_messages SET status = 'held', reason = 'provider_down', updated_at = $2 WHERE id = $1 AND status = 'queued'`, m.ID, now)
			return false
		}
	}
	claimed, err := s.Q.ChanMessageClaim(ctx, store.ChanMessageClaimParams{At: now, ID: m.ID})
	if err != nil {
		return false
	}
	if claimed.Kind == "campaign" && claimed.CampaignID != nil {
		_ = s.Q.SmsSendTouch(ctx, store.SmsSendTouchParams{At: &now, CampaignID: *claimed.CampaignID})
	}
	s.deliver(ctx, ws, prov, claimed)
	return true
}

// deliver hands a claimed text to its provider and records the answer: a
// temporary failure goes back in the queue, a failure that says the number
// can't get texts suppresses it.
func (s *Service) deliver(ctx context.Context, ws *store.Workspace, prov store.SmsProvider, m store.ChannelMessage) {
	now := s.now()
	body := ""
	if m.Body != nil {
		body = *m.Body
	}
	ad, err := s.open(prov)
	if err == nil {
		var res sms.SendResult
		cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		res, err = ad.Send(cctx, sms.Message{ID: m.ID, To: m.Recipient, From: m.Sender, Body: body, Encoding: sms.Encoding(m.Encoding), StatusURL: s.hookURL(prov)})
		cancel()
		if err == nil {
			_ = s.Q.ChanMessageSent(ctx, store.ChanMessageSentParams{ProviderMessageID: res.ProviderMessageID, At: &now, Units: int32(res.Segments), ID: m.ID})
			s.event(ctx, ws.ID, m.ID, "sent", "", nil, false, now)
			s.recordCall(ctx, prov, nil)
			if prov.Demo {
				s.demoReceipt(prov, res.ProviderMessageID, m.Recipient, m.Sender)
			}
			return
		}
	}
	s.recordCall(ctx, prov, err)
	var se *sms.SendError
	switch {
	case errors.As(err, &se) && (se.Temporary || se.Throttled) && m.Attempts < maxAttempts:
		next := now.Add(time.Duration(30*max(m.Attempts, 1)) * time.Second)
		_ = s.Q.ChanMessageRequeue(ctx, store.ChanMessageRequeueParams{SendAfter: &next, Reason: string(se.Reason), ProviderCode: se.Code, At: now, ID: m.ID})
		return
	case errors.Is(err, sms.ErrUnavailable) || errors.Is(err, sms.ErrCredentialsRejected):
		// Nothing is lost: the text waits for the provider, inside its window.
		if m.Kind == "test" || m.Kind == "reply" {
			break
		}
		_, _ = s.Pool.Exec(ctx, `UPDATE channel_messages SET status = 'held', reason = 'provider_down', updated_at = $2 WHERE id = $1`, m.ID, now)
		return
	}
	status, reason, code := "failed", string(sms.ReasonOther), ""
	if se != nil {
		reason, code = string(se.Reason), se.Code
		if se.Reason.Suppresses() {
			s.suppressFromFailure(ctx, ws, prov, m.Recipient, se.Reason, codeLabel(prov, code), m.CampaignID, m.AutomationID, contextOf(m.Kind))
		}
	} else if errors.Is(err, sms.ErrUnavailable) || errors.Is(err, sms.ErrCredentialsRejected) {
		reason = "provider_down"
	}
	_ = s.Q.ChanMessageFail(ctx, store.ChanMessageFailParams{Status: status, Reason: reason, ProviderCode: code, At: &now, Cost: 0, ID: m.ID})
	s.event(ctx, ws.ID, m.ID, "failed", code, map[string]any{"reason": reason}, false, now)
	if m.Kind == "confirmation" && m.ContactID != nil {
		s.confirmationFailed(ctx, ws, m, reason)
	}
	_ = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		_, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "sms.failed", s.hookData(m, map[string]any{"reason": reason, "provider_code": code}))
		return err
	})
	if m.Kind == "campaign" && m.CampaignID != nil {
		s.checkAutoPause(ctx, ws, *m.CampaignID)
	}
}

// startScheduled starts the sends whose time has come.
func (s *Service) startScheduled(ctx context.Context, now time.Time) error {
	rows, err := s.Pool.Query(ctx, `SELECT s.campaign_id FROM sms_sends s WHERE s.phase = 'scheduled' AND s.scheduled_at <= $1`, now)
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return err
	}
	for _, id := range ids {
		var wsID uuid.UUID
		var name string
		if err := s.Pool.QueryRow(ctx, `SELECT workspace_id, name FROM campaigns WHERE id = $1`, id).Scan(&wsID, &name); err != nil {
			continue
		}
		ws, err := s.Q.GetWorkspace(ctx, wsID)
		if err != nil {
			continue
		}
		err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
			tag, err := tx.Exec(ctx, `UPDATE sms_sends SET phase = 'sending' WHERE campaign_id = $1 AND phase = 'scheduled'`, id)
			if err != nil || tag.RowsAffected() == 0 {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE campaigns SET status = 'sending', started_at = coalesce(started_at, $2), updated_at = $2 WHERE id = $1`, id, now); err != nil {
				return err
			}
			if _, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "campaign.started", map[string]any{"campaign_id": id.String(), "name": name, "channel": "sms"}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.send_started", TargetType: "campaign", TargetID: id.String(), TargetName: name,
				Actor: systemActor, Before: map[string]any{"status": "scheduled"}, After: map[string]any{"status": "sending"}, Metadata: map[string]any{"channel": "sms"}})
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// expire moves texts whose window closed before they went: queued ones to
// their next window, ones held for a provider outage are cancelled, ones
// held for a pending sender are cancelled after their seven days.
func (s *Service) expire(ctx context.Context, now time.Time) error {
	if _, err := s.Pool.Exec(ctx, `UPDATE channel_messages SET status = 'cancelled', updated_at = $1
		WHERE status = 'held' AND reason IN ('provider_down','pending_sender') AND expires_at IS NOT NULL AND expires_at < $1
		AND kind IN ('campaign','automation')`, now); err != nil {
		return err
	}
	rows, err := s.Pool.Query(ctx, `SELECT m.id, m.time_zone, m.workspace_id FROM channel_messages m
		WHERE m.status = 'queued' AND m.expires_at IS NOT NULL AND m.expires_at < $1 AND m.kind <> 'test' LIMIT 1000`, now)
	if err != nil {
		return err
	}
	type late struct {
		id   uuid.UUID
		zone string
		ws   uuid.UUID
	}
	lates := []late{}
	for rows.Next() {
		var l late
		if err := rows.Scan(&l.id, &l.zone, &l.ws); err != nil {
			rows.Close()
			return err
		}
		lates = append(lates, l)
	}
	rows.Close()
	windows := map[uuid.UUID]channels.Window{}
	for _, l := range lates {
		w, ok := windows[l.ws]
		if !ok {
			ws, err := s.Q.GetWorkspace(ctx, l.ws)
			if err != nil {
				continue
			}
			r, err := s.rules(ctx, s.Q, &ws)
			if err != nil {
				return err
			}
			w = r.Window
			windows[l.ws] = w
		}
		loc := location(l.zone)
		next := w.NextOpen(now, loc)
		if _, err := s.Pool.Exec(ctx, `UPDATE channel_messages SET send_after = $2, expires_at = $3, updated_at = $4 WHERE id = $1 AND status = 'queued'`,
			l.id, next, w.Closes(next, loc), now); err != nil {
			return err
		}
	}
	return nil
}

// finishSends completes the sends with nothing left to send.
func (s *Service) finishSends(ctx context.Context, now time.Time) error {
	rows, err := s.Pool.Query(ctx, `SELECT s.campaign_id FROM sms_sends s WHERE s.phase = 'sending'
		AND NOT EXISTS (SELECT 1 FROM channel_messages m WHERE m.campaign_id = s.campaign_id AND m.status IN ('queued','sending'))`)
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return err
	}
	for _, id := range ids {
		var wsID uuid.UUID
		var name string
		if err := s.Pool.QueryRow(ctx, `SELECT workspace_id, name FROM campaigns WHERE id = $1`, id).Scan(&wsID, &name); err != nil {
			continue
		}
		ws, err := s.Q.GetWorkspace(ctx, wsID)
		if err != nil {
			continue
		}
		var sent, delivered, failed int64
		var cost float64
		_ = s.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE sent_at IS NOT NULL), count(*) FILTER (WHERE status = 'delivered'),
			count(*) FILTER (WHERE status IN ('undelivered','failed')), coalesce(sum(cost), 0)::float8 FROM channel_messages WHERE campaign_id = $1`, id).
			Scan(&sent, &delivered, &failed, &cost)
		err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
			tag, err := tx.Exec(ctx, `UPDATE sms_sends SET phase = 'completed', completed_at = $2 WHERE campaign_id = $1 AND phase = 'sending'`, id, now)
			if err != nil || tag.RowsAffected() == 0 {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE campaigns SET status = 'sent', completed_at = $2, updated_at = $2 WHERE id = $1`, id, now); err != nil {
				return err
			}
			if _, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "campaign.completed", map[string]any{"campaign_id": id.String(), "name": name, "channel": "sms",
				"status": "sent", "sent": sent, "delivered": delivered, "failed": failed, "cost": round2(cost)}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.send_completed", TargetType: "campaign", TargetID: id.String(), TargetName: name,
				Actor: systemActor, Before: map[string]any{"status": "sending"}, After: map[string]any{"status": "sent"},
				Metadata: map[string]any{"channel": "sms", "sent": sent, "delivered": delivered, "failed": failed, "cost": round2(cost)}})
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Auto-pause metrics.
const (
	MetricFailed   = "failed"
	MetricFiltered = "filtered"
	MetricOptOuts  = "opt_outs"
)

// checkAutoPause pauses a send whose failed, filtered or opt-out rate is
// over its threshold, once 50 texts went out. Failed and filtered rates are
// of the texts with a receipt; opt-outs of the texts sent.
func (s *Service) checkAutoPause(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) {
	snd, err := s.Q.SmsSend(ctx, campaignID)
	if err != nil || snd.Phase != "sending" {
		return
	}
	var sent, receipts, failed, filtered, optOuts int64
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE sent_at IS NOT NULL),
			count(*) FILTER (WHERE status IN ('delivered','undelivered','failed') AND sent_at IS NOT NULL),
			count(*) FILTER (WHERE status IN ('undelivered','failed') AND reason <> 'filtered' AND sent_at IS NOT NULL),
			count(*) FILTER (WHERE reason = 'filtered'),
			(SELECT count(*) FROM sms_inbound i WHERE i.campaign_id = $1 AND i.kind = 'opt_out')
		FROM channel_messages WHERE campaign_id = $1 AND kind = 'campaign'`, campaignID).Scan(&sent, &receipts, &failed, &filtered, &optOuts); err != nil {
		return
	}
	if sent < MinSentForPause {
		return
	}
	r, err := s.rules(ctx, s.Q, ws)
	if err != nil {
		return
	}
	th := thresholdsOf(snd.Thresholds, r)
	rate := func(n, d int64) float64 {
		if d == 0 {
			return 0
		}
		return float64(n) * 100 / float64(d)
	}
	metric, value, limit, events, base := "", 0.0, 0.0, int64(0), int64(0)
	switch {
	case rate(filtered, receipts) > th.Filtered:
		metric, value, limit, events, base = MetricFiltered, rate(filtered, receipts), th.Filtered, filtered, receipts
	case rate(failed, receipts) > th.Failed:
		metric, value, limit, events, base = MetricFailed, rate(failed, receipts), th.Failed, failed, receipts
	case rate(optOuts, sent) > th.OptOuts:
		metric, value, limit, events, base = MetricOptOuts, rate(optOuts, sent), th.OptOuts, optOuts, sent
	default:
		return
	}
	c, err := s.campaign(ctx, ws, campaignID)
	if err != nil {
		return
	}
	// The provider's most frequent code, for the report's explanation.
	var code, provName string
	var codeCount int64
	_ = s.Pool.QueryRow(ctx, `SELECT m.provider_code, coalesce(p.name, ''), count(*) FROM channel_messages m LEFT JOIN sms_providers p ON p.id = m.provider_id
		WHERE m.campaign_id = $1 AND m.status IN ('undelivered','failed') AND m.provider_code <> '' AND ($2 = '' OR m.reason = $2)
		GROUP BY 1, 2 ORDER BY 3 DESC LIMIT 1`, campaignID, map[string]string{MetricFiltered: "filtered"}[metric]).Scan(&code, &provName, &codeCount)
	value = math.Round(value*10) / 10
	pause := map[string]any{"metric": metric, "value": value, "limit": limit, "events": events, "base": base, "code": code, "provider": provName,
		"codeCount": codeCount, "sent": sent}
	if err := s.setPhase(ctx, ws, c, snd, "paused", "auto", pause, nil); err != nil {
		s.log().WarnContext(ctx, "sms auto-pause", "campaign", campaignID, "err", err)
		return
	}
	var held int64
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM channel_messages WHERE campaign_id = $1 AND status = 'held'`, campaignID).Scan(&held)
	// The notification and the audit entry keep their lines as messages:
	// each member reads them in their own language, the rates in their
	// region's format.
	args := pauseArgs(metric, value, limit)
	args["name"], args["held"], args["events"], args["base"] = c.Name, held, events, base
	reason, meta := audit.KeepReason(map[string]any{"channel": "sms", "metric": metric, "value": value, "limit": limit, "events": events, "base": base, "held": held},
		mAuditPausedReason.Keep(i18n.Args{"metric": metric}))
	meta = audit.KeepText(meta, "summary", mAuditPausedSummary.Keep(args))
	_ = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if err := notify.ToMembersWith(ctx, tx, ws.ID, authz.CampaignsSend, notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindSMSSendPaused, Severity: "bad",
			TitleText: mNotifyPausedTitle.Keep(i18n.Args{"name": c.Name}), SummaryText: mNotifyPausedSummary.Keep(args), BodyText: mNotifyPausedSummary.Keep(args),
			Link: "/w/" + ws.Slug + "/campaigns/" + c.ID.String() + "/report", Data: map[string]any{"campaignId": c.ID.String(), "metric": metric}}); err != nil {
			return err
		}
		if _, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "campaign.paused", map[string]any{"campaign_id": c.ID.String(), "name": c.Name, "channel": "sms",
			"paused_by": "auto_pause", "metric": metric, "value": value, "limit": limit, "held": held}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.auto_paused", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Reason: reason, Actor: systemActor, Metadata: meta})
	})
}

// pauseArgs are an auto-pause's rate and limit, in percent, for its
// messages: the rate with one decimal (6.2%), the limit as it was set (5%,
// 2.5%), each written by the reader's region format.
func pauseArgs(metric string, value, limit float64) i18n.Args {
	whole := "no"
	if limit == math.Trunc(limit) {
		whole = "yes"
	}
	return i18n.Args{"metric": metric, "value": value / 100, "limit": limit / 100, "limitWhole": whole}
}

// budgetReached holds a text the month's budget can't cover: a campaign's
// send pauses, an automation's text waits until the budget is raised.
func (s *Service) budgetReached(ctx context.Context, ws *store.Workspace, m store.ChannelMessage) {
	now := s.now()
	if m.Kind == "campaign" && m.CampaignID != nil {
		snd, err := s.Q.SmsSend(ctx, *m.CampaignID)
		if err != nil || snd.Phase != "sending" {
			return
		}
		c, err := s.campaign(ctx, ws, *m.CampaignID)
		if err != nil {
			return
		}
		reason, meta := audit.KeepReason(nil, mAuditBudgetReason.Keep(nil))
		_ = s.setPhase(ctx, ws, c, snd, "paused", "budget", map[string]any{"budget": true}, &audit.Event{Action: "campaign.auto_paused", Actor: systemActor,
			Reason: reason, Metadata: meta})
		return
	}
	_, _ = s.Pool.Exec(ctx, `UPDATE channel_messages SET status = 'held', reason = 'budget', updated_at = $2 WHERE id = $1 AND status = 'queued'`, m.ID, now)
}

// budgetAlerts tells the people who manage providers when the month's
// spend reaches the alert share and when it reaches the budget, once each.
func (s *Service) budgetAlerts(ctx context.Context, ws *store.Workspace, r Rules, b *budgetState) {
	spent, err := s.spentThisMonth(ctx, ws)
	if err != nil || r.Budget <= 0 {
		return
	}
	now := s.now()
	month := monthStart(now, location(ws.Timezone))
	level := 0
	switch {
	case spent >= r.Budget-1e-9:
		level = 100
	case spent >= r.Budget*float64(r.BudgetAlertPct)/100:
		level = r.BudgetAlertPct
	default:
		return
	}
	day := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	first, err := s.Q.SmsBudgetAlertInsert(ctx, store.SmsBudgetAlertInsertParams{WorkspaceID: ws.ID, Month: day, Level: int32(level), At: now})
	if err != nil || !first {
		return
	}
	sev, title, summary := "warn", mNotifyBudgetAlertTitle, mNotifyBudgetAlertBody
	if level == 100 {
		sev, title, summary = "bad", mNotifyBudgetUsedTitle, mNotifyBudgetUsedBody
	}
	// The notice keeps its messages, so each member reads it in their own
	// language; a message can't write money, so the amounts are written in
	// each member's region format (the row keeps them in English, as before).
	args := func(l i18n.Lang) i18n.Args {
		return i18n.Args{"level": float64(level) / 100, "month": int(month.Month()), "spent": l.Region.Money(round2(spent), 2), "budget": l.Region.Money(r.Budget, 2)}
	}
	english := args(i18n.Default)
	ids, err := notify.MembersWith(ctx, s.Pool, ws.ID, authz.ProvidersManage)
	if err != nil {
		return
	}
	for _, id := range ids {
		mine := args(applang.ForUserID(ctx, s.Q, id))
		_ = notify.ToUser(ctx, s.Pool, id, notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindSMSBudget, Severity: sev,
			Title: title.Text(i18n.Default, english), Summary: summary.Text(i18n.Default, english), Body: summary.Text(i18n.Default, english),
			TitleText: title.Keep(mine), SummaryText: summary.Keep(mine), BodyText: summary.Keep(mine),
			Link: "/w/" + ws.Slug + "/providers/sms/rules", Data: map[string]any{"level": level, "spent": round2(spent), "budget": r.Budget}})
	}
}

// releaseHeld puts texts held for a reason back in the queue (a provider
// that answers again, a budget raised); they keep quiet hours.
func (s *Service) releaseHeld(ctx context.Context, workspaceID uuid.UUID, providerID *uuid.UUID, reasons ...string) int64 {
	tag, err := s.Pool.Exec(ctx, `UPDATE channel_messages SET status = 'queued', reason = '', updated_at = $4
		WHERE workspace_id = $1 AND status = 'held' AND reason = ANY($2::text[]) AND ($3::uuid IS NULL OR provider_id = $3)`,
		workspaceID, reasons, providerID, s.now())
	if err != nil {
		return 0
	}
	if tag.RowsAffected() > 0 {
		s.kick(ctx)
	}
	return tag.RowsAffected()
}

// releasePending re-routes texts held for a sender that wasn't registered:
// those whose sender can now text their country go back in the queue.
func (s *Service) releasePending(ctx context.Context, ws *store.Workspace) {
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, country, time_zone FROM channel_messages WHERE workspace_id = $1 AND status = 'held'
		AND reason IN ('pending_sender','no_sender','sender_pending','sender_not_registered','sender_rejected','no_route')`, ws.ID)
	if err != nil {
		return
	}
	type held struct {
		id      uuid.UUID
		country string
		zone    string
	}
	hs := []held{}
	for rows.Next() {
		var h held
		if rows.Scan(&h.id, &h.country, &h.zone) == nil {
			hs = append(hs, h)
		}
	}
	rows.Close()
	now := s.now()
	n := 0
	for _, h := range hs {
		d := g.decide(h.country)
		if d.Sender == nil {
			continue
		}
		loc := location(h.zone)
		next := g.rules.Window.NextOpen(now, loc)
		if _, err := s.Pool.Exec(ctx, `UPDATE channel_messages SET status = 'queued', reason = '', provider_id = $2, sender_id = $3, sender = $4,
			price = $5, cost = $5 * units, send_after = $6, expires_at = $7, updated_at = $8 WHERE id = $1 AND status = 'held'`,
			h.id, d.Sender.ProviderID, d.Sender.ID, d.Sender.Value, d.Price, next, g.rules.Window.Closes(next, loc), now); err == nil {
			n++
		}
	}
	if n > 0 {
		s.kick(ctx)
	}
}
