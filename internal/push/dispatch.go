package push

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/push/webpush"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// Sending (brief §3.8): a due campaign is planned into the message log, one
// notification per subscription, after the frequency cap and quiet hours;
// the dispatcher hands each to its push service over the Web Push protocol.
// A subscription the push service reports gone (404, 410) is removed; a busy
// service is retried with its Retry-After.

var systemActor = &audit.Actor{Type: authz.ActorSystem, Name: "System"}

// maxAttempts is how often a notification is tried before it fails.
const maxAttempts = 5

// reasonWorkerMissing holds a notification while its site's service worker
// is missing (board PushStates: "Notifications to acme.io's 1,318
// subscribers are held … until the file is back").
const reasonWorkerMissing = "worker_missing"

// holdSite holds a site's queued notifications: its worker went missing.
func holdSite(ctx context.Context, tx pgx.Tx, siteID uuid.UUID, now time.Time) error {
	_, err := tx.Exec(ctx, `UPDATE channel_messages m SET status = 'held', reason = $2, updated_at = $3
		FROM push_messages p WHERE p.message_id = m.id AND p.site_id = $1 AND m.status = 'queued' AND m.kind <> 'test'`, siteID, reasonWorkerMissing, now)
	return err
}

// releaseSite puts a site's held notifications back in the queue: its
// worker is back. Those waiting for quiet hours keep their time.
func releaseSite(ctx context.Context, tx pgx.Tx, siteID uuid.UUID, now time.Time) (int64, error) {
	tag, err := tx.Exec(ctx, `UPDATE channel_messages m SET status = 'queued',
			reason = CASE WHEN m.send_after > $3 THEN 'quiet_hours' ELSE '' END, updated_at = $3
		FROM push_messages p WHERE p.message_id = m.id AND p.site_id = $1 AND m.status = 'held' AND m.reason = $2`, siteID, reasonWorkerMissing, now)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// expireHeld skips the held notifications whose time to live ran out
// before their site's worker came back.
func (s *Service) expireHeld(ctx context.Context, now time.Time) error {
	_, err := s.Pool.Exec(ctx, `UPDATE channel_messages SET status = 'skipped', updated_at = $2
		WHERE channel = 'push' AND status = 'held' AND reason = $1
		  AND created_at + make_interval(mins => coalesce((body::jsonb ->> 'ttl')::int, 240)) < $2`, reasonWorkerMissing, now)
	return err
}

// Content is a notification's content, kept with the message until it
// goes.
type Content struct {
	Title      string       `json:"title"`
	Message    string       `json:"message"`
	Link       string       `json:"link"`
	Image      string       `json:"image,omitempty"`
	Icon       string       `json:"icon,omitempty"`
	Buttons    []PushButton `json:"buttons,omitempty"`
	Tag        string       `json:"tag,omitempty"`
	Replace    bool         `json:"replace,omitempty"`
	TTLMinutes int          `json:"ttl"`
	// UTM is the utm_campaign its links carry.
	UTM string `json:"utm,omitempty"`
}

// Payload is what the service worker receives (static/sw.js reads it).
type Payload struct {
	V        int             `json:"v"`
	ID       string          `json:"id"`
	Token    string          `json:"t"`
	Title    string          `json:"title"`
	Body     string          `json:"body"`
	Icon     string          `json:"icon,omitempty"`
	Image    string          `json:"image,omitempty"`
	URL      string          `json:"url"`
	Tag      string          `json:"tag,omitempty"`
	Renotify bool            `json:"renotify,omitempty"`
	Actions  []PayloadAction `json:"actions,omitempty"`
	// Events is where the worker reports shown and closed.
	Events string `json:"events"`
	// Expires is when the notification is stale (the TTL from its send).
	Expires int64 `json:"expires"`
}

// PayloadAction is a button.
type PayloadAction struct {
	Action string `json:"action"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

// clickURL is a notification's click link: it records the click and goes
// on to the target with UTM parameters.
func (s *Service) clickURL(id uuid.UUID, tok string, action int) string {
	u := s.origin() + "/t/p/" + id.String() + "." + tok
	if action > 0 {
		u += "/" + string(rune('0'+action))
	}
	return u
}

// payload builds what the service worker receives.
func (s *Service) payload(ws *store.Workspace, id uuid.UUID, tok string, c Content, sentAt time.Time) ([]byte, error) {
	p := Payload{V: WorkerVersion, ID: id.String(), Token: tok, Title: c.Title, Body: c.Message, Icon: c.Icon, Image: c.Image,
		URL: s.clickURL(id, tok, 0), Events: s.origin() + "/p/" + ws.Slug + "/push/events", Expires: sentAt.Add(time.Duration(c.TTLMinutes) * time.Minute).Unix()}
	if c.Tag != "" {
		p.Tag, p.Renotify = c.Tag, c.Replace
	}
	for i, b := range c.Buttons {
		if i >= 2 {
			break
		}
		p.Actions = append(p.Actions, PayloadAction{Action: string(rune('1' + i)), Title: b.Label, URL: s.clickURL(id, tok, i+1)})
	}
	b, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	if len(b) > webpush.MaxPayload {
		return nil, webpush.ErrTooLarge
	}
	return b, nil
}

// campaignContent is a campaign's notification.
func (s *Service) campaignContent(ctx context.Context, ws *store.Workspace, c store.PushCampaignRow) Content {
	out := Content{Title: c.Title, Message: c.Message, Link: c.Link, Tag: c.Tag, Replace: c.ReplaceTag, TTLMinutes: int(c.TtlMinutes), UTM: utmCampaign(c.Name)}
	_ = json.Unmarshal(c.Buttons, &out.Buttons)
	if c.ImageMediaID != nil {
		if im := s.image(ctx, *c.ImageMediaID); im != nil {
			out.Image = im.URL
		}
	}
	return out
}

// queued is one notification to write to the message log.
type queued struct {
	id       uuid.UUID
	r        recipient
	status   string
	reason   string
	after    *time.Time
	contact  *uuid.UUID
	timeZone string
	token    string
}

// insertMessages writes notifications with COPY: the message log rows,
// then push's own.
func insertMessages(ctx context.Context, tx pgx.Tx, ws *store.Workspace, kind string, campaignID, automationID *uuid.UUID, step string, runID *uuid.UUID,
	body []byte, rows []queued, now time.Time) error {
	if len(rows) == 0 {
		return nil
	}
	msgs := make([][]any, 0, len(rows))
	push := make([][]any, 0, len(rows))
	for _, q := range rows {
		msgs = append(msgs, []any{q.id, ws.ID, "push", kind, campaignID, automationID, step, runID, q.contact, q.r.ID.String(), q.timeZone, string(body), 1, "",
			q.status, q.reason, q.after, now, now})
		push = append(push, []any{q.id, ws.ID, q.r.ID, q.r.SiteID, q.r.Browser, q.token})
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"channel_messages"}, []string{"id", "workspace_id", "channel", "kind", "campaign_id", "automation_id",
		"automation_step", "run_id", "contact_id", "recipient", "time_zone", "body", "units", "encoding", "status", "reason", "send_after", "created_at", "updated_at"},
		pgx.CopyFromRows(msgs)); err != nil {
		return err
	}
	_, err := tx.CopyFrom(ctx, pgx.Identifier{"push_messages"}, []string{"message_id", "workspace_id", "subscription_id", "site_id", "browser", "token"},
		pgx.CopyFromRows(push))
	return err
}

// plan applies the frequency cap and quiet hours to an audience at now.
func (s *Service) plan(ctx context.Context, ws *store.Workspace, g Guards, list []recipient, outside string, now time.Time) ([]queued, map[string]int64, error) {
	counts, err := s.todayCounts(ctx, ws, now)
	if err != nil {
		return nil, nil, err
	}
	left := map[string]int64{}
	out := make([]queued, 0, len(list))
	for _, r := range list {
		q := queued{id: uuid.Must(uuid.NewV7()), r: r, status: "queued", contact: r.Contact, timeZone: r.TimeZone, token: token()}
		loc, _ := zoneOf(r, ws)
		inside := !g.Quiet || g.Window.Inside(now, loc)
		switch {
		case counts[r.ID] >= g.Cap:
			q.status, q.reason = "skipped", "frequency_cap"
			left["frequency_cap"]++
		case !inside && outside == "skip":
			q.status, q.reason = "skipped", "quiet_hours"
			left["quiet_hours"]++
		default:
			at := now
			if !inside {
				at, q.reason = g.Window.NextOpen(now, loc), "quiet_hours"
			}
			q.after = &at
			switch {
			case r.WorkerMissing:
				// The site's service worker is missing: held until it's back
				// (or until the notification expires).
				q.status, q.reason = "held", reasonWorkerMissing
				left[reasonWorkerMissing]++
			case !inside:
				left["held"]++
			}
		}
		out = append(out, q)
	}
	return out, left, nil
}

// StartDue starts the sends whose time has come.
func (s *Service) StartDue(ctx context.Context) error {
	if err := s.expireHeld(ctx, s.now()); err != nil {
		return err
	}
	due, err := s.Q.PushSendsDue(ctx, s.now())
	if err != nil {
		return err
	}
	for _, snd := range due {
		if err := s.start(ctx, snd); err != nil {
			s.log().WarnContext(ctx, "push send start", "campaign", snd.CampaignID, "err", err)
		}
	}
	return nil
}

// start plans a send into the message log.
func (s *Service) start(ctx context.Context, snd store.PushSend) error {
	ws, err := s.Q.GetWorkspace(ctx, snd.WorkspaceID)
	if err != nil {
		return err
	}
	c, err := s.campaign(ctx, &ws, snd.CampaignID)
	if err != nil {
		return err
	}
	g, err := s.guards(ctx, s.Q, &ws)
	if err != nil {
		return err
	}
	aud, err := s.audience(ctx, &ws, c.SiteIds, c.Lists, c.Segments)
	if err != nil {
		return err
	}
	now := s.now()
	rows, left, err := s.plan(ctx, &ws, g, aud.list, c.Outside, now)
	if err != nil {
		return err
	}
	content := s.campaignContent(ctx, &ws, c)
	if sites, err := s.sites(ctx, &ws); err == nil {
		for _, st := range sites {
			for _, id := range c.SiteIds {
				if id.String() == st.ID && content.Icon == "" {
					content.Icon = st.IconURL
				}
			}
		}
	}
	body, _ := json.Marshal(content)
	sending := int64(0)
	for _, r := range rows {
		if r.status == "queued" {
			sending++
		}
	}
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE push_sends SET phase = 'sending', first_send_at = $2, recipients = $3, left_out = $4
			WHERE campaign_id = $1 AND phase = 'scheduled'`, c.ID, now, sending, jsonOf(left))
		if err != nil || tag.RowsAffected() == 0 {
			return err
		}
		if err := insertMessages(ctx, tx, &ws, "campaign", &c.ID, nil, "", nil, body, rows, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE campaigns SET status = 'sending', started_at = coalesce(started_at, $2), audience_count = $3, updated_at = $2 WHERE id = $1`,
			c.ID, now, sending); err != nil {
			return err
		}
		if _, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "campaign.started", map[string]any{"campaign_id": c.ID.String(), "name": c.Name, "channel": "push"}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.send_started", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Actor: systemActor, Before: map[string]any{"status": "scheduled"}, After: map[string]any{"status": "sending"},
			Metadata: map[string]any{"channel": "push", "recipients": sending, "left_out": left}})
	})
	if err != nil {
		return err
	}
	s.kick(ctx)
	return nil
}

func jsonOf(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// due is a notification picked for sending.
type due struct {
	id         uuid.UUID
	ws         uuid.UUID
	kind       string
	campaign   *uuid.UUID
	body       []byte
	attempts   int32
	token      string
	sub        uuid.UUID
	endpoint   string
	p256dh     []byte
	auth       []byte
	demo       bool
	removed    bool
	keyID      uuid.UUID
	contact    *uuid.UUID
	automation *uuid.UUID
}

// pick claims a batch of due notifications.
func (s *Service) pick(ctx context.Context, now time.Time, limit int) ([]due, error) {
	out := []due{}
	err := s.tx(ctx, func(_ *store.Queries, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT m.id, m.workspace_id, m.kind, m.campaign_id, coalesce(m.body, ''), m.attempts, p.token, p.subscription_id,
				coalesce(s.endpoint, t.endpoint, ''), coalesce(s.p256dh, t.p256dh), coalesce(s.auth, t.auth),
				coalesce(s.demo, (SELECT count(*) > 0 AND count(*) = count(*) FILTER (WHERE x.demo) FROM push_sites x
					WHERE x.workspace_id = m.workspace_id AND x.removed_at IS NULL)),
				(s.id IS NOT NULL AND s.removed_at IS NOT NULL), coalesce(s.key_id, t.key_id), m.contact_id, m.automation_id
			FROM channel_messages m
			JOIN push_messages p ON p.message_id = m.id
			LEFT JOIN push_subscriptions s ON s.id = p.subscription_id
			LEFT JOIN push_test_subscriptions t ON t.id = p.subscription_id
			WHERE m.channel = 'push' AND m.status = 'queued' AND (m.send_after IS NULL OR m.send_after <= $1)
			ORDER BY m.send_after NULLS FIRST, m.created_at
			LIMIT $2 FOR UPDATE OF m SKIP LOCKED`, now, limit)
		if err != nil {
			return err
		}
		for rows.Next() {
			var d due
			var body string
			var key *uuid.UUID
			if err := rows.Scan(&d.id, &d.ws, &d.kind, &d.campaign, &body, &d.attempts, &d.token, &d.sub, &d.endpoint, &d.p256dh, &d.auth, &d.demo,
				&d.removed, &key, &d.contact, &d.automation); err != nil {
				rows.Close()
				return err
			}
			d.body = []byte(body)
			if key != nil {
				d.keyID = *key
			}
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
			s.log().WarnContext(ctx, "push dispatch kick", "err", err)
		}
	}
}

// kickStart asks the starter to look now.
func (s *Service) kickStart(ctx context.Context) {
	if c := s.jobs(); c != nil {
		if _, err := c.Insert(ctx, StartArgs{}, nil); err != nil {
			s.log().WarnContext(ctx, "push start kick", "err", err)
		}
	}
}

// senderFor opens a workspace's key as a sender.
func (s *Service) senderFor(ctx context.Context, wsID, keyID uuid.UUID, demo bool, cache map[string]*webpush.Sender) (*webpush.Sender, error) {
	ck := keyID.String()
	if demo {
		ck += ":demo"
	}
	if snd, ok := cache[ck]; ok {
		return snd, nil
	}
	k, err := s.Q.PushKeyByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if k.WorkspaceID != wsID {
		return nil, errors.New("push: the key belongs to another workspace")
	}
	vk, err := s.vapid(k)
	if err != nil {
		return nil, err
	}
	client := s.HTTP
	switch {
	case demo && s.Demo != nil:
		// The demo's subscriptions go to the in-process push service.
		client = s.Demo().Client()
	case client == nil:
		client = &http.Client{Timeout: 15 * time.Second}
	}
	snd := &webpush.Sender{Key: vk, Contact: k.Contact, HTTP: client, Now: s.now}
	cache[ck] = snd
	return snd, nil
}

// Dispatch sends due notifications for up to budget. again is when to look
// next; done is true when nothing is queued at all.
func (s *Service) Dispatch(ctx context.Context, budget time.Duration) (again time.Duration, done bool, err error) {
	start := time.Now()
	senders := map[string]*webpush.Sender{}
	workspaces := map[uuid.UUID]*store.Workspace{}
	for time.Since(start) < budget {
		batch, err := s.pick(ctx, s.now(), 200)
		if err != nil {
			return 0, false, err
		}
		if len(batch) == 0 {
			break
		}
		var wg sync.WaitGroup
		sem := make(chan struct{}, 16)
		var mu sync.Mutex
		for _, d := range batch {
			ws, ok := workspaces[d.ws]
			if !ok {
				w, err := s.Q.GetWorkspace(ctx, d.ws)
				if err != nil {
					continue
				}
				ws = &w
				workspaces[d.ws] = ws
			}
			mu.Lock()
			snd, serr := s.senderFor(ctx, d.ws, d.keyID, d.demo, senders)
			mu.Unlock()
			wg.Add(1)
			sem <- struct{}{}
			go func(d due, ws *store.Workspace, snd *webpush.Sender, serr error) {
				defer wg.Done()
				defer func() { <-sem }()
				s.deliver(ctx, ws, d, snd, serr)
			}(d, ws, snd, serr)
		}
		wg.Wait()
	}
	if err := s.finishSends(ctx); err != nil {
		return 0, false, err
	}
	var next *time.Time
	var n int64
	if err := s.Pool.QueryRow(ctx, `SELECT min(coalesce(send_after, created_at)), count(*) FROM channel_messages WHERE status = 'queued' AND channel = 'push'`).
		Scan(&next, &n); err != nil {
		return 0, false, err
	}
	if n == 0 || next == nil {
		return 0, true, nil
	}
	return max(time.Until(*next), 0), false, nil
}

// deliver sends one notification and records what the push service did.
func (s *Service) deliver(ctx context.Context, ws *store.Workspace, d due, snd *webpush.Sender, serr error) {
	now := s.now()
	fail := func(reason, answer string, removeSub string) {
		err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `UPDATE channel_messages SET status = 'failed', reason = $2, failed_at = $3, updated_at = $3 WHERE id = $1`, d.id, reason, now); err != nil {
				return err
			}
			if err := q.PushMessageAnswer(ctx, store.PushMessageAnswerParams{MessageID: d.id, Answer: answer}); err != nil {
				return err
			}
			kind := "failed"
			if reason == "gone" {
				kind = "gone"
			}
			if err := q.ChanMessageEventInsert(ctx, store.ChanMessageEventInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, MessageID: d.id, Kind: kind,
				ProviderCode: answer, Detail: json.RawMessage(`{}`), OccurredAt: now}); err != nil {
				return err
			}
			if removeSub != "" && d.kind != "test" {
				return q.PushSubscriptionRemove(ctx, store.PushSubscriptionRemoveParams{ID: d.sub, At: &now, Reason: removeSub})
			}
			return nil
		})
		if err != nil {
			s.log().WarnContext(ctx, "push failed", "message", d.id, "err", err)
		}
	}
	if d.removed || d.endpoint == "" {
		fail("gone", "subscription removed", "")
		return
	}
	if serr != nil {
		fail("keys", "the keys couldn't be opened", "")
		s.log().ErrorContext(ctx, "push key", "workspace", ws.ID, "err", serr)
		return
	}
	var c Content
	if err := json.Unmarshal(d.body, &c); err != nil {
		fail("content", "no content", "")
		return
	}
	if c.Link != "" {
		c.Link = withUTM(c.Link, c.UTM)
	}
	payload, err := s.payload(ws, d.id, d.token, c, now)
	if err != nil {
		fail("too_large", "payload over 4 KB", "")
		return
	}
	sub := webpush.Subscription{Endpoint: d.endpoint, P256dh: d.p256dh, Auth: d.auth}
	msg := webpush.Message{Payload: payload, TTL: time.Duration(max(c.TTLMinutes, 1)) * time.Minute, Urgency: webpush.UrgencyNormal}
	if c.Tag != "" && c.Replace {
		msg.Topic = webpush.TopicFor(c.Tag)
	}
	if c.TTLMinutes <= 60 {
		msg.Urgency = webpush.UrgencyHigh
	}
	res, err := snd.Send(ctx, sub, msg)
	if err != nil {
		fail("invalid_subscription", clip(err.Error(), 120), "rejected")
		return
	}
	switch res.Outcome {
	case webpush.Accepted:
		err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `UPDATE channel_messages SET status = 'sent', sent_at = $2, provider_message_id = $3, reason = '', updated_at = $2 WHERE id = $1`,
				d.id, now, clip(res.Location, 300)); err != nil {
				return err
			}
			if err := q.PushMessageAnswer(ctx, store.PushMessageAnswerParams{MessageID: d.id, Answer: res.Detail}); err != nil {
				return err
			}
			if err := q.ChanMessageEventInsert(ctx, store.ChanMessageEventInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, MessageID: d.id, Kind: "sent",
				ProviderCode: res.Detail, Detail: json.RawMessage(`{}`), OccurredAt: now}); err != nil {
				return err
			}
			return q.PushKeyAccepted(ctx, store.PushKeyAcceptedParams{ID: d.keyID, At: now})
		})
		if err != nil {
			s.log().WarnContext(ctx, "push sent", "message", d.id, "err", err)
		}
		if d.demo {
			s.demoBrowser(ws, d)
		}
	case webpush.Gone:
		fail("gone", res.Detail, "gone")
	case webpush.TooLarge:
		fail("too_large", res.Detail, "")
	case webpush.Rejected:
		remove := ""
		if d.kind != "test" {
			if n, err := s.Q.PushSubscriptionFailure(ctx, d.sub); err == nil && n >= 3 {
				remove = "rejected"
			}
		}
		fail("rejected", res.Detail, remove)
	default:
		if int(d.attempts)+1 >= maxAttempts {
			fail("unavailable", res.Detail, "")
			return
		}
		wait := res.RetryAfter
		if wait <= 0 {
			wait = time.Duration(1<<min(d.attempts, 6)) * 30 * time.Second
		}
		if _, err := s.Pool.Exec(ctx, `UPDATE channel_messages SET status = 'queued', send_after = $2, reason = 'retry', provider_code = $3, updated_at = $4 WHERE id = $1`,
			d.id, now.Add(wait), res.Detail, now); err != nil {
			s.log().WarnContext(ctx, "push retry", "message", d.id, "err", err)
		}
	}
}

// demoBrowser plays a browser's part for the demo's subscriptions: most
// notifications are shown a moment after the push service accepts them,
// and some are clicked.
func (s *Service) demoBrowser(ws *store.Workspace, d due) {
	if rand.IntN(100) >= 96 { //nolint:gosec // G404: the demo's simulated browser, not a secret
		return
	}
	wsCopy := *ws
	time.AfterFunc(time.Duration(1500+rand.IntN(3000))*time.Millisecond, func() { //nolint:gosec // G404: as above
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := s.Report(ctx, &wsCopy, d.id, d.token, "shown"); err != nil {
			s.log().WarnContext(ctx, "push demo shown", "err", err)
		}
		if rand.IntN(100) < 15 { //nolint:gosec // G404: as above
			time.Sleep(time.Duration(2+rand.IntN(6)) * time.Second) //nolint:gosec // G404: as above
			if _, err := s.Click(ctx, d.id, d.token, 0, "Demo browser"); err != nil {
				s.log().WarnContext(ctx, "push demo click", "err", err)
			}
		}
	})
}

// finishSends completes the sends with nothing left to send.
func (s *Service) finishSends(ctx context.Context) error {
	rows, err := s.Pool.Query(ctx, `SELECT s.campaign_id FROM push_sends s WHERE s.phase = 'sending'
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
		var sent, failed, gone int64
		_ = s.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE sent_at IS NOT NULL), count(*) FILTER (WHERE status = 'failed'),
			count(*) FILTER (WHERE reason = 'gone') FROM channel_messages WHERE campaign_id = $1 AND kind = 'campaign'`, id).Scan(&sent, &failed, &gone)
		err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
			tag, err := tx.Exec(ctx, `UPDATE push_sends SET phase = 'completed', completed_at = $2, last_send_at = $2 WHERE campaign_id = $1 AND phase = 'sending'`, id, now)
			if err != nil || tag.RowsAffected() == 0 {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE campaigns SET status = 'sent', completed_at = $2, updated_at = $2 WHERE id = $1`, id, now); err != nil {
				return err
			}
			if _, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "campaign.completed", map[string]any{"campaign_id": id.String(), "name": name, "channel": "push",
				"sent": sent, "failed": failed}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.completed", TargetType: "campaign", TargetID: id.String(), TargetName: name,
				Actor: systemActor, Before: map[string]any{"status": "sending"}, After: map[string]any{"status": "sent"},
				Metadata: map[string]any{"channel": "push", "sent": sent, "failed": failed, "expired_removed": gone}})
		})
		if err != nil {
			return err
		}
	}
	return nil
}
