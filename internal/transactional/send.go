package transactional

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"mime"
	"net/http"
	"net/mail"
	"net/netip"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// Sending limits.
const (
	MaxBatch       = 100
	MaxScheduleAge = 72 * time.Hour
	// IdempotencyTTL is how long an idempotency key is remembered.
	IdempotencyTTL = 24 * time.Hour
)

// TransactionalAttachmentInput is a file sent with a message.
type TransactionalAttachmentInput struct {
	Filename    string `json:"filename" minLength:"1" maxLength:"255" example:"INV-2026-0418.pdf"`
	ContentType string `json:"content_type,omitempty" maxLength:"200" example:"application/pdf" doc:"Guessed from the file name when omitted"`
	Content     string `json:"content" doc:"The file, base64-encoded"`
	ContentID   string `json:"content_id,omitempty" maxLength:"200" doc:"Makes it an inline image the HTML references as cid:<content_id>"`
}

// TransactionalSend is one message: a template, a recipient and the
// template's variables.
type TransactionalSend struct {
	Template  string                 `json:"template" minLength:"1" maxLength:"64" example:"password-reset" doc:"The template's ID"`
	To        TransactionalAddress   `json:"to"`
	Data      map[string]any         `json:"data,omitempty" doc:"The template's variables"`
	Cc        []TransactionalAddress `json:"cc,omitempty" maxItems:"20"`
	Bcc       []TransactionalAddress `json:"bcc,omitempty" maxItems:"20"`
	Subject   string                 `json:"subject,omitempty" maxLength:"998" doc:"Overrides the template's subject (Liquid)"`
	Preheader string                 `json:"preheader,omitempty" maxLength:"500" doc:"Overrides the template's preheader (Liquid)"`
	From      string                 `json:"from,omitempty" maxLength:"320" doc:"A verified Transactional sender, as an address or \"Name <address>\"; the template's sender when omitted"`
	ReplyTo   string                 `json:"reply_to,omitempty" maxLength:"320"`
	Tags      []string               `json:"tags,omitempty" maxItems:"10" doc:"Returned with every webhook event of the message"`
	// Metadata is key–value pairs returned with every webhook event.
	Metadata    map[string]string              `json:"metadata,omitempty"`
	Attachments []TransactionalAttachmentInput `json:"attachments,omitempty" maxItems:"20"`
	SendAt      *time.Time                     `json:"send_at,omitempty" doc:"Send later, up to 72 hours ahead"`
	// Language picks a language version (es, pt-BR); without it the linked
	// contact's language is used, then the workspace's fallback.
	Language string `json:"language,omitempty" maxLength:"16" example:"es" doc:"The language version to send; the linked contact's language, then the fallback, when omitted"`
	// Localize writes values that are sentences, dates or prices in the
	// language of the version the send picks (its content language code,
	// "" without language versions); Data has them in English. Only
	// ReputeCast's own senders set it (Earn's member emails), never the API.
	Localize func(language string) map[string]any `json:"-"`
}

// TransactionalSendResult answers an accepted message (202).
type TransactionalSendResult struct {
	ID     string     `json:"id" example:"0192f1c4-7b2e-7c51-9d0a-3f5e8b1a2c44"`
	Status string     `json:"status" enum:"queued,scheduled,not_delivered" doc:"not_delivered: a test-mode key's message to an address that isn't allowed"`
	SendAt *time.Time `json:"send_at,omitempty"`
	Test   bool       `json:"test,omitempty" doc:"Sent with a test-mode key"`
	Detail string     `json:"detail,omitempty"`
}

// caller is who sends: a person (the guide, a resend, a test) or a key.
type caller struct {
	p      *authz.Principal
	key    *store.TxnKeyRow
	ip     *netip.Addr
	source string // api, batch, smtp, resend, guide, test
}

func (s *Service) callerFrom(ctx context.Context, ws *store.Workspace, source string) (caller, error) {
	p := api.PrincipalFrom(ctx)
	c := caller{p: p, ip: audit.RequestFrom(ctx).IP, source: source}
	if p != nil && p.Type == authz.ActorAPIKey {
		k, err := s.Q.TxnKey(ctx, store.TxnKeyParams{ID: p.ID, WorkspaceID: ws.ID})
		if err != nil {
			return c, err
		}
		c.key = &k
	}
	return c, nil
}

func (c caller) keyID() *uuid.UUID {
	if c.key == nil {
		return nil
	}
	id := c.key.ID
	return &id
}

func (c caller) keyName() string {
	if c.key == nil {
		return ""
	}
	return c.key.Name
}

func (c caller) testMode() bool { return c.key != nil && c.key.Mode == "test" }

// prepared is a message ready to be recorded and queued.
type prepared struct {
	id            uuid.UUID
	accepted      time.Time
	rendered      time.Time
	lt            liveTemplate
	raw           bool // SMTP: already rendered
	to            mail.Address
	cc, bcc       []mail.Address
	from          mail.Address
	replyTo       *mail.Address
	data          map[string]any
	masked        map[string]any
	out           rendered
	stored        *rendered
	headers       map[string]string
	files         []payloadFile
	fileMeta      []map[string]any
	size          int
	route         routeChoice
	contactID     *uuid.UUID
	mailbox       string
	test          bool
	testDelivered bool
	reject        string
	rejectDetail  map[string]any
	sendAt        *time.Time
	links         []string
	tags          []string
	metadata      map[string]string
	idemKey       string
	resentFrom    *uuid.UUID
	dropped       []string
	language      LanguageChoice
}

// routeChoice is where a message goes and why.
type routeChoice struct {
	Provider     uuid.UUID  `json:"provider"`
	ProviderName string     `json:"provider_name"`
	Adapter      string     `json:"adapter"`
	Primary      uuid.UUID  `json:"primary"`
	PrimaryName  string     `json:"primary_name"`
	Fallback     *uuid.UUID `json:"fallback,omitempty"`
	FallbackName string     `json:"fallback_name,omitempty"`
	Policy       string     `json:"policy"`
	Rule         string     `json:"rule"`
	Matched      string     `json:"matched,omitempty"`
	Value        string     `json:"value,omitempty"`
	// FailedOver is true when the primary was down and the message went
	// straight to the fallback.
	FailedOver bool `json:"failed_over,omitempty"`
}

// providerLimits are the largest message each adapter's provider accepts,
// attachments included (base64), in bytes.
var providerLimits = map[string]int64{
	"postmark": 10 << 20, "ses": 40 << 20, "smtp": 25 << 20, "postal": 14 << 20, "mailgun": 25 << 20, "sendgrid": 30 << 20,
	"resend": 40 << 20, "sparkpost": 20 << 20, "mailjet": 15 << 20, "brevo": 10 << 20, "azureacs": 10 << 20,
}

func providerLimit(adapter string) int64 {
	if n, ok := providerLimits[adapter]; ok {
		return n
	}
	return 10 << 20
}

func mb(n int64) string { return fmt.Sprintf("%d MB", n>>20) }

// route picks the provider of a transactional message: the Transactional
// rules (conditions by category and mailbox provider), the fallback at
// once when the primary is down, no_healthy_route when nobody can send.
func (s *Service) route(ctx context.Context, q *store.Queries, wsID uuid.UUID, email, mailbox, category string, needExtras bool, size int64) (routeChoice, []store.Provider, error) {
	set, err := routing.Load(ctx, q, wsID)
	if err != nil {
		return routeChoice{}, nil, err
	}
	d, ok := set.Resolve(routing.Recipient{Stream: routing.StreamTransactional, Email: email, MailboxProvider: mailbox, Category: category})
	if !ok {
		return routeChoice{}, nil, problem(http.StatusServiceUnavailable, CodeNoHealthyRoute,
			"Transactional mail has no route yet, so nothing was queued. Choose a provider for the Transactional stream in Sending providers › Routing, then retry with the same idempotency key.")
	}
	provs, err := q.ProvListActive(ctx, wsID)
	if err != nil {
		return routeChoice{}, nil, err
	}
	byID := map[uuid.UUID]store.Provider{}
	for _, p := range provs {
		byID[p.ID] = p
	}
	rc := routeChoice{Primary: d.ProviderID, Fallback: d.Fallback, Policy: string(d.Policy), Rule: d.Rule.Name, Matched: d.Matched, Value: d.MatchedValue}
	if rc.Rule == "" {
		rc.Rule = "Default"
	}
	primary, okP := byID[d.ProviderID]
	if okP {
		rc.PrimaryName = primary.Name
	}
	var fb store.Provider
	okF := false
	if d.Fallback != nil {
		fb, okF = byID[*d.Fallback]
		if okF {
			rc.FallbackName = fb.Name
		}
	}
	// The message must fit every provider it may go through.
	route := []store.Provider{}
	if okP {
		route = append(route, primary)
	}
	if okF {
		route = append(route, fb)
	}
	limit, limitName := int64(0), ""
	for _, p := range route {
		l := providerLimit(p.Adapter)
		if needExtras && !supportsExtras(p.Adapter) {
			l = 0
		}
		if limitName == "" || l < limit {
			limit, limitName = l, p.Name
		}
	}
	if limitName != "" && size > limit {
		if limit == 0 {
			return rc, route, problem(http.StatusUnprocessableEntity, CodeAttachmentsTooLarge,
				fmt.Sprintf("%s's adapter can't send attachments or copies yet, and it's on the Transactional route. Route Transactional to Postmark, Amazon SES, Postal or an SMTP server, or send without attachments.", limitName))
		}
		return rc, route, problem(http.StatusUnprocessableEntity, CodeAttachmentsTooLarge,
			fmt.Sprintf("The message is %s with its attachments; the route's limit is %s (%s). Send smaller files, or links to them.", mbCeil(size), mb(limit), limitName)).
			WithMetaProblem("limitBytes", limit, "provider", limitName, "sizeBytes", size)
	}
	switch {
	case okP && state.Available(primary):
		rc.Provider, rc.ProviderName, rc.Adapter = primary.ID, primary.Name, primary.Adapter
	case okF && state.Available(fb) && d.Policy != routing.PauseAndAsk:
		rc.Provider, rc.ProviderName, rc.Adapter, rc.FailedOver = fb.ID, fb.Name, fb.Adapter, true
	default:
		names := []string{}
		for _, p := range route {
			names = append(names, p.Name)
		}
		what := "Every provider in the Transactional route is down"
		if len(names) > 0 {
			what = joinAnd(names) + " " + plural(len(names), "is", "are") + " down"
		}
		return rc, route, retryAfter(problem(http.StatusServiceUnavailable, CodeNoHealthyRoute,
			what+", so nothing was queued. Retry with the same idempotency key; it is sent once."), 30)
	}
	return rc, route, nil
}

// WithMetaProblem adds structured details to a transactional problem.
func (p *Problem) WithMetaProblem(kv ...any) *Problem {
	if p.Meta == nil {
		p.Meta = map[string]any{}
	}
	for i := 0; i+1 < len(kv); i += 2 {
		if k, ok := kv[i].(string); ok {
			p.Meta[k] = kv[i+1]
		}
	}
	return p
}

func mbCeil(n int64) string {
	return fmt.Sprintf("%.1f MB", float64(n)/float64(1<<20))
}

// supportsExtras reports whether an adapter sends copies and attachments.
func supportsExtras(adapter string) bool {
	switch adapter {
	case "postmark", "ses", "smtp", "postal":
		return true
	}
	return false
}

// parseAddress reads "Name <address>" or a bare address.
func parseAddress(s string) (mail.Address, error) {
	s = strings.TrimSpace(s)
	if a, err := mail.ParseAddress(s); err == nil {
		a.Address = strings.ToLower(a.Address)
		return *a, nil
	}
	return mail.Address{}, errors.New("not an email address")
}

func validEmail(s string) (string, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" || len(s) > 254 || strings.ContainsAny(s, " <>\"(),;:\\[]") {
		return s, false
	}
	a, err := mail.ParseAddress(s)
	if err != nil || a.Address != s {
		return s, false
	}
	at := strings.LastIndexByte(s, '@')
	if at < 1 || at == len(s)-1 || !strings.Contains(s[at+1:], ".") {
		return s, false
	}
	return s, true
}

// resolveSender picks the From: the call's from (a verified Transactional
// sender), else the template's sender, else the workspace default, else
// its first verified Transactional sender.
func (s *Service) resolveSender(ctx context.Context, q *store.Queries, wsID uuid.UUID, from string, templateSender *uuid.UUID) (mail.Address, string, error) {
	if strings.TrimSpace(from) != "" {
		a, err := parseAddress(from)
		if err != nil {
			return mail.Address{}, "", problemInvalid("body.from", "from isn't a valid email address.")
		}
		snd, err := q.TxnSenderByEmail(ctx, store.TxnSenderByEmailParams{WorkspaceID: wsID, Email: a.Address})
		if err != nil || !snd.Transactional || !senderVerified(snd.DomainStatus, snd.Verification) {
			return mail.Address{}, "", problem(http.StatusUnprocessableEntity, CodeSenderNotVerified,
				fmt.Sprintf("%s isn't a verified Transactional sender. Add it on a Transactional domain in Settings › Senders and verify its DNS records, or leave from out to use the template's sender.", a.Address))
		}
		name := a.Name
		if name == "" {
			name = snd.Name
		}
		return mail.Address{Name: name, Address: snd.Email}, snd.ReplyTo, nil
	}
	ids := []*uuid.UUID{templateSender}
	if st, err := s.settingsRow(ctx, q, wsID); err == nil {
		ids = append(ids, st.DefaultSenderID)
	}
	for _, id := range ids {
		if id == nil {
			continue
		}
		if snd, err := q.TxnSenderByID(ctx, store.TxnSenderByIDParams{WorkspaceID: wsID, ID: *id}); err == nil && snd.Transactional && senderVerified(snd.DomainStatus, snd.Verification) {
			return mail.Address{Name: snd.Name, Address: snd.Email}, snd.ReplyTo, nil
		}
	}
	senders, err := s.transactionalSenders(ctx, q, wsID)
	if err != nil {
		return mail.Address{}, "", err
	}
	for _, snd := range senders {
		if snd.Verified {
			return mail.Address{Name: snd.Name, Address: snd.Email}, snd.ReplyTo, nil
		}
	}
	return mail.Address{}, "", problem(http.StatusUnprocessableEntity, CodeSenderNotVerified,
		"There is no verified sender on a Transactional domain yet, so nothing was sent. Add one, such as no-reply@notify.yourdomain.com, in Settings › Senders and verify its DNS records.")
}

// testAllowed reports whether a test-mode key may deliver to an address:
// the workspace's members and the key's allowed test addresses (patterns
// such as qa+*@acme.io).
func (s *Service) testAllowed(ctx context.Context, q *store.Queries, wsID uuid.UUID, key *store.TxnKeyRow, email string) bool {
	for _, pat := range key.TestAddresses {
		pat = strings.ToLower(strings.TrimSpace(pat))
		if pat == email {
			return true
		}
		if ok, err := path.Match(pat, email); err == nil && ok {
			return true
		}
	}
	members, err := q.TxnMemberEmails(ctx, wsID)
	if err != nil {
		return false
	}
	return slices.Contains(members, email)
}

func cleanTags(in []string) []string {
	out := []string{}
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t != "" && len(t) <= 100 && !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out
}

// prepare validates a send and renders it; nothing is written.
func (s *Service) prepare(ctx context.Context, ws *store.Workspace, c caller, in TransactionalSend, accepted time.Time) (*prepared, error) {
	q := s.Q
	lt, err := s.live(ctx, q, ws.ID, in.Template)
	if err != nil {
		return nil, err
	}
	if c.key != nil && len(c.key.Templates) > 0 && !slices.Contains(c.key.Templates, lt.Key) {
		return nil, problem(http.StatusForbidden, CodeTemplateNotAllowed,
			fmt.Sprintf("The key “%s” can only send %s. Ask a member with apikeys:manage to add %s to its templates, or use another key.", c.key.Name, joinAnd(c.key.Templates), lt.Key))
	}
	if c.key != nil && len(c.key.Categories) > 0 && (lt.CategoryID == nil || !slices.Contains(c.key.Categories, *lt.CategoryID)) {
		return nil, problem(http.StatusForbidden, CodeTemplateNotAllowed,
			fmt.Sprintf("The key “%s” is limited to other categories, so it can't send %s.", c.key.Name, lt.Key))
	}
	email, ok := validEmail(in.To.Email)
	if !ok {
		return nil, problem(http.StatusUnprocessableEntity, CodeInvalidRecipient,
			fmt.Sprintf("“%s” isn't a valid email address, so nothing was sent. Validate addresses in your app before calling.", clip(in.To.Email, 100)))
	}
	data := in.Data
	if data == nil {
		data = map[string]any{}
	}
	if missing := missingRequired(lt.Content.Variables, data); len(missing) > 0 {
		return nil, missingProblem(lt.Key, lt.Version, missing)
	}
	p := &prepared{id: newID(), accepted: accepted, lt: lt, to: mail.Address{Name: strings.TrimSpace(in.To.Name), Address: email}, data: data,
		masked: maskData(lt.Content.Variables, data), tags: cleanTags(in.Tags), metadata: in.Metadata}
	if p.metadata == nil {
		p.metadata = map[string]string{}
	}
	if len(p.metadata) > 20 {
		return nil, problemInvalid("body.metadata", "Send at most 20 metadata pairs.")
	}
	for k, v := range p.metadata {
		if len(k) > 100 || len(v) > 500 {
			return nil, problemInvalid("body.metadata", "Metadata keys are at most 100 characters and values 500.")
		}
	}
	for i, a := range in.Cc {
		e, ok := validEmail(a.Email)
		if !ok {
			return nil, problem(http.StatusUnprocessableEntity, CodeInvalidRecipient, fmt.Sprintf("cc[%d] “%s” isn't a valid email address.", i, clip(a.Email, 100)))
		}
		p.cc = append(p.cc, mail.Address{Name: a.Name, Address: e})
	}
	for i, a := range in.Bcc {
		e, ok := validEmail(a.Email)
		if !ok {
			return nil, problem(http.StatusUnprocessableEntity, CodeInvalidRecipient, fmt.Sprintf("bcc[%d] “%s” isn't a valid email address.", i, clip(a.Email, 100)))
		}
		p.bcc = append(p.bcc, mail.Address{Name: a.Name, Address: e})
	}
	if in.SendAt != nil {
		at := *in.SendAt
		if at.After(accepted.Add(MaxScheduleAge)) {
			return nil, problemInvalid("body.send_at", "send_at can be at most 72 hours ahead. Schedule later sends in your app.")
		}
		if at.After(accepted.Add(time.Second)) {
			p.sendAt = &at
		}
	}
	from, defaultReply, err := s.resolveSender(ctx, q, ws.ID, in.From, lt.SenderID)
	if err != nil {
		return nil, err
	}
	p.from = from
	reply := strings.TrimSpace(in.ReplyTo)
	if reply == "" {
		reply = lt.ReplyTo
	}
	if reply == "" {
		reply = defaultReply
	}
	if reply != "" {
		a, err := parseAddress(reply)
		if err != nil {
			return nil, problemInvalid("body.reply_to", "reply_to isn't a valid email address.")
		}
		p.replyTo = &a
	}
	// Attachments.
	size := 0
	for i, a := range in.Attachments {
		b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(a.Content))
		if err != nil {
			return nil, problemInvalid(fmt.Sprintf("body.attachments[%d].content", i), fmt.Sprintf("%s isn't base64. Encode the file with standard base64.", a.Filename))
		}
		ct := strings.TrimSpace(a.ContentType)
		if ct == "" {
			ct = mime.TypeByExtension(strings.ToLower(path.Ext(a.Filename)))
		}
		if ct == "" {
			ct = "application/octet-stream"
		}
		name := path.Base(strings.ReplaceAll(a.Filename, "\\", "/"))
		p.files = append(p.files, payloadFile{Filename: name, ContentType: ct, ContentID: strings.Trim(a.ContentID, "<>"), Content: b})
		p.fileMeta = append(p.fileMeta, map[string]any{"filename": name, "content_type": ct, "size": len(b), "content_id": strings.Trim(a.ContentID, "<>")})
		size += base64.StdEncoding.EncodedLen(len(b))
	}
	// Contact linking: a recipient who matches a contact shows in its
	// timeline; nobody is ever added as a contact.
	p.mailbox = s.mailboxFor(ctx, email)
	if st, err := s.settingsRow(ctx, q, ws.ID); err == nil && st.LinkContacts {
		if ct, err := q.TxnContactByEmail(ctx, store.TxnContactByEmailParams{WorkspaceID: ws.ID, Email: email}); err == nil {
			id := ct.ID
			p.contactID = &id
			if ct.MailboxProvider != "" && ct.MailboxProvider != routing.Other {
				p.mailbox = ct.MailboxProvider
			}
		}
	}
	// The language version the recipient gets (internal/languages).
	if s.Translations != nil {
		if p.language, err = s.Translations.Choose(ctx, q, ws, lt.ID, lt.Version, in.Language, p.contactID); err != nil {
			return nil, err
		}
	}
	// Values written in the language of the version the reader gets.
	if in.Localize != nil {
		data = maps.Clone(data)
		maps.Copy(data, in.Localize(p.language.Language))
		p.data, p.masked = data, maskData(lt.Content.Variables, data)
	}
	// Render: what the provider gets, and what the log keeps.
	links := s.messageLinks(ws, p.id, p.contactID, lt.CategoryKind == KindOptional)
	cnt := lt.Content.withLanguage(p.language.Content)
	if in.Subject != "" {
		cnt.Subject = in.Subject
	}
	if in.Preheader != "" {
		cnt.Preheader = in.Preheader
	}
	to := recipient{Email: email, Name: p.to.Name}
	p.out, err = render(cnt, data, to, links)
	if err != nil {
		return nil, problem(http.StatusUnprocessableEntity, CodeTemplateDoesntRun, fmt.Sprintf("%s v%d doesn't render with this data: %s", lt.Key, lt.Version, err))
	}
	masked, err := render(cnt, p.masked, to, links)
	if err == nil {
		p.stored = &masked
	}
	p.rendered = s.now()
	if p.rendered.Before(accepted) {
		p.rendered = accepted
	}
	size += len(p.out.HTML) + len(p.out.Text)
	p.size = size
	categoryID := ""
	if lt.CategoryID != nil {
		categoryID = lt.CategoryID.String()
	}
	rc, _, err := s.route(ctx, q, ws.ID, email, p.mailbox, categoryID, len(p.files) > 0 || len(p.cc) > 0 || len(p.bcc) > 0, int64(size))
	if err != nil {
		return nil, err
	}
	p.route = rc
	// Tracking, through the workspace's link tracking domain.
	if lt.TrackOpens || lt.TrackClicks {
		domain := s.trackingDomain(ctx, q, ws)
		contact := uuid.Nil
		if p.contactID != nil {
			contact = *p.contactID
		}
		open := ""
		if lt.TrackOpens {
			open = trackingURL(s.URLs.Open(s.Links.Sign(linktoken.Claims{Kind: linktoken.Open, WorkspaceID: ws.ID, MessageID: p.id, ContactID: contact})), domain)
		}
		p.out.HTML, p.links = trackLinks(p.out.HTML, lt.TrackOpens, lt.TrackClicks, sensitiveValues(lt.Content.Variables, data),
			func(pos int) string {
				return trackingURL(s.URLs.Click(s.clickToken(ws.ID, p.id, contact, pos)), domain)
			}, open)
	}
	p.headers = map[string]string{"Feedback-ID": clip(lt.Key, 40) + ":" + ws.Slug + ":transactional:reputecast"}
	if lt.CategoryKind == KindOptional {
		p.headers["List-Unsubscribe"] = "<" + links["unsubscribe_url"] + ">"
		p.headers["List-Unsubscribe-Post"] = "List-Unsubscribe=One-Click"
	}
	return p, nil
}

// messageLinks are the links ReputeCast fills in for a message: the
// category's unsubscribe link and the preference center.
func (s *Service) messageLinks(ws *store.Workspace, msg uuid.UUID, contact *uuid.UUID, optional bool) map[string]string {
	if s.Links == nil {
		return map[string]string{"unsubscribe_url": "#", "preferences_url": "#"}
	}
	cid := uuid.Nil
	if contact != nil {
		cid = *contact
	}
	out := map[string]string{
		"preferences_url": s.URLs.Preferences(ws.Slug, s.Links.Sign(linktoken.Claims{Kind: linktoken.Preferences, WorkspaceID: ws.ID, MessageID: msg, ContactID: cid})),
	}
	out["unsubscribe_url"] = s.URLs.Unsubscribe(ws.Slug, s.Links.Sign(linktoken.Claims{Kind: linktoken.Unsubscribe, WorkspaceID: ws.ID, MessageID: msg, ContactID: cid}))
	if !optional {
		// Essential categories can't be turned off: their emails carry no
		// unsubscribe link, only the preference center.
		out["unsubscribe_url"] = out["preferences_url"]
	}
	return out
}

// trackingDomain is the workspace's link tracking domain, once its DNS
// record is verified; until then tracking goes through the public URL.
func (s *Service) trackingDomain(ctx context.Context, q *store.Queries, ws *store.Workspace) string {
	var sd struct {
		TrackingDomain string `json:"tracking_domain"`
	}
	_ = json.Unmarshal(ws.SendingDefaults, &sd)
	d := strings.ToLower(strings.TrimSpace(sd.TrackingDomain))
	if d == "" {
		return ""
	}
	if ok, err := q.TxnTrackingDomainVerified(ctx, store.TxnTrackingDomainVerifiedParams{WorkspaceID: ws.ID, Domain: d}); err != nil || !ok {
		return ""
	}
	return d
}

// screen applies what stops a message before sending: an all-mail
// suppression, the recipient's opt-out of an Optional category, and a
// test-mode key's allowed addresses. Suppressed copies are left out.
func (s *Service) screen(ctx context.Context, q *store.Queries, ws *store.Workspace, c caller, p *prepared) error {
	email := p.to.Address
	if sup, err := q.TxnSuppression(ctx, store.TxnSuppressionParams{WorkspaceID: ws.ID, Email: email}); err == nil {
		p.reject = "suppressed"
		p.rejectDetail = map[string]any{"reason": sup.Reason, "since": sup.CreatedAt, "source": suppressionSource(sup.Source, sup.ProviderName), "code": sup.OriginalCode, "detail": sup.Detail}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if p.reject == "" && p.lt.CategoryKind == KindOptional && p.lt.CategoryID != nil {
		if oo, err := q.TxnOptOut(ctx, store.TxnOptOutParams{WorkspaceID: ws.ID, Email: email, CategoryID: *p.lt.CategoryID}); err == nil {
			p.reject = "opted_out"
			p.rejectDetail = map[string]any{"category": oo.CategoryName, "since": oo.OptedOutAt, "source": oo.Source}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}
	keep := func(list []mail.Address) []mail.Address {
		out := []mail.Address{}
		for _, a := range list {
			if _, err := q.TxnSuppression(ctx, store.TxnSuppressionParams{WorkspaceID: ws.ID, Email: a.Address}); err == nil {
				p.dropped = append(p.dropped, a.Address)
				continue
			}
			out = append(out, a)
		}
		return out
	}
	p.cc, p.bcc = keep(p.cc), keep(p.bcc)
	if c.testMode() {
		p.test = true
		p.testDelivered = p.reject == "" && s.testAllowed(ctx, q, ws.ID, c.key, email)
		if !p.testDelivered && p.reject == "" {
			p.reject = "test_not_allowed"
			p.rejectDetail = map[string]any{"allowed": c.key.TestAddresses}
		}
	}
	return nil
}

func suppressionSource(source, provider string) string {
	if provider != "" {
		return provider
	}
	switch source {
	case "webhook", "bounce_mailbox", "backfill":
		return "a provider"
	case "manual", "":
		return "added by hand"
	}
	return source
}

func reasonLabel(r string) string {
	switch r {
	case "hard_bounce":
		return "hard bounce"
	case "spam_trap":
		return "spam trap"
	case "imported":
		return "imported from a provider"
	case "invalid":
		return "invalid address"
	case "manual":
		return "added by hand"
	}
	return r
}

// suppressedProblem answers a call to a suppressed or opted-out address.
func suppressedProblem(p *prepared, loc *time.Location) *Problem {
	var detail string
	switch p.reject {
	case "opted_out":
		since, _ := p.rejectDetail["since"].(time.Time)
		cat, _ := p.rejectDetail["category"].(string)
		detail = fmt.Sprintf("%s turned off %s on %s, so nothing was sent. Essential emails, such as account and billing ones, still reach them.", p.to.Address, cat, since.In(loc).Format("2 Jan 2006"))
	default:
		since, _ := p.rejectDetail["since"].(time.Time)
		reason, _ := p.rejectDetail["reason"].(string)
		detail = fmt.Sprintf("%s is on the suppression list (%s on %s), so nothing was sent. Transactional mail skips unsubscribes, but never goes to an address that bounced, complained or is a spam trap.",
			p.to.Address, reasonLabel(reason), since.In(loc).Format("2 Jan 2006"))
	}
	pr := problem(http.StatusUnprocessableEntity, CodeRecipientSuppressed, detail)
	meta := []any{"messageId", p.id.String(), "reason", p.reject}
	for k, v := range p.rejectDetail {
		meta = append(meta, k, v)
	}
	pr = pr.WithMetaProblem(meta...)
	if p.reject == "suppressed" {
		pr.Meta["reason"] = p.rejectDetail["reason"]
		pr.Meta["rejected"] = "suppressed"
	}
	return pr
}

// record writes a prepared message: its messages row, its log entry, its
// content (for the retention period) and its sealed delivery payload.
func (s *Service) record(ctx context.Context, tx pgx.Tx, ws *store.Workspace, c caller, p *prepared) error {
	q := store.New(tx)
	st, err := s.settingsRow(ctx, q, ws.ID)
	if err != nil {
		return err
	}
	status, kind := "queued", c.source
	switch c.source {
	case "batch", "resend", "guide", "test":
		kind = "api"
	}
	var next *time.Time
	if p.sendAt != nil {
		next = p.sendAt
	}
	errText := ""
	var provider *uuid.UUID
	if p.reject != "" {
		status = "rejected"
		if p.reject == "test_not_allowed" {
			status, errText = "cancelled", "Test mode: not a member or an allowed test address, so not delivered."
		}
	} else {
		id := p.route.Provider
		provider = &id
	}
	note := ""
	if p.route.FailedOver {
		note = "failover"
	}
	if err := q.TxnInsertMessage(ctx, store.TxnInsertMessageParams{ID: p.id, WorkspaceID: ws.ID, ContactID: p.contactID, Email: p.to.Address, ProviderID: provider,
		MailboxProvider: p.mailbox, Status: status, QueuedAt: p.accepted, NextAttemptAt: next, Kind: kind, Error: errText, RouteNote: note}); err != nil {
		return err
	}
	if s.Translations != nil && p.language.Language != "" {
		if err := s.Translations.Record(ctx, q, p.id, p.language); err != nil {
			return err
		}
	}
	var tplID *uuid.UUID
	var tplVersion *int32
	if !p.raw {
		id := p.lt.ID
		v := int32(p.lt.Version)
		tplID, tplVersion = &id, &v
	}
	var expires *time.Time
	if st.ContentRetentionDays > 0 && p.reject != "suppressed" && p.reject != "opted_out" {
		e := p.accepted.Add(time.Duration(st.ContentRetentionDays) * 24 * time.Hour)
		expires = &e
	}
	var since *time.Time
	if provider != nil {
		t := p.accepted
		since = &t
	}
	rendered := p.rendered
	detail := p.rejectDetail
	if len(p.dropped) > 0 {
		if detail == nil {
			detail = map[string]any{}
		}
		detail["dropped"] = p.dropped
	}
	if detail == nil {
		detail = map[string]any{}
	}
	files := p.fileMeta
	if files == nil {
		files = []map[string]any{}
	}
	if err := q.TxnInsertMessageLog(ctx, store.TxnInsertMessageLogParams{
		MessageID: p.id, WorkspaceID: ws.ID, Source: c.source, TemplateID: tplID, TemplateKey: p.lt.Key, TemplateName: p.lt.Name, TemplateVersion: tplVersion,
		CategoryID: p.lt.CategoryID, CategoryName: p.lt.CategoryName, CategoryKind: p.lt.CategoryKind, KeyID: c.keyID(), KeyName: c.keyName(), UserID: userIfPerson(c.p),
		Ip: c.ip, IdempotencyKey: p.idemKey, Subject: p.out.Subject, FromEmail: p.from.Address, FromName: p.from.Name, ReplyTo: addrString(p.replyTo),
		ToName: p.to.Name, Cc: addrList(p.cc), Bcc: addrList(p.bcc), Tags: p.tags, Metadata: mustJSON(p.metadata), Data: mustJSON(p.masked),
		DataKeys: dataKeys(p.data), Sensitive: sensitiveNames(p.lt.Content.Variables), Attachments: mustJSON(files), SizeBytes: int32(p.size),
		Test: p.test, TestDelivered: p.testDelivered, SendAt: p.sendAt, RejectReason: p.reject, RejectDetail: mustJSON(detail),
		TrackOpens: p.lt.TrackOpens, TrackClicks: p.lt.TrackClicks, Links: nonNilStrings(p.links), Route: mustJSON(p.route), ProviderSince: since,
		AcceptedAt: p.accepted, RenderedAt: &rendered, ContentExpiresAt: expires, ResentFrom: p.resentFrom,
	}); err != nil {
		return err
	}
	if provider != nil && p.route.FailedOver {
		// The primary was down when the call came: the message went to the
		// fallback at once, and counts in the failover incident.
		primary := p.route.Primary
		if err := q.TxnMarkFailedOver(ctx, store.TxnMarkFailedOverParams{MessageID: p.id, At: p.accepted, FromProvider: &primary}); err != nil {
			return err
		}
	}
	if expires != nil && p.stored != nil {
		hdr := map[string]string{"From": p.from.String(), "To": p.to.String(), "Subject": p.stored.Subject, "Date": p.accepted.UTC().Format(time.RFC1123Z),
			"Message-ID": "<" + p.id.String() + "@" + domainOf(p.from.Address) + ">"}
		if p.replyTo != nil {
			hdr["Reply-To"] = p.replyTo.String()
		}
		if len(p.cc) > 0 {
			hdr["Cc"] = strings.Join(addrList(p.cc), ", ")
		}
		for k, v := range p.headers {
			hdr[k] = v
		}
		if err := q.TxnInsertContent(ctx, store.TxnInsertContentParams{MessageID: p.id, Html: p.stored.HTML, Text: p.stored.Text, Headers: mustJSON(hdr), ExpiresAt: *expires}); err != nil {
			return err
		}
	}
	if provider == nil {
		return nil
	}
	tags := map[string]string{"transactional": p.lt.Key, "workspace": ws.Slug}
	sealed, err := s.seal(p.id, payload{From: p.from, ReplyTo: p.replyTo, To: p.to, Cc: p.cc, Bcc: p.bcc, Subject: p.out.Subject, HTML: p.out.HTML, Text: p.out.Text,
		Headers: p.headers, Tags: tags, Attachments: p.files})
	if err != nil {
		return err
	}
	if err := q.TxnInsertOutbox(ctx, store.TxnInsertOutboxParams{MessageID: p.id, Sealed: sealed, CreatedAt: p.accepted}); err != nil {
		return err
	}
	if s.Outbox != nil {
		return s.Outbox(ctx, tx)
	}
	return nil
}

func userIfPerson(p *authz.Principal) *uuid.UUID {
	if p == nil || p.Type != authz.ActorUser {
		return nil
	}
	id := p.UserID
	return &id
}

func addrString(a *mail.Address) string {
	if a == nil {
		return ""
	}
	return a.String()
}

func addrList(as []mail.Address) []string {
	out := make([]string, 0, len(as))
	for _, a := range as {
		out = append(out, a.Address)
	}
	return out
}

func domainOf(email string) string {
	if i := strings.LastIndexByte(email, '@'); i >= 0 {
		return email[i+1:]
	}
	return "reputecast.local"
}

// requestHash fingerprints a call for its idempotency key.
func requestHash(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:16])
}

// idempotent runs fn once per idempotency key: a retry within 24 hours
// with the same body gets the first answer and sends nothing; the same key
// with another body is refused. fn returns the status and body to remember.
func (s *Service) idempotent(ctx context.Context, tx pgx.Tx, wsID uuid.UUID, key, hash string, fn func() (int, any, error)) (int, json.RawMessage, bool, error) {
	q := store.New(tx)
	if key == "" {
		status, body, err := fn()
		if err != nil {
			return 0, nil, false, err
		}
		return status, mustJSON(body), false, nil
	}
	if _, err := q.TxnIdemInsert(ctx, store.TxnIdemInsertParams{WorkspaceID: wsID, Key: key, RequestHash: hash, CreatedAt: s.now()}); errors.Is(err, pgx.ErrNoRows) {
		prev, err := q.TxnIdemGet(ctx, store.TxnIdemGetParams{WorkspaceID: wsID, Key: key})
		if err != nil {
			return 0, nil, false, err
		}
		if prev.CreatedAt.Before(s.now().Add(-IdempotencyTTL)) {
			// An expired key from yesterday: start over.
			if _, err := tx.Exec(ctx, `UPDATE transactional_idempotency SET request_hash = $3, created_at = $4, status = 0, response = '{}' WHERE workspace_id = $1 AND key = $2`,
				wsID, key, hash, s.now()); err != nil {
				return 0, nil, false, err
			}
		} else {
			if prev.RequestHash != hash {
				return 0, nil, false, problem(http.StatusConflict, CodeIdempotencyReused,
					fmt.Sprintf("The idempotency key %s was used within the last 24 hours for a different request. Use a new key for a new message.", clip(key, 80)))
			}
			return int(prev.Status), prev.Response, true, nil
		}
	} else if err != nil {
		return 0, nil, false, err
	}
	status, body, err := fn()
	if err != nil {
		return 0, nil, false, err
	}
	raw := mustJSON(body)
	if err := q.TxnIdemSave(ctx, store.TxnIdemSaveParams{WorkspaceID: wsID, Key: key, Status: int32(status), Response: raw}); err != nil {
		return 0, nil, false, err
	}
	return status, raw, false, nil
}

// logRejectedCall keeps a refused call for the first-send guide and the
// breaking-change check (no message exists for it).
func (s *Service) logRejectedCall(ctx context.Context, ws *store.Workspace, c caller, in TransactionalSend, idem string, err error) {
	var pr *Problem
	if !errors.As(err, &pr) {
		return
	}
	if pr.Code == CodeRateLimited || pr.Code == CodeIdempotencyReused {
		return
	}
	_ = s.Q.TxnInsertRejectedCall(ctx, store.TxnInsertRejectedCallParams{ID: newID(), WorkspaceID: ws.ID, KeyID: c.keyID(), UserID: userIfPerson(c.p), At: s.now(),
		Status: int32(pr.Status), Code: pr.Code, Detail: clip(pr.Detail, 500), TemplateKey: clip(strings.ToLower(in.Template), 64), Recipient: clip(strings.ToLower(in.To.Email), 254),
		IdempotencyKey: clip(idem, 255), Ip: c.ip})
}

// rateLimit applies the caller's send rate (per key; people share 50 per
// second per workspace member).
func (s *Service) rateLimit(c caller, n int) error {
	id, per := uuid.Nil, 50
	switch {
	case c.key != nil:
		id, per = c.key.ID, int(c.key.RatePerSecond)
	case c.p != nil:
		id = c.p.UserID
	}
	if ok, wait := s.limiter.allow(id, per, n, s.now()); !ok {
		secs := retrySeconds(wait)
		return retryAfter(problem(http.StatusTooManyRequests, CodeRateLimited,
			fmt.Sprintf("This key sends at most %d messages per second. Wait %d %s (Retry-After) and send again with the same idempotency key.", per, secs, plural(secs, "second", "seconds"))), secs)
	}
	return nil
}

// sendOne prepares, screens and records one message inside tx. It returns
// the API's answer: 202 with the result, or 422 for a recipient who is
// suppressed or opted out (the message is still in the log).
func (s *Service) sendOne(ctx context.Context, tx pgx.Tx, ws *store.Workspace, c caller, p *prepared) (int, any, error) {
	q := store.New(tx)
	if err := s.screen(ctx, q, ws, c, p); err != nil {
		return 0, nil, err
	}
	if err := s.record(ctx, tx, ws, c, p); err != nil {
		return 0, nil, err
	}
	switch p.reject {
	case "suppressed", "opted_out":
		if err := s.emitMessage(ctx, tx, ws, p.id, "message.rejected"); err != nil {
			return 0, nil, err
		}
		return http.StatusUnprocessableEntity, suppressedProblem(p, location(ws)), nil
	case "test_not_allowed":
		return http.StatusAccepted, TransactionalSendResult{ID: p.id.String(), Status: "not_delivered", Test: true,
			Detail: p.to.Address + " isn't a member or an allowed test address, so the message was rendered and logged but not delivered."}, nil
	}
	res := TransactionalSendResult{ID: p.id.String(), Status: "queued", Test: p.test}
	if p.sendAt != nil {
		res.Status, res.SendAt = "scheduled", p.sendAt
	}
	return http.StatusAccepted, res, nil
}

// Send sends one message (POST …/transactional/messages).
func (s *Service) Send(ctx context.Context, ws *store.Workspace, in TransactionalSend, idem string, source string) (int, json.RawMessage, error) {
	accepted := s.now()
	c, err := s.callerFrom(ctx, ws, source)
	if err != nil {
		return 0, nil, err
	}
	if err := s.rateLimit(c, 1); err != nil {
		return 0, nil, err
	}
	hash := requestHash(in)
	// A retry of an accepted call answers from the key before anything is
	// checked again (the template may have changed since).
	if idem != "" {
		if prev, err := s.Q.TxnIdemGet(ctx, store.TxnIdemGetParams{WorkspaceID: ws.ID, Key: idem}); err == nil && prev.Status != 0 && prev.CreatedAt.After(s.now().Add(-IdempotencyTTL)) {
			if prev.RequestHash != hash {
				return 0, nil, problem(http.StatusConflict, CodeIdempotencyReused,
					fmt.Sprintf("The idempotency key %s was used within the last 24 hours for a different request. Use a new key for a new message.", clip(idem, 80)))
			}
			return int(prev.Status), prev.Response, nil
		}
	}
	p, err := s.prepare(ctx, ws, c, in, accepted)
	if err != nil {
		s.logRejectedCall(ctx, ws, c, in, idem, err)
		return 0, nil, err
	}
	p.idemKey = idem
	var status int
	var body json.RawMessage
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var err error
		status, body, _, err = s.idempotent(ctx, tx, ws.ID, idem, hash, func() (int, any, error) {
			return s.sendOne(ctx, tx, ws, c, p)
		})
		return err
	})
	return status, body, err
}

// TransactionalBatch sends up to 100 messages, each with its own recipient
// and data.
type TransactionalBatch struct {
	Messages []TransactionalSend `json:"messages" minItems:"1" maxItems:"100"`
}

// TransactionalBatchItem is one message's answer in a batch.
type TransactionalBatchItem struct {
	Index  int        `json:"index"`
	ID     string     `json:"id,omitempty"`
	Status string     `json:"status" enum:"queued,scheduled,not_delivered,error"`
	SendAt *time.Time `json:"send_at,omitempty"`
	Test   bool       `json:"test,omitempty"`
	// Error is the problem that stopped this message (same codes as one send).
	Error *Problem `json:"error,omitempty"`
}

// TransactionalBatchResult answers a batch: each message gets its own answer.
type TransactionalBatchResult struct {
	Results  []TransactionalBatchItem `json:"results"`
	Accepted int                      `json:"accepted"`
	Failed   int                      `json:"failed"`
}

// SendBatch sends a batch.
func (s *Service) SendBatch(ctx context.Context, ws *store.Workspace, in TransactionalBatch, idem string) (json.RawMessage, error) {
	if len(in.Messages) > MaxBatch {
		return nil, problemInvalid("body.messages", fmt.Sprintf("A batch carries at most %d messages; split it.", MaxBatch))
	}
	accepted := s.now()
	c, err := s.callerFrom(ctx, ws, "batch")
	if err != nil {
		return nil, err
	}
	if err := s.rateLimit(c, len(in.Messages)); err != nil {
		return nil, err
	}
	hash := requestHash(in)
	if idem != "" {
		if prev, err := s.Q.TxnIdemGet(ctx, store.TxnIdemGetParams{WorkspaceID: ws.ID, Key: idem}); err == nil && prev.Status != 0 && prev.CreatedAt.After(s.now().Add(-IdempotencyTTL)) {
			if prev.RequestHash != hash {
				return nil, problem(http.StatusConflict, CodeIdempotencyReused,
					fmt.Sprintf("The idempotency key %s was used within the last 24 hours for a different request. Use a new key for a new batch.", clip(idem, 80)))
			}
			return prev.Response, nil
		}
	}
	items := make([]TransactionalBatchItem, len(in.Messages))
	preps := make([]*prepared, len(in.Messages))
	for i, m := range in.Messages {
		items[i] = TransactionalBatchItem{Index: i, Status: "error"}
		p, err := s.prepare(ctx, ws, c, m, accepted)
		if err != nil {
			s.logRejectedCall(ctx, ws, c, m, idem, err)
			var pr *Problem
			if !errors.As(err, &pr) {
				var ap *api.Problem
				if errors.As(err, &ap) {
					pr = &Problem{Problem: *ap}
				} else {
					return nil, err
				}
			}
			items[i].Error = pr
			continue
		}
		if idem != "" {
			p.idemKey = fmt.Sprintf("%s#%d", idem, i)
		}
		preps[i] = p
	}
	var body json.RawMessage
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var err error
		_, body, _, err = s.idempotent(ctx, tx, ws.ID, idem, hash, func() (int, any, error) {
			out := TransactionalBatchResult{Results: items}
			for i, p := range preps {
				if p == nil {
					out.Failed++
					continue
				}
				status, res, err := s.sendOne(ctx, tx, ws, c, p)
				if err != nil {
					return 0, nil, err
				}
				switch r := res.(type) {
				case TransactionalSendResult:
					out.Results[i] = TransactionalBatchItem{Index: i, ID: r.ID, Status: r.Status, SendAt: r.SendAt, Test: r.Test}
					out.Accepted++
				case *Problem:
					out.Results[i] = TransactionalBatchItem{Index: i, ID: p.id.String(), Status: "error", Error: r}
					out.Failed++
				}
				_ = status
			}
			return http.StatusOK, out, nil
		})
		return err
	})
	return body, err
}
