// Package tracking serves the open pixel and the click redirect that go in
// every tracked email, and records opens and clicks honestly: human opens
// separated from machine opens (Apple Mail Privacy Protection, image proxy
// prefetches, security scanners) and bot clicks kept out of the counts.
//
//	GET /t/o/{token}   1×1 transparent GIF, never cached; records the open
//	GET /t/c/{token}   302 to the email's link; records the click
//
// Tokens are signed (internal/linktoken) and name the message, the contact
// and, for clicks, the link's position in the campaign's links: the
// redirect only ever goes to a link of that message (no open redirect).
// Writes are group-committed in the background, so the pixel and the
// redirect stay fast under load. The unsubscribe page and RFC 8058
// one-click POST live with the public pages.
package tracking

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
)

// pixelGIF is a 1×1 transparent GIF.
var pixelGIF = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xff, 0xff, 0xff, 0x21, 0xf9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00, 0x00,
	0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44, 0x01, 0x00, 0x3b,
}

// Service records opens and clicks.
type Service struct {
	Pool        *pgxpool.Pool
	Q           *store.Queries
	Links       *linktoken.Signer
	Log         *slog.Logger
	Automations *automations.Service
	// Transactional resolves the tracked links of transactional messages
	// and fires their message.opened and message.clicked webhook events.
	Transactional TransactionalTracking
	// Placement resolves the links of placement test copies, whose opens
	// and clicks are never counted (internal/placement).
	Placement PlacementLinks
	// Decorators may add parameters to a click's destination: links to a
	// connected store get UTM parameters and the click token
	// (internal/commerce).
	Decorators []LinkDecorator
	// MessageVars are an automation email's own link variables, from what
	// its trigger passed ("checkout_url": the store's recovery link).
	MessageVars func(ctx context.Context, msgCtx map[string]string) map[string]string
	Now         func() time.Time
	// Clicked, when set, hears of each person's click (bot clicks never
	// reach it) in the transaction that records it: Earn counts clicks on
	// upgrade links per campaign.
	Clicked func(ctx context.Context, q *store.Queries, c HumanClick) error
	// Decorate, when set, may add to a click's destination before the
	// redirect (Earn's click token on the publisher's own domains); "" keeps
	// the destination as it is. It runs after Decorators, and adds nothing
	// to a link that already carries rc_click.
	Decorate func(ctx context.Context, c ClickTarget) string
	// Recipients add the per-reader variables of blocks (Grow's
	// referral.* links) to personalized links; nil adds none.
	Recipients func() []liquid.RecipientData

	once  sync.Once
	in    chan *pending
	mu    sync.Mutex
	links map[uuid.UUID]linkSet
	burst map[uuid.UUID][]clickMark
}

// HumanClick is a person's click, as Clicked hears of it.
type HumanClick struct {
	WorkspaceID uuid.UUID
	MessageID   uuid.UUID
	CampaignID  *uuid.UUID
	ContactID   *uuid.UUID
	Stream      string
	// URL is the link as the email has it (it may hold Liquid).
	URL string
	At  time.Time
}

// ClickTarget is a click about to be redirected, as Decorate sees it.
type ClickTarget struct {
	WorkspaceID uuid.UUID
	MessageID   uuid.UUID
	ContactID   uuid.UUID
	URL         string
	At          time.Time
}

type linkSet struct {
	urls  map[int]string
	until time.Time
}

type clickMark struct {
	link int
	at   time.Time
}

type pending struct {
	ev   sending.Engagement
	done chan error
}

var _ sending.Recorder = (*Service)(nil)

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

// TransactionalTracking is what the tracking endpoints need from
// transactional email: each message keeps its own tracked links (their
// targets differ per recipient), and engagement fires webhooks.
type TransactionalTracking interface {
	TrackedLink(ctx context.Context, q *store.Queries, messageID uuid.UUID, position int) (string, bool)
	Engagement(ctx context.Context, tx pgx.Tx, messageID uuid.UUID, click bool, at time.Time) error
}

// LinkDecorator may rewrite a click's destination before the redirect.
// It runs on every click, so it must answer fast (cache what it reads).
type LinkDecorator interface {
	Decorate(ctx context.Context, c ClickedLink) string
}

// ClickedLink is a click about to be redirected: the message's email and
// the destination.
type ClickedLink struct {
	WorkspaceID  uuid.UUID
	MessageID    uuid.UUID
	ContactID    *uuid.UUID
	CampaignID   *uuid.UUID
	AutomationID *uuid.UUID
	Target       string
	At           time.Time
}

// PlacementLinks resolves a placement test copy's tracked links.
type PlacementLinks interface {
	TrackedLink(ctx context.Context, copyID uuid.UUID, position int) (string, bool)
}

// placementKind is messages.kind of a placement test copy.
const placementKind = "placement"

// Routes mounts the tracking endpoints.
func (s *Service) Routes(r chi.Router) {
	r.Get("/t/o/{token}", s.open)
	r.Head("/t/o/{token}", s.open)
	r.Get("/t/c/{token}", s.click)
	r.Head("/t/c/{token}", s.click)
}

func noStore(h http.Header) {
	h.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	h.Set("Pragma", "no-cache")
	h.Set("Expires", "0")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("X-Robots-Tag", "noindex, nofollow")
}

// open answers the pixel at once with the GIF; the open is recorded in the
// background (the response doesn't wait for more than a few milliseconds).
func (s *Service) open(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	noStore(h)
	h.Set("Content-Type", "image/gif")
	token := strings.TrimSuffix(chi.URLParam(r, "token"), ".gif")
	if c, err := s.Links.Verify(token, linktoken.Open); err == nil && c.MessageID != uuid.Nil {
		ev := sending.Engagement{MessageID: c.MessageID, UserAgent: r.UserAgent(), IP: clientIP(r), Method: r.Method, At: s.now()}
		ctx, cancel := context.WithTimeout(r.Context(), 150*time.Millisecond)
		if err := s.enqueue(ctx, ev); err != nil && !errors.Is(err, context.DeadlineExceeded) {
			s.log().WarnContext(r.Context(), "record open", "err", err)
		}
		cancel()
	}
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write(pixelGIF)
	}
}

// click redirects to the link of the message the token names, then records
// the click in the background.
func (s *Service) click(w http.ResponseWriter, r *http.Request) {
	noStore(w.Header())
	token := chi.URLParam(r, "token")
	c, err := s.Links.Verify(token, linktoken.Click)
	if err != nil || c.MessageID == uuid.Nil {
		gone(w)
		return
	}
	target, m, err := s.targetOf(r.Context(), c)
	if err != nil || target == "" {
		gone(w)
		return
	}
	at := s.now()
	if m != nil && m.Kind != placementKind {
		for _, d := range s.Decorators {
			target = d.Decorate(r.Context(), ClickedLink{WorkspaceID: m.WorkspaceID, MessageID: m.ID, ContactID: m.ContactID,
				CampaignID: m.CampaignID, AutomationID: m.AutomationID, Target: target, At: at})
		}
	}
	ev := sending.Engagement{MessageID: c.MessageID, Click: true, Link: int(c.Link), UserAgent: r.UserAgent(), IP: clientIP(r), Method: r.Method, At: at}
	if s.Decorate != nil {
		if d := s.Decorate(r.Context(), ClickTarget{WorkspaceID: c.WorkspaceID, MessageID: c.MessageID, ContactID: c.ContactID, URL: target, At: ev.At}); d != "" {
			target = d
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 150*time.Millisecond)
	if err := s.enqueue(ctx, ev); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		s.log().WarnContext(r.Context(), "record click", "err", err)
	}
	cancel()
	// target is one of the signed message's own links (its position is in
	// the signed token, the URL in that campaign's links, an http(s) URL):
	// never a URL taken from the request.
	http.Redirect(w, r, target, http.StatusFound) //nolint:gosec // G710: see target
}

func gone(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("This link is no longer valid. If you copied it, check that the whole address came along.\n"))
}

// targetOf resolves a click token to the original link and the message.
func (s *Service) targetOf(ctx context.Context, c linktoken.Claims) (string, *store.TrackingMessageRow, error) {
	m, err := s.Q.TrackingMessage(ctx, c.MessageID)
	if err != nil {
		return "", nil, err
	}
	t, err := s.resolve(ctx, c, m)
	return t, &m, err
}

// resolve finds the original link of a click: only the links of that
// message's email, personalized for its contact when they contain Liquid,
// and only http(s) URLs.
func (s *Service) resolve(ctx context.Context, c linktoken.Claims, m store.TrackingMessageRow) (string, error) {
	if m.WorkspaceID != c.WorkspaceID {
		return "", errors.New("token workspace mismatch")
	}
	if m.Kind == placementKind {
		if s.Placement == nil {
			return "", errors.New("no links")
		}
		raw, ok := s.Placement.TrackedLink(ctx, m.ID, int(c.Link))
		if !ok {
			return "", errors.New("unknown link")
		}
		u, err := url.Parse(raw)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return "", errors.New("not a web link")
		}
		return u.String(), nil
	}
	if m.Stream == "transactional" {
		if s.Transactional == nil {
			return "", errors.New("no links")
		}
		raw, ok := s.Transactional.TrackedLink(ctx, s.Q, m.ID, int(c.Link))
		if !ok {
			return "", errors.New("unknown link")
		}
		u, err := url.Parse(raw)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return "", errors.New("not a web link")
		}
		return u.String(), nil
	}
	var key uuid.UUID
	var load func() (map[int]string, error)
	switch {
	case m.CampaignID != nil:
		key = *m.CampaignID
		load = func() (map[int]string, error) {
			rows, err := s.Q.TrackingCampaignLinks(ctx, *m.CampaignID)
			if err != nil {
				return nil, err
			}
			out := map[int]string{}
			for _, l := range rows {
				out[int(l.Position)] = l.Url
			}
			return out, nil
		}
	default:
		ref := refOf(m.ContentRef)
		if ref.Email == uuid.Nil {
			return "", errors.New("no links")
		}
		key = ref.linkKey()
		load = func() (map[int]string, error) {
			rows, err := s.Q.TrackingAutomationLinks(ctx, store.TrackingAutomationLinksParams{EmailID: ref.Email, Version: int32(ref.Version), Language: ref.Language})
			if err != nil {
				return nil, err
			}
			out := map[int]string{}
			for _, l := range rows {
				out[int(l.Position)] = l.Url
			}
			return out, nil
		}
	}
	urls, err := s.cachedLinks(key, load)
	if err != nil {
		return "", err
	}
	raw, ok := urls[int(c.Link)]
	if !ok {
		return "", errors.New("unknown link")
	}
	if strings.Contains(raw, "{{") && m.ContactID != nil {
		var links map[string]string
		if ref := refOf(m.ContentRef); m.CampaignID == nil && len(ref.Ctx) > 0 && s.MessageVars != nil {
			links = s.MessageVars(ctx, ref.Ctx)
		}
		if ct, err := s.Q.TrackingContactVars(ctx, *m.ContactID); err == nil {
			var attrs map[string]any
			_ = json.Unmarshal(ct.Attributes, &attrs)
			vars := liquid.Vars(ct.Email, ct.FirstName, ct.LastName, attrs, links)
			if s.Recipients != nil {
				r := liquid.Recipient{ContactID: *m.ContactID, MessageID: m.ID, EmailID: refOf(m.ContentRef).Email}
				vars = liquid.Fill(ctx, s.Recipients(), m.WorkspaceID, r, raw, vars)
			}
			if out, err := liquid.Render(raw, vars); err == nil {
				raw = strings.TrimSpace(out)
			}
		}
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", errors.New("not a web link")
	}
	return u.String(), nil
}

func (s *Service) cachedLinks(key uuid.UUID, load func() (map[int]string, error)) (map[int]string, error) {
	s.mu.Lock()
	if s.links == nil {
		s.links = map[uuid.UUID]linkSet{}
	}
	if ls, ok := s.links[key]; ok && time.Now().Before(ls.until) {
		s.mu.Unlock()
		return ls.urls, nil
	}
	s.mu.Unlock()
	urls, err := load()
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if len(s.links) > 2000 {
		s.links = map[uuid.UUID]linkSet{}
	}
	s.links[key] = linkSet{urls: urls, until: time.Now().Add(5 * time.Minute)}
	s.mu.Unlock()
	return urls, nil
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Group commit: requests a few milliseconds apart share one transaction.
const (
	batchWindow = 15 * time.Millisecond
	batchMax    = 200
)

func (s *Service) start() {
	s.once.Do(func() {
		s.in = make(chan *pending, 4096)
		go s.loop()
	})
}

// enqueue hands an event to the background writer and waits until its
// batch committed (or ctx ends; the write still happens).
func (s *Service) enqueue(ctx context.Context, ev sending.Engagement) error {
	s.start()
	p := &pending{ev: ev, done: make(chan error, 1)}
	select {
	case s.in <- p:
	default:
		// The writer is far behind: write this one directly.
		return s.RecordEngagement(ctx, []sending.Engagement{ev})
	}
	select {
	case err := <-p.done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Service) loop() {
	for first := range s.in {
		batch := []*pending{first}
		timer := time.NewTimer(batchWindow)
	collect:
		for len(batch) < batchMax {
			select {
			case p := <-s.in:
				batch = append(batch, p)
			case <-timer.C:
				break collect
			}
		}
		timer.Stop()
		evs := make([]sending.Engagement, len(batch))
		for i, p := range batch {
			evs[i] = p.ev
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := s.RecordEngagement(ctx, evs)
		cancel()
		if err != nil {
			s.log().Warn("record opens and clicks", "events", len(evs), "err", err)
		}
		for _, p := range batch {
			p.done <- err
		}
	}
}

// RecordEngagement implements sending.Recorder: it classifies and records a
// batch of opens and clicks in one transaction.
func (s *Service) RecordEngagement(ctx context.Context, evs []sending.Engagement) error {
	if len(evs) == 0 {
		return nil
	}
	return sending.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		for _, ev := range evs {
			if err := s.apply(ctx, tx, q, ev); err != nil {
				return err
			}
		}
		return nil
	})
}

// apply records one open or click.
func (s *Service) apply(ctx context.Context, tx pgx.Tx, q *store.Queries, ev sending.Engagement) error {
	m, err := q.TrackingMessage(ctx, ev.MessageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if m.Kind == placementKind {
		return nil // placement test copies: opens and clicks are never counted
	}
	since := time.Duration(-1)
	if m.SentAt != nil {
		since = ev.At.Sub(*m.SentAt)
	}
	ip, _ := netip.ParseAddr(ev.IP)
	sig := Signal{UserAgent: ev.UserAgent, IP: ip, Method: ev.Method, SinceSend: since}
	var v Verdict
	link := ""
	if ev.Click {
		v = ClassifyClick(sig, s.isBurst(ev))
		if urls, err := s.linksOf(ctx, q, m); err == nil {
			link = urls[ev.Link]
		}
	} else {
		v = ClassifyOpen(sig)
	}
	typ := "open"
	if ev.Click {
		typ = "click"
	}
	diag := v.Client
	if v.Machine {
		diag = v.Reason
	}
	var ipPtr *netip.Addr
	if ip.IsValid() {
		ipPtr = &ip
	}
	label, _ := q.TrackingSourceLabel(ctx, m.ID)
	mid := m.ID
	if err := q.TrackingInsertEvent(ctx, store.TrackingInsertEventParams{
		ID: uuid.Must(uuid.NewV7()), WorkspaceID: m.WorkspaceID, MessageID: &mid, CampaignID: m.CampaignID, ContactID: m.ContactID, ProviderID: m.ProviderID,
		Type: typ, OccurredAt: ev.At, ReceivedAt: ev.At, MailboxProvider: m.MailboxProvider, OriginalType: "ReputeCast tracking", Diagnostic: clip(diag, 1000),
		Machine: v.Machine, Url: clip(link, 2048), UserAgent: clip(ev.UserAgent, 500), Ip: ipPtr, Recipient: m.Email, Source: label,
	}); err != nil {
		return err
	}
	if v.Machine && ev.Click {
		return nil // bot clicks are logged, never counted
	}
	var human, machine, clicked int64
	var prevOpen *time.Time // when a machine prefetch opened it, if a person now does
	firstOpen, firstClick := false, false
	if ev.Click {
		r, err := q.TrackingMarkClick(ctx, store.TrackingMarkClickParams{ID: m.ID, At: ev.At})
		if err != nil {
			return err
		}
		firstOpen, firstClick = r.FirstOpen, r.FirstClick
		if r.FirstClick {
			clicked = 1
		}
		if r.FirstHuman {
			human = 1
			if !r.FirstOpen {
				machine, prevOpen = -1, r.PrevOpenedAt // a machine prefetch came first; now a person
			}
		}
		if m.CampaignID != nil {
			unique := int64(0)
			if _, err := q.TrackingFirstLinkClick(ctx, store.TrackingFirstLinkClickParams{MessageID: m.ID, Link: int32(ev.Link), At: ev.At}); err == nil {
				unique = 1
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			if err := q.TrackingBumpLink(ctx, store.TrackingBumpLinkParams{CampaignID: *m.CampaignID, Position: int32(ev.Link), UniqueClicks: unique}); err != nil {
				return err
			}
		} else if ref := refOf(m.ContentRef); ref.Email != uuid.Nil {
			if err := q.TrackingBumpAutomationLink(ctx, store.TrackingBumpAutomationLinkParams{EmailID: ref.Email, Version: int32(ref.Version), Language: ref.Language, Position: int32(ev.Link)}); err != nil {
				return err
			}
		}
		if m.ContactID != nil {
			if err := q.TrackingTouchContactClick(ctx, store.TrackingTouchContactClickParams{ID: *m.ContactID, At: ev.At}); err != nil {
				return err
			}
		}
		if s.Clicked != nil {
			if err := s.Clicked(ctx, q, HumanClick{WorkspaceID: m.WorkspaceID, MessageID: m.ID, CampaignID: m.CampaignID, ContactID: m.ContactID, Stream: m.Stream,
				URL: link, At: ev.At}); err != nil {
				return err
			}
		}
	} else {
		r, err := q.TrackingMarkOpen(ctx, store.TrackingMarkOpenParams{ID: m.ID, At: ev.At, Human: !v.Machine})
		if err != nil {
			return err
		}
		firstOpen = r.FirstOpen
		switch {
		case r.FirstOpen && !v.Machine:
			human = 1
		case r.FirstOpen && v.Machine:
			machine = 1
		case r.FirstHuman:
			human, machine, prevOpen = 1, -1, r.PrevOpenedAt
		}
		if m.ContactID != nil {
			if err := q.TrackingTouchContactOpen(ctx, store.TrackingTouchContactOpenParams{ID: *m.ContactID, At: ev.At, Human: !v.Machine}); err != nil {
				return err
			}
		}
	}
	// A person opening after a machine prefetch moves the open from machine
	// to human (+1, -1): the deltas cancel out in a sum, so test each one.
	if m.CampaignID != nil && m.ProviderID != nil && (human != 0 || machine != 0 || clicked != 0) {
		if err := q.TrackingBumpCounters(ctx, store.TrackingBumpCountersParams{CampaignID: *m.CampaignID, ProviderID: *m.ProviderID, MailboxProvider: m.MailboxProvider,
			Variant: m.Variant, OpenedHuman: human, OpenedMachine: machine, Clicked: clicked}); err != nil {
			return err
		}
		hourOf := func(at time.Time) int32 {
			if m.CampaignStartedAt == nil {
				return 0
			}
			return int32(min(max(at.Sub(*m.CampaignStartedAt).Hours(), 0), 720))
		}
		hh, hm := max(human, 0), max(machine, 0)
		if hh+hm+clicked > 0 {
			if err := q.TrackingBumpHourly(ctx, store.TrackingBumpHourlyParams{CampaignID: *m.CampaignID, Hour: hourOf(ev.At), OpenedHuman: hh, OpenedMachine: hm, Clicked: clicked}); err != nil {
				return err
			}
		}
		// The open moved from machine to human: take the prefetch out of the
		// hour it was counted in, so the engagement chart adds up to the
		// counters.
		if machine < 0 && prevOpen != nil {
			if err := q.TrackingBumpHourly(ctx, store.TrackingBumpHourlyParams{CampaignID: *m.CampaignID, Hour: hourOf(*prevOpen), OpenedMachine: -1}); err != nil {
				return err
			}
		}
	}
	if m.Stream == "transactional" && s.Transactional != nil && (firstOpen || firstClick) {
		if err := s.Transactional.Engagement(ctx, tx, m.ID, ev.Click && firstClick, ev.At); err != nil {
			return err
		}
	}
	if m.Stream == "automation" && s.Automations != nil {
		if firstOpen {
			if err := s.Automations.RecordMessageEvent(ctx, q, automations.MessageEvent{MessageID: m.ID, Type: "opened", Machine: v.Machine && !ev.Click, At: ev.At}); err != nil {
				return err
			}
		}
		if firstClick {
			if err := s.Automations.RecordMessageEvent(ctx, q, automations.MessageEvent{MessageID: m.ID, Type: "clicked", At: ev.At}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) linksOf(ctx context.Context, q *store.Queries, m store.TrackingMessageRow) (map[int]string, error) {
	if m.Stream == "transactional" {
		out := map[int]string{}
		if s.Transactional != nil {
			for i := 1; i <= 50; i++ {
				l, ok := s.Transactional.TrackedLink(ctx, q, m.ID, i)
				if !ok {
					break
				}
				out[i] = l
			}
		}
		return out, nil
	}
	if m.CampaignID != nil {
		return s.cachedLinks(*m.CampaignID, func() (map[int]string, error) {
			rows, err := q.TrackingCampaignLinks(ctx, *m.CampaignID)
			if err != nil {
				return nil, err
			}
			out := map[int]string{}
			for _, l := range rows {
				out[int(l.Position)] = l.Url
			}
			return out, nil
		})
	}
	ref := refOf(m.ContentRef)
	if ref.Email == uuid.Nil {
		return map[int]string{}, nil
	}
	return s.cachedLinks(ref.linkKey(), func() (map[int]string, error) {
		rows, err := q.TrackingAutomationLinks(ctx, store.TrackingAutomationLinksParams{EmailID: ref.Email, Version: int32(ref.Version), Language: ref.Language})
		if err != nil {
			return nil, err
		}
		out := map[int]string{}
		for _, l := range rows {
			out[int(l.Position)] = l.Url
		}
		return out, nil
	})
}

type contentRef struct {
	Email   uuid.UUID `json:"email"`
	Version int       `json:"version"`
	// Language is the language version sent ("" is the source).
	Language string `json:"language,omitempty"`
	// Ctx is what the flow's trigger passed to the email (a store's
	// checkout or order): its link variables come from MessageVars.
	Ctx map[string]string `json:"ctx,omitempty"`
}

// linkKey identifies the link set of an automation email version and language.
func (r contentRef) linkKey() uuid.UUID {
	return uuid.NewSHA1(r.Email, append([]byte{byte(r.Version >> 8), byte(r.Version)}, r.Language...))
}

func refOf(raw json.RawMessage) contentRef {
	var r contentRef
	_ = json.Unmarshal(raw, &r)
	return r
}

// isBurst remembers clicks per message for a few seconds: a message that
// clicked two other links in the last five seconds is a scanner following
// every link.
func (s *Service) isBurst(ev sending.Engagement) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.burst == nil {
		s.burst = map[uuid.UUID][]clickMark{}
	}
	if len(s.burst) > 10000 {
		s.burst = map[uuid.UUID][]clickMark{}
	}
	marks := s.burst[ev.MessageID][:0:0]
	others := map[int]bool{}
	for _, m := range s.burst[ev.MessageID] {
		if ev.At.Sub(m.at) <= 5*time.Second && ev.At.Sub(m.at) >= 0 {
			marks = append(marks, m)
			if m.link != ev.Link {
				others[m.link] = true
			}
		}
	}
	s.burst[ev.MessageID] = append(marks, clickMark{link: ev.Link, at: ev.At})
	return len(others) >= 2
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
