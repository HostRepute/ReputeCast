package demo

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/domains"
	"github.com/hostrepute/reputecast/internal/store"
)

func init() { seedSenders = seedSendersImpl }

// The sending domains and senders of the sample data (SettingsSenders, and
// every board that names a From address): news.acme.io, the primary sending
// domain set up on Amazon SES, Postmark and Postal with a tracking domain,
// and acme.io, the corporate domain Marcus sends personal mail from, still
// waiting for two of its three SES DKIM records.
//
// This seeder runs right after the core data; the per-provider DNS records
// and each sender's verification need the providers, so they are written
// in an After step, and checked against the demo DNS zone exactly the way
// the re-verification job checks them.

// sendersAcmeDKIM are the SES Easy DKIM tokens of acme.io; only the first
// is published in the demo DNS ("Pending CNAME 2 of 3").
var sendersAcmeDKIM = []string{"q2w7e4r9t1y6u3i8o5p0a7s2d9f4g1h6", "m3n8b5v2c7x4z9l1k6j3h8g5f2d7s4a9", "w6e1r8t3y0u5i2o7p4a1s8d5f2g9h6j3"}

// SendersNameservers are the demo zone's nameservers (Cloudflare).
var SendersNameservers = []string{"dana.ns.cloudflare.com", "rob.ns.cloudflare.com"}

// SendersFooterAddress is Acme's postal address in email footers.
const SendersFooterAddress = "Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105"

type sendersDemoSender struct {
	name, email, replyTo, note, domain string
	created                            time.Duration
}

var sendersDemo = []sendersDemoSender{
	{"Acme Newsletter", "digest@news.acme.io", "hello@acme.io", "", "news.acme.io", 400 * 24 * time.Hour},
	{"Acme", "hello@news.acme.io", "hello@acme.io", "", "news.acme.io", 399 * 24 * time.Hour},
	{"Acme accounts", "no-reply@news.acme.io", "support@acme.io", "", "news.acme.io", 398 * 24 * time.Hour},
	{"Marcus Chen", "marcus@acme.io", "marcus@acme.io", "personal sends", "acme.io", 12 * 24 * time.Hour},
	{"Acme Product", "product@news.acme.io", "hello@acme.io", "product updates", "news.acme.io", 300 * 24 * time.Hour},
	{"Acme Beta", "beta@news.acme.io", "hello@acme.io", "beta program", "news.acme.io", 200 * 24 * time.Hour},
}

func seedSendersImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	linh := s.Users["linh@acme.io"]
	for _, d := range []struct {
		name, note, tracking string
		created              time.Duration
	}{
		{"news.acme.io", "primary sending domain", "go.news.acme.io", 400 * 24 * time.Hour},
		{"acme.io", "corporate domain, personal sends", "", 12 * 24 * time.Hour},
	} {
		if _, ok := s.Domains[d.name]; ok {
			continue
		}
		id := ID()
		created := s.Ago(d.created)
		q := store.New(s.Tx)
		if _, err := q.DomCreate(ctx, store.DomCreateParams{ID: id, WorkspaceID: ws.ID, Domain: d.name, Note: d.note, DnsRecords: json.RawMessage(`[]`), CreatedBy: &linh.ID, CreatedAt: created}); err != nil {
			return err
		}
		if d.tracking != "" {
			if err := q.DomUpdateTracking(ctx, store.DomUpdateTrackingParams{ID: id, TrackingDomain: d.tracking, TrackingStatus: domains.StatusPending, UpdatedAt: created}); err != nil {
				return err
			}
		}
		s.Domains[d.name] = id
	}
	for _, sn := range sendersDemo {
		if _, ok := s.Senders[sn.email]; ok {
			continue
		}
		id := ID()
		if _, err := store.New(s.Tx).SenderCreate(ctx, store.SenderCreateParams{ID: id, WorkspaceID: ws.ID, DomainID: s.Domains[sn.domain], Name: sn.name, Email: sn.email,
			ReplyTo: sn.replyTo, Note: sn.note, Verification: json.RawMessage(`{}`), CreatedBy: &linh.ID, CreatedAt: s.Ago(sn.created)}); err != nil {
			return err
		}
		s.Senders[sn.email] = id
	}
	digest, hello, noReply := s.Senders["digest@news.acme.io"], s.Senders["hello@news.acme.io"], s.Senders["no-reply@news.acme.io"]
	defaults, _ := json.Marshal(map[string]any{"reply_handling": "forward", "reply_to": "hello@acme.io", "tracking_domain": "go.news.acme.io"})
	if _, err := s.Tx.Exec(ctx, `UPDATE workspaces SET default_sender_id = $2, default_automation_sender_id = $3, default_system_sender_id = $4,
		sending_defaults = sending_defaults || $5::jsonb, branding = jsonb_set(branding, '{footer_address}', to_jsonb($6::text)) WHERE id = $1`,
		ws.ID, digest, hello, noReply, defaults, SendersFooterAddress); err != nil {
		return err
	}
	s.After(seedSendersRecords)
	return nil
}

// seedSendersRecords writes the records each demo provider needs for the
// two domains, marks Postal as never sending as no-reply@, and runs the
// live check against the demo DNS zone.
func seedSendersRecords(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	ses, postmark, postal := s.Providers["ses"], s.Providers["postmark"], s.Providers["postal"]
	if ses == uuid.Nil {
		return nil // no providers in this demo
	}
	q := store.New(s.Tx)
	rua, _ := q.DomDMARCReports(ctx, ws.ID)
	// news.acme.io: the deliverability seeder's records (the demo DNS zone
	// publishes exactly these), in the senders domain's shape.
	news := []domains.Record{}
	for _, r := range deliverabilityDemoRecords(s) {
		rec := domains.Record{ProviderID: r["provider_id"].(string), Purpose: r["purpose"].(string), Type: r["type"].(string), Name: r["name"].(string),
			Value: r["value"].(string), Source: "provider"}
		if rec.Purpose == domains.PurposeSPF && rec.Name == "news.acme.io" {
			rec.Mechanisms = domains.SPFMechanisms("v=spf1 " + rec.Value)
			rec.Source = "reputecast"
		}
		news = append(news, rec)
	}
	news = append(news, domains.Record{ProviderID: ses.String(), Purpose: domains.PurposeSPF, Type: "TXT", Name: "news.acme.io",
		Mechanisms: []string{"include:amazonses.com"}, Source: "reputecast"})
	news = append(news, domains.Record{Purpose: domains.PurposeDMARC, Type: "TXT", Name: "_dmarc.news.acme.io", Value: domains.SuggestDMARC(rua), Source: "reputecast"})
	acme := []domains.Record{}
	for _, t := range sendersAcmeDKIM {
		acme = append(acme, domains.Record{ProviderID: ses.String(), Purpose: domains.PurposeDKIM, Type: "CNAME", Name: t + "._domainkey.acme.io", Value: t + ".dkim.amazonses.com", Source: "provider"})
	}
	acme = append(acme, domains.Record{Purpose: domains.PurposeDMARC, Type: "TXT", Name: "_dmarc.acme.io", Value: domains.SuggestDMARC(rua), Source: "reputecast"})
	for name, recs := range map[string][]domains.Record{"news.acme.io": news, "acme.io": acme} {
		if err := q.DomUpdateRecords(ctx, store.DomUpdateRecordsParams{ID: s.Domains[name], DnsRecords: domains.EncodeRecords(recs), UpdatedAt: s.Ago(time.Hour)}); err != nil {
			return err
		}
	}
	// no-reply@ is the System sender: Postal (Broadcasts only) never sends as it.
	if id, ok := s.Senders["no-reply@news.acme.io"]; ok && postal != uuid.Nil {
		b, _ := json.Marshal(map[string]string{postal.String(): "not_used"})
		if err := q.SenderSetVerification(ctx, store.SenderSetVerificationParams{ID: id, Verification: b}); err != nil {
			return err
		}
	}
	_ = postmark
	svc := SendersDomainService(s.Tx, s.Cfg.HTTP.PublicURL)
	svc.Now = func() time.Time { return s.Ago(4 * time.Minute) }
	for _, name := range []string{"news.acme.io", "acme.io"} {
		d, err := q.DomGetAny(ctx, s.Domains[name])
		if err != nil {
			return err
		}
		if _, err := svc.Check(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

// SendersDomainService is a domains service on the demo DNS zone, for the
// seeder's checks.
func SendersDomainService(tx store.DBTX, publicURL string) *domains.Service {
	target := TrackingTarget(publicURL)
	z, owned := DeliverabilityDNSZone()
	r := SendersDNSOverlay(dnscheck.Overlay(z, owned, dnscheck.NewZone()), target)
	return &domains.Service{Q: store.New(tx), DNS: r, NS: SendersNS(nil), TrackingTarget: func() string { return target }}
}

// TrackingTarget is the host custom tracking domains point to: the host of
// the public URL.
func TrackingTarget(publicURL string) string {
	u, err := url.Parse(publicURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

// SendersDNSOverlay answers the names only the senders demo needs (acme.io's
// published SES DKIM key and the tracking domain's CNAME) on top of r, which
// is normally the deliverability demo zone.
func SendersDNSOverlay(r dnscheck.Resolver, trackingTarget string) dnscheck.Resolver {
	z := dnscheck.NewZone()
	t := sendersAcmeDKIM[0]
	z.Add(t+"._domainkey.acme.io", "CNAME", t+".dkim.amazonses.com")
	z.Add(t+".dkim.amazonses.com", "TXT", "p="+deliverabilityDemoKey)
	if trackingTarget != "" {
		z.Add("go.news.acme.io", "CNAME", trackingTarget)
	}
	return dnscheck.Overlay(z, nil, r)
}

// SendersNS answers the demo zone's nameservers (acme.io and its
// subdomains) and asks next for everything else.
func SendersNS(next func(context.Context, string) ([]string, error)) func(context.Context, string) ([]string, error) {
	return func(ctx context.Context, name string) ([]string, error) {
		n := strings.ToLower(strings.TrimSuffix(name, "."))
		if n == "acme.io" {
			return append([]string{}, SendersNameservers...), nil
		}
		if strings.HasSuffix(n, ".acme.io") {
			return nil, &net.DNSError{Err: "no such host", Name: name, IsNotFound: true}
		}
		if next == nil {
			return nil, errors.New("no nameserver lookup")
		}
		return next(ctx, name)
	}
}
