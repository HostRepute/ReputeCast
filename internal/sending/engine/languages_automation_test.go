package engine_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/sending/engine"
)

// testAutomationLanguages: an automation email's readers get the language
// version of the email version their step sends, and a version that stops
// being sent after it was queued (its language removed) goes out in the
// source.
func testAutomationLanguages(t *testing.T, f *fixture) {
	f.exec(`INSERT INTO workspace_languages (workspace_id, code, page_copy_status) VALUES ($1, 'de', 'live')
		ON CONFLICT (workspace_id, code) DO UPDATE SET removed_at = NULL`, f.ws)
	email := uuid.Must(uuid.NewV7())
	html := `<html><body><p>Welcome aboard</p><p><a href="https://acme.io/start">Get started</a></p>` +
		`<p><a href="{{ unsubscribe_url }}">Unsubscribe</a> · Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105</p></body></html>`
	f.exec(`INSERT INTO automation_emails (id, workspace_id, name, latest_version) VALUES ($1, $2, 'Language test', 1)`, email, f.ws)
	f.exec(`INSERT INTO automation_email_versions (email_id, version, subject, html, plain_text) VALUES ($1, 1, 'Welcome', $2, 'Welcome aboard')`, email, html)
	f.exec(`INSERT INTO content_translations (id, workspace_id, owner_kind, owner_id, owner_version, language, status, subject, html, plain_text)
		VALUES (gen_random_uuid(), $1, 'automation_email', $2, 1, 'de', 'translated', 'Willkommen', $3, 'Willkommen an Bord')`,
		f.ws, email, strings.NewReplacer("Welcome aboard", "Willkommen an Bord", "https://acme.io/start", "https://acme.io/de/start").Replace(html))
	f.addContacts(contactSpec{n: 3, mailbox: "gmail", domain: "gmail.com", engaged: true})
	type contact struct {
		id    uuid.UUID
		email string
	}
	var contacts []contact
	rows, err := f.env.App.Pool.Query(f.ctx, `SELECT c.id, c.email FROM contacts c JOIN list_memberships m ON m.contact_id = c.id WHERE m.list_id = $1 ORDER BY c.email`, f.list)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var c contact
		if err := rows.Scan(&c.id, &c.email); err != nil {
			t.Fatal(err)
		}
		contacts = append(contacts, c)
	}
	rows.Close()
	for i, lang := range []string{"de-AT", "fr", "de"} {
		f.exec(`UPDATE contacts SET language = $2 WHERE id = $1`, contacts[i].id, lang)
	}
	var flow uuid.UUID
	if err := f.env.App.Pool.QueryRow(f.ctx, `SELECT id FROM automations WHERE workspace_id = $1 ORDER BY created_at LIMIT 1`, f.ws).Scan(&flow); err != nil {
		t.Fatal(err)
	}
	m := engine.AutomationMailer{E: f.e}
	queue := func(c contact) uuid.UUID {
		t.Helper()
		tx, err := f.env.App.Pool.Begin(f.ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(f.ctx) //nolint:errcheck
		out, err := m.QueueAutomationEmail(f.ctx, tx, automations.OutgoingEmail{WorkspaceID: f.ws, AutomationID: flow, NodeID: "m", ContactID: c.id,
			To: c.email, EmailID: email, EmailVersion: 1, Subject: "Welcome"})
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(f.ctx); err != nil {
			t.Fatal(err)
		}
		return out.MessageID
	}
	german, french := queue(contacts[0]), queue(contacts[1])
	language := func(id uuid.UUID) (lang, contactLang, reason string, ref map[string]any) {
		t.Helper()
		var raw []byte
		if err := f.env.App.Pool.QueryRow(f.ctx, `SELECT language, contact_language, language_reason, content_ref FROM messages WHERE id = $1`, id).
			Scan(&lang, &contactLang, &reason, &raw); err != nil {
			t.Fatal(err)
		}
		_ = json.Unmarshal(raw, &ref)
		return lang, contactLang, reason, ref
	}
	if l, c, r, ref := language(german); l != "de" || c != "de" || r != "match" || ref["language"] != "de" {
		t.Fatalf("de-AT reader: %q %q %q %v", l, c, r, ref)
	}
	if l, c, r, ref := language(french); l != "en" || c != "fr" || r != "not_offered" || ref["language"] != nil {
		t.Fatalf("fr reader: %q %q %q %v", l, c, r, ref)
	}
	send := func(now time.Time) {
		t.Helper()
		for i := range 5 {
			if _, err := f.e.OutboxStep(f.ctx, now.Add(time.Duration(i)*time.Second)); err != nil {
				t.Fatal(err)
			}
		}
	}
	send(f.start)
	// The German reader got the German version: its own links are indexed
	// under the language.
	if n := f.count(`SELECT count(*) FROM automation_email_links WHERE email_id = $1 AND language = 'de' AND url = 'https://acme.io/de/start'`, email); n != 1 {
		t.Fatalf("%d German links indexed, want 1", n)
	}
	if n := f.count(`SELECT count(*) FROM automation_email_links WHERE email_id = $1 AND language = '' AND url = 'https://acme.io/start'`, email); n != 1 {
		t.Fatalf("%d source links indexed, want 1", n)
	}
	if n := f.count(`SELECT count(*) FROM messages WHERE id = ANY($1) AND sent_at IS NOT NULL`, []uuid.UUID{german, french}); n != 2 {
		t.Fatalf("%d of 2 automation emails sent", n)
	}
	// Queued in German, then German is removed: it goes out in English.
	late := queue(contacts[2])
	f.exec(`UPDATE workspace_languages SET removed_at = now() WHERE workspace_id = $1 AND code = 'de'`, f.ws)
	send(f.start.Add(time.Minute))
	if l, c, r, ref := language(late); l != "en" || c != "de" || r != "not_offered" || ref["language"] != nil {
		t.Fatalf("removed German: %q %q %q %v", l, c, r, ref)
	}
	if n := f.count(`SELECT count(*) FROM messages WHERE id = $1 AND sent_at IS NOT NULL`, late); n != 1 {
		t.Fatalf("the email queued in German wasn't sent in English")
	}
}
