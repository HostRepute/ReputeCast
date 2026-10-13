package connect

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/mail"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/domains"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/systemmail"
)

// ConnectSenderOption is a sender the test can come from.
type ConnectSenderOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	// Verified is true when the new provider can sign for its domain.
	Verified bool `json:"verified"`
}

// ConnectTimelineStep is one line of the delivery timeline.
type ConnectTimelineStep struct {
	Key    string     `json:"key" enum:"queued,accepted,sent,delivered,opened"`
	Title  string     `json:"title"`
	Detail string     `json:"detail"`
	Status string     `json:"status" enum:"done,waiting,failed"`
	At     *time.Time `json:"at,omitempty"`
}

// ConnectAuthResult is what the receiving mailbox provider will see for the
// test's authentication, from the DNS records of the sender's domain.
type ConnectAuthResult struct {
	Receiver string `json:"receiver"`
	Header   string `json:"header"`
	DKIM     string `json:"dkim" enum:"pass,fail,none"`
	SPF      string `json:"spf" enum:"pass,fail,none"`
	DMARC    string `json:"dmarc" enum:"pass,fail,none"`
	SizeKB   int    `json:"sizeKb"`
	Clipped  bool   `json:"clipped"`
}

// ConnectTestStatus is step 4.
type ConnectTestStatus struct {
	MessageID string                `json:"messageId"`
	SenderID  string                `json:"senderId"`
	To        string                `json:"to"`
	Subject   string                `json:"subject"`
	Senders   []ConnectSenderOption `json:"senders"`
	// Listening is true while events may still arrive.
	Listening bool                  `json:"listening"`
	Timeline  []ConnectTimelineStep `json:"timeline"`
	// RoundTrip is the closing line once delivered ("Round trip 2.4 s …").
	RoundTrip string             `json:"roundTrip"`
	Auth      *ConnectAuthResult `json:"auth,omitempty"`
	Seeds     []string           `json:"seeds"`
	// SeedList is the workspace's saved seed addresses by mailbox provider.
	SeedList map[string]string `json:"seedList"`
}

// ConnectTestInput sends a test through the new provider.
type ConnectTestInput struct {
	SenderID string `json:"senderId" format:"uuid"`
	To       string `json:"to,omitempty" maxLength:"254"`
	Subject  string `json:"subject" minLength:"1" maxLength:"200"`
	// Seeds sends the test to a seed list instead (one address per mailbox
	// provider: gmail, outlook, yahoo, icloud); they are saved for next time.
	Seeds map[string]string `json:"seeds,omitempty"`
}

// mailboxOf names the mailbox provider of an address from its MX records
// (Google Workspace domains are Gmail), falling back to the domain name.
func (s *Service) mailboxOf(ctx context.Context, email string) (string, string) {
	domain := email[strings.LastIndexByte(email, '@')+1:]
	mx := ""
	if s.Domains != nil && s.Domains.DNS != nil {
		cctx, cancel := context.WithTimeout(ctx, 4*time.Second)
		recs, err := s.Domains.DNS.MX(cctx, domain)
		cancel()
		if err == nil && len(recs) > 0 {
			slices.SortFunc(recs, func(a, b dnscheck.MX) int { return int(a.Pref) - int(b.Pref) })
			mx = strings.TrimSuffix(recs[0].Host, ".")
		}
	}
	low := strings.ToLower(mx)
	switch {
	case strings.Contains(low, "google.com") || strings.Contains(low, "googlemail.com"):
		return routing.Gmail, mx
	case strings.Contains(low, "outlook.com") || strings.Contains(low, "hotmail.com"):
		return routing.Outlook, mx
	case strings.Contains(low, "yahoodns.net") || strings.Contains(low, "yahoo.com"):
		return routing.Yahoo, mx
	case strings.Contains(low, "icloud.com") || strings.Contains(low, "me.com"):
		return routing.ICloud, mx
	}
	return routing.MailboxProvider(email), mx
}

// SendTest sends a test message through the setup's provider (never
// through routing) and records it like any send, so the timeline can watch
// its events arrive.
func (s *Service) SendTest(ctx context.Context, ws *store.Workspace, rawID string, in ConnectTestInput) (ConnectTestStatus, error) {
	st, err := s.setup(ctx, ws, rawID)
	if err != nil {
		return ConnectTestStatus{}, err
	}
	p, err := s.provider(ctx, ws, st)
	if err != nil {
		return ConnectTestStatus{}, err
	}
	sid, err := uuid.Parse(in.SenderID)
	if err != nil {
		return ConnectTestStatus{}, api.Invalid("body.senderId", "Pick the From address.")
	}
	sender, err := s.Q.SenderGet(ctx, store.SenderGetParams{WorkspaceID: ws.ID, ID: sid})
	if err != nil {
		return ConnectTestStatus{}, api.Invalid("body.senderId", "Pick a sender of this workspace.")
	}
	recipients := []string{}
	if len(in.Seeds) > 0 {
		for _, mb := range []string{"gmail", "outlook", "yahoo", "icloud"} {
			if v := strings.TrimSpace(in.Seeds[mb]); v != "" {
				a, err := mail.ParseAddress(v)
				if err != nil {
					return ConnectTestStatus{}, api.Invalid("body.seeds."+mb, "Enter an address like you@"+mb+".com.")
				}
				recipients = append(recipients, strings.ToLower(a.Address))
			}
		}
		if len(recipients) == 0 {
			return ConnectTestStatus{}, api.Invalid("body.seeds", "Enter at least one seed address.")
		}
		if err := s.saveSeeds(ctx, ws, in.Seeds); err != nil {
			return ConnectTestStatus{}, err
		}
	} else {
		a, err := mail.ParseAddress(strings.TrimSpace(in.To))
		if err != nil {
			return ConnectTestStatus{}, api.Invalid("body.to", "Enter the address to send the test to.")
		}
		recipients = append(recipients, strings.ToLower(a.Address))
	}
	d, err := descriptor(st.Adapter)
	if err != nil {
		return ConnectTestStatus{}, err
	}
	c := copyFor(d)
	brand := systemmail.Brand{Name: "ReputeCast", Accent: "#0F766E", Footer: "Sent by ReputeCast for " + ws.Name}
	var first uuid.UUID
	size := 0
	for _, to := range recipients {
		id := uuid.Must(uuid.NewV7())
		pixel := ""
		if s.Links != nil && s.URLs.Base() != "" {
			pixel = `<img src="` + html.EscapeString(s.URLs.Open(s.Links.Sign(linkClaims(ws.ID, id)))) + `" width="1" height="1" alt="" style="display:block">`
		}
		body := fmt.Sprintf(`<p style="margin:0 0 12px">This is a test from <b>%s</b> through <b>%s</b>.</p><p style="margin:0 0 12px">If it reached your inbox, the provider works. The delivery timeline in ReputeCast shows each step, and opening this message shows up there too.</p>%s`,
			html.EscapeString(ws.Name), html.EscapeString(p.Name), pixel)
		text := fmt.Sprintf("This is a test from %s through %s.\n\nIf it reached your inbox, the provider works. The delivery timeline in ReputeCast shows each step.\n", ws.Name, p.Name)
		rendered, err := systemmail.Render("test", brand, in.Subject, text, body)
		if err != nil {
			return ConnectTestStatus{}, err
		}
		var replyTo *mail.Address
		if sender.ReplyTo != "" && !strings.EqualFold(sender.ReplyTo, sender.Email) {
			replyTo = &mail.Address{Address: sender.ReplyTo}
		}
		msg := providers.Message{ID: id, Stream: testStream(p), From: mail.Address{Name: sender.Name, Address: sender.Email}, ReplyTo: replyTo,
			To: mail.Address{Address: to}, Subject: in.Subject, HTML: rendered.HTML, Text: rendered.Text,
			Headers: map[string]string{"X-ReputeCast-Kind": "test"}, Tags: map[string]string{"rc_kind": "test"}}
		if _, err := s.Deliver.Send(ctx, p, msg, "test"); err != nil {
			if first == uuid.Nil {
				return ConnectTestStatus{}, api.Err(422, api.CodeProviderUnavailable, p.Name+" didn't accept the test: "+callMessage(err))
			}
			continue
		}
		if first == uuid.Nil {
			first = id
			// Headers and MIME parts add about a kilobyte to the bodies.
			size = len(rendered.HTML) + len(rendered.Text) + 1024
		}
	}
	_, err = s.update(ctx, ws, st.ID, "provider.test_send", map[string]any{"provider": p.Name, "recipients": len(recipients), "kind": c.Short},
		func(_ *store.ProviderSetup, state *State) error {
			state.Test = &TestStep{MessageID: first.String(), SenderID: sender.ID.String(), From: sender.Email, To: recipients[0], Subject: in.Subject, QueuedAt: s.now(),
				SizeBytes: size}
			if len(in.Seeds) > 0 {
				state.Test.Seeds = recipients
			}
			return nil
		})
	if err != nil {
		return ConnectTestStatus{}, err
	}
	return s.TestStatus(ctx, ws, rawID)
}

func (s *Service) saveSeeds(ctx context.Context, ws *store.Workspace, seeds map[string]string) error {
	row, err := s.Q.SenderWorkspaceDefaults(ctx, ws.ID)
	if err != nil {
		return err
	}
	sd := map[string]any{}
	_ = json.Unmarshal(row.SendingDefaults, &sd)
	clean := map[string]string{}
	for k, v := range seeds {
		if v = strings.TrimSpace(v); v != "" {
			clean[k] = strings.ToLower(v)
		}
	}
	sd["seed_list"] = clean
	b, _ := json.Marshal(sd)
	return s.Q.SenderSetSendingDefaults(ctx, store.SenderSetSendingDefaultsParams{ID: ws.ID, SendingDefaults: b, UpdatedAt: s.now()})
}

// TestStatus is the test form and the latest test's delivery timeline.
func (s *Service) TestStatus(ctx context.Context, ws *store.Workspace, rawID string) (ConnectTestStatus, error) {
	st, err := s.setup(ctx, ws, rawID)
	if err != nil {
		return ConnectTestStatus{}, err
	}
	p, err := s.provider(ctx, ws, st)
	if err != nil {
		return ConnectTestStatus{}, err
	}
	d, err := descriptor(st.Adapter)
	if err != nil {
		return ConnectTestStatus{}, err
	}
	c := copyFor(d)
	state := decodeState(st.State)
	out := ConnectTestStatus{Senders: []ConnectSenderOption{}, Timeline: []ConnectTimelineStep{}, Seeds: []string{}, SeedList: map[string]string{},
		Subject: "ReputeCast test from " + p.Name}
	if row, err := s.Q.SenderWorkspaceDefaults(ctx, ws.ID); err == nil {
		var sd struct {
			Seeds map[string]string `json:"seed_list"`
		}
		_ = json.Unmarshal(row.SendingDefaults, &sd)
		for k, v := range sd.Seeds {
			out.SeedList[k] = v
		}
	}
	senders, err := s.Q.SenderList(ctx, ws.ID)
	if err != nil {
		return ConnectTestStatus{}, err
	}
	for _, sn := range senders {
		ver := map[string]string{}
		_ = json.Unmarshal(sn.Verification, &ver)
		out.Senders = append(out.Senders, ConnectSenderOption{ID: sn.ID.String(), Label: sn.Name + " <" + sn.Email + ">", Verified: ver[p.ID.String()] == "verified"})
	}
	// The default From: the workspace's default sender when the provider
	// can sign for it, else the first verified one.
	if ws.DefaultSenderID != nil {
		out.SenderID = ws.DefaultSenderID.String()
	}
	for _, o := range out.Senders {
		if o.ID == out.SenderID && !o.Verified {
			out.SenderID = ""
		}
	}
	if out.SenderID == "" {
		for _, o := range out.Senders {
			if o.Verified {
				out.SenderID = o.ID
				break
			}
		}
	}
	if out.SenderID == "" && len(out.Senders) > 0 {
		out.SenderID = out.Senders[0].ID
	}
	if u := principalUser(ctx); u != nil {
		if usr, err := s.Q.GetUser(ctx, *u); err == nil {
			out.To = usr.Email
		}
	}
	if state.Test == nil {
		return out, nil
	}
	t := state.Test
	out.MessageID, out.SenderID, out.To, out.Subject, out.Seeds = t.MessageID, t.SenderID, t.To, t.Subject, append([]string{}, t.Seeds...)
	// The form alone when the test message is gone.
	m, ok := s.testMessage(ctx, ws.ID, t.MessageID)
	if !ok {
		return out, nil
	}
	loc := state2Location(ws.Timezone)
	clockMs := func(t time.Time) string { return t.In(loc).Format("15:04:05.000") }
	queued := m.QueuedAt
	out.Timeline = append(out.Timeline, ConnectTimelineStep{Key: "queued", Title: "Queued by ReputeCast", Status: "done", At: &queued,
		Detail: clockMs(queued) + " · message rc_" + strings.ReplaceAll(m.ID.String(), "-", "")[:8] + "…"})
	accepted := ConnectTimelineStep{Key: "accepted", Title: "Accepted by " + p.Name, Status: "waiting", Detail: "Waiting for " + c.Short}
	switch {
	case m.Status == "failed":
		accepted.Status, accepted.Detail = "failed", m.Error
	case m.SentAt != nil:
		accepted.Status, accepted.At = "done", m.SentAt
		detail := clockMs(*m.SentAt) + " · " + fmt.Sprint(m.SentAt.Sub(queued).Milliseconds()) + " ms"
		if m.ProviderMessageID != nil && *m.ProviderMessageID != "" {
			detail += " · MessageId " + shorten(*m.ProviderMessageID)
		}
		var cfg struct {
			Sets map[string]string `json:"configuration_sets"`
			Set  string            `json:"configuration_set"`
		}
		_ = json.Unmarshal(p.Config, &cfg)
		if cs := cfg.Sets[m.Stream]; cs != "" {
			detail += " · configuration set " + cs
		} else if cfg.Set != "" {
			detail += " · configuration set " + cfg.Set
		}
		accepted.Detail = detail
	}
	out.Timeline = append(out.Timeline, accepted)
	mailbox, mx := s.mailboxOf(ctx, t.To)
	events, _ := s.Q.ConnectMessageEvents(ctx, &m.ID)
	var delivered, failed, opened *store.DeliveryEvent
	for i := range events {
		e := &events[i]
		switch e.Type {
		case "delivered":
			if delivered == nil {
				delivered = e
			}
		case "hard_bounce", "soft_bounce", "rejected", "complaint", "deferred":
			if failed == nil {
				failed = e
			}
		case "open":
			if opened == nil {
				opened = e
			}
		}
	}
	sent := ConnectTimelineStep{Key: "sent", Title: "Sent to " + routing.MailboxName(mailbox), Status: "waiting", Detail: "Waiting for the first event from " + c.Short}
	webhook := state.Connection.Webhook
	// In development a demo provider's test goes to Mailpit, which records
	// the delivery itself; the demo's own test message has real events.
	mailpit := delivered != nil && strings.HasPrefix(delivered.OriginalType, "Delivered to Mailpit")
	inferred := d.Capabilities.Webhooks == "" || d.Capabilities.Webhooks == "Bounce mailbox" || mailpit
	if m.SentAt != nil {
		first := t.FirstEvent
		if first == nil && !inferred {
			if first = s.firstWebhook(ctx, p, m, d); first != nil {
				_ = s.setFirstEvent(ctx, st.ID, *first)
			}
		}
		if first != nil && !inferred {
			at := first.At
			event := c.Short + " event"
			if first.Name != "" {
				event = c.Short + " \"" + first.Name + "\" event"
			}
			sent.Status, sent.At = "done", &at
			sent.Detail = clockMs(at) + " · " + event + " via " + first.Via + " · " + fmt.Sprint(at.Sub(*m.SentAt).Milliseconds()) + " ms after accept"
		} else if delivered != nil {
			at := delivered.OccurredAt
			sent.Status, sent.At = "done", &at
			if inferred {
				sent.Detail = clockMs(at) + " · " + smtpVia(p, mailpit) + " accepted it for delivery"
			} else {
				sent.Detail = clockMs(at) + " · " + c.Short + " event"
			}
		}
	}
	if webhook != nil && webhook.Mode == "polling" && sent.Status == "waiting" {
		sent.Detail = "Waiting · ReputeCast polls " + c.Short + "'s events API every minute"
	}
	out.Timeline = append(out.Timeline, sent)
	host := mx
	if delivered != nil {
		// The receiving server, as the provider reported it.
		if parts := strings.SplitN(delivered.Diagnostic, " · smtpResponse from ", 2); len(parts) == 2 && parts[1] != "" {
			host = parts[1]
		}
	}
	if host == "" {
		host = t.To[strings.LastIndexByte(t.To, '@')+1:]
	}
	del := ConnectTimelineStep{Key: "delivered", Title: "Delivered to " + host, Status: "waiting", Detail: "Waiting for the delivery event"}
	switch {
	case delivered != nil:
		at := delivered.OccurredAt
		del.Status, del.At = "done", &at
		code := strings.TrimSpace(delivered.OriginalCode + " " + strings.SplitN(delivered.Diagnostic, " · smtpResponse from ", 2)[0])
		detail := clockMs(at) + " · " + c.Short + " \"" + strings.SplitN(delivered.OriginalType, " ·", 2)[0] + "\" event"
		if code != "" {
			detail += " · `" + code + "`"
		}
		del.Detail = detail
		if mailpit {
			del.Title = "Delivered to Mailpit (development)"
			del.Detail = clockMs(at) + " · SMTP reply"
			if code != "" {
				del.Detail += " `" + code + "`"
			}
		}
		out.RoundTrip = roundTrip(at.Sub(queued), d, mailpit)
		ms := at.Sub(queued).Milliseconds()
		if t.RoundTripMs != ms {
			_ = s.setRoundTrip(ctx, st.ID, ms)
		}
	case failed != nil:
		at := failed.OccurredAt
		del.Status, del.At = "failed", &at
		del.Title = map[string]string{"hard_bounce": "Bounced", "soft_bounce": "Bounced (temporary)", "rejected": "Rejected", "complaint": "Marked as spam", "deferred": "Deferred"}[failed.Type]
		del.Detail = clockMs(at) + " · `" + strings.TrimSpace(failed.OriginalCode+" "+failed.Diagnostic) + "`"
	}
	out.Timeline = append(out.Timeline, del)
	op := ConnectTimelineStep{Key: "opened", Title: "Opened", Status: "waiting", Detail: "Waiting · open the message in your inbox. Apple Mail prefetches count as machine opens."}
	if m.OpenedAt != nil {
		op.Status, op.At, op.Detail = "done", m.OpenedAt, clockMs(*m.OpenedAt)+" · counted by ReputeCast's open pixel"
	} else if opened != nil {
		at := opened.OccurredAt
		op.Status, op.At, op.Detail = "done", &at, clockMs(at)+" · "+c.Short+" open event"
	}
	out.Timeline = append(out.Timeline, op)
	out.Listening = m.Status != "failed" && (delivered == nil && failed == nil || op.Status == "waiting") && s.now().Sub(queued) < 30*time.Minute
	if m.SentAt != nil {
		out.Auth = s.authResult(ctx, ws, p, t, mailbox, mx)
	}
	return out, nil
}

func state2Location(tz string) *time.Location { return state.Location(tz) }

// testStream is the stream a test goes on: Broadcasts, the provider's
// main use, unless its terms keep it to transactional mail.
func testStream(p store.Provider) providers.Stream {
	if state.Caps(p).BulkAllowed() {
		return providers.StreamBroadcast
	}
	return providers.StreamSystem
}

func smtpVia(p store.Provider, mailpit bool) string {
	if mailpit {
		return "Mailpit (development)"
	}
	var c struct {
		Host string `json:"host"`
	}
	_ = json.Unmarshal(p.Config, &c)
	if c.Host != "" {
		return c.Host
	}
	return p.Name
}

func shorten(id string) string {
	id = strings.Trim(id, "<>")
	if len(id) <= 14 {
		return id
	}
	return id[:7] + "…" + id[len(id)-4:]
}

func roundTrip(d time.Duration, desc providers.Descriptor, mailpit bool) string {
	secs := fmt.Sprintf("%.1f s", d.Seconds())
	switch {
	case mailpit:
		return "Round trip " + secs + " from queue to Mailpit. In development the demo's providers deliver there."
	case desc.Capabilities.Webhooks == "Bounce mailbox" || desc.Capabilities.Webhooks == "":
		return "Round trip " + secs + " from queue to the server's 250 reply. Bounces arrive through the bounce mailbox."
	}
	return "Round trip " + secs + " from queue to delivery event. Webhooks work."
}

// firstWebhook finds the provider's first webhook about the test message:
// its provider message id is in the payload, and it came while the
// timeline listens (30 minutes after the send).
func (s *Service) firstWebhook(ctx context.Context, p store.Provider, m store.Message, d providers.Descriptor) *TestEvent {
	if m.SentAt == nil || m.ProviderMessageID == nil {
		return nil
	}
	pmid := strings.Trim(strings.TrimSpace(*m.ProviderMessageID), "<>")
	if pmid == "" {
		return nil
	}
	hook, err := s.Q.ConnectFirstWebhook(ctx, store.ConnectFirstWebhookParams{ProviderID: p.ID, Since: *m.SentAt, Until: m.SentAt.Add(30 * time.Minute), ProviderMessageID: pmid})
	if err != nil {
		return nil
	}
	via := "webhook"
	if d.Capabilities.Webhooks == "SNS" {
		via = "SNS"
	}
	return &TestEvent{At: hook.ReceivedAt, Name: webhookEventName(hook.Body), Via: via}
}

// setFirstEvent keeps the first provider event of the setup's test.
func (s *Service) setFirstEvent(ctx context.Context, setupID uuid.UUID, ev TestEvent) error {
	return s.updateTest(ctx, setupID, func(t *TestStep) bool {
		if t.FirstEvent != nil {
			return false
		}
		t.FirstEvent = &ev
		return true
	})
}

func (s *Service) setRoundTrip(ctx context.Context, setupID uuid.UUID, ms int64) error {
	return s.updateTest(ctx, setupID, func(t *TestStep) bool {
		t.RoundTripMs = ms
		return true
	})
}

// updateTest changes what the setup remembers about its test message,
// under the row lock (the status poll writes while people move on).
func (s *Service) updateTest(ctx context.Context, setupID uuid.UUID, fn func(t *TestStep) bool) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		st, err := q.SetupLock(ctx, setupID)
		if err != nil {
			return err
		}
		state := decodeState(st.State)
		if state.Test == nil || !fn(state.Test) {
			return nil
		}
		_, err = s.save(ctx, q, st, state)
		return err
	})
}

// authResult predicts the receiver's Authentication-Results from the
// sender domain's records for this provider, the same checks a mailbox
// provider runs.
func (s *Service) authResult(ctx context.Context, ws *store.Workspace, p store.Provider, t *TestStep, mailbox, mx string) *ConnectAuthResult {
	from := strings.ToLower(t.From)
	domain := from[strings.LastIndexByte(from, '@')+1:]
	size := t.SizeBytes
	if size == 0 {
		size = 4 << 10
	}
	out := &ConnectAuthResult{Receiver: routing.MailboxName(mailbox), DKIM: "none", SPF: "none", DMARC: "none", SizeKB: (size + 1023) / 1024, Clipped: size > 102<<10}
	authserv := map[string]string{routing.Gmail: "mx.google.com", routing.Outlook: "spf.protection.outlook.com", routing.Yahoo: "atlas.yahoo.com", routing.ICloud: "mx.icloud.com"}[mailbox]
	if authserv == "" {
		authserv = mx
		if out.Receiver == "Other" && mx != "" {
			out.Receiver = mx
		}
	}
	d, err := s.Q.DomGetByName(ctx, store.DomGetByNameParams{WorkspaceID: ws.ID, Domain: domain})
	if err != nil {
		return out
	}
	recs := domains.DecodeRecords(d.DnsRecords)
	pid := p.ID.String()
	dkimN, dkimOK := 0, 0
	mailFrom, mfOK := "", false
	for _, r := range recs {
		if r.ProviderID != pid {
			continue
		}
		switch r.Purpose {
		case domains.PurposeDKIM:
			dkimN++
			if r.Status == domains.StatusVerified {
				dkimOK++
			}
		case domains.PurposeReturnPath:
			mailFrom, mfOK = r.Name, r.Status == domains.StatusVerified
		}
	}
	if dkimN > 0 {
		out.DKIM = map[bool]string{true: "pass", false: "fail"}[dkimOK == dkimN]
	}
	if mailFrom != "" {
		out.SPF = map[bool]string{true: "pass", false: "fail"}[mfOK]
	}
	var det domains.AuthDetail
	_ = json.Unmarshal(d.AuthDetail, &det)
	policy := strings.ToUpper(det.DMARCPolicy)
	switch {
	case det.DMARCStatus != dnscheck.StatusPass:
		out.DMARC = "none"
	case out.DKIM == "pass" || out.SPF == "pass" && dnscheck.Aligned(mailFrom, domain, false):
		out.DMARC = "pass"
	default:
		out.DMARC = "fail"
	}
	parts := []string{authserv}
	if out.DKIM != "none" {
		parts = append(parts, "dkim="+out.DKIM+" header.d="+domain)
	}
	if out.SPF != "none" {
		parts = append(parts, "spf="+out.SPF+" smtp.mailfrom="+mailFrom)
	}
	dm := "dmarc=" + out.DMARC
	if policy != "" {
		dm += " (p=" + policy + ")"
	}
	parts = append(parts, dm+" header.from="+domain)
	out.Header = "Authentication-Results: " + strings.Join(parts, "; ")
	return out
}

// openURL is the open-pixel URL of a test message.
// linkClaims signs the test message's open pixel.
func linkClaims(ws, msg uuid.UUID) linktoken.Claims {
	return linktoken.Claims{Kind: linktoken.Open, WorkspaceID: ws, MessageID: msg}
}

// testMessage loads the setup's test message.
func (s *Service) testMessage(ctx context.Context, wsID uuid.UUID, rawID string) (store.Message, bool) {
	mid, err := uuid.Parse(rawID)
	if err != nil {
		return store.Message{}, false
	}
	m, err := s.Q.ConnectMessage(ctx, store.ConnectMessageParams{WorkspaceID: wsID, ID: mid})
	return m, err == nil
}

// webhookEventName reads the provider's own name for the first event of a
// webhook payload ("Send" from SES through SNS, "Delivery" from Postmark,
// "delivered" from Mailgun), for the timeline. Payloads are provider JSON:
// an SNS envelope's Message is unwrapped, arrays give their first item and
// the usual event keys are looked up a few levels down. Empty when the
// payload names no event.
func webhookEventName(body []byte) string {
	var v any
	if json.Unmarshal(body, &v) != nil {
		return ""
	}
	return eventNameIn(v, 0)
}

func eventNameIn(v any, depth int) string {
	if depth > 3 {
		return ""
	}
	switch x := v.(type) {
	case []any:
		if len(x) > 0 {
			return eventNameIn(x[0], depth+1)
		}
	case map[string]any:
		if msg, ok := x["Message"].(string); ok && x["Type"] == "Notification" {
			var inner any
			if json.Unmarshal([]byte(msg), &inner) == nil {
				return eventNameIn(inner, depth+1)
			}
		}
		for _, k := range []string{"eventType", "RecordType", "event", "type"} {
			if s, ok := x[k].(string); ok && s != "" && len(s) <= 60 && !strings.ContainsAny(s, "\n\"`") {
				return s
			}
		}
		for _, k := range []string{"event-data", "msys", "message_event", "data"} {
			if inner, ok := x[k]; ok {
				if s := eventNameIn(inner, depth+1); s != "" {
					return s
				}
			}
		}
		if m, ok := x["msys"].(map[string]any); ok {
			for _, inner := range m {
				if s := eventNameIn(inner, depth+1); s != "" {
					return s
				}
			}
		}
	}
	return ""
}
