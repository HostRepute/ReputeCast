package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/whatsapp"
	"github.com/hostrepute/reputecast/internal/whatsapp/meta"
	"github.com/hostrepute/reputecast/internal/whatsapp/meta/metatest"
)

// The WhatsApp part of the X22 seed.

// WhatsAppConsentText is the registration form's WhatsApp consent, v1.
const WhatsAppConsentText = "Send me reminders about Acme Summit 2026 on WhatsApp. Up to 3 messages. Reply STOP to stop."

// whatsAppAccount connects Acme Inc (8 Oct, Marcus Chen) with its templates
// as synced 15 Oct 08:00, Meta's rate card and the settings.
func whatsAppAccount(ctx context.Context, s *Seeder, st *pwState) error {
	ws := s.WS()
	marcus := s.Users["marcus@acme.io"]
	connected, verified := s.pushDay(8, 9, 2), s.pushDay(8, 9, 14)
	st.account = ID()
	creds, _ := json.Marshal(meta.Credentials{BusinessAccountID: WhatsAppBusinessID, PhoneNumberID: WhatsAppPhoneID, AccessToken: WhatsAppToken, AppSecret: WhatsAppAppSecret}) //nolint:gosec // G117: the demo's, sealed at once
	sealed, err := s.Box.Seal(creds, "whatsapp_account:"+st.account.String()+":credentials")
	if err != nil {
		return err
	}
	synced := s.pushDay(15, 8, 0)
	if _, err := s.Tx.Exec(ctx, `INSERT INTO whatsapp_accounts (id, workspace_id, credentials_enc, business_hint, business_name, phone_number, display_name, name_status,
			phone_status, quality, messaging_limit, verify_token, verified_at, last_event_at, status, status_since, templates_synced_at, checked_at, demo,
			connected_by, connected_at)
		VALUES ($1,$2,$3,'…4410','Acme Inc','+13125550110','Acme','approved','connected','high',1000,'rc-wa-7Q2mX9vKp4',$4,$5,'connected',$4,$6,$7,true,$8,$9)`,
		st.account, ws.ID, sealed, verified, s.Ago(4*time.Minute), synced, s.Ago(4*time.Minute), marcus.ID, connected); err != nil {
		return err
	}
	type tv struct{ key, label, example string }
	labels := map[string][]tv{
		"summit_day_before":   {{"1", "first name", "Sam"}, {"2", "doors-open time", "8:30"}},
		"summit_livestream":   {{"1", "livestream link", "go.acme.io/k3P9xq"}},
		"summit_venue_change": {{"1", "new venue", "Harbor Hall, Chicago"}},
		"digest_highlights":   {{"1", "headline", ""}, {"2", "post link", ""}},
	}
	since := map[string]time.Time{"summit_day_before": s.pushDay(9, 10, 40), "summit_livestream": s.pushDay(10, 15, 12),
		"summit_venue_change": s.pushDay(14, 11, 2), "digest_highlights": s.pushDay(13, 16, 20)}
	for _, t := range WhatsAppTemplates() {
		vars := []map[string]string{}
		for _, v := range labels[t.Name] {
			vars = append(vars, map[string]string{"key": v.key, "label": v.label, "example": v.example})
		}
		buttons := []map[string]string{}
		for _, b := range t.Buttons() {
			typ := "url"
			if b.Type == "QUICK_REPLY" {
				typ = "quick_reply"
			}
			buttons = append(buttons, map[string]string{"type": typ, "text": b.Text, "url": b.URL})
		}
		status, code, reason := map[string]string{"APPROVED": "approved", "PENDING": "pending", "REJECTED": "rejected"}[t.Status], "", ""
		if status == "rejected" {
			code, reason = "INVALID_FORMAT", "Add sample values for every variable."
		}
		category := map[string]string{"UTILITY": "utility", "MARKETING": "marketing"}[t.Category]
		id := ID()
		st.templates[t.Name] = id
		vj, _ := json.Marshal(vars)
		bj, _ := json.Marshal(buttons)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO whatsapp_templates (id, workspace_id, account_id, meta_id, name, language, category, status, status_since, reason_code,
				reason, body, variables, buttons, synced_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
			id, ws.ID, st.account, t.ID, t.Name, t.Language, category, status, since[t.Name], code, reason, t.Body(), vj, bj, synced); err != nil {
			return err
		}
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO whatsapp_settings (workspace_id, rates_as_of, quiet_hours, updated_at) VALUES ($1,$2,true,$3)`,
		ws.ID, whatsapp.RatesAsOf, connected); err != nil {
		return err
	}
	rows := [][]any{}
	for _, r := range whatsapp.DefaultRates {
		rows = append(rows, []any{ws.ID, r.Country, r.Category, r.Price})
	}
	return smsCopy(ctx, s, "whatsapp_rates", []string{"workspace_id", "country", "category", "price"}, rows)
}

// whatsAppPeople gives 214 registrants WhatsApp consent, Sam Rivera his
// conversation, Sofia her draft, and the Events tab what Meta posted.
func whatsAppPeople(ctx context.Context, s *Seeder, st *pwState) error {
	ws := s.WS()
	sofia := s.Users["sofia@acme.io"]
	// Sam first, then registrants in the US and Canada who can get texts.
	rows, err := s.Tx.Query(ctx, `SELECT c.id, c.phone, lm.subscribed_at FROM channel_consents cc
		JOIN contacts c ON c.id = cc.contact_id
		JOIN list_memberships lm ON lm.contact_id = c.id AND lm.list_id = cc.list_id
		WHERE cc.list_id = $1 AND cc.channel = 'sms' AND cc.status = 'subscribed' AND c.phone_country IN ('US','CA')
		  AND NOT EXISTS (SELECT 1 FROM channel_suppressions x WHERE x.workspace_id = c.workspace_id AND x.channel = 'sms' AND x.phone = c.phone AND x.removed_at IS NULL)
		ORDER BY (c.id = $2) DESC, c.email LIMIT 214`, st.summit, st.sam)
	if err != nil {
		return err
	}
	type person struct {
		id    uuid.UUID
		phone string
		at    *time.Time
	}
	people, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (person, error) {
		var p person
		return p, r.Scan(&p.id, &p.phone, &p.at)
	})
	if err != nil {
		return err
	}
	if len(people) != 214 || people[0].id != st.sam {
		return fmt.Errorf("WhatsApp consent: %d registrants", len(people))
	}
	consents, events := [][]any{}, [][]any{}
	for i, p := range people {
		at := s.pushDay(2, 18, 0)
		if p.at != nil {
			at = *p.at
		}
		ip := netip.AddrFrom4([4]byte{198, 51, 100, byte(10 + i%240)})
		consents = append(consents, []any{ws.ID, st.summit, p.id, "whatsapp", "subscribed", "form:summit", "", at})
		events = append(events, []any{ID(), ws.ID, p.id, &st.summit, "whatsapp", "subscribed", p.phone, "form:summit", "news.acme.io/summit · registration page",
			WhatsAppConsentText, "1", &ip, at})
	}
	if err := smsCopy(ctx, s, "channel_consents", []string{"workspace_id", "list_id", "contact_id", "channel", "status", "source", "keyword", "changed_at"},
		consents); err != nil {
		return err
	}
	if err := smsCopy(ctx, s, "channel_consent_events", []string{"id", "workspace_id", "contact_id", "list_id", "channel", "action", "phone", "source", "page",
		"consent_text", "consent_version", "ip", "occurred_at"}, events); err != nil {
		return err
	}
	// Sam's conversation: the registration message (2 Oct 19:30, read),
	// and his question, which opened the 24-hour window. The window stays
	// open while the demo is fresh, as on the board.
	sentAt := s.pushDay(2, 19, 30)
	msg := ID()
	wamid := "wamid.HBgLMTMxMjU1NTAxMjMVAgARGBI5QUQ2RDk0RjI3MDBFNDc0QzMA"
	price := 0.004
	if _, err := s.Tx.Exec(ctx, `INSERT INTO channel_messages (id, workspace_id, channel, kind, contact_id, list_id, recipient, country, time_zone, zone_source, body,
			units, price, cost, status, provider_message_id, send_after, created_at, sent_at, delivered_at, updated_at)
		VALUES ($1,$2,'whatsapp','automation',$3,$4,$5,'US','America/Chicago','contact',$6,1,$7,$7,'delivered',$8,$9,$9,$9,$10,$10)`,
		msg, ws.ID, st.sam, st.summit, st.samPhone, "Hi Sam, thanks for registering for Acme Summit 2026 on 22 Oct. Reply here with any question.", price, wamid,
		sentAt, sentAt.Add(3*time.Second)); err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO whatsapp_messages (message_id, workspace_id, account_id, kind, template, language, category, params, read_at, billable)
		VALUES ($1,$2,$3,'template','summit_registration','en_US','utility','["Sam"]',$4,true)`, msg, ws.ID, st.account, sentAt.Add(11*time.Minute)); err != nil {
		return err
	}
	wrote := s.pushDay(14, 18, 2)
	if s.Now.Sub(wrote) > 22*time.Hour {
		wrote = s.Ago(15*time.Hour + 50*time.Minute)
	}
	inID := "wamid.in.HBgLMTMxMjU1NTAxMjMVAgASGBQzQTk2NjE3NUQ4QTk2RjU2QzNGNwA="
	if _, err := s.Tx.Exec(ctx, `INSERT INTO whatsapp_inbound (id, workspace_id, account_id, phone, name, body, kind, message_id, contact_ids, effect, meta_id, received_at)
		VALUES ($1,$2,$3,$4,'Sam Rivera','Is there parking at the venue?','reply',$5,$6,'{"contacts":1}',$7,$8)`,
		ID(), ws.ID, st.account, st.samPhone, msg, []uuid.UUID{st.sam}, inID, wrote); err != nil {
		return err
	}
	// Sofia's draft: summit_day_before, 21 Oct at 09:00 in each recipient's
	// time zone, autosaved today at 09:41.
	st.waDraft = ID()
	vars, _ := json.Marshal([]map[string]string{{"key": "1", "source": "field", "field": "first_name", "fallback": "there"}, {"key": "2", "source": "text", "text": "8:30"}})
	audience, _ := json.Marshal(map[string]any{"lists": []string{st.summit.String()}, "segments": []string{}})
	created, saved := s.pushDay(15, 9, 12), s.pushDay(15, 9, 41)
	tid := st.templates["summit_day_before"]
	if _, err := s.Tx.Exec(ctx, `INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, audience, editor_mode, mjml, html, plain_text, version, channel,
			content_hash, created_by, updated_by, created_at, updated_at, autosaved_at)
		VALUES ($1,$2,'Summit: see you tomorrow','draft','','',$3,'html','','','',3,'whatsapp',$4,$5,$5,$6,$7,$7)`,
		st.waDraft, ws.ID, audience, "demo-wa-day-before", sofia.ID, created, saved); err != nil {
		return err
	}
	day := s.pushDay(21, 12, 0)
	localDate := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	if _, err := s.Tx.Exec(ctx, `INSERT INTO whatsapp_campaigns (campaign_id, workspace_id, template_id, template_name, template_language, variables, schedule_mode,
			local_date, local_minute, over_limit) VALUES ($1,$2,$3,'summit_day_before','en_US',$4,'local',$5,540,'next_day')`,
		st.waDraft, ws.ID, tid, vars, localDate); err != nil {
		return err
	}
	return whatsAppEvents(ctx, s, st, msg, wamid, sentAt, wrote)
}

// whatsAppEvents are Meta's webhooks as the Events tab lists them, each
// signed with the app secret, stored as Meta sent it and normalized.
func whatsAppEvents(ctx context.Context, s *Seeder, st *pwState, msg uuid.UUID, wamid string, sentAt, wrote time.Time) error {
	ws := s.WS()
	type ev struct {
		at      time.Time
		body    []byte
		sig     string
		summary string
	}
	evs := []ev{}
	add := func(at time.Time, summary string, body []byte, sig string) {
		evs = append(evs, ev{at, body, sig, summary})
	}
	for _, x := range []struct {
		status string
		after  time.Duration
	}{{"sent", time.Second}, {"delivered", 3 * time.Second}, {"read", 11 * time.Minute}} {
		b, sig := metatest.StatusWebhook(WhatsAppAppSecret, WhatsAppBusinessID, WhatsAppPhoneID, wamid, st.samPhone, x.status, sentAt.Add(x.after), 0)
		add(sentAt.Add(x.after+time.Second), "statuses: "+x.status, b, sig)
	}
	b, sig := metatest.TemplateWebhook(WhatsAppAppSecret, WhatsAppBusinessID, "1290033512244410", "summit_day_before", "en_US", "APPROVED", "NONE", "")
	add(s.pushDay(9, 10, 40), "message_template_status_update: summit_day_before approved", b, sig)
	b, sig = metatest.TemplateWebhook(WhatsAppAppSecret, WhatsAppBusinessID, "1290033512244411", "summit_livestream", "en_US", "APPROVED", "NONE", "")
	add(s.pushDay(10, 15, 12), "message_template_status_update: summit_livestream approved", b, sig)
	b, sig = metatest.TemplateWebhook(WhatsAppAppSecret, WhatsAppBusinessID, "1290033512244413", "digest_highlights", "en_US", "REJECTED", "INVALID_FORMAT",
		"Add sample values for every variable.")
	add(s.pushDay(13, 16, 20), "message_template_status_update: digest_highlights rejected", b, sig)
	b, sig = metatest.MessageWebhook(WhatsAppAppSecret, WhatsAppBusinessID, WhatsAppPhoneID, st.samPhone, "Sam Rivera", "Is there parking at the venue?", wrote)
	add(wrote.Add(time.Second), "messages: text", b, sig)
	rows := [][]any{}
	for _, e := range evs {
		headers, _ := json.Marshal(map[string]string{"Content-Type": "application/json", "User-Agent": "facebookexternalua", "X-Hub-Signature-256": e.sig})
		done := e.at.Add(400 * time.Millisecond)
		rows = append(rows, []any{ID(), ws.ID, &st.account, e.at, headers, e.body, true, e.summary, &done, int32(1)})
	}
	_ = msg
	return smsCopy(ctx, s, "whatsapp_raw_events", []string{"id", "workspace_id", "account_id", "received_at", "headers", "body", "signature_ok", "summary",
		"normalized_at", "events"}, rows)
}

// pwAudit records who did what: Marcus's keys, sites and WhatsApp; Sofia's
// prompts, the push campaign and the WhatsApp draft; the system's syncs
// and the send.
func pwAudit(ctx context.Context, s *Seeder, st *pwState) error {
	ws := s.WS()
	m := func(kv ...any) map[string]any {
		out := map[string]any{}
		for i := 0; i+1 < len(kv); i += 2 {
			out[kv[i].(string)] = kv[i+1]
		}
		return out
	}
	var pub string
	if err := s.Tx.QueryRow(ctx, `SELECT public_key FROM push_keys WHERE id = $1`, st.key).Scan(&pub); err != nil {
		return err
	}
	ff := "New post: Feature flags at scale"
	events := []auditSeed{
		{at: s.pushDay(3, 10, 14), actorType: "user", actor: "marcus@acme.io", action: "push_keys.created", targetType: "push_keys", targetID: st.key.String(),
			target: "VAPID keys", ip: "203.0.113.9", ua: uaChromeMac, after: m("public_key", pub, "contact", "mailto:support@acme.io")},
		{at: s.pushDay(3, 10, 14), actorType: "user", actor: "marcus@acme.io", action: "push_site.added", targetType: "push_site", targetID: st.news.String(),
			target: "news.acme.io", ip: "203.0.113.9", ua: uaChromeMac, after: m("origin", "https://news.acme.io", "kind", "publication", "label", "Publication site")},
		{at: s.pushDay(3, 10, 40), actorType: "user", actor: "sofia@acme.io", action: "form.create", targetType: "form", targetID: st.newsPrompt.String(),
			target: "Get new posts as notifications", ip: "198.51.100.23", ua: uaChromeMac, after: m("kind", "push", "site", "news.acme.io")},
		{at: s.pushDay(3, 11, 2), actorType: "user", actor: "sofia@acme.io", action: "push_prompt.published", targetType: "form", targetID: st.newsPrompt.String(),
			target: "Get new posts as notifications", ip: "198.51.100.23", ua: uaChromeMac, before: m("version", 0), after: m("version", 1, "style", "bar")},
		{at: s.pushDay(5, 10, 52), actorType: "user", actor: "marcus@acme.io", action: "push_site.added", targetType: "push_site", targetID: st.acmeSite.String(),
			target: "acme.io", ip: "203.0.113.9", ua: uaChromeMac, after: m("origin", "https://acme.io", "kind", "snippet", "label", "Company site")},
		{at: s.pushDay(5, 11, 10), actorType: "user", actor: "sofia@acme.io", action: "form.create", targetType: "form", targetID: st.bellPrompt.String(),
			target: "Release notes bell", ip: "198.51.100.23", ua: uaChromeMac, after: m("kind", "push", "site", "acme.io")},
		{at: s.pushDay(5, 11, 34), actorType: "user", actor: "sofia@acme.io", action: "push_prompt.published", targetType: "form", targetID: st.bellPrompt.String(),
			target: "Release notes bell", ip: "198.51.100.23", ua: uaChromeMac, before: m("version", 0), after: m("version", 1, "style", "bell")},
		{at: s.pushDay(6, 14, 18), actorType: "user", actor: "sofia@acme.io", action: "push_prompt.published", targetType: "form", targetID: st.newsPrompt.String(),
			target: "Get new posts as notifications", ip: "198.51.100.23", ua: uaChromeMac, before: m("version", 1), after: m("version", 2, "style", "bar")},
		{at: s.pushDay(8, 9, 2), actorType: "user", actor: "marcus@acme.io", action: "whatsapp.connected", targetType: "whatsapp_account", targetID: st.account.String(),
			target: "Acme Inc", ip: "203.0.113.9", ua: uaChromeMac, after: m("business_account", "…4410", "number", "+13125550110", "display_name", "Acme")},
		{at: s.pushDay(8, 9, 3), actorType: "user", actor: "marcus@acme.io", action: "whatsapp_templates.synced", targetType: "whatsapp_account",
			targetID: st.account.String(), target: "Acme Inc", ip: "203.0.113.9", ua: uaChromeMac, meta: m("templates", 3, "added", []string{"summit_day_before",
				"summit_livestream", "digest_highlights"}, "changed", []string{}, "removed", []string{})},
		{at: s.pushDay(8, 10, 31), actorType: "user", actor: "sofia@acme.io", action: "push_prompt.published", targetType: "form", targetID: st.newsPrompt.String(),
			target: "Get new posts as notifications", ip: "198.51.100.23", ua: uaChromeMac, before: m("version", 2), after: m("version", 3, "style", "bar")},
		{at: s.pushDay(9, 10, 40), actorType: "system", action: "whatsapp_templates.synced", targetType: "whatsapp_account", targetID: st.account.String(),
			target: "Acme Inc", meta: m("changed", []string{"summit_day_before: pending → approved"}, "source", "webhook")},
		{at: s.pushDay(10, 15, 12), actorType: "system", action: "whatsapp_templates.synced", targetType: "whatsapp_account", targetID: st.account.String(),
			target: "Acme Inc", meta: m("changed", []string{"summit_livestream: pending → approved"}, "source", "webhook")},
		{at: s.pushDay(13, 16, 20), actorType: "system", action: "whatsapp_templates.synced", targetType: "whatsapp_account", targetID: st.account.String(),
			target: "Acme Inc", meta: m("changed", []string{"digest_highlights: pending → rejected"}, "source", "webhook")},
		{at: s.pushDay(14, 8, 36), actorType: "user", actor: "sofia@acme.io", action: "campaign.create", targetType: "campaign", targetID: st.campaign.String(),
			target: ff, ip: "198.51.100.23", ua: uaChromeMac, after: m("name", ff, "channel", "push", "sites", 1)},
		{at: s.pushDay(14, 9, 5), actorType: "user", actor: "sofia@acme.io", action: "campaign.scheduled", targetType: "campaign", targetID: st.campaign.String(),
			target: ff, ip: "198.51.100.23", ua: uaChromeMac, before: m("status", "draft"), after: m("status", "scheduled"),
			meta: m("channel", "push", "recipients", 2931, "sites", []string{"news.acme.io"})},
		{at: s.pushDay(14, 9, 5), actorType: "system", action: "campaign.send_started", targetType: "campaign", targetID: st.campaign.String(), target: ff,
			before: m("status", "scheduled"), after: m("status", "sending"), meta: m("channel", "push", "recipients", 2931, "left_out", m())},
		{at: s.pushDay(14, 9, 7), actorType: "system", action: "campaign.completed", targetType: "campaign", targetID: st.campaign.String(), target: ff,
			before: m("status", "sending"), after: m("status", "sent"), meta: m("channel", "push", "sent", 2912, "failed", 19, "expired_removed", 19)},
		{at: s.pushDay(14, 11, 2), actorType: "system", action: "whatsapp_templates.synced", targetType: "whatsapp_account", targetID: st.account.String(),
			target: "Acme Inc", meta: m("templates", 4, "added", []string{"summit_venue_change"}, "changed", []string{}, "removed", []string{})},
		{at: s.pushDay(15, 9, 12), actorType: "user", actor: "sofia@acme.io", action: "campaign.create", targetType: "campaign", targetID: st.waDraft.String(),
			target: "Summit: see you tomorrow", ip: "198.51.100.23", ua: uaChromeMac, after: m("name", "Summit: see you tomorrow", "channel", "whatsapp",
				"template", "summit_day_before")},
	}
	for _, e := range events {
		if err := s.insertAudit(ctx, ws.ID, e); err != nil {
			return err
		}
	}
	// Meta's rejection, as Linh and Marcus were told (read since).
	at := s.pushDay(13, 16, 20)
	for _, u := range []string{"linh@acme.io", "marcus@acme.io"} {
		read := at.Add(2 * time.Hour)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO notifications (id, user_id, workspace_id, kind, severity, title, summary, body, link, created_at, read_at)
			VALUES ($1,$2,$3,'whatsapp_attention','warn',$4,$5,$6,$7,$8,$9)`, ID(), s.Users[u].ID, ws.ID, "WhatsApp template digest_highlights rejected",
			"Meta rejected digest_highlights: “Add sample values for every variable.”",
			"Fix the template in WhatsApp Manager and submit it again; ReputeCast picks up the new status at the next sync.",
			"/w/"+ws.Slug+"/providers/whatsapp/templates", at, read); err != nil {
			return err
		}
	}
	return nil
}
