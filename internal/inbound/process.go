package inbound

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/textproto"
	"path"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/optout"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/smtp/dsn"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// Kinds of received message (inbound_messages.kind).
const (
	KindReply          = "reply"
	KindUnsubscribe    = "unsubscribe"
	KindDMARCReport    = "dmarc_report"
	KindFeedbackReport = "feedback_report"
	KindTest           = "test"
	KindOther          = "other"
)

// Webhook events (G-20).
const (
	EventReplied  = "email.replied"
	EventReceived = "email.received"
)

// ProcessArgs parses one stored arrival (inbound_raw) and applies what each
// message in it asks for.
type ProcessArgs struct {
	RawID uuid.UUID `json:"raw_id"`
}

// Kind implements river.JobArgs.
func (ProcessArgs) Kind() string { return "inbound.process" }

// processAttempts is how often a parse is tried before the row is kept
// with its error (a provider's API that doesn't answer).
const processAttempts = 8

// InsertOpts implements river.JobArgsWithInsertOpts.
func (ProcessArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, MaxAttempts: processAttempts}
}

// ProcessWorker runs ProcessArgs.
type ProcessWorker struct {
	river.WorkerDefaults[ProcessArgs]
	S *Service
}

// Timeout bounds one arrival (Resend's raw message is downloaded here).
func (w *ProcessWorker) Timeout(*river.Job[ProcessArgs]) time.Duration { return 2 * time.Minute }

// Work implements river.Worker.
func (w *ProcessWorker) Work(ctx context.Context, job *river.Job[ProcessArgs]) error {
	err := w.S.Process(ctx, job.Args.RawID)
	if err != nil && job.Attempt >= processAttempts {
		// The last try: keep the row with what went wrong rather than
		// retrying forever; the sweep doesn't pick processed rows up.
		now := w.S.now()
		_ = w.S.Q.InboundRawProcessed(ctx, store.InboundRawProcessedParams{ID: job.Args.RawID, At: &now, ProcessError: clipString(err.Error(), 1000)})
		return nil
	}
	return err
}

// received is one message of an arrival, read.
type received struct {
	in     providers.InboundMessage
	parsed Parsed
	// raw is the whole message when ReputeCast has it (reports are read
	// from it).
	raw []byte
}

// Process parses a stored arrival and applies it, once.
func (s *Service) Process(ctx context.Context, rawID uuid.UUID) error {
	raw, err := s.Q.InboundRawGet(ctx, rawID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // purged, or its route was removed
	}
	if err != nil {
		return err
	}
	if raw.ProcessedAt != nil {
		return nil
	}
	now := s.now()
	done := func(msg string) error {
		return s.Q.InboundRawProcessed(ctx, store.InboundRawProcessedParams{ID: raw.ID, At: &now, ProcessError: msg})
	}
	if !raw.SignatureOk {
		return done("signature verification failed")
	}
	route, err := s.Q.InboundRouteByID(ctx, raw.RouteID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	ws, err := s.Q.GetWorkspace(ctx, raw.WorkspaceID)
	if err != nil {
		return err
	}
	msgs, err := s.messagesOf(ctx, route, raw)
	if err != nil {
		if errors.Is(err, errPermanent) {
			return done(err.Error())
		}
		return err
	}
	suppressed := 0
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		suppressed = 0
		detail := ""
		for _, m := range msgs {
			n, note, err := s.apply(ctx, tx, &ws, route, raw, m)
			if err != nil {
				return err
			}
			suppressed += n
			if !note.IsZero() {
				detail = keepDetail(note)
			}
		}
		q := store.New(tx)
		if len(msgs) > 0 {
			if err := q.InboundRouteReceived(ctx, store.InboundRouteReceivedParams{ID: route.ID, At: raw.ReceivedAt, StatusDetail: detail}); err != nil {
				return err
			}
		}
		return q.InboundRawProcessed(ctx, store.InboundRawProcessedParams{ID: raw.ID, At: &now})
	})
	if err != nil {
		return err
	}
	if suppressed > 0 && s.OnSuppressed != nil {
		s.OnSuppressed(ctx, ws.ID)
	}
	return nil
}

// errPermanent marks an arrival that can never be read (retrying won't help).
var errPermanent = errors.New("inbound")

// messagesOf reads an arrival: a provider's request through its adapter,
// a mailbox's or the test's message as it is.
func (s *Service) messagesOf(ctx context.Context, route store.InboundRoute, raw store.InboundRaw) ([]received, error) {
	if raw.Source != "hook" {
		p, err := Parse(raw.Body)
		if err != nil {
			return nil, fmt.Errorf("%w: the message can't be read: %w", errPermanent, err)
		}
		return []received{{in: providers.InboundMessage{Raw: raw.Body, ReceivedAt: raw.ReceivedAt}, parsed: p, raw: raw.Body}}, nil
	}
	rc, _, err := s.receiver(ctx, route)
	if errors.Is(err, errUnsupported) || errors.Is(err, pgx.ErrNoRows) || errors.Is(err, errNoProvider) {
		return nil, fmt.Errorf("%w: %w", errPermanent, err)
	}
	if err != nil {
		return nil, err
	}
	ar, err := s.adapterRoute(route)
	if err != nil {
		return nil, err
	}
	in, err := rc.ParseInbound(providers.WithSignatureVerified(ctx), ar, rebuild(raw), raw.Body)
	if err != nil {
		return nil, err // the provider's API may answer later; River retries
	}
	out := make([]received, 0, len(in))
	for _, m := range in {
		if m.ReceivedAt.IsZero() {
			m.ReceivedAt = raw.ReceivedAt
		}
		p, err := FromInbound(m)
		if err != nil {
			s.Log.WarnContext(ctx, "inbound message unreadable", "raw", raw.ID, "err", err)
			continue
		}
		out = append(out, received{in: m, parsed: p, raw: m.Raw})
	}
	return out, nil
}

// pathHeader keeps the part of the hook's path after the token (Mailgun
// posts to …/mime), so the rebuilt request has it.
const pathHeader = "X-Reputecast-Hook-Path"

// rebuild recreates the request the adapter parses from what was stored.
func rebuild(raw store.InboundRaw) *http.Request {
	var hs map[string]string
	_ = json.Unmarshal(raw.Headers, &hs)
	target := "http://inbound.internal/hooks/inbound/stored"
	if p := hs[pathHeader]; p != "" {
		target += "/" + strings.TrimPrefix(path.Clean("/"+p), "/")
	}
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, target, bytes.NewReader(raw.Body))
	for k, v := range hs {
		if k != pathHeader {
			r.Header.Set(k, v)
		}
	}
	if raw.ContentType != "" {
		r.Header.Set("Content-Type", raw.ContentType)
	}
	return r
}

// target is what a message was sent to on the route: the kind its address
// says, our message it answers and the address itself.
type target struct {
	kind    string
	address string
	letter  byte
	message uuid.UUID
}

// route finds the address of the box a message came to: the envelope's
// recipients first, then its headers. An address on the box with a tag that
// doesn't verify, or one ReputeCast didn't hand out, is other mail.
func (s *Service) route(box Box, workspaceID uuid.UUID, m received) target {
	cands := make([]string, 0, len(m.in.Recipients)+len(m.parsed.To))
	cands = append(cands, m.in.Recipients...)
	cands = append(cands, m.parsed.To...)
	first := ""
	other := ""
	for _, a := range cands {
		a = strings.ToLower(strings.Trim(strings.TrimSpace(a), "<>"))
		if a == "" {
			continue
		}
		if first == "" {
			first = a
		}
		tag, ok := box.tagOf(a)
		if !ok {
			continue
		}
		switch tag {
		case LocalDMARC:
			return target{kind: KindDMARCReport, address: a}
		case LocalFBL:
			return target{kind: KindFeedbackReport, address: a}
		}
		if letter, id, ok := Decode(s.Signer, workspaceID, tag); ok {
			t := target{address: a, letter: letter, message: id}
			switch letter {
			case LetterReply:
				t.kind = KindReply
			case LetterUnsubscribe:
				t.kind = KindUnsubscribe
			case LetterTest:
				t.kind = KindTest
			}
			return t
		}
		if other == "" {
			other = a
		}
	}
	if other == "" {
		other = first
	}
	return target{kind: KindOther, address: other}
}

// stored is the part of an inbound message the effects read.
type stored struct {
	id           uuid.UUID
	kind         string
	message      *store.InboundOurMessageRow
	contactID    *uuid.UUID
	campaignID   *uuid.UUID
	automationID *uuid.UUID
	autoReply    bool
	spam         bool
}

// apply stores one message and applies its kind's effect, in tx. It
// returns how many addresses a feedback report newly suppressed and a note
// for the route's status (a report about a domain the workspace doesn't
// send from).
func (s *Service) apply(ctx context.Context, tx pgx.Tx, ws *store.Workspace, route store.InboundRoute, raw store.InboundRaw, m received) (int, i18n.Text, error) {
	q := store.New(tx)
	p := m.parsed
	t := s.route(boxOf(route), ws.ID, m)
	st := stored{id: newID(), kind: t.kind}
	if t.kind == KindReply || t.kind == KindUnsubscribe {
		msg, err := q.InboundOurMessage(ctx, store.InboundOurMessageParams{ID: t.message, WorkspaceID: ws.ID})
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			// The message is gone (its campaign was deleted): other mail.
			st.kind = KindOther
		case err != nil:
			return 0, i18n.Text{}, err
		default:
			st.message = &msg
			st.contactID, st.campaignID, st.automationID = msg.ContactID, msg.CampaignID, msg.AutomationID
			if t.kind == KindUnsubscribe && msg.ContactID == nil {
				st.kind = KindOther
			}
		}
	}
	header := p.Header
	if header == nil {
		header = textproto.MIMEHeader{}
	}
	st.autoReply = st.kind == KindReply && isAutoReply(header)
	st.spam = (st.kind == KindReply || st.kind == KindOther) && spamVerdict(m.in.Spam, header)
	status := "open"
	switch st.kind {
	case KindUnsubscribe, KindDMARCReport, KindFeedbackReport, KindTest:
		status = "done"
	}
	attachments := make([]attachmentView, 0, len(p.Attachments))
	for _, a := range p.Attachments {
		attachments = append(attachments, attachmentView{Name: clipString(a.Name, 255), Type: clipString(a.ContentType, 120), Size: a.Size})
	}
	attJSON, _ := json.Marshal(attachments)
	authJSON, _ := json.Marshal(authView{SPF: m.in.SPF, DKIM: m.in.DKIM, DMARC: m.in.DMARC})
	var msgID *uuid.UUID
	if st.message != nil {
		id := st.message.ID
		msgID = &id
	}
	routeID, rawID := route.ID, raw.ID
	headerID := strings.ToLower(clipString(strings.Trim(p.MessageID, "<> "), 300))
	if st.kind == KindTest && headerID == "" {
		headerID = "test:" + t.address
	}
	at := m.in.ReceivedAt
	if at.IsZero() {
		at = raw.ReceivedAt
	}
	_, err := q.InboundMessageInsert(ctx, store.InboundMessageInsertParams{
		ID: st.id, WorkspaceID: ws.ID, RouteID: &routeID, RawID: &rawID, Kind: st.kind, MessageID: msgID, ContactID: st.contactID,
		CampaignID: st.campaignID, AutomationID: st.automationID, FromEmail: clipString(p.From, 320), FromName: clipString(p.FromName, 200),
		ToAddress: clipString(t.address, 320), Subject: clipString(p.Subject, 998), TextBody: p.Text, HasHtml: p.HasHTML, Attachments: attJSON,
		AutoReply: st.autoReply, Spam: st.spam, AsksUnsubscribe: st.kind == KindReply && asksUnsubscribe(p.Subject, p.Text), Auth: authJSON,
		Status: status, HeaderMessageID: headerID, ReceivedAt: at,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, i18n.Text{}, nil // the same message again (Message-ID): applied once
	}
	if err != nil {
		return 0, i18n.Text{}, err
	}
	switch st.kind {
	case KindReply:
		return 0, i18n.Text{}, s.applyReply(ctx, tx, ws, st, p, at)
	case KindUnsubscribe:
		return 0, i18n.Text{}, s.applyUnsubscribe(ctx, tx, ws, st, p, at)
	case KindTest:
		return 0, i18n.Text{}, q.InboundRouteTestReceived(ctx, store.InboundRouteTestReceivedParams{ID: route.ID, At: at})
	case KindDMARCReport:
		note, err := s.applyDMARC(ctx, tx, ws, p)
		return 0, note, err
	case KindFeedbackReport:
		n, err := s.applyFeedback(ctx, tx, ws, m, at)
		return n, i18n.Text{}, err
	default:
		if st.spam {
			return 0, i18n.Text{}, nil
		}
		_, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, EventReceived, map[string]any{
			"id": st.id.String(), "to": t.address, "from": p.From, "subject": p.Subject, "text": p.Text, "received_at": at,
		})
		return 0, i18n.Text{}, err
	}
}

// attachmentView is an attachment as it's kept: never its content.
type attachmentView struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Size int64  `json:"size"`
}

// authView are the receiving server's verdicts.
type authView struct {
	SPF   string `json:"spf,omitempty"`
	DKIM  string `json:"dkim,omitempty"`
	DMARC string `json:"dmarc,omitempty"`
}

// applyReply stops the flows a person's reply stops and tells developers.
func (s *Service) applyReply(ctx context.Context, tx pgx.Tx, ws *store.Workspace, st stored, p Parsed, at time.Time) error {
	if st.spam {
		return nil
	}
	human := !st.autoReply
	if human && st.automationID != nil && st.contactID != nil && s.EndRunsOnReply != nil {
		if _, err := s.EndRunsOnReply(ctx, tx, ws.ID, *st.automationID, *st.contactID, at); err != nil {
			return err
		}
	}
	data := map[string]any{
		"reply_id": st.id.String(), "from": p.From, "subject": p.Subject, "text": p.Text, "auto_reply": st.autoReply, "received_at": at,
	}
	if st.message != nil {
		data["message_id"] = st.message.ID.String()
	}
	for k, v := range map[string]*uuid.UUID{"campaign_id": st.campaignID, "automation_id": st.automationID, "contact_id": st.contactID} {
		if v != nil {
			data[k] = v.String()
		}
	}
	_, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, EventReplied, data)
	return err
}

// applyUnsubscribe applies an email to a message's mailto: unsubscribe
// address as its one-click link would be.
func (s *Service) applyUnsubscribe(ctx context.Context, tx pgx.Tx, ws *store.Workspace, st stored, p Parsed, at time.Time) error {
	if s.Audience == nil || st.contactID == nil || st.message == nil {
		return nil
	}
	args := i18n.Args{"email": p.From}
	ev := audience.PublicEvidence{Source: audience.SourceMailto, Note: mUnsubscribedByEmail.Text(i18n.Default, args), NoteText: mUnsubscribedByEmail.Keep(args), At: at}
	_, err := optout.Apply(ctx, tx, optout.Deps{Audience: s.Audience, MessageUnsubscribed: s.MessageUnsubscribed}, ws,
		optout.MessageRef{MessageID: st.message.ID, ContactID: *st.contactID}, ev, "", "")
	if errors.Is(err, audience.ErrContactGone) {
		return nil
	}
	return err
}

// reportFile is an attachment that can be a DMARC aggregate report.
func reportFile(a providers.InboundAttachment) bool {
	name := strings.ToLower(a.Name)
	for _, ext := range []string{".xml", ".xml.gz", ".gz", ".zip"} {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	switch strings.ToLower(a.ContentType) {
	case "application/xml", "text/xml", "application/gzip", "application/x-gzip", "application/zip", "application/x-zip-compressed":
		return true
	}
	return false
}

// applyDMARC keeps the aggregate reports a message carries. Reports about
// domains the workspace doesn't send from are dropped; the note says how
// many.
func (s *Service) applyDMARC(ctx context.Context, tx pgx.Tx, ws *store.Workspace, p Parsed) (i18n.Text, error) {
	if s.StoreDMARC == nil {
		return i18n.Text{}, nil
	}
	dropped := []string{}
	for _, a := range p.Attachments {
		if len(a.Data) == 0 || !reportFile(a) {
			continue
		}
		domain, err := s.StoreDMARC(ctx, tx, ws, a.Data)
		switch {
		case errors.Is(err, ErrNotOwned):
			if domain != "" {
				dropped = append(dropped, domain)
			}
		case errors.Is(err, ErrNotAReport):
			s.Log.InfoContext(ctx, "inbound dmarc attachment unreadable", "workspace", ws.ID, "name", a.Name)
		case err != nil:
			return i18n.Text{}, err
		}
	}
	if len(dropped) == 0 {
		return i18n.Text{}, nil
	}
	return mDroppedReports.Keep(i18n.Args{"count": len(dropped), "domain": dropped[0]}), nil
}

// ErrNotOwned is StoreDMARC's answer for a report about a domain the
// workspace doesn't send from; ErrNotAReport for a file that isn't one.
var (
	ErrNotOwned   = errors.New("inbound: the report is about a domain the workspace doesn't send from")
	ErrNotAReport = errors.New("inbound: not a DMARC aggregate report")
)

var msgIDRe = regexp.MustCompile(`(?i)<?([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})@`)

// applyFeedback turns an ARF complaint about a message ReputeCast sent
// into a complaint event, through the rules provider events follow (only
// for a message we sent, to that recipient).
func (s *Service) applyFeedback(ctx context.Context, tx pgx.Tx, ws *store.Workspace, m received, at time.Time) (int, error) {
	if s.ApplyReport == nil {
		return 0, nil
	}
	orig, recipient, ok := feedbackOf(m)
	if !ok {
		return 0, nil
	}
	id, ok := ourMessageID(orig)
	if !ok {
		return 0, nil
	}
	msg, err := store.New(tx).InboundOurMessage(ctx, store.InboundOurMessageParams{ID: id, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) || err == nil && msg.ProviderID == nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if recipient == "" {
		recipient = msg.Email
	}
	ev := providers.Event{Type: providers.EventComplaint, MessageID: &msg.ID, Recipient: recipient, OccurredAt: at, OriginalType: "arf"}
	body := m.raw
	if len(body) > 64<<10 {
		body = body[:64<<10]
	}
	return s.ApplyReport(ctx, tx, *msg.ProviderID, "inbound", at, body, []providers.Event{ev})
}

// feedbackOf reads an ARF report: the headers of the message it's about
// and the complaining recipient.
func feedbackOf(m received) (textproto.MIMEHeader, string, bool) {
	if len(m.raw) > 0 {
		rep, err := dsn.Parse(m.raw)
		if err != nil || rep.Kind != dsn.KindARF {
			return nil, "", false
		}
		return rep.Original, strings.ToLower(strings.TrimSpace(rep.ARFRecipient)), true
	}
	// A provider that parsed the report itself: its parts are attachments.
	arf := false
	var orig textproto.MIMEHeader
	for _, a := range m.parsed.Attachments {
		switch strings.ToLower(a.ContentType) {
		case "message/feedback-report":
			arf = true
		case "message/rfc822", "text/rfc822-headers":
			if h, err := textproto.NewReader(bufio.NewReader(bytes.NewReader(a.Data))).ReadMIMEHeader(); err == nil || len(h) > 0 {
				orig = h
			}
		}
	}
	return orig, "", arf && orig != nil
}

// ourMessageID finds the message a report is about in its headers: the
// X-ReputeCast-Message header, else a Message-ID made from our message id.
func ourMessageID(h textproto.MIMEHeader) (uuid.UUID, bool) {
	if h == nil {
		return uuid.Nil, false
	}
	if id, err := uuid.Parse(strings.TrimSpace(h.Get("X-Reputecast-Message"))); err == nil {
		return id, true
	}
	if sm := msgIDRe.FindStringSubmatch(h.Get("Message-Id")); sm != nil {
		if id, err := uuid.Parse(sm[1]); err == nil {
			return id, true
		}
	}
	return uuid.Nil, false
}

// clipString keeps at most n bytes of s, whole runes.
func clipString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	s = s[:n]
	for len(s) > 0 && !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}
