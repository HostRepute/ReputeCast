package publicpages_test

// The public pages in the visitor's language (design expansion M17, phase
// 2 brief §3.6). The demo's pages are Live in English, Spanish and German;
// Vietnamese is offered but its page copy isn't set up, so its visitors get
// the fallback's pages (English). Pages reached from an email follow the
// contact's language, the others the menu's pick, then the browser, then
// the default; the menu is plain links, so every page works without
// JavaScript. Archived issues show the version a reader of the visitor's
// language gets, with the others under "Also in" and as hreflang
// alternates.

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/linktoken"
)

func TestVisitorLanguage(t *testing.T) {
	env, ids := demo(t)
	page := func(path string, h hdr) resp {
		t.Helper()
		return do(t, env, http.MethodGet, path, nil, h)
	}
	const subscribe = "/p/acme-newsletter/subscribe"

	t.Run("the browser's language", func(t *testing.T) {
		r := page(subscribe, hdr{"Accept-Language": "es-MX,es;q=0.9,en;q=0.5"})
		must(t, r, 200, `<html lang="es" dir="ltr">`, "El boletín de Acme", "Suscribirme", `<nav class="langs" aria-label="Idioma">`,
			`<a href="/p/acme-newsletter/subscribe?lang=es" lang="es" hreflang="es" aria-current="true">Español</a>`,
			`<a href="/p/acme-newsletter/subscribe?lang=en" lang="en" hreflang="en">English</a>`,
			`<a href="/p/acme-newsletter/subscribe?lang=de" lang="de" hreflang="de">Deutsch</a>`,
			`<link rel="alternate" hreflang="de" href="http://news.acme.io/subscribe?lang=de">`,
			`<link rel="alternate" hreflang="x-default" href="http://news.acme.io/subscribe">`,
			`<link rel="canonical" href="http://news.acme.io/subscribe">`)
		lacks(t, r, "Tiếng Việt", "Subscribe to")
		if vary := r.Header.Values("Vary"); !slices.Contains(vary, "Accept-Language") || !slices.Contains(vary, "Cookie") {
			t.Errorf("Vary %v: the page depends on the browser's language and the menu's cookie", vary)
		}
		if c := r.Header.Get("Set-Cookie"); c != "" {
			t.Errorf("the browser's language sets no cookie: %s", c)
		}
	})

	t.Run("a language that isn't Live gets the fallback's pages, whole", func(t *testing.T) {
		r := page(subscribe, hdr{"Accept-Language": "vi-VN,vi;q=0.9"})
		must(t, r, 200, `<html lang="en" dir="ltr">`, `aria-label="Language"`, `aria-current="true">English</a>`)
		lacks(t, r, "Suscribirme", "Tiếng Việt")
	})

	t.Run("a language the workspace doesn't offer gets the default", func(t *testing.T) {
		must(t, page(subscribe, hdr{"Accept-Language": "fr-FR,fr;q=0.9"}), 200, `<html lang="en" dir="ltr">`)
		must(t, page(subscribe, nil), 200, `<html lang="en" dir="ltr">`)
	})

	t.Run("the menu's pick beats the browser and is remembered", func(t *testing.T) {
		r := page(subscribe+"?lang=de", hdr{"Accept-Language": "es"})
		must(t, r, 200, `<html lang="de" dir="ltr">`, "Der Acme-Newsletter", "Abonnieren", `aria-current="true">Deutsch</a>`,
			// Picked by address, the page is its own canonical address.
			`<link rel="canonical" href="http://news.acme.io/subscribe?lang=de">`)
		cookie := r.Header.Get("Set-Cookie")
		for _, want := range []string{"rc_lang=de", "Path=/p/acme-newsletter/", "HttpOnly", "SameSite=Lax"} {
			if !strings.Contains(cookie, want) {
				t.Fatalf("Set-Cookie %q lacks %s", cookie, want)
			}
		}
		must(t, page(subscribe, hdr{"Accept-Language": "es", "Cookie": "rc_lang=de"}), 200, `<html lang="de" dir="ltr">`)
		// A pick that isn't Live is ignored, and not remembered.
		r = page(subscribe+"?lang=vi", hdr{"Accept-Language": "es"})
		must(t, r, 200, `<html lang="es" dir="ltr">`)
		if c := r.Header.Get("Set-Cookie"); c != "" {
			t.Errorf("a pick that can't be shown sets no cookie: %s", c)
		}
	})

	t.Run("without JavaScript", func(t *testing.T) {
		// The hosted page posts the language it showed; the answer comes in it.
		f := url.Values{"_t": {formToken(env, "frm_8c2e1a", time.Minute)}, "email": {"maria.garcia@fastmail.com"}, "list": {ids.digest.String()},
			"consent": {"1"}, "_lang": {"es"}}
		r := do(t, env, http.MethodPost, subscribe, f, hdr{"Origin": env.Server.URL})
		must(t, r, 200, `<html lang="es" dir="ltr">`, "Revisa tu correo", "maria.garcia@fastmail.com")
		lacks(t, r, "<script")
		if n := count(t, env, `SELECT count(*) FROM form_submissions fs JOIN contacts c ON c.id = fs.contact_id
			WHERE c.email = 'maria.garcia@fastmail.com' AND fs.language = 'es'`); n != 1 {
			t.Errorf("%d submissions in Spanish, want 1", n)
		}
		// A field error shows the form again, in the same language.
		f.Set("email", "not-an-address")
		r = do(t, env, http.MethodPost, subscribe, f, hdr{"Origin": env.Server.URL})
		must(t, r, 422, `<html lang="es" dir="ltr">`, "Suscribirme")
	})

	t.Run("pages from an email follow the contact's language", func(t *testing.T) {
		setLang := func(code string) {
			t.Helper()
			if _, err := env.App.Pool.Exec(context.Background(), `UPDATE contacts SET language = $2 WHERE id = $1`, ids.sam, code); err != nil {
				t.Fatal(err)
			}
		}
		prefs := "/p/acme-newsletter/preferences/" + sign(env, linktoken.Preferences, ids.ws, ids.sam, uuid.Nil)
		unsub := "/p/acme-newsletter/unsubscribe/" + sign(env, linktoken.Unsubscribe, ids.ws, ids.sam, uuid.Nil)

		setLang("es-MX")
		r := page(prefs, hdr{"Accept-Language": "de"})
		must(t, r, 200, `<html lang="es" dir="ltr">`, "Preferencias de correo", "Idioma de los correos",
			`<label lang="en"><input type="radio" name="language" value="en">English</label>`,
			`<label lang="es"><input type="radio" name="language" value="es" checked>Español</label>`,
			`<label lang="de"><input type="radio" name="language" value="de">Deutsch</label>`,
			// Every workspace language, even one whose pages aren't Live.
			`<label lang="vi"><input type="radio" name="language" value="vi">Tiếng Việt</label>`,
			"Los correos de Acme llegan en este idioma cuando tienen una versión en él, y si no, en English.")
		must(t, page(unsub, nil), 200, `<html lang="es" dir="ltr">`, "¿Dar de baja sam.rivera@outlook.com?")

		// Picking German in "Email language" changes the contact's language,
		// and the pages follow it.
		shown := strings.Join([]string{ids.digest.String(), ids.updates.String(), ids.beta.String()}, ",")
		r = do(t, env, http.MethodPost, prefs, url.Values{"shown": {shown}, "list": {ids.digest.String(), ids.updates.String()}, "frequency": {"weekly"},
			"pause": {"0"}, "language": {"de"}}, nil)
		if r.Status != 303 {
			t.Fatalf("save: %d\n%s", r.Status, clip(r.Body))
		}
		var lang, source string
		if err := env.App.Pool.QueryRow(context.Background(), `SELECT language, language_source FROM contacts WHERE id = $1`, ids.sam).Scan(&lang, &source); err != nil {
			t.Fatal(err)
		}
		if lang != "de" || source != "preference_center" {
			t.Fatalf("Sam's language %q from %q, want de from the preference center", lang, source)
		}
		must(t, page(prefs+"?saved=1", nil), 200, `<html lang="de" dir="ltr">`, "Gespeichert. Änderungen gelten ab dem nächsten Versand.",
			`value="de" checked>Deutsch</label>`)

		// A language that isn't one of the workspace's is refused.
		r = do(t, env, http.MethodPost, prefs, url.Values{"language": {"fr"}}, nil)
		must(t, r, 422, `<html lang="de" dir="ltr">`, "Wähle eine der aufgeführten Sprachen.")

		// Vietnamese isn't Live: the fallback's pages. With no language,
		// the fallback too, as the contact's emails.
		setLang("vi")
		must(t, page(prefs, hdr{"Accept-Language": "es"}), 200, `<html lang="en" dir="ltr">`, "Email preferences", `value="vi" checked>Tiếng Việt</label>`)
		setLang("")
		must(t, page(prefs, hdr{"Accept-Language": "es"}), 200, `<html lang="en" dir="ltr">`)
		// The menu's pick beats the contact's language.
		must(t, page(prefs+"?lang=es", nil), 200, `<html lang="es" dir="ltr">`)
	})

	t.Run("a page in one language says so to search engines", func(t *testing.T) {
		// Only the default's pages exist in every workspace: one language,
		// no menu and no alternates.
		r := page("/p/acme-engineering-blog/subscribe", hdr{"Accept-Language": "es"})
		if r.Status != 200 {
			t.Skipf("the second workspace has no subscribe page: %d", r.Status)
		}
		must(t, r, 200, `<html lang="en" dir="ltr">`)
		lacks(t, r, `class="langs"`, `hreflang=`)
	})
}

func TestArchiveLanguages(t *testing.T) {
	env, ids := demo(t)
	websiteOff(t, env, ids.ws)
	page := func(path string, h hdr) resp {
		t.Helper()
		return do(t, env, http.MethodGet, path, nil, h)
	}
	// The archive files its issues on its first read.
	must(t, page("/p/acme-newsletter/archive", nil), 200)
	const slug = "the-case-for-boring-infrastructure"
	issue := "/p/acme-newsletter/archive/" + slug
	canon := "http://news.acme.io/archive/" + slug
	campaign := one[uuid.UUID](t, env, `SELECT campaign_id FROM archive_issues WHERE workspace_id = $1 AND slug = $2`, ids.ws, slug)
	// #141 is written in English, with a Spanish version, a German one
	// that's held ("Send English to them instead") and a Vietnamese draft.
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(context.Background(), sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO content_sources (workspace_id, owner_kind, owner_id, owner_version, language) VALUES ($1, 'campaign', $2, 0, 'en')
		ON CONFLICT (owner_kind, owner_id, owner_version) DO UPDATE SET language = 'en'`, ids.ws, campaign)
	version := func(lang, status string, held bool, subject, body string) {
		t.Helper()
		exec(`INSERT INTO content_translations (id, workspace_id, owner_kind, owner_id, owner_version, language, status, held, subject, preheader, html, plain_text)
			VALUES ($1, $2, 'campaign', $3, 0, $4, $5, $6, $7, $8, $9, $10)`, uuid.Must(uuid.NewV7()), ids.ws, campaign, lang, status, held, subject,
			"Volvimos a Postgres", `<!doctype html><html lang="en"><head><title>x</title></head><body><p>`+body+`</p></body></html>`, body)
	}
	version("es", "translated", false, "El caso de la infraestructura aburrida", "Hola, lectores del Weekly Digest")
	version("de", "translated", true, "Plädoyer für langweilige Infrastruktur", "Hallo zusammen")
	version("vi", "draft", false, "Lý do chọn hạ tầng nhàm chán", "Xin chào")

	t.Run("the visitor's language", func(t *testing.T) {
		r := page(issue, hdr{"Accept-Language": "es-ES,es;q=0.9"})
		must(t, r, 200, `<html lang="es" dir="ltr">`, "Hola, lectores del Weekly Digest",
			"<title>El caso de la infraestructura aburrida · Boletín de Acme</title>",
			`<link rel="canonical" href="`+canon+`/es">`,
			`<link rel="alternate" hreflang="en" href="`+canon+`">`,
			`<link rel="alternate" hreflang="es" href="`+canon+`/es">`,
			`<link rel="alternate" hreflang="x-default" href="`+canon+`">`,
			"Acme · Archivo del boletín", ">Suscribirme</a>", `<header class="rcia">`,
			`<p class="rcia-also">También en: <a href="`+env.Server.URL+issue+`/en" lang="en" hreflang="en">English</a></p>`,
			`<nav class="rcia-langs" aria-label="Idioma">`,
			`<a href="`+env.Server.URL+issue+`?lang=es" lang="es" hreflang="es" aria-current="true">Español</a>`)
		// Held and draft versions aren't readers'.
		lacks(t, r, `hreflang="de" href="`+canon, `hreflang="vi"`, "Hallo zusammen", "Xin chào", "The case for boring infrastructure")
		if vary := r.Header.Values("Vary"); !slices.Contains(vary, "Accept-Language") || !slices.Contains(vary, "Cookie") {
			t.Errorf("Vary %v", vary)
		}
		ld := regexp.MustCompile(`"inLanguage":"([a-z]+)"`).FindStringSubmatch(r.Body)
		if ld == nil || ld[1] != "es" {
			t.Errorf("JSON-LD inLanguage %v", ld)
		}
	})

	t.Run("no ready version in the visitor's language", func(t *testing.T) {
		// German is held: a German reader gets the fallback's version
		// (English), under a bar in German, whose pages are Live.
		r := page(issue, hdr{"Accept-Language": "de-DE"})
		must(t, r, 200, `<html lang="en" dir="ltr"`, "The case for boring infrastructure", `<link rel="canonical" href="`+canon+`">`,
			`<header class="rcia" lang="de" dir="ltr">`, "Acme · Newsletter-Archiv", ">Abonnieren</a>",
			`Auch auf: <a href="`+env.Server.URL+issue+`/es" lang="es" hreflang="es">Español</a>`,
			`<footer class="rcia rcia-foot" lang="de" dir="ltr">`)
		// Vietnamese has a draft, and its pages aren't Live.
		r = page(issue, hdr{"Accept-Language": "vi"})
		must(t, r, 200, `<html lang="en" dir="ltr"`, "Acme · Newsletter archive", "Also in: ")
		lacks(t, r, "Xin chào")
	})

	t.Run("each version's own address", func(t *testing.T) {
		r := page(issue+"/es", nil)
		must(t, r, 200, `<html lang="es" dir="ltr">`, "Hola, lectores del Weekly Digest", `<link rel="canonical" href="`+canon+`/es">`, "Acme · Archivo del boletín")
		// The language it's written in, by address: canonical at the issue's own.
		r = page(issue+"/en", hdr{"Accept-Language": "es"})
		must(t, r, 200, "The case for boring infrastructure", `<link rel="canonical" href="`+canon+`">`, "Acme · Newsletter archive",
			`Also in: <a href="`+env.Server.URL+issue+`/es" lang="es" hreflang="es">Español</a>`)
		for _, lang := range []string{"de", "vi", "fr", "ES"} {
			r := page(issue+"/"+lang, nil)
			if r.Status != http.StatusFound || r.Header.Get("Location") != env.Server.URL+issue {
				t.Errorf("/%s: %d %q, want a redirect to the issue in the visitor's language", lang, r.Status, r.Header.Get("Location"))
			}
		}
	})

	t.Run("the list in the visitor's language", func(t *testing.T) {
		r := page("/p/acme-newsletter/archive", hdr{"Accept-Language": "es"})
		must(t, r, 200, `<html lang="es" dir="ltr">`, "Archivo del boletín", "Todos los números de Weekly Digest y Product Updates desde ", " 46 números.",
			">Todos</a>", "1 a 10 de 46", "Números anteriores", "Buscar números",
			`<a class="t" href="`+env.Server.URL+issue+`/es">El caso de la infraestructura aburrida</a>`,
			// The issues without a Spanish version say they're in English.
			`<article class="issue" lang="en">`,
			`<link rel="alternate" hreflang="es" href="http://news.acme.io/archive?lang=es">`)
		lacks(t, r, ">The case for boring infrastructure<")
		// A search finds an issue by its versions' words too.
		r = page("/p/acme-newsletter/archive?q=aburrida", hdr{"Accept-Language": "es"})
		must(t, r, 200, "El caso de la infraestructura aburrida", "1 a 1 de 1")
		r = page("/p/acme-newsletter/archive?q=aburrida", nil)
		must(t, r, 200, "The case for boring infrastructure", "1 to 1 of 1")
		// A pick in the menu is kept in the search form and the pages' links.
		r = page("/p/acme-newsletter/archive?lang=de", nil)
		must(t, r, 200, `<html lang="de" dir="ltr">`, `<input type="hidden" name="lang" value="de">`, "/p/acme-newsletter/archive?lang=de&page=2")
	})

	t.Run("view online shows the version the message delivered", func(t *testing.T) {
		var msg, cp uuid.UUID
		if err := env.App.Pool.QueryRow(context.Background(), `SELECT m.id, m.campaign_id FROM messages m JOIN campaigns c ON c.id = m.campaign_id
			WHERE m.contact_id = $1 AND c.name = 'October Product Update' LIMIT 1`, ids.sam).Scan(&msg, &cp); err != nil {
			t.Fatalf("Sam's October Product Update: %v", err)
		}
		exec(`INSERT INTO content_sources (workspace_id, owner_kind, owner_id, owner_version, language) VALUES ($1, 'campaign', $2, 0, 'en')
			ON CONFLICT (owner_kind, owner_id, owner_version) DO UPDATE SET language = 'en'`, ids.ws, cp)
		exec(`INSERT INTO content_translations (id, workspace_id, owner_kind, owner_id, owner_version, language, status, subject, html, plain_text)
			VALUES ($1, $2, 'campaign', $3, 0, 'es', 'translated', 'Novedades de octubre', '<html><body><p>Hola {{ first_name }}</p></body></html>', 'Hola')`,
			uuid.Must(uuid.NewV7()), ids.ws, cp)
		view := "/p/acme-newsletter/view/" + sign(env, linktoken.ViewOnline, ids.ws, ids.sam, msg)
		r := page(view, nil)
		must(t, r, 200, `<html lang="en" dir="ltr"`)
		lacks(t, r, "Novedades de octubre")
		exec(`UPDATE messages SET language = 'es' WHERE id = $1`, msg)
		r = page(view, nil)
		must(t, r, 200, `<html lang="es" dir="ltr">`, "<title>Novedades de octubre</title>", "Hola Sam")
	})
}

// A transactional message's category unsubscribe page and the preference
// center of someone who isn't a contact (X02's pages) come in the language
// the message was sent in, with the menu, like every public page.
func TestNotificationPagesLanguage(t *testing.T) {
	env, ids := demo(t)
	var msg uuid.UUID
	var email string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT m.id, m.email FROM messages m JOIN transactional_messages tm ON tm.message_id = m.id
		JOIN transactional_categories c ON c.id = tm.category_id
		WHERE m.workspace_id = $1 AND c.kind = 'optional' AND NOT EXISTS (SELECT 1 FROM contacts ct WHERE ct.workspace_id = m.workspace_id AND ct.email = m.email)
		ORDER BY m.queued_at DESC LIMIT 1`, ids.ws).Scan(&msg, &email); err != nil {
		t.Fatalf("a transactional message of an Optional category to someone who isn't a contact: %v", err)
	}
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE messages SET language = 'de' WHERE id = $1`, msg); err != nil {
		t.Fatal(err)
	}
	unsub := "/p/acme-newsletter/unsubscribe/" + sign(env, linktoken.Unsubscribe, ids.ws, uuid.Nil, msg)
	r := get(t, env, unsub)
	must(t, r, 200, `<html lang="de" dir="ltr">`, "nicht mehr", "Konto- und Rechnungs-E-Mails kommen weiterhin an", "Aus Versehen ausgeschaltet?",
		"wieder einschalten", "Warum hast du es ausgeschaltet?", "Zu oft", "Andere Benachrichtigungen: ", "Der Abmelde-Button in deiner Mail-App macht dasselbe.",
		`<nav class="langs" aria-label="Sprache">`, `aria-current="true">Deutsch</a>`)
	lacks(t, r, "Turned it off by mistake?", "You won't get")
	// The menu's pick beats the message's language; turning it back on
	// answers in the language picked.
	r = do(t, env, http.MethodPost, unsub+"?lang=es", url.Values{"action": {"on"}}, nil)
	must(t, r, 200, `<html lang="es" dir="ltr">`, "vuelve a estar activado", "Desactivar ")
	prefs := "/p/acme-newsletter/preferences/" + sign(env, linktoken.Preferences, ids.ws, uuid.Nil, msg)
	r = get(t, env, prefs)
	must(t, r, 200, `<html lang="de" dir="ltr">`, "E-Mail-Einstellungen", "für "+email, "Benachrichtigungen von ")
	lacks(t, r, "Notifications from", "nicht du?")
	// Without a language on the message, the fallback's pages.
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE messages SET language = '' WHERE id = $1`, msg); err != nil {
		t.Fatal(err)
	}
	must(t, do(t, env, http.MethodGet, prefs, nil, hdr{"Accept-Language": "de"}), 200, `<html lang="en" dir="ltr">`, "Notifications from ")
}
