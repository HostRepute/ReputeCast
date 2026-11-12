package demo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/dnscheck"
)

func init() { seedInbound = seedInboundImpl }

// The receiving domain of the M58 boards: Acme Newsletter receives at
// reply.news.acme.io through its Postmark connection, keeps replies in
// ReputeCast's inbox, and has the 14 messages of Audience › Replies
// (sample data: 6 open human replies, 3 done, 2 auto-replies, 1 spam, 1
// unsubscribe email and 1 other message), plus yesterday's DMARC report
// from Google received by email.

// InboundDemoDomain is the demo's receiving domain.
const InboundDemoDomain = "reply.news.acme.io"

// inboundDemoZone adds the receiving domain's MX record to the demo zone.
func inboundDemoZone(z *dnscheck.Zone) {
	z.Add(InboundDemoDomain, "MX", "10 inbound.postmarkapp.com")
}

type inboundSample struct {
	name, email, subject, body string
	// source: "campaign" or "automation" and its name ("" = none).
	source, sourceName string
	kind               string
	auto, spam, asks   bool
	done               bool
	ago                time.Duration
	// pick: when email is "", a demo contact is picked: "any" or
	// "unsubscribed".
	pick string
	to   string
}

const (
	inboundOctober = "October Product Update"
	inboundDigest  = "Weekly Digest #141"
	inboundWelcome = "Welcome series"
)

var inboundSamples = []inboundSample{
	{name: "Sam Rivera", email: "sam.rivera@outlook.com", subject: "Re: October Product Update: three launches", source: "campaign", sourceName: inboundOctober, kind: "reply", ago: 2 * time.Hour,
		body: "Loved the new routing rules. Could we get early access to the API changes you mentioned? We'd like to test them with our staging workspace before they ship.\n\nThanks,\nSam\n\n> On Tue, Acme wrote:\n> October Product Update: three launches"},
	{name: "Priya Natarajan", email: "priya.n@gmail.com", subject: "Re: Weekly Digest #141", source: "campaign", sourceName: inboundDigest, kind: "reply", ago: 5 * time.Hour,
		body: "The DMARC article was exactly what our team needed this week. Do you have a checklist version we could share internally?"},
	{name: "Aisha Rahman", email: "aisha@rahmanlabs.io", subject: "Automatic reply: Weekly Digest #141", source: "campaign", sourceName: inboundDigest, kind: "reply", auto: true, ago: 5*time.Hour + 4*time.Minute,
		body: "I'm out of the office until Monday with limited access to email. For anything urgent, contact support@rahmanlabs.io."},
	{name: "Diego Alvarez", email: "d.alvarez@yahoo.com", subject: "Re: Weekly Digest #141", source: "campaign", sourceName: inboundDigest, kind: "reply", asks: true, ago: 6 * time.Hour,
		body: "Please take me off this list, I don't use Acme anymore."},
	{name: "Omar Haddad", email: "omar.haddad@gmail.com", subject: "Re: You're in. Here's what to expect from Acme", source: "automation", sourceName: inboundWelcome, kind: "reply", ago: 26 * time.Hour,
		body: "Thanks! Quick question before I invite my team: does the Starter plan include SSO?"},
	{name: "Hannah Berg", email: "hannah@bergconsulting.de", subject: "Re: October Product Update: three launches", source: "campaign", sourceName: inboundOctober, kind: "reply", ago: 30 * time.Hour,
		body: "Is the German translation of the docs coming as well? Our support team would use it."},
	{name: "Kenji Watanabe", email: "kenji.w@icloud.com", subject: "Re: Weekly Digest #141", source: "campaign", sourceName: inboundDigest, kind: "reply", ago: 50 * time.Hour,
		body: "Heads up: the second link in this issue goes to a 404 page."},
	{pick: "any", subject: "Out of office", source: "campaign", sourceName: inboundDigest, kind: "reply", auto: true, ago: 74 * time.Hour,
		body: "Thank you for your message. I am currently away and will reply when I return."},
	{name: "SEO Rank Boost", email: "deals@seo-rank-boost.biz", subject: "Guaranteed page one ranking for acme.io", source: "campaign", sourceName: inboundDigest, kind: "reply", spam: true, ago: 76 * time.Hour,
		body: "We can put acme.io on page one of Google in 30 days. Reply YES for a free audit."},
	{pick: "unsubscribed", subject: "unsubscribe", source: "campaign", sourceName: inboundDigest, kind: "unsubscribe", done: true, ago: 98 * time.Hour,
		body: "unsubscribe"},
	{name: "Jamie Park", email: "jamie@brightwave.io", subject: "Partnership idea", kind: "other", ago: 100 * time.Hour, to: "partners@" + InboundDemoDomain,
		body: "Hi Acme team, I run partnerships at Brightwave. Would you be open to a co-marketing newsletter swap next quarter?"},
	{name: "Tomasz Nowak", email: "t.nowak@gmail.com", subject: "Re: October Product Update: three launches", source: "campaign", sourceName: inboundOctober, kind: "reply", done: true, ago: 122 * time.Hour,
		body: "Thanks, all set now."},
	{name: "Grace Okafor", email: "g.okafor@outlook.com", subject: "Re: Three things to try first", source: "automation", sourceName: inboundWelcome, kind: "reply", done: true, ago: 145 * time.Hour,
		body: "Tried the first two already. The third link doesn't open on my phone."},
	{name: "Lena Fischer", email: "lena.fischer@hotmail.com", subject: "Re: Weekly Digest #141", source: "campaign", sourceName: inboundDigest, kind: "reply", done: true, ago: 150 * time.Hour,
		body: "Can you resend last week's issue? I deleted it by accident."},
}

func seedInboundImpl(_ context.Context, s *Seeder) error {
	// Campaigns, flows and their messages are seeded later in the order.
	s.After(seedInboundAfter)
	return nil
}

func seedInboundAfter(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	postmark, ok := s.Providers["postmark"]
	if !ok {
		return errors.New("inbound: the demo's Postmark provider is missing")
	}
	linh := s.Users["linh@acme.io"]
	route := ID()
	secret, _ := authn.NewToken()
	token, hash := authn.NewToken()
	sealedSecret, err := s.Box.SealString(secret, "inbound_route:"+route.String()+":secret")
	if err != nil {
		return err
	}
	sealedToken, err := s.Box.SealString(token, "inbound_route:"+route.String()+":hook_token")
	if err != nil {
		return err
	}
	set := s.Ago(12 * 24 * time.Hour)
	if _, err := s.Tx.Exec(ctx, `INSERT INTO inbound_routes (id, workspace_id, kind, provider_id, domain, secret_sealed, hook_token_hash, hook_token_sealed,
		status, registered_at, verified_at, last_received_at, notified_at, created_by, created_at, updated_at)
		VALUES ($1, $2, 'provider', $3, $4, $5, $6, $7, 'receiving', $8, $9, $10, $10, $11, $8, $8)`,
		route, ws.ID, postmark, InboundDemoDomain, sealedSecret, hash, sealedToken, set, set.Add(4*time.Minute), s.Ago(2*time.Hour), linh.ID); err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `UPDATE workspaces SET sending_defaults = jsonb_set(coalesce(sending_defaults, '{}'::jsonb), '{reply_handling}', '"inbox"') WHERE id = $1`, ws.ID); err != nil {
		return err
	}
	var flow uuid.UUID
	if err := s.Tx.QueryRow(ctx, `SELECT id FROM automations WHERE workspace_id = $1 AND name = $2`, ws.ID, inboundWelcome).Scan(&flow); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	used := []uuid.UUID{}
	for i, m := range inboundSamples {
		var contact, message, campaign, automation *uuid.UUID
		if m.source == "campaign" {
			if id, ok := s.Campaigns[m.sourceName]; ok {
				campaign = &id
			}
		} else if m.source == "automation" && flow != uuid.Nil {
			f := flow
			automation = &f
		}
		name, email := m.name, m.email
		switch {
		case m.pick != "":
			// A demo contact the sample doesn't name, with a message of the
			// campaign.
			// "unsubscribed": someone whose address the unsubscribe put on the
			// suppression list.
			unsubscribed := m.pick == "unsubscribed"
			var c uuid.UUID
			var msg *uuid.UUID
			var first, last string
			// Preferably one the campaign was sent to.
			err := s.Tx.QueryRow(ctx, `SELECT ct.id, ct.email, ct.first_name, ct.last_name, m.id FROM contacts ct
				LEFT JOIN LATERAL (SELECT id FROM messages WHERE contact_id = ct.id AND campaign_id = $2 ORDER BY id LIMIT 1) m ON true
				WHERE ct.workspace_id = $1 AND ct.erased_at IS NULL AND ct.id <> ALL ($4) AND ct.first_name <> ''
				  AND EXISTS (SELECT 1 FROM suppressions su WHERE su.workspace_id = $1 AND su.email = lower(ct.email) AND su.reason = 'unsubscribe') = $3
				  AND (ct.status = 'active' OR $3)
				ORDER BY m.id IS NULL, ct.email LIMIT 1`, ws.ID, campaign, unsubscribed, used).Scan(&c, &email, &first, &last, &msg)
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			if err != nil {
				return err
			}
			contact, message, name = &c, msg, first+" "+last
			used = append(used, c)
		case m.kind != "other" && !m.spam:
			var c uuid.UUID
			if err := s.Tx.QueryRow(ctx, `SELECT id FROM contacts WHERE workspace_id = $1 AND lower(email) = $2`, ws.ID, email).Scan(&c); err == nil {
				contact = &c
				used = append(used, c)
				var msg uuid.UUID
				q := `SELECT id FROM messages WHERE contact_id = $1 AND campaign_id = $2 ORDER BY id LIMIT 1`
				var arg any = campaign
				if automation != nil {
					q, arg = `SELECT id FROM messages WHERE contact_id = $1 AND automation_id = $2 ORDER BY id LIMIT 1`, automation
				}
				if err := s.Tx.QueryRow(ctx, q, c, arg).Scan(&msg); err == nil {
					message = &msg
				}
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
		}
		at := s.Ago(m.ago)
		status := "open"
		var handledBy *uuid.UUID
		var handledAt *time.Time
		if m.done {
			status = "done"
			if m.kind == "reply" {
				h, ha := linh.ID, at.Add(40*time.Minute)
				handledBy, handledAt = &h, &ha
			}
		}
		to := m.to
		if to == "" {
			letter := "r-"
			if m.kind == "unsubscribe" {
				letter = "u-"
			}
			to = letter + inboundDemoTag(i) + "@" + InboundDemoDomain
		}
		auth, _ := json.Marshal(map[string]string{"spf": "pass", "dkim": "pass", "dmarc": "pass"})
		if m.spam {
			auth, _ = json.Marshal(map[string]string{"spf": "fail", "dkim": "none", "dmarc": "fail"})
		}
		attachments := "[]"
		if m.email == "kenji.w@icloud.com" {
			// RepliesStates draws Kenji's screenshot: kept by name only.
			attachments = `[{"name":"screenshot.png","type":"image/png","size":188416}]`
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO inbound_messages (id, workspace_id, route_id, kind, message_id, contact_id, campaign_id, automation_id,
			from_email, from_name, to_address, subject, text_body, has_html, attachments, auto_reply, spam, asks_unsubscribe, auth, status, handled_by, handled_at,
			header_message_id, received_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, false, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)`,
			ID(), ws.ID, route, m.kind, message, contact, campaign, automation, email, name, to, m.subject, m.body, attachments, m.auto, m.spam, m.asks, auth,
			status, handledBy, handledAt, "demo-"+inboundDemoTag(i)+"@mail.example", at); err != nil {
			return err
		}
	}
	// Yesterday's DMARC aggregate report from Google arrived by email.
	_, err = s.Tx.Exec(ctx, `UPDATE dmarc_reports SET via = 'inbound', received_at = $2 WHERE id = (
		SELECT id FROM dmarc_reports WHERE workspace_id = $1 AND org_name ILIKE 'google%' ORDER BY date_end DESC LIMIT 1)`, ws.ID, s.Ago(26*time.Hour))
	return err
}

// inboundDemoTag is a stable tag-shaped local part for a sample (the demo's
// addresses aren't signed: nobody posts to them).
func inboundDemoTag(i int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz234567"
	b := make([]byte, 39)
	for j := range b {
		b[j] = alphabet[(i*7+j*13+5)%32]
	}
	return string(b)
}
