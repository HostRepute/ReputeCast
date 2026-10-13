package publicpages

// Unit tests of the page helpers: no database. The pages themselves are
// tested over HTTP in pages_test.go (package publicpages_test).

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

var jsonLDBlock = regexp.MustCompile(`(?s)<script type="application/ld\+json">(.*?)</script>`)

func TestInjectHeadReplacesTitleAndRobots(t *testing.T) {
	doc := `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Weekly Digest #141</title>` +
		`<meta name="robots" content="index, follow"><style>p{color:red}</style></head><body><p>Hi there</p></body></html>`
	ld := map[string]any{"@context": "https://schema.org", "@type": "BlogPosting", "headline": `</script><script>alert(1)</script> & more`}
	out := injectHead(doc, emailHead{
		Title: `The case for <boring> & "plain" infrastructure`, Description: `Why we moved "three" services back`, Robots: "noindex, nofollow",
		Canonical: "https://news.acme.io/archive/the-case-for-boring-infrastructure",
		OG:        &og{Type: "article", SiteName: "Acme", Title: "The case", Description: "Why", URL: "https://news.acme.io/archive/x", Image: "https://cdn.example/a.png", Published: "2026-10-06T13:00:00Z"},
		Feed:      "https://news.acme.io/archive.rss", FeedTitle: "Acme newsletter archive", Icon: "https://rc.example/p/acme/logo?v=1", JSONLD: ld,
	})
	if strings.Contains(out, "Weekly Digest #141") {
		t.Error("the email's own <title> is still there")
	}
	if n := strings.Count(strings.ToLower(out), "<title"); n != 1 {
		t.Errorf("%d <title> elements, want 1", n)
	}
	if !strings.Contains(out, `<title>The case for &lt;boring&gt; &amp; &#34;plain&#34; infrastructure</title>`) {
		t.Errorf("escaped title missing:\n%s", out)
	}
	if strings.Contains(out, `content="index, follow"`) || !strings.Contains(out, `<meta name="robots" content="noindex, nofollow">`) {
		t.Errorf("the email's robots tag isn't replaced:\n%s", out)
	}
	for _, want := range []string{
		`<meta name="description" content="Why we moved &#34;three&#34; services back">`,
		`<link rel="canonical" href="https://news.acme.io/archive/the-case-for-boring-infrastructure">`,
		`<meta property="og:type" content="article">`,
		`<meta property="og:image" content="https://cdn.example/a.png">`,
		`<meta property="article:published_time" content="2026-10-06T13:00:00Z">`,
		`<meta name="twitter:card" content="summary_large_image">`,
		`<link rel="alternate" type="application/rss+xml" title="Acme newsletter archive" href="https://news.acme.io/archive.rss">`,
		`<link rel="icon" href="https://rc.example/p/acme/logo?v=1">`,
		`<style>p{color:red}</style>`,
		`<body><p>Hi there</p></body>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s", want)
		}
	}
	// The metadata goes inside <head>.
	head := out[:strings.Index(out, "</head>")]
	if !strings.Contains(head, `rel="canonical"`) || !strings.Contains(head, "application/ld+json") {
		t.Error("the metadata isn't in <head>")
	}
	// JSON-LD can't close its script element and round-trips.
	m := jsonLDBlock.FindAllStringSubmatch(out, -1)
	if len(m) != 1 {
		t.Fatalf("%d JSON-LD blocks, want 1:\n%s", len(m), out)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(m[0][1]), &got); err != nil {
		t.Fatalf("JSON-LD isn't JSON: %v\n%s", err, m[0][1])
	}
	if got["headline"] != ld["headline"] || got["@type"] != "BlogPosting" {
		t.Errorf("JSON-LD %v", got)
	}
	if strings.Count(out, "</script>") != 1 {
		t.Error("data inside the JSON-LD closed the script element")
	}
}

func TestInjectHeadFragments(t *testing.T) {
	head := emailHead{Title: "Hello", Robots: "noindex, nofollow"}
	t.Run("a fragment gets a whole document", func(t *testing.T) {
		out := injectHead(`<p>Just a paragraph</p>`, head)
		if !strings.HasPrefix(out, "<!doctype html>") {
			t.Errorf("no doctype:\n%s", out)
		}
		for _, want := range []string{`<html lang="en">`, `<meta charset="utf-8">`, `<meta name="viewport"`, "<title>Hello</title>", `content="noindex, nofollow"`,
			"</head>\n<body>\n<p>Just a paragraph</p>\n</body>\n</html>"} {
			if !strings.Contains(out, want) {
				t.Errorf("missing %q:\n%s", want, out)
			}
		}
	})
	t.Run("an <html> without <head> gets one", func(t *testing.T) {
		out := injectHead(`<html lang="fr"><body><header>Top</header><p>x</p></body></html>`, head)
		if !strings.HasPrefix(out, `<html lang="fr">`+"\n<head>") {
			t.Errorf("<head> isn't right after <html>:\n%s", out)
		}
		if !strings.Contains(out, "<body><header>Top</header><p>x</p></body>") {
			t.Errorf("the body changed (a <header> isn't a <head>):\n%s", out)
		}
		if strings.Count(out, "<html") != 1 {
			t.Error("a second <html>")
		}
	})
	t.Run("upper case tags", func(t *testing.T) {
		out := injectHead(`<HTML><HEAD><TITLE>Old</TITLE><META NAME="ROBOTS" CONTENT="index"></HEAD><BODY>x</BODY></HTML>`, head)
		if strings.Contains(out, "Old") || strings.Contains(out, `CONTENT="index"`) {
			t.Errorf("upper-case title or robots kept:\n%s", out)
		}
		if !strings.HasPrefix(out, "<HTML><HEAD>\n<title>Hello</title>") {
			t.Errorf("metadata isn't inside <HEAD>:\n%s", out)
		}
	})
	t.Run("injectTop puts the bar after <body>", func(t *testing.T) {
		if got := injectTop(`<html><body class="x"><p>a</p></body></html>`, "<header>bar</header>"); got != `<html><body class="x"><header>bar</header><p>a</p></body></html>` {
			t.Errorf("injectTop: %s", got)
		}
		if got := injectTop(`<p>a</p>`, "<header>bar</header>"); got != `<header>bar</header><p>a</p>` {
			t.Errorf("injectTop without <body>: %s", got)
		}
	})
	t.Run("withStylesheet links in <head>", func(t *testing.T) {
		got := withStylesheet(`<html><head><title>x</title></head><body></body></html>`, "https://rc.example/p/_/issue.css?v=1&x=2")
		if !strings.Contains(got, "<head>\n"+`<link rel="stylesheet" href="https://rc.example/p/_/issue.css?v=1&amp;x=2">`) {
			t.Errorf("withStylesheet: %s", got)
		}
	})
}

func TestAbsolutizeMedia(t *testing.T) {
	base := "https://rc.example/"
	cases := []struct{ in, want string }{
		{`<img src="/media/0192/a.png" alt="">`, `<img src="https://rc.example/media/0192/a.png" alt="">`},
		{`<a href='/media/0192/guide.pdf'>PDF</a>`, `<a href='https://rc.example/media/0192/guide.pdf'>PDF</a>`},
		{`<td background="/media/0192/bg.png">`, `<td background="https://rc.example/media/0192/bg.png">`},
		{`<IMG SRC = "/media/0192/b.gif">`, `<IMG SRC = "https://rc.example/media/0192/b.gif">`},
		{`<div style="background:url(/media/0192/bg.png) center">`, `<div style="background:url(https://rc.example/media/0192/bg.png) center">`},
		{`<img src="https://cdn.example/media/x.png">`, `<img src="https://cdn.example/media/x.png">`},
		{`<img src="/mediakit/x.png">`, `<img src="/mediakit/x.png">`},
		{`<a href="/p/acme/media/x">x</a>`, `<a href="/p/acme/media/x">x</a>`},
		{`<p>Text about /media/ paths</p>`, `<p>Text about /media/ paths</p>`},
	}
	for _, tc := range cases {
		if got := absolutizeMedia(tc.in, base); got != tc.want {
			t.Errorf("absolutizeMedia(%s)\n got %s\nwant %s", tc.in, got, tc.want)
		}
	}
	if got := absolutizeMedia(`<img src="/media/a.png">`, ""); got != `<img src="/media/a.png">` {
		t.Errorf("without a base the document is unchanged: %s", got)
	}
	if got := absolutizeMedia(`<img src="/media/a.png"><img src="/media/b.png">`, "https://rc.example"); strings.Count(got, "https://rc.example/media/") != 2 {
		t.Errorf("every reference is rewritten: %s", got)
	}
}

func TestFirstImage(t *testing.T) {
	cases := []struct{ doc, want string }{
		{`<p>x</p><img alt="logo" width="100" src="https://cdn.example/a.png?x=1&amp;y=2"><img src="https://cdn.example/b.png">`, "https://cdn.example/a.png?x=1&y=2"},
		{`<IMG SRC='http://cdn.example/c.gif'>`, "http://cdn.example/c.gif"},
		{`<img src="/media/relative.png">`, ""},
		{`<img src="data:image/png;base64,AAAA">`, ""},
		{`<p>No images</p>`, ""},
	}
	for _, tc := range cases {
		if got := firstImage(tc.doc); got != tc.want {
			t.Errorf("firstImage(%s) = %q, want %q", tc.doc, got, tc.want)
		}
	}
}

func TestArchiveSummary(t *testing.T) {
	first := time.Date(2025, 3, 4, 9, 0, 0, 0, time.UTC)
	cases := []struct {
		names []string
		n     int64
		want  string
	}{
		{[]string{"Weekly Digest", "Product Updates"}, 44, "Every issue of the Weekly Digest and Product Updates since 2025. 44 issues."},
		{[]string{"Weekly Digest"}, 1, "Every issue of the Weekly Digest since 2025. 1 issue."},
		{[]string{"A", "B", "C"}, 1234, "Every issue of the A, B and C since 2025. 1,234 issues."},
		{nil, 3, "Every issue since 2025. 3 issues."},
		{[]string{"Weekly Digest"}, 0, "Issues appear here as they're sent."},
	}
	for _, tc := range cases {
		if got := archiveSummary(tc.names, first, tc.n); got != tc.want {
			t.Errorf("archiveSummary(%v, %d) = %q, want %q", tc.names, tc.n, got, tc.want)
		}
	}
}

func TestArchiveParams(t *testing.T) {
	updates := uuid.Must(uuid.NewV7())
	lists := map[string]uuid.UUID{"product-updates": updates}
	p := archiveParamsOf(url.Values{"q": {"  boring  "}, "list": {"product-updates"}, "page": {"3"}}, lists)
	if p.q != "boring" || p.list != "product-updates" || p.listID == nil || *p.listID != updates || p.page != 3 {
		t.Errorf("params %+v", p)
	}
	for _, page := range []string{"", "0", "-2", "1", "abc", "100000", "9999999999999999999"} {
		if p := archiveParamsOf(url.Values{"page": {page}}, lists); p.page != 1 {
			t.Errorf("?page=%s is page %d, want 1", page, p.page)
		}
	}
	if p := archiveParamsOf(url.Values{"list": {"no-such-list"}}, lists); p.list != "" || p.listID != nil {
		t.Errorf("an unknown list filters: %+v", p)
	}
	if p := archiveParamsOf(url.Values{"q": {strings.Repeat("x", 300)}}, lists); len(p.q) != 100 {
		t.Errorf("search is cut to 100: %d", len(p.q))
	}
	// Hrefs keep the filters and drop page 1.
	base := "https://news.acme.io/archive"
	if got := p.href(base, 1, "", ""); got != base {
		t.Errorf("href %s", got)
	}
	if got := p.href(base, 2, "product-updates", "boring infra"); got != base+"?list=product-updates&page=2&q=boring+infra" {
		t.Errorf("href %s", got)
	}
}

// The archive's ?q is cut to 100 bytes, which can split a multi-byte
// character: a 34-character Japanese search (102 bytes) becomes invalid
// UTF-8, which Postgres refuses in the ILIKE parameter, so the public
// archive answers 500 (see TestArchive in pages_test.go).
func TestArchiveParamsKeepUTF8(t *testing.T) {
	for _, q := range []string{strings.Repeat("設計", 17), "a" + strings.Repeat("é", 60)} {
		p := archiveParamsOf(url.Values{"q": {q}}, nil)
		if !utf8.ValidString(p.q) {
			t.Errorf("?q=%s… is cut to %d bytes of invalid UTF-8 (archive.go archiveParamsOf: cut at a rune boundary, e.g. on []rune(q)[:100])", q[:9], len(p.q))
		}
	}
}

func TestVariantContent(t *testing.T) {
	content := json.RawMessage(`{"kind":"content","winner":"B","variants":[{"key":"A","subject":"Subject A","html":"<p>A</p>"},{"key":"B","subject":"Subject B","html":"<p>B</p>"}]}`)
	subjectTest := json.RawMessage(`{"kind":"subject","variants":[{"key":"A","subject":"Subject A"},{"key":"B","subject":"Subject B","html":"<p>ignored</p>"}]}`)
	cases := []struct {
		name          string
		ab            json.RawMessage
		variant       string
		subject, body string
	}{
		{"content test, variant B", content, "B", "Subject B", "<p>B</p>"},
		{"content test, variant A is the campaign", content, "A", "Campaign subject", "<p>campaign</p>"},
		{"no variant", content, "", "Campaign subject", "<p>campaign</p>"},
		{"unknown variant", content, "C", "Campaign subject", "<p>campaign</p>"},
		{"subject test keeps the campaign's body", subjectTest, "B", "Subject B", "<p>campaign</p>"},
		{"no test", nil, "B", "Campaign subject", "<p>campaign</p>"},
		{"broken test", json.RawMessage(`{"kind":`), "B", "Campaign subject", "<p>campaign</p>"},
	}
	for _, tc := range cases {
		s, b := variantContent("Campaign subject", "<p>campaign</p>", tc.ab, tc.variant)
		if s != tc.subject || b != tc.body {
			t.Errorf("%s: %q %q, want %q %q", tc.name, s, b, tc.subject, tc.body)
		}
	}
	if s, b := winnerContent("Campaign subject", "<p>campaign</p>", content); s != "Subject B" || b != "<p>B</p>" {
		t.Errorf("the archive shows the winner: %q %q", s, b)
	}
	if s, b := winnerContent("Campaign subject", "<p>campaign</p>", subjectTest); s != "Campaign subject" || b != "<p>campaign</p>" {
		t.Errorf("no winner yet: %q %q", s, b)
	}
}

func TestSplitConsent(t *testing.T) {
	c := splitConsent("I agree to receive emails from Acme. See the privacy policy.", "https://acme.io/privacy")
	if c.Before != "I agree to receive emails from Acme. See the " || c.Link != "privacy policy" || c.After != "." || c.Href != "https://acme.io/privacy" {
		t.Errorf("split %+v", c)
	}
	c = splitConsent("Read our Privacy Policy first", "https://acme.io/privacy")
	if c.Link != "Privacy Policy" || c.Before != "Read our " || c.After != " first" {
		t.Errorf("the link keeps the sentence's case: %+v", c)
	}
	c = splitConsent("I agree. See the privacy policy.", "")
	if c.Before != "I agree. See the privacy policy." || c.Href != "" || c.Link != "" {
		t.Errorf("no privacy URL, no link: %+v", c)
	}
	c = splitConsent("I agree to emails.", "https://acme.io/privacy")
	if c.Before != "I agree to emails." || c.Href != "" {
		t.Errorf("no privacy policy in the sentence: %+v", c)
	}
}

// splitConsent has the index bug of forms.PlainHTML: the hosted page's
// consent sentence is cut in the wrong place (or the page panics) when the
// sentence has letters whose lowercase form has another UTF-8 length.
func TestSplitConsentNonASCII(t *testing.T) {
	for _, text := range []string{"İstanbul'dan e-posta almayı kabul ediyorum. See the privacy policy.", "Ⱥ privacy policy"} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("splitConsent(%q) panicked: %v (subscribe.go splitConsent slices text with an index of strings.ToLower(text))", text, r)
				}
			}()
			c := splitConsent(text, "https://acme.io/privacy")
			if c.Link != "privacy policy" || c.Before+c.Link+c.After != text {
				t.Errorf("splitConsent(%q) = %q | %q | %q (subscribe.go splitConsent slices text with an index of strings.ToLower(text))", text, c.Before, c.Link, c.After)
			}
		}()
	}
}

func TestFormatCount(t *testing.T) {
	cases := map[int64]string{0: "0", 7: "7", 999: "999", 1000: "1,000", 31988: "31,988", 1234567: "1,234,567", -5: "-5"}
	for n, want := range cases {
		if got := formatCount(n); got != want {
			t.Errorf("formatCount(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestSubscribeTitle(t *testing.T) {
	cases := map[string]string{
		"The Acme newsletter": "Subscribe to the Acme newsletter",
		"Acme Weekly":         "Subscribe to Acme Weekly",
	}
	for in, want := range cases {
		if got := subscribeTitle(in); got != want {
			t.Errorf("subscribeTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestListNames(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{nil, ""}, {[]string{"A"}, "A"}, {[]string{"A", "B"}, "A and B"}, {[]string{"A", "B", "C"}, "A, B and C"},
	}
	for _, tc := range cases {
		if got := listNames(tc.in); got != tc.want {
			t.Errorf("listNames(%v) = %q", tc.in, got)
		}
		if got := andNames(tc.in); got != tc.want {
			t.Errorf("andNames(%v) = %q", tc.in, got)
		}
	}
}

func TestCSP(t *testing.T) {
	strict := csp{}.String()
	for _, want := range []string{"default-src 'none'", "script-src 'self'", "style-src 'self'", "form-action 'self'", "frame-ancestors 'none'", "base-uri 'none'", "connect-src 'self'"} {
		if !strings.Contains(strict, want) {
			t.Errorf("CSP lacks %q: %s", want, strict)
		}
	}
	if strings.Contains(strict, "unsafe-inline") || strings.Contains(strict, "unsafe-eval") || strings.Contains(strict, "frame-src") {
		t.Errorf("strict CSP: %s", strict)
	}
	challenge := csp{scripts: []string{"https://challenges.cloudflare.com"}, frames: []string{"https://challenges.cloudflare.com"},
		fonts: []string{"https://fonts.example"}, formAction: []string{"https://acme.io"}, ancestors: []string{"'self'", "https://acme.io"}}.String()
	for _, want := range []string{"script-src 'self' https://challenges.cloudflare.com", "frame-src https://challenges.cloudflare.com",
		"font-src 'self' https://fonts.example", "style-src 'self' https://fonts.example", "form-action 'self' https://acme.io", "frame-ancestors 'self' https://acme.io"} {
		if !strings.Contains(challenge, want) {
			t.Errorf("CSP lacks %q: %s", want, challenge)
		}
	}
	if !strings.Contains(emailCSP, "script-src 'none'") || !strings.Contains(emailCSP, "form-action 'none'") || !strings.Contains(emailCSP, "frame-ancestors 'none'") {
		t.Errorf("emails in the browser never run scripts: %s", emailCSP)
	}
}

func TestPlainDocumentEscapes(t *testing.T) {
	doc := plainDocument("Hi <b>there</b> & welcome")
	if !strings.Contains(doc, "Hi &lt;b&gt;there&lt;/b&gt; &amp; welcome") || strings.Contains(doc, "<b>") {
		t.Errorf("plain text isn't escaped: %s", doc)
	}
}
