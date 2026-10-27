package tracking

import (
	"context"
	"errors"
	"net/netip"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/counters"
	"github.com/hostrepute/reputecast/internal/ingest"
	"github.com/hostrepute/reputecast/internal/store"
)

// Opens and clicks the sending provider reports (review AN-4). When a
// workspace keeps the provider's tracking instead of ReputeCast's, the
// email goes out without ReputeCast's pixel and redirects, and the
// provider's webhook reports the opens and clicks. Ingest stores each one
// (dropping the provider's redeliveries); here it is classified in place
// and counted exactly like one of ReputeCast's own: an open at once (a
// person's, a privacy proxy's or a scanner's), a click matched to the
// email's link and settled before it counts.

// Begin implements ingest.Engagement: the webhook batch's opens and clicks
// move its own ledger, so its counters are locked once, in order.
func (s *Service) Begin(ledger *counters.Ledger) ingest.EngagementBatch {
	return &providerBatch{s: s, b: &batch{ledger: ledger}}
}

type providerBatch struct {
	s *Service
	b *batch
}

// Record implements ingest.EngagementBatch.
func (p *providerBatch) Record(ctx context.Context, tx pgx.Tx, ev ingest.ProviderEngagement) error {
	return p.s.recordProvider(ctx, tx, store.New(tx), p.b, ev)
}

// Flush implements ingest.EngagementBatch: the contacts' activity, last.
func (p *providerBatch) Flush(ctx context.Context, q *store.Queries) error {
	return p.b.touches.flush(ctx, q)
}

func (s *Service) recordProvider(ctx context.Context, tx pgx.Tx, q *store.Queries, b *batch, ev ingest.ProviderEngagement) error {
	m, err := q.TrackingMessage(ctx, ev.MessageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if m.Kind == placementKind {
		return nil
	}
	if err := q.CountersLockMessage(ctx, m.ID); err != nil {
		return err
	}
	ip, _ := netip.ParseAddr(strings.TrimSpace(ev.IP))
	var ipPtr *netip.Addr
	if ip.IsValid() {
		ipPtr = &ip
	}
	sig := s.signal(ctx, m, ev.UserAgent, ipPtr, "GET", ev.At)
	sig.Reported = true
	var v Verdict
	link := NoLink
	if ev.Click {
		v = ClassifyClick(sig, false)
		if urls, err := s.linksOf(ctx, q, m); err == nil {
			link = LinkPosition(urls, ev.URL)
		}
	} else {
		v = ClassifyOpen(sig)
	}
	st := classified(v, ev.Click, link)
	if err := q.TrackingClassifyEvent(ctx, store.TrackingClassifyEventParams{ID: ev.EventID, Class: st.class, Machine: v.Machine,
		Diagnostic: clip(st.diag, 1000), Link: st.link, Ip: ipPtr}); err != nil {
		return err
	}
	return s.count(ctx, tx, q, b, m, v, ev.EventID, ev.Click, link, ev.URL, ev.At)
}

// LinkPosition finds which of an email's links (position → the link as
// written, possibly personalized with Liquid) a provider-reported click
// was on: the same address, else the one whose personalized parts match;
// NoLink when none does. Two links to the same address are the first's.
func LinkPosition(links map[int]string, clicked string) int {
	clicked = strings.TrimSpace(clicked)
	if clicked == "" {
		return NoLink
	}
	best := NoLink
	for pos, l := range links {
		if strings.TrimSpace(l) == clicked && (best == NoLink || pos < best) {
			best = pos
		}
	}
	if best != NoLink {
		return best
	}
	for pos, l := range links {
		if !strings.Contains(l, "{{") && !strings.Contains(l, "{%") {
			continue
		}
		if re := templatePattern(strings.TrimSpace(l)); re != nil && re.MatchString(clicked) && (best == NoLink || pos < best) {
			best = pos
		}
	}
	return best
}

// liquidTag matches a Liquid output or tag.
var liquidTag = regexp.MustCompile(`\{\{.*?\}\}|\{%.*?%\}`)

// templatePattern turns a personalized link into a pattern: the written
// parts as they are, each Liquid part any text.
func templatePattern(l string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("^")
	last := 0
	for _, m := range liquidTag.FindAllStringIndex(l, -1) {
		b.WriteString(regexp.QuoteMeta(l[last:m[0]]))
		b.WriteString(".*?")
		last = m[1]
	}
	b.WriteString(regexp.QuoteMeta(l[last:]))
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	if err != nil {
		return nil
	}
	return re
}
