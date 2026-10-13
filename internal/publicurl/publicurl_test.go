package publicurl

import "testing"

func TestBuilder(t *testing.T) {
	b := New("https://reputecast.acme.io/")
	cases := map[string]string{
		b.Open("tok"):                           "https://reputecast.acme.io/t/o/tok",
		b.Click("tok"):                          "https://reputecast.acme.io/t/c/tok",
		b.Unsubscribe("acme-newsletter", "tok"): "https://reputecast.acme.io/p/acme-newsletter/unsubscribe/tok",
		b.Preferences("acme-newsletter", "tok"): "https://reputecast.acme.io/p/acme-newsletter/preferences/tok",
		b.Confirm("acme-newsletter", "tok"):     "https://reputecast.acme.io/p/acme-newsletter/confirm/tok",
		b.ViewOnline("acme-newsletter", "tok"):  "https://reputecast.acme.io/p/acme-newsletter/view/tok",
		b.Subscribe("acme-newsletter", ""):      "https://reputecast.acme.io/p/acme-newsletter/subscribe",
		b.Subscribe("acme-newsletter", "beta"):  "https://reputecast.acme.io/p/acme-newsletter/subscribe/beta",
		b.Archive("acme-newsletter"):            "https://reputecast.acme.io/p/acme-newsletter/archive",
		b.ArchiveIssue("acme-newsletter", "x"):  "https://reputecast.acme.io/p/acme-newsletter/archive/x",
		b.ArchiveFeed("acme-newsletter"):        "https://reputecast.acme.io/p/acme-newsletter/archive.rss",
		b.Report("tok"):                         "https://reputecast.acme.io/r/tok",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
	if got := b.Subscribe("a b", "c/d"); got != "https://reputecast.acme.io/p/a%20b/subscribe/c%2Fd" {
		t.Errorf("path segments are not escaped: %q", got)
	}
}

func TestSourceFollowsChanges(t *testing.T) {
	src := NewSource("https://reputecast.acme.io/")
	b := From(src)
	if b.Report("tok") != "https://reputecast.acme.io/r/tok" {
		t.Fatalf("report: %q", b.Report("tok"))
	}
	if src.Set("https://reputecast.acme.io") {
		t.Fatal("the same URL without its slash is no change")
	}
	if !src.Set(" https://news.acme.io/ ") || b.Base() != "https://news.acme.io" || b.Open("x") != "https://news.acme.io/t/o/x" {
		t.Fatalf("after a change: %q", b.Base())
	}
	if (Builder{}).Base() != "" {
		t.Fatal("a zero Builder has no base")
	}
}

func TestEffective(t *testing.T) {
	for _, c := range []struct{ saved, configured, want string }{
		{"https://news.acme.io/", "http://localhost:8080", "https://news.acme.io"},
		{"  ", "http://localhost:8080/", "http://localhost:8080"},
		{"", "", ""},
	} {
		if got := Effective(c.saved, c.configured); got != c.want {
			t.Errorf("Effective(%q, %q) = %q, want %q", c.saved, c.configured, got, c.want)
		}
	}
}
