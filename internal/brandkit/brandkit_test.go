package brandkit_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"math"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/brandkit"
)

// CO-11: the brand kit, applying it to starters and blocks, synced blocks
// reaching drafts and scheduled emails, and importing from a website.

// Acme Newsletter has the demo's kit (internal/demo/seed_brandkit.go); the
// engineering blog has none, so a first save happens there.
const (
	ws  = "/workspaces/acme-newsletter"
	eng = "/workspaces/acme-engineering-blog"
)

type kit struct {
	Saved     bool
	LogoLight *struct{ MediaID, URL string }
	LogoAlt   string
	Colors    struct{ Button, Text, Background, Link string }
	Heading   struct{ Name, URL, Fallback string }
	Body      struct{ Name, URL, Fallback string }
	Button    struct {
		Shape string
		Size  int
	}
	Synced []struct {
		Kind, MJML string
		Version    int
		Uses       struct{ Drafts, Scheduled, Held, Sent int }
	}
}

func logoPNG(t *testing.T) []byte {
	t.Helper()
	m := image.NewNRGBA(image.Rect(0, 0, 240, 64))
	for i := 3; i < len(m.Pix); i += 4 {
		m.Pix[i-3], m.Pix[i-2], m.Pix[i-1], m.Pix[i] = 0x43, 0x38, 0xCA, 255
	}
	m.Set(0, 0, color.NRGBA{})
	var b bytes.Buffer
	if err := png.Encode(&b, m); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func upload(t *testing.T, c *apptest.Client, base, name string, data []byte) string {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", name)
	_, _ = fw.Write(data)
	_ = mw.Close()
	var a struct{ ID string }
	c.DoRaw("POST", base+"/media", mw.FormDataContentType(), &body).MustStatus(t, 201).JSON(t, &a)
	return a.ID
}

func input(logo string) map[string]any {
	return map[string]any{
		"logoLightId": logo, "logoAlt": "Acme",
		"colors":  map[string]any{"button": "#4338ca", "text": "#1F2937", "background": "#F5F3FF", "link": "#4338CA"},
		"heading": map[string]any{"name": "Source Serif 4", "url": "https://fonts.googleapis.com/css2?family=Source+Serif+4:wght@400;700&display=swap", "fallback": "Georgia, 'Times New Roman', serif"},
		"body":    map[string]any{"name": "Inter", "url": "https://fonts.googleapis.com/css2?family=Inter:wght@400;700&display=swap", "fallback": "Arial, Helvetica, sans-serif"},
		"button":  map[string]any{"shape": "pill", "size": 15},
		"website": "acme.io",
	}
}

func campaign(t *testing.T, c *apptest.Client, base, mjml string) string {
	t.Helper()
	var d struct{ ID string }
	c.Do("POST", base+"/campaigns", map[string]any{}).MustStatus(t, 201).JSON(t, &d)
	c.Do("PUT", base+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 1, "subject": "Hello", "mjml": mjml}).MustStatus(t, 200)
	return d.ID
}

func TestBrandKit(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	aisha := env.Client()
	aisha.SignInDemo("aisha@acme.io")

	var k kit
	// A viewer reads Acme's kit and can't change it.
	aisha.Do("GET", ws+"/brand-kit", nil).MustStatus(t, 200).JSON(t, &k)
	if !k.Saved {
		t.Fatalf("Acme's kit: %+v", k)
	}
	aisha.Do("PUT", ws+"/brand-kit", input("")).MustStatus(t, 403)
	// Without a kit: ReputeCast's defaults, until the first save.
	linh.Do("GET", eng+"/brand-kit", nil).MustStatus(t, 200).JSON(t, &k)
	if k.Saved || k.Colors.Button != "#0F766E" || len(k.Synced) != 0 {
		t.Fatalf("defaults: %+v", k)
	}
	bad := input("")
	bad["colors"].(map[string]any)["link"] = "indigo"
	linh.Do("PUT", eng+"/brand-kit", bad).MustStatus(t, 422)
	linh.Do("PUT", eng+"/brand-kit", input("0192e8f0-0000-7000-8000-000000000000")).MustStatus(t, 422)

	logo := upload(t, linh, eng, "acme-logo.png", logoPNG(t))
	linh.Do("PUT", eng+"/brand-kit", input(logo)).MustStatus(t, 200).JSON(t, &k)
	if !k.Saved || k.Colors.Button != "#4338CA" || k.LogoLight == nil || k.LogoLight.MediaID != logo || k.Button.Shape != "pill" || len(k.Synced) != 2 {
		t.Fatalf("saved: %+v", k)
	}
	footer, header := k.Synced[0], k.Synced[1]
	if footer.Kind != "footer" || !strings.Contains(footer.MJML, `rc-synced="footer"`) || !strings.Contains(footer.MJML, "{{ unsubscribe_url }}") || !strings.Contains(footer.MJML, "color:#4338CA") {
		t.Fatalf("footer: %s", footer.MJML)
	}
	if header.Kind != "header" || !strings.Contains(header.MJML, k.LogoLight.URL) || !strings.Contains(header.MJML, `rc-logo="light"`) || !strings.Contains(header.MJML, `alt="Acme"`) {
		t.Fatalf("header: %s", header.MJML)
	}

	// Starters start from the kit: its colours, fonts, button, logo and synced blocks.
	var tpl struct{ MJML string }
	linh.Do("GET", eng+"/templates/starter:weekly-digest", nil).MustStatus(t, 200).JSON(t, &tpl)
	for _, want := range []string{`background-color="#4338CA"`, `border-radius="999px"`, `font-size="15px"`, `'Inter', Arial, Helvetica, sans-serif`, `<mj-font name="Inter"`,
		`rc-synced="footer"`, `rc-synced="header"`, k.LogoLight.URL, `background-color="#F5F3FF"`} {
		if !strings.Contains(tpl.MJML, want) {
			t.Fatalf("starter misses %s:\n%s", want, tpl.MJML)
		}
	}
	if strings.Contains(tpl.MJML, "#0F766E") || strings.Contains(tpl.MJML, "/media/starters/logo.png") {
		t.Fatalf("starter keeps defaults:\n%s", tpl.MJML)
	}

	// Emails holding the footer: a draft, a sent one and one in an approval.
	doc := func(block string) string {
		return `<mjml><mj-body><mj-section><mj-column><mj-text>News</mj-text></mj-column></mj-section>` + block + `</mj-body></mjml>`
	}
	draft := campaign(t, linh, eng, doc(footer.MJML))
	sent := campaign(t, linh, eng, doc(footer.MJML))
	held := campaign(t, linh, eng, doc(footer.MJML))
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE campaigns SET status = 'sent' WHERE id = $1`, sent); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE campaigns SET status = 'approval' WHERE id = $1`, held); err != nil {
		t.Fatal(err)
	}
	linh.Do("GET", eng+"/brand-kit", nil).MustStatus(t, 200).JSON(t, &k)
	if u := k.Synced[0].Uses; u.Drafts != 1 || u.Sent != 1 || u.Held != 1 {
		t.Fatalf("uses: %+v", u)
	}

	// The footer needs its unsubscribe link, and is one section.
	linh.Do("PUT", eng+"/brand-kit/synced/footer", map[string]any{"mjml": `<mj-section><mj-column><mj-text>No way out</mj-text></mj-column></mj-section>`}).MustStatus(t, 422)
	linh.Do("PUT", eng+"/brand-kit/synced/footer", map[string]any{"mjml": `<mj-text>{{ unsubscribe_url }}</mj-text>`}).MustStatus(t, 422)
	aisha.Do("PUT", ws+"/brand-kit/synced/footer", map[string]any{"mjml": footer.MJML}).MustStatus(t, 403)

	next := `<mj-section background-color="#F5F3FF" padding="18px 24px"><mj-column><mj-text align="center">Acme Inc · 77 Harbour Rd<br><a href="{{ unsubscribe_url }}">Unsubscribe</a> · <a href="{{ preferences_url }}">Email preferences</a></mj-text></mj-column></mj-section>`
	var saved struct {
		Block   struct{ Version int }
		Applied struct{ Drafts, Scheduled, Held, Sent int }
	}
	linh.Do("PUT", eng+"/brand-kit/synced/footer", map[string]any{"mjml": next}).MustStatus(t, 200).JSON(t, &saved)
	if saved.Block.Version != 2 || saved.Applied.Drafts != 1 || saved.Applied.Sent != 1 || saved.Applied.Held != 1 {
		t.Fatalf("saved: %+v", saved)
	}
	content := func(id string) (mjml, html string) {
		if err := env.App.Pool.QueryRow(context.Background(), `SELECT mjml, html FROM campaigns WHERE id = $1`, id).Scan(&mjml, &html); err != nil {
			t.Fatal(err)
		}
		return
	}
	if m, h := content(draft); !strings.Contains(m, "77 Harbour Rd") || !strings.Contains(m, `rc-synced="footer"`) || !strings.Contains(h, "77 Harbour Rd") || strings.Contains(h, "rc-synced") {
		t.Fatalf("draft: %s", m)
	}
	for _, id := range []string{sent, held} {
		if m, _ := content(id); strings.Contains(m, "77 Harbour Rd") {
			t.Fatalf("%s changed", id)
		}
	}
	var source, label string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT source, label FROM campaign_versions WHERE campaign_id = $1 ORDER BY version DESC LIMIT 1`, draft).Scan(&source, &label); err != nil || source != "synced" || label != "Footer updated in Brand kit" {
		t.Fatalf("version: %s %s %v", source, label, err)
	}

	// A new logo reaches the synced Header and the drafts holding it.
	withHeader := campaign(t, linh, eng, doc(header.MJML))
	logo2 := upload(t, linh, eng, "acme-logo-2.png", append(logoPNG(t), 0)) // other bytes, another image
	in := input(logo2)
	in["logoAlt"] = "Acme, Inc."
	linh.Do("PUT", eng+"/brand-kit", in).MustStatus(t, 200).JSON(t, &k)
	if k.Synced[1].Version != 2 || !strings.Contains(k.Synced[1].MJML, k.LogoLight.URL) || !strings.Contains(k.Synced[1].MJML, `alt="Acme, Inc."`) {
		t.Fatalf("header after the new logo: %s", k.Synced[1].MJML)
	}
	if m, _ := content(withHeader); !strings.Contains(m, k.LogoLight.URL) {
		t.Fatalf("draft header: %s", m)
	}

	for action, want := range map[string]int{"brand_kit.save": 2, "synced_block.save": 2} {
		var n int
		if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = $1`, action).Scan(&n); err != nil || n != want {
			t.Fatalf("audit %s: %d, want %d (%v)", action, n, want, err)
		}
	}
}

// The site: a theme colour too light for links, a stylesheet, Google fonts
// and a logo.
func fakeSite(t *testing.T, logo []byte) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`<!doctype html><html><head><title>Acme | Feature flags</title>
<meta name="theme-color" content="#6366F1">
<link rel="preconnect" href="https://fonts.gstatic.com">
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Source+Serif+4:wght@600&family=Inter:wght@400;700&display=swap">
<link rel="stylesheet" href="/site.css">
<style>.hero{background:#F5F3FF}</style></head>
<body><header><a href="/"><img class="site-logo" src="/img/acme-logo.png" alt="Acme"></a></header>
<h1>Ship safely</h1><p>Hello</p></body></html>`))
	})
	mux.HandleFunc("/site.css", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`:root{--brand-primary:#4338CA}
body{font-family:Inter,Arial,sans-serif;color:#1F2937}
h1,h2{font-family:"Source Serif 4",Georgia,serif}
a{color:#4338CA}.btn{background:#4338CA;color:#fff}.card{border-color:#E5E7EB}.tint{background:#F5F3FF}`))
	})
	mux.HandleFunc("/img/acme-logo.png", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(logo)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestImportFromWebsite(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	srv := fakeSite(t, logoPNG(t))

	var imp struct {
		Website   string
		LogoLight *struct{ MediaID, URL string }
		LogoAlt   string
		Style     struct {
			Colors  struct{ Button, Text, Background, Link string }
			Heading struct{ Name, URL, Fallback string }
			Body    struct{ Name, URL, Fallback string }
		}
		Colors []struct {
			Hex  string
			Used bool
		}
		Fonts []string
		Notes []string
	}
	linh.Do("POST", eng+"/brand-kit/import", map[string]any{"website": srv.URL}).MustStatus(t, 200).JSON(t, &imp)
	st := imp.Style
	if st.Colors.Button != "#4338CA" || st.Colors.Link != "#4338CA" || st.Colors.Text != "#1F2937" || st.Colors.Background != "#F5F3FF" {
		t.Fatalf("colours: %+v", st.Colors)
	}
	if st.Heading.Name != "Source Serif 4" || st.Heading.Fallback != "Georgia, 'Times New Roman', serif" || st.Body.Name != "Inter" || !strings.HasPrefix(st.Body.URL, "https://fonts.googleapis.com/css2?family=Inter") {
		t.Fatalf("fonts: %+v %+v", st.Heading, st.Body)
	}
	if imp.LogoLight == nil || imp.LogoAlt != "Acme" {
		t.Fatalf("logo: %+v %q", imp.LogoLight, imp.LogoAlt)
	}
	unused := false
	for _, c := range imp.Colors {
		if c.Hex == "#6366F1" && !c.Used {
			unused = true
		}
	}
	if !unused || len(imp.Notes) != 1 || !strings.Contains(imp.Notes[0], "#6366F1 isn't used for links. It's 4.47:1 on white") {
		t.Fatalf("notes: %+v %v", imp.Colors, imp.Notes)
	}
	// Nothing saved.
	var k kit
	linh.Do("GET", eng+"/brand-kit", nil).MustStatus(t, 200).JSON(t, &k)
	if k.Saved {
		t.Fatal("an import saves nothing")
	}
	linh.Do("POST", eng+"/brand-kit/import", map[string]any{"website": "http://127.0.0.1:1"}).MustStatus(t, 502)
}

func TestContrastAndApply(t *testing.T) {
	if r := brandkit.Contrast("#4338CA", "#FFFFFF"); math.Abs(r-7.9) > 0.1 {
		t.Fatalf("#4338CA on white: %.2f", r)
	}
	if r := brandkit.Contrast("#6366F1", "#FFFFFF"); math.Round(r*100)/100 != 4.47 {
		t.Fatalf("#6366F1 on white: %.3f", r)
	}
	k := brandkit.BrandKit{Saved: true, BrandStyle: brandkit.DefaultStyle()}
	k.Colors = brandkit.BrandColors{Button: "#4338CA", Text: "#111111", Background: "#F5F3FF", Link: "#4338CA"}
	k.Heading = brandkit.BrandFont{Name: "Source Serif 4", Fallback: "Georgia, 'Times New Roman', serif"}
	k.Button = brandkit.BrandButtonStyle{Shape: "square", Size: 15}
	src := `<mj-section background-color="#F3F4F6"><mj-column><mj-text color="#1F2937"><h1 style="margin:0">Hi</h1></mj-text>` +
		`<mj-text color="#1F2937">See <a href="x" style="color:#0F766E">this</a></mj-text>` +
		`<mj-button background-color="#0F766E" font-family="'IBM Plex Sans', Helvetica, Arial, sans-serif" font-size="13px" border-radius="6px" inner-padding="9px 14px">Go</mj-button></mj-column></mj-section>`
	out := brandkit.Apply(src, k)
	for _, want := range []string{`background-color="#F5F3FF"`, `color="#111111"`, `style="color:#4338CA"`, `background-color="#4338CA"`, `border-radius="0px"`, `font-size="15px"`,
		`inner-padding="14px 22px"`, `font-family="'Source Serif 4', Georgia, 'Times New Roman', serif"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("apply misses %s:\n%s", want, out)
		}
	}
	if brandkit.Apply(src, brandkit.BrandKit{}) != src {
		t.Fatal("no kit, no change")
	}
}
