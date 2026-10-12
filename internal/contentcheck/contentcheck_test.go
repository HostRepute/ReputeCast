package contentcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/mjml"
)

func TestSubjectHeuristics(t *testing.T) {
	cases := map[string]string{
		"FLASH SALE!!! 48 HOURS ONLY":                             "all caps, repeated punctuation, urgency",
		"Three launches, one deprecation, and the roadmap for Q4": "",
		"Get it FREE today":                                       "money words",
		"Last chance?!":                                           "repeated punctuation, urgency",
	}
	for subject, want := range cases {
		if got := strings.Join(SubjectHeuristics(subject), ", "); got != want {
			t.Fatalf("%q: got %q want %q", subject, got, want)
		}
	}
	if got := SuggestSubject("FLASH SALE!!! 48 HOURS ONLY"); got != "Flash sale: 48 hours only" {
		t.Fatalf("suggestion: %q", got)
	}
	if got := SuggestSubject("Big news!!"); got != "Big news" {
		t.Fatalf("suggestion: %q", got)
	}
}

func TestPlainTextAndMinify(t *testing.T) {
	src := `<mjml><mj-body>
	<mj-section><mj-column><mj-text font-size="26px" font-weight="700">Three launches</mj-text>
	<mj-text>Hi Sam, here's <a href="https://acme.io/x">what shipped</a>.</mj-text>
	<mj-button href="https://docs.acme.io/flags">Read the docs</mj-button></mj-column></mj-section>
	<mj-section><mj-column><mj-text>Acme Inc · 500 Market St<br><a href="{{ unsubscribe_url }}">Unsubscribe</a></mj-text></mj-column></mj-section>
	</mj-body></mjml>`
	html := mjml.Compile(src).HTML
	text := PlainText(html)
	for _, want := range []string{"THREE LAUNCHES", "Hi Sam, here's what shipped (https://acme.io/x).", "Read the docs: https://docs.acme.io/flags", "--\nAcme Inc · 500 Market St", "Unsubscribe: {{ unsubscribe_url }}"} {
		if !strings.Contains(text, want) {
			t.Fatalf("plain text misses %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "\n\n\n") {
		t.Fatal("blank lines must be collapsed")
	}
	min := Minify(html)
	if len(min) >= len(html) || !strings.Contains(min, "<!--[if mso") || strings.Contains(min, "style=\"font-size:0px; ") {
		t.Fatalf("minify: %d → %d", len(html), len(min))
	}
}

func TestCheckFindsEveryInlineWarning(t *testing.T) {
	big := strings.Repeat(`<span style="font-family:Georgia;color:#374151;font-size:13px;line-height:19px">x</span>`, 1100)
	src := `<mjml><mj-body>
<mj-section><mj-column><mj-image src="/media/1/logo.png" alt="Acme" width="96px"></mj-image><mj-text>Pricing</mj-text></mj-column></mj-section>
<mj-section><mj-column><mj-image src="/media/1/hero.png" width="552px" height="220px" padding="0px"></mj-image><mj-text>48 hours. 30% off.</mj-text></mj-column></mj-section>
<mj-section><mj-column><mj-button href="https://go.acme-promo.link/sale">Claim your discount</mj-button><mj-button href="https://bit.ly/abc">Short</mj-button><mj-button href="#">Empty</mj-button></mj-column></mj-section>
<mj-section><mj-column><mj-text>` + big + `</mj-text></mj-column><mj-column><mj-text>{{ frist_name }}</mj-text></mj-column></mj-section>
</mj-body></mjml>`
	c := &Checker{Blocklist: StaticBlocklist{"go.acme-promo.link": {"SURBL", "Spamhaus DBL"}}}
	r, err := c.Check(context.Background(), Input{Subject: "FLASH SALE!!! 48 HOURS ONLY", MJML: src, OwnDomains: []string{"news.acme.io"}, Fields: []string{"first_name", "email"}})
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]ContentIssue{}
	for _, is := range r.Issues {
		ids[is.ID] = is
	}
	for _, want := range []string{"content.alt_text", "content.size", "links.blocklist", "links.shortener", "links.empty", "compliance.footer", "personalization.unknown_field", "subject.spam"} {
		if _, ok := ids[want]; !ok {
			t.Fatalf("missing %s: %+v", want, r.Issues)
		}
	}
	if r.Verdict != "blocked" || r.Issues[0].Severity != Block {
		t.Fatalf("order and verdict: %s %+v", r.Verdict, r.Issues[0])
	}
	bl := ids["links.blocklist"]
	if bl.Fixes[0].Value != "https://acme.io/sale" || len(bl.Path) != 3 || bl.Path[0] != 2 {
		t.Fatalf("blocklist fix: %+v", bl)
	}
	if alt := ids["content.alt_text"]; alt.Title != "Hero image has no alt text" || alt.Section != 1 {
		t.Fatalf("alt: %+v", alt)
	}
	if r.Sections[0].Images != 1 || r.Sections[0].ImagesWithAlt != 1 {
		t.Fatalf("section summary: %+v", r.Sections[0])
	}
}

func TestCleanEmailIsReady(t *testing.T) {
	src := `<mjml><mj-body>
<mj-section><mj-column><mj-image src="/media/1/hero.png" alt="Roadmap" width="552px" height="180px" padding="0px"></mj-image><mj-text>Hi {{ first_name | default: "there" }}, here's what shipped this week at Acme, what we're retiring in November, and where the next quarter is headed. Feature flags are generally available and audit logs export to S3.</mj-text></mj-column></mj-section>
<mj-section><mj-column><mj-text>Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105<br><a href="{{ unsubscribe_url }}">Unsubscribe</a></mj-text></mj-column></mj-section>
</mj-body></mjml>`
	r, err := (&Checker{}).Check(context.Background(), Input{Subject: "Three launches", MJML: src, Fields: []string{"first_name"}})
	if err != nil {
		t.Fatal(err)
	}
	if r.Verdict != "ready" {
		t.Fatalf("issues: %+v", r.Issues)
	}
}

func TestProberRefusesInternalAddresses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer srv.Close()
	p := NewHTTPProber()
	res := p.Probe(context.Background(), []string{srv.URL + "/ok"})
	if r := res[srv.URL+"/ok"]; !r.Broken || !strings.Contains(r.Error, "internal address") {
		t.Fatalf("loopback must be refused: %+v", r)
	}
	// Without the guard (tests only) broken links are found.
	notFound := httptest.NewServer(http.NotFoundHandler())
	defer notFound.Close()
	open := &HTTPProber{Client: notFound.Client()}
	if r := open.Probe(context.Background(), []string{notFound.URL + "/gone"})[notFound.URL+"/gone"]; !r.Broken || r.Status != 404 {
		t.Fatalf("404: %+v", r)
	}
}
