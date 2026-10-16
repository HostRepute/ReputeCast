package demo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func init() { seedPageLanguages = seedPageLanguagesImpl }

// seedPageLanguagesImpl seeds the content languages of forms, public pages
// and contacts (brief §4, workstream X08): where each contact's language
// came from, the sample contacts' languages and sources ("Contacts in the
// examples").
func seedPageLanguagesImpl(ctx context.Context, s *Seeder) error {
	p := &pageLangSeed{s: s, ws: s.WS().ID}
	for _, step := range []func(context.Context) error{p.contactSources, p.sampleContacts} {
		if err := step(ctx); err != nil {
			return err
		}
	}
	return nil
}

type pageLangSeed struct {
	s  *Seeder
	ws uuid.UUID
}

// contactSources says where every contact's language came from, by how the
// contact arrived: an import's column, a signup form reading the browser,
// the API, or the team.
func (p *pageLangSeed) contactSources(ctx context.Context) error {
	linh := p.s.Users["linh@acme.io"]
	_, err := p.s.Tx.Exec(ctx, `UPDATE contacts c SET
			language_source = CASE
				WHEN c.source LIKE 'import:%' THEN 'import'
				WHEN c.source LIKE 'form:%' OR c.source = 'subscribe_page' THEN 'form'
				WHEN c.source = 'api' THEN 'api'
				WHEN c.source = 'manual' THEN 'team'
				ELSE '' END,
			language_source_name = CASE
				WHEN c.source = 'import:brevo' THEN 'Brevo'
				WHEN c.source = 'import:mailchimp' THEN 'Mailchimp'
				WHEN c.source = 'import:csv' THEN 'CSV'
				WHEN c.source LIKE 'import:%' THEN initcap(split_part(c.source, ':', 2))
				WHEN c.source LIKE 'form:%' THEN coalesce((SELECT f.name FROM forms f WHERE f.workspace_id = c.workspace_id AND f.slug = split_part(c.source, ':', 2)), '')
				ELSE '' END,
			language_browser = (c.source LIKE 'form:%' OR c.source = 'subscribe_page'),
			language_set_by = CASE WHEN c.source = 'manual' THEN $2::uuid END,
			language_set_at = c.created_at
		WHERE c.workspace_id = $1 AND c.language <> '' AND c.erased_at IS NULL`, p.ws, linh.ID)
	return err
}

// sampleContacts gives the brief's example contacts their languages and
// sources: Sam Rivera (English, Brevo import), Lucía Fernández (es-MX,
// Newsletter signup form, from the browser, 3 Oct), Jonas Weber (de-AT,
// preference center, 9 Oct), Phạm Minh Anh (Vietnamese, Brevo import) and
// Rafael Souza (pt-BR, which Acme doesn't offer, set by the API on 12 Oct).
func (p *pageLangSeed) sampleContacts(ctx context.Context) error {
	s := p.s
	type sample struct {
		email, code, source, name string
		browser                   bool
		at                        *time.Time
	}
	lucia, jonas, rafael := s.langAt(12, 10, 14), s.langAt(6, 18, 32), s.langAt(3, 11, 2)
	for _, c := range []sample{
		{email: "sam.rivera@outlook.com", code: "en", source: "import", name: "Brevo"},
		{email: "minhanh.pham@example.vn", code: "vi", source: "import", name: "Brevo"},
		{email: "lucia.fernandez@example.com", code: "es-MX", source: "form", name: "Newsletter signup", browser: true, at: &lucia},
		{email: "jonas.weber@example.de", code: "de-AT", source: "preference_center", at: &jonas},
		{email: "rafael.souza@example.com.br", code: "pt-BR", source: "api", at: &rafael},
	} {
		var id uuid.UUID
		var imported *time.Time
		err := s.Tx.QueryRow(ctx, `SELECT c.id, (SELECT min(e.occurred_at) FROM consent_events e WHERE e.contact_id = c.id AND e.action = 'imported')
			FROM contacts c WHERE c.workspace_id = $1 AND c.email = $2`, p.ws, c.email).Scan(&id, &imported)
		if err != nil {
			continue // another seeder didn't create this contact
		}
		at := c.at
		if at == nil {
			at = imported
		}
		if _, err := s.Tx.Exec(ctx, `UPDATE contacts SET language = $2, language_source = $3, language_source_name = $4, language_browser = $5,
				language_set_at = coalesce($6, created_at), language_set_by = NULL WHERE id = $1`,
			id, c.code, c.source, c.name, c.browser, at); err != nil {
			return fmt.Errorf("language of %s: %w", c.email, err)
		}
	}
	// The audience seeder writes the sample contacts' deliveries last.
	s.After(func(ctx context.Context, s *Seeder) error { return p.messageLanguages(ctx) })
	return nil
}

// messageLanguages names the version each sample delivery sent.
func (p *pageLangSeed) messageLanguages(ctx context.Context) error {
	s := p.s
	// Rafael got Weekly Digest #141 like Sam (the demo's sends left him out).
	if _, err := s.Tx.Exec(ctx, `WITH sam AS (
			SELECT m.* FROM messages m JOIN contacts c ON c.id = m.contact_id JOIN campaigns cp ON cp.id = m.campaign_id
			WHERE c.workspace_id = $1 AND c.email = 'sam.rivera@outlook.com' AND cp.name = 'Weekly Digest #141' AND m.delivered_at IS NOT NULL LIMIT 1),
		rafael AS (SELECT id FROM contacts WHERE workspace_id = $1 AND email = 'rafael.souza@example.com.br'
			AND NOT EXISTS (SELECT 1 FROM messages m WHERE m.contact_id = contacts.id)),
		msg AS (
			INSERT INTO messages (id, workspace_id, stream, campaign_id, contact_id, email, provider_id, mailbox_provider, status, queued_at, sent_at, delivered_at)
			SELECT $2::uuid, sam.workspace_id, sam.stream, sam.campaign_id, rafael.id, 'rafael.souza@example.com.br', sam.provider_id, 'other', 'delivered',
				sam.queued_at, sam.sent_at, sam.delivered_at - interval '1 minute'
			FROM sam, rafael RETURNING *)
		INSERT INTO delivery_events (id, workspace_id, message_id, campaign_id, contact_id, provider_id, type, occurred_at, mailbox_provider, original_type, original_code)
		SELECT $3::uuid, msg.workspace_id, msg.id, msg.campaign_id, msg.contact_id, msg.provider_id, 'delivered', msg.delivered_at, 'other', 'Delivery', '250 2.6.0'
		FROM msg`, p.ws, ID(), ID()); err != nil {
		return fmt.Errorf("rafael's delivery: %w", err)
	}
	// The sends name the version each reader got: Sam's Weekly Digest #141
	// went out in English (ContactProfile), and Rafael's latest email in
	// English, the fallback, because Portuguese isn't one of Acme's
	// languages (LanguageStates).
	for _, m := range []struct{ email, campaign, reason, contact string }{
		{"sam.rivera@outlook.com", "Weekly Digest #141", "match", "en"},
		{"rafael.souza@example.com.br", "", "not_offered", "pt"},
	} {
		if _, err := s.Tx.Exec(ctx, `UPDATE messages SET language = 'en', language_reason = $3, contact_language = $4
			WHERE id = (SELECT m.id FROM messages m JOIN contacts c ON c.id = m.contact_id JOIN campaigns cp ON cp.id = m.campaign_id
				WHERE c.workspace_id = $1 AND c.email = $2 AND ($5::text = '' OR cp.name = $5::text) ORDER BY m.queued_at DESC LIMIT 1)`,
			p.ws, m.email, m.reason, m.contact, m.campaign); err != nil {
			return err
		}
	}
	return nil
}
