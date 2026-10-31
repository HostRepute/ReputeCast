package publicpages_test

// Forms in the visitor's language (design expansion M17): the embed script
// carries every language the form is Live in; a signup records the
// language it was shown in with the consent text of that language, sets
// the new contact's language from the browser, and its confirmation email
// goes out in that language. A language that isn't Live falls back.

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestFormLanguages(t *testing.T) {
	env, ids := demo(t)
	mail := &fakeMail{}
	env.App.Mail = mail
	submit := "/p/acme-newsletter/forms/frm_8c2e1a/submit"
	post := func(email, lang, accept string) resp {
		t.Helper()
		return do(t, env, http.MethodPost, submit, url.Values{"_t": {formToken(env, "frm_8c2e1a", time.Minute)}, "email": {email},
			"list": {ids.digest.String(), ids.updates.String()}, "consent": {"1"}, "_js": {"1"}, "_lang": {lang}},
			hdr{"Origin": "https://acme.io", "Accept": "application/json", "Accept-Language": accept})
	}
	type evidence struct{ language, text, version string }
	consent := func(email string) evidence {
		t.Helper()
		var e evidence
		if err := env.App.Pool.QueryRow(context.Background(), `SELECT ce.language, ce.consent_text, ce.consent_version FROM consent_events ce
			JOIN contacts c ON c.id = ce.contact_id WHERE c.workspace_id = $1 AND c.email = $2 AND ce.action = 'subscribed' ORDER BY ce.occurred_at DESC LIMIT 1`,
			ids.ws, email).Scan(&e.language, &e.text, &e.version); err != nil {
			t.Fatalf("consent of %s: %v", email, err)
		}
		return e
	}

	t.Run("the script carries the Live languages", func(t *testing.T) {
		r := get(t, env, "/p/acme-newsletter/forms/frm_8c2e1a.js")
		must(t, r, 200, `"mode":"page"`, `"fallback":"en"`, `"es":{"lang":"es"`, `"button":"Suscribirme"`, `"de":{"lang":"de"`, `"privacyText":"política de privacidad"`)
		if strings.Contains(r.Body, `"vi":{`) {
			t.Error("Vietnamese isn't Live, so the script shouldn't offer it")
		}
	})

	t.Run("a Spanish signup", func(t *testing.T) {
		r := post("ana.lopez@fastmail.com", "es-MX", "es-MX,es;q=0.9,en;q=0.5")
		if r.Status != 200 || jsonBody(t, r)["status"] != "confirm" {
			t.Fatalf("signup: %d %s", r.Status, r.Body)
		}
		e := consent("ana.lopez@fastmail.com")
		if e.language != "es" || e.version != "v1" || !strings.HasPrefix(e.text, "Acepto recibir correos de Acme.") {
			t.Fatalf("consent evidence: %+v", e)
		}
		var lang, source, name string
		var browser bool
		if err := env.App.Pool.QueryRow(context.Background(), `SELECT language, language_source, language_source_name, language_browser FROM contacts
			WHERE workspace_id = $1 AND email = 'ana.lopez@fastmail.com'`, ids.ws).Scan(&lang, &source, &name, &browser); err != nil {
			t.Fatal(err)
		}
		if lang != "es-MX" || source != "form" || name != "Newsletter signup" || !browser {
			t.Fatalf("contact language %q from %q %q (browser %v)", lang, source, name, browser)
		}
		if n := count(t, env, `SELECT count(*) FROM form_submissions fs JOIN contacts c ON c.id = fs.contact_id WHERE c.email = 'ana.lopez@fastmail.com' AND fs.language = 'es'`); n != 1 {
			t.Fatalf("submissions in Spanish: %d", n)
		}
		sent := mail.to("ana.lopez@fastmail.com")
		if len(sent) != 1 || sent[0].Subject != "Confirma tu suscripción a Acme" {
			t.Fatalf("confirmation email: %+v", sent)
		}
		if !strings.Contains(sent[0].Text, "Hola:") || !strings.Contains(sent[0].Text, "Weekly Digest y Product Updates") ||
			!strings.Contains(sent[0].HTML, `<html lang="es" dir="ltr">`) || !strings.Contains(sent[0].HTML, "Confirmar mi suscripción") {
			t.Fatalf("confirmation copy:\n%s\n%s", sent[0].Text, sent[0].HTML)
		}
		// The confirmation page comes in the language the signup was shown
		// in, whatever the browser that opens the link.
		link, err := url.Parse(linkIn.FindString(sent[0].Text))
		if err != nil || link.Path == "" {
			t.Fatalf("no confirmation link in:\n%s", sent[0].Text)
		}
		must(t, do(t, env, http.MethodGet, link.Path, nil, hdr{"Accept-Language": "de"}), 200, `<html lang="es" dir="ltr">`, "Confirma tu suscripción")
		must(t, confirmClick(t, env, link.Path, hdr{"Accept-Language": "de"}), 200, `<html lang="es" dir="ltr">`, "Ya estás dentro")
	})

	t.Run("a language that isn't Live falls back", func(t *testing.T) {
		r := post("thao.tran@fastmail.com", "vi", "vi-VN,vi;q=0.9")
		if r.Status != 200 {
			t.Fatalf("signup: %d %s", r.Status, r.Body)
		}
		if e := consent("thao.tran@fastmail.com"); e.language != "en" || e.version != "v3" || e.text != consentText {
			t.Fatalf("consent evidence: %+v", e)
		}
		sent := mail.to("thao.tran@fastmail.com")
		if len(sent) != 1 || sent[0].Subject != "Confirm your subscription to Acme" {
			t.Fatalf("confirmation email: %+v", sent)
		}
		if link, err := url.Parse(linkIn.FindString(sent[0].Text)); err == nil && link.Path != "" {
			must(t, confirmClick(t, env, link.Path, hdr{"Accept-Language": "vi"}), 200, `<html lang="en" dir="ltr">`, "You're in")
		} else {
			t.Fatalf("no confirmation link in:\n%s", sent[0].Text)
		}
		// The contact still gets the browser's language: Vietnamese is offered.
		if n := count(t, env, `SELECT count(*) FROM contacts WHERE email = 'thao.tran@fastmail.com' AND language = 'vi-VN'`); n != 1 {
			t.Fatal("the contact's language should come from the browser")
		}
	})

	t.Run("errors come in the form's language", func(t *testing.T) {
		r := post("not-an-address", "de", "de-DE")
		if r.Status != 422 {
			t.Fatalf("status %d: %s", r.Status, r.Body)
		}
		errs, _ := jsonBody(t, r)["errors"].(map[string]any)
		if errs["email"] != "Diese E-Mail-Adresse scheint unvollständig zu sein." {
			t.Fatalf("errors: %v", errs)
		}
	})
}
