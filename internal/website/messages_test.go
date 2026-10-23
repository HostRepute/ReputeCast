package website

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Every catalog key the package names is in messages/en.json (msgs.M
// panics on a missing key, which would otherwise surface only when that
// error or page is first shown).
func TestMessageKeysExist(t *testing.T) {
	keys := msgs.Keys()
	re := regexp.MustCompile(`(?:tr\(ctx, |\.Text\(|T\(|\.T |words\.Text\()"((?:post|tag|nav|identity|home|meter|access|paywall|site|check|gate|signin|mail|join|account|report|notify)\.[A-Za-z_.]+)"`)
	seen := 0
	files, _ := filepath.Glob("*.go")
	tmpl, _ := filepath.Glob("../publicpages/templates/site/*.html")
	more, _ := filepath.Glob("../publicpages/site*.go")
	for _, f := range append(append(files, tmpl...), more...) {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			key := m[1]
			if strings.HasSuffix(key, ".") {
				continue
			}
			seen++
			if !slices.Contains(keys, key) {
				t.Errorf("%s: %q isn't in messages/en.json", f, key)
			}
		}
	}
	if seen < 60 {
		t.Fatalf("found only %d keys; the pattern is out of date", seen)
	}
	// Keys built from parts.
	for _, kind := range []string{"view_online", "footer", "referral", "header", "block"} {
		if !slices.Contains(keys, "check.hidden.kind."+kind) {
			t.Errorf("check.hidden.kind.%s is missing", kind)
		}
	}
}

func TestCheckText(t *testing.T) {
	ctx := context.Background()
	title, detail := CheckText(ctx, WebsiteWebCheck{Key: "address", Status: "block", Path: "/posts/feature-flags", Suggestion: "feature-flags-at-scale",
		Taken: &WebsiteSlugOwner{Title: "Feature flags (2025)"}})
	if title != "The address is taken" || !strings.Contains(detail, "Use feature-flags-at-scale?") {
		t.Fatalf("taken: %q %q", title, detail)
	}
	title, detail = CheckText(ctx, WebsiteWebCheck{Key: "tokens", Status: "warn", Fallbacks: []WebsiteFallback{{Source: "{{ first_name }}", Sample: "Hi there"}}})
	if title != "1 token shows its fallback" || detail != "{{ first_name }} reads “Hi there” on the web" {
		t.Fatalf("tokens: %q %q", title, detail)
	}
	title, detail = CheckText(ctx, WebsiteWebCheck{Key: "hidden", Status: "info", Hidden: []WebsiteHiddenBlock{{Kind: "view_online"}, {Name: "Beta Program"}, {Kind: "footer"}}})
	if title != "3 blocks are hidden on the web" || detail != "View in browser, Beta Program, Footer + unsubscribe" {
		t.Fatalf("hidden: %q %q", title, detail)
	}
}

func TestReferrerLabels(t *testing.T) {
	for in, want := range map[string]string{"": "Email clients and direct", "www.google.com": "Google", "news.google.com": "Google", "lnkd.in": "LinkedIn",
		"news.ycombinator.com": "news.ycombinator.com"} {
		if got := referrerLabel(context.Background(), in); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
	order := []string{"a", "b", "c"}
	sortByViews(order, map[string]int64{"a": 1, "b": 3, "c": 2})
	if strings.Join(order, ",") != "b,c,a" {
		t.Fatalf("order: %v", order)
	}
}

// A member event's detail is stored in English and reads in the member's
// language; a detail ReputeCast didn't write reads as stored.
func TestMemberEventDetail(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	if got := memberEventDetail("memberEvent.suppressed"); got != "the address is suppressed" {
		t.Fatalf("stored: %q", got)
	}
	for stored, want := range map[string]string{"the address is suppressed": "địa chỉ đang bị chặn gửi", "from their account page": "từ trang tài khoản của họ",
		"/posts/feature-flags": "/posts/feature-flags", "new@example.com": "new@example.com", "": ""} {
		if got := memberEventDetailIn(vi, stored); got != want {
			t.Errorf("%q in Vietnamese: %q, want %q", stored, got, want)
		}
		if got := memberEventDetailIn(i18n.Default, stored); got != stored {
			t.Errorf("%q in English: %q", stored, got)
		}
	}
}

// A sign-in's return address stays on the site: no other host, however
// it's spelled.
func TestSafeReturn(t *testing.T) {
	for in, want := range map[string]string{
		"":                             "",
		"/":                            "/",
		"/p/acme/posts/x?ref=1":        "/p/acme/posts/x?ref=1",
		"  /posts/x  ":                 "/posts/x",
		"//evil.example":               "",
		"/\\evil.example":              "",
		"\\\\evil.example":             "",
		"/\t/evil.example":             "",
		"/\n/evil.example":             "",
		"https://evil.example/":        "",
		"javascript:alert(1)":          "",
		"posts/x":                      "",
		"/" + strings.Repeat("a", 600): "",
	} {
		if got := SafeReturn(in); got != want {
			t.Errorf("SafeReturn(%q) = %q, want %q", in, got, want)
		}
	}
}

// A post's 680 px column goes back to an email's 600 px when the post is
// sent as a campaign; other widths stay.
func TestEmailWidth(t *testing.T) {
	for in, want := range map[string]string{
		`<mjml><mj-body width="680px"><mj-section></mj-section></mj-body></mjml>`:                                   `<mjml><mj-body width="600px"><mj-section></mj-section></mj-body></mjml>`,
		`<mjml><mj-body background-color="#fff" width="680px"></mj-body></mjml>`:                                    `<mjml><mj-body background-color="#fff" width="600px"></mj-body></mjml>`,
		`<mjml><mj-body width="640px"></mj-body></mjml>`:                                                            `<mjml><mj-body width="640px"></mj-body></mjml>`,
		`<mjml><mj-body><mj-section><mj-column><mj-image width="680px"/></mj-column></mj-section></mj-body></mjml>`: `<mjml><mj-body><mj-section><mj-column><mj-image width="680px"/></mj-column></mj-section></mj-body></mjml>`,
	} {
		if got := emailWidth(in); got != want {
			t.Errorf("emailWidth(%q) = %q", in, got)
		}
	}
}

// A post's default cover is the email's first real image: not its logo.
func TestFirstImage(t *testing.T) {
	doc := `<mjml><mj-body>
  <mj-section><mj-column><mj-image src="/media/starters/logo.png" alt="Acme" width="96px"></mj-image></mj-column></mj-section>
  <mj-section><mj-column><mj-image src="https://cdn.example/hero.png" alt="Rows of switches" width="552px"></mj-image></mj-column></mj-section>
</mj-body></mjml>`
	if src, alt := firstImage(doc); src != "https://cdn.example/hero.png" || alt != "Rows of switches" {
		t.Fatalf("cover: %q %q", src, alt)
	}
	if src, _ := firstImage(`<mjml><mj-body><mj-section><mj-column><mj-image src="/media/a.png"></mj-image></mj-column></mj-section></mj-body></mjml>`); src != "/media/a.png" {
		t.Fatalf("an image without a width: %q", src)
	}
	if src, _ := firstImage(`<mjml><mj-body><mj-section><mj-column><mj-image src="/media/icon.png" width="40px"></mj-image></mj-column></mj-section></mj-body></mjml>`); src != "" {
		t.Fatalf("an icon: %q", src)
	}
}
