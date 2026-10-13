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

	"github.com/hostrepute/reputecast/internal/connect"
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
	return seedSendersSetup(ctx, s)
}

// seedSendersSetup is the setup that connected Amazon SES (SetupSES to
// SetupDone): CloudFormation access in us-east-1, provisioning, the domain,
// the test message and its delivery events, the three streams and the
// suppression import.
func seedSendersSetup(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	ses := s.Providers["ses"]
	linh := s.Users["linh@acme.io"]
	var webhookToken string
	if err := s.Tx.QueryRow(ctx, `SELECT webhook_token FROM providers WHERE id = $1`, ses).Scan(&webhookToken); err != nil {
		return err
	}
	at := func(h, m, sec, ms int) time.Time {
		d := s.designDay(210, h, m)
		return d.Add(time.Duration(sec)*time.Second + time.Duration(ms)*time.Millisecond)
	}
	hook := strings.TrimRight(s.Cfg.HTTP.PublicURL, "/") + "/hooks/ses/" + webhookToken
	done := func(key, label, text string, t time.Time) connect.ProvisionState {
		return connect.ProvisionState{Key: key, Label: label, Text: text, Status: "done", At: &t}
	}
	connected := at(9, 13, 20, 0)
	state := connect.State{
		Connection: connect.Connection{
			Status: "connected", At: &connected, Account: "Production · no enforcement", AccountID: "123456789012", Summary: "us-east-1 · production",
			Access: "role", ExternalID: "rc-6Hd2kQx9Lm4Vp7Rs1Tz0", RoleARN: "arn:aws:iam::123456789012:role/ReputeCastSES", Region: "us-east-1",
			Settings: map[string]string{"region": "us-east-1", "access": "role"},
			Regions: []connect.RegionState{
				{Region: "us-east-1", Name: "N. Virginia", Production: true, Max24h: 50000, MaxRate: 14},
				{Region: "eu-west-1", Name: "Ireland", Max24h: 200, MaxRate: 1},
				{Region: "us-west-2", Name: "Oregon", Max24h: 200, MaxRate: 1},
			},
			Provisioning: []connect.ProvisionState{
				done("verify", "Verify credentials and permissions", "Verified credentials and permissions", at(9, 13, 20, 0)),
				done("account", "Read the account", "Read account: production access, 50,000 per 24 h, 14 per second", at(9, 13, 21, 0)),
				done("sns_topic", "Create SNS topic `reputecast-events` and subscribe `"+hook+"`", "Created SNS topic `reputecast-events` and subscribed `"+hook+"`", at(9, 13, 23, 0)),
				done("sns_confirm", "Confirm SNS subscription (handshake)", "Confirmed SNS subscription (handshake received)", at(9, 13, 24, 0)),
				done("config_set:broadcast", "Create configuration set `rc-broadcasts`", "Created configuration set `rc-broadcasts` with event destination, tracking off", at(9, 13, 26, 0)),
				done("config_set:automation", "Create configuration set `rc-automations`", "Created configuration set `rc-automations` with event destination, tracking off", at(9, 13, 27, 0)),
				done("config_set:system", "Create configuration set `rc-system`", "Created configuration set `rc-system` with event destination, tracking off", at(9, 13, 28, 0)),
			},
		},
	}
	checked := at(9, 19, 40, 0)
	state.Domain = &connect.DomainStep{DomainID: s.Domains["news.acme.io"].String(), Name: "news.acme.io", Verified: 7, Total: 7, At: &checked}
	// The test message: queued, accepted by SES, the SNS "Send" event, the delivery.
	msgID := ID()
	queued, sent, event, delivered := at(9, 21, 4, 112), at(9, 21, 4, 380), at(9, 21, 5, 16), at(9, 21, 6, 471)
	pmid := "010001929f3a8b2e-6c1d0f4e-9a7b-4c3e-b1f2-0d9e8c7b6a5f-000000"
	if _, err := s.Tx.Exec(ctx, `INSERT INTO messages (id, workspace_id, stream, email, provider_id, provider_message_id, mailbox_provider, status, queued_at, sent_at, delivered_at, kind)
		VALUES ($1,$2,'system','linh@acme.io',$3,$4,'gmail','delivered',$5,$6,$7,'test')`, msgID, ws.ID, ses, pmid, queued, sent, delivered); err != nil {
		return err
	}
	rawID := ID()
	if _, err := s.Tx.Exec(ctx, `INSERT INTO provider_raw_events (id, workspace_id, provider_id, received_at, content_type, body, signature_ok, normalized_at, source, events)
		VALUES ($1,$2,$3,$4,'text/plain',$5,true,$4,'webhook',0)`, rawID, ws.ID, ses, event, snsBody("Send")); err != nil {
		return err
	}
	deliveryRaw := ID()
	if _, err := s.Tx.Exec(ctx, `INSERT INTO provider_raw_events (id, workspace_id, provider_id, received_at, content_type, body, signature_ok, normalized_at, source, events)
		VALUES ($1,$2,$3,$4,'text/plain',$5,true,$4,'webhook',1)`, deliveryRaw, ws.ID, ses, delivered.Add(120*time.Millisecond), snsBody("Delivery")); err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO delivery_events (id, workspace_id, message_id, provider_id, raw_event_id, type, occurred_at, mailbox_provider, original_type, original_code, diagnostic,
		recipient, provider_message_id, source, received_at, dedupe_key)
		VALUES ($1,$2,$3,$4,$5,'delivered',$6,'gmail','Delivery','250','2.0.0 OK 1760260866 d2e1a72fcca58-… · smtpResponse from gmail-smtp-in.l.google.com',
		'linh@acme.io',$7,'Test message',$8,$9)`, ID(), ws.ID, msgID, ses, deliveryRaw, delivered, pmid, delivered.Add(120*time.Millisecond), "demo-test-"+msgID.String()); err != nil {
		return err
	}
	hello := s.Senders["hello@news.acme.io"]
	state.Test = &connect.TestStep{MessageID: msgID.String(), SenderID: hello.String(), From: "hello@news.acme.io", To: "linh@acme.io", Subject: "ReputeCast test from Amazon SES",
		QueuedAt: queued, RoundTripMs: delivered.Sub(queued).Milliseconds(), SizeBytes: 4 << 10}
	streamsAt := at(9, 21, 30, 0)
	state.Streams = &connect.StreamsStep{Saved: true, At: &streamsAt, Streams: map[string]connect.StreamChoice{
		"broadcast": {Enabled: true, Role: "primary", Weight: 100}, "automation": {Enabled: true, Role: "primary"}, "system": {Enabled: true, Role: "primary"}}}
	imported := int64(612)
	contactsAt := at(9, 21, 50, 0)
	state.Contacts = &connect.ContactsStep{Suppressions: &connect.SuppressionPreview{Available: true, Total: 612, Bounces: 540, Complaints: 72, At: contactsAt},
		Choice: "empty", Pull: true, ListsCreated: []string{"Weekly Digest", "Product Updates", "Beta Program"}, Imported: &imported, At: &contactsAt}
	// What the setup provisioned lives in the provider's settings, as the
	// connect flow writes it: the topic (kept with the ones SNS may post
	// from), a configuration set per stream and the account.
	topic := "arn:aws:sns:us-east-1:123456789012:reputecast-events"
	if _, err := s.Tx.Exec(ctx, `UPDATE providers SET config = jsonb_set(config, '{topic_arns}', COALESCE(config->'topic_arns', '[]'::jsonb) || to_jsonb($2::text))
		|| jsonb_build_object('topic_arn', $2::text, 'account_id', '123456789012', 'external_id', $3::text, 'role_arn', $4::text,
			'configuration_sets', jsonb_build_object('broadcast', 'rc-broadcasts', 'automation', 'rc-automations', 'system', 'rc-system'))
		WHERE id = $1`, ses, topic, state.Connection.ExternalID, state.Connection.RoleARN); err != nil {
		return err
	}
	stateJSON, _ := json.Marshal(state)
	created, finished := at(9, 12, 30, 0), at(9, 21, 50, 0)
	_, err := s.Tx.Exec(ctx, `INSERT INTO provider_setups (id, workspace_id, adapter, provider_id, step, status, state, created_by, created_at, updated_at, finished_at)
		VALUES ($1,$2,'ses',$3,'done','finished',$4,$5,$6,$7,$7)`, ID(), ws.ID, ses, stateJSON, linh.ID, created, finished)
	return err
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

// snsBody is a minimal SNS notification of an SES event, as the webhook stored it.
func snsBody(eventType string) []byte {
	inner, _ := json.Marshal(map[string]any{"eventType": eventType})
	b, _ := json.Marshal(map[string]any{"Type": "Notification", "Message": string(inner)})
	return b
}
