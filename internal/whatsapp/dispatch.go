package whatsapp

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
	"github.com/hostrepute/reputecast/internal/whatsapp/meta"
)

// Sending (brief §3.9): a due campaign is planned into the message log,
// one message per person with consent, after quiet hours, the schedule
// ("21 Oct, 09:00 in each recipient's time zone") and the messaging limit
// (the people over it go the next day, or are skipped); the dispatcher
// sends each to Meta. A token Meta rejects holds every send until it's
// replaced; a busy Meta is retried.

// maxAttempts is how often a message is tried before it fails.
const maxAttempts = 5

// recipient is one person a campaign or step goes to.
type recipient struct {
	ContactID  uuid.UUID
	ListID     uuid.UUID
	Phone      string
	Country    string
	Zone       string
	ZoneSource string
	FirstName  string
	LastName   string
	Email      string
	Attributes map[string]any
}

// value is a contact field for a template variable.
func (r recipient) value(field string) string {
	switch field {
	case "first_name":
		return r.FirstName
	case "last_name":
		return r.LastName
	case "full_name", "name":
		return fullName(r.FirstName, r.LastName)
	case "email":
		return r.Email
	case "phone":
		return phone.Display(r.Phone)
	}
	if v, ok := r.Attributes[field]; ok && v != nil {
		switch x := v.(type) {
		case string:
			return x
		case float64:
			if x == float64(int64(x)) {
				return itoa(int(x))
			}
			b, _ := json.Marshal(x)
			return string(b)
		case bool:
			if x {
				return "yes"
			}
			return "no"
		default:
			b, _ := json.Marshal(x)
			return string(b)
		}
	}
	return ""
}

// WhatsAppVariable maps a template variable to a contact field or a fixed
// text, with a fallback.
type WhatsAppVariable struct {
	Key string `json:"key" maxLength:"4"`
	// Source is "field" or "text".
	Source   string `json:"source" enum:"field,text"`
	Field    string `json:"field,omitempty" maxLength:"64"`
	Text     string `json:"text,omitempty" maxLength:"500"`
	Fallback string `json:"fallback,omitempty" maxLength:"500"`
}

// paramsFor fills a template's variables for one person, in order.
func paramsFor(keys []string, vars []WhatsAppVariable, r recipient) []string {
	byKey := map[string]WhatsAppVariable{}
	for _, v := range vars {
		byKey[v.Key] = v
	}
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		v := byKey[k]
		val := ""
		if v.Source == "text" {
			val = v.Text
		} else if v.Field != "" {
			val = r.value(v.Field)
		}
		if val == "" {
			val = v.Fallback
		}
		if val == "" {
			// Meta refuses an empty parameter.
			val = "-"
		}
		out = append(out, clip(val, 1000))
	}
	return out
}

// zoneOf is a recipient's time zone: their own, else their number's.
func zoneOf(r recipient, ws *store.Workspace) (*time.Location, string) {
	if r.Zone != "" {
		if l, err := time.LoadLocation(r.Zone); err == nil {
			return l, r.Zone
		}
	}
	return wsLocation(ws), ws.Timezone
}

// used is who got a message the business started in the last 24 hours:
// Meta's messaging limit counts them, oldest first.
func (s *Service) used(ctx context.Context, ws *store.Workspace, now time.Time) ([]time.Time, error) {
	rows, err := s.Pool.Query(ctx, `SELECT min(coalesce(m.sent_at, m.send_after, m.created_at)) FROM channel_messages m
		JOIN whatsapp_messages w ON w.message_id = m.id AND w.kind = 'template'
		WHERE m.workspace_id = $1 AND m.channel = 'whatsapp' AND m.status IN ('queued','sending','sent','delivered')
		  AND coalesce(m.sent_at, m.send_after, m.created_at) > $2::timestamptz - interval '24 hours'
		  AND coalesce(m.sent_at, m.send_after, m.created_at) <= $2::timestamptz
		GROUP BY m.recipient ORDER BY 1`, ws.ID, now)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[time.Time])
}

// limitPlan is how a send fits the messaging limit: Now people now, the
// rest Later at the times slots free up (one each).
type limitPlan struct {
	Limit int
	Used  int
	Left  int
	// Frees are when the people counted now leave the rolling 24 hours.
	Frees []time.Time
}

func (s *Service) limitPlan(ctx context.Context, ws *store.Workspace, limit int, now time.Time) (limitPlan, error) {
	lp := limitPlan{Limit: limit}
	if limit <= 0 {
		lp.Left = 1 << 30
		return lp, nil
	}
	used, err := s.used(ctx, ws, now)
	if err != nil {
		return lp, err
	}
	lp.Used = len(used)
	lp.Left = max(limit-lp.Used, 0)
	for _, t := range used {
		lp.Frees = append(lp.Frees, t.Add(24*time.Hour))
	}
	return lp, nil
}

// later is when the i-th person over the limit can go.
func (lp limitPlan) later(i int, now time.Time) time.Time {
	if i < len(lp.Frees) {
		return lp.Frees[i]
	}
	return now.Add(24 * time.Hour)
}

// planned is a campaign's people in the message log.
type planned struct {
	rows []outgoing
	left map[string]int64
}

// plan writes a campaign's people: consent and suppressions were applied
// by the audience; this applies the schedule, quiet hours and the limit.
func (s *Service) plan(ctx context.Context, ws *store.Workspace, c store.WhatsAppCampaignRow, tpl store.WhatsappTemplate, aud []recipient, rl rules,
	limit int, overLimit string, now time.Time) (planned, error) {
	out := planned{left: map[string]int64{}}
	var vars []WhatsAppVariable
	_ = json.Unmarshal(c.Variables, &vars)
	keys := variablesOf(tpl.Body)
	type slot struct {
		r  recipient
		at time.Time
	}
	slots := make([]slot, 0, len(aud))
	for _, r := range aud {
		loc, _ := zoneOf(r, ws)
		at := now
		switch c.ScheduleMode {
		case "at":
			if c.SendAt != nil && c.SendAt.After(now) {
				at = *c.SendAt
			}
		case "local":
			if c.LocalDate != nil && c.LocalMinute != nil {
				d := *c.LocalDate
				t := time.Date(d.Year(), d.Month(), d.Day(), int(*c.LocalMinute)/60, int(*c.LocalMinute)%60, 0, 0, loc)
				if t.After(now) {
					at = t
				}
			}
		}
		if rl.Quiet {
			at = rl.Window.NextOpen(at, loc)
		}
		slots = append(slots, slot{r: r, at: at})
	}
	// The limit counts people in the order they go.
	sort.SliceStable(slots, func(i, j int) bool { return slots[i].at.Before(slots[j].at) })
	lp, err := s.limitPlan(ctx, ws, limit, now)
	if err != nil {
		return out, err
	}
	rc := s.rateCard(ctx, s.Q, ws)
	over := 0
	for i, sl := range slots {
		r := sl.r
		loc, _ := zoneOf(r, ws)
		params := paramsFor(keys, vars, r)
		price := rc.price(r.Country, tpl.Category)
		o := outgoing{id: uuid.Must(uuid.NewV7()), kind: "campaign", campaignID: &c.ID, contactID: &r.ContactID, listID: uuidPtr(r.ListID), recipient: r.Phone,
			country: r.Country, timeZone: r.Zone, zoneSource: r.ZoneSource, body: render(tpl.Body, params), status: "queued", template: tpl.Name,
			language: tpl.Language, category: tpl.Category, params: params, price: &price}
		at := sl.at
		if i >= lp.Left {
			if overLimit == "skip" {
				o.status, o.reason = "skipped", "messaging_limit"
				out.left["messaging_limit"]++
				out.rows = append(out.rows, o)
				continue
			}
			at = lp.later(over, now)
			if rl.Quiet {
				at = rl.Window.NextOpen(at, loc)
			}
			over++
			o.reason = "messaging_limit"
			out.left["next_day"]++
		} else if at.After(now) {
			o.reason = "scheduled"
			if rl.Quiet && !rl.Window.Inside(now, loc) && c.ScheduleMode == "now" {
				o.reason = "quiet_hours"
				out.left["held"]++
			}
		}
		o.after = &at
		out.rows = append(out.rows, o)
	}
	return out, nil
}

// StartDue starts the sends whose time has come.
func (s *Service) StartDue(ctx context.Context) error {
	rows, err := s.Pool.Query(ctx, `SELECT campaign_id FROM whatsapp_sends WHERE phase = 'scheduled' AND scheduled_at <= $1 ORDER BY scheduled_at`, s.now())
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.start(ctx, id); err != nil {
			s.log().WarnContext(ctx, "whatsapp send start", "campaign", id, "err", err)
		}
	}
	return nil
}

// start plans one send into the message log.
func (s *Service) start(ctx context.Context, id uuid.UUID) error {
	var wsID uuid.UUID
	if err := s.Pool.QueryRow(ctx, `SELECT workspace_id FROM whatsapp_sends WHERE campaign_id = $1`, id).Scan(&wsID); err != nil {
		return err
	}
	ws, err := s.Q.GetWorkspace(ctx, wsID)
	if err != nil {
		return err
	}
	c, err := s.Q.WhatsAppCampaign(ctx, store.WhatsAppCampaignParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return err
	}
	snd, err := s.Q.WhatsAppSend(ctx, id)
	if err != nil {
		return err
	}
	a, err := s.account(ctx, s.Q, &ws)
	if err != nil {
		return err
	}
	if c.TemplateID == nil {
		return errors.New("whatsapp: the campaign has no template")
	}
	tpl, err := s.Q.WhatsAppTemplate(ctx, store.WhatsAppTemplateParams{ID: *c.TemplateID, WorkspaceID: ws.ID})
	if err != nil {
		return err
	}
	lists, segments := audienceOf(c.Audience)
	aud, err := s.audience(ctx, &ws, lists, segments)
	if err != nil {
		return err
	}
	now := s.now()
	rl := s.rules(ctx, s.Q, &ws)
	var left map[string]int64
	_ = json.Unmarshal(snd.LeftOut, &left)
	overLimit := c.OverLimit
	if v, ok := left["over_limit_skip"]; ok && v > 0 {
		overLimit = "skip"
	}
	p, err := s.plan(ctx, &ws, c, tpl, aud.list, rl, int(a.MessagingLimit), overLimit, now)
	if err != nil {
		return err
	}
	sending := int64(0)
	for _, o := range p.rows {
		if o.status == "queued" {
			sending++
		}
	}
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE whatsapp_sends SET phase = 'sending', first_send_at = $2, recipients = $3, left_out = $4
			WHERE campaign_id = $1 AND phase = 'scheduled'`, c.ID, now, sending, jsonOf(p.left))
		if err != nil || tag.RowsAffected() == 0 {
			return err
		}
		for _, o := range p.rows {
			if err := insertMessage(ctx, q, &ws, a.ID, o, now); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE campaigns SET status = 'sending', started_at = coalesce(started_at, $2), audience_count = $3, updated_at = $2 WHERE id = $1`,
			c.ID, now, sending); err != nil {
			return err
		}
		if _, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "campaign.started", map[string]any{"campaign_id": c.ID.String(), "name": c.Name, "channel": "whatsapp"}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.send_started", TargetType: "campaign", TargetID: c.ID.String(),
			TargetName: c.Name, Actor: systemActor, Before: map[string]any{"status": "scheduled"}, After: map[string]any{"status": "sending"},
			Metadata: map[string]any{"channel": "whatsapp", "recipients": sending, "left_out": p.left, "template": tpl.Name}})
	})
	if err != nil {
		return err
	}
	if a.Status == "token_rejected" {
		_, _ = s.Pool.Exec(ctx, `UPDATE channel_messages SET status = 'held', reason = 'token_rejected', updated_at = $2
			WHERE campaign_id = $1 AND status = 'queued'`, c.ID, now)
	}
	s.kick(ctx)
	return nil
}

// due is a message picked for sending.
type due struct {
	id        uuid.UUID
	ws        uuid.UUID
	kind      string
	campaign  *uuid.UUID
	contact   *uuid.UUID
	recipient string
	body      string
	attempts  int32
	waKind    string
	template  string
	language  string
	params    []string
	account   *uuid.UUID
}

// pick claims a batch of due messages.
func (s *Service) pick(ctx context.Context, now time.Time, limit int) ([]due, error) {
	out := []due{}
	err := s.tx(ctx, func(_ *store.Queries, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT m.id, m.workspace_id, m.kind, m.campaign_id, m.contact_id, m.recipient, coalesce(m.body, ''), m.attempts,
				w.kind, w.template, w.language, w.params, w.account_id
			FROM channel_messages m JOIN whatsapp_messages w ON w.message_id = m.id
			WHERE m.channel = 'whatsapp' AND m.status = 'queued' AND (m.send_after IS NULL OR m.send_after <= $1)
			ORDER BY m.send_after NULLS FIRST, m.created_at
			LIMIT $2 FOR UPDATE OF m SKIP LOCKED`, now, limit)
		if err != nil {
			return err
		}
		for rows.Next() {
			var d due
			var params []byte
			if err := rows.Scan(&d.id, &d.ws, &d.kind, &d.campaign, &d.contact, &d.recipient, &d.body, &d.attempts, &d.waKind, &d.template, &d.language,
				&params, &d.account); err != nil {
				rows.Close()
				return err
			}
			_ = json.Unmarshal(params, &d.params)
			out = append(out, d)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		ids := make([]uuid.UUID, 0, len(out))
		for _, d := range out {
			ids = append(ids, d.id)
		}
		_, err = tx.Exec(ctx, `UPDATE channel_messages SET status = 'sending', attempts = attempts + 1, updated_at = $2 WHERE id = ANY($1)`, ids, now)
		return err
	})
	return out, err
}

// kick asks the dispatcher to look now.
func (s *Service) kick(ctx context.Context) {
	if c := s.jobs(); c != nil {
		if _, err := c.Insert(ctx, DispatchArgs{}, nil); err != nil {
			s.log().WarnContext(ctx, "whatsapp dispatch kick", "err", err)
		}
	}
}

// kickStart asks the starter to look now.
func (s *Service) kickStart(ctx context.Context) {
	if c := s.jobs(); c != nil {
		if _, err := c.Insert(ctx, StartArgs{}, nil); err != nil {
			s.log().WarnContext(ctx, "whatsapp start kick", "err", err)
		}
	}
}

// sender is a workspace's account, opened for sending.
type sender struct {
	ws      store.Workspace
	account store.WhatsappAccount
	client  *meta.Client
	err     error
	// stop is set once Meta rejected the token: nothing more is sent.
	stop bool
}

// Dispatch sends due messages for up to budget. again is when to look
// next; done is true when nothing is queued at all.
func (s *Service) Dispatch(ctx context.Context, budget time.Duration) (again time.Duration, done bool, err error) {
	start := time.Now()
	senders := map[uuid.UUID]*sender{}
	for time.Since(start) < budget {
		batch, err := s.pick(ctx, s.now(), 100)
		if err != nil {
			return 0, false, err
		}
		if len(batch) == 0 {
			break
		}
		for _, d := range batch {
			snd, ok := senders[d.ws]
			if !ok {
				snd = s.senderFor(ctx, d.ws)
				senders[d.ws] = snd
			}
			s.deliver(ctx, snd, d)
		}
	}
	if err := s.finishSends(ctx); err != nil {
		return 0, false, err
	}
	var next *time.Time
	var n int64
	if err := s.Pool.QueryRow(ctx, `SELECT min(coalesce(send_after, created_at)), count(*) FROM channel_messages WHERE status = 'queued' AND channel = 'whatsapp'`).
		Scan(&next, &n); err != nil {
		return 0, false, err
	}
	if n == 0 || next == nil {
		return 0, true, nil
	}
	return max(time.Until(*next), 0), false, nil
}

func (s *Service) senderFor(ctx context.Context, wsID uuid.UUID) *sender {
	snd := &sender{}
	ws, err := s.Q.GetWorkspace(ctx, wsID)
	if err != nil {
		snd.err = err
		return snd
	}
	snd.ws = ws
	a, err := s.account(ctx, s.Q, &ws)
	if err != nil {
		snd.err = err
		return snd
	}
	snd.account = accountOf(a)
	snd.client, snd.err = s.client(a.ID, a.CredentialsEnc, a.Demo)
	snd.stop = a.Status == "token_rejected"
	return snd
}

// deliver sends one message and records what Meta answered.
func (s *Service) deliver(ctx context.Context, snd *sender, d due) {
	now := s.now()
	fail := func(status, reason, code string) {
		err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
			if err := q.ChanMessageFail(ctx, store.ChanMessageFailParams{Status: status, Reason: reason, ProviderCode: code, At: &now, ID: d.id}); err != nil {
				return err
			}
			kind := "failed"
			if status == "cancelled" || status == "held" {
				return nil
			}
			return q.ChanMessageEventInsert(ctx, store.ChanMessageEventInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: d.ws, MessageID: d.id, Kind: kind,
				ProviderCode: code, Detail: jsonOf(map[string]any{"reason": reason}), OccurredAt: now})
		})
		if err != nil {
			s.log().WarnContext(ctx, "whatsapp failed", "message", d.id, "err", err)
		}
	}
	hold := func(reason string) {
		if _, err := s.Pool.Exec(ctx, `UPDATE channel_messages SET status = 'held', reason = $2, updated_at = $3 WHERE id = $1`, d.id, reason, now); err != nil {
			s.log().WarnContext(ctx, "whatsapp hold", "message", d.id, "err", err)
		}
	}
	switch {
	case errors.Is(snd.err, errNoAccount):
		fail("cancelled", "disconnected", "")
		return
	case snd.err != nil:
		hold("account_error")
		s.log().ErrorContext(ctx, "whatsapp account", "workspace", d.ws, "err", snd.err)
		return
	case snd.stop:
		hold("token_rejected")
		return
	}
	// A suppressed number gets nothing, whatever queued it.
	if _, err := s.Q.ChanSuppressionByPhone(ctx, store.ChanSuppressionByPhoneParams{WorkspaceID: d.ws, Channel: "whatsapp", Phone: d.recipient}); err == nil {
		fail("skipped", "suppressed", "")
		return
	}
	msg := meta.Message{To: d.recipient}
	if d.waKind == "template" {
		msg.Template, msg.Language, msg.Params = d.template, d.language, d.params
	} else {
		msg.Text = d.body
	}
	sent, err := snd.client.Send(ctx, msg)
	if err != nil {
		var me *meta.Error
		if !errors.As(err, &me) {
			me = &meta.Error{Message: err.Error()}
		}
		switch {
		case me.TokenRejected():
			hold("token_rejected")
			snd.stop = true
			if terr := s.tokenRejected(ctx, &snd.ws, snd.account, me); terr != nil {
				s.log().WarnContext(ctx, "whatsapp token rejected", "err", terr)
			}
			snd.account.Status = "token_rejected"
		case me.RateLimited() || me.Unavailable():
			if int(d.attempts)+1 >= maxAttempts {
				fail("failed", "unavailable", itoa(me.Code))
				return
			}
			wait := time.Duration(1<<min(d.attempts, 6)) * 30 * time.Second
			if _, err := s.Pool.Exec(ctx, `UPDATE channel_messages SET status = 'queued', send_after = $2, reason = 'retry', provider_code = $3, updated_at = $4
				WHERE id = $1`, d.id, now.Add(wait), itoa(me.Code), now); err != nil {
				s.log().WarnContext(ctx, "whatsapp retry", "message", d.id, "err", err)
			}
		default:
			fail("failed", meta.ReasonOf(me.Code), itoa(me.Code))
			s.emitFailed(ctx, &snd.ws, d, meta.ReasonOf(me.Code), me.Code)
		}
		return
	}
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.ChanMessageSent(ctx, store.ChanMessageSentParams{ProviderMessageID: sent.ID, At: &now, ID: d.id}); err != nil {
			return err
		}
		return q.ChanMessageEventInsert(ctx, store.ChanMessageEventInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: d.ws, MessageID: d.id, Kind: "sent",
			Detail: jsonOf(map[string]any{"wamid": sent.ID}), OccurredAt: now})
	})
	if err != nil {
		s.log().WarnContext(ctx, "whatsapp sent", "message", d.id, "err", err)
	}
	if snd.account.Demo {
		s.demoReceipts(snd.ws, sent.ID, d.recipient, d.waKind == "template")
	}
}

// emitFailed sends the whatsapp.failed event for a message Meta refused.
func (s *Service) emitFailed(ctx context.Context, ws *store.Workspace, d due, reason string, code int) {
	data := map[string]any{"message_id": d.id.String(), "phone": d.recipient, "kind": d.kind, "reason": reason, "meta_code": code}
	if d.campaign != nil {
		data["campaign_id"] = d.campaign.String()
	}
	if d.contact != nil {
		data["contact_id"] = d.contact.String()
	}
	if _, err := webhooks.Emit(ctx, s.Pool, ws.ID, ws.Slug, "whatsapp.failed", data); err != nil {
		s.log().WarnContext(ctx, "whatsapp failed event", "err", err)
	}
}

// demoReceipts plays Meta's part for the demo's account: a message is
// delivered a moment after it's sent, and most people read it.
func (s *Service) demoReceipts(ws store.Workspace, wamid, to string, template bool) {
	at := s.now()
	time.AfterFunc(time.Duration(1000+rand.IntN(2000))*time.Millisecond, func() { //nolint:gosec // G404: the demo's simulated phones, not a secret
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		a, err := s.Q.WhatsAppAccount(ctx, ws.ID)
		if err != nil {
			return
		}
		cat := ""
		if template {
			cat = "utility"
		}
		evs := []meta.Event{{Kind: meta.EventStatus, MessageID: wamid, Status: "delivered", Recipient: to, At: at.Add(2 * time.Second), Billable: template, Category: cat}}
		if rand.IntN(100) < 70 { //nolint:gosec // G404: as above
			evs = append(evs, meta.Event{Kind: meta.EventStatus, MessageID: wamid, Status: "read", Recipient: to, At: at.Add(time.Duration(30+rand.IntN(600)) * time.Second)}) //nolint:gosec // G404: as above
		}
		for _, e := range evs {
			if err := s.applyStatus(ctx, &ws, accountOf(a), nil, e); err != nil {
				s.log().WarnContext(ctx, "whatsapp demo receipt", "err", err)
			}
		}
	})
}

// finishSends completes the sends with nothing left to send.
func (s *Service) finishSends(ctx context.Context) error {
	rows, err := s.Pool.Query(ctx, `SELECT s.campaign_id FROM whatsapp_sends s WHERE s.phase = 'sending'
		AND NOT EXISTS (SELECT 1 FROM channel_messages m WHERE m.campaign_id = s.campaign_id AND m.status IN ('queued','sending','held'))`)
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return err
	}
	now := s.now()
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
		var sent, failed int64
		_ = s.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE sent_at IS NOT NULL), count(*) FILTER (WHERE status IN ('failed','undelivered'))
			FROM channel_messages WHERE campaign_id = $1 AND kind = 'campaign'`, id).Scan(&sent, &failed)
		err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
			tag, err := tx.Exec(ctx, `UPDATE whatsapp_sends SET phase = 'completed', completed_at = $2, last_send_at = $2 WHERE campaign_id = $1 AND phase = 'sending'`, id, now)
			if err != nil || tag.RowsAffected() == 0 {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE campaigns SET status = 'sent', completed_at = $2, updated_at = $2 WHERE id = $1`, id, now); err != nil {
				return err
			}
			if _, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "campaign.completed", map[string]any{"campaign_id": id.String(), "name": name, "channel": "whatsapp",
				"sent": sent, "failed": failed}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.completed", TargetType: "campaign", TargetID: id.String(), TargetName: name,
				Actor: systemActor, Before: map[string]any{"status": "sending"}, After: map[string]any{"status": "sent"},
				Metadata: map[string]any{"channel": "whatsapp", "sent": sent, "failed": failed}})
		})
		if err != nil {
			return err
		}
	}
	return nil
}
