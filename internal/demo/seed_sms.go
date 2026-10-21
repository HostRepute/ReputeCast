package demo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/sms/twilio/twiliotest"
	"github.com/hostrepute/reputecast/internal/sms/vonage/vonagetest"
)

// SMS (design expansion phase 8, X21; brief §4): Twilio and Vonage with
// their senders and registrations, the routes and rules, the text message
// terms, the Acme Summit 2026 list with its registrants' phones and SMS
// consent, the opt-in confirmations, the sent campaign "Summit: 8 days to
// go" with its receipts, clicks and replies, Daniel's draft, the SMS
// suppression list and the "Acme Summit reminders" automation.
//
// Days are the design's: today is Wednesday 15 October (designDay(0)); the
// campaign went out yesterday at 10:00 in each recipient's time zone.

func init() { seedSMS = seedSMSImpl }

// SMSDNSOverlay answers the demo's short-link domain (go.acme.io) from the
// demo zone: a CNAME to the instance, as Marcus set it up.
func SMSDNSOverlay(r dnscheck.Resolver, urls publicurl.Builder) dnscheck.Resolver {
	host := "localhost"
	if u, err := url.Parse(urls.Base()); err == nil && u.Hostname() != "" {
		host = strings.ToLower(u.Hostname())
	}
	z := dnscheck.NewZone()
	z.Add("go.acme.io", "CNAME", host+".")
	return dnscheck.Overlay(z, nil, r)
}

// TwilioDemoWebhookSecret signs the demo Twilio account's webhooks (the
// shared key ReputeCast registered).
const TwilioDemoWebhookSecret = "demo-twilio-shared-key-secret-acme-summit" //nolint:gosec // the demo account's, in the open on purpose

// smsDay is a day of the design's October at a local time: smsDay(14, 10,
// 0, chicago) is yesterday 10:00 in Chicago.
func (s *Seeder) smsDay(day, hour, minute int, loc *time.Location) time.Time {
	base := s.designDay(15-day, 12, 0)
	y, m, d := base.Date()
	return time.Date(y, m, d, hour, minute, 0, 0, loc)
}

func smsLoc(name string) *time.Location {
	l, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return l
}

// smsState carries the SMS seed's ids between its parts.
type smsState struct {
	twilio, vonage             uuid.UUID
	tollFree, tenDLC, acme     uuid.UUID
	summit                     uuid.UUID
	campaign, draft            uuid.UUID
	automation                 uuid.UUID
	summitDate                 time.Time
	confirmation, help, optOut string
	body                       string
}

func seedSMSImpl(ctx context.Context, s *Seeder) error {
	st := &smsState{}
	// The registrants' numbers and lists are history, not news for the
	// workspace's webhooks: what the seed queues for them is dropped at the end.
	var lastEvent int64
	if err := s.Tx.QueryRow(ctx, `SELECT coalesce(max(id), 0) FROM webhook_contact_events`).Scan(&lastEvent); err != nil {
		return err
	}
	defer func() {
		_, _ = s.Tx.Exec(ctx, `DELETE FROM webhook_contact_events WHERE id > $1`, lastEvent)
	}()
	if err := smsProviders(ctx, s, st); err != nil {
		return fmt.Errorf("providers: %w", err)
	}
	if err := smsRules(ctx, s, st); err != nil {
		return fmt.Errorf("rules: %w", err)
	}
	if err := smsCampaignRows(ctx, s, st); err != nil {
		return fmt.Errorf("campaigns: %w", err)
	}
	if err := smsRegistrants(ctx, s, st); err != nil {
		return fmt.Errorf("registrants: %w", err)
	}
	if err := smsAutomation(ctx, s, st); err != nil {
		return fmt.Errorf("automation: %w", err)
	}
	return smsAudit(ctx, s, st)
}

func smsToken(seed string) string {
	sum := sha256.Sum256([]byte("reputecast-demo-sms:" + seed))
	return hex.EncodeToString(sum[:16])
}

// smsProviders connects Twilio (1 Oct) and Vonage (2 Oct), Marcus Chen's
// doing, with their senders, registrations, prices and opt-out sync.
func smsProviders(ctx context.Context, s *Seeder, st *smsState) error {
	ws := s.WS()
	marcus := s.Users["marcus@acme.io"]
	chicago := smsLoc("America/Chicago")
	now := s.Now
	st.twilio, st.vonage = ID(), ID()
	type prov struct {
		id                   uuid.UUID
		adapter, name, label string
		hint                 string
		config               map[string]string
		creds                map[string]string
		secret               string
		connected            time.Time
		lastReceipt          time.Duration
	}
	provs := []prov{
		{st.twilio, "twilio", "Twilio", "Acme Inc", "AC…3f9", map[string]string{"account_sid": twiliotest.AccountSID, "api_key_sid": twiliotest.KeySID, "mode": "live"},
			map[string]string{"api_key_secret": twiliotest.KeySecret}, TwilioDemoWebhookSecret, s.smsDay(1, 10, 12, chicago), 3 * time.Minute},
		{st.vonage, "vonage", "Vonage", "Acme Inc", "…7c21", map[string]string{"api_key": vonagetest.APIKey},
			map[string]string{"api_secret": vonagetest.APISecret, "signature_secret": vonagetest.SignatureSecret}, "", s.smsDay(2, 11, 40, chicago), 41 * time.Minute},
	}
	for _, p := range provs {
		cfg, _ := json.Marshal(p.config)
		credsJSON, _ := json.Marshal(p.creds)
		creds, err := s.Box.Seal(credsJSON, "sms_provider:"+p.id.String()+":credentials")
		if err != nil {
			return err
		}
		var secret []byte
		if p.secret != "" {
			if secret, err = s.Box.Seal([]byte(p.secret), "sms_provider:"+p.id.String()+":webhook_secret"); err != nil {
				return err
			}
		}
		health, _ := json.Marshal(map[string]any{"lastSuccessAt": now.Add(-time.Minute), "pricesReadAt": s.smsDay(14, 6, 0, chicago),
			"optOutsReadAt": now.Add(-4 * time.Minute)})
		if _, err := s.Tx.Exec(ctx, `INSERT INTO sms_providers (id, workspace_id, adapter, name, account_label, account_hint, config, credentials_enc, webhook_token,
				webhook_secret_enc, webhooks, status, status_since, health, last_receipt_at, last_inbound_at, last_checked_at, registrations_read_at, demo,
				connected_by, connected_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'registered','healthy',$11,$12,$13,$14,$15,$16,true,$17,$11)`,
			p.id, ws.ID, p.adapter, p.name, p.label, p.hint, cfg, creds, smsToken(p.adapter), secret, p.connected, health, now.Add(-p.lastReceipt),
			s.smsDay(14, 10, 15, chicago), now.Add(-time.Minute), s.smsDay(15, 9, 0, chicago), marcus.ID); err != nil {
			return err
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO sms_setups (id, workspace_id, adapter, provider_id, step, status, state, created_by, created_at, updated_at)
			VALUES ($1,$2,$3,$4,'done','finished','{}',$5,$6,$7)`, ID(), ws.ID, p.adapter, p.id, marcus.ID, p.connected.Add(-5*time.Minute), p.connected.Add(20*time.Minute)); err != nil {
			return err
		}
	}
	// Senders.
	st.tollFree, st.tenDLC, st.acme = ID(), ID(), ID()
	senders := []struct {
		id         uuid.UUID
		provider   uuid.UUID
		kind       string
		value      string
		countries  []string
		throughput float64
		ref        string
		added      time.Time
	}{
		{st.tollFree, st.twilio, "toll_free", "+18885550142", []string{"US", "CA"}, 3, "PN0142a1b2c3d4e5f60718293a4b5c6d7e", s.smsDay(1, 10, 20, chicago)},
		{st.tenDLC, st.twilio, "10dlc", "+13125550187", []string{"US"}, 1, "PN0187f6e5d4c3b2a1908172635a4b3c2d", s.smsDay(1, 10, 20, chicago)},
		{st.acme, st.vonage, "sender_id", "ACME", []string{"GB", "DE", "NL", "IE"}, 30, "", s.smsDay(2, 11, 48, chicago)},
	}
	for _, sn := range senders {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO sms_senders (id, workspace_id, provider_id, kind, value, countries, throughput, provider_ref, added_by, added_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, sn.id, ws.ID, sn.provider, sn.kind, sn.value, sn.countries, sn.throughput, sn.ref, marcus.ID, sn.added); err != nil {
			return err
		}
	}
	date := func(day int) time.Time {
		return time.Date(s.smsDay(day, 12, 0, chicago).Year(), s.smsDay(day, 12, 0, chicago).Month(), s.smsDay(day, 12, 0, chicago).Day(), 0, 0, 0, 0, time.UTC)
	}
	readAt := s.smsDay(15, 9, 0, chicago)
	regs := []struct {
		sender              uuid.UUID
		kind, country, name string
		status, source      string
		on                  time.Time
		details             map[string]string
		setBy               *uuid.UUID
		read                bool
	}{
		{st.tollFree, "toll_free", "", "", "verified", "api", date(6), map[string]string{"useCase": "Event reminders and updates for registrants", "volume": "Up to 10,000 texts",
			"optIn": "Web form at news.acme.io/summit · consent text v1 · unchecked by default", "terms": "news.acme.io/sms-terms", "submitted": date(1).Format("2006-01-02"),
			"submittedBy": "Marcus Chen"}, nil, true},
		{st.tenDLC, "brand", "", "Acme Inc", "approved", "api", date(2), map[string]string{}, nil, true},
		{st.tenDLC, "campaign", "", "Acme Summit attendee updates", "pending", "api", date(9), map[string]string{"useCase": "Event reminders and updates for registrants"}, nil, true},
		{st.acme, "sender_id", "GB", "", "allowed", "rule", date(2), map[string]string{}, nil, false},
		{st.acme, "sender_id", "DE", "", "allowed", "rule", date(2), map[string]string{}, nil, false},
		{st.acme, "sender_id", "NL", "", "allowed", "rule", date(2), map[string]string{}, nil, false},
		{st.acme, "sender_id", "IE", "", "pending", "manual", date(9), map[string]string{}, &marcus.ID, false},
	}
	for _, r := range regs {
		details, _ := json.Marshal(r.details)
		var read any
		if r.read {
			read = readAt
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO sms_registrations (id, workspace_id, sender_id, kind, country, name, status, status_on, source, details, set_by, read_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, ID(), ws.ID, r.sender, r.kind, r.country, r.name, r.status, r.on, r.source, details, r.setBy, read,
			r.on.Add(15*time.Hour)); err != nil {
			return err
		}
	}
	// Routes: US and Canada by toll-free; the UK, Germany, the Netherlands
	// and Ireland by the sender ID (pending in Ireland); every other
	// country: don't send.
	for i, rt := range []struct {
		countries []string
		sender    uuid.UUID
	}{{[]string{"US", "CA"}, st.tollFree}, {[]string{"GB", "DE", "NL", "IE"}, st.acme}} {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO sms_routes (id, workspace_id, position, countries, sender_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			ID(), ws.ID, i, rt.countries, rt.sender, s.smsDay(2, 12, 5, chicago), s.smsDay(13, 16, 2, chicago)); err != nil {
			return err
		}
	}
	// Prices per segment, from each provider's pricing on 14 Oct.
	pricedOn := date(14)
	for _, p := range []struct {
		provider uuid.UUID
		country  string
		price    float64
	}{{st.twilio, "US", 0.011}, {st.twilio, "CA", 0.011}, {st.vonage, "GB", 0.045}, {st.vonage, "DE", 0.088}, {st.vonage, "NL", 0.096}, {st.vonage, "IE", 0.076}} {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO sms_prices (workspace_id, provider_id, country, price, source, as_of) VALUES ($1,$2,$3,$4,'api',$5)`,
			ws.ID, p.provider, p.country, p.price, pricedOn); err != nil {
			return err
		}
	}
	// Opt-out sync: neither API lists the account's opt-outs, so both learn
	// them from replies and refused texts (the adapters' manifests).
	for _, o := range []struct {
		provider     uuid.UUID
		mode         string
		pulled, push int
	}{{st.twilio, "read_only", 0, 0}, {st.vonage, "read_only", 0, 0}} {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO sms_opt_out_sync (provider_id, workspace_id, mode, pulled, pushed, status, last_sync_at) VALUES ($1,$2,$3,$4,$5,'in_sync',$6)`,
			o.provider, ws.ID, o.mode, o.pulled, o.push, now.Add(-4*time.Minute)); err != nil {
			return err
		}
	}
	return nil
}

// smsRules writes the SMS rules Marcus set up, Linh's budget, the test
// numbers and the text message terms page.
func smsRules(ctx context.Context, s *Seeder, st *smsState) error {
	ws := s.WS()
	marcus, linh, sofia := s.Users["marcus@acme.io"], s.Users["linh@acme.io"], s.Users["sofia@acme.io"]
	chicago := smsLoc("America/Chicago")
	// The approval policy's rule for texts (SmsReview: "the cost is under the
	// $50 rule for SMS and WhatsApp").
	if _, err := s.Tx.Exec(ctx, `UPDATE workspaces SET approval_policy = jsonb_set(approval_policy, '{rules,channel_cost_min}', '{"enabled":true,"amount":50}'::jsonb, true)
		WHERE id = $1 AND approval_policy ? 'rules'`, ws.ID); err != nil {
		return err
	}
	st.help = "Acme: Help at acme.io/sms or support@acme.io. Msg & data rates may apply. Reply STOP to cancel."
	st.optOut = "Acme: You're unsubscribed from Acme texts and won't get more. Reply START to resubscribe."
	st.confirmation = "Acme: You're signed up for Acme Summit 2026 texts. Up to 4 msgs. Msg & data rates may apply. Reply HELP for help, STOP to cancel."
	if _, err := s.Tx.Exec(ctx, `INSERT INTO sms_settings (workspace_id, brand_prefix, opt_out_text, opt_out_in_steps, confirmation_on, confirmation_text, require_yes,
			quiet_from, quiet_to, help_reply, opt_out_reply, opt_in_reply, keywords_by, budget, budget_alert_pct, budget_set_by, budget_set_at,
			pause_failed_pct, pause_filtered_pct, pause_opt_out_pct, other_countries, retention_days, short_domain, short_domain_status, short_domain_target,
			short_domain_checked_at, short_domain_verified_at, short_domain_detail, updated_by, updated_at)
		VALUES ($1,'Acme:','Reply STOP to opt out',true,true,$2,false,540,1200,$3,$4,$5,'reputecast',100,80,$6,$7,5,2,3,'dont_send',30,'go.acme.io','verified',
			'reputecast.acme.io',$8,$9,'CNAME points at reputecast.acme.io.',$10,$11)`,
		ws.ID, st.confirmation, st.help, st.optOut, "Acme: You're subscribed to Acme texts again. Reply HELP for help, STOP to cancel.",
		linh.ID, s.smsDay(1, 9, 30, chicago), s.Now.Add(-25*time.Minute), s.smsDay(3, 15, 10, chicago), marcus.ID, s.smsDay(13, 16, 2, chicago)); err != nil {
		return err
	}
	for _, t := range []struct {
		user  uuid.UUID
		phone string
		added time.Time
	}{{marcus.ID, "+13125550161", s.smsDay(1, 10, 25, chicago)}, {sofia.ID, "+447700900456", s.smsDay(2, 12, 30, chicago)}} {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO sms_test_numbers (id, workspace_id, user_id, phone, added_by, added_at) VALUES ($1,$2,$3,$4,$5,$6)`,
			ID(), ws.ID, t.user, t.phone, marcus.ID, t.added); err != nil {
			return err
		}
	}
	terms, _ := json.Marshal(map[string]string{
		"intro":      "These terms cover the texts you agree to when you tick the SMS box while registering for Acme Summit 2026. Email and texts are separate: one never signs you up for the other.",
		"program":    "Acme texts: reminders and updates for Acme Summit 2026.",
		"frequency":  "Up to 4 messages per event.",
		"cost":       "Message and data rates may apply.",
		"help":       "Text HELP to +1 888 555 0142, or email support@acme.io.",
		"optOut":     "Text STOP at any time. We'll confirm once and send nothing more.",
		"senders":    "+1 888 555 0142 in the US and Canada; ACME in the UK, Germany, the Netherlands and Ireland.",
		"privacy":    "We never sell or share your number for marketing.",
		"carriers":   "Carriers aren't liable for delayed or undelivered messages.",
		"privacyUrl": "https://news.acme.io/privacy",
		"helpNumber": "+1 888 555 0142",
		"helpEmail":  "support@acme.io",
		"address":    "Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105",
	})
	_, err := s.Tx.Exec(ctx, `INSERT INTO sms_terms_versions (workspace_id, version, content, published_by, published_at) VALUES ($1,1,$2,$3,$4)`,
		ws.ID, terms, marcus.ID, s.smsDay(1, 11, 0, chicago))
	return err
}

// smsCampaignRows creates the list, the summit date field and the two SMS
// campaigns: Sofia's sent "Summit: 8 days to go" and Daniel's draft.
func smsCampaignRows(ctx context.Context, s *Seeder, st *smsState) error {
	ws := s.WS()
	sofia, daniel := s.Users["sofia@acme.io"], s.Users["daniel@acme.io"]
	chicago := smsLoc("America/Chicago")
	st.summit = ID()
	if _, err := s.Tx.Exec(ctx, `INSERT INTO lists (id, workspace_id, name, description, double_opt_in, public, created_at) VALUES ($1,$2,'Acme Summit 2026',$3,false,false,$4)`,
		st.summit, ws.ID, "Registrants of Acme Summit 2026, from news.acme.io/summit.", s.smsDay(1, 9, 0, chicago)); err != nil {
		return err
	}
	s.Lists["Acme Summit 2026"] = st.summit
	if _, err := s.Tx.Exec(ctx, `INSERT INTO custom_fields (id, workspace_id, key, label, type, options, created_at) VALUES ($1,$2,'summit_date','Summit date','date','[]',$3)`,
		ID(), ws.ID, s.smsDay(1, 9, 5, chicago)); err != nil {
		return err
	}
	st.summitDate = s.smsDay(22, 9, 0, chicago)
	day := st.summitDate.Format("2 Jan")
	st.body = "8 days to Acme Summit 2026! The agenda is live: acme.io/summit/agenda See you " + day + " in Chicago or online."
	audience, _ := json.Marshal(map[string]any{"lists": []string{st.summit.String()}})
	full := "Acme: " + st.body + " Reply STOP to opt out"
	sum := sha256.Sum256([]byte(full))
	st.campaign = ID()
	created := s.smsDay(13, 15, 12, chicago)
	scheduled := s.smsDay(14, 10, 0, smsLoc("Europe/Berlin")) // 08:00 UTC: Germany and the Netherlands first
	done := s.smsDay(14, 10, 2, smsLoc("America/Los_Angeles"))
	if _, err := s.Tx.Exec(ctx, `INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, audience, editor_mode, mjml, html, plain_text, version, channel,
			content_hash, created_by, updated_by, created_at, updated_at, autosaved_at, scheduled_at, started_at, started_by, completed_at, audience_count)
		VALUES ($1,$2,'Summit: 8 days to go','sent','','',$3,'html','','','',3,'sms',$4,$5,$5,$6,$7,$7,$8,$8,$5,$9,854)`,
		st.campaign, ws.ID, audience, hex.EncodeToString(sum[:])[:32], sofia.ID, created, s.smsDay(13, 16, 20, chicago), scheduled, done); err != nil {
		return err
	}
	s.Campaigns["Summit: 8 days to go"] = st.campaign
	localDate := time.Date(scheduled.Year(), scheduled.Month(), scheduled.Day(), 0, 0, 0, 0, time.UTC)
	if _, err := s.Tx.Exec(ctx, `INSERT INTO sms_campaigns (campaign_id, workspace_id, body, schedule_mode, local_date, local_minute, pending_policy)
		VALUES ($1,$2,$3,'local',$4,600,'skip')`, st.campaign, ws.ID, st.body, localDate); err != nil {
		return err
	}
	for v, b := range []string{
		"8 days to Acme Summit 2026! The agenda is live: acme.io/summit/agenda",
		"8 days to Acme Summit 2026! The agenda is live: acme.io/summit/agenda See you in Chicago or online.",
		st.body,
	} {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO sms_campaign_versions (campaign_id, version, body, saved_by, saved_at) VALUES ($1,$2,$3,$4,$5)`,
			st.campaign, v+1, b, sofia.ID, created.Add(time.Duration(v*25)*time.Minute)); err != nil {
			return err
		}
	}
	leftOut, _ := json.Marshal(map[string]any{"base": 1240, "noConsent": 359, "suppressed": 7, "reasons": map[string]int{"landline": 5, "invalid": 2}, "shared": 0,
		"canReceive": 874, "cost": 19.535})
	thresholds, _ := json.Marshal(map[string]float64{"failed": 5, "filtered": 2, "optOuts": 3})
	if _, err := s.Tx.Exec(ctx, `INSERT INTO sms_sends (campaign_id, workspace_id, phase, recipients, skipped, left_out, started_by, scheduled_at, first_send_at, last_send_at,
			thresholds, completed_at)
		VALUES ($1,$2,'completed',854,20,$3,$4,$5,$5,$6,$7,$6)`, st.campaign, ws.ID, leftOut, sofia.ID, scheduled, done, thresholds); err != nil {
		return err
	}
	// Daniel's draft: a curly apostrophe makes it UCS-2, 2 segments.
	st.draft = ID()
	draftBody := "Doors open at 8:30 tomorrow. You’ll find badges at the north entrance."
	dfull := "Acme: " + draftBody + " Reply STOP to opt out"
	dsum := sha256.Sum256([]byte(dfull))
	dcreated := s.smsDay(15, 8, 40, chicago)
	if _, err := s.Tx.Exec(ctx, `INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, audience, editor_mode, mjml, html, plain_text, version, channel,
			content_hash, created_by, updated_by, created_at, updated_at, autosaved_at)
		VALUES ($1,$2,'Summit: doors open','draft','','',$3,'html','','','',2,'sms',$4,$5,$5,$6,$7,$7)`,
		st.draft, ws.ID, audience, hex.EncodeToString(dsum[:])[:32], daniel.ID, dcreated, s.Now.Add(-12*time.Minute)); err != nil {
		return err
	}
	s.Campaigns["Summit: doors open"] = st.draft
	ddate := time.Date(st.summitDate.Year(), st.summitDate.Month(), st.summitDate.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	if _, err := s.Tx.Exec(ctx, `INSERT INTO sms_campaigns (campaign_id, workspace_id, body, schedule_mode, local_date, local_minute, pending_policy)
		VALUES ($1,$2,$3,'local',$4,1080,'skip')`, st.draft, ws.ID, draftBody, ddate); err != nil {
		return err
	}
	for v, b := range []string{"Doors open at 8:30 tomorrow.", draftBody} {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO sms_campaign_versions (campaign_id, version, body, saved_by, saved_at) VALUES ($1,$2,$3,$4,$5)`,
			st.draft, v+1, b, daniel.ID, dcreated.Add(time.Duration(v*30)*time.Minute)); err != nil {
			return err
		}
	}
	return nil
}

// smsAutomation adds "Acme Summit reminders", live since 9 Oct, built by
// Sofia Rossi: an email the day before, then a text to those with SMS
// consent. (Push and WhatsApp steps come with those channels.)
func smsAutomation(ctx context.Context, s *Seeder, st *smsState) error {
	ws := s.WS()
	sofia := s.Users["sofia@acme.io"]
	chicago := smsLoc("America/Chicago")
	created := s.smsDay(8, 15, 30, chicago)
	emailID := ID()
	if _, err := s.Tx.Exec(ctx, `INSERT INTO automation_emails (id, workspace_id, name, latest_version, created_by, created_at, updated_at) VALUES ($1,$2,$3,1,$4,$5,$5)`,
		emailID, ws.ID, `Summit "See you tomorrow at Acme Summit"`, sofia.ID, created); err != nil {
		return err
	}
	c := automations.DemoEmail("See you tomorrow at Acme Summit", "Doors open at 8:30 at Lakeside Hall, and online.", "See you tomorrow",
		"See the agenda", "Acme Summit 2026 starts tomorrow at 09:00 Central, at Lakeside Hall in Chicago and online.", "Badges are at the north entrance.")
	if _, err := s.Tx.Exec(ctx, `INSERT INTO automation_email_versions (email_id, version, subject, preheader, mjml, html, plain_text, created_by, created_at) VALUES ($1,1,$2,$3,$4,$5,$6,$7,$8)`,
		emailID, c.Subject, c.Preheader, c.MJML, c.HTML, c.Text, sofia.ID, created); err != nil {
		return err
	}
	g := automations.AutomationGraph{
		Nodes: []autoFlowNode{
			autoNode("t1", automations.NodeTrigger, 300, 20, autoFlowData{Trigger: &automations.AutomationTrigger{Kind: automations.TriggerDateField, Field: "summit_date",
				OffsetDays: -1, At: "09:00", Reentry: "once"}}),
			autoNode("e1", automations.NodeEmail, 300, 150, autoFlowData{Email: &automations.AutomationEmailStep{EmailID: emailID.String(), EmailVersion: 1,
				Subject: "See you tomorrow at Acme Summit"}}),
			autoNode("b1", automations.NodeBranch, 300, 290, autoFlowData{Condition: &automations.AutomationCondition{Kind: automations.CondSMSConsent, ListID: st.summit.String()}}),
			autoNode("s1", automations.NodeSMS, 120, 430, autoFlowData{Label: "Day-before reminder", SMS: &automations.AutomationSMSStep{
				Body: "See you tomorrow! Doors open 8:30 at Lakeside Hall. Livestream: acme.io/summit/live", ConsentListID: st.summit.String(), Outside: "wait"}}),
			autoNode("d1", automations.NodeDelay, 120, 580, autoFlowData{Delay: autoDur(1425, "minutes")}),
			autoNode("x1", automations.NodeExit, 120, 700, autoFlowData{}),
			autoNode("x2", automations.NodeExit, 480, 430, autoFlowData{Label: "Email only"}),
		},
		Edges: []automations.AutomationEdge{
			autoEdge("t1", "e1", ""), autoEdge("e1", "b1", ""), autoEdge("b1", "s1", automations.PathYes), autoEdge("b1", "x2", automations.PathNo),
			autoEdge("s1", "d1", ""), autoEdge("d1", "x1", ""),
		},
	}
	st.automation = ID()
	published := s.smsDay(9, 11, 5, chicago)
	return autoInsertFlow(ctx, s, autoFlowSeed{id: st.automation, name: "Acme Summit reminders", description: "The day before Acme Summit 2026: an email, then a text to registrants with SMS consent.",
		status: "live", live: 1, createdBy: sofia.ID, created: created, updated: published,
		versions: []autoVersionSeed{{graph: g, by: sofia.ID, created: created, published: &published, summary: "First version"}}})
}

// smsAudit records who did what: connections, routes and rules, the
// budget, the terms page, Ireland's registration, the send.
func smsAudit(ctx context.Context, s *Seeder, st *smsState) error {
	ws := s.WS()
	chicago := smsLoc("America/Chicago")
	m := func(kv ...any) map[string]any {
		out := map[string]any{}
		for i := 0; i+1 < len(kv); i += 2 {
			out[kv[i].(string)] = kv[i+1]
		}
		return out
	}
	events := []auditSeed{
		{at: s.smsDay(1, 10, 12, chicago), actorType: "user", actor: "marcus@acme.io", action: "sms_provider.connected", targetType: "sms_provider", targetID: st.twilio.String(),
			target: "Twilio", ip: "203.0.113.9", ua: uaChromeMac, after: m("adapter", "twilio", "account", "Acme Inc", "hint", "AC…3f9", "webhooks", "registered", "live", true)},
		{at: s.smsDay(1, 9, 30, chicago), actorType: "user", actor: "linh@acme.io", action: "sms_rules.updated", targetType: "workspace", targetID: ws.ID.String(), target: ws.Name,
			ip: "203.0.113.9", ua: uaChromeMac, before: m("budget", 0), after: m("budget", 100, "budget_alert_pct", 80)},
		{at: s.smsDay(1, 11, 0, chicago), actorType: "user", actor: "marcus@acme.io", action: "sms_terms.published", targetType: "workspace", targetID: ws.ID.String(), target: ws.Name,
			ip: "203.0.113.9", ua: uaChromeMac, before: m("version", 0), after: m("version", 1)},
		{at: s.smsDay(2, 11, 40, chicago), actorType: "user", actor: "marcus@acme.io", action: "sms_provider.connected", targetType: "sms_provider", targetID: st.vonage.String(),
			target: "Vonage", ip: "203.0.113.9", ua: uaChromeMac, after: m("adapter", "vonage", "account", "Acme Inc", "hint", "…7c21", "webhooks", "registered", "live", true)},
		{at: s.smsDay(2, 11, 48, chicago), actorType: "user", actor: "marcus@acme.io", action: "sms_sender.added", targetType: "sms_sender", targetID: st.acme.String(), target: "ACME",
			ip: "203.0.113.9", ua: uaChromeMac, after: m("kind", "sender_id", "provider", "Vonage", "countries", []string{"GB", "DE", "NL", "IE"})},
		{at: s.smsDay(3, 15, 10, chicago), actorType: "user", actor: "marcus@acme.io", action: "sms_short_domain.checked", targetType: "workspace", targetID: ws.ID.String(),
			target: ws.Name, ip: "203.0.113.9", ua: uaChromeMac, before: m("status", "pending"), after: m("status", "verified", "domain", "go.acme.io")},
		{at: s.smsDay(9, 10, 30, chicago), actorType: "user", actor: "marcus@acme.io", action: "sms_registration.set", targetType: "sms_sender", targetID: st.acme.String(), target: "ACME",
			ip: "203.0.113.9", ua: uaChromeMac, before: m("status", "not_registered"), after: m("kind", "sender_id", "country", "IE", "status", "pending", "on", s.smsDay(9, 12, 0, chicago).Format("2006-01-02"))},
		{at: s.smsDay(13, 16, 2, chicago), actorType: "user", actor: "marcus@acme.io", action: "sms_routes.updated", targetType: "workspace", targetID: ws.ID.String(), target: ws.Name,
			ip: "203.0.113.9", ua: uaChromeMac, after: m("routes", []map[string]any{{"countries": []string{"US", "CA"}, "sender": "Twilio · +1 888 555 0142"},
				{"countries": []string{"GB", "DE", "NL", "IE"}, "sender": "Vonage · ACME"}}, "other", "dont_send")},
		{at: s.smsDay(13, 16, 20, chicago), actorType: "user", actor: "sofia@acme.io", action: "campaign.scheduled", targetType: "campaign", targetID: st.campaign.String(),
			target: "Summit: 8 days to go", ip: "198.51.100.23", ua: uaChromeMac, before: m("status", "draft"), after: m("status", "scheduled"),
			meta: m("channel", "sms", "recipients", 854, "skipped", 20, "cost", 19.54, "schedule", "14 Oct at 10:00 in each recipient's time zone")},
	}
	for _, e := range events {
		if err := s.insertAudit(ctx, ws.ID, e); err != nil {
			return err
		}
	}
	return nil
}

// smsCopy writes rows with COPY.
func smsCopy(ctx context.Context, s *Seeder, table string, cols []string, rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}
	_, err := s.Tx.CopyFrom(ctx, pgx.Identifier{table}, cols, pgx.CopyFromRows(rows))
	if err != nil {
		return fmt.Errorf("%s: %w", table, err)
	}
	return nil
}
