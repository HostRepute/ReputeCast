package forms

// Unit tests of the forms package: no database. Integration tests that run
// the HTTP API are in forms_api_test.go (package forms_test).

import (
	"bytes"
	"cmp"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// --- colours -----------------------------------------------------------------

func mustColor(t *testing.T, s string) rgb {
	t.Helper()
	c, ok := ParseColor(s)
	if !ok {
		t.Fatalf("ParseColor(%q) failed", s)
	}
	return c
}

func TestParseColor(t *testing.T) {
	short, ok := ParseColor("#43C")
	if !ok {
		t.Fatal("#43C doesn't parse")
	}
	long := mustColor(t, "#4433CC")
	if short != long {
		t.Fatalf("#43C = %+v, want %+v (#4433CC)", short, long)
	}
	if got := short.Hex(); got != "#4433CC" {
		t.Fatalf("#43C.Hex() = %s, want #4433CC", got)
	}
	for _, s := range []string{"#4338ca", "4338CA", " #4338CA ", "#4338Ca"} {
		if c, ok := ParseColor(s); !ok || c.Hex() != "#4338CA" {
			t.Errorf("ParseColor(%q) = %v %v, want #4338CA", s, c.Hex(), ok)
		}
	}
	for _, s := range []string{"", "#", "#43", "#4338C", "#4338CAA", "#GGGGGG", "red", "#43C8", "rgb(1,2,3)"} {
		if _, ok := ParseColor(s); ok {
			t.Errorf("ParseColor(%q) accepted an invalid colour", s)
		}
	}
	if got := Contrast(mustColor(t, "#000"), mustColor(t, "#FFF")); got < 20.99 || got > 21.01 {
		t.Errorf("black on white is %.3f:1, want 21:1", got)
	}
	if got := Contrast(mustColor(t, "#777"), mustColor(t, "#777")); got != 1 {
		t.Errorf("a colour on itself is %.3f:1, want 1:1", got)
	}
}

// contrastMiss is a derived text colour under WCAG AA.
type contrastMiss struct {
	what  string
	ratio float64
}

func (m contrastMiss) String() string { return fmt.Sprintf("%s is %.3f:1", m.what, m.ratio) }

// paletteMisses lists the derived text colours of p that miss WCAG AA
// (4.5:1). Contrasts are measured on the hex values the pages use.
func paletteMisses(t *testing.T, p Palette, buttons bool) []contrastMiss {
	t.Helper()
	white := mustColor(t, "#FFFFFF")
	bg, accent := mustColor(t, p.Background), mustColor(t, p.Accent)
	text, hover, soft, on := mustColor(t, p.Text), mustColor(t, p.Hover), mustColor(t, p.Soft), mustColor(t, p.OnAccent)
	var out []contrastMiss
	check := func(what string, a, b rgb) {
		if c := Contrast(a, b); c < 4.5 {
			out = append(out, contrastMiss{what, c})
		}
	}
	if buttons {
		check("button text "+p.OnAccent+" on "+p.Accent, on, accent)
		return out
	}
	check("link text "+p.Text+" on white", text, white)
	check("link text "+p.Text+" on "+p.Background, text, bg)
	check("hover "+p.Hover+" on white", hover, white)
	check("hover "+p.Hover+" on "+p.Background, hover, bg)
	check("hover "+p.Hover+" on tint "+p.Soft, hover, soft)
	return out
}

func paletteProblems(t *testing.T, p Palette, buttons bool) []string {
	t.Helper()
	var out []string
	for _, m := range paletteMisses(t, p, buttons) {
		out = append(out, m.String())
	}
	return out
}

func TestDerivePaletteDesignColours(t *testing.T) {
	p, err := DerivePalette("#4338CA", "#F5F3FF")
	if err != nil {
		t.Fatal(err)
	}
	if p.Accent != "#4338CA" || p.Background != "#F5F3FF" {
		t.Fatalf("palette keeps the publisher's colours: %+v", p)
	}
	if p.OnAccent != "#FFFFFF" {
		t.Errorf("white is readable on #4338CA, got button text %s", p.OnAccent)
	}
	if p.Text != "#4338CA" || len(p.Notes) != 0 {
		t.Errorf("#4338CA already reaches 4.5:1 on white and #F5F3FF: text %s, notes %v", p.Text, p.Notes)
	}
	if probs := paletteProblems(t, p, false); len(probs) > 0 {
		t.Errorf("design colours: %v", probs)
	}
	if probs := paletteProblems(t, p, true); len(probs) > 0 {
		t.Errorf("design colours: %v", probs)
	}
	if p.Notes == nil {
		t.Error("Notes must be an empty list, not nil (it is serialized)")
	}
}

func TestDerivePaletteAdjustsForContrast(t *testing.T) {
	cases := []struct {
		name, accent, bg string
		darkButtonText   bool
		darkerLinks      bool
	}{
		{"yellow accent gets dark button text and darker links", "#FACC15", "#FFFFFF", true, true},
		{"light teal on a mint page", "#2DD4BF", "#ECFDF5", true, true},
		{"sky blue", "#0EA5E9", "#F0F9FF", true, true},
		{"orange on cream", "#F97316", "#FFF7ED", true, true},
		{"near black", "#111111", "#FFFFFF", false, false},
		{"dark green", "#166534", "#F0FDF4", false, false},
		{"pure red gets dark button text (white is 4.0:1)", "#FF0000", "#FFFFFF", true, true},
		{"white accent", "#FFFFFF", "#FFFFFF", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := DerivePalette(tc.accent, tc.bg)
			if err != nil {
				t.Fatal(err)
			}
			if got := p.OnAccent == inkColor; got != tc.darkButtonText {
				t.Errorf("button text %s (dark: %v, want %v)", p.OnAccent, got, tc.darkButtonText)
			}
			if got := p.Text != p.Accent; got != tc.darkerLinks {
				t.Errorf("link colour %s for accent %s (darkened: %v, want %v)", p.Text, p.Accent, got, tc.darkerLinks)
			}
			if tc.darkButtonText || tc.darkerLinks {
				if len(p.Notes) == 0 {
					t.Error("an adjusted palette says what was adjusted (notes)")
				}
			}
			for _, probs := range [][]string{paletteProblems(t, p, false), paletteProblems(t, p, true)} {
				if len(probs) > 0 {
					t.Errorf("%s on %s: %v", tc.accent, tc.bg, probs)
				}
			}
		})
	}
}

// sweep runs DerivePalette over a grid of accents (and, for the text
// colours, light backgrounds) and returns the misses, worst first.
func sweep(t *testing.T, buttons bool) (fails []string, total int) {
	t.Helper()
	backgrounds := []string{"#FFFFFF", "#F5F3FF", "#FEF3C7", "#ECFDF5", "#F1F5F9"}
	if buttons {
		backgrounds = backgrounds[:1] // button text doesn't depend on the page
	}
	var misses []contrastMiss
	for r := 0; r < 256; r += 17 {
		for g := 0; g < 256; g += 17 {
			for b := 0; b < 256; b += 17 {
				accent := fmt.Sprintf("#%02X%02X%02X", r, g, b)
				for _, bg := range backgrounds {
					total++
					p, err := DerivePalette(accent, bg)
					if err != nil {
						t.Fatalf("%s on %s: %v", accent, bg, err)
					}
					for _, m := range paletteMisses(t, p, buttons) {
						m.what = "accent " + accent + " on " + bg + ": " + m.what
						misses = append(misses, m)
					}
				}
			}
		}
	}
	slices.SortFunc(misses, func(a, b contrastMiss) int { return cmp.Compare(a.ratio, b.ratio) })
	for _, m := range misses {
		fails = append(fails, m.String())
	}
	return fails, total
}

// Every link, hover and tint text reaches 4.5:1, whatever the publisher
// picks. darkenUntil checks the contrast of the unrounded colour and stops
// at the first shade over 4.5:1; Hex() then rounds the channels, and the
// colour the pages use can land just under (#007F61 on #FEF3C7 is 4.48:1).
func TestDerivePaletteTextAlwaysReachesAA(t *testing.T) {
	fails, total := sweep(t, false)
	if len(fails) > 0 {
		t.Errorf("%d of %d palettes have text under 4.5:1 after rounding to hex (theme.go darkenUntil: check the rounded colour, or aim a little above 4.5), worst:\n%s",
			len(fails), total, strings.Join(fails[:min(len(fails), 5)], "\n"))
	}
}

// Button text reaches 4.5:1 on the accent. DerivePalette uses white, and
// ink (#0B1220) when white misses 4.5:1, without checking ink: for accents
// whose luminance is around 0.17 to 0.21 ink is worse than white or misses
// too. #0077DD, a common brand blue, gets ink at 4.18:1 where white was
// 4.48:1; the mid grey #797979 reaches neither (4.35:1 white, 4.30:1 ink).
// Black (#000000) reaches 4.5:1 on every accent where white doesn't.
func TestDerivePaletteButtonTextReachesAA(t *testing.T) {
	for _, accent := range []string{"#0077DD", "#797979"} {
		p, err := DerivePalette(accent, "#FFFFFF")
		if err != nil {
			t.Fatal(err)
		}
		if probs := paletteProblems(t, p, true); len(probs) > 0 {
			a := mustColor(t, accent)
			t.Errorf("%v; white would be %.2f:1, black %.2f:1 (theme.go DerivePalette: when white fails, pick ink only if it reaches 4.5:1, else #000000)",
				probs, Contrast(white, a), Contrast(black, a))
		}
	}
	fails, total := sweep(t, true)
	if len(fails) > 0 {
		t.Errorf("%d of %d accents have button text under 4.5:1, worst:\n%s", len(fails), total, strings.Join(fails[:min(len(fails), 5)], "\n"))
	}
}

func TestDerivePaletteRejects(t *testing.T) {
	cases := []struct {
		accent, bg, want string
	}{
		{"#4338CA", "#333333", "too dark for the page background"},
		{"#4338CA", "#000000", "too dark"},
		{"#4338CA", "#6B7280", "too dark"},
		{"blue", "#FFFFFF", "isn't a colour"},
		{"#4338CA", "#FFFFF", "isn't a colour"},
	}
	for _, tc := range cases {
		_, err := DerivePalette(tc.accent, tc.bg)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("DerivePalette(%s, %s) = %v, want an error containing %q", tc.accent, tc.bg, err, tc.want)
		}
	}
	if _, err := DerivePalette("#4338CA", "#F5F3FF"); err != nil {
		t.Errorf("the design background is light enough: %v", err)
	}
}

// --- origins -------------------------------------------------------------------

func TestNormalizeOrigin(t *testing.T) {
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"https://Docs.Acme.io/path?q=1", "docs.acme.io", true},
		{"docs.acme.io", "docs.acme.io", true},
		{"  BLOG.acme.io  ", "blog.acme.io", true},
		{"http://staging.acme.io:8080/x", "staging.acme.io:8080", true},
		{"staging.acme.io:8443", "staging.acme.io:8443", true},
		{"acme.io.", "acme.io", true},
		{"localhost", "localhost", true},
		{"http://localhost:8080", "localhost:8080", true},
		{"xn--bcher-kva.example", "xn--bcher-kva.example", true},
		{"", "", false},
		{"   ", "", false},
		{"acme", "", false},
		{"*.acme.io", "", false},
		{"acme_io.com", "", false},
		{"https://user:pw@acme.io", "", false},
		{"not a domain", "", false},
		{"javascript:alert(1)", "", false},
		{"192.168.1.10", "", false},
		{"-acme.io", "", false},
	}
	for _, tc := range cases {
		got, ok := NormalizeOrigin(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("NormalizeOrigin(%q) = %q, %v; want %q, %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestOriginHost(t *testing.T) {
	cases := []struct{ origin, referer, want string }{
		{"https://Docs.Acme.io", "", "docs.acme.io"},
		{"https://acme.io:8443", "", "acme.io:8443"},
		{"", "https://blog.acme.io/post/1?x=y", "blog.acme.io"},
		{"null", "https://blog.acme.io/post", "blog.acme.io"},
		{"https://acme.io", "https://evil.example/", "acme.io"},
		{"", "", ""},
		{"null", "", ""},
		{"not a url", "", ""},
	}
	for _, tc := range cases {
		if got := OriginHost(tc.origin, tc.referer); got != tc.want {
			t.Errorf("OriginHost(%q, %q) = %q, want %q", tc.origin, tc.referer, got, tc.want)
		}
	}
}

func TestOriginAllowed(t *testing.T) {
	allowed := []string{"acme.io", "docs.acme.io", "blog.acme.io", "staging.acme.io:8443"}
	own := []string{"localhost:8080", "news.acme.io"}
	cases := []struct {
		host string
		want bool
	}{
		{"acme.io", true},
		{"ACME.io", true},
		{"docs.acme.io", true},
		{"acme.io:8443", true}, // an entry without a port allows any port
		{"staging.acme.io:8443", true},
		{"staging.acme.io:9000", false}, // an entry with a port needs that port
		{"staging.acme.io", false},
		{"www.acme.io", false}, // no wildcard subdomains
		{"acme.io.evil.example", false},
		{"evilacme.io", false},
		{"localhost:8080", true}, // the instance
		{"localhost:9090", false},
		{"localhost", false},
		{"news.acme.io", true}, // the custom domain
		{"news.acme.io:443", true},
		{"", false},
	}
	for _, tc := range cases {
		if got := OriginAllowed(tc.host, allowed, own); got != tc.want {
			t.Errorf("OriginAllowed(%q) = %v, want %v", tc.host, got, tc.want)
		}
	}
	if OriginAllowed("acme.io", nil, nil) {
		t.Error("a form without allowed sites only takes its own origins")
	}
	if !OriginAllowed("[::1]:8080", nil, []string{"[::1]:8080"}) {
		t.Error("IPv6 own origin with a port")
	}
}

// --- slugs, labels, excerpts ------------------------------------------------------

func TestSlugify(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"Newsletter signup", 50, "newsletter-signup"},
		{"  Footer   mini form  ", 50, "footer-mini-form"},
		{"Héllo Wörld!", 50, "hello-world"},
		{"Ça va? Très bien", 50, "ca-va-tres-bien"},
		{"Straße & Ærø", 50, "strasse-aero"},
		{"Sam's form", 50, "sams-form"},
		{"Beta & Partners — 2026", 50, "beta-partners-2026"},
		{"!!!", 50, ""},
		{"", 50, ""},
		{"abcd efgh", 5, "abcd"},
		{"a very long form name", 10, "a-very-lon"},
		{"The case for boring infrastructure", 80, "the-case-for-boring-infrastructure"},
	}
	for _, tc := range cases {
		if got := Slugify(tc.in, tc.max); got != tc.want {
			t.Errorf("Slugify(%q, %d) = %q, want %q", tc.in, tc.max, got, tc.want)
		}
		if got := Slugify(tc.in, tc.max); len(got) > tc.max || strings.HasPrefix(got, "-") || strings.HasSuffix(got, "-") {
			t.Errorf("Slugify(%q, %d) = %q: too long or dangling dash", tc.in, tc.max, got)
		}
	}
}

func TestIssueLabel(t *testing.T) {
	cases := map[string]string{
		"Weekly Digest #141":     "#141",
		"Weekly Digest #141  ":   "#141",
		"Weekly Digest #7":       "#7",
		"October Product Update": "",
		"#12 things we shipped":  "",
		"Issue #1234567":         "",
		"Weekly Digest 141":      "",
		"":                       "",
	}
	for in, want := range cases {
		if got := IssueLabel(in); got != want {
			t.Errorf("IssueLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExcerpt(t *testing.T) {
	cases := []struct {
		in   string
		n    int
		want string
	}{
		{"Short enough.", 180, "Short enough."},
		{"  spaces\n\n and\tnewlines  collapse ", 180, "spaces and newlines collapse"},
		{"one two three four five six", 12, "one two…"},
		{"Feature flags are generally available, audit logs stream", 40, "Feature flags are generally available…"},
		{"abcdefghijklmnop qrs", 10, "abcdefghij…"},
		{"Hello, world. Again and again", 13, "Hello, world…"},
		{"éé éé éé éé éé éé", 8, "éé éé…"},
	}
	for _, tc := range cases {
		got := Excerpt(tc.in, tc.n)
		if got != tc.want {
			t.Errorf("Excerpt(%q, %d) = %q, want %q", tc.in, tc.n, got, tc.want)
		}
		if strings.HasSuffix(got, "…") && strings.Contains(strings.TrimSuffix(got, "…"), "  ") {
			t.Errorf("Excerpt(%q) left double spaces: %q", tc.in, got)
		}
	}
}

// --- timing tokens ------------------------------------------------------------------

func testBox(t *testing.T) *secrets.Box {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(secrets.GenerateKey())
	if err != nil {
		t.Fatal(err)
	}
	box, err := secrets.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	return box
}

func TestFormToken(t *testing.T) {
	s := &Service{Box: testBox(t)}
	now := time.Now()
	tok := s.FormToken("frm_8c2e1a", now.Add(-10*time.Second))
	if tok == "" || len(tok) > 200 {
		t.Fatalf("token %q", tok)
	}
	if strings.ContainsAny(tok, "+/=") {
		t.Errorf("token %q isn't URL-safe base64 without padding", tok)
	}
	age, ok := s.tokenAge("frm_8c2e1a", tok, now)
	if !ok || age < 9*time.Second || age > 11*time.Second {
		t.Fatalf("age %v ok %v, want about 10s", age, ok)
	}
	if _, ok := s.tokenAge("frm_3f91d0", tok, now); ok {
		t.Error("a token of one form opened for another form")
	}
	if _, ok := s.tokenAge("builtin:"+uuid.NewString(), tok, now); ok {
		t.Error("a form's token opened for the built-in form")
	}
	// Issued yesterday: too old.
	old := s.FormToken("frm_8c2e1a", now.Add(-25*time.Hour))
	if _, ok := s.tokenAge("frm_8c2e1a", old, now); ok {
		t.Error("a token older than a day is accepted")
	}
	// Clock skew of a few seconds is fine; a token from the future isn't.
	if _, ok := s.tokenAge("frm_8c2e1a", s.FormToken("frm_8c2e1a", now.Add(30*time.Second)), now); !ok {
		t.Error("30 seconds of clock skew is refused")
	}
	if _, ok := s.tokenAge("frm_8c2e1a", s.FormToken("frm_8c2e1a", now.Add(2*time.Minute)), now); ok {
		t.Error("a token from two minutes in the future is accepted")
	}
	// Forged, cut or empty tokens.
	raw, _ := base64.RawURLEncoding.DecodeString(tok)
	raw[len(raw)-1] ^= 1
	for name, bad := range map[string]string{
		"empty":     "",
		"garbage":   "not-a-token",
		"tampered":  base64.RawURLEncoding.EncodeToString(raw),
		"truncated": tok[:len(tok)-4],
		"too long":  strings.Repeat("A", 201),
	} {
		if _, ok := s.tokenAge("frm_8c2e1a", bad, now); ok {
			t.Errorf("%s token accepted", name)
		}
	}
	// Another instance's key can't open it.
	other := &Service{Box: testBox(t)}
	if _, ok := other.tokenAge("frm_8c2e1a", tok, now); ok {
		t.Error("a token sealed with another key opened")
	}
	// Without a key there are no tokens.
	none := &Service{}
	if none.FormToken("frm_8c2e1a", now) != "" {
		t.Error("FormToken without a Box returned a token")
	}
	if _, ok := none.tokenAge("frm_8c2e1a", tok, now); ok {
		t.Error("tokenAge without a Box accepted a token")
	}
}

func TestFormKey(t *testing.T) {
	ws := uuid.Must(uuid.NewV7())
	if got := formKey(Config{PublicID: "frm_8c2e1a"}); got != "frm_8c2e1a" {
		t.Errorf("formKey = %q", got)
	}
	if got := formKey(Config{Builtin: true, WorkspaceID: ws, PublicID: "frm_8c2e1a"}); got != "builtin:"+ws.String() {
		t.Errorf("built-in formKey = %q", got)
	}
}

// --- form configuration ---------------------------------------------------------------

func TestConfigOfDefaults(t *testing.T) {
	c := ConfigOf(store.Form{Name: "Bare", Slug: "bare", PublicID: "frm_abcdef", Status: StatusLive, ConsentText: "I agree.", ConsentVersion: 3,
		Fields: []byte(`[{"key":"first_name"}]`), Content: []byte(`{}`), SpamProtection: []byte(`{"honeypot":true}`)})
	if len(c.Fields) != 2 || c.Fields[0].Key != FieldEmail || !c.Fields[0].Required {
		t.Errorf("the email field comes first and is required: %+v", c.Fields)
	}
	if c.Spam.MinSeconds != 3 || c.Spam.RateLimitPerIP != 5 || !c.Spam.Honeypot {
		t.Errorf("spam defaults: %+v", c.Spam)
	}
	if c.Content.Button != "Subscribe" || c.Content.EmailPlaceholder == "" || c.Content.PopupDelay != 8 || c.Content.PopupRepeatDays != 14 {
		t.Errorf("content defaults: %+v", c.Content)
	}
	if c.Lists == nil || c.DefaultLists == nil || c.AllowedOrigins == nil {
		t.Error("lists and origins are never nil")
	}
	if c.ConsentVersionLabel() != "v3" {
		t.Errorf("consent version label %q, want v3", c.ConsentVersionLabel())
	}
	if (Config{}).ConsentVersionLabel() != "" {
		t.Error("a form without consent text has no consent version")
	}
	if c.Source() != "form:bare" {
		t.Errorf("source %q", c.Source())
	}
	if b := BuiltinForm(&store.Workspace{ID: uuid.Must(uuid.NewV7())}, "Acme", nil); b.Source() != audience.SourceSubscribePage || !b.Builtin || b.Status != StatusLive {
		t.Errorf("built-in form: %+v", b)
	}
}

func TestBuiltinFormOffersPublicLists(t *testing.T) {
	pub1, priv, pub2 := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	ws := &store.Workspace{ID: uuid.Must(uuid.NewV7())}
	c := BuiltinForm(ws, "Acme", []audience.PublicList{{ID: pub1, Name: "Digest", Public: true}, {ID: priv, Name: "VIP"}, {ID: pub2, Name: "Updates", Public: true}})
	if len(c.Lists) != 2 || c.Lists[0] != pub1 || c.Lists[1] != pub2 {
		t.Errorf("offered %v, want the public lists", c.Lists)
	}
	if len(c.DefaultLists) != 1 || c.DefaultLists[0] != pub1 {
		t.Errorf("defaults %v, want the first public list", c.DefaultLists)
	}
	if !c.DoubleOptIn || !strings.Contains(c.ConsentText, "Acme") || c.Content.Heading != "Subscribe to Acme" {
		t.Errorf("built-in form: %+v", c)
	}
}

// --- plain HTML and the embed script -------------------------------------------------

func embedFixture() (Config, PageLinks, []audience.PublicList, []store.CustomField, Branding) {
	digest, updates := uuid.MustParse("0192f0a0-0000-7000-8000-000000000001"), uuid.MustParse("0192f0a0-0000-7000-8000-000000000002")
	c := Config{
		ID: uuid.Must(uuid.NewV7()), Name: "Newsletter signup", Kind: KindEmbedded, Slug: "newsletter-signup", PublicID: "frm_8c2e1a", Status: StatusLive,
		Fields:     []fieldDoc{{Key: FieldEmail, Required: true}, {Key: FieldFirstName, Label: `Name <b>"you"</b>`}, {Key: "role", Required: true}},
		Lists:      []uuid.UUID{digest, updates},
		ListChoice: true, DefaultLists: []uuid.UUID{digest}, DoubleOptIn: true,
		ConsentText: `I agree to emails from "Acme & Co" <script>alert(1)</script>. See the privacy policy.`, ConsentVersion: 3,
		Spam: defaultSpam(),
	}
	c.Content = withContentDefaults(contentDoc{Heading: `</script><script>alert("x")</script>`, Description: "line\u2028separator", Button: `Join "now" & <go>`}, true)
	links := LinksOn("https://rc.example", "acme-newsletter", false)
	lists := []audience.PublicList{{ID: digest, Name: "Weekly <Digest>", Public: true}, {ID: updates, Name: "Product & Updates", Public: true}}
	customs := []store.CustomField{{Key: "role", Label: "Role", Type: "select", Options: []byte(`["Engineer","Designer"]`)}}
	b := Branding{PublisherName: "Acme", PrivacyURL: "https://acme.io/privacy?a=1&b=2", Palette: Palette{Accent: "#4338CA", Hover: "#3730A3", OnAccent: "#FFFFFF", Text: "#4338CA"}}
	return c, links, lists, customs, b
}

func TestPlainHTML(t *testing.T) {
	c, links, lists, customs, b := embedFixture()
	out := PlainHTML(c, links, lists, customs, b)
	for _, want := range []string{
		`<form action="https://rc.example/p/acme-newsletter/subscribe/newsletter-signup" method="post"`,
		`<input type="hidden" name="_plain" value="1">`,
		`name="email"`, `type="email"`, `name="first_name"`, `name="f.role" required`,
		`Name &lt;b&gt;&#34;you&#34;&lt;/b&gt;`,
		`value="0192f0a0-0000-7000-8000-000000000001" checked>`,
		`value="0192f0a0-0000-7000-8000-000000000002">`,
		`Weekly &lt;Digest&gt;`, `Product &amp; Updates`,
		`name="consent" value="1" required`,
		`&lt;script&gt;alert(1)&lt;/script&gt;`,
		`<a href="https://acme.io/privacy?a=1&amp;b=2">privacy policy</a>.`,
		`name="website" tabindex="-1"`,
		`Join &#34;now&#34; &amp; &lt;go&gt;</button>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("plain HTML lacks %s:\n%s", want, out)
		}
	}
	if strings.Contains(out, "<script") || strings.Contains(out, "<b>") || strings.Contains(out, "<Digest>") {
		t.Errorf("plain HTML lets markup through:\n%s", out)
	}
	if strings.Count(out, "<form") != 1 || !strings.HasSuffix(out, "</form>") {
		t.Errorf("one form element:\n%s", out)
	}
	// No honeypot when it is off; no list choice with one list; no privacy link without a URL.
	c.Spam.Honeypot = false
	c.Lists = c.Lists[:1]
	b.PrivacyURL = ""
	out = PlainHTML(c, links, lists, customs, b)
	if strings.Contains(out, `name="website"`) || strings.Contains(out, `name="list"`) || strings.Contains(out, "<a ") {
		t.Errorf("honeypot off, one list, no privacy URL:\n%s", out)
	}
}

// The consent sentence links "privacy policy" by searching a lowercased
// copy of the text and slicing the original with that index. Letters whose
// lowercase form has another UTF-8 length (Turkish "İ" is 2 bytes, "i" is 1)
// shift the index: the link wraps the wrong bytes, and a letter that grows
// when lowercased ("Ⱥ" → "ⱥ") slices past the end and panics.
func TestPlainHTMLPrivacyLinkWithNonASCIIConsent(t *testing.T) {
	c, links, lists, customs, b := embedFixture()
	for _, consent := range []string{
		"İstanbul Haber'den e-posta almayı kabul ediyorum. See the privacy policy.",
		"İİ privacy policy",
		"Ⱥ privacy policy",
	} {
		t.Run(consent, func(t *testing.T) {
			c.ConsentText = consent
			var out string
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("PlainHTML panicked: %v (embed.go: the privacy-policy index comes from strings.ToLower(text) but slices text)", r)
					}
				}()
				out = PlainHTML(c, links, lists, customs, b)
			}()
			if out == "" {
				return
			}
			line := ""
			for _, l := range strings.Split(out, "\n") {
				if strings.Contains(l, `name="consent"`) {
					line = strings.TrimSpace(l)
				}
			}
			before := html.EscapeString(strings.Split(consent, " privacy policy")[0])
			if !strings.Contains(line, before+` <a href="https://acme.io/privacy?a=1&amp;b=2">privacy policy</a>`) || !utf8.ValidString(line) {
				t.Errorf("the link must wrap exactly \"privacy policy\" and keep the sentence (embed.go PlainHTML: the index of strings.ToLower(text) slices text; use a case-insensitive search that returns indexes into the original text):\n%s", line)
			}
		})
	}
}

func TestEmbedScript(t *testing.T) {
	c, links, lists, customs, b := embedFixture()
	b.Logo = &logoMeta{Name: "acme-mark.svg", Type: "image/svg+xml", Hash: "abc123"}
	js := string(EmbedScript(c, links, b, lists, customs))
	if strings.Contains(js, "__RC_FORM__") {
		t.Fatal("the configuration placeholder is still in the script")
	}
	if strings.Contains(js, "</script>") || strings.Contains(js, "<script>") || strings.Contains(js, "<go>") {
		t.Fatal("form settings can close the script element (</script>) or inject markup")
	}
	// The configuration is the argument of the script's IIFE.
	i := strings.LastIndex(js, "})(")
	j := strings.LastIndex(js, ")")
	if i < 0 || j <= i+3 {
		t.Fatalf("no configuration argument in the script")
	}
	var cfg struct {
		ID, Kind, Submit, Session, CSS, Hosted, Heading, Button, Consent, Privacy, Logo, Publisher, Accent, OnAccent string
		Honeypot                                                                                                     bool
		ListChoice                                                                                                   bool
		Fields                                                                                                       []struct {
			Key, Name, Type string
			Required        bool
			Options         []string
		}
		Lists []struct {
			ID, Name string
			Checked  bool
		}
		Popup    map[string]int
		Messages map[string]struct{ Title, Text string }
	}
	if err := json.Unmarshal([]byte(js[i+3:j]), &cfg); err != nil {
		t.Fatalf("the configuration isn't JSON: %v\n%s", err, js[i+3:j])
	}
	if cfg.Heading != c.Content.Heading || cfg.Button != c.Content.Button || cfg.Consent != c.ConsentText {
		t.Errorf("text round trip: heading %q, button %q, consent %q", cfg.Heading, cfg.Button, cfg.Consent)
	}
	if cfg.ID != "frm_8c2e1a" || cfg.Submit != "https://rc.example/p/acme-newsletter/forms/frm_8c2e1a/submit" ||
		cfg.Session != "https://rc.example/p/acme-newsletter/forms/frm_8c2e1a/session" ||
		cfg.Hosted != "https://rc.example/p/acme-newsletter/subscribe/newsletter-signup" ||
		!strings.HasPrefix(cfg.CSS, "https://rc.example/p/_/embed.css?v=") {
		t.Errorf("links: %+v", cfg)
	}
	if cfg.Logo != "https://rc.example/p/acme-newsletter/logo?v=abc123" || cfg.Privacy != b.PrivacyURL || cfg.Publisher != "Acme" {
		t.Errorf("branding: logo %q privacy %q publisher %q", cfg.Logo, cfg.Privacy, cfg.Publisher)
	}
	if !cfg.Honeypot || !cfg.ListChoice || len(cfg.Lists) != 2 || !cfg.Lists[0].Checked || cfg.Lists[1].Checked {
		t.Errorf("lists and honeypot: %+v", cfg.Lists)
	}
	if len(cfg.Fields) != 3 || cfg.Fields[0].Type != "email" || !cfg.Fields[0].Required || cfg.Fields[2].Name != "f.role" ||
		cfg.Fields[2].Type != "select" || len(cfg.Fields[2].Options) != 2 {
		t.Errorf("fields: %+v", cfg.Fields)
	}
	if cfg.Messages[OutcomeConfirm].Title != "Check your inbox" || cfg.Popup["delay"] != 8 {
		t.Errorf("messages %+v, popup %+v", cfg.Messages, cfg.Popup)
	}
	// A failed submit shows the error's sentence (FormsStates: "We couldn't
	// save your subscription just now…"), never the message object.
	if cfg.Messages["error"].Text != msgSaveFailed || strings.Count(js, "cfg.messages.error.text") != 2 || strings.Contains(js, "cfg.messages.error)") {
		t.Errorf("the failed-submit message: %+v", cfg.Messages["error"])
	}
	if strings.ContainsRune(js, '\u2028') || strings.ContainsRune(js, '\u2029') {
		t.Error("a raw U+2028/U+2029 line separator from the settings is in the script")
	}
}

func TestPageLinks(t *testing.T) {
	inst := LinksOn("https://rc.example/", "acme-newsletter", false)
	custom := LinksOn("https://news.acme.io", "acme-newsletter", true)
	cases := []struct{ got, want string }{
		{inst.Subscribe(""), "https://rc.example/p/acme-newsletter/subscribe"},
		{inst.Subscribe("beta-apply"), "https://rc.example/p/acme-newsletter/subscribe/beta-apply"},
		{inst.Confirm("tok"), "https://rc.example/p/acme-newsletter/confirm/tok"},
		{inst.Archive(), "https://rc.example/p/acme-newsletter/archive"},
		{inst.Issue("a b"), "https://rc.example/p/acme-newsletter/archive/a%20b"},
		{inst.Feed(), "https://rc.example/p/acme-newsletter/archive.rss"},
		{custom.Subscribe(""), "https://news.acme.io/subscribe"},
		{custom.Unsubscribe("tok"), "https://news.acme.io/unsubscribe/tok"},
		{custom.Sitemap(), "https://news.acme.io/sitemap.xml"},
		// Scripts, submissions, themes and logos always live under /p/.
		{custom.Script("frm_8c2e1a"), "https://news.acme.io/p/acme-newsletter/forms/frm_8c2e1a.js"},
		{custom.Submit("frm_8c2e1a"), "https://news.acme.io/p/acme-newsletter/forms/frm_8c2e1a/submit"},
		{custom.PlainAction("beta-apply"), "https://news.acme.io/p/acme-newsletter/subscribe/beta-apply"},
		{custom.Logo("v1"), "https://news.acme.io/p/acme-newsletter/logo?v=v1"},
		{custom.Static("public.css"), "https://news.acme.io/p/_/public.css"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("got %s, want %s", tc.got, tc.want)
		}
	}
}

// --- logos ------------------------------------------------------------------------------

func TestSniffLogo(t *testing.T) {
	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 64)...)
	clean := []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><rect width="32" height="32" rx="8" fill="#4338CA"/><style>rect{opacity:.9}</style></svg>`)
	ok := []struct {
		name string
		data []byte
		want string
	}{
		{"clean svg", clean, "image/svg+xml"},
		{"png", png, "image/png"},
		{"gif", append([]byte("GIF89a"), bytes.Repeat([]byte{0}, 32)...), "image/gif"},
	}
	for _, tc := range ok {
		ctype, problem := sniffLogo(tc.data)
		if ctype != tc.want || problem != "" {
			t.Errorf("%s: %q %q, want %s", tc.name, ctype, problem, tc.want)
		}
	}
	bad := []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"too big", append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, MaxLogoBytes)...)},
		{"text", []byte("hello, I'm not an image")},
		{"html", []byte("<!doctype html><html><body>hi</body></html>")},
		{"script element", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)},
		{"script uppercase", []byte(`<SVG xmlns="http://www.w3.org/2000/svg"><SCRIPT>alert(1)</SCRIPT></SVG>`)},
		{"onload", []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"><rect/></svg>`)},
		{"onload late in the file", append(append([]byte(`<svg xmlns="http://www.w3.org/2000/svg">`), bytes.Repeat([]byte("<g></g>"), 200)...), []byte(`<image onerror="alert(1)" href="x"/></svg>`)...)},
		{"javascript link", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><a href="javascript:alert(1)"><rect/></a></svg>`)},
		{"foreignObject", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><foreignObject><iframe src="x"></iframe></foreignObject></svg>`)},
		{"entity", []byte(`<?xml version="1.0"?><!DOCTYPE svg [<!ENTITY x "y">]><svg xmlns="http://www.w3.org/2000/svg">&x;</svg>`)},
	}
	for _, tc := range bad {
		ctype, problem := sniffLogo(tc.data)
		if problem == "" || ctype != "" {
			t.Errorf("%s: accepted as %q", tc.name, ctype)
		}
	}
	if _, problem := sniffLogo([]byte(`<svg onload="x()"></svg>`)); !strings.Contains(problem, "scripts") {
		t.Errorf("the message explains the problem: %q", problem)
	}
}

// sniffLogo looks for a fixed list of byte strings ("onload=", "onerror=",
// "onclick="…). Event handler attributes written with spaces around "=",
// or other handlers (onmouseover, onbegin of SVG animations, onfocus), slip
// through. The logo is served with "sandbox" in its CSP, which keeps them
// from running when the file is opened directly, so this is
// defence-in-depth, but the check's promise ("no scripts") isn't kept.
func TestSniffLogoRejectsEveryEventHandler(t *testing.T) {
	for name, svg := range map[string]string{
		"onload with spaces": `<svg xmlns="http://www.w3.org/2000/svg" onload = "alert(1)"><rect/></svg>`,
		"onmouseover":        `<svg xmlns="http://www.w3.org/2000/svg"><rect width="9" height="9" onmouseover="alert(1)"/></svg>`,
		"animation onbegin":  `<svg xmlns="http://www.w3.org/2000/svg"><animate attributeName="x" dur="1s" onbegin="alert(1)"/></svg>`,
	} {
		if ctype, problem := sniffLogo([]byte(svg)); problem == "" {
			t.Errorf("%s: accepted as %s (settings.go sniffLogo matches literal strings; reject any on*= attribute, e.g. with the regexp (?i)\\son[a-z]+\\s*=)", name, ctype)
		}
	}
}

func TestSafeFileName(t *testing.T) {
	cases := []struct{ name, ctype, want string }{
		{"Acme Mark.SVG", "image/svg+xml", "acme-mark.svg"},
		{`C:\Users\me\logo final.png`, "image/png", "logo-final.png"},
		{"../../etc/passwd", "image/png", "passwd.png"},
		{"", "image/webp", "logo.webp"},
		{"évènement.jpeg", "image/jpeg", "evenement.jpg"},
	}
	for _, tc := range cases {
		if got := safeFileName(tc.name, tc.ctype); got != tc.want {
			t.Errorf("safeFileName(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// --- page options ---------------------------------------------------------------------

func TestApplyPages(t *testing.T) {
	digest := uuid.Must(uuid.NewV7())
	lists := []audience.PublicList{{ID: digest, Name: "Weekly Digest", Public: true}}
	before := defaultPages()
	heading := "  The Acme newsletter  "
	after, changed, err := applyPages(before, PublicPageInput{Subscribe: &SubscribePage{Heading: heading, ListText: map[string]string{
		digest.String(): "  Every Tuesday.  ", uuid.NewString(): "a list of another workspace", digest.String() + "x": "junk",
	}}}, lists)
	if err != nil {
		t.Fatal(err)
	}
	if after.Subscribe.Heading != "The Acme newsletter" || len(after.Subscribe.ListText) != 1 || after.Subscribe.ListText[digest.String()] != "Every Tuesday." {
		t.Errorf("subscribe page cleaned: %+v", after.Subscribe)
	}
	if len(changed) != 1 || changed[0] != "subscribe" || after.Unsubscribe != before.Unsubscribe {
		t.Errorf("only the subscribe page changes: %v", changed)
	}
	if _, _, err := applyPages(before, PublicPageInput{Archive: &ArchivePage{Lists: []string{uuid.NewString()}}}, lists); err == nil {
		t.Error("archive lists of another workspace are accepted")
	}
	if _, _, err := applyPages(before, PublicPageInput{Unsubscribe: &UnsubscribePage{Fewer: true, FewerTitle: " ", FewerButton: "Fewer"}}, lists); err == nil {
		t.Error("the fewer-emails offer needs a heading")
	}
	if _, _, err := applyPages(before, PublicPageInput{Subscribe: &SubscribePage{ListText: map[string]string{digest.String(): strings.Repeat("x", 301)}}}, lists); err == nil {
		t.Error("list lines over 300 characters are accepted")
	}
	bad := "sam <script>"
	if _, _, err := applyPages(before, PublicPageInput{PreviewEmail: &bad}, lists); err == nil {
		t.Error("a preview address with markup is accepted")
	}
	email := "  Sam.Rivera@Outlook.com "
	after, _, err = applyPages(before, PublicPageInput{PreviewEmail: &email}, lists)
	if err != nil || after.PreviewEmail != "sam.rivera@outlook.com" {
		t.Errorf("preview email %q, %v", after.PreviewEmail, err)
	}
	empty := ""
	after, _, _ = applyPages(after, PublicPageInput{PreviewEmail: &empty}, lists)
	if after.PreviewEmail != "" {
		t.Error("an empty preview email goes back to the latest confirmation")
	}
}

func TestPagesOfDefaults(t *testing.T) {
	p := pagesOf(nil)
	if !p.Archive.Enabled || !p.Subscribe.Index || !p.Unsubscribe.Fewer || p.Subscribe.ListText == nil || p.Preferences.Lists == nil {
		t.Errorf("defaults: %+v", p)
	}
	p = pagesOf([]byte(`{"archive":{"enabled":false},"subscribe":{"listText":null}}`))
	if p.Archive.Enabled || p.Subscribe.ListText == nil || p.Archive.Lists == nil {
		t.Errorf("stored options over defaults: %+v", p)
	}
}

// --- emails ------------------------------------------------------------------------------

func TestConfirmationEmail(t *testing.T) {
	b := Branding{PublisherName: `Acme <b>&</b>`, Palette: Palette{Accent: "#4338CA", OnAccent: "#FFFFFF", Background: "#F5F3FF"}}
	link := "https://rc.example/p/acme-newsletter/confirm/abc?x=1&y=2"
	m := ConfirmationEmail(b, "  ", []string{"Weekly Digest", "Product Updates", "Beta"}, link, false)
	if m.Subject != "Confirm your subscription to Acme <b>&</b>" || m.Kind != "double_opt_in" {
		t.Errorf("subject %q kind %q", m.Subject, m.Kind)
	}
	if !strings.Contains(m.Text, "Hi there,") || !strings.Contains(m.Text, "Weekly Digest, Product Updates and Beta") || !strings.Contains(m.Text, link) {
		t.Errorf("text:\n%s", m.Text)
	}
	if strings.Contains(m.HTML, "<b>&</b>") || !strings.Contains(m.HTML, "Acme &lt;b&gt;&amp;&lt;/b&gt;") {
		t.Error("the HTML version doesn't escape the publisher name")
	}
	if !strings.Contains(m.HTML, `href="https://rc.example/p/acme-newsletter/confirm/abc?x=1&amp;y=2"`) {
		t.Error("the HTML version lacks the escaped confirmation link")
	}
	r := ConfirmationEmail(b, "Sam", []string{"Weekly Digest"}, link, true)
	if !strings.HasPrefix(r.Subject, "Reminder: ") || !strings.Contains(r.Text, "Hi Sam,") || !strings.Contains(r.Text, "Yesterday you asked for Weekly Digest") {
		t.Errorf("reminder: %q\n%s", r.Subject, r.Text)
	}
}
