package growth

import (
	"strings"
	"testing"
	"time"
)

func TestTouchOf(t *testing.T) {
	tc := TouchOf("https://www.acme.io/pricing/?utm_source=newsletter&utm_medium=email&utm_campaign=Oct%20launch&utm_content=hero&utm_term=team#top",
		"https://news.ycombinator.com/item?id=1")
	want := Touch{UTMSource: "newsletter", UTMMedium: "email", UTMCampaign: "Oct launch", UTMContent: "hero", UTMTerm: "team",
		Page: "acme.io/pricing", Referrer: "news.ycombinator.com/item"}
	if tc != want {
		t.Fatalf("touch: %+v, want %+v", tc, want)
	}
	if !TouchOf("", "").Empty() {
		t.Fatal("an empty touch says something")
	}
	// Control characters go; long values are cut on a rune boundary.
	long := strings.Repeat("é", 150)
	tc = TouchOf("https://acme.io/?utm_source=a%0Ab&utm_campaign="+long, "javascript:alert(1)")
	if tc.UTMSource != "ab" || tc.Referrer != "" {
		t.Fatalf("cleaned: %+v", tc)
	}
	if len(tc.UTMCampaign) > maxValue || !strings.HasPrefix(long, tc.UTMCampaign) {
		t.Fatalf("cut: %d bytes", len(tc.UTMCampaign))
	}
}

func TestPageAndDomain(t *testing.T) {
	cases := map[string]string{
		"https://acme.io/pricing?plan=team#top": "acme.io/pricing",
		"http://WWW.Acme.io/":                   "acme.io",
		"https://acme.io:443/a":                 "acme.io/a",
		"http://localhost:5313/subscribe":       "localhost:5313/subscribe",
		"ftp://acme.io/file":                    "",
		"not a url":                             "",
		"":                                      "",
	}
	for in, want := range cases {
		if got := PageOf(in); got != want {
			t.Errorf("PageOf(%q) = %q, want %q", in, got, want)
		}
	}
	if d := DomainOf("blog.acme.io/2026/10/x"); d != "blog.acme.io" {
		t.Fatalf("domain: %q", d)
	}
	if d := DomainOf("localhost:5313/subscribe"); d != "localhost" {
		t.Fatalf("domain with port: %q", d)
	}
}

func TestSourceKeys(t *testing.T) {
	if k := SourceKey(KindForm, "abc"); k != "form:abc" {
		t.Fatalf("key: %q", k)
	}
	if k := SourceKey(KindPerson, ""); k != "person" {
		t.Fatalf("key without id: %q", k)
	}
	if kind, id := splitKey("carried:import:mailchimp"); kind != KindCarried || id != "import:mailchimp" {
		t.Fatalf("split: %q %q", kind, id)
	}
	if !kindOf(KindImport).Imported || kindOf(KindForm).Imported {
		t.Fatal("imports are counted apart from growth")
	}
	if kindOf("something_new").Order == 0 {
		t.Fatal("an unknown kind sorts after the known ones")
	}
}

func TestNames(t *testing.T) {
	if n := workflowShort("Beta application (Typeform) → ReputeCast contact"); n != "Beta application (Typeform)" {
		t.Fatalf("workflow: %q", n)
	}
	if n := workflowShort("Plain"); n != "Plain" {
		t.Fatalf("workflow without arrow: %q", n)
	}
	for in, want := range map[string]string{"mailchimp": "Mailchimp", "csv": "CSV", "mailerlite": "MailerLite", "": "File", "acmemail": "Acmemail"} {
		if got := platformName(in); got != want {
			t.Errorf("platformName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPeriodOf(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	now := time.Date(2026, 10, 15, 20, 0, 0, 0, time.UTC) // 16 Oct 03:00 in Ho Chi Minh City
	w, err := periodOf(GrowthQuery{}, now, loc)
	if err != nil || w.kind != "30d" || w.days != 30 {
		t.Fatalf("default: %+v %v", w, err)
	}
	if got := w.fromDay.Format("2006-01-02"); got != "2026-09-17" {
		t.Fatalf("30 days start on %s", got)
	}
	if got := w.to.In(loc).Format("2006-01-02 15:04"); got != "2026-10-17 00:00" {
		t.Fatalf("30 days end at %s", got)
	}
	w, err = periodOf(GrowthQuery{Period: "custom", From: "2026-01-01", To: "2026-01-31"}, now, loc)
	if err != nil || w.days != 31 || !w.from.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, loc)) {
		t.Fatalf("custom: %+v %v", w, err)
	}
	if _, err := periodOf(GrowthQuery{Period: "custom", From: "2026-02-01", To: "2026-01-01"}, now, loc); err == nil {
		t.Fatal("a period that ends before it starts")
	}
	if _, err := periodOf(GrowthQuery{Period: "custom", From: "2020-01-01", To: "2026-01-01"}, now, loc); err == nil {
		t.Fatal("a period longer than two years")
	}
	if w, _ := periodOf(GrowthQuery{Period: "bogus"}, now, loc); w.kind != "30d" {
		t.Fatalf("unknown period: %q", w.kind)
	}
}
